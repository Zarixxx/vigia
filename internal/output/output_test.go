package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"vigia/internal/probe"
)

func TestWriteJSON(t *testing.T) {
	var output bytes.Buffer
	results := []probe.Result{{
		Target:   "servidor.example:443",
		NotAfter: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
		DaysLeft: 5,
		Verify:   "ok",
	}}
	if err := Write(&output, "json", results, Options{WarnDays: 30, CritDays: 7}); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"state": "CRÍTICO"`, `"target": "servidor.example:443"`, `"days_left": 5`} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("JSON no contiene %q: %s", expected, output.String())
		}
	}
}

func TestWriteTable(t *testing.T) {
	var output bytes.Buffer
	results := []probe.Result{{
		Target:   "servidor.example:443",
		NotAfter: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
		DaysLeft: 5,
		KeyInfo:  "ECDSA P-256",
		Issuer:   "CA de prueba",
		Verify:   "ok",
	}}
	if err := Write(&output, "table", results, Options{WarnDays: 30, CritDays: 7}); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"ESTADO", "CRÍTICO", "servidor.example:443", "ECDSA P-256", "CA de prueba"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("tabla no contiene %q: %s", expected, output.String())
		}
	}
}

func TestWriteRejectsUnknownFormat(t *testing.T) {
	if err := Write(&bytes.Buffer{}, "csv", nil, Options{}); err == nil {
		t.Fatal("Write() error = nil, se esperaba formato no admitido")
	}
}
