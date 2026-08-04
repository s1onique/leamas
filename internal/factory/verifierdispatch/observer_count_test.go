// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// countingObserver records the number of times Observe is invoked. It is
// the test seam for proving the cheap local-safe verify path.
type countingObserver struct {
	ctx     atomic.Int64
	counter atomic.Int64
}

func (c *countingObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	c.counter.Add(1)
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: verifierauthority.AuthorityMarker,
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd",
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}

// TestLocalSafeVerifySkipsObserver proves the cheap local-safe verify
// path performs zero full Git observations.
func TestLocalSafeVerifySkipsObserver(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "local-fast",
		Authority:  verifierauthority.AuthorityLocalSafe,
		Lane:       registry.VerifierLaneFast,
		Scope:      registry.InvocationGate,
		Operations: verifyOnly(),
		Run:        func(root string) []checks.Finding { return nil },
	}}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &countingObserver{}
	request := Request{
		VerifierID: "local-fast",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}
	result := dispatcher.Dispatch(context.Background(), request, obs, func() func(string) []checks.Finding {
		return func(root string) []checks.Finding { return nil }
	})
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if got := obs.counter.Load(); got != 0 {
		t.Errorf("observer calls = %d, want 0 (cheap local-safe verify)", got)
	}
}

// TestLocalSafeUpdateUsesObserver proves local_safe + update_baseline
// triggers exactly one full observer call.
func TestLocalSafeUpdateUsesObserver(t *testing.T) {
	verifiers := []registry.Verifier{testMutationVerifier("dupcode-update-baseline")}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &countingObserver{}
	request := Request{
		VerifierID: "dupcode-update-baseline",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       ".",
	}
	// Command-only entries are not registered as Run functions; the
	// typed binder sets up the runner. For this test we just need to
	// prove the observer is invoked once before the runner factory
	// short-circuits with a stubbed Run that returns nil.
	dispatcher.Dispatch(context.Background(), request, obs, func() func(string) []checks.Finding {
		return func(root string) []checks.Finding {
			return []checks.Finding{
				{Path: "test", Kind: "fake", Message: "fake", Severity: checks.SeverityError},
			}
		}
	})
	if got := obs.counter.Load(); got != 1 {
		t.Errorf("observer calls = %d, want 1 (local_safe + update needs classification)", got)
	}
}

// TestCIVerifyUsesObserver proves ci_exact_checkout + verify triggers
// exactly one full observer call.
func TestCIVerifyUsesObserver(t *testing.T) {
	verifiers := []registry.Verifier{
		testVerifier("dupcode", verifierauthority.AuthorityCIExactCheckout, registry.VerifierLaneDupcode),
	}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &countingObserver{}
	request := Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}
	result := dispatcher.Dispatch(context.Background(), request, obs, func() func(string) []checks.Finding {
		return func(root string) []checks.Finding { return nil }
	})
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if got := obs.counter.Load(); got != 1 {
		t.Errorf("observer calls = %d, want 1 (ci_exact_checkout needs full observation)", got)
	}
}
