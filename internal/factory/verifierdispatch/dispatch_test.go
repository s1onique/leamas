// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// fakeObserver is a test double for ContextObserver.
type fakeObserver struct {
	ObserveCount int
	Context      verifierauthority.ExecutionContext
}

func (f *fakeObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	f.ObserveCount++
	f.Context = verifierauthority.ExecutionContext{
		GitHubActions: "true",
		GitHubSHA:     "abc123",
	}
	return f.Context
}

// fakeLocalObserver is a test double that never observes (local-only).
type fakeLocalObserver struct {
	ObserveCount int
}

func (f *fakeLocalObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	f.ObserveCount++
	return verifierauthority.ExecutionContext{}
}

// testVerifier creates a test verifier with the given parameters.
func testVerifier(name string, authority verifierauthority.ExecutionAuthority, lane registry.VerifierLane) registry.Verifier {
	return registry.Verifier{
		Name:       name,
		Authority:  authority,
		Lane:       lane,
		Scope:      registry.InvocationGate,
		Operations: verifyOnly(),
		Run:        func(root string) []checks.Finding { return nil },
	}
}

// testMutationVerifier creates a command-only mutation verifier.
func testMutationVerifier(name string) registry.Verifier {
	return registry.Verifier{
		Name:      name,
		Lane:      registry.VerifierLaneDupcode,
		Authority: verifierauthority.AuthorityLocalSafe,
		Scope:     registry.InvocationCommandOnly,
		Operations: []verifierauthority.VerifierOperation{
			verifierauthority.OperationUpdateBaseline,
		},
	}
}

func TestAuthorizeProfileLocalSafeDoesNotObserve(t *testing.T) {
	verifiers := []registry.Verifier{
		testVerifier("local-safe", verifierauthority.AuthorityLocalSafe, registry.VerifierLaneFast),
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	observer := &fakeObserver{}

	requests := []ProfileRequest{
		{VerifierID: "local-safe", Operation: verifierauthority.OperationVerify},
	}

	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, observer)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}

	if !profile.AuthorizationSucceeded() {
		t.Error("expected AuthorizationSucceeded=true for local_safe verifier")
	}

	if observer.ObserveCount != 0 {
		t.Errorf("observer.ObserveCount = %d, want 0 (local_safe should not call observer)", observer.ObserveCount)
	}
}

func TestAuthorizeProfileObservesRemoteContextOnce(t *testing.T) {
	// Test that observer is called exactly once for CI authority.
	// Note: The authorization itself may fail if not in a real CI environment,
	// but we can still verify the observer call count.
	verifiers := []registry.Verifier{
		testVerifier("ci-verifier", verifierauthority.AuthorityCIExactCheckout, registry.VerifierLaneDupcode),
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	observer := &fakeObserver{}

	requests := []ProfileRequest{
		{VerifierID: "ci-verifier", Operation: verifierauthority.OperationVerify},
	}

	_, err = dispatcher.AuthorizeProfile(context.Background(), "/test", requests, observer)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}

	// Observer should be called exactly once for CI authority
	if observer.ObserveCount != 1 {
		t.Errorf("observer.ObserveCount = %d, want 1", observer.ObserveCount)
	}
}

func TestAuthorizeProfileAllOrNothing(t *testing.T) {
	// Only register one verifier, but request two (one doesn't exist)
	verifiers := []registry.Verifier{
		testVerifier("authorized", verifierauthority.AuthorityLocalSafe, registry.VerifierLaneFast),
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	observer := &fakeLocalObserver{}

	// Request one valid and one that doesn't exist in registry
	requests := []ProfileRequest{
		{VerifierID: "authorized", Operation: verifierauthority.OperationVerify},
		{VerifierID: "not-in-registry", Operation: verifierauthority.OperationVerify},
	}

	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, observer)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}

	// All-or-nothing: should fail because one verifier is not found in registry
	if profile.AuthorizationSucceeded() {
		t.Error("expected AuthorizationSucceeded=false when any verifier is denied")
	}

	if len(profile.Denials()) != 1 {
		t.Errorf("len(profile.Denials()) = %d, want 1", len(profile.Denials()))
	}

	if len(profile.VerifierIDs()) != 0 {
		t.Errorf("len(profile.VerifierIDs()) = %d, want 0 (all-or-nothing)", len(profile.VerifierIDs()))
	}
}

func TestAuthorizeProfileRejectsNotFound(t *testing.T) {
	// Register one verifier but request a different one
	verifiers := []registry.Verifier{
		testVerifier("registered", verifierauthority.AuthorityLocalSafe, registry.VerifierLaneFast),
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	observer := &fakeLocalObserver{}

	// Request a verifier that doesn't exist
	requests := []ProfileRequest{
		{VerifierID: "not-registered", Operation: verifierauthority.OperationVerify},
	}

	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, observer)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}

	if profile.AuthorizationSucceeded() {
		t.Error("expected AuthorizationSucceeded=false for not-found verifier")
	}

	if len(profile.Denials()) != 1 {
		t.Errorf("len(profile.Denials()) = %d, want 1", len(profile.Denials()))
	}

	if profile.Denials()[0].VerifierID != "not-registered" {
		t.Errorf("denial.VerifierID = %q, want %q", profile.Denials()[0].VerifierID, "not-registered")
	}
}

func TestAuthorizedProfileBindsRootAndOperations(t *testing.T) {
	verifiers := []registry.Verifier{
		testVerifier("bound", verifierauthority.AuthorityLocalSafe, registry.VerifierLaneFast),
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	observer := &fakeLocalObserver{}

	requests := []ProfileRequest{
		{VerifierID: "bound", Operation: verifierauthority.OperationVerify},
	}

	const wantRoot = "/bound/root"
	profile, err := dispatcher.AuthorizeProfile(context.Background(), wantRoot, requests, observer)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}

	if profile.RepositoryRoot() != wantRoot {
		t.Errorf("profile.RepositoryRoot() = %q, want %q", profile.RepositoryRoot(), wantRoot)
	}

	if len(profile.Requests()) != 1 {
		t.Fatalf("len(profile.Requests()) = %d, want 1", len(profile.Requests()))
	}

	if profile.Requests()[0].VerifierID != "bound" {
		t.Errorf("profile.Requests()[0].VerifierID = %q, want %q", profile.Requests()[0].VerifierID, "bound")
	}

	if profile.Requests()[0].Operation != verifierauthority.OperationVerify {
		t.Errorf("profile.Requests()[0].Operation = %v, want OperationVerify", profile.Requests()[0].Operation)
	}
}

func TestAuthorizeProfileMultipleLocalVerifiers(t *testing.T) {
	// Test that multiple local verifiers can be authorized together
	verifiers := []registry.Verifier{
		testVerifier("local1", verifierauthority.AuthorityLocalSafe, registry.VerifierLaneFast),
		testVerifier("local2", verifierauthority.AuthorityLocalSafe, registry.VerifierLaneFast),
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	observer := &fakeLocalObserver{}

	requests := []ProfileRequest{
		{VerifierID: "local1", Operation: verifierauthority.OperationVerify},
		{VerifierID: "local2", Operation: verifierauthority.OperationVerify},
	}

	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, observer)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}

	if !profile.AuthorizationSucceeded() {
		t.Error("expected AuthorizationSucceeded=true for multiple local verifiers")
	}

	// Observer should NOT be called for local-only verifiers
	if observer.ObserveCount != 0 {
		t.Errorf("observer.ObserveCount = %d, want 0 for local verifiers", observer.ObserveCount)
	}

	// Both should be authorized
	if len(profile.VerifierIDs()) != 2 {
		t.Errorf("len(profile.VerifierIDs()) = %d, want 2", len(profile.VerifierIDs()))
	}
}

func TestAuthorizeProfileUpdateBaselineDeniedForCI(t *testing.T) {
	verifiers := []registry.Verifier{
		testVerifier("ci-baseline", verifierauthority.AuthorityCIExactCheckout, registry.VerifierLaneDupcode),
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	observer := &fakeObserver{}

	requests := []ProfileRequest{
		{VerifierID: "ci-baseline", Operation: verifierauthority.OperationUpdateBaseline},
	}

	profile, err := dispatcher.AuthorizeProfile(context.Background(), "/test", requests, observer)
	if err != nil {
		t.Fatalf("AuthorizeProfile: %v", err)
	}

	// update_baseline should be denied for ci_exact_checkout
	if profile.AuthorizationSucceeded() {
		t.Error("expected AuthorizationSucceeded=false for update_baseline on ci_exact_checkout")
	}

	if len(profile.Denials()) != 1 {
		t.Errorf("len(profile.Denials()) = %d, want 1", len(profile.Denials()))
	}
}
