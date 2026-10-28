package probe

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"time"
)

type Result struct {
	Target   string    `json:"target"`
	NotAfter time.Time `json:"not_after,omitempty"`
	DaysLeft int       `json:"days_left"`
	KeyInfo  string    `json:"key_info,omitempty"`
	Issuer   string    `json:"issuer,omitempty"`
	Verify   string    `json:"verification,omitempty"`
	Error    string    `json:"error,omitempty"`
	Err      error     `json:"-"`
}

func NormalizeTarget(target string) (host, addr string) {
	if h, _, err := net.SplitHostPort(target); err == nil {
		return h, target
	}
	if net.ParseIP(target) != nil {
		return target, net.JoinHostPort(target, "443")
	}
	return target, net.JoinHostPort(target, "443")
}

func Probe(ctx context.Context, target string, timeout time.Duration, currentTime time.Time) Result {
	host, addr := NormalizeTarget(target)
	result := Result{Target: addr}
	dialContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: timeout},
		Config: &tls.Config{
			ServerName: host,
			// Se omite la validación del handshake para poder inspeccionar certificados caducados o inválidos.
			// La cadena y el nombre se validan manualmente después de recibirlos.
			InsecureSkipVerify: true,
		},
	}
	conn, err := dialer.DialContext(dialContext, "tcp", addr)
	if err != nil {
		return failed(result, err)
	}
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return failed(result, fmt.Errorf("la conexión no es TLS"))
	}
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return failed(result, fmt.Errorf("el servidor no presentó ningún certificado"))
	}

	leaf := certs[0]
	result.NotAfter = leaf.NotAfter
	result.DaysLeft = int(math.Floor(leaf.NotAfter.Sub(currentTime).Hours() / 24))
	result.KeyInfo = keyInfo(leaf)
	result.Issuer = issuerName(leaf)
	result.Verify = verify(leaf, certs[1:], host, currentTime)
	return result
}

func ProbeAll(ctx context.Context, targets []string, workers int, timeout time.Duration, currentTime time.Time) []Result {
	results := make([]Result, len(targets))
	if workers < 1 {
		workers = 1
	}
	semaphore := make(chan struct{}, workers)
	var wait sync.WaitGroup
	for i, target := range targets {
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			_, addr := NormalizeTarget(target)
			results[i] = failed(Result{Target: addr}, ctx.Err())
			continue
		}
		wait.Add(1)
		go func(index int, target string) {
			defer wait.Done()
			defer func() { <-semaphore }()
			results[index] = Probe(ctx, target, timeout, currentTime)
		}(i, target)
	}
	wait.Wait()
	return results
}

func failed(result Result, err error) Result {
	result.Err = err
	result.Error = shorten(err.Error())
	return result
}

func verify(leaf *x509.Certificate, chain []*x509.Certificate, host string, currentTime time.Time) string {
	intermediates := x509.NewCertPool()
	for _, cert := range chain {
		intermediates.AddCert(cert)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{
		Intermediates: intermediates,
		CurrentTime:   currentTime,
	}); err != nil {
		return "cadena: " + shorten(err.Error())
	}
	if err := leaf.VerifyHostname(host); err != nil {
		return "nombre: " + shorten(err.Error())
	}
	return "ok"
}

func keyInfo(cert *x509.Certificate) string {
	switch key := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		return fmt.Sprintf("RSA %d", key.N.BitLen())
	case *ecdsa.PublicKey:
		return "ECDSA " + key.Curve.Params().Name
	case ed25519.PublicKey:
		return "Ed25519"
	default:
		return cert.PublicKeyAlgorithm.String()
	}
}

func issuerName(cert *x509.Certificate) string {
	if cert.Issuer.CommonName != "" {
		return cert.Issuer.CommonName
	}
	if len(cert.Issuer.Organization) > 0 {
		return cert.Issuer.Organization[0]
	}
	return ""
}

func shorten(value string) string {
	if len(value) > 60 {
		return value[:57] + "..."
	}
	return strings.TrimSpace(value)
}
