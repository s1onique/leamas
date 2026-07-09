package main

import (
	"testing"
)

func TestParseWitnessProxyArgs_Defaults(t *testing.T) {
	cfg, err := parseWitnessProxyArgs([]string{"--upstream", "http://127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("parseWitnessProxyArgs() error = %v", err)
	}

	if cfg.ListenAddr != "127.0.0.1:0" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, "127.0.0.1:0")
	}

	if cfg.MaxRecords != 100 {
		t.Errorf("MaxRecords = %d, want %d", cfg.MaxRecords, 100)
	}

	if cfg.CaptureHeaders != false {
		t.Errorf("CaptureHeaders = %v, want %v", cfg.CaptureHeaders, false)
	}
}

func TestParseWitnessProxyArgs_RequiresUpstream(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for missing --upstream")
	}
}

func TestParseWitnessProxyArgs_MissingUpstreamValue(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{"--upstream"})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for missing --upstream value")
	}
}

func TestParseWitnessProxyArgs_CustomListen(t *testing.T) {
	cfg, err := parseWitnessProxyArgs([]string{"--listen", "127.0.0.1:8765", "--upstream", "http://127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("parseWitnessProxyArgs() error = %v", err)
	}

	if cfg.ListenAddr != "127.0.0.1:8765" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, "127.0.0.1:8765")
	}
}

func TestParseWitnessProxyArgs_Localhost(t *testing.T) {
	cfg, err := parseWitnessProxyArgs([]string{"--listen", "localhost:8766", "--upstream", "http://127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("parseWitnessProxyArgs() error = %v", err)
	}

	if cfg.ListenAddr != "localhost:8766" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, "localhost:8766")
	}
}

func TestParseWitnessProxyArgs_Rejects0_0_0_0(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{"--listen", "0.0.0.0:8765", "--upstream", "http://127.0.0.1:8080"})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for 0.0.0.0")
	}
}

func TestParseWitnessProxyArgs_RejectsBarePort(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{"--listen", ":8765", "--upstream", "http://127.0.0.1:8080"})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for bare port")
	}
}

func TestParseWitnessProxyArgs_RejectsIPv6All(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{"--listen", "[::]:8765", "--upstream", "http://127.0.0.1:8080"})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for [::]")
	}
}

func TestParseWitnessProxyArgs_RejectsPrivateNetwork(t *testing.T) {
	privateAddrs := []string{
		"192.168.1.1:8080",
		"10.0.0.1:8080",
		"172.16.0.1:8080",
	}

	for _, addr := range privateAddrs {
		_, err := parseWitnessProxyArgs([]string{"--listen", addr, "--upstream", "http://127.0.0.1:8080"})
		if err == nil {
			t.Errorf("parseWitnessProxyArgs() expected error for private address %q", addr)
		}
	}
}

func TestParseWitnessProxyArgs_MissingListenArg(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{"--listen", "--upstream", "http://127.0.0.1:8080"})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for missing --listen argument")
	}
}

func TestParseWitnessProxyArgs_UnknownFlag(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{"--unknown", "--upstream", "http://127.0.0.1:8080"})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for unknown flag")
	}
}

func TestParseWitnessProxyArgs_MissingMaxRecordsValue(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{"--max-records", "--upstream", "http://127.0.0.1:8080"})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for missing --max-records value")
	}
}

func TestParseWitnessProxyArgs_NonIntegerMaxRecords(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{"--max-records", "abc", "--upstream", "http://127.0.0.1:8080"})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for non-integer --max-records")
	}
}

func TestParseWitnessProxyArgs_NegativeMaxRecords(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{"--max-records", "-1", "--upstream", "http://127.0.0.1:8080"})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for negative --max-records")
	}
}

func TestParseWitnessProxyArgs_ZeroMaxRecords(t *testing.T) {
	// Zero should be allowed (package default behavior)
	cfg, err := parseWitnessProxyArgs([]string{"--max-records", "0", "--upstream", "http://127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("parseWitnessProxyArgs() error = %v", err)
	}

	if cfg.MaxRecords != 0 {
		t.Errorf("MaxRecords = %d, want %d", cfg.MaxRecords, 0)
	}
}

func TestParseWitnessProxyArgs_CaptureHeaders(t *testing.T) {
	cfg, err := parseWitnessProxyArgs([]string{"--capture-headers", "--upstream", "http://127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("parseWitnessProxyArgs() error = %v", err)
	}

	if cfg.CaptureHeaders != true {
		t.Errorf("CaptureHeaders = %v, want %v", cfg.CaptureHeaders, true)
	}
}

func TestParseWitnessProxyArgs_UpstreamRequiresScheme(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{"--upstream", "127.0.0.1:8080"})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for upstream without scheme")
	}
}

func TestParseWitnessProxyArgs_HTTPScheme(t *testing.T) {
	cfg, err := parseWitnessProxyArgs([]string{"--upstream", "http://localhost:8080"})
	if err != nil {
		t.Fatalf("parseWitnessProxyArgs() error = %v", err)
	}

	if cfg.UpstreamURL != "http://localhost:8080" {
		t.Errorf("UpstreamURL = %q, want %q", cfg.UpstreamURL, "http://localhost:8080")
	}
}

func TestParseWitnessProxyArgs_HTTPSScheme(t *testing.T) {
	cfg, err := parseWitnessProxyArgs([]string{"--upstream", "https://localhost:8080"})
	if err != nil {
		t.Fatalf("parseWitnessProxyArgs() error = %v", err)
	}

	if cfg.UpstreamURL != "https://localhost:8080" {
		t.Errorf("UpstreamURL = %q, want %q", cfg.UpstreamURL, "https://localhost:8080")
	}
}

func TestParseWitnessProxyArgs_AllFlags(t *testing.T) {
	cfg, err := parseWitnessProxyArgs([]string{
		"--listen", "127.0.0.1:9000",
		"--upstream", "http://localhost:8080",
		"--max-records", "250",
		"--capture-headers",
	})
	if err != nil {
		t.Fatalf("parseWitnessProxyArgs() error = %v", err)
	}

	if cfg.ListenAddr != "127.0.0.1:9000" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, "127.0.0.1:9000")
	}

	if cfg.UpstreamURL != "http://localhost:8080" {
		t.Errorf("UpstreamURL = %q, want %q", cfg.UpstreamURL, "http://localhost:8080")
	}

	if cfg.MaxRecords != 250 {
		t.Errorf("MaxRecords = %d, want %d", cfg.MaxRecords, 250)
	}

	if cfg.CaptureHeaders != true {
		t.Errorf("CaptureHeaders = %v, want %v", cfg.CaptureHeaders, true)
	}
}

// Test isLoopbackAddr_Extended tests additional loopback scenarios.
func TestIsLoopbackAddr_Extended(t *testing.T) {
	tests := []struct {
		addr  string
		allow bool
	}{
		// Allowed
		{"127.0.0.1:0", true},
		{"127.0.0.1:8080", true},
		{"127.0.0.1:65535", true},
		{"localhost:8080", true},
		{"localhost:0", true},
		{"127.1.2.3:8080", true}, // 127.x.x.x is loopback

		// Rejected
		{"0.0.0.0:0", false},
		{"0.0.0.0:8080", false},
		{":0", false},
		{":8080", false},
		{"[::]:8080", false},
		{"[::1]:8080", false},
		{"192.168.1.1:8080", false},
		{"10.0.0.1:8080", false},
		{"172.16.0.1:8080", false},
		{"172.31.255.255:8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := isLoopbackAddr(tt.addr)
			if got != tt.allow {
				t.Errorf("isLoopbackAddr(%q) = %v, want %v", tt.addr, got, tt.allow)
			}
		})
	}
}
