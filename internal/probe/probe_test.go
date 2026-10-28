package probe

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNormalizeTarget(t *testing.T) {
	tests := []struct {
		target string
		host   string
		addr   string
	}{
		{target: "vigia.example", host: "vigia.example", addr: "vigia.example:443"},
		{target: "vigia.example:8443", host: "vigia.example", addr: "vigia.example:8443"},
		{target: "127.0.0.1", host: "127.0.0.1", addr: "127.0.0.1:443"},
		{target: "[::1]:9443", host: "::1", addr: "[::1]:9443"},
	}
	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			host, addr := NormalizeTarget(test.target)
			if host != test.host || addr != test.addr {
				t.Fatalf("NormalizeTarget(%q) = (%q, %q), want (%q, %q)", test.target, host, addr, test.host, test.addr)
			}
		})
	}
}

func TestProbeReportsDaysLeftAndExpiredCertificates(t *testing.T) {
	currentTime := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name     string
		notAfter time.Time
		wantDays int
	}{
		{name: "cinco días", notAfter: currentTime.Add(5*24*time.Hour + 23*time.Hour), wantDays: 5},
		{name: "caducado", notAfter: currentTime.Add(-time.Hour), wantDays: -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := startTLSServer(t, test.notAfter)
			result := Probe(context.Background(), server.Listener.Addr().String(), time.Second, currentTime)
			if result.Err != nil {
				t.Fatalf("Probe() error = %v", result.Err)
			}
			if result.DaysLeft != test.wantDays {
				t.Fatalf("DaysLeft = %d, want %d", result.DaysLeft, test.wantDays)
			}
			if result.NotAfter.IsZero() || result.KeyInfo == "" {
				t.Fatalf("Probe() returned incomplete certificate information: %+v", result)
			}
		})
	}
}

func TestProbeClosedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	result := Probe(context.Background(), addr, time.Second, time.Now())
	if result.Err == nil {
		t.Fatal("Probe() error = nil, want connection error")
	}
}

func TestProbeAllPreservesInputOrder(t *testing.T) {
	currentTime := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	first := startTLSServer(t, currentTime.Add(5*24*time.Hour))
	second := startTLSServer(t, currentTime.Add(2*24*time.Hour))
	targets := []string{first.Listener.Addr().String(), second.Listener.Addr().String()}
	results := ProbeAll(context.Background(), targets, 2, time.Second, currentTime)
	if len(results) != len(targets) {
		t.Fatalf("got %d results, want %d", len(results), len(targets))
	}
	for i, target := range targets {
		if results[i].Target != target {
			t.Errorf("results[%d].Target = %q, want %q", i, results[i].Target, target)
		}
	}
}

func startTLSServer(t *testing.T, notAfter time.Time) *httptest.Server {
	t.Helper()
	currentTime := notAfter.Add(-7 * 24 * time.Hour)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             currentTime.Add(-time.Hour),
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	certificate := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: parsed}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}
