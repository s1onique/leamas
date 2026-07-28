// SPDX-License-Identifier: Apache-2.0

package dupcodeauthority

import (
	"errors"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"testing"
)

// TestDupcodeAuthorityErrorMessageStability proves the canonical error message
// contains the expected diagnostic text.
func TestDupcodeAuthorityErrorMessageStability(t *testing.T) {
	ctx := DupcodeExecutionContext{
		CI:              "",
		GitHubActions:   "true",
		Authority:       AuthorityMarker,
		GitHubSHA:       "a71c0340dd08a821e66832488a83e665ba09f02c",
		GitHubWorkspace: "/workspace",
		RepositoryRoot:  "/workspace",
		HeadCommit:      "a71c0340dd08a821e66832488a83e665ba09f02c",
		WorktreeClean:   true,
	}

	err := ValidateDupcodeExecutionAuthority(ctx, verifierauthority.OperationVerify)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDupcodeDenied(err) {
		t.Fatalf("expected ErrDupcodeDenied, got %v", err)
	}

	msg := err.Error()
	expected := []string{
		"dupcode is a CI-only verifier lane",
		"local execution is prohibited",
		"Factory Dupcode status check",
	}
	for _, s := range expected {
		if !contains(msg, s) {
			t.Errorf("error message should contain %q, got %q", s, msg)
		}
	}
}

// TestIsDupcodeDeniedFalseForOtherErrors proves IsDupcodeDenied only
// returns true for errors from this package.
func TestIsDupcodeDeniedFalseForOtherErrors(t *testing.T) {
	otherErr := errors.New("some other error")
	if IsDupcodeDenied(otherErr) {
		t.Error("IsDupcodeDenied should return false for unrelated errors")
	}
}

// TestDupcodeAuthorityIsDupcodeErrorInterface proves ErrDupcodeDenied satisfies
// the standard error interface and errors.Is works correctly.
func TestDupcodeAuthorityIsDupcodeErrorInterface(t *testing.T) {
	ctx := DupcodeExecutionContext{
		CI:              "",
		GitHubActions:   "true",
		Authority:       AuthorityMarker,
		GitHubSHA:       "a71c0340dd08a821e66832488a83e665ba09f02c",
		GitHubWorkspace: "/workspace",
		RepositoryRoot:  "/workspace",
		HeadCommit:      "a71c0340dd08a821e66832488a83e665ba09f02c",
		WorktreeClean:   true,
	}

	err := ValidateDupcodeExecutionAuthority(ctx, verifierauthority.OperationVerify)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrDupcodeDenied) {
		t.Error("errors.Is should return true for ErrDupcodeDenied")
	}
}

// contains is a helper for substring checking.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
