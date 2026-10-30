package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDashboardAssetsAreServed(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(time.Second, 2).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), "Vigía") {
		t.Fatal("GET / no contiene la interfaz de Vigía")
	}
}

func TestCheckEndpointProbesLocalTLSAndListsResult(t *testing.T) {
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	tlsServer.CloseClientConnections()
	defer tlsServer.Close()
	address := strings.TrimPrefix(tlsServer.URL, "https://")
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatal(err)
	}
	var port int
	if _, err := fmt.Sscan(portText, &port); err != nil {
		t.Fatal(err)
	}
	localHandler := NewHandler(2*time.Second, 1)
	apiServer := httptest.NewServer(localHandler)
	defer apiServer.Close()

	body := fmt.Sprintf(`{"host":%q,"port":%d}`, host, port)
	response, err := http.Post(apiServer.URL+"/api/check", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/check status = %d; body: %s", response.StatusCode, readBody(t, response))
	}
	var result certificate
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Target != address || result.ScannedAt.IsZero() || result.Status == "" {
		t.Fatalf("respuesta de sondeo incompleta: %+v", result)
	}

	listed, err := http.Get(apiServer.URL + "/api/certificates")
	if err != nil {
		t.Fatal(err)
	}
	defer listed.Body.Close()
	var inventory struct {
		Certificates []certificate `json:"certificates"`
	}
	if err := json.NewDecoder(listed.Body).Decode(&inventory); err != nil {
		t.Fatal(err)
	}
	if len(inventory.Certificates) != 1 || inventory.Certificates[0].Target != address {
		t.Fatalf("inventario inesperado: %+v", inventory.Certificates)
	}
}

func TestCheckEndpointRejectsURLsAndInvalidPorts(t *testing.T) {
	handler := NewHandler(time.Second, 1)
	for _, body := range []string{
		`{"host":"https://example.test","port":443}`,
		`{"host":"example.test:443","port":443}`,
		`{"host":"example.test","port":70000}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/api/check", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Errorf("POST con %s status = %d, want 400", body, response.Code)
		}
	}
}

func TestCheckEndpointRejectsCrossOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/check", strings.NewReader(`{"host":"localhost","port":443}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	NewHandler(time.Second, 1).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("POST cruzado status = %d, want 403", response.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	response := httptest.NewRecorder()
	NewHandler(time.Second, 1).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"ok"`) {
		t.Fatalf("respuesta de healthz inesperada: %d %s", response.Code, response.Body.String())
	}
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
