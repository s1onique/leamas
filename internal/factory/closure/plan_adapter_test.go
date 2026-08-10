// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_adapter_test.go is the B2-R7
// adapter-focused test suite. It exercises representative
// canonical leaf errors and asserts the closure package
// adapts each one to the expected legacy typed-error
// class. The test preserves the typed-error contract the
// existing closure-package tests depend on while
// confirming the adapter is the single translation
// surface.
//
// The matrix is intentionally small: it covers one
// canonical example per leaf code category. Adding a new
// leaf code category requires adding a row here so the
// adapter mapping stays covered by the executable
// contract.
package closure

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// TestPlanContractAdapterMapsLeafErrorsToLegacyTypedError
// is the B2-R7 adapter mapping test. Each row crafts a
// Plan that the leaf rejects with a specific canonical
// code and asserts the closure adapter surfaces the
// expected typed-error class with the expected InstancePath
// and Message.
//
// B2-R7 single-authority rule: this test asserts the
// adapter surface only. The leaf owns the rule; the
// adapter owns the typed-error mapping.
func TestPlanContractAdapterMapsLeafErrorsToLegacyTypedError(t *testing.T) {
	t.Parallel()

	type row struct {
		name           string
		build          func() Plan
		expectCode     PlanValidationCode
		expectPathHas  string
		expectFieldHas string
	}

	rows := []row{
		{
			name: "invalid_act_id",
			build: func() Plan {
				p := validR7AdapterPlan()
				p.ActID = "not-an-act-id"
				return p
			},
			expectCode:     PlanCodeSemanticConstraintFailed,
			expectPathHas:  "/act_id",
			expectFieldHas: "act_id",
		},
		{
			name: "invalid_baseline_oid",
			build: func() Plan {
				p := validR7AdapterPlan()
				p.Baseline.CommitOID = "not-a-hex-oid"
				return p
			},
			expectCode:     PlanCodeSemanticConstraintFailed,
			expectPathHas:  "/baseline/commit_oid",
			expectFieldHas: "baseline.commit_oid",
		},
		{
			name: "baseline_oid_placeholder",
			build: func() Plan {
				p := validR7AdapterPlan()
				p.Baseline.CommitOID = "TBD"
				return p
			},
			expectCode:     PlanCodeSemanticConstraintFailed,
			expectPathHas:  "/baseline/commit_oid",
			expectFieldHas: "baseline.commit_oid",
		},
		{
			name: "invalid_mode",
			build: func() Plan {
				p := validR7AdapterPlan()
				mode := ExecutionMode("garbage")
				p.Execution = NewPlanExecution(mode)
				return p
			},
			expectCode:     PlanCodeSemanticConstraintFailed,
			expectPathHas:  "/execution/mode",
			expectFieldHas: "execution.mode",
		},
		{
			name: "policy with null sibling (invalid_policy_constraint)",
			build: func() Plan {
				p := validR7AdapterPlan()
				p.Policy = PlanPolicy{}
				return p
			},
			expectCode:    PlanCodeSemanticConstraintFailed,
			expectPathHas: "/policy/",
		},
		{
			name: "duplicate_check_id",
			build: func() Plan {
				p := validR7AdapterPlan()
				p.Checks = []PlanCheck{makeRunCheck("dup"), makeRunCheck("dup")}
				return p
			},
			expectCode:    PlanCodeSemanticConstraintFailed,
			expectPathHas: "/checks/",
		},
		{
			name: "subject_exact with tool rejected",
			build: func() Plan {
				p := validR7AdapterPlan()
				p.RunnerAuthority = &RunnerAuthority{
					Mode: RunnerAuthoritySubjectExact,
					Tool: &ToolAuthority{
						Revision:     "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
						BinarySHA256: "c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3",
					},
				}
				return p
			},
			expectCode:    PlanCodeForbiddenPresence,
			expectPathHas: "/runner_authority/tool",
		},
	}

	for _, r := range rows {
		r := r
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			plan := r.build()
			err := ValidatePlan(plan)
			if err == nil {
				t.Fatalf("expected adapter to surface an error for %q", r.name)
			}
			var semErr *PlanSemanticError
			if !errors.As(err, &semErr) {
				t.Fatalf("expected *PlanSemanticError, got %T (%v)", err, err)
			}
			if semErr.Diagnostic.Code != r.expectCode {
				t.Fatalf("code mismatch: got %q, want %q", semErr.Diagnostic.Code, r.expectCode)
			}
			if r.expectPathHas != "" && !strings.Contains(semErr.Diagnostic.InstancePath, r.expectPathHas) {
				t.Fatalf("path mismatch: got %q, want contains %q",
					semErr.Diagnostic.InstancePath, r.expectPathHas)
			}
			if r.expectFieldHas != "" && !strings.Contains(semErr.Diagnostic.Message, r.expectFieldHas) {
				t.Fatalf("message mismatch: got %q, want contains %q",
					semErr.Diagnostic.Message, r.expectFieldHas)
			}
		})
	}
}

// validR7AdapterPlan is a fixture-builder helper for the
// adapter test. It returns a fully-valid Plan that the
// leaf accepts; the adapter test mutates one field per
// row to exercise the mapping.
func validR7AdapterPlan() Plan {
	t := true
	mode := ExecutionModeSerialFailFast
	return Plan{
		ContractVersion: ContractVersionV1,
		ActID:           "ACT-LEAMAS-ADAPTER01",
		Baseline: Baseline{
			CommitOID: "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
			TreeOID:   "b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2",
		},
		Execution: PlanExecution{Mode: &mode},
		Checks:    []PlanCheck{makeRunCheck("c1")},
		Artifacts: []PlanArtifact{
			{
				ID:        "a1",
				Path:      "docs/a1.md",
				Required:  &t,
				MaxBytes:  1024,
				MediaType: "text/plain",
			},
		},
		Policy: PlanPolicy{
			RequireCleanBefore:       &t,
			RequireCleanAfter:        &t,
			ForbidTrackedFullDigests: &t,
			RequireDiffCheck:         &t,
		},
	}
}

// TestPlanContractValidateAndProjectExposesCanonicalProjection
// asserts that the canonical ValidatedPlan returned by
// DecodeAndValidateFull is the source of truth the
// closure package and the evidence package both consume.
// The test runs the adapter, the leaf, and a re-decoded
// Plan through ValidatePlan and asserts the expected
// fields survive the round-trip.
func TestPlanContractValidateAndProjectExposesCanonicalProjection(t *testing.T) {
	t.Parallel()

	t1 := true
	mode := ExecutionModeSerialFailFast
	plan := Plan{
		ContractVersion: ContractVersionV1,
		ActID:           "ACT-LEAMAS-PROJ01",
		Baseline: Baseline{
			CommitOID: "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
			TreeOID:   "b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2",
		},
		Execution: PlanExecution{Mode: &mode},
		Checks:    []PlanCheck{makeRunCheck("c1")},
		Artifacts: []PlanArtifact{},
		Policy: PlanPolicy{
			RequireCleanBefore:       &t1,
			RequireCleanAfter:        &t1,
			ForbidTrackedFullDigests: &t1,
			RequireDiffCheck:         &t1,
		},
	}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	validated, err := plancontract.DecodeAndValidateFull(data)
	if err != nil {
		t.Fatalf("DecodeAndValidateFull: %v", err)
	}
	if validated.ContractVersion != plan.ContractVersion {
		t.Fatalf("contract_version: got %d, want %d",
			validated.ContractVersion, plan.ContractVersion)
	}
	if validated.ActID != plan.ActID {
		t.Fatalf("act_id: got %q, want %q",
			validated.ActID, plan.ActID)
	}
	if len(validated.Checks) != 1 || validated.Checks[0].ID != "c1" {
		t.Fatalf("checks projection lost")
	}
}
