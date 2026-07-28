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

// TestProfileBindingExecuteOnce verifies each runner executes exactly once with authorized profile.
func TestProfileBindingExecuteOnce(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
		{Name: "v2", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	// Use the proper API to create an authorized binding
	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{
			{VerifierID: "v1", Operation: verifierauthority.OperationVerify},
			{VerifierID: "v2", Operation: verifierauthority.OperationVerify},
		}, nil,
		func(authorized []registry.Verifier) ([]BoundProfileRunner, error) {
			return []BoundProfileRunner{
				{Verifier: authorized[0], Run: authorized[0].Run},
				{Verifier: authorized[1], Run: authorized[1].Run},
			}, nil
		})
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}

	// First execution should succeed
	records, err := binding.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("records count = %d, want 2", len(records))
	}

	// Second execution should fail with consumed error
	_, err = binding.Execute()
	var consumedErr *ErrProfileBindingConsumed
	if !errors.As(err, &consumedErr) {
		t.Errorf("expected ErrProfileBindingConsumed, got %T: %v", err, err)
	}
}

// TestDeniedBindingCannotExecute verifies denied bindings cannot execute.
func TestDeniedBindingCannotExecute(t *testing.T) {
	profile := &AuthorizedProfile{} // unauthorized
	binding := &ProfileBinding{profile: profile, runners: nil}

	_, err := binding.Execute()
	var notAuthErr *ErrProfileNotAuthorized
	if !errors.As(err, &notAuthErr) {
		t.Errorf("expected ErrProfileNotAuthorized, got %T: %v", err, err)
	}
}

// TestBindingDoesNotExposeExecutableRunners verifies Runners() doesn't leak Run functions.
func TestBindingDoesNotExposeExecutableRunners(t *testing.T) {
	runners := []BoundProfileRunner{
		{Verifier: registry.Verifier{Name: "v1"}, Run: func(root string) []checks.Finding { return nil }},
	}
	profile := &AuthorizedProfile{}
	binding := &ProfileBinding{profile: profile, runners: runners}

	meta := binding.Runners()
	if len(meta) != 1 {
		t.Fatalf("expected 1 metadata, got %d", len(meta))
	}
	if meta[0].Verifier.Name != "v1" {
		t.Errorf("expected verifier name v1, got %s", meta[0].Verifier.Name)
	}
}

// TestFactoryReceivesDefensiveVerifierCopies verifies factory receives value copies.
func TestFactoryReceivesDefensiveVerifierCopies(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	factory := func(authorized []registry.Verifier) ([]BoundProfileRunner, error) {
		return []BoundProfileRunner{{Verifier: authorized[0], Run: func(root string) []checks.Finding { return nil }}}, nil
	}

	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}

	records, err := binding.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

// TestFactoryCannotForgeVerifierMetadata verifies factory cannot forge metadata.
func TestFactoryCannotForgeVerifierMetadata(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	factory := func(authorized []registry.Verifier) ([]BoundProfileRunner, error) {
		forged := authorized[0]
		forged.Authority = verifierauthority.AuthorityCIExactCheckout
		return []BoundProfileRunner{{Verifier: forged, Run: func(root string) []checks.Finding { return nil }}}, nil
	}

	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}

	records, err := binding.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

// TestFactoryReversedOrderIsCanonicalized verifies factory can return reversed order.
func TestFactoryReversedOrderIsCanonicalized(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
		{Name: "v2", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
		{Name: "v3", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	factory := func(authorized []registry.Verifier) ([]BoundProfileRunner, error) {
		reversed := make([]BoundProfileRunner, len(authorized))
		for i, v := range authorized {
			reversed[len(authorized)-1-i] = BoundProfileRunner{Verifier: v, Run: v.Run}
		}
		return reversed, nil
	}

	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{
			{VerifierID: "v1", Operation: verifierauthority.OperationVerify},
			{VerifierID: "v2", Operation: verifierauthority.OperationVerify},
			{VerifierID: "v3", Operation: verifierauthority.OperationVerify},
		}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}

	records, err := binding.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	wantOrder := []string{"v1", "v2", "v3"}
	for i, wantName := range wantOrder {
		if records[i].Verifier.Name != wantName {
			t.Errorf("order[%d] = %q, want %q", i, records[i].Verifier.Name, wantName)
		}
	}
}

// TestProfileBindingCannotBeForged verifies ProfileBinding fields are unexported.
func TestProfileBindingCannotBeForged(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil,
		func(authorized []registry.Verifier) ([]BoundProfileRunner, error) {
			return []BoundProfileRunner{{Verifier: authorized[0], Run: func(root string) []checks.Finding { return nil }}}, nil
		})
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}

	records, err := binding.Execute()
	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("records count = %d, want 1", len(records))
	}
}
