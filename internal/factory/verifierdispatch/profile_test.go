// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"fmt"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// verifyOnly returns the canonical ordinary verifier operation list.
func verifyOnly() []verifierauthority.VerifierOperation {
	return []verifierauthority.VerifierOperation{verifierauthority.OperationVerify}
}

func TestAuthorizedProfileRequestsAreDefensivelyCopied(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "fast-local",
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
	requests := []ProfileRequest{{VerifierID: "fast-local", Operation: verifierauthority.OperationVerify}}
	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, nil)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if !profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to succeed")
	}
	returned := profile.Requests()
	if len(returned) > 0 {
		returned[0].VerifierID = "mutated"
		again := profile.Requests()
		if again[0].VerifierID != "fast-local" {
			t.Errorf("internal state was mutated: got %q, want %q", again[0].VerifierID, "fast-local")
		}
	}
}

func TestAuthorizedProfileVerifierIDsAreDefensivelyCopied(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "fast-local-ids",
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
	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "fast-local-ids", Operation: verifierauthority.OperationVerify}}, nil)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if !profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to succeed")
	}
	returned := profile.VerifierIDs()
	if len(returned) > 0 {
		returned[0] = "mutated"
		again := profile.VerifierIDs()
		if again[0] != "fast-local-ids" {
			t.Errorf("internal state was mutated: got %q, want %q", again[0], "fast-local-ids")
		}
	}
}

type fakeObserverForDenial struct{}

func (f *fakeObserverForDenial) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{GitHubActions: "true", GitHubSHA: "abc123"}
}

func TestAuthorizedProfileDenialsAreDeepCopied(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "ci-fast",
		Authority:  verifierauthority.AuthorityCIExactCheckout,
		Lane:       registry.VerifierLaneDupcode,
		Run:        func(root string) []checks.Finding { return nil },
		Scope:      registry.InvocationGate,
		Operations: verifyOnly(),
	}}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "ci-fast", Operation: verifierauthority.OperationVerify}}, &fakeObserverForDenial{})
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to fail")
	}
	denials := profile.Denials()
	if len(denials) == 0 {
		t.Fatal("expected at least one denial")
	}
	denials[0].VerifierID = "mutated-id"
	denials[0].Findings[0].Message = "mutated-message"
	again := profile.Denials()
	if again[0].VerifierID != "ci-fast" {
		t.Errorf("internal denial.VerifierID was mutated: got %q", again[0].VerifierID)
	}
	if again[0].Findings[0].Message == "mutated-message" {
		t.Error("internal denial.Findings[0].Message was mutated")
	}
}

func TestAuthorizedProfileContextIsCloned(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "local-safe",
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
	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "local-safe", Operation: verifierauthority.OperationVerify}}, nil)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if !profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to succeed")
	}
	if ctx := profile.Context(); ctx != nil {
		t.Error("expected nil context for local-safe authority")
	}
}

func TestProfileDigestChangesOnOperationDrift(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "local-test",
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
	profile1, err := dispatcher.AuthorizeProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "local-test", Operation: verifierauthority.OperationVerify}}, obs)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	profile2, err := dispatcher.AuthorizeProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "local-test", Operation: verifierauthority.OperationUpdateBaseline}}, obs)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if profile1.RegistryDigest() == profile2.RegistryDigest() {
		t.Error("digests should differ when operation differs")
	}
}

func TestProfileDigestChangesOnImplementationDrift(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "verifier",
		Authority:  verifierauthority.AuthorityLocalSafe,
		Lane:       registry.VerifierLaneFast,
		Scope:      registry.InvocationGate,
		Operations: verifyOnly(),
		Execution: registry.ExecutionDefinition{
			Kind:             registry.ExecutionInProcess,
			ImplementationID: "impl-v1",
		},
		Run: func(root string) []checks.Finding { return nil },
	}}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	requests := []ProfileRequest{{VerifierID: "verifier", Operation: verifierauthority.OperationVerify}}
	profile1, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, nil)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if v := dispatcher.resolveVerifier("verifier"); v != nil {
		v.Execution.ImplementationID = "impl-v2"
	}
	profile2, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, nil)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if profile1.RegistryDigest() == profile2.RegistryDigest() {
		t.Error("digests should differ when implementation ID differs")
	}
}

func TestProfileDigestHasNoFixedSizePanic(t *testing.T) {
	verifiers := make([]registry.Verifier, 100)
	for i := 0; i < 100; i++ {
		verifiers[i] = registry.Verifier{
			Name:       fmt.Sprintf("verifier-stress-%03d", i),
			Authority:  verifierauthority.AuthorityLocalSafe,
			Lane:       registry.VerifierLaneFast,
			Scope:      registry.InvocationGate,
			Operations: verifyOnly(),
			Execution: registry.ExecutionDefinition{
				Kind:             registry.ExecutionInProcess,
				ImplementationID: "impl-with-very-long-name-that-adds-bytes",
			},
			Run: func(root string) []checks.Finding { return nil },
		}
	}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	requests := make([]ProfileRequest, len(verifiers))
	for i, v := range verifiers {
		requests[i] = ProfileRequest{VerifierID: v.Name, Operation: verifierauthority.OperationVerify}
	}
	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, nil)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}
	if !profile.AuthorizationSucceeded() {
		t.Fatal("expected authorization to succeed")
	}
	if profile.RegistryDigest() == [32]byte{} {
		t.Error("digest should not be zero")
	}
}

func TestAuthorizeProfileRejectsNilObserverForRemoteAuthority(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "ci-only",
		Authority:  verifierauthority.AuthorityCIExactCheckout,
		Lane:       registry.VerifierLaneDupcode,
		Run:        func(root string) []checks.Finding { return nil },
		Scope:      registry.InvocationGate,
		Operations: verifyOnly(),
	}}
	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	_, err = dispatcher.AuthorizeProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "ci-only", Operation: verifierauthority.OperationVerify}}, nil)
	if err == nil {
		t.Error("expected error for nil observer with remote authority")
	}
}

func TestAuthorizeProfileRejectsDuplicateRequests(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "local-test",
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
	requests := []ProfileRequest{
		{VerifierID: "local-test", Operation: verifierauthority.OperationVerify},
		{VerifierID: "local-test", Operation: verifierauthority.OperationVerify},
	}
	_, err = dispatcher.AuthorizeProfile(context.Background(), "/test", requests, nil)
	if err == nil {
		t.Error("expected error for duplicate requests")
	}
}

func TestAuthorizeProfileRejectsEmptyRoot(t *testing.T) {
	verifiers := []registry.Verifier{{
		Name:       "local-test",
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
	_, err = dispatcher.AuthorizeProfile(context.Background(), "",
		[]ProfileRequest{{VerifierID: "local-test", Operation: verifierauthority.OperationVerify}}, nil)
	if err == nil {
		t.Error("expected error for empty root")
	}
}

func TestNewDispatcherRejectsEmptyVerifierSlice(t *testing.T) {
	_, err := NewDispatcher([]registry.Verifier{})
	if err == nil {
		t.Error("expected error for empty verifier slice")
	}
}
