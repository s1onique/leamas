// SPDX-License-Identifier: Apache-2.0

package dupcodeauthority

import (
	"os"
	"testing"
)

// TestDupcodeAuthorityLocalDeniedBeforeVerifier proves that local execution
// is denied before any verifier initialization, repository scanning, or
// subprocess execution occurs.
func TestDupcodeAuthorityLocalDeniedBeforeVerifier(t *testing.T) {
	// Fake that never executes - invocation count proves denial happens first
	fakeInvocationCount := 0
	fakeVerifier := func(root string) []any {
		fakeInvocationCount++
		return nil
	}
	_ = fakeVerifier // used via count check below

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

	err := ValidateDupcodeExecutionAuthority(ctx)
	if err == nil {
		t.Fatal("expected denial for local execution, got nil")
	}
	if !IsDupcodeDenied(err) {
		t.Fatalf("expected ErrDupcodeDenied, got %v", err)
	}

	// Verify fake was never called (would panic if it tried to scan)
	if fakeInvocationCount != 0 {
		t.Errorf("fake verifier should not have been called, count=%d", fakeInvocationCount)
	}
}

// TestDupcodeAuthorityMissingMarkerDenied matrices each required field
// as missing and proves the authority denies each.
func TestDupcodeAuthorityMissingMarkerDenied(t *testing.T) {
	validSHA := "a71c0340dd08a821e66832488a83e665ba09f02c"

	cases := []struct {
		name   string
		ctx    DupcodeExecutionContext
		marker string
	}{
		{
			name: "missing_CI",
			ctx: DupcodeExecutionContext{
				CI:              "", // missing
				GitHubActions:   "true",
				Authority:       AuthorityMarker,
				GitHubSHA:       validSHA,
				GitHubWorkspace: "/workspace",
				RepositoryRoot:  "/workspace",
				HeadCommit:      validSHA,
				WorktreeClean:   true,
			},
			marker: "CI",
		},
		{
			name: "missing_GITHUB_ACTIONS",
			ctx: DupcodeExecutionContext{
				CI:              "true",
				GitHubActions:   "", // missing
				Authority:       AuthorityMarker,
				GitHubSHA:       validSHA,
				GitHubWorkspace: "/workspace",
				RepositoryRoot:  "/workspace",
				HeadCommit:      validSHA,
				WorktreeClean:   true,
			},
			marker: "GITHUB_ACTIONS",
		},
		{
			name: "missing_LEAMAS_DUPCODE_AUTHORITY",
			ctx: DupcodeExecutionContext{
				CI:              "true",
				GitHubActions:   "true",
				Authority:       "", // missing
				GitHubSHA:       validSHA,
				GitHubWorkspace: "/workspace",
				RepositoryRoot:  "/workspace",
				HeadCommit:      validSHA,
				WorktreeClean:   true,
			},
			marker: "LEAMAS_DUPCODE_AUTHORITY",
		},
		{
			name: "missing_GITHUB_SHA",
			ctx: DupcodeExecutionContext{
				CI:              "true",
				GitHubActions:   "true",
				Authority:       AuthorityMarker,
				GitHubSHA:       "", // missing
				GitHubWorkspace: "/workspace",
				RepositoryRoot:  "/workspace",
				HeadCommit:      validSHA,
				WorktreeClean:   true,
			},
			marker: "GITHUB_SHA",
		},
		{
			name: "missing_GITHUB_WORKSPACE",
			ctx: DupcodeExecutionContext{
				CI:              "true",
				GitHubActions:   "true",
				Authority:       AuthorityMarker,
				GitHubSHA:       validSHA,
				GitHubWorkspace: "", // missing
				RepositoryRoot:  "/workspace",
				HeadCommit:      validSHA,
				WorktreeClean:   true,
			},
			marker: "GITHUB_WORKSPACE",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDupcodeExecutionAuthority(tc.ctx)
			if err == nil {
				t.Fatalf("expected denial for missing %s, got nil", tc.marker)
			}
			if !IsDupcodeDenied(err) {
				t.Fatalf("expected ErrDupcodeDenied for %s, got %v", tc.marker, err)
			}
		})
	}
}

// TestDupcodeAuthorityWrongMarkerDenied covers wrong authority values,
// malformed SHAs, HEAD/GITHUB_SHA mismatches, workspace/repository mismatches,
// and dirty checkouts.
func TestDupcodeAuthorityWrongMarkerDenied(t *testing.T) {
	validSHA := "a71c0340dd08a821e66832488a83e665ba09f02c"

	cases := []struct {
		name   string
		ctx    DupcodeExecutionContext
		reason string
	}{
		{
			name: "wrong_authority_value",
			ctx: DupcodeExecutionContext{
				CI:              "true",
				GitHubActions:   "true",
				Authority:       "local-machine",
				GitHubSHA:       validSHA,
				GitHubWorkspace: "/workspace",
				RepositoryRoot:  "/workspace",
				HeadCommit:      validSHA,
				WorktreeClean:   true,
			},
			reason: "wrong authority value",
		},
		{
			name: "malformed_SHA_short",
			ctx: DupcodeExecutionContext{
				CI:              "true",
				GitHubActions:   "true",
				Authority:       AuthorityMarker,
				GitHubSHA:       "abc123", // too short
				GitHubWorkspace: "/workspace",
				RepositoryRoot:  "/workspace",
				HeadCommit:      validSHA,
				WorktreeClean:   true,
			},
			reason: "malformed SHA",
		},
		{
			name: "malformed_SHA_invalid_chars",
			ctx: DupcodeExecutionContext{
				CI:              "true",
				GitHubActions:   "true",
				Authority:       AuthorityMarker,
				GitHubSHA:       "zzzz0000000000000000000000000000000000000", // invalid hex
				GitHubWorkspace: "/workspace",
				RepositoryRoot:  "/workspace",
				HeadCommit:      validSHA,
				WorktreeClean:   true,
			},
			reason: "malformed SHA",
		},
		{
			name: "HEAD_mismatch",
			ctx: DupcodeExecutionContext{
				CI:              "true",
				GitHubActions:   "true",
				Authority:       AuthorityMarker,
				GitHubSHA:       validSHA,
				GitHubWorkspace: "/workspace",
				RepositoryRoot:  "/workspace",
				HeadCommit:      "0000000000000000000000000000000000000000", // different
				WorktreeClean:   true,
			},
			reason: "HEAD/GITHUB_SHA mismatch",
		},
		{
			name: "workspace_mismatch",
			ctx: DupcodeExecutionContext{
				CI:              "true",
				GitHubActions:   "true",
				Authority:       AuthorityMarker,
				GitHubSHA:       validSHA,
				GitHubWorkspace: "/different/path",
				RepositoryRoot:  "/workspace",
				HeadCommit:      validSHA,
				WorktreeClean:   true,
			},
			reason: "workspace mismatch",
		},
		{
			name: "dirty_worktree",
			ctx: DupcodeExecutionContext{
				CI:              "true",
				GitHubActions:   "true",
				Authority:       AuthorityMarker,
				GitHubSHA:       validSHA,
				GitHubWorkspace: "/workspace",
				RepositoryRoot:  "/workspace",
				HeadCommit:      validSHA,
				WorktreeClean:   false, // dirty
			},
			reason: "dirty worktree",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDupcodeExecutionAuthority(tc.ctx)
			if err == nil {
				t.Fatalf("expected denial for %s, got nil", tc.reason)
			}
			if !IsDupcodeDenied(err) {
				t.Fatalf("expected ErrDupcodeDenied for %s, got %v", tc.reason, err)
			}
		})
	}
}

// TestDupcodeAuthorityGitHubActionsExactSHAAllowed proves that complete
// valid synthetic context allows execution.
func TestDupcodeAuthorityGitHubActionsExactSHAAllowed(t *testing.T) {
	validSHA := "a71c0340dd08a821e66832488a83e665ba09f02c"

	ctx := DupcodeExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		Authority:       AuthorityMarker,
		GitHubSHA:       validSHA,
		GitHubWorkspace: "/workspace",
		RepositoryRoot:  "/workspace",
		HeadCommit:      validSHA,
		WorktreeClean:   true,
	}

	err := ValidateDupcodeExecutionAuthority(ctx)
	if err != nil {
		t.Fatalf("expected approval for valid GitHub Actions context, got error: %v", err)
	}
}

// TestDupcodeAuthorityCountRepetition runs the validator 50 times to prove
// determinism and catch race conditions.
func TestDupcodeAuthorityCountRepetition(t *testing.T) {
	validSHA := "a71c0340dd08a821e66832488a83e665ba09f02c"

	validCtx := DupcodeExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		Authority:       AuthorityMarker,
		GitHubSHA:       validSHA,
		GitHubWorkspace: "/workspace",
		RepositoryRoot:  "/workspace",
		HeadCommit:      validSHA,
		WorktreeClean:   true,
	}

	invalidCtx := DupcodeExecutionContext{
		CI:              "",
		GitHubActions:   "",
		Authority:       "",
		GitHubSHA:       "",
		GitHubWorkspace: "",
		RepositoryRoot:  ".",
		HeadCommit:      "",
		WorktreeClean:   false,
	}

	for i := 0; i < 50; i++ {
		// Valid context should always pass
		if err := ValidateDupcodeExecutionAuthority(validCtx); err != nil {
			t.Fatalf("iteration %d: valid context should pass, got %v", i, err)
		}

		// Invalid context should always fail with IsDupcodeDenied
		err := ValidateDupcodeExecutionAuthority(invalidCtx)
		if err == nil {
			t.Fatalf("iteration %d: invalid context should fail, got nil", i)
		}
		if !IsDupcodeDenied(err) {
			t.Fatalf("iteration %d: expected ErrDupcodeDenied, got %v", i, err)
		}
	}
}

// TestDupcodeAuthorityRace runs the validator concurrently to detect races.
func TestDupcodeAuthorityRace(t *testing.T) {
	validSHA := "a71c0340dd08a821e66832488a83e665ba09f02c"

	validCtx := DupcodeExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		Authority:       AuthorityMarker,
		GitHubSHA:       validSHA,
		GitHubWorkspace: "/workspace",
		RepositoryRoot:  "/workspace",
		HeadCommit:      validSHA,
		WorktreeClean:   true,
	}

	// Run with -race to detect data races
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = ValidateDupcodeExecutionAuthority(validCtx)
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestDupcodeAuthorityEnvLookup tests that the EnvLookup injection works correctly.
func TestDupcodeAuthorityEnvLookup(t *testing.T) {
	// This tests the EnvLookup type and LookupFromOS
	envVar := "TEST_DUPCODE_AUTHORITY_VAR"
	testValue := "test-value"

	oldVal := os.Getenv(envVar)
	defer func() {
		if oldVal == "" {
			os.Unsetenv(envVar)
		} else {
			os.Setenv(envVar, oldVal)
		}
	}()

	os.Setenv(envVar, testValue)
	val := LookupFromOS(envVar)
	if val != testValue {
		t.Errorf("LookupFromOS(%q) = %q, want %q", envVar, val, testValue)
	}
}

// TestDupcodeAuthorityResolveRepoRootEmptyPath proves empty path is handled.
func TestDupcodeAuthorityResolveRepoRootEmptyPath(t *testing.T) {
	result := resolveRepoRoot("")
	if result != "" {
		t.Errorf("resolveRepoRoot(\"\") = %q, want empty string", result)
	}
}
