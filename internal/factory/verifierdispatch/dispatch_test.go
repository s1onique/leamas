// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"errors"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// fakeRunnerFactory tracks whether the factory was invoked.
type fakeRunnerFactory struct {
	called bool
	runner func(root string) []checks.Finding
}

func (f *fakeRunnerFactory) Factory() RunnerFactory {
	return func() func(root string) []checks.Finding {
		f.called = true
		return f.runner
	}
}

func TestDispatch_AuthorityDenied(t *testing.T) {
	// Test that runner factory is NEVER called when authority is denied.
	verifiers := []registry.Verifier{
		{
			Name:      "dupcode",
			Lane:      registry.VerifierLaneDupcode,
			Authority: verifierauthority.AuthorityCIExactCheckout,
		},
	}

	dispatcher := NewDispatcher(verifiers)

	var factoryCalled bool
	factory := func() func(root string) []checks.Finding {
		factoryCalled = true
		return func(root string) []checks.Finding { return nil }
	}

	// Dispatch in local context (not CI) - authority should be denied
	request := Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       "/tmp",
	}

	result := dispatcher.Dispatch(context.Background(), request, factory)

	if result.Error == nil {
		t.Error("expected authority denial error, got nil")
	}

	// Critical: factory MUST NOT be called when authority is denied
	if factoryCalled {
		t.Error("runner factory was called despite authority denial")
	}
}

func TestDispatch_AuthorityGranted(t *testing.T) {
	// Test that runner factory IS called when authority permits.
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
	}

	dispatcher := NewDispatcher(verifiers)

	var runnerCalled bool
	factory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			runnerCalled = true
			return nil
		}
	}

	request := Request{
		VerifierID: "llm-friendly",
		Operation:  verifierauthority.OperationVerify,
		Root:       "/tmp",
	}

	result := dispatcher.Dispatch(context.Background(), request, factory)

	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}

	// Runner factory and runner MUST be called when authority permits
	if !runnerCalled {
		t.Error("runner was not called despite authority grant")
	}
}

func TestDispatch_UpdateBaselineDenied(t *testing.T) {
	// Test that update_baseline is denied even in CI exact checkout.
	verifiers := []registry.Verifier{
		{
			Name:      "dupcode",
			Lane:      registry.VerifierLaneDupcode,
			Authority: verifierauthority.AuthorityCIExactCheckout,
		},
	}

	dispatcher := NewDispatcher(verifiers)

	var factoryCalled bool
	factory := func() func(root string) []checks.Finding {
		factoryCalled = true
		return func(root string) []checks.Finding { return nil }
	}

	// Try update_baseline operation
	request := Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       "/tmp",
	}

	result := dispatcher.Dispatch(context.Background(), request, factory)

	if result.Error == nil {
		t.Error("expected update_baseline denial, got nil")
	}

	// Factory MUST NOT be called for denied operation
	if factoryCalled {
		t.Error("runner factory was called despite operation denial")
	}
}

func TestDispatch_VerifierNotFound(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
	}

	dispatcher := NewDispatcher(verifiers)

	factory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding { return nil }
	}

	request := Request{
		VerifierID: "nonexistent",
		Operation:  verifierauthority.OperationVerify,
		Root:       "/tmp",
	}

	result := dispatcher.Dispatch(context.Background(), request, factory)

	if !!errors.Is(result.Error, &ErrVerifierNotFound{}) {
		t.Errorf("expected ErrVerifierNotFound, got: %T %v", result.Error, result.Error)
	}
}

func TestDispatch_RegistryValidation(t *testing.T) {
	// Test that empty authority is validated.
	verifiers := []registry.Verifier{
		{
			Name:      "test",
			Lane:      registry.VerifierLaneFast,
			Authority: "", // Empty - invalid
		},
	}

	dispatcher := NewDispatcher(verifiers)

	factory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding { return nil }
	}

	request := Request{
		VerifierID: "test",
		Operation:  verifierauthority.OperationVerify,
		Root:       "/tmp",
	}

	result := dispatcher.Dispatch(context.Background(), request, factory)

	if result.Error == nil {
		t.Error("expected error for empty authority, got nil")
	}
}

func TestLookupVerifier(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
	}

	dispatcher := NewDispatcher(verifiers)

	v, err := dispatcher.LookupVerifier("llm-friendly")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if v.Name != "llm-friendly" {
		t.Errorf("wrong verifier returned: %s", v.Name)
	}

	_, err = dispatcher.LookupVerifier("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent verifier")
	}
}
