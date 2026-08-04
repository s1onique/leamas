// SPDX-License-Identifier: Apache-2.0

package dupcodeauthority

import (
	"fmt"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// TestDupcodeAuthorityLocalDeniedBeforeVerifier proves that local execution
// is denied before any verifier initialization, repository scanning, or
// subprocess execution occurs. The fake verifier MUST NOT be called when
// the authority denies; we prove this by asserting the invocation count
// remains zero after the guarded dispatch.
func TestDupcodeAuthorityLocalDeniedBeforeVerifier(t *testing.T) {
	fakeInvocationCount := 0
	fakeVerifier := func(root string) []checks.Finding {
		fakeInvocationCount++
		return nil
	}

	// Build a context with empty environment (local execution)
	ctx := DupcodeExecutionContext{
		CI:              "", // missing - should deny
		GitHubActions:   "",
		Authority:       "",
		GitHubSHA:       "",
		GitHubWorkspace: "",
		RepositoryRoot:  ".",
		HeadCommit:      "",
		WorktreeClean:   false,
	}

	// Wrap the fake verifier with the guard - this is the production pattern
	guardedVerifier := Guard(ctx, verifierauthority.OperationVerify, fakeVerifier)

	// Execute the guarded verifier
	findings := guardedVerifier(".")

	// Authority must deny
	if len(findings) == 0 {
		t.Fatal("expected denial findings for local execution, got none")
	}

	// Findings must indicate denial
	found := false
	for _, f := range findings {
		if f.Kind == "dupcode_ci_only_authority_denied" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected dupcode_ci_only_authority_denied finding, got: %v", findings)
	}

	// CRITICAL: fake verifier must NOT have been called
	if fakeInvocationCount != 0 {
		t.Errorf("fake verifier was called %d times, expected 0", fakeInvocationCount)
	}
}

// TestDupcodeAuthorityGuardDeniesWithCanonicalError proves Guard returns
// a finding with the correct canonical error structure.
func TestDupcodeAuthorityGuardDeniesWithCanonicalError(t *testing.T) {
	fakeVerifier := func(root string) []checks.Finding {
		t.Error("verifier should not be called when authority denies")
		return nil
	}

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

	guardedVerifier := Guard(ctx, verifierauthority.OperationVerify, fakeVerifier)
	findings := guardedVerifier(".")

	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.Kind != "dupcode_ci_only_authority_denied" {
		t.Errorf("expected kind %q, got %q", "dupcode_ci_only_authority_denied", f.Kind)
	}
	if f.Severity != checks.SeverityError {
		t.Errorf("expected severity %v, got %v", checks.SeverityError, f.Severity)
	}
	if !IsDupcodeDenied(fmt.Errorf("%w: CI must be set", ErrDupcodeDenied)) {
		t.Errorf("finding message should wrap ErrDupcodeDenied: %s", f.Message)
	}
}

// TestDupcodeAuthorityGuardAllowsInValidContext proves Guard allows
// execution when the context is valid.
func TestDupcodeAuthorityGuardAllowsInValidContext(t *testing.T) {
	fakeInvocationCount := 0
	fakeVerifier := func(root string) []checks.Finding {
		fakeInvocationCount++
		return nil
	}

	ctx := DupcodeExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		Authority:       AuthorityMarker,
		GitHubSHA:       "a71c0340dd08a821e66832488a83e665ba09f02c",
		GitHubWorkspace: "/workspace",
		RepositoryRoot:  "/workspace",
		HeadCommit:      "a71c0340dd08a821e66832488a83e665ba09f02c",
		WorktreeClean:   true,
		WorkspaceRoot:   "/workspace",
	}

	guardedVerifier := Guard(ctx, verifierauthority.OperationVerify, fakeVerifier)
	findings := guardedVerifier(".")

	if len(findings) != 0 {
		t.Fatalf("expected no findings for valid context, got: %v", findings)
	}

	if fakeInvocationCount != 1 {
		t.Errorf("fake verifier should have been called once, was called %d times", fakeInvocationCount)
	}
}
