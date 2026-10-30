package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"vigia/internal/probe"
)

const maxTargets = 500

//go:embed web/*
var webFiles embed.FS

type targetRequest struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type certificate struct {
	probe.Result
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Status    string    `json:"status"`
	ScannedAt time.Time `json:"scanned_at"`
}

type apiError struct {
	Error string `json:"error"`
}

type handler struct {
	mu      sync.RWMutex
	results map[string]certificate
	probes  chan struct{}
	timeout time.Duration
}

func NewHandler(timeout time.Duration, workers int) http.Handler {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if workers < 1 {
		workers = 1
	}

	h := &handler{
		results: make(map[string]certificate),
		probes:  make(chan struct{}, workers),
		timeout: timeout,
	}
	assets, _ := fs.Sub(webFiles, "web")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", h.health)
	mux.HandleFunc("GET /api/certificates", h.listCertificates)
	mux.HandleFunc("POST /api/check", h.checkTarget)
	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "ruta de API inexistente")
	}))
	mux.Handle("/", http.FileServer(http.FS(assets)))
	return securityHeaders(mux)
}

func NewHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) listCertificates(w http.ResponseWriter, _ *http.Request) {
	h.mu.RLock()
	results := make([]certificate, 0, len(h.results))
	for _, item := range h.results {
		results = append(results, item)
	}
	h.mu.RUnlock()
	sort.Slice(results, func(i, j int) bool {
		if (results[i].Error != "") != (results[j].Error != "") {
			return results[i].Error != ""
		}
		return results[i].DaysLeft < results[j].DaysLeft
	})
	writeJSON(w, http.StatusOK, map[string]any{"certificates": results})
}

func (h *handler) checkTarget(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "origen no permitido")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "se requiere application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request targetRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "petición JSON no válida")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "la petición debe contener un solo objeto JSON")
		return
	}
	host, err := normalizeHost(request.Host)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.Port < 1 || request.Port > 65535 {
		writeError(w, http.StatusBadRequest, "el puerto debe estar entre 1 y 65535")
		return
	}
	target := net.JoinHostPort(host, strconv.Itoa(request.Port))

	h.mu.RLock()
	_, exists := h.results[target]
	full := !exists && len(h.results) >= maxTargets
	h.mu.RUnlock()
	if full {
		writeError(w, http.StatusInsufficientStorage, "el inventario local alcanzó el límite de objetivos")
		return
	}
	select {
	case h.probes <- struct{}{}:
		defer func() { <-h.probes }()
	default:
		writeError(w, http.StatusTooManyRequests, "hay demasiados sondeos en curso; vuelve a intentarlo")
		return
	}

	result := probe.Probe(r.Context(), target, h.timeout, time.Now())
	item := certificate{
		Result:    result,
		Host:      host,
		Port:      request.Port,
		Status:    resultStatus(result),
		ScannedAt: time.Now().UTC(),
	}
	h.mu.Lock()
	if _, exists := h.results[target]; !exists && len(h.results) >= maxTargets {
		h.mu.Unlock()
		writeError(w, http.StatusInsufficientStorage, "el inventario local alcanzó el límite de objetivos")
		return
	}
	h.results[target] = item
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, item)
}

func normalizeHost(value string) (string, error) {
	host := strings.TrimSpace(value)
	if len(host) >= 2 && strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}
	if host == "" {
		return "", fmt.Errorf("el host es obligatorio")
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), nil
	}
	if len(host) > 253 || strings.ContainsAny(host, ":/\\@?#% \t\r\n") {
		return "", fmt.Errorf("indica un nombre de host o una dirección IP, no una URL")
	}
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return "", fmt.Errorf("el nombre de host no es válido")
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", fmt.Errorf("el nombre de host no es válido")
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
				return "", fmt.Errorf("el nombre de host contiene caracteres no permitidos")
			}
		}
	}
	return strings.ToLower(host), nil
}

func resultStatus(result probe.Result) string {
	if result.Error != "" {
		return "error"
	}
	switch {
	case result.DaysLeft < 0 || result.DaysLeft < 7:
		return "critical"
	case result.DaysLeft < 30:
		return "warning"
	default:
		return "healthy"
	}
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Scheme == "http" && parsed.Host == r.Host && parsed.User == nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; font-src 'self'; img-src 'self' data:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, apiError{Error: message})
}
