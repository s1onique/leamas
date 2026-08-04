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

// countingFactory records how many times the runner factory is invoked.
type countingFactory struct {
	calls atomic.Int64
}

func (c *countingFactory) Factory() RunnerFactory {
	return func() func(root string) []checks.Finding {
		c.calls.Add(1)
		return func(root string) []checks.Finding { return nil }
	}
}

// noObserverFactory records the call but never invokes the observer.
type noObserverFactory struct{ called bool }

func (n *noObserverFactory) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	n.called = true
	return verifierauthority.ExecutionContext{}
}

// TestDispatchRejectsDisallowedOperationOnMutationIdentity proves that
// requesting the verify operation on the dupcode-update-baseline
// mutation identity is denied before the observer, factory, scan, or
// write run.
func TestDispatchRejectsDisallowedOperationOnMutationIdentity(t *testing.T) {
	verifiers := []registry.Verifier{testMutationVerifier("dupcode-update-baseline")}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &noObserverFactory{}
	factory := &countingFactory{}

	request := Request{
		VerifierID: "dupcode-update-baseline",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}
	result := dispatcher.Dispatch(context.Background(), request, obs, factory.Factory())
	if result.Error == nil {
		t.Fatal("expected denial when mutation identity receives verify operation")
	}
	if obs.called {
		t.Error("observer must NOT be called on disallowed operation")
	}
	if factory.calls.Load() != 0 {
		t.Errorf("factory calls = %d, want 0 (denial before factory)", factory.calls.Load())
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected exactly one finding, got %d", len(result.Findings))
	}
	if result.Findings[0].Kind != "verifier_operation_not_allowed" {
		t.Errorf("finding kind = %q, want verifier_operation_not_allowed",
			result.Findings[0].Kind)
	}
	if _, ok := result.Error.(*AuthorityOperationError); !ok {
		t.Errorf("result.Error type = %T, want *AuthorityOperationError", result.Error)
	}
}

// TestDispatchRejectsVerifyAliasForDupcodeUpdate proves a plain dupcode
// (verify-only) verifier also cannot accept update_baseline.
func TestDispatchRejectsVerifyAliasForDupcodeUpdate(t *testing.T) {
	verifiers := []registry.Verifier{
		testVerifier("dupcode", verifierauthority.AuthorityCIExactCheckout, registry.VerifierLaneDupcode),
	}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &noObserverFactory{}
	factory := &countingFactory{}
	request := Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       ".",
	}
	result := dispatcher.Dispatch(context.Background(), request, obs, factory.Factory())
	if result.Error == nil {
		t.Fatal("expected denial when ordinary verifier receives update_baseline")
	}
	if factory.calls.Load() != 0 {
		t.Errorf("factory calls = %d, want 0", factory.calls.Load())
	}
}

// TestDispatchRejectsUpdateBaselineOnDupcodeBaseline proves the
// dupcode-baseline verifier also cannot accept update_baseline.
func TestDispatchRejectsUpdateBaselineOnDupcodeBaseline(t *testing.T) {
	verifiers := []registry.Verifier{
		testVerifier("dupcode-baseline", verifierauthority.AuthorityCIExactCheckout, registry.VerifierLaneDupcode),
	}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &noObserverFactory{}
	factory := &countingFactory{}
	request := Request{
		VerifierID: "dupcode-baseline",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       ".",
	}
	result := dispatcher.Dispatch(context.Background(), request, obs, factory.Factory())
	if result.Error == nil {
		t.Fatal("expected denial when dupcode-baseline receives update_baseline")
	}
	if factory.calls.Load() != 0 {
		t.Errorf("factory calls = %d, want 0", factory.calls.Load())
	}
}

// TestDispatchRejectsUpdateBaselineOnFastVerifier proves the ordinary
// fast verifier also cannot accept update_baseline.
func TestDispatchRejectsUpdateBaselineOnFastVerifier(t *testing.T) {
	verifiers := []registry.Verifier{
		testVerifier("llm-friendly", verifierauthority.AuthorityLocalSafe, registry.VerifierLaneFast),
	}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &noObserverFactory{}
	factory := &countingFactory{}
	request := Request{
		VerifierID: "llm-friendly",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       ".",
	}
	result := dispatcher.Dispatch(context.Background(), request, obs, factory.Factory())
	if result.Error == nil {
		t.Fatal("expected denial when fast verifier receives update_baseline")
	}
	if factory.calls.Load() != 0 {
		t.Errorf("factory calls = %d, want 0", factory.calls.Load())
	}
}

// TestDispatchRejectsUnknownOperationSyntax proves the dispatcher
// rejects syntactically unknown operation values before the observer.
func TestDispatchRejectsUnknownOperationSyntax(t *testing.T) {
	verifiers := []registry.Verifier{
		testMutationVerifier("dupcode-update-baseline"),
	}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &noObserverFactory{}
	factory := &countingFactory{}
	request := Request{
		VerifierID: "dupcode-update-baseline",
		Operation:  verifierauthority.VerifierOperation("unknown"),
		Root:       ".",
	}
	result := dispatcher.Dispatch(context.Background(), request, obs, factory.Factory())
	if result.Error == nil {
		t.Fatal("expected denial on unknown operation syntax")
	}
	if obs.called {
		t.Error("observer must NOT be called on unknown operation syntax")
	}
	if factory.calls.Load() != 0 {
		t.Errorf("factory calls = %d, want 0", factory.calls.Load())
	}
}

// TestDispatchCheapPathRequiresBothConditions proves the cheap local-safe
// verify path requires BOTH the request operation to be verify AND the
// verifier to declare verify. A verify request to a verifier that does
// NOT declare verify must not take the cheap path.
func TestDispatchCheapPathRequiresBothConditions(t *testing.T) {
	// Construct a verifier whose declared operations are intentionally
	// not [verify] -- the cheap path must NOT apply even when the
	// request operation is verify.
	verifiers := []registry.Verifier{{
		Name:      "dupcode-update-baseline",
		Lane:      registry.VerifierLaneDupcode,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     registry.InvocationCommandOnly,
		Operations: []verifierauthority.VerifierOperation{
			verifierauthority.OperationUpdateBaseline,
		},
	}}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &noObserverFactory{}
	factory := &countingFactory{}
	request := Request{
		VerifierID: "dupcode-update-baseline",
		Operation:  verifierauthority.OperationVerify,
		Root:       ".",
	}
	result := dispatcher.Dispatch(context.Background(), request, obs, factory.Factory())
	// The verifier does not declare verify, so the cheap path is
	// denied before observation.
	if result.Error == nil {
		t.Fatal("expected denial: verifier does not declare verify")
	}
	if obs.called {
		t.Error("observer must not be called when the verifier does not declare the requested operation")
	}
}

// TestProfileDeniesOperationMismatchBeforeObserver proves a profile
// request with a disallowed operation is denied before the observer is
// ever invoked.
func TestProfileDeniesOperationMismatchBeforeObserver(t *testing.T) {
	verifiers := []registry.Verifier{
		testVerifier("dupcode", verifierauthority.AuthorityCIExactCheckout, registry.VerifierLaneDupcode),
	}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &noObserverFactory{}
	requests := []ProfileRequest{
		{VerifierID: "dupcode", Operation: verifierauthority.OperationUpdateBaseline},
	}
	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, obs)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if profile.AuthorizationSucceeded() {
		t.Error("expected authorization to fail on disallowed operation")
	}
	if obs.called {
		t.Error("observer must not be called when any request has disallowed operation")
	}
	if len(profile.Denials()) != 1 {
		t.Fatalf("Denials = %d, want 1", len(profile.Denials()))
	}
	if profile.Denials()[0].Findings[0].Kind != "verifier_operation_not_allowed" {
		t.Errorf("denial kind = %q, want verifier_operation_not_allowed",
			profile.Denials()[0].Findings[0].Kind)
	}
}
