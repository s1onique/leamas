// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// noopRun is a non-nil run function used by tests that need a
// gate-scoped verifier to satisfy the new "Run required" rule.
func noopRun(root string) []checks.Finding {
	return nil
}

func TestVerifier_Validate_EmptyAuthority(t *testing.T) {
	v := Verifier{
		Name:      "test",
		Lane:      VerifierLaneFast,
		Authority: "",
		Scope:     InvocationGate,
	}

	err := v.Validate()
	if err == nil {
		t.Error("expected error for empty authority")
	}
}

func TestVerifier_Validate_UnknownAuthority(t *testing.T) {
	v := Verifier{
		Name:      "test",
		Lane:      VerifierLaneFast,
		Authority: "unknown_authority",
		Scope:     InvocationGate,
	}

	err := v.Validate()
	if err == nil {
		t.Error("expected error for unknown authority")
	}
}

func TestVerifier_Validate_DupcodeLocalSafe(t *testing.T) {
	// Dupcode verifiers should NOT be local_safe (security violation).
	// The dupcode-update-baseline entry is the single explicit exception.
	v := Verifier{
		Name:      "dupcode",
		Lane:      VerifierLaneDupcode,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     InvocationGate,
	}

	err := v.Validate()
	if err == nil {
		t.Error("expected error for dupcode/local_safe combination")
	}
}

func TestVerifier_Validate_FastCIExactCheckout(t *testing.T) {
	// Fast verifiers should NOT require CI exact checkout (too restrictive)
	v := Verifier{
		Name:      "tooling-boundaries",
		Lane:      VerifierLaneFast,
		Authority: verifierauthority.AuthorityCIExactCheckout,
		Scope:     InvocationGate,
	}

	err := v.Validate()
	if err == nil {
		t.Error("expected error for fast/ci_exact_checkout combination")
	}
}

func TestVerifier_Validate_ValidFast(t *testing.T) {
	v := Verifier{
		Name:      "llm-friendly",
		Lane:      VerifierLaneFast,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     InvocationGate,
		Run:       noopRun,
	}

	err := v.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifier_Validate_ValidDupcode(t *testing.T) {
	v := Verifier{
		Name:      "dupcode",
		Lane:      VerifierLaneDupcode,
		Authority: verifierauthority.AuthorityCIExactCheckout,
		Scope:     InvocationGate,
		Run:       noopRun,
	}

	err := v.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifier_Validate_EmptyScope(t *testing.T) {
	v := Verifier{
		Name:      "test",
		Lane:      VerifierLaneFast,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     "",
	}

	err := v.Validate()
	if err == nil {
		t.Error("expected error for empty invocation scope")
	}
}

func TestVerifier_Validate_UnknownScope(t *testing.T) {
	v := Verifier{
		Name:      "test",
		Lane:      VerifierLaneFast,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     "fictional-scope",
	}

	err := v.Validate()
	if err == nil {
		t.Error("expected error for unknown invocation scope")
	}
}

func TestVerifier_Validate_GateScopeNilRun(t *testing.T) {
	v := Verifier{
		Name:      "test",
		Lane:      VerifierLaneFast,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     InvocationGate,
		Run:       nil,
	}

	err := v.Validate()
	if err == nil {
		t.Error("expected error for gate-scoped verifier with nil Run")
	}
}

func TestVerifier_Validate_CommandOnlyNilRun(t *testing.T) {
	v := Verifier{
		Name:      "dupcode-update-baseline",
		Lane:      VerifierLaneDupcode,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     InvocationCommandOnly,
		Run:       nil,
	}

	err := v.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifier_Validate_CommandOnlyWithRunRejected(t *testing.T) {
	v := Verifier{
		Name:      "dupcode-update-baseline",
		Lane:      VerifierLaneDupcode,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     InvocationCommandOnly,
		Run:       noopRun,
	}

	err := v.Validate()
	if err == nil {
		t.Error("expected error for command-only verifier with non-nil Run")
	}
}
