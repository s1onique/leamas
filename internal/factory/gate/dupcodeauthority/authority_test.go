// SPDX-License-Identifier: Apache-2.0

package dupcodeauthority

import (
	"errors"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"strings"
	"testing"
)

// TestDupcodeAuthorityMissingMarkerDenied matrices each required field
// as missing and proves the authority denies each.
func TestDupcodeAuthorityMissingMarkerDenied(t *testing.T) {
	validSHA := "a71c0340dd08a821e66832488a83e665ba09f02c"

	cases := []struct {
		name   string
		ctx    DupcodeExecutionContext
		marker string
	}{
		{name: "missing_CI", ctx: DupcodeExecutionContext{
			CI: "", GitHubActions: "true", Authority: AuthorityMarker,
			GitHubSHA: validSHA, GitHubWorkspace: "/workspace",
			RepositoryRoot: "/workspace", HeadCommit: validSHA, WorktreeClean: true,
		}, marker: "CI"},
		{name: "missing_GITHUB_ACTIONS", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "", Authority: AuthorityMarker,
			GitHubSHA: validSHA, GitHubWorkspace: "/workspace",
			RepositoryRoot: "/workspace", HeadCommit: validSHA, WorktreeClean: true,
		}, marker: "GITHUB_ACTIONS"},
		{name: "missing_LEAMAS_DUPCODE_AUTHORITY", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: "",
			GitHubSHA: validSHA, GitHubWorkspace: "/workspace",
			RepositoryRoot: "/workspace", HeadCommit: validSHA, WorktreeClean: true,
		}, marker: "LEAMAS_DUPCODE_AUTHORITY"},
		{name: "missing_GITHUB_SHA", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
			GitHubSHA: "", GitHubWorkspace: "/workspace",
			RepositoryRoot: "/workspace", HeadCommit: validSHA, WorktreeClean: true,
		}, marker: "GITHUB_SHA"},
		{name: "missing_GITHUB_WORKSPACE", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
			GitHubSHA: validSHA, GitHubWorkspace: "",
			RepositoryRoot: "/workspace", HeadCommit: validSHA, WorktreeClean: true,
		}, marker: "GITHUB_WORKSPACE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDupcodeExecutionAuthority(tc.ctx, verifierauthority.OperationVerify)
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
		{name: "wrong_authority_value", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: "local-machine",
			GitHubSHA: validSHA, GitHubWorkspace: "/workspace",
			RepositoryRoot: "/workspace", HeadCommit: validSHA, WorktreeClean: true,
		}, reason: "wrong authority value"},
		{name: "malformed_SHA_short", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
			GitHubSHA: "abc123", GitHubWorkspace: "/workspace",
			RepositoryRoot: "/workspace", HeadCommit: validSHA, WorktreeClean: true,
		}, reason: "malformed SHA"},
		{name: "malformed_SHA_invalid_chars", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
			GitHubSHA:       "zzzz0000000000000000000000000000000000000",
			GitHubWorkspace: "/workspace", RepositoryRoot: "/workspace", WorkspaceRoot: "/workspace",
			HeadCommit: validSHA, WorktreeClean: true,
		}, reason: "malformed SHA"},
		{name: "HEAD_mismatch", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
			GitHubSHA: validSHA, GitHubWorkspace: "/workspace",
			RepositoryRoot: "/workspace",
			HeadCommit:     "0000000000000000000000000000000000000000", WorktreeClean: true,
		}, reason: "HEAD/GITHUB_SHA mismatch"},
		{name: "workspace_mismatch", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
			GitHubSHA: validSHA, GitHubWorkspace: "/different/path",
			RepositoryRoot: "/workspace", HeadCommit: validSHA, WorktreeClean: true,
		}, reason: "workspace mismatch"},
		{name: "dirty_worktree", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
			GitHubSHA: validSHA, GitHubWorkspace: "/workspace",
			RepositoryRoot: "/workspace", HeadCommit: validSHA, WorktreeClean: false,
		}, reason: "dirty worktree"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDupcodeExecutionAuthority(tc.ctx, verifierauthority.OperationVerify)
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
	ctx := DupcodeExecutionContext{
		CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
		GitHubSHA:       "a71c0340dd08a821e66832488a83e665ba09f02c",
		GitHubWorkspace: "/workspace", RepositoryRoot: "/workspace", WorkspaceRoot: "/workspace",
		HeadCommit: "a71c0340dd08a821e66832488a83e665ba09f02c", WorktreeClean: true,
	}

	err := ValidateDupcodeExecutionAuthority(ctx, verifierauthority.OperationVerify)
	if err != nil {
		t.Fatalf("expected approval for valid GitHub Actions context, got error: %v", err)
	}
}

// TestDupcodeAuthorityCountRepetition runs the validator 50 times to prove
// determinism and catch race conditions.
func TestDupcodeAuthorityCountRepetition(t *testing.T) {
	validCtx := DupcodeExecutionContext{
		CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
		GitHubSHA:       "a71c0340dd08a821e66832488a83e665ba09f02c",
		GitHubWorkspace: "/workspace", RepositoryRoot: "/workspace", WorkspaceRoot: "/workspace",
		HeadCommit: "a71c0340dd08a821e66832488a83e665ba09f02c", WorktreeClean: true,
	}
	invalidCtx := DupcodeExecutionContext{
		CI: "", GitHubActions: "", Authority: "", GitHubSHA: "",
		GitHubWorkspace: "", RepositoryRoot: ".", HeadCommit: "", WorktreeClean: false,
	}

	for i := 0; i < 50; i++ {
		if err := ValidateDupcodeExecutionAuthority(validCtx, verifierauthority.OperationVerify); err != nil {
			t.Fatalf("iteration %d: valid context should pass, got %v", i, err)
		}
		err := ValidateDupcodeExecutionAuthority(invalidCtx, verifierauthority.OperationVerify)
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
	validCtx := DupcodeExecutionContext{
		CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
		GitHubSHA:       "a71c0340dd08a821e66832488a83e665ba09f02c",
		GitHubWorkspace: "/workspace", RepositoryRoot: "/workspace", WorkspaceRoot: "/workspace",
		HeadCommit: "a71c0340dd08a821e66832488a83e665ba09f02c", WorktreeClean: true,
	}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = ValidateDupcodeExecutionAuthority(validCtx, verifierauthority.OperationVerify)
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestDupcodeAuthorityGitErrorsFailClosed proves that Git execution errors
// cause denial rather than being treated as clean worktree.
func TestDupcodeAuthorityGitErrorsFailClosed(t *testing.T) {
	validSHA := "a71c0340dd08a821e66832488a83e665ba09f02c"

	cases := []struct {
		name      string
		ctx       DupcodeExecutionContext
		expectErr string
	}{
		{name: "git_head_error", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
			GitHubSHA: validSHA, GitHubWorkspace: "/workspace",
			RepositoryRoot: "/workspace", HeadCommit: "",
			HeadErr:       errors.New("git rev-parse failed: fatal: not a git repository"),
			WorktreeClean: true,
		}, expectErr: "git rev-parse HEAD failed"},
		{name: "git_status_error", ctx: DupcodeExecutionContext{
			CI: "true", GitHubActions: "true", Authority: AuthorityMarker,
			GitHubSHA: validSHA, GitHubWorkspace: "/workspace",
			RepositoryRoot: "/workspace", HeadCommit: validSHA,
			WorktreeClean: true,
			StatusErr:     errors.New("git status failed: fatal: not a git repository"),
		}, expectErr: "git status failed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDupcodeExecutionAuthority(tc.ctx, verifierauthority.OperationVerify)
			if err == nil {
				t.Fatal("expected denial due to Git error, got nil")
			}
			if !IsDupcodeDenied(err) {
				t.Fatalf("expected ErrDupcodeDenied, got %v", err)
			}
			if tc.expectErr != "" && !strings.Contains(err.Error(), tc.expectErr) {
				t.Errorf("expected error containing %q, got: %v", tc.expectErr, err)
			}
		})
	}
}
