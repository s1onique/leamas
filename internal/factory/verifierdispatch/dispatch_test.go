// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// fakeObserver tracks context observation calls.
type fakeObserver struct {
	count int
	ec    verifierauthority.ExecutionContext
}

func (f *fakeObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	f.count++
	return f.ec
}

func TestDispatch_AuthorityDenied(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "dupcode",
			Lane:      registry.VerifierLaneDupcode,
			Authority: verifierauthority.AuthorityCIExactCheckout,
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	observer := &fakeObserver{ec: verifierauthority.ExecutionContext{}}

	var factoryCalled bool
	factory := func() func(root string) []checks.Finding {
		factoryCalled = true
		return func(root string) []checks.Finding { return nil }
	}

	request := Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       "/tmp",
	}

	result := dispatcher.Dispatch(context.Background(), request, observer, factory)

	if result.Error == nil {
		t.Error("expected authority denial error, got nil")
	}

	if observer.count != 1 {
		t.Errorf("expected observer count 1, got %d", observer.count)
	}

	if factoryCalled {
		t.Error("runner factory was called despite authority denial")
	}
}

func TestDispatch_LocalSafeSkipsObserver(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	observer := &fakeObserver{}

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

	result := dispatcher.Dispatch(context.Background(), request, observer, factory)

	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}

	if observer.count != 0 {
		t.Errorf("expected observer count 0 for local_safe, got %d", observer.count)
	}

	if !runnerCalled {
		t.Error("runner was not called despite authority grant")
	}
}

func TestDispatch_AuthorityGranted(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	observer := &fakeObserver{}

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

	result := dispatcher.Dispatch(context.Background(), request, observer, factory)

	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}

	if !runnerCalled {
		t.Error("runner was not called despite authority grant")
	}
}

func TestDispatch_UpdateBaselineDenied(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "dupcode",
			Lane:      registry.VerifierLaneDupcode,
			Authority: verifierauthority.AuthorityCIExactCheckout,
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	observer := &fakeObserver{
		ec: verifierauthority.ExecutionContext{
			CI:              "true",
			GitHubActions:   "true",
			AuthorityMarker: verifierauthority.AuthorityMarker,
			GitHubSHA:       strings.Repeat("a", 40),
			GitHubWorkspace: "/repo",
			HeadCommit:      strings.Repeat("a", 40),
			WorktreeStatus:  "",
			RepositoryRoot:  "/repo",
			WorkspaceRoot:   "/repo",
		},
	}

	var factoryCalled bool
	factory := func() func(root string) []checks.Finding {
		factoryCalled = true
		return func(root string) []checks.Finding { return nil }
	}

	request := Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       "/tmp",
	}

	result := dispatcher.Dispatch(context.Background(), request, observer, factory)

	if result.Error == nil {
		t.Error("expected update_baseline denial, got nil")
	}

	if observer.count != 1 {
		t.Errorf("expected observer count 1, got %d", observer.count)
	}

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

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	observer := &fakeObserver{}

	factory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding { return nil }
	}

	request := Request{
		VerifierID: "nonexistent",
		Operation:  verifierauthority.OperationVerify,
		Root:       "/tmp",
	}

	result := dispatcher.Dispatch(context.Background(), request, observer, factory)

	var target *ErrVerifierNotFound
	if !errors.As(result.Error, &target) {
		t.Errorf("expected ErrVerifierNotFound, got: %T %v", result.Error, result.Error)
	}

	if observer.count != 0 {
		t.Errorf("expected observer count 0 for missing verifier, got %d", observer.count)
	}
}

func TestDispatch_CIExactCheckoutDenied(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "dupcode",
			Lane:      registry.VerifierLaneDupcode,
			Authority: verifierauthority.AuthorityCIExactCheckout,
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	observer := &fakeObserver{ec: verifierauthority.ExecutionContext{}}

	var factoryCalled bool
	factory := func() func(root string) []checks.Finding {
		factoryCalled = true
		return func(root string) []checks.Finding { return nil }
	}

	request := Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       "/tmp",
	}

	result := dispatcher.Dispatch(context.Background(), request, observer, factory)

	if result.Error == nil {
		t.Error("expected authority denial error, got nil")
	}

	if observer.count != 1 {
		t.Errorf("expected observer count 1, got %d", observer.count)
	}

	if factoryCalled {
		t.Error("runner factory was called despite authority denial")
	}
}

func TestDispatch_PropagatesVerifierResult(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "llm-friendly",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	observer := &fakeObserver{}

	expectedFindings := []checks.Finding{
		{
			Path:     "test.go",
			Kind:     "error",
			Message:  "test error",
			Severity: checks.SeverityError,
		},
	}

	factory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			return expectedFindings
		}
	}

	request := Request{
		VerifierID: "llm-friendly",
		Operation:  verifierauthority.OperationVerify,
		Root:       "/tmp",
	}

	result := dispatcher.Dispatch(context.Background(), request, observer, factory)

	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}

	if len(result.Findings) != len(expectedFindings) {
		t.Errorf("expected %d findings, got %d", len(expectedFindings), len(result.Findings))
	}

	if result.Findings[0].Message != expectedFindings[0].Message {
		t.Errorf("wrong message: %s", result.Findings[0].Message)
	}
}

func TestDispatch_RegistryDefensivelyCopied(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "test",
			Lane:      registry.VerifierLaneFast,
			Authority: verifierauthority.AuthorityLocalSafe,
		},
	}

	dispatcher, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	verifiers[0].Name = "modified"

	v, err := dispatcher.LookupVerifier("test")
	if err != nil {
		t.Errorf("verifier test should still exist: %v", err)
	}
	if v.Name != "test" {
		t.Errorf("expected verifier name 'test', got %q", v.Name)
	}

	_, err = dispatcher.LookupVerifier("modified")
	if err == nil {
		t.Error("modified verifier should not exist")
	}
}
