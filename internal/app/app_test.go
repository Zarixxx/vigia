package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(version) = %d, salida de error: %s", code, stderr.String())
	}
	if got, want := stdout.String(), "vigia "+Version+"\n"; got != want {
		t.Fatalf("salida = %q, se esperaba %q", got, want)
	}
}

func TestReadTargets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts.txt")
	content := "# servidor público\n\n ejemplo.test:8443 \n# final\nlocalhost\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	targets, err := ReadTargets(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(targets, ","), "ejemplo.test:8443,localhost"; got != want {
		t.Fatalf("ReadTargets() = %q, want %q", got, want)
	}
}

func TestRunRejectsMissingTargets(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"check"}, &stdout, &stderr); code != 3 {
		t.Fatalf("Run(check) = %d, want 3; salida: %s", code, stderr.String())
	}
}

func TestDashboardHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"dashboard", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(dashboard --help) = %d, salida de error: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--port") {
		t.Fatalf("la ayuda del dashboard no describe --port: %s", stderr.String())
	}
}

func TestDashboardRejectsInvalidPort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"dashboard", "--port", "65536"}, &stdout, &stderr); code != 3 {
		t.Fatalf("Run(dashboard --port 65536) = %d, want 3", code)
	}
}
