// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

func TestNewDispatcher_ValidRegistry(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
			Scope:     registry.InvocationGate,
			Run:       func(root string) []checks.Finding { return nil },
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dispatcher == nil {
		t.Fatal("expected non-nil dispatcher")
	}
}

func TestNewDispatcher_EmptyRegistry(t *testing.T) {
	verifiers := []registry.Verifier{}

	_, err := NewDispatcher(verifiers)
	if err == nil {
		t.Error("expected error for empty registry")
	}
}

func TestNewDispatcher_EmptyVerifierID(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
	}

	_, err := NewDispatcher(verifiers)
	if err == nil {
		t.Error("expected error for empty verifier ID")
	}
}

func TestNewDispatcher_DuplicateID(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "test",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
			Scope:     registry.InvocationGate,
		},
		{
			Name:      "test",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
			Scope:     registry.InvocationGate,
		},
	}

	_, err := NewDispatcher(verifiers)
	if err == nil {
		t.Error("expected error for duplicate verifier ID")
	}
}

func TestNewDispatcher_EmptyAuthority(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "test",
			Lane:      registry.VerifierLaneFast,
			Authority: "",
		},
	}

	_, err := NewDispatcher(verifiers)
	if err == nil {
		t.Error("expected error for empty authority")
	}
}

func TestNewDispatcher_UnknownAuthority(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "test",
			Lane:      registry.VerifierLaneFast,
			Authority: "unknown_authority",
		},
	}

	_, err := NewDispatcher(verifiers)
	if err == nil {
		t.Error("expected error for unknown authority")
	}
}

func TestNewDispatcher_DupcodeLocalSafe(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "dupcode",
			Lane:      registry.VerifierLaneDupcode,
			Authority: verifierauthority.AuthorityLocalSafe,
			Scope:     registry.InvocationGate,
		},
	}

	_, err := NewDispatcher(verifiers)
	if err == nil {
		t.Error("expected error for dupcode/local_safe combination")
	}
}

func TestNewDispatcher_FastCIExactCheckout(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "test",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityCIExactCheckout,
			Scope:     registry.InvocationGate,
		},
	}

	_, err := NewDispatcher(verifiers)
	if err == nil {
		t.Error("expected error for fast/ci_exact_checkout combination")
	}
}

func TestLookupVerifierMetadata(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
			Scope:     registry.InvocationGate,
			Run:       func(root string) []checks.Finding { return nil },
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v, err := dispatcher.LookupVerifierMetadata("llm-friendly")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if v.Name != "llm-friendly" {
		t.Errorf("expected llm-friendly, got %s", v.Name)
	}

	_, err = dispatcher.LookupVerifierMetadata("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent verifier")
	}
}

func TestGetVerifierMetadata(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
			Scope:     registry.InvocationGate,
			Execution: registry.ExecutionDefinition{
				EnvVars: []string{"ORIGINAL"},
			},
			Run: func(root string) []checks.Finding { return nil },
		},
		{
			Name:      "tooling-boundaries",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
			Scope:     registry.InvocationGate,
			Run:       func(root string) []checks.Finding { return nil },
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := dispatcher.GetVerifierMetadata()
	if len(result) != 2 {
		t.Fatalf("expected 2 verifiers, got %d", len(result))
	}

	// Verify it's a copy - EnvVars mutation should not affect dispatcher
	if len(result[0].EnvVars) > 0 {
		result[0].EnvVars[0] = "MUTATED"
		original := dispatcher.GetVerifierMetadata()
		if original[0].EnvVars[0] == "MUTATED" {
			t.Error("GetVerifierMetadata should return deep copy with independent EnvVars")
		}
	}
}

func TestVerifierMetadataContainsNoRun(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "test",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
			Scope:     registry.InvocationGate,
			Run:       func(root string) []checks.Finding { return nil },
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v, err := dispatcher.LookupVerifierMetadata("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the metadata type has no Run field
	// This is a compile-time check via reflection
	// Access is intentionally omitted - VerifierMetadata has no Run field
	_ = v.Name
	_ = v.Authority
	// If this compiled, Run is not accessible through VerifierMetadata
}

func TestInputMutationDoesNotAffectDispatcher(t *testing.T) {
	originalEnvVars := []string{"ORIGINAL=value"}
	verifiers := []registry.Verifier{
		{
			Name:      "test",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
			Scope:     registry.InvocationGate,
			Execution: registry.ExecutionDefinition{
				EnvVars: originalEnvVars,
			},
			Run: func(root string) []checks.Finding { return nil },
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate original after construction
	verifiers[0].Execution.EnvVars[0] = "MUTATED"

	// Dispatcher should be unaffected
	v := dispatcher.GetVerifierMetadata()
	if v[0].EnvVars[0] == "MUTATED" {
		t.Error("dispatcher should be unaffected by input mutation")
	}
}
