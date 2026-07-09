// Package boundary provides verification for domain boundary import policies.
package boundary

import (
	"testing"
)

// TestCockpitCLIAllowsContext verifies that cockpit CLI allows context.
func TestCockpitCLIAllowsContext(t *testing.T) {
	result := Check(repoRoot())
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "context" {
			t.Errorf("cockpit CLI should allow context import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitCLIAllowsNetHTTP verifies that cockpit CLI allows net/http.
func TestCockpitCLIAllowsNetHTTP(t *testing.T) {
	result := Check(repoRoot())
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "net/http" {
			t.Errorf("cockpit CLI should allow net/http import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitCLIAllowsOSSignal verifies that cockpit CLI allows os/signal.
func TestCockpitCLIAllowsOSSignal(t *testing.T) {
	result := Check(repoRoot())
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "os/signal" {
			t.Errorf("cockpit CLI should allow os/signal import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitCLIAllowsInternalCockpit verifies that cockpit CLI allows internal/web/cockpit.
func TestCockpitCLIAllowsInternalCockpit(t *testing.T) {
	result := Check(repoRoot())
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/cockpit.go" && f.Import == "github.com/s1onique/leamas/internal/web/cockpit" {
			t.Errorf("cockpit CLI should allow internal/web/cockpit import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessCLIAllowsContext verifies that witness CLI allows context.
func TestWitnessCLIAllowsContext(t *testing.T) {
	result := Check(repoRoot())
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/witness.go" && f.Import == "context" {
			t.Errorf("witness CLI should allow context import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessCLIAllowsNetHTTP verifies that witness CLI allows net/http.
func TestWitnessCLIAllowsNetHTTP(t *testing.T) {
	result := Check(repoRoot())
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/witness.go" && f.Import == "net/http" {
			t.Errorf("witness CLI should allow net/http import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessCLIAllowsOSSignal verifies that witness CLI allows os/signal.
func TestWitnessCLIAllowsOSSignal(t *testing.T) {
	result := Check(repoRoot())
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/witness.go" && f.Import == "os/signal" {
			t.Errorf("witness CLI should allow os/signal import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessCLIAllowsInternalWitnessProxy verifies that witness CLI allows internal/witness/proxy.
func TestWitnessCLIAllowsInternalWitnessProxy(t *testing.T) {
	result := Check(repoRoot())
	for _, f := range result.Findings {
		if f.File == "cmd/leamas/witness.go" && f.Import == "github.com/s1onique/leamas/internal/witness/proxy" {
			t.Errorf("witness CLI should allow internal/witness/proxy import, but got violation: %s", f.Reason)
		}
	}
}
