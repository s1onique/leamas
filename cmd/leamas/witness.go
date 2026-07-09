// Package main provides the Leamas CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/s1onique/leamas/internal/witness/proxy"
)

// handleWitness handles the witness command group.
func handleWitness() {
	if len(os.Args) < 3 {
		printWitnessUsage()
		os.Exit(1)
	}

	switch os.Args[2] {
	case "proxy":
		handleWitnessProxy()
	case "run-bundle":
		handleWitnessRunBundle()
	case "--help", "-h":
		printWitnessUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown witness command: %s\n", os.Args[2])
		printWitnessUsage()
		os.Exit(1)
	}
}

func printWitnessUsage() {
	fmt.Println("Witness commands:")
	fmt.Println("  leamas witness proxy [flags]   Start local witness proxy")
	fmt.Println("  leamas witness run-bundle       Manage local run bundles")
	fmt.Println()
	printWitnessProxyUsage()
}

func printWitnessProxyUsage() {
	fmt.Println("Usage: leamas witness proxy [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --upstream <url>        Upstream target URL (required)")
	fmt.Println("  --listen <addr>         Listen address (default: 127.0.0.1:0)")
	fmt.Println("  --max-records <n>       Maximum records to retain (default: 100)")
	fmt.Println("  --capture-headers       Enable header capture (default: disabled)")
	fmt.Println("  --help, -h             Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  leamas witness proxy --upstream http://127.0.0.1:8080")
	fmt.Println("  leamas witness proxy --listen 127.0.0.1:8766 --upstream http://127.0.0.1:8080")
	fmt.Println("  leamas witness proxy --upstream http://localhost:8080 --capture-headers --max-records 250")
	fmt.Println()
	fmt.Println("The witness proxy is local-only. Only loopback addresses are allowed.")
	fmt.Println("Headers are not captured by default. Bodies are never captured.")
	fmt.Println()
	fmt.Println("Press Ctrl-C to stop.")
}

// handleWitnessProxy starts the witness proxy server.
func handleWitnessProxy() {
	config, err := parseWitnessProxyArgs(os.Args[3:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		printWitnessProxyUsage()
		os.Exit(1)
	}

	p, err := proxy.New(proxy.Config{
		ListenAddr:     config.ListenAddr,
		UpstreamURL:    config.UpstreamURL,
		MaxRecords:     config.MaxRecords,
		CaptureHeaders: config.CaptureHeaders,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Bind listener to get actual port when using :0
	ln, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to bind listener: %v\n", err)
		os.Exit(1)
	}

	addr := ln.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("http://127.0.0.1:%d", addr.Port)

	captureStr := "false"
	if config.CaptureHeaders {
		captureStr = "true"
	}

	fmt.Printf("Leamas witness proxy listening on %s\n", url)
	fmt.Printf("Upstream: %s\n", config.UpstreamURL)
	fmt.Printf("Capture headers: %s\n", captureStr)
	fmt.Println("Press Ctrl-C to stop.")

	server := &http.Server{
		Handler: p.Handler(),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ln)
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		return
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		fmt.Fprintf(os.Stderr, "ERROR: server error: %v\n", err)
		os.Exit(1)
	}
}

// witnessProxyConfig holds parsed configuration for the proxy subcommand.
type witnessProxyConfig struct {
	ListenAddr     string
	UpstreamURL    string
	MaxRecords     int
	CaptureHeaders bool
}

// parseWitnessProxyArgs parses command-line arguments for the witness proxy subcommand.
// Returns the parsed configuration or an error.
func parseWitnessProxyArgs(args []string) (witnessProxyConfig, error) {
	cfg := witnessProxyConfig{
		ListenAddr: "127.0.0.1:0", // default
		MaxRecords: 100,           // default
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--upstream":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--upstream requires a URL argument")
			}
			cfg.UpstreamURL = args[i+1]
			i++
		case "--listen":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--listen requires an address argument")
			}
			cfg.ListenAddr = args[i+1]
			i++
		case "--max-records":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--max-records requires a number argument")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return cfg, fmt.Errorf("--max-records must be an integer: %s", args[i+1])
			}
			if n < 0 {
				return cfg, fmt.Errorf("--max-records must be non-negative")
			}
			cfg.MaxRecords = n
			i++
		case "--capture-headers":
			cfg.CaptureHeaders = true
		case "--help", "-h":
			printWitnessProxyUsage()
			os.Exit(0)
		default:
			return cfg, fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	// Validate required --upstream
	if cfg.UpstreamURL == "" {
		return cfg, fmt.Errorf("--upstream is required")
	}

	// Validate upstream URL has a scheme
	if !strings.HasPrefix(cfg.UpstreamURL, "http://") && !strings.HasPrefix(cfg.UpstreamURL, "https://") {
		return cfg, fmt.Errorf("--upstream must start with http:// or https://")
	}

	// Validate loopback-only constraint for listen address
	if !isLoopbackAddr(cfg.ListenAddr) {
		return cfg, fmt.Errorf("unsafe listen address: %s (only loopback allowed: 127.0.0.1, localhost)", cfg.ListenAddr)
	}

	return cfg, nil
}
