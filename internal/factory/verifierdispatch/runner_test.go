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

// TestAuthorizeAndBindProfileDeniedFactoryNotCalled verifies factory is not called on denial.
func TestAuthorizeAndBindProfileDeniedFactoryNotCalled(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name: "local", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast,
		Run: func(root string) []checks.Finding { return nil },
	}}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factoryCalled := false
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		factoryCalled = true
		return nil, nil
	}
	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "non-existent", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}
	if binding.Profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to fail")
	}
	if factoryCalled {
		t.Error("factory was called but should not have been")
	}
}

// TestAuthorizeAndBindProfileExecutesExactInventory verifies factory receives exact authorized verifiers.
func TestAuthorizeAndBindProfileExecutesExactInventory(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
		{Name: "v2", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	var received []*registry.Verifier
	factory := func(authorized []*registry.Verifier) ([]BoundProfileRunner, error) {
		received = authorized
		return []BoundProfileRunner{{Verifier: registry.Verifier{Name: "v1"}, Run: func(root string) []checks.Finding { return nil }}}, nil
	}
	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}
	if !binding.Profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to succeed")
	}
	if len(received) != 1 || received[0].Name != "v1" {
		t.Errorf("received %v, want [v1]", received)
	}
}

// TestAuthorizeAndBindProfileRejectsMissingRunner verifies contract rejects missing runner.
func TestAuthorizeAndBindProfileRejectsMissingRunner(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
		{Name: "v2", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		// Only return v1, missing v2
		return []BoundProfileRunner{{Verifier: registry.Verifier{Name: "v1"}, Run: func(root string) []checks.Finding { return nil }}}, nil
	}
	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{
			{VerifierID: "v1", Operation: verifierauthority.OperationVerify},
			{VerifierID: "v2", Operation: verifierauthority.OperationVerify},
		}, nil, factory)
	if err == nil {
		t.Error("expected error for missing runner")
	}
	if len(binding.Runners) != 0 {
		t.Errorf("binding has %d runners, want 0", len(binding.Runners))
	}
}

// TestAuthorizeAndBindProfileRejectsExtraRunner verifies contract rejects extra runner.
func TestAuthorizeAndBindProfileRejectsExtraRunner(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		return []BoundProfileRunner{
			{Verifier: registry.Verifier{Name: "v1"}, Run: func(root string) []checks.Finding { return nil }},
			{Verifier: registry.Verifier{Name: "v2"}, Run: func(root string) []checks.Finding { return nil }}, // extra
		}, nil
	}
	_, err = d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestAuthorizeAndBindProfileRejectsDuplicateRunnerID verifies contract rejects duplicate IDs.
func TestAuthorizeAndBindProfileRejectsDuplicateRunnerID(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
		{Name: "v2", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		return []BoundProfileRunner{
			{Verifier: registry.Verifier{Name: "v1"}, Run: func(root string) []checks.Finding { return nil }},
			{Verifier: registry.Verifier{Name: "v1"}, Run: func(root string) []checks.Finding { return nil }}, // duplicate
		}, nil
	}
	_, err = d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{
			{VerifierID: "v1", Operation: verifierauthority.OperationVerify},
			{VerifierID: "v2", Operation: verifierauthority.OperationVerify},
		}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestAuthorizeAndBindProfileRejectsNilRunner verifies contract rejects nil Run function.
func TestAuthorizeAndBindProfileRejectsNilRunner(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		return []BoundProfileRunner{{Verifier: registry.Verifier{Name: "v1"}, Run: nil}}, nil
	}
	_, err = d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestAuthorizeAndBindProfileRejectsUnknownRunnerID verifies contract rejects unknown IDs.
func TestAuthorizeAndBindProfileRejectsUnknownRunnerID(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		return []BoundProfileRunner{{Verifier: registry.Verifier{Name: "unknown"}, Run: func(root string) []checks.Finding { return nil }}}, nil
	}
	_, err = d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestAuthorizeAndBindProfileFactoryErrorPropagated verifies factory errors propagate.
func TestAuthorizeAndBindProfileFactoryErrorPropagated(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		return nil, errors.New("baseline load failed")
	}
	_, err = d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err == nil {
		t.Error("expected factory error to propagate")
	}
}

// TestProfileBindingExecutesBoundRunnersOnce verifies each runner executes exactly once.
func TestProfileBindingExecutesBoundRunnersOnce(t *testing.T) {
	v1Calls, v2Calls := 0, 0
	runners := []BoundProfileRunner{
		{Verifier: registry.Verifier{Name: "v1"}, Run: func(root string) []checks.Finding { v1Calls++; return nil }},
		{Verifier: registry.Verifier{Name: "v2"}, Run: func(root string) []checks.Finding { v2Calls++; return nil }},
	}

	profile := &AuthorizedProfile{}
	binding := &ProfileBinding{Profile: profile, Runners: runners}

	var executedRunners []string
	err := binding.ExecuteBoundRunners(func(p *AuthorizedProfile, r []BoundProfileRunner) error {
		for _, runner := range r {
			executedRunners = append(executedRunners, runner.Verifier.Name)
			runner.Run(p.RepositoryRoot())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ExecuteBoundRunners: %v", err)
	}

	if v1Calls != 1 {
		t.Errorf("v1 called %d times, want 1", v1Calls)
	}
	if v2Calls != 1 {
		t.Errorf("v2 called %d times, want 1", v2Calls)
	}
	if len(executedRunners) != 2 {
		t.Errorf("executed %d runners, want 2", len(executedRunners))
	}
}

// TestProfileBindingPreservesRunnerOrder verifies runners execute in canonical order.
func TestProfileBindingPreservesRunnerOrder(t *testing.T) {
	runners := []BoundProfileRunner{
		{Verifier: registry.Verifier{Name: "alpha"}, Run: func(root string) []checks.Finding { return nil }},
		{Verifier: registry.Verifier{Name: "beta"}, Run: func(root string) []checks.Finding { return nil }},
		{Verifier: registry.Verifier{Name: "gamma"}, Run: func(root string) []checks.Finding { return nil }},
	}

	profile := &AuthorizedProfile{}
	binding := &ProfileBinding{Profile: profile, Runners: runners}

	var executedOrder []string
	binding.ExecuteBoundRunners(func(p *AuthorizedProfile, r []BoundProfileRunner) error {
		for _, runner := range r {
			executedOrder = append(executedOrder, runner.Verifier.Name)
			runner.Run(p.RepositoryRoot())
		}
		return nil
	})

	want := []string{"alpha", "beta", "gamma"}
	for i, wantName := range want {
		if executedOrder[i] != wantName {
			t.Errorf("order[%d] = %q, want %q", i, executedOrder[i], wantName)
		}
	}
}
