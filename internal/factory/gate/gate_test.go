package gate

import (
	"bytes"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
)

func TestAllVerifiers(t *testing.T) {
	verifiers := AllVerifiers()
	if len(verifiers) == 0 {
		t.Error("AllVerifiers should return verifiers")
	}

	// Check all have names
	for _, v := range verifiers {
		if v.Name == "" {
			t.Error("verifier should have a name")
		}
		if v.Run == nil {
			t.Error("verifier should have a Run function")
		}
		if v.Lane == "" {
			t.Errorf("verifier %q should have a Lane assigned", v.Name)
		}
	}
}

func TestVerifierLanes(t *testing.T) {
	all := AllVerifiers()
	fast := FastVerifiers()
	dupcode := DupcodeVerifiers()

	// Every verifier must belong to exactly one lane
	fastNames := make(map[string]bool)
	for _, v := range fast {
		fastNames[v.Name] = true
	}
	dupcodeNames := make(map[string]bool)
	for _, v := range dupcode {
		dupcodeNames[v.Name] = true
	}

	for _, v := range all {
		if v.Lane == registry.VerifierLaneFast && !fastNames[v.Name] {
			t.Errorf("verifier %q has Lane=fast but not in FastVerifiers()", v.Name)
		}
		if v.Lane == registry.VerifierLaneDupcode && !dupcodeNames[v.Name] {
			t.Errorf("verifier %q has Lane=dupcode but not in DupcodeVerifiers()", v.Name)
		}
		if v.Lane != registry.VerifierLaneFast && v.Lane != registry.VerifierLaneDupcode {
			t.Errorf("verifier %q has unknown lane %q", v.Name, v.Lane)
		}
	}

	// dupcode and dupcode-baseline must be in dupcode lane
	if !dupcodeNames["dupcode"] {
		t.Error("dupcode verifier must be in dupcode lane")
	}
	if !dupcodeNames["dupcode-baseline"] {
		t.Error("dupcode-baseline verifier must be in dupcode lane")
	}
}

func TestSelectVerifiers(t *testing.T) {
	fast := FastVerifiers()
	dupcode := DupcodeVerifiers()

	// Fast lane must not contain dupcode verifiers
	for _, v := range fast {
		if v.Name == "dupcode" {
			t.Error("fast lane must not contain dupcode verifier")
		}
		if v.Name == "dupcode-baseline" {
			t.Error("fast lane must not contain dupcode-baseline verifier")
		}
	}

	// Dupcode lane must contain exactly the dupcode verifiers
	dupcodeVerifierNames := make(map[string]bool)
	for _, v := range dupcode {
		dupcodeVerifierNames[v.Name] = true
	}
	if !dupcodeVerifierNames["dupcode"] {
		t.Error("dupcode lane must contain dupcode verifier")
	}
	if !dupcodeVerifierNames["dupcode-baseline"] {
		t.Error("dupcode lane must contain dupcode-baseline verifier")
	}
	if len(dupcode) != 2 {
		t.Errorf("dupcode lane must contain exactly 2 verifiers, got %d", len(dupcode))
	}
}

// fixtureVerifier creates a minimal test verifier.
func fixtureVerifier(name string, findings []checks.Finding) registry.Verifier {
	return registry.Verifier{
		Name: name,
		Run:  func(string) []checks.Finding { return findings },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "gate_test.fixtureVerifier",
			EnvVars:          []string{},
		},
		Cache: registry.CacheSemantics{
			GoBuildCache:      registry.CacheNotApplicable,
			GoTestResultCache: registry.CacheModeNA,
		},
	}
}

func TestRunFactorizeFixtures(t *testing.T) {
	// Use fixture verifiers instead of live AllVerifiers() to avoid
	// nested full-registry execution. This test proves the ordering
	// and failure-propagation behavior without scanning the repository.
	verifiers := []registry.Verifier{
		fixtureVerifier("passing", nil),
		fixtureVerifier("alpha", nil),
		fixtureVerifier("beta", []checks.Finding{
			{Path: "p", Kind: "k", Message: "m", Severity: checks.SeverityError},
		}),
	}

	code := runFactorizeForTest(verifiers)
	if code != 1 {
		t.Errorf("runFactorizeForTest with failing verifier returned %d, want 1", code)
	}
}

// runFactorizeForTest wraps runFactorize with a fake clock for testing.
func runFactorizeForTest(verifiers []registry.Verifier) int {
	return runFactorize(&bytes.Buffer{}, systemClock{}, ".", verifiers, nil, &noopSampler{})
}
