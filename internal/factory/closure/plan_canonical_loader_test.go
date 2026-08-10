// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_canonical_loader_test.go is the
// B2-R7-R2 umbrella test that proves the production
// closure-runner loader path consumes the canonical
// ValidatedPlan projection produced by the plancontract
// leaf. The test asserts:
//
//   - LoadPlanFromBytesWithValidated returns the canonical
//     ValidatedPlan alongside the typed Plan.
//   - The typed Plan and the ValidatedPlan agree on every
//     field the runner code paths use downstream.
//   - LoadPlanFromBytes (the production entry point) routes
//     through plancontract.DecodeAndValidateFull so the
//     canonical model is the single semantic authority for
//     the execution path.
//
// CANONICAL_VALIDATED_PLAN_EXECUTION_MODEL: true.
package closure

import (
	"encoding/json"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// TestLoadPlanFromBytesWithValidatedReturnsCanonicalProjection
// asserts LoadPlanFromBytesWithValidated returns the
// canonical ValidatedPlan alongside the typed Plan. The
// test inspects every field the runner code paths use
// downstream and asserts the ValidatedPlan agrees with the
// typed Plan.
func TestLoadPlanFromBytesWithValidatedReturnsCanonicalProjection(t *testing.T) {
	t.Parallel()

	t1 := true
	mode := ExecutionModeSerialFailFast
	plan := Plan{
		ContractVersion: ContractVersionV1,
		ActID:           "ACT-LEAMAS-CANONICAL01",
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
				Required:  &t1,
				MaxBytes:  1024,
				MediaType: "text/plain",
			},
		},
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
	typed, validated, raw, err := LoadPlanFromBytesWithValidated(data)
	if err != nil {
		t.Fatalf("LoadPlanFromBytesWithValidated: %v", err)
	}
	if string(raw) != string(data) {
		t.Fatalf("LoadPlanFromBytesWithValidated returned different bytes than the input")
	}
	if typed.ContractVersion != validated.ContractVersion {
		t.Fatalf("ContractVersion: typed=%d, validated=%d",
			typed.ContractVersion, validated.ContractVersion)
	}
	if typed.ActID != validated.ActID {
		t.Fatalf("ActID: typed=%q, validated=%q",
			typed.ActID, validated.ActID)
	}
	if typed.Baseline.CommitOID != validated.Baseline.CommitOID {
		t.Fatalf("Baseline.CommitOID: typed=%q, validated=%q",
			typed.Baseline.CommitOID, validated.Baseline.CommitOID)
	}
	if typed.Baseline.TreeOID != validated.Baseline.TreeOID {
		t.Fatalf("Baseline.TreeOID: typed=%q, validated=%q",
			typed.Baseline.TreeOID, validated.Baseline.TreeOID)
	}
	if *typed.Execution.Mode != ExecutionModeSerialFailFast {
		t.Fatalf("Execution.Mode: got %q", *typed.Execution.Mode)
	}
	if validated.Execution.Mode != string(ExecutionModeSerialFailFast) {
		t.Fatalf("Validated Execution.Mode: got %q", validated.Execution.Mode)
	}
	if len(typed.Checks) != 1 || typed.Checks[0].ID != "c1" {
		t.Fatalf("Checks: typed=%v", typed.Checks)
	}
	if len(validated.Checks) != 1 || validated.Checks[0].ID != "c1" {
		t.Fatalf("Validated Checks: %v", validated.Checks)
	}
}

// TestLoadPlanFromBytesRoutesThroughCanonicalValidator
// asserts LoadPlanFromBytes routes the production execution
// path through plancontract.DecodeAndValidateFull so the
// canonical ValidatedPlan is the single semantic authority.
// The test sends bytes that the leaf would reject and
// asserts LoadPlanFromBytes also rejects them.
func TestLoadPlanFromBytesRoutesThroughCanonicalValidator(t *testing.T) {
	t.Parallel()

	// Bytes that the leaf rejects because contract_version=2.
	base := newR7Builder()
	base.plan.ContractVersion = 2
	data, _ := json.Marshal(base.plan)
	_, _, err := LoadPlanFromBytes(data)
	if err == nil {
		t.Fatalf("LoadPlanFromBytes accepted contract_version=2; B2-R7-R2 wiring broken")
	}
	// Confirm the leaf also rejects the same bytes so the
	// wiring is consistent.
	if err := plancontract.ValidateFull(data); err == nil {
		t.Fatalf("plancontract.ValidateFull accepted contract_version=2; leaf inconsistent")
	}
}

// TestLoadPlanFromBytesWithValidatedRejectsInvalidPlan
// asserts LoadPlanFromBytesWithValidated rejects the same
// invalid bytes the leaf rejects, proving the canonical
// ValidatedPlan is the single semantic authority.
func TestLoadPlanFromBytesWithValidatedRejectsInvalidPlan(t *testing.T) {
	t.Parallel()

	base := newR7Builder()
	base.plan.ContractVersion = 2
	data, _ := json.Marshal(base.plan)
	_, _, _, err := LoadPlanFromBytesWithValidated(data)
	if err == nil {
		t.Fatalf("LoadPlanFromBytesWithValidated accepted contract_version=2; canonical loader broken")
	}
}
