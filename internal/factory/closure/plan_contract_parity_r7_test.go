// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_contract_parity_r7_test.go is the
// B2-R7 valid-JSON differential matrix. Every fixture is
// built via the typed FixturePlanBuilder so the matrix
// itself proves that json.Valid(bytes) == true for every
// row before either authority sees it.
//
// B2-R7 closes the last remaining semantic-drift path
// between the closure runner and the evidence package:
// both consume the same canonical ValidatedPlan produced
// by plancontract.DecodeAndValidateFull. The legacy
// validatePlanTyped helper in the closure package is
// deleted; closure.ValidatePlan is now an adapter that
// serialises the typed Plan, calls the canonical leaf,
// and adapts any DecodeError back to the legacy
// typed-error contract.
//
// The matrix below covers every row the B2-R7 spec lists.
// For every row the assertion is identical:
//
//	fixture_json_valid == true
//	execution_accept   == evidence_accept
//
// where execution_accept is closure.ValidatePlan(plan)
// after LoadPlanFromBytes and evidence_accept is
// plancontract.ValidateFull on the same bytes.
package closure

import (
	"encoding/json"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// r7Fixture is the typed fixture builder the matrix uses.
// Every fixture is constructed via append rather than by
// mutating a pre-rendered JSON string so the matrix
// guarantees json.Valid(bytes) == true for every row.
//
// A fixture is a pair of bytes (the raw wire shape) and a
// builder (the typed shape) so the execution path can
// reconstruct the typed Plan from the builder without
// round-tripping through the bytes.
type r7Fixture struct {
	name    string
	bytes   []byte
	plan    Plan
	wantErr bool
}

// r7Builder accumulates typed Plan Contract fields and
// produces the wire bytes via the canonical
// json.Marshal(Plan). The builder only exposes methods for
// fields the spec actually exercises so the matrix stays
// LLM-friendly.
type r7Builder struct {
	plan Plan
}

func newR7Builder() *r7Builder {
	t := true
	return &r7Builder{
		plan: Plan{
			ContractVersion: ContractVersionV1,
			ActID:           "ACT-R7-PARITY-01",
			Baseline: Baseline{
				CommitOID: "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
				TreeOID:   "b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2",
			},
			Execution: NewPlanExecution(ExecutionModeSerialFailFast),
			Checks: []PlanCheck{
				makeRunCheck("c1"),
			},
			Artifacts: []PlanArtifact{
				makeArtifact("a1"),
			},
			Policy: PlanPolicy{
				RequireCleanBefore:       &t,
				RequireCleanAfter:        &t,
				ForbidTrackedFullDigests: &t,
				RequireDiffCheck:         &t,
			},
		},
	}
}

func makeRunCheck(id string) PlanCheck {
	return PlanCheck{
		ID:               id,
		Mode:             CheckModeRun,
		Argv:             []string{"go", "test"},
		WorkingDirectory: ".",
		TimeoutSeconds:   60,
		Environment:      map[string]string{"K": "V"},
	}
}

// r7MinRunCheck is the minimal run-mode check shape used
// only by the MaxChecks boundary fixture so the resulting
// JSON stays under MaxPlanBytes (1 MiB). The shape is
// still a valid Plan Contract v1 run check (argv is
// required and non-empty; timeout_seconds is in range).
func r7MinRunCheck(id string) PlanCheck {
	return PlanCheck{
		ID:               id,
		Mode:             CheckModeRun,
		Argv:             []string{"x"},
		WorkingDirectory: ".",
		TimeoutSeconds:   1,
		Environment:      map[string]string{},
	}
}

// r7MinArtifact is the minimal artifact shape used only
// by the MaxArtifacts boundary fixture so the resulting
// JSON stays under MaxPlanBytes (1 MiB).
func r7MinArtifact(id string) PlanArtifact {
	t := true
	return PlanArtifact{
		ID:        id,
		Path:      "d",
		Required:  &t,
		MaxBytes:  1,
		MediaType: "x",
	}
}

func makeExcludeCheck(id, reason string) PlanCheck {
	return PlanCheck{
		ID:     id,
		Mode:   CheckModeExclude,
		Reason: reason,
	}
}

func makeArtifact(id string) PlanArtifact {
	t := true
	return PlanArtifact{
		ID:        id,
		Path:      "docs/" + id + ".md",
		Required:  &t,
		MaxBytes:  1024,
		MediaType: "text/plain",
	}
}

func (b *r7Builder) bytes(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(b.plan)
	if err != nil {
		t.Fatalf("fixture marshal: %v", err)
	}
	return data
}

// r7ValidJSON asserts every fixture's bytes satisfy
// json.Valid. The matrix uses this guard before calling
// either authority so the precondition is enforceable
// from the test itself, not from the test author.
func r7ValidJSON(t *testing.T, rows []r7Fixture) {
	t.Helper()
	for _, r := range rows {
		if !json.Valid(r.bytes) {
			t.Fatalf("fixture %q is NOT valid JSON; the B2-R7 spec requires json.Valid == true for every row", r.name)
		}
	}
}

// r7ExecutionAccept runs the closure-runnable path on the
// fixture's bytes and returns whether the closure runner
// accepts the plan. Acceptance here means
// LoadPlanFromBytes + ValidatePlan return nil.
func r7ExecutionAccept(t *testing.T, f r7Fixture) bool {
	t.Helper()
	plan, _, err := LoadPlanFromBytes(f.bytes)
	if err != nil {
		return false
	}
	return ValidatePlan(plan) == nil
}

// r7EvidenceAccept runs the evidence path on the fixture's
// bytes and returns whether the evidence package accepts
// the plan. Acceptance means plancontract.ValidateFull
// returns nil.
func r7EvidenceAccept(f r7Fixture) bool {
	return plancontract.ValidateFull(f.bytes) == nil
}

// TestPlanContractExecutionEvidenceParityR7 is the
// umbrella acceptance matrix. Every row is a typed
// fixture; json.Valid == true is asserted once up-front
// via r7ValidJSON; per-row, the assertion is identical
// execution_accept == evidence_accept.
//
// The matrix covers every row the B2-R7 spec lists, in
// spec order.
func TestPlanContractExecutionEvidenceParityR7(t *testing.T) {
	t.Parallel()

	rows := r7ParityRows(t)

	// Pre-condition: every fixture must be valid JSON
	// before either authority sees it.
	r7ValidJSON(t, rows)

	for _, r := range rows {
		r := r
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			execAccept := r7ExecutionAccept(t, r)
			evAccept := r7EvidenceAccept(r)
			if execAccept != evAccept {
				t.Fatalf("execution/evidence parity broken for %q:\n  execution accept=%v\n  evidence  accept=%v",
					r.name, execAccept, evAccept)
			}
			if r.wantErr && execAccept {
				t.Fatalf("both authorities accepted %q but the matrix expected rejection", r.name)
			}
			if !r.wantErr && !execAccept {
				t.Fatalf("both authorities rejected %q but the matrix expected acceptance", r.name)
			}
		})
	}
}

// r7ParityRows constructs every fixture the B2-R7 spec
// requires. Each fixture is built via the typed builder
// so json.Valid == true holds by construction.
//
// The list mirrors the spec order so a reviewer can match
// every fixture back to its spec row one-for-one.
