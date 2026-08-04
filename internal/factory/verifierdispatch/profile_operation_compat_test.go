// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// TestAuthorizeProfileRejectsDisallowedOperation proves the profile path
// rejects a request whose operation is not declared by the selected
// verifier. The profile must deny atomically and never call the
// observer or any downstream factory.
func TestAuthorizeProfileRejectsDisallowedOperation(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "verify-only",
		Authority:  verifierauthority.AuthorityLocalSafe,
		Lane:       registry.VerifierLaneFast,
		Run:        func(root string) []checks.Finding { return nil },
		Scope:      registry.InvocationGate,
		Operations: verifyOnly(),
	}}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &countingObserver{}
	requests := []ProfileRequest{
		{VerifierID: "verify-only", Operation: verifierauthority.OperationUpdateBaseline},
	}
	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, obs)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if profile.AuthorizationSucceeded() {
		t.Error("expected authorization to fail when operation is not declared")
	}
	if obs.counter.Load() != 0 {
		t.Errorf("observer calls = %d, want 0 (denial before observation)", obs.counter.Load())
	}
	if len(profile.VerifierIDs()) != 0 {
		t.Errorf("VerifierIDs = %v, want empty on denial", profile.VerifierIDs())
	}
	if len(profile.Denials()) != 1 {
		t.Fatalf("Denials = %d, want 1", len(profile.Denials()))
	}
	if profile.Denials()[0].Findings[0].Kind != "verifier_operation_not_allowed" {
		t.Errorf("denial kind = %q, want verifier_operation_not_allowed",
			profile.Denials()[0].Findings[0].Kind)
	}
}

// TestAuthorizeProfileAtomicDenialOnMixedRequests proves a profile
// containing a valid verify and an invalid update is denied atomically.
func TestAuthorizeProfileAtomicDenialOnMixedRequests(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:       "verify-only",
			Authority:  verifierauthority.AuthorityLocalSafe,
			Lane:       registry.VerifierLaneFast,
			Run:        func(root string) []checks.Finding { return nil },
			Scope:      registry.InvocationGate,
			Operations: verifyOnly(),
		},
		{
			Name:      "dupcode-update-baseline",
			Lane:      registry.VerifierLaneDupcode,
			Authority: verifierauthority.AuthorityLocalSafe,
			Scope:     registry.InvocationCommandOnly,
			Operations: []verifierauthority.VerifierOperation{
				verifierauthority.OperationUpdateBaseline,
			},
		},
	}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	obs := &countingObserver{}
	requests := []ProfileRequest{
		{VerifierID: "verify-only", Operation: verifierauthority.OperationVerify},
		{VerifierID: "dupcode-update-baseline", Operation: verifierauthority.OperationVerify},
	}
	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, obs)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if profile.AuthorizationSucceeded() {
		t.Error("expected profile to fail when any request has disallowed operation")
	}
	if len(profile.VerifierIDs()) != 0 {
		t.Errorf("VerifierIDs = %v, want empty on atomic denial", profile.VerifierIDs())
	}
	if obs.counter.Load() != 0 {
		t.Errorf("observer calls = %d, want 0 on atomic denial", obs.counter.Load())
	}
}
