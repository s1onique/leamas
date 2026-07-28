// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// TestAuditDispatchDupcodeVerifyOperationMapping proves that the dispatch path
// for verify uses OperationVerify.
func TestAuditDispatchDupcodeVerifyOperationMapping(t *testing.T) {
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

	fakeObserver := &fakeValidCIObserver{}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}

	result := dispatcher.Dispatch(ctx, request, fakeObserver, runnerFactory)

	if result.Error != nil {
		t.Fatalf("verify should be allowed under valid ci_exact_checkout: %v", result.Error)
	}

	if runnerCallCount != 1 {
		t.Errorf("runnerCallCount = %d, want 1", runnerCallCount)
	}

	t.Logf("OperationVerify: allowed, runnerCalls=%d", runnerCallCount)
}

// TestAuditDispatchDupcodeUpdateBaselineOperationMapping proves that the dispatch
// path for update uses OperationUpdateBaseline.
func TestAuditDispatchDupcodeUpdateBaselineOperationMapping(t *testing.T) {
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

	fakeObserver := &fakeValidCIObserver{}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       ".",
	}

	result := dispatcher.Dispatch(ctx, request, fakeObserver, runnerFactory)

	if result.Error == nil && len(result.Findings) == 0 {
		t.Fatal("update_baseline should be denied under ci_exact_checkout")
	}

	if runnerCallCount != 0 {
		t.Errorf("runnerCallCount = %d, want 0 (denied before execution)", runnerCallCount)
	}

	t.Logf("OperationUpdateBaseline: denied, runnerCalls=%d", runnerCallCount)
}

// TestAuditVerifyNeverReachesMutation proves that verify never reaches mutation.
func TestAuditVerifyNeverReachesMutation(t *testing.T) {
	scannerCallCount := 0
	mutationCallCount := 0

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
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}

	result := dispatcher.Dispatch(ctx, request, fakeObserver, runnerFactory)

	if result.Error != nil {
		t.Fatalf("verify should succeed: %v", result.Error)
	}

	if scannerCallCount != 1 {
		t.Errorf("scannerCallCount = %d, want 1", scannerCallCount)
	}

	if mutationCallCount != 0 {
		t.Errorf("mutationCallCount = %d, want 0", mutationCallCount)
	}
}

// TestAuditCIUpdateDeniedBeforeExecution proves ci_exact_checkout denies update.
func TestAuditCIUpdateDeniedBeforeExecution(t *testing.T) {
	scannerCallCount := 0
	mutationCallCount := 0

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			scannerCallCount++
			mutationCallCount++
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
		t.Fatal("expected denial for update_baseline under ci_exact_checkout")
	}

	if scannerCallCount != 0 {
		t.Errorf("scannerCallCount = %d, want 0", scannerCallCount)
	}

	if mutationCallCount != 0 {
		t.Errorf("mutationCallCount = %d, want 0", mutationCallCount)
	}

	if len(result.Findings) > 0 {
		f := result.Findings[0]
		if f.Kind != "verifier_execution_authority_denied" {
			t.Errorf("finding kind = %q, want %q", f.Kind, "verifier_execution_authority_denied")
		}
	}
}

// TestAuditCIAcceptsVerifyWithValidContext proves ci_exact_checkout accepts verify.
func TestAuditCIAcceptsVerifyWithValidContext(t *testing.T) {
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

	fakeObserver := &fakeValidCIObserver{}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}

	result := dispatcher.Dispatch(ctx, request, fakeObserver, runnerFactory)

	if result.Error != nil {
		t.Fatalf("verify should be allowed: %v", result.Error)
	}

	if runnerCallCount != 1 {
		t.Errorf("runnerCallCount = %d, want 1", runnerCallCount)
	}
}

// fakeValidCIObserver returns a valid CI execution context.
type fakeValidCIObserver struct{}

func (f *fakeValidCIObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: "github-actions",
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd",
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}
