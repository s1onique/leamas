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
	factory := func([]registry.Verifier) ([]FactoryRunner, error) {
		factoryCalled = true
		return nil, nil
	}
	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "non-existent", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}
	if binding.Profile().AuthorizationSucceeded() {
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
	var received []registry.Verifier
	factory := func(authorized []registry.Verifier) ([]FactoryRunner, error) {
		received = authorized
		return []FactoryRunner{{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }}}, nil
	}
	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}
	if !binding.Profile().AuthorizationSucceeded() {
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
	factory := func([]registry.Verifier) ([]FactoryRunner, error) {
		// Only return v1, missing v2
		return []FactoryRunner{{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }}}, nil
	}
	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{
			{VerifierID: "v1", Operation: verifierauthority.OperationVerify},
			{VerifierID: "v2", Operation: verifierauthority.OperationVerify},
		}, nil, factory)
	if err == nil {
		t.Error("expected error for missing runner")
	}
	if len(binding.Runners()) != 0 {
		t.Errorf("binding has %d runners, want 0", len(binding.Runners()))
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
	factory := func([]registry.Verifier) ([]FactoryRunner, error) {
		return []FactoryRunner{
			{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }},
			{VerifierID: "v2", Run: func(root string) []checks.Finding { return nil }}, // extra
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
	factory := func([]registry.Verifier) ([]FactoryRunner, error) {
		return []FactoryRunner{
			{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }},
			{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }}, // duplicate
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
	factory := func([]registry.Verifier) ([]FactoryRunner, error) {
		return []FactoryRunner{{VerifierID: "v1", Run: nil}}, nil
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
	factory := func([]registry.Verifier) ([]FactoryRunner, error) {
		return []FactoryRunner{{VerifierID: "unknown", Run: func(root string) []checks.Finding { return nil }}}, nil
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
	factory := func([]registry.Verifier) ([]FactoryRunner, error) {
		return nil, errors.New("baseline load failed")
	}
	_, err = d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err == nil {
		t.Error("expected factory error to propagate")
	}
}
