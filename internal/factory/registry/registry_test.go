// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

func TestVerifier_Validate_EmptyAuthority(t *testing.T) {
	v := Verifier{
		Name:      "test",
		Lane:      VerifierLaneFast,
		Authority: "",
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
	}

	err := v.Validate()
	if err == nil {
		t.Error("expected error for unknown authority")
	}
}

func TestVerifier_Validate_DupcodeLocalSafe(t *testing.T) {
	// Dupcode verifiers should NOT be local_safe (security violation)
	v := Verifier{
		Name:      "dupcode",
		Lane:      VerifierLaneDupcode,
		Authority: verifierauthority.AuthorityLocalSafe,
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
	}

	err := v.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
