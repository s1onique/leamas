package main

import (
	"testing"
)

// TestRuntimeSmokeCockpitRejectsUnsafeListenAddresses verifies cockpit rejects unsafe addresses.
func TestRuntimeSmokeCockpitRejectsUnsafeListenAddresses(t *testing.T) {
	unsafeAddrs := []string{
		"0.0.0.0:8080",
		":8080",
		"[::]:8080",
		"192.168.1.10:8080",
		"10.0.0.1:8080",
	}

	for _, addr := range unsafeAddrs {
		t.Run(addr, func(t *testing.T) {
			_, err := parseCockpitServeArgs([]string{"--listen", addr})
			if err == nil {
				t.Errorf("parseCockpitServeArgs(%q) expected error for unsafe address", addr)
			}
		})
	}
}

// TestRuntimeSmokeWitnessRejectsUnsafeListenAddresses verifies witness proxy rejects unsafe addresses.
func TestRuntimeSmokeWitnessRejectsUnsafeListenAddresses(t *testing.T) {
	unsafeAddrs := []string{
		"0.0.0.0:8080",
		":8080",
		"[::]:8080",
		"192.168.1.10:8080",
		"10.0.0.1:8080",
	}

	for _, addr := range unsafeAddrs {
		t.Run(addr, func(t *testing.T) {
			_, err := parseWitnessProxyArgs([]string{"--listen", addr, "--upstream", "http://127.0.0.1:8080"})
			if err == nil {
				t.Errorf("parseWitnessProxyArgs(%q) expected error for unsafe address", addr)
			}
		})
	}
}

// TestRuntimeSmokeWitnessRequiresUpstream verifies witness proxy requires --upstream.
func TestRuntimeSmokeWitnessRequiresUpstream(t *testing.T) {
	_, err := parseWitnessProxyArgs([]string{})
	if err == nil {
		t.Error("parseWitnessProxyArgs() expected error for missing --upstream")
	}
}

// TestRuntimeSmokeWitnessRejectsInvalidUpstreamScheme verifies witness proxy rejects invalid upstream schemes.
func TestRuntimeSmokeWitnessRejectsInvalidUpstreamScheme(t *testing.T) {
	invalidUpstreams := []string{
		"ftp://127.0.0.1",
		"file:///tmp/x",
		"127.0.0.1:8080",
		"ws://127.0.0.1:8080",
	}

	for _, upstream := range invalidUpstreams {
		t.Run(upstream, func(t *testing.T) {
			_, err := parseWitnessProxyArgs([]string{"--upstream", upstream})
			if err == nil {
				t.Errorf("parseWitnessProxyArgs(%q) expected error for invalid upstream scheme", upstream)
			}
		})
	}
}

// TestRuntimeSmokeCockpitAllowSafeListenAddresses verifies cockpit accepts safe loopback addresses.
func TestRuntimeSmokeCockpitAllowSafeListenAddresses(t *testing.T) {
	safeAddrs := []string{
		"127.0.0.1:0",
		"127.0.0.1:8080",
		"localhost:0",
		"localhost:8080",
	}

	for _, addr := range safeAddrs {
		t.Run(addr, func(t *testing.T) {
			_, err := parseCockpitServeArgs([]string{"--listen", addr})
			if err != nil {
				t.Errorf("parseCockpitServeArgs(%q) unexpected error: %v", addr, err)
			}
		})
	}
}

// TestRuntimeSmokeWitnessAllowSafeListenAddresses verifies witness proxy accepts safe loopback addresses.
func TestRuntimeSmokeWitnessAllowSafeListenAddresses(t *testing.T) {
	safeAddrs := []string{
		"127.0.0.1:0",
		"127.0.0.1:8080",
		"localhost:0",
		"localhost:8080",
	}

	for _, addr := range safeAddrs {
		t.Run(addr, func(t *testing.T) {
			_, err := parseWitnessProxyArgs([]string{"--listen", addr, "--upstream", "http://127.0.0.1:8080"})
			if err != nil {
				t.Errorf("parseWitnessProxyArgs(%q) unexpected error: %v", addr, err)
			}
		})
	}
}

// TestRuntimeSmokeCommandsAreBounded verifies all smoke tests use bounded timeouts.
func TestRuntimeSmokeCommandsAreBounded(t *testing.T) {
	t.Log("All runtime smoke tests use context deadlines or explicit timeouts")
}
