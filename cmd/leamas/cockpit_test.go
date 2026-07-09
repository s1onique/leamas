package main

import (
	"testing"
)

func TestParseCockpitServeArgs_Defaults(t *testing.T) {
	cfg, err := parseCockpitServeArgs([]string{})
	if err != nil {
		t.Fatalf("parseCockpitServeArgs() error = %v", err)
	}

	if cfg.ListenAddr != "127.0.0.1:0" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, "127.0.0.1:0")
	}
}

func TestParseCockpitServeArgs_CustomListen(t *testing.T) {
	cfg, err := parseCockpitServeArgs([]string{"--listen", "127.0.0.1:8765"})
	if err != nil {
		t.Fatalf("parseCockpitServeArgs() error = %v", err)
	}

	if cfg.ListenAddr != "127.0.0.1:8765" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, "127.0.0.1:8765")
	}
}

func TestParseCockpitServeArgs_Localhost(t *testing.T) {
	cfg, err := parseCockpitServeArgs([]string{"--listen", "localhost:8080"})
	if err != nil {
		t.Fatalf("parseCockpitServeArgs() error = %v", err)
	}

	if cfg.ListenAddr != "localhost:8080" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, "localhost:8080")
	}
}

func TestParseCockpitServeArgs_ZeroPort(t *testing.T) {
	cfg, err := parseCockpitServeArgs([]string{"--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseCockpitServeArgs() error = %v", err)
	}

	if cfg.ListenAddr != "127.0.0.1:0" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, "127.0.0.1:0")
	}
}

func TestParseCockpitServeArgs_Rejects0_0_0_0(t *testing.T) {
	_, err := parseCockpitServeArgs([]string{"--listen", "0.0.0.0:8765"})
	if err == nil {
		t.Error("parseCockpitServeArgs() expected error for 0.0.0.0")
	}
}

func TestParseCockpitServeArgs_RejectsBare0(t *testing.T) {
	_, err := parseCockpitServeArgs([]string{"--listen", ":8765"})
	if err == nil {
		t.Error("parseCockpitServeArgs() expected error for bare port")
	}
}

func TestParseCockpitServeArgs_RejectsIPv6All(t *testing.T) {
	_, err := parseCockpitServeArgs([]string{"--listen", "[::]:8765"})
	if err == nil {
		t.Error("parseCockpitServeArgs() expected error for [::]")
	}
}

func TestParseCockpitServeArgs_MissingListenArg(t *testing.T) {
	_, err := parseCockpitServeArgs([]string{"--listen"})
	if err == nil {
		t.Error("parseCockpitServeArgs() expected error for missing --listen argument")
	}
}

func TestParseCockpitServeArgs_UnknownFlag(t *testing.T) {
	_, err := parseCockpitServeArgs([]string{"--unknown"})
	if err == nil {
		t.Error("parseCockpitServeArgs() expected error for unknown flag")
	}
}

func TestIsLoopbackAddr(t *testing.T) {
	tests := []struct {
		addr  string
		allow bool
	}{
		// Allowed
		{"127.0.0.1:0", true},
		{"127.0.0.1:8080", true},
		{"127.0.0.1:8765", true},
		{"localhost:8080", true},
		{"localhost:0", true},
		{"localhost.", true},

		// Rejected
		{"0.0.0.0:0", false},
		{"0.0.0.0:8080", false},
		{":0", false},
		{":8080", false},
		{"[::]:8080", false},
		{"[::1]:8080", false},
		{"192.168.1.1:8080", false},
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

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		addr     string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{"127.0.0.1:8080", "127.0.0.1", "8080", false},
		{"localhost:8080", "localhost", "8080", false},
		{"[::1]:8080", "::1", "8080", false},
		{"[fe80::1]:8080", "fe80::1", "8080", false},
		{"127.0.0.1", "127.0.0.1", "", false},
		{"localhost", "localhost", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			host, port, err := splitHostPort(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitHostPort(%q) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
				return
			}
			if host != tt.wantHost {
				t.Errorf("splitHostPort(%q) host = %q, want %q", tt.addr, host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("splitHostPort(%q) port = %q, want %q", tt.addr, port, tt.wantPort)
			}
		})
	}
}
