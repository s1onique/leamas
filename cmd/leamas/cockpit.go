package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/s1onique/leamas/internal/web/cockpit"
)

// handleCockpit handles the cockpit command group.
func handleCockpit() {
	if len(os.Args) < 3 {
		printCockpitUsage()
		os.Exit(1)
	}

	switch os.Args[2] {
	case "serve":
		handleCockpitServe()
	case "--help", "-h":
		printCockpitUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown cockpit command: %s\n", os.Args[2])
		printCockpitUsage()
		os.Exit(1)
	}
}

func printCockpitUsage() {
	fmt.Println("Cockpit commands:")
	fmt.Println("  leamas cockpit serve [flags]   Start local web cockpit")
	fmt.Println()
	printCockpitServeUsage()
}

func printCockpitServeUsage() {
	fmt.Println("Usage: leamas cockpit serve [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --listen <addr>   Listen address (default: 127.0.0.1:0)")
	fmt.Println()
	fmt.Println("The cockpit is local-only with no authentication.")
	fmt.Println("Only loopback addresses are allowed (127.0.0.1, localhost).")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  leamas cockpit serve              # Random port on 127.0.0.1")
	fmt.Println("  leamas cockpit serve --listen 127.0.0.1:8080")
	fmt.Println()
	fmt.Println("Press Ctrl-C to stop.")
}

// handleCockpitServe starts the cockpit server.
func handleCockpitServe() {
	config, err := parseCockpitServeArgs(os.Args[3:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		printCockpitServeUsage()
		os.Exit(1)
	}

	c, err := cockpit.New(cockpit.Config{ListenAddr: config.ListenAddr})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Bind listener to get actual port when using :0
	ln, err := net.Listen("tcp", c.ListenAddr())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to bind listener: %v\n", err)
		os.Exit(1)
	}

	addr := ln.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("http://127.0.0.1:%d", addr.Port)
	fmt.Printf("Leamas cockpit listening on %s\n", url)
	fmt.Println("Press Ctrl-C to stop.")

	server := &http.Server{
		Handler: c.Handler(),
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

// cockpitServeConfig holds parsed configuration for the serve subcommand.
type cockpitServeConfig struct {
	ListenAddr string
}

// parseCockpitServeArgs parses command-line arguments for the serve subcommand.
// Returns the parsed configuration or an error.
func parseCockpitServeArgs(args []string) (cockpitServeConfig, error) {
	cfg := cockpitServeConfig{
		ListenAddr: "127.0.0.1:0", // default
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--listen":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--listen requires an address argument")
			}
			cfg.ListenAddr = args[i+1]
			i++
		case "--help", "-h":
			printCockpitServeUsage()
			os.Exit(0)
		default:
			return cfg, fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	// Validate loopback-only constraint
	if !isLoopbackAddr(cfg.ListenAddr) {
		return cfg, fmt.Errorf("unsafe listen address: %s (only loopback allowed: 127.0.0.1, localhost)", cfg.ListenAddr)
	}

	return cfg, nil
}

// isLoopbackAddr checks if the address is a loopback-only address.
// Allows: 127.0.0.1:*, localhost:*
// Rejects: 0.0.0.0:*, [::]:*, :*, etc.
func isLoopbackAddr(addr string) bool {
	// Handle localhost specially
	host, _, err := splitHostPort(addr)
	if err != nil {
		return false
	}

	if host == "localhost" || host == "localhost." {
		return true
	}

	// Check for explicit loopback
	if strings.HasPrefix(host, "127.") {
		return true
	}

	// Reject anything else
	return false
}

// splitHostPort is a simple host:port splitter.
func splitHostPort(addr string) (host, port string, err error) {
	// Handle IPv6 brackets
	if len(addr) >= 2 && addr[0] == '[' {
		// IPv6 format: [::1]:8080
		end := -1
		for i := len(addr) - 1; i >= 0; i-- {
			if addr[i] == ']' {
				end = i
				break
			}
		}
		if end == -1 {
			return "", "", fmt.Errorf("invalid IPv6 address: %s", addr)
		}
		host = addr[1:end]
		if end+1 < len(addr) && addr[end+1] == ':' {
			port = addr[end+2:]
		}
		return host, port, nil
	}

	// IPv4 or hostname format
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			// Only split on the last colon (port separator)
			return addr[:i], addr[i+1:], nil
		}
	}
	return addr, "", nil
}
