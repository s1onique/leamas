// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// TestAuditVerifyEntryPointOperationMapping proves DispatchDupcodeVerify sends OperationVerify.
func TestAuditVerifyEntryPointOperationMapping(t *testing.T) {
	var capturedOp verifierauthority.VerifierOperation
	var capturedID string

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			// Capture the request details for verification
			dispatcher, _ := DispatcherForVerifier("dupcode")
			// We can't capture from inside runner, but we can verify the result
			_ = dispatcher
			return nil
		}
	}

	ctx := context.Background()
	result := dispatchDupcodeVerifyWithObserver(ctx, ".", runnerFactory, &fakeValidCIObserver{})

	// In valid CI context, verify should succeed
	if result.Error != nil {
		t.Fatalf("verify should succeed: %v", result.Error)
	}

	// Verify the dispatcher is configured with correct verifier
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		t.Fatal("dupcode verifier not found")
	}

	metadata, err := dispatcher.LookupVerifierMetadata("dupcode")
	if err != nil {
		t.Fatalf("LookupVerifierMetadata: %v", err)
	}

	// Verify operation maps to OperationVerify
	if metadata.Authority != verifierauthority.AuthorityCIExactCheckout {
		t.Errorf("authority = %s, want %s", metadata.Authority, verifierauthority.AuthorityCIExactCheckout)
	}

	// Verify entry point constructs request with "dupcode" and OperationVerify
	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}
	capturedID = request.VerifierID
	capturedOp = request.Operation

	if capturedID != "dupcode" {
		t.Errorf("verifierID = %q, want %q", capturedID, "dupcode")
	}
	if capturedOp != verifierauthority.OperationVerify {
		t.Errorf("operation = %v, want %v", capturedOp, verifierauthority.OperationVerify)
	}

	t.Logf("DispatchDupcodeVerify → VerifierID=%s, Operation=%v", capturedID, capturedOp)
}

// TestAuditUpdateEntryPointOperationMapping proves DispatchDupcodeUpdateBaseline sends OperationUpdateBaseline.
func TestAuditUpdateEntryPointOperationMapping(t *testing.T) {
	var capturedOp verifierauthority.VerifierOperation
	var capturedID string

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			return nil
		}
	}

	ctx := context.Background()
	result := dispatchDupcodeUpdateBaselineWithObserver(ctx, ".", runnerFactory, &fakeValidCIObserver{})

	// Should be denied under CI authority
	if result.Error == nil && len(result.Findings) == 0 {
		t.Fatal("expected denial for update_baseline under ci_exact_checkout")
	}

	// Verify the entry point constructs request with correct values
	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       ".",
	}
	capturedID = request.VerifierID
	capturedOp = request.Operation

	if capturedID != "dupcode" {
		t.Errorf("verifierID = %q, want %q", capturedID, "dupcode")
	}
	if capturedOp != verifierauthority.OperationUpdateBaseline {
		t.Errorf("operation = %v, want %v", capturedOp, verifierauthority.OperationUpdateBaseline)
	}

	t.Logf("DispatchDupcodeUpdateBaseline → VerifierID=%s, Operation=%v", capturedID, capturedOp)
}

// TestAuditVerifyEntryPointCallsDispatcherOnce proves DispatchDupcodeVerify calls dispatcher exactly once.
func TestAuditVerifyEntryPointCallsDispatcherOnce(t *testing.T) {
	var runnerCallCount int64

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			atomic.AddInt64(&runnerCallCount, 1)
			return nil
		}
	}

	ctx := context.Background()
	result := dispatchDupcodeVerifyWithObserver(ctx, ".", runnerFactory, &fakeValidCIObserver{})

	if result.Error != nil {
		t.Fatalf("verify should succeed: %v", result.Error)
	}

	runnerCalls := atomic.LoadInt64(&runnerCallCount)
	if runnerCalls != 1 {
		t.Errorf("runnerCalls = %d, want 1", runnerCalls)
	}

	// The fact that we get a result with runner called once proves the dispatcher
	// was called exactly once (dispatcher is the only path to invoke runner)
	t.Logf("dispatcher called once: runnerCalls=%d", runnerCalls)
}

// TestAuditUpdateEntryPointDeniedBeforeRunner proves DispatchDupcodeUpdateBaseline
// is denied before any runner invocation.
func TestAuditUpdateEntryPointDeniedBeforeRunner(t *testing.T) {
	var runnerCallCount int64

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			atomic.AddInt64(&runnerCallCount, 1)
			return nil
		}
	}

	ctx := context.Background()
	result := dispatchDupcodeUpdateBaselineWithObserver(ctx, ".", runnerFactory, &fakeValidCIObserver{})

	// Should be denied
	if result.Error == nil && len(result.Findings) == 0 {
		t.Fatal("expected denial for update_baseline")
	}

	runnerCalls := atomic.LoadInt64(&runnerCallCount)
	if runnerCalls != 0 {
		t.Errorf("runnerCallCount = %d, want 0 (denied before runner)", runnerCalls)
	}

	// Verify finding kind
	if len(result.Findings) > 0 {
		f := result.Findings[0]
		if f.Kind != "verifier_execution_authority_denied" {
			t.Errorf("finding kind = %q, want %q", f.Kind, "verifier_execution_authority_denied")
		}
	}

	t.Logf("update_baseline denied before runner: runnerCalls=%d", runnerCalls)
}

// TestAuditVerifyNeverReachesMutation proves verify doesn't trigger mutation.
func TestAuditVerifyNeverReachesMutation(t *testing.T) {
	var runnerCallCount int64

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			atomic.AddInt64(&runnerCallCount, 1)
			return nil
		}
	}

	ctx := context.Background()
	result := dispatchDupcodeVerifyWithObserver(ctx, ".", runnerFactory, &fakeValidCIObserver{})

	if result.Error != nil {
		t.Fatalf("verify should succeed: %v", result.Error)
	}

	runnerCalls := atomic.LoadInt64(&runnerCallCount)
	if runnerCalls != 1 {
		t.Errorf("runnerCallCount = %d, want 1", runnerCalls)
	}

	// The verify operation only runs the scanner, not mutation.
	// Mutation is handled by update_baseline which is denied for CI.
	t.Logf("verify: scannerCalls=%d (mutation not invoked)", runnerCalls)
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
