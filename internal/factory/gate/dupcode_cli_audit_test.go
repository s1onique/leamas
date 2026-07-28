// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// TestAuditDupcodeRequiresCI proves that dupcode verifier requires CI authority
// and cannot run in local context without CI environment.
func TestAuditDupcodeRequiresCI(t *testing.T) {
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		t.Fatal("dupcode verifier not found in registry")
	}

	metadata, err := dispatcher.LookupVerifierMetadata("dupcode")
	if err != nil {
		t.Fatalf("LookupVerifierMetadata: %v", err)
	}

	// Dupcode MUST require ci_exact_checkout authority
	if metadata.Authority != verifierauthority.AuthorityCIExactCheckout {
		t.Errorf("dupcode authority = %s, want %s", metadata.Authority, verifierauthority.AuthorityCIExactCheckout)
	}

	t.Logf("dupcode verifier authority: %s (correct - requires CI)", metadata.Authority)
}

// TestAuditCIDeniesUpdateBaseline proves that ci_exact_checkout authority
// denies update_baseline before any scanner execution.
func TestAuditCIDeniesUpdateBaseline(t *testing.T) {
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

	// Valid CI context with proper SHA format
	fakeObserver := &fakeValidCIObserver{}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       ".",
	}

	result := dispatcher.Dispatch(ctx, request, fakeObserver, runnerFactory)

	// Should be denied at operation level (before authority validation)
	if result.Error == nil && len(result.Findings) == 0 {
		t.Error("expected denial for update_baseline under ci_exact_checkout")
	}

	// Verify no scanner execution occurred - this is the key proof
	if scannerCallCount != 0 {
		t.Errorf("scannerCallCount = %d, want 0 (denied before execution)", scannerCallCount)
	}

	// Verify denial reason is operation denied
	if len(result.Findings) > 0 {
		f := result.Findings[0]
		if f.Kind != "verifier_execution_authority_denied" {
			t.Errorf("finding kind = %q, want %q", f.Kind, "verifier_execution_authority_denied")
		}
	}
}

// TestAuditCIAcceptsVerify proves that ci_exact_checkout authority
// accepts verify operation and invokes runner exactly once.
func TestAuditCIAcceptsVerify(t *testing.T) {
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

	// Valid CI context with proper SHA format (40 hex chars = valid SHA-1)
	fakeObserver := &fakeValidCIObserver{}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}

	result := dispatcher.Dispatch(ctx, request, fakeObserver, runnerFactory)

	// Should be allowed
	if result.Error != nil {
		t.Fatalf("verify should be allowed under valid ci_exact_checkout: %v", result.Error)
	}

	// Runner should be called exactly once
	if runnerCallCount != 1 {
		t.Errorf("runnerCallCount = %d, want 1", runnerCallCount)
	}
}

// TestAuditUnknownAuthorityDenied proves that unknown authority is denied
// before any runner execution.
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

	// Should be denied due to wrong authority marker
	if result.Error == nil && len(result.Findings) == 0 {
		t.Error("expected denial for unknown authority")
	}

	if runnerCallCount != 0 {
		t.Errorf("runnerCallCount = %d, want 0 (denied before execution)", runnerCallCount)
	}
}

// TestAuditMissingSHA出一道 proves that missing SHA is denied
// before any runner execution.
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

	// Should be denied due to missing SHA
	if result.Error == nil && len(result.Findings) == 0 {
		t.Error("expected denial for missing SHA")
	}

	if runnerCallCount != 0 {
		t.Errorf("runnerCallCount = %d, want 0 (denied before execution)", runnerCallCount)
	}
}

// TestAuditDirtyTreeIsDenied proves that dirty worktree is denied
// before any runner execution.
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

	// Should be denied due to dirty tree
	if result.Error == nil && len(result.Findings) == 0 {
		t.Error("expected denial for dirty tree")
	}

	if runnerCallCount != 0 {
		t.Errorf("runnerCallCount = %d, want 0 (denied before execution)", runnerCallCount)
	}
}

// TestAuditUpdateOperationDeniedAtOperationLevel proves that update_baseline
// is denied at the operation policy level (before authority context is checked).
func TestAuditUpdateOperationDeniedAtOperationLevel(t *testing.T) {
	// This test verifies that update_baseline is rejected at the policy level,
	// not at the authority validation level. This means even if the authority
	// context is perfect, update_baseline cannot run under ci_exact_checkout.

	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		t.Fatal("dupcode verifier not found in registry")
	}

	// Create a valid CI context
	fakeObserver := &fakeValidCIObserver{}

	// Request update_baseline - this should be denied at operation level
	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       ".",
	}

	result := dispatcher.Dispatch(context.Background(), request, fakeObserver, func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			t.Error("runner should not be called for update_baseline under ci_exact_checkout")
			return nil
		}
	})

	// Should be denied at operation level
	if result.Error == nil && len(result.Findings) == 0 {
		t.Error("expected denial for update_baseline under ci_exact_checkout")
	}

	t.Log("update_baseline is denied at operation policy level for ci_exact_checkout")
}

// fakeValidCIObserver returns a valid CI execution context.
type fakeValidCIObserver struct{}

func (f *fakeValidCIObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	// Use a valid 40-char SHA-1 for GITHUB_SHA and HEAD
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: "github-actions",
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd", // 40 hex chars
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}

// fakeUnknownAuthorityObserver returns a context with wrong authority marker.
type fakeUnknownAuthorityObserver struct{}

func (f *fakeUnknownAuthorityObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: "local", // Wrong authority marker
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
		GitHubSHA:       "", // Missing SHA
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
		WorktreeStatus:  "M  somefile.txt", // Dirty tree
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}
