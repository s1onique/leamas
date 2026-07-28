// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

func TestNewDispatcher_ValidRegistry(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
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
		},
		{
			Name:      "test",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
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
		},
	}

	_, err := NewDispatcher(verifiers)
	if err == nil {
		t.Error("expected error for fast/ci_exact_checkout combination")
	}
}

func TestLookupVerifier(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v, err := dispatcher.LookupVerifier("llm-friendly")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if v.Name != "llm-friendly" {
		t.Errorf("expected llm-friendly, got %s", v.Name)
	}

	_, err = dispatcher.LookupVerifier("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent verifier")
	}
}

func TestGetVerifiers(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
		{
			Name:      "tooling-boundaries",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := dispatcher.GetVerifiers()
	if len(result) != 2 {
		t.Fatalf("expected 2 verifiers, got %d", len(result))
	}

	// Verify it's a copy
	result[0].Name = "modified"
	original := dispatcher.GetVerifiers()
	if original[0].Name == "modified" {
		t.Error("GetVerifiers should return a copy")
	}
}
