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

// TestAuthorizeAndRunProfileDeniedFactoryNotCalled verifies factory is not called on denial.
func TestAuthorizeAndRunProfileDeniedFactoryNotCalled(t *testing.T) {
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
	result, err := d.AuthorizeAndRunProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "non-existent", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndRunProfile: %v", err)
	}
	if result.Profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to fail")
	}
	if factoryCalled {
		t.Error("factory was called but should not have been")
	}
}

// TestAuthorizeAndRunProfileExecutesExactInventory verifies factory receives exact authorized verifiers.
func TestAuthorizeAndRunProfileExecutesExactInventory(t *testing.T) {
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
		return []BoundProfileRunner{{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }}}, nil
	}
	result, err := d.AuthorizeAndRunProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndRunProfile: %v", err)
	}
	if !result.Profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to succeed")
	}
	if len(received) != 1 || received[0].Name != "v1" {
		t.Errorf("received %v, want [v1]", received)
	}
}

// TestAuthorizeAndRunProfileRejectsMissingRunner verifies contract rejects missing runner.
func TestAuthorizeAndRunProfileRejectsMissingRunner(t *testing.T) {
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
		return []BoundProfileRunner{{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }}}, nil
	}
	result, err := d.AuthorizeAndRunProfile(context.Background(), "/test",
		[]ProfileRequest{
			{VerifierID: "v1", Operation: verifierauthority.OperationVerify},
			{VerifierID: "v2", Operation: verifierauthority.OperationVerify},
		}, nil, factory)
	if err == nil {
		t.Error("expected error for missing runner")
	}
	if result.AllRun {
		t.Error("AllRun should be false when contract violated")
	}
}

// TestAuthorizeAndRunProfileRejectsExtraRunner verifies contract rejects extra runner.
func TestAuthorizeAndRunProfileRejectsExtraRunner(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		return []BoundProfileRunner{
			{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }},
			{VerifierID: "v2", Run: func(root string) []checks.Finding { return nil }}, // extra
		}, nil
	}
	_, err = d.AuthorizeAndRunProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestAuthorizeAndRunProfileRejectsDuplicateRunnerID verifies contract rejects duplicate IDs.
func TestAuthorizeAndRunProfileRejectsDuplicateRunnerID(t *testing.T) {
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
			{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }},
			{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }}, // duplicate
		}, nil
	}
	_, err = d.AuthorizeAndRunProfile(context.Background(), "/test",
		[]ProfileRequest{
			{VerifierID: "v1", Operation: verifierauthority.OperationVerify},
			{VerifierID: "v2", Operation: verifierauthority.OperationVerify},
		}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestAuthorizeAndRunProfileRejectsNilRunner verifies contract rejects nil Run function.
func TestAuthorizeAndRunProfileRejectsNilRunner(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		return []BoundProfileRunner{{VerifierID: "v1", Run: nil}}, nil
	}
	_, err = d.AuthorizeAndRunProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestAuthorizeAndRunProfileRejectsUnknownRunnerID verifies contract rejects unknown IDs.
func TestAuthorizeAndRunProfileRejectsUnknownRunnerID(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		return []BoundProfileRunner{{VerifierID: "unknown", Run: func(root string) []checks.Finding { return nil }}}, nil
	}
	_, err = d.AuthorizeAndRunProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestAuthorizeAndRunProfileUsesAuthorizedRoot verifies runner receives authorized root.
func TestAuthorizeAndRunProfileUsesAuthorizedRoot(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	var receivedRoot string
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		return []BoundProfileRunner{{
			VerifierID: "v1",
			Run: func(root string) []checks.Finding {
				receivedRoot = root
				return nil
			},
		}}, nil
	}
	_, err = d.AuthorizeAndRunProfile(context.Background(), "/authorized/root",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndRunProfile: %v", err)
	}
	if receivedRoot != "/authorized/root" {
		t.Errorf("runner received root %q, want %q", receivedRoot, "/authorized/root")
	}
}

// TestAuthorizeAndRunProfileExecutesEachAuthorizedRunnerOnce verifies each runner executes exactly once.
func TestAuthorizeAndRunProfileExecutesEachAuthorizedRunnerOnce(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
		{Name: "v2", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	v1Calls, v2Calls := 0, 0
	factory := func([]*registry.Verifier) ([]BoundProfileRunner, error) {
		return []BoundProfileRunner{
			{VerifierID: "v1", Run: func(root string) []checks.Finding { v1Calls++; return nil }},
			{VerifierID: "v2", Run: func(root string) []checks.Finding { v2Calls++; return nil }},
		}, nil
	}
	result, err := d.AuthorizeAndRunProfile(context.Background(), "/test",
		[]ProfileRequest{
			{VerifierID: "v1", Operation: verifierauthority.OperationVerify},
			{VerifierID: "v2", Operation: verifierauthority.OperationVerify},
		}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndRunProfile: %v", err)
	}
	if !result.AllRun {
		t.Error("AllRun should be true")
	}
	if v1Calls != 1 {
		t.Errorf("v1 called %d times, want 1", v1Calls)
	}
	if v2Calls != 1 {
		t.Errorf("v2 called %d times, want 1", v2Calls)
	}
}

// TestAuthorizeAndRunProfileFactoryErrorPropagated verifies factory errors propagate.
func TestAuthorizeAndRunProfileFactoryErrorPropagated(t *testing.T) {
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
	_, err = d.AuthorizeAndRunProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err == nil {
		t.Error("expected factory error to propagate")
	}
}
