// Package main provides the Leamas CLI.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
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
	cmd, err := parseWitnessCommand(os.Args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		printWitnessUsage()
		os.Exit(1)
	}

	switch cmd.Name {
	case "proxy":
		handleWitnessProxy()
	case "run-bundle":
		handleWitnessRunBundle()
	case "claim":
		handleWitnessClaim()
	case "evidence":
		handleWitnessEvidence()
	}
}

// witnessCommand represents the parsed witness subcommand.
type witnessCommand struct {
	Name string // "proxy", "run-bundle", "claim", "evidence"
}

// parseWitnessCommand parses the witness subcommand from args.
// Returns the parsed command or an error if missing/unknown.
func parseWitnessCommand(args []string) (witnessCommand, error) {
	if len(args) < 1 {
		return witnessCommand{}, errors.New("missing witness command")
	}

	cmd := args[0]
	switch cmd {
	case "proxy", "run-bundle", "claim", "evidence":
		return witnessCommand{Name: cmd}, nil
	default:
		return witnessCommand{}, fmt.Errorf("unknown witness command: %s", cmd)
	}
}

// runWitness dispatches witness subcommands with dependency injection.
func runWitness(args []string, stdout, stderr io.Writer, deps witnessDeps) int {
	if len(args) < 1 {
		printWitnessUsageTo(stderr)
		return 1
	}

	switch args[0] {
	case "proxy":
		// Proxy requires full server setup, not suitable for DI in unit tests
		return 1 // Fall through to error
	case "run-bundle":
		return runWitnessRunBundle(args[1:])
	case "claim":
		return runWitnessClaimWithDeps(args[1:], stdout, stderr, deps)
	case "evidence":
		return runWitnessEvidence(args[1:])
	case "--help", "-h":
		printWitnessUsageTo(stderr)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown witness command: %s\n", args[0])
		printWitnessUsageTo(stderr)
		return 1
	}
}

// witnessDeps holds injectable dependencies for witness commands.
type witnessDeps struct {
	ListClaims     func(opts claimListOptions) (string, int, error)
	ShowClaim      func(opts claimShowOptions) (string, int, error)
	ListEvidence   func(opts evidenceListOptions) (string, int, error)
	ShowEvidence   func(opts evidenceShowOptions) (string, int, error)
	AttachEvidence func(opts claimAttachEvidenceOptions) (string, int, error)
}

// runWitnessClaimWithDeps runs claim subcommands with injectable dependencies.
func runWitnessClaimWithDeps(args []string, stdout, stderr io.Writer, deps witnessDeps) int {
	if len(args) < 1 {
		printClaimUsageTo(stderr)
		return 1
	}

	switch args[0] {
	case "create":
		return runWitnessClaimCreate(args[1:])
	case "list":
		if deps.ListClaims != nil {
			var opts claimListOptions
			fs := flag.NewFlagSet("list", flag.ContinueOnError)
			fs.SetOutput(stderr)
			fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
			fs.StringVar(&opts.RunID, "run-id", "", "run bundle ID (required)")
			fs.BoolVar(&opts.JSON, "json", false, "output JSON")
			if err := fs.Parse(args[1:]); err != nil {
				return 1
			}
			output, code, _ := deps.ListClaims(opts)
			fmt.Fprint(stdout, output)
			return code
		}
		return runWitnessClaimList(args[1:])
	case "show":
		if deps.ShowClaim != nil {
			var opts claimShowOptions
			fs := flag.NewFlagSet("show", flag.ContinueOnError)
			fs.SetOutput(stderr)
			fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
			fs.StringVar(&opts.RunID, "run-id", "", "run bundle ID (required)")
			fs.BoolVar(&opts.JSON, "json", false, "output JSON")
			if err := fs.Parse(args[1:]); err != nil {
				return 1
			}
			output, code, _ := deps.ShowClaim(opts)
			fmt.Fprint(stdout, output)
			return code
		}
		return runWitnessClaimShow(args[1:])
	case "attach-evidence":
		if deps.AttachEvidence != nil {
			var opts claimAttachEvidenceOptions
			fs := flag.NewFlagSet("attach-evidence", flag.ContinueOnError)
			fs.SetOutput(stderr)
			fs.StringVar(&opts.Root, "root", defaultRunBundleRoot, "root directory for run bundles")
			fs.StringVar(&opts.RunID, "run-id", "", "run bundle ID (required)")
			fs.StringVar(&opts.ClaimID, "claim-id", "", "claim ID (required)")
			fs.StringVar(&opts.EvidenceID, "evidence-id", "", "evidence ID (required)")
			fs.BoolVar(&opts.JSON, "json", false, "output JSON")
			if err := fs.Parse(args[1:]); err != nil {
				return 1
			}
			output, code, _ := deps.AttachEvidence(opts)
			fmt.Fprint(stdout, output)
			return code
		}
		return runWitnessClaimAttachEvidence(args[1:])
	case "--help", "-h":
		printClaimUsageTo(stderr)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown claim subcommand: %s\n", args[0])
		printClaimUsageTo(stderr)
		return 1
	}
}

func printWitnessUsage() {
	printWitnessUsageTo(os.Stdout)
}

func printWitnessUsageTo(w io.Writer) {
	fmt.Fprintln(w, "Witness commands:")
	fmt.Fprintln(w, "  leamas witness proxy [flags]   Start local witness proxy")
	fmt.Fprintln(w, "  leamas witness run-bundle       Manage local run bundles")
	fmt.Fprintln(w, "  leamas witness claim            Manage claims")
	fmt.Fprintln(w, "  leamas witness evidence         Manage evidence")
	fmt.Fprintln(w)
	printWitnessProxyUsageTo(w)
}

func printWitnessProxyUsage() {
	printWitnessProxyUsageTo(os.Stdout)
}

func printWitnessProxyUsageTo(w io.Writer) {
	fmt.Fprintln(w, "Usage: leamas witness proxy [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --upstream <url>        Upstream target URL (required)")
	fmt.Fprintln(w, "  --listen <addr>         Listen address (default: 127.0.0.1:0)")
	fmt.Fprintln(w, "  --max-records <n>       Maximum records to retain (default: 100)")
	fmt.Fprintln(w, "  --capture-headers       Enable header capture (default: disabled)")
	fmt.Fprintln(w, "  --help, -h             Show this help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  leamas witness proxy --upstream http://127.0.0.1:8080")
	fmt.Fprintln(w, "  leamas witness proxy --listen 127.0.0.1:8766 --upstream http://127.0.0.1:8080")
	fmt.Fprintln(w, "  leamas witness proxy --upstream http://localhost:8080 --capture-headers --max-records 250")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The witness proxy is local-only. Only loopback addresses are allowed.")
	fmt.Fprintln(w, "Headers are not captured by default. Bodies are never captured.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Press Ctrl-C to stop.")
}

// handleWitnessClaim dispatches claim subcommands from os.Args.
func handleWitnessClaim() {
	if len(os.Args) < 4 {
		printClaimUsage()
		os.Exit(1)
	}
	os.Exit(runWitnessClaim(os.Args[3:]))
}

// handleWitnessEvidence dispatches evidence subcommands from os.Args.
func handleWitnessEvidence() {
	if len(os.Args) < 4 {
		printEvidenceUsage()
		os.Exit(1)
	}
	os.Exit(runWitnessEvidence(os.Args[3:]))
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
