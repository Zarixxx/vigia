package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"vigia/internal/clock"
	"vigia/internal/output"
	"vigia/internal/probe"
)

const Version = "0.1.0"

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "version":
			if len(args) != 1 {
				fmt.Fprintln(stderr, "uso: vigia version")
				return 3
			}
			fmt.Fprintf(stdout, "vigia %s\n", Version)
			return 0
		case "help", "-h", "--help":
			writeHelp(stdout)
			return 0
		case "check":
			args = args[1:]
		}
	}

	flags := flag.NewFlagSet("vigia check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	warnDays := flags.Int("days", 30, "avisar si quedan menos de N días")
	critDays := flags.Int("crit", 7, "estado crítico si quedan menos de N días")
	timeout := flags.Duration("timeout", 5*time.Second, "tiempo máximo por conexión")
	workers := flags.Int("workers", 20, "conexiones simultáneas")
	file := flags.String("f", "", "archivo con un objetivo por línea (# para comentarios)")
	format := flags.String("format", "table", "formato de salida: table o json")
	flags.Usage = func() { writeUsage(stderr, flags) }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 3
	}
	if *warnDays < 0 || *critDays < 0 || *timeout <= 0 || *workers < 1 {
		fmt.Fprintln(stderr, "error: los días deben ser no negativos, el tiempo debe ser positivo y workers debe ser al menos 1")
		return 3
	}
	if *format != "table" && *format != "json" {
		fmt.Fprintf(stderr, "error: formato no admitido: %s\n", *format)
		return 3
	}
	targets := append([]string(nil), flags.Args()...)
	if *file != "" {
		more, err := ReadTargets(*file)
		if err != nil {
			fmt.Fprintf(stderr, "error leyendo el archivo: %s\n", err)
			return 3
		}
		targets = append(targets, more...)
	}
	if len(targets) == 0 {
		fmt.Fprintln(stderr, "error: indica al menos un objetivo (usa --help para ver el uso)")
		return 3
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	results := probe.ProbeAll(ctx, targets, *workers, *timeout, clock.System{}.Now())
	if err := output.Write(stdout, *format, results, output.Options{WarnDays: *warnDays, CritDays: *critDays}); err != nil {
		fmt.Fprintf(stderr, "error al escribir la salida: %s\n", err)
		return 3
	}
	for _, result := range results {
		if result.Err != nil {
			return 2
		}
	}
	for _, result := range results {
		if result.DaysLeft < *warnDays {
			return 1
		}
	}
	return 0
}

func writeHelp(w io.Writer) {
	fmt.Fprintln(w, "Vigía: inventario de certificados TLS servidos por tus equipos.")
	fmt.Fprintln(w, "Uso: vigia check [opciones] host[:puerto] ...")
	fmt.Fprintln(w, "     vigia [opciones] host[:puerto] ...")
	fmt.Fprintln(w, "     vigia version")
	fmt.Fprintln(w, "Opciones:")
	flags := flag.NewFlagSet("vigia check", flag.ContinueOnError)
	flags.SetOutput(w)
	flags.Int("days", 30, "avisar si quedan menos de N días")
	flags.Int("crit", 7, "estado crítico si quedan menos de N días")
	flags.Duration("timeout", 5*time.Second, "tiempo máximo por conexión")
	flags.Int("workers", 20, "conexiones simultáneas")
	flags.String("f", "", "archivo con un objetivo por línea (# para comentarios)")
	flags.String("format", "table", "formato de salida: table o json")
	flags.PrintDefaults()
}

func writeUsage(w io.Writer, flags *flag.FlagSet) {
	fmt.Fprintln(w, "Uso: vigia check [opciones] host[:puerto] ...")
	fmt.Fprintln(w, "     vigia [opciones] host[:puerto] ...")
	flags.PrintDefaults()
}

func ReadTargets(path string) ([]string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var targets []string
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		targets = append(targets, line)
	}
	return targets, nil
}
