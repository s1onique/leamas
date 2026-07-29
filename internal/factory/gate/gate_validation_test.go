// Package gate provides tests for verifier validation.
package gate

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
)

// TestValidateVerifier_EmptyName verifies validation fails for empty name.
func TestValidateVerifier_EmptyName(t *testing.T) {
	v := registry.Verifier{Name: "", Run: func(string) []checks.Finding { return nil }, Lane: registry.VerifierLaneFast}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

// TestValidateVerifier_NilRun verifies validation fails for nil Run.
func TestValidateVerifier_NilRun(t *testing.T) {
	v := registry.Verifier{Name: "test", Run: nil, Lane: registry.VerifierLaneFast}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for nil Run")
	}
}

// TestValidateVerifier_InvalidLane verifies validation fails for invalid lane.
func TestValidateVerifier_InvalidLane(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: "unknown",
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "test",
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for invalid lane")
	}
}

// TestValidateVerifier_InvalidKind verifies validation fails for invalid kind.
func TestValidateVerifier_InvalidKind(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             "invalid",
			ImplementationID: "test",
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for invalid kind")
	}
}

// TestValidateVerifier_EmptyImplID verifies validation fails for empty ImplementationID.
func TestValidateVerifier_EmptyImplID(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "",
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for empty ImplementationID")
	}
}

// TestValidateVerifier_InvalidGoBuildCache verifies validation fails for invalid GoBuildCache.
func TestValidateVerifier_InvalidGoBuildCache(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "test",
		},
		Cache: registry.CacheSemantics{
			GoBuildCache:      "invalid",
			GoTestResultCache: registry.CacheModeNA,
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for invalid GoBuildCache")
	}
}

// TestValidateVerifier_InvalidGoTestResultCache verifies validation fails for invalid GoTestResultCache.
func TestValidateVerifier_InvalidGoTestResultCache(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "test",
		},
		Cache: registry.CacheSemantics{
			GoBuildCache:      registry.CacheNotApplicable,
			GoTestResultCache: "invalid",
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for invalid GoTestResultCache")
	}
}

// TestValidateVerifier_DuplicateEnvKey verifies validation fails for duplicate env keys.
func TestValidateVerifier_DuplicateEnvKey(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "test",
			EnvVars:          []string{"GOFLAGS", "GOFLAGS"},
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for duplicate env key")
	}
}

// TestValidateVerifier_MalformedEnvKeyWithEquals verifies validation fails for env key with =.
func TestValidateVerifier_MalformedEnvKeyWithEquals(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "test",
			EnvVars:          []string{"GOFLAGS=value"},
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for malformed env key with =")
	}
}

// TestValidateVerifier_MalformedEnvKeyWithWhitespace verifies validation fails for env key with whitespace.
func TestValidateVerifier_MalformedEnvKeyWithWhitespace(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "test",
			EnvVars:          []string{" GOFLAGS"},
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for env key with whitespace")
	}
}

// TestValidateVerifier_MalformedEnvKeyInvalidName verifies validation fails for invalid env key name.
func TestValidateVerifier_MalformedEnvKeyInvalidName(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "test",
			EnvVars:          []string{"123INVALID"},
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for invalid env key name")
	}
}

// TestValidateVerifier_ValidEmptyEnvVars verifies validation passes for empty env vars.
func TestValidateVerifier_ValidEmptyEnvVars(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "test",
			EnvVars:          []string{},
		},
		Cache: registry.CacheSemantics{
			GoBuildCache:      registry.CacheNotApplicable,
			GoTestResultCache: registry.CacheModeNA,
		},
	}
	err := ValidateVerifier(v)
	if err != nil {
		t.Errorf("expected no error for empty env vars: %v", err)
	}
}

// TestValidateVerifiers_NoDuplicates verifies validation fails for duplicate names.
func TestValidateVerifiers_NoDuplicates(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: registry.VerifierLaneFast,
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "test",
		},
	}
	verifiers := []registry.Verifier{v, v}
	err := ValidateVerifiers(verifiers)
	if err == nil {
		t.Error("expected error for duplicate names")
	}
}

// TestValidateVerifiers_AllCanonical verifies all canonical verifiers pass validation.
func TestValidateVerifiers_AllCanonical(t *testing.T) {
	verifiers := AllVerifiers()
	err := ValidateVerifiers(verifiers)
	if err != nil {
		t.Errorf("canonical verifiers should pass validation: %v", err)
	}
}

// TestPartitionVerifiers validates partition of all verifiers into fast and dupcode lanes.
func TestPartitionVerifiers(t *testing.T) {
	all := AllVerifiers()
	fast, dupcode, err := PartitionVerifiers(all)
	if err != nil {
		t.Fatalf("PartitionVerifiers failed: %v", err)
	}
	// PartitionVerifiers excludes the command-only dupcode-update-baseline
	// from the partitioned lanes; the partition count therefore equals
	// the gate-scoped subset of AllVerifiers, not the full set.
	if len(fast)+len(dupcode) != len(GateVerifiers()) {
		t.Errorf("partition size mismatch: got %d fast + %d dupcode = %d, want %d (gate-scoped count)",
			len(fast), len(dupcode), len(fast)+len(dupcode), len(GateVerifiers()))
	}
	// And every gate-scoped entry must land in exactly one lane.
	if len(fast)+len(dupcode) >= len(all) {
		t.Errorf("partition must not include command-only entries (got %d, want %d)",
			len(fast)+len(dupcode), len(all))
	}
	for _, v := range fast {
		if v.Lane != registry.VerifierLaneFast {
			t.Errorf("fast verifier %q has lane %q", v.Name, v.Lane)
		}
	}
	for _, v := range dupcode {
		if v.Lane != registry.VerifierLaneDupcode {
			t.Errorf("dupcode verifier %q has lane %q", v.Name, v.Lane)
		}
	}
}

// TestPartitionVerifiers_InvalidLane verifies partition fails for unknown lane.
func TestPartitionVerifiers_InvalidLane(t *testing.T) {
	v := registry.Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Lane: "invalid",
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "test",
		},
	}
	_, _, err := PartitionVerifiers([]registry.Verifier{v})
	if err == nil {
		t.Error("expected error for invalid lane")
	}
}
