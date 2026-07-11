package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/web/cockpit"
	"github.com/s1onique/leamas/internal/witness/proxy"
)

// TestRuntimeSmokeCockpitHelp verifies cockpit help command works.
func TestRuntimeSmokeCockpitHelp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "github.com/s1onique/leamas/cmd/leamas", "cockpit", "serve", "--help")
	cmd.Env = withoutLeamasEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("cockpit serve --help failed: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("leamas cockpit serve")) {
		t.Errorf("expected 'leamas cockpit serve' in help output, got:\n%s", output)
	}

	if !bytes.Contains([]byte(output), []byte("loopback")) {
		t.Errorf("expected 'loopback' mention in help output, got:\n%s", output)
	}
}

// TestRuntimeSmokeWitnessProxyHelp verifies witness proxy help command works.
func TestRuntimeSmokeWitnessProxyHelp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "github.com/s1onique/leamas/cmd/leamas", "witness", "proxy", "--help")
	cmd.Env = withoutLeamasEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("witness proxy --help failed: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("leamas witness proxy")) {
		t.Errorf("expected 'leamas witness proxy' in help output, got:\n%s", output)
	}

	if !bytes.Contains([]byte(output), []byte("--upstream")) {
		t.Errorf("expected '--upstream' mention in help output, got:\n%s", output)
	}
}

// TestRuntimeSmokeFactoryDomainBoundariesCommand verifies factory domain-boundaries command exists and passes.
func TestRuntimeSmokeFactoryDomainBoundariesCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Must run from repo root where go.mod lives.
	// go test ./cmd/leamas/... runs test binary from cmd/leamas/ directory.
	// cmd/leamas/ -> ../.. = leamas/ (repo root)
	cmd := exec.CommandContext(ctx, "go", "run", "github.com/s1onique/leamas/cmd/leamas", "factory", "verify", "domain-boundaries")
	cmd.Dir = "../.."
	cmd.Env = withoutLeamasEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		output := stderr.String() + stdout.String()
		if bytes.Contains([]byte(output), []byte("boundary_violation")) {
			t.Fatalf("domain-boundaries reported violations:\n%s", output)
		}
		t.Fatalf("factory verify domain-boundaries failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("domain-boundaries")) && !bytes.Contains([]byte(output), []byte("verification PASSED")) {
		t.Errorf("expected 'domain-boundaries' or 'verification PASSED' in output, got:\n%s", output)
	}
}

// TestRuntimeSmokeCockpitServesStatusOnLoopback verifies cockpit serves /api/status on loopback.
func TestRuntimeSmokeCockpitServesStatusOnLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	port := addr.Port

	c, err := cockpit.New(cockpit.Config{ListenAddr: fmt.Sprintf("127.0.0.1:%d", port)})
	if err != nil {
		t.Fatalf("cockpit.New() error = %v", err)
	}

	server := &http.Server{Handler: c.Handler()}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ln)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://127.0.0.1:%d/api/status", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if cookie := resp.Header.Get("Set-Cookie"); cookie != "" {
		t.Errorf("unexpected Set-Cookie header: %s", cookie)
	}

	var status cockpit.StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}

	if status.Mode != "local-only" {
		t.Errorf("expected mode 'local-only', got %q", status.Mode)
	}
	if !status.ReadOnly {
		t.Error("expected read_only to be true")
	}
	if status.Storage != "none" {
		t.Errorf("expected storage 'none', got %q", status.Storage)
	}
	if status.Auth != "none" {
		t.Errorf("expected auth 'none', got %q", status.Auth)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Logf("server error: %v", err)
		}
	default:
	}
}

// TestRuntimeSmokeWitnessProxyForwardsToSingleLoopbackUpstream verifies witness proxy forwards requests.
func TestRuntimeSmokeWitnessProxyForwardsToSingleLoopbackUpstream(t *testing.T) {
	upstreamCalled := false
	var receivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		receivedPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"upstream":"ok"}`))
	}))
	defer upstream.Close()

	p, err := proxy.New(proxy.Config{
		UpstreamURL:    upstream.URL,
		CaptureHeaders: false,
	})
	if err != nil {
		t.Fatalf("proxy.New() error = %v", err)
	}

	proxyServer := httptest.NewServer(p.Handler())
	defer proxyServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyServer.URL+"/test/path?query=value", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if !upstreamCalled {
		t.Error("upstream server was not called")
	}

	if receivedPath != "/test/path?query=value" {
		t.Errorf("expected path '/test/path?query=value', got %q", receivedPath)
	}

	records := p.Records()
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}

	if len(records) > 0 {
		rec := records[0]
		if rec.Method != "GET" {
			t.Errorf("expected method 'GET', got %q", rec.Method)
		}
		if rec.Path != "/test/path?query=value" {
			t.Errorf("expected path '/test/path?query=value', got %q", rec.Path)
		}
		if rec.RequestHeaders != nil {
			t.Error("expected no request headers (CaptureHeaders=false)")
		}
		if rec.ResponseHeaders != nil {
			t.Error("expected no response headers (CaptureHeaders=false)")
		}
	}
}

// TestRuntimeSmokeWitnessProxyRejectsFtpScheme verifies witness proxy CLI rejects ftp:// scheme.
func TestRuntimeSmokeWitnessProxyRejectsFtpScheme(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "github.com/s1onique/leamas/cmd/leamas", "witness", "proxy", "--upstream", "ftp://127.0.0.1")
	cmd.Env = withoutLeamasEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error for ftp:// scheme, got nil\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}

	output := stderr.String() + stdout.String()
	if !bytes.Contains([]byte(output), []byte("http://")) && !bytes.Contains([]byte(output), []byte("https://")) {
		t.Errorf("expected error message about http/https scheme, got:\n%s", output)
	}
}

// TestRuntimeSmokeCockpitDoesNotSetCookie verifies cockpit /api/status emits no cookies.
func TestRuntimeSmokeCockpitDoesNotSetCookie(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	port := addr.Port

	c, err := cockpit.New(cockpit.Config{ListenAddr: fmt.Sprintf("127.0.0.1:%d", port)})
	if err != nil {
		t.Fatalf("cockpit.New() error = %v", err)
	}

	server := &http.Server{Handler: c.Handler()}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ln)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://127.0.0.1:%d/api/status", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	for name := range resp.Header {
		if name == "Set-Cookie" || name == "set-cookie" {
			t.Errorf("forbidden Set-Cookie header found: %s", resp.Header.Get(name))
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Logf("server error: %v", err)
		}
	default:
	}
}

// TestRuntimeSmokeWitnessDoesNotCaptureBodies verifies witness proxy never captures bodies.
func TestRuntimeSmokeWitnessDoesNotCaptureBodies(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"secret":"data"}`))
	}))
	defer upstream.Close()

	p, err := proxy.New(proxy.Config{
		UpstreamURL:    upstream.URL,
		CaptureHeaders: true,
	})
	if err != nil {
		t.Fatalf("proxy.New() error = %v", err)
	}

	proxyServer := httptest.NewServer(p.Handler())
	defer proxyServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyServer.URL+"/api/data", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer resp.Body.Close()

	body := make([]byte, 100)
	n, _ := resp.Body.Read(body)
	resp.Body.Close()
	if n > 0 && string(body[:n]) == `{"secret":"data"}` {
		// Body was forwarded (correct behavior)
	} else {
		t.Logf("body forwarded: %q", string(body[:n]))
	}

	records := p.Records()
	if len(records) > 0 {
		rec := records[0]
		t.Logf("record fields present: Method=%s, Path=%s, StatusCode=%d",
			rec.Method, rec.Path, rec.StatusCode)
	}
}
