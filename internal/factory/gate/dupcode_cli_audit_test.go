// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// TestAuditDispatcherDenialCases table-driven test for all denial scenarios.
func TestAuditDispatcherDenialCases(t *testing.T) {
	tests := []struct {
		name       string
		operation  verifierauthority.VerifierOperation
		observer   verifierdispatch.ContextObserver
		wantDenied bool
		wantRunner int
	}{
		{
			name:       "ci_update_denied",
			operation:  verifierauthority.OperationUpdateBaseline,
			observer:   &fakeValidCIObserver{},
			wantDenied: true,
			wantRunner: 0,
		},
		{
			name:       "ci_verify_allowed",
			operation:  verifierauthority.OperationVerify,
			observer:   &fakeValidCIObserver{},
			wantDenied: false,
			wantRunner: 1,
		},
		{
			name:       "unknown_authority_denied",
			operation:  verifierauthority.OperationVerify,
			observer:   &fakeUnknownAuthorityObserver{},
			wantDenied: true,
			wantRunner: 0,
		},
		{
			name:       "missing_sha_denied",
			operation:  verifierauthority.OperationVerify,
			observer:   &fakeMissingSHAObserver{},
			wantDenied: true,
			wantRunner: 0,
		},
		{
			name:       "dirty_tree_denied",
			operation:  verifierauthority.OperationVerify,
			observer:   &fakeDirtyTreeObserver{},
			wantDenied: true,
			wantRunner: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runnerCallCount := 0

			runnerFactory := func() func(root string) []checks.Finding {
				return func(root string) []checks.Finding {
					runnerCallCount++
					return nil
				}
			}

			ctx := context.Background()
			dispatcher, ok := DispatcherForVerifier("dupcode")
			if !ok {
				t.Fatal("dupcode verifier not found")
			}

			request := verifierdispatch.Request{
				VerifierID: "dupcode",
				Operation:  tc.operation,
				Root:       ".",
			}

			result := dispatcher.Dispatch(ctx, request, tc.observer, runnerFactory)

			gotDenied := result.Error != nil || len(result.Findings) > 0

			if gotDenied != tc.wantDenied {
				t.Errorf("denied = %v, want %v", gotDenied, tc.wantDenied)
			}

			if runnerCallCount != tc.wantRunner {
				t.Errorf("runnerCallCount = %d, want %d", runnerCallCount, tc.wantRunner)
			}
		})
	}
}

// TestAuditUnknownAuthorityDenied proves unknown authority is denied before execution.
func TestAuditUnknownAuthorityDenied(t *testing.T) {
	runnerCallCount := 0

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			runnerCallCount++
			return nil
		}
	}

	ctx := context.Background()
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		t.Fatal("dupcode verifier not found in registry")
	}

	fakeObserver := &fakeUnknownAuthorityObserver{}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}

	result := dispatcher.Dispatch(ctx, request, fakeObserver, runnerFactory)

	if result.Error == nil && len(result.Findings) == 0 {
		t.Fatal("expected denial for unknown authority")
	}

	if runnerCallCount != 0 {
		t.Errorf("runnerCallCount = %d, want 0", runnerCallCount)
	}
}

// TestAuditMissingSHAIsDenied proves missing SHA is denied before execution.
func TestAuditMissingSHAIsDenied(t *testing.T) {
	runnerCallCount := 0

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			runnerCallCount++
			return nil
		}
	}

	ctx := context.Background()
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		t.Fatal("dupcode verifier not found in registry")
	}

	fakeObserver := &fakeMissingSHAObserver{}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}

	result := dispatcher.Dispatch(ctx, request, fakeObserver, runnerFactory)

	if result.Error == nil && len(result.Findings) == 0 {
		t.Fatal("expected denial for missing SHA")
	}

	if runnerCallCount != 0 {
		t.Errorf("runnerCallCount = %d, want 0", runnerCallCount)
	}
}

// TestAuditDirtyTreeIsDenied proves dirty worktree is denied before execution.
func TestAuditDirtyTreeIsDenied(t *testing.T) {
	runnerCallCount := 0

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			runnerCallCount++
			return nil
		}
	}

	ctx := context.Background()
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		t.Fatal("dupcode verifier not found in registry")
	}

	fakeObserver := &fakeDirtyTreeObserver{}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}

	result := dispatcher.Dispatch(ctx, request, fakeObserver, runnerFactory)

	if result.Error == nil && len(result.Findings) == 0 {
		t.Fatal("expected denial for dirty tree")
	}

	if runnerCallCount != 0 {
		t.Errorf("runnerCallCount = %d, want 0", runnerCallCount)
	}
}

// TestAuditOperationPolicyDeniesUpdate proves update_baseline denied at policy level.
func TestAuditOperationPolicyDeniesUpdate(t *testing.T) {
	scannerCallCount := 0

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			scannerCallCount++
			return nil
		}
	}

	ctx := context.Background()
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		t.Fatal("dupcode verifier not found in registry")
	}

	fakeObserver := &fakeValidCIObserver{}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       ".",
	}

	result := dispatcher.Dispatch(ctx, request, fakeObserver, runnerFactory)

	if result.Error == nil && len(result.Findings) == 0 {
		t.Fatal("expected denial for update_baseline at operation policy level")
	}

	if scannerCallCount != 0 {
		t.Errorf("scannerCallCount = %d, want 0", scannerCallCount)
	}

	if len(result.Findings) > 0 {
		f := result.Findings[0]
		if f.Kind != "verifier_execution_authority_denied" {
			t.Errorf("finding kind = %q, want %q", f.Kind, "verifier_execution_authority_denied")
		}
	}
}

// Fake observers for testing.

// fakeUnknownAuthorityObserver returns a context with wrong authority marker.
type fakeUnknownAuthorityObserver struct{}

func (f *fakeUnknownAuthorityObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: "local",
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd",
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}

// fakeMissingSHAObserver returns a context without GITHUB_SHA.
type fakeMissingSHAObserver struct{}

func (f *fakeMissingSHAObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: "github-actions",
		GitHubSHA:       "",
		GitHubWorkspace: root,
		HeadCommit:      "",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}

// fakeDirtyTreeObserver returns a context with dirty worktree.
type fakeDirtyTreeObserver struct{}

func (f *fakeDirtyTreeObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: "github-actions",
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd",
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "M  somefile.txt",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}
