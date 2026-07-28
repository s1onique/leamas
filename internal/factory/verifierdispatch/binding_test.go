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

	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{
			{VerifierID: "v1", Operation: verifierauthority.OperationVerify},
			{VerifierID: "v2", Operation: verifierauthority.OperationVerify},
		}, nil,
		func(authorized []VerifierMetadata) ([]FactoryRunner, error) {
			return []FactoryRunner{
				{VerifierID: authorized[0].Name, Run: verifiers[0].Run},
				{VerifierID: authorized[1].Name, Run: verifiers[1].Run},
			}, nil
		})
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}

	records, err := binding.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("records count = %d, want 2", len(records))
	}

	_, err = binding.Execute()
	var consumedErr *ErrProfileBindingConsumed
	if !errors.As(err, &consumedErr) {
		t.Errorf("expected ErrProfileBindingConsumed, got %T: %v", err, err)
	}
}

// TestDeniedBindingCannotExecute verifies denied bindings cannot execute.
func TestDeniedBindingCannotExecute(t *testing.T) {
	profile := &AuthorizedProfile{}
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
		{Metadata: VerifierMetadata{Name: "v1"}, Run: func(root string) []checks.Finding { return nil }},
	}
	profile := &AuthorizedProfile{}
	binding := &ProfileBinding{profile: profile, runners: runners}

	meta := binding.Runners()
	if len(meta) != 1 {
		t.Fatalf("expected 1 metadata, got %d", len(meta))
	}
	if meta[0].Name != "v1" {
		t.Errorf("expected verifier name v1, got %s", meta[0].Name)
	}
}

// TestFactoryReceivesNonExecutableMetadata verifies factory receives VerifierMetadata.
func TestFactoryReceivesNonExecutableMetadata(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	var received []VerifierMetadata
	factory := func(authorized []VerifierMetadata) ([]FactoryRunner, error) {
		received = authorized
		return []FactoryRunner{{VerifierID: authorized[0].Name, Run: func(root string) []checks.Finding { return nil }}}, nil
	}

	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 metadata, got %d", len(received))
	}
	if received[0].Name != "v1" {
		t.Errorf("expected name v1, got %s", received[0].Name)
	}

	records, err := binding.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

// TestFactoryCannotForgeMetadata verifies factory cannot forge metadata (only provides ID + Run).
func TestFactoryCannotForgeMetadata(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	factory := func(authorized []VerifierMetadata) ([]FactoryRunner, error) {
		return []FactoryRunner{{VerifierID: authorized[0].Name, Run: func(root string) []checks.Finding { return nil }}}, nil
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

	if records[0].Metadata.Authority != verifierauthority.AuthorityLocalSafe {
		t.Errorf("expected AuthorityLocalSafe, got %v", records[0].Metadata.Authority)
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

	factory := func(authorized []VerifierMetadata) ([]FactoryRunner, error) {
		reversed := make([]FactoryRunner, len(authorized))
		for i, v := range authorized {
			reversed[len(authorized)-1-i] = FactoryRunner{VerifierID: v.Name, Run: verifiers[len(authorized)-1-i].Run}
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
		if records[i].Metadata.Name != wantName {
			t.Errorf("order[%d] = %q, want %q", i, records[i].Metadata.Name, wantName)
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
		func(authorized []VerifierMetadata) ([]FactoryRunner, error) {
			return []FactoryRunner{{VerifierID: authorized[0].Name, Run: func(root string) []checks.Finding { return nil }}}, nil
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

// TestFactoryContractRejectsEmptyVerifierID verifies factory contract rejects empty verifier ID.
func TestFactoryContractRejectsEmptyVerifierID(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	factory := func(authorized []VerifierMetadata) ([]FactoryRunner, error) {
		return []FactoryRunner{{VerifierID: "", Run: func(root string) []checks.Finding { return nil }}}, nil
	}

	_, err = d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestFactoryContractRejectsNilRunFunction verifies factory contract rejects nil Run.
func TestFactoryContractRejectsNilRunFunction(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	factory := func(authorized []VerifierMetadata) ([]FactoryRunner, error) {
		return []FactoryRunner{{VerifierID: "v1", Run: nil}}, nil
	}

	_, err = d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestFactoryContractRejectsDuplicateVerifierID verifies factory contract rejects duplicate IDs.
func TestFactoryContractRejectsDuplicateVerifierID(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
		{Name: "v2", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	factory := func(authorized []VerifierMetadata) ([]FactoryRunner, error) {
		return []FactoryRunner{
			{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }},
			{VerifierID: "v1", Run: func(root string) []checks.Finding { return nil }},
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

// TestFactoryContractRejectsUnknownVerifierID verifies factory contract rejects unknown IDs.
func TestFactoryContractRejectsUnknownVerifierID(t *testing.T) {
	verifiers := []registry.Verifier{
		{Name: "v1", Authority: verifierauthority.AuthorityLocalSafe, Lane: registry.VerifierLaneFast, Run: func(root string) []checks.Finding { return nil }},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	factory := func(authorized []VerifierMetadata) ([]FactoryRunner, error) {
		return []FactoryRunner{{VerifierID: "unknown", Run: func(root string) []checks.Finding { return nil }}}, nil
	}

	_, err = d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil, factory)
	var contractErr *ErrProfileFactoryContract
	if !errors.As(err, &contractErr) {
		t.Errorf("expected ErrProfileFactoryContract, got %T: %v", err, err)
	}
}

// TestExecutionRecordHasCompleteMetadata verifies ExecutionRecord has complete metadata.
func TestExecutionRecordHasCompleteMetadata(t *testing.T) {
	verifiers := []registry.Verifier{
		{
			Name:      "v1",
			Authority: verifierauthority.AuthorityLocalSafe,
			Lane:      registry.VerifierLaneFast,
			Execution: registry.ExecutionDefinition{
				Kind:             registry.ExecutionInProcess,
				ImplementationID: "test-impl",
				EnvVars:          []string{"TEST=1"},
			},
			Run: func(root string) []checks.Finding { return nil },
		},
	}
	d, err := NewDispatcher(verifiers)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	binding, err := d.AuthorizeAndBindProfile(context.Background(), "/test",
		[]ProfileRequest{{VerifierID: "v1", Operation: verifierauthority.OperationVerify}}, nil,
		func(authorized []VerifierMetadata) ([]FactoryRunner, error) {
			return []FactoryRunner{{VerifierID: authorized[0].Name, Run: verifiers[0].Run}}, nil
		})
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}

	records, err := binding.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if records[0].Metadata.Name != "v1" {
		t.Errorf("expected Name=v1, got %s", records[0].Metadata.Name)
	}
	if records[0].Metadata.Kind != registry.ExecutionInProcess {
		t.Errorf("expected Kind=ExecutionInProcess, got %s", records[0].Metadata.Kind)
	}
	if records[0].Metadata.ImplID != "test-impl" {
		t.Errorf("expected ImplID=test-impl, got %s", records[0].Metadata.ImplID)
	}
	if len(records[0].Metadata.EnvVars) != 1 || records[0].Metadata.EnvVars[0] != "TEST=1" {
		t.Errorf("expected EnvVars=[TEST=1], got %v", records[0].Metadata.EnvVars)
	}
}
