// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// TestAuthorizeAndRunProfileDeniedFactoryCountZero verifies that when authorization fails,
// the factory is NOT called (factory count remains zero).
func TestAuthorizeAndRunProfileDeniedFactoryCountZero(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:      "local-only",
		Authority: verifierauthority.AuthorityLocalSafe,
		Lane:      registry.VerifierLaneFast,
		Run:       func(root string) []checks.Finding { return nil },
	}}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	requests := []ProfileRequest{{VerifierID: "non-existent", Operation: verifierauthority.OperationVerify}}

	var factoryCalls int
	factory := func(authorized []*registry.Verifier) []func(root string) []checks.Finding {
		factoryCalls++
		if len(authorized) != 0 {
			t.Errorf("factory received %d verifiers, want 0", len(authorized))
		}
		return nil
	}

	result, err := dispatcher.AuthorizeAndRunProfile(context.Background(), "/test", requests, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndRunProfile: %v", err)
	}
	if result.Profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to fail")
	}
	if factoryCalls != 0 {
		t.Errorf("factory called %d times, want 0", factoryCalls)
	}
}

// TestAuthorizeAndRunProfileExecutesExactInventory verifies that the factory receives
// exactly the authorized verifiers, not the full registry.
func TestAuthorizeAndRunProfileExecutesExactInventory(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "verifier-1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
		{Name: "verifier-2", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	requests := []ProfileRequest{{VerifierID: "verifier-1", Operation: verifierauthority.OperationVerify}}

	var received []*registry.Verifier
	factory := func(authorized []*registry.Verifier) []func(root string) []checks.Finding {
		received = authorized
		return []func(root string) []checks.Finding{func(root string) []checks.Finding { return nil }}
	}

	result, err := dispatcher.AuthorizeAndRunProfile(context.Background(), "/test", requests, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndRunProfile: %v", err)
	}
	if !result.Profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to succeed")
	}
	if len(received) != 1 {
		t.Fatalf("factory received %d verifiers, want 1", len(received))
	}
	if received[0].Name != "verifier-1" {
		t.Errorf("factory received %q, want %q", received[0].Name, "verifier-1")
	}
}

// TestAuthorizeAndRunProfileDenialReturnsEmptyResults verifies that on denial, results is nil.
func TestAuthorizeAndRunProfileDenialReturnsEmptyResults(t *testing.T) {
	verifiers := []registry.Verifier{{Name: "local", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }}}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	factory := func(authorized []*registry.Verifier) []func(root string) []checks.Finding { return nil }

	result, err := dispatcher.AuthorizeAndRunProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "non-existent", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndRunProfile: %v", err)
	}
	if result.Profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to fail")
	}
	if result.Results != nil {
		t.Errorf("Results is %v, want nil", result.Results)
	}
	if result.AllFound {
		t.Error("AllFound is true, want false")
	}
}
