package closure

import (
	"encoding/json"
	"sort"
	"testing"
)

// plan_contract_typed_stage_test.go exercises the typed-decode
// stage through the composedValidationDeps seam. Each test name
// matches a documented regression-gap prefix so the focused-test
// regex TypedStage|TypedFailure|TypedStageJSON selects them.

// sentinelTypedDecodeError is the stable, hand-written error the
// sentinel decoder returns. The Error() message is short and
// unique so the wrapped diagnostic can be asserted deterministically.
type sentinelTypedDecodeError struct{}

func (sentinelTypedDecodeError) Error() string {
	return "sentinel typed decode failure"
}

// sentinelTypedDecode is the typed-decode binding tests inject
// through composedValidationDeps. It records exactly one
// TypedDecoded event so the observer proves the typed stage was
// reached. It never invokes SemanticValidated so tests prove the
// semantic stage is short-circuited.
func sentinelTypedDecode(root any, observer compositionObserver) (Plan, error) {
	observer.TypedDecoded()
	return Plan{}, sentinelTypedDecodeError{}
}

func TestTypedStageWrapperReturnsDecodeError(t *testing.T) {
	deps := composedValidationDeps{DecodeTyped: sentinelTypedDecode}
	obs := &countingObserver{}
	composed := validatePlanComposedWithObserverAndDeps(canonicalComposedPlan(), obs, deps)
	if !composed.Structural.Valid {
		t.Fatalf("structural must succeed before typed runs: %+v", composed.Structural.Errors)
	}
	if composed.Decoded {
		t.Fatalf("Decoded must be false on typed decode failure")
	}
	if obs.parsedCount != 1 {
		t.Fatalf("parsed count = %d, want exactly 1", obs.parsedCount)
	}
	if obs.typedDecodedCount != 1 {
		t.Fatalf("typed decode count = %d, want exactly 1", obs.typedDecodedCount)
	}
	if obs.semanticValidatedCount != 0 {
		t.Fatalf("semantic must NOT be invoked on typed failure: got %d", obs.semanticValidatedCount)
	}
	if composed.SemanticValid {
		t.Fatalf("SemanticValid must be false on typed failure")
	}
	if composed.Valid {
		t.Fatalf("Valid must be false on typed failure")
	}
}

func TestTypedFailureFullTruthTable(t *testing.T) {
	deps := composedValidationDeps{DecodeTyped: sentinelTypedDecode}
	obs := &countingObserver{}
	composed := validatePlanComposedWithObserverAndDeps(canonicalComposedPlan(), obs, deps)

	if !composed.Structural.Valid {
		t.Fatalf("Structural.Valid must be true, errors=%v", composed.Structural.Errors)
	}
	if composed.Decoded {
		t.Fatalf("Decoded must be false")
	}
	if len(composed.DecodeErrors) != 1 {
		t.Fatalf("DecodeErrors length = %d, want exactly 1", len(composed.DecodeErrors))
	}
	if composed.SemanticValid {
		t.Fatalf("SemanticValid must be false")
	}
	if len(composed.SemanticErrors) != 0 {
		t.Fatalf("SemanticErrors length = %d, want 0", len(composed.SemanticErrors))
	}
	if composed.Valid {
		t.Fatalf("Valid must be false")
	}
	if obs.parsedCount != 1 {
		t.Fatalf("parser calls = %d, want 1", obs.parsedCount)
	}
	if obs.typedDecodedCount != 1 {
		t.Fatalf("typed calls = %d, want 1", obs.typedDecodedCount)
	}
	if obs.semanticValidatedCount != 0 {
		t.Fatalf("semantic calls = %d, want 0", obs.semanticValidatedCount)
	}
	decode := composed.DecodeErrors[0]
	if decode.Code != PlanCodeInvalidType {
		t.Fatalf("decode code = %q, want %q", decode.Code, PlanCodeInvalidType)
	}
	if decode.Keyword != KeywordType {
		t.Fatalf("decode keyword = %q, want %q", decode.Keyword, KeywordType)
	}
	if decode.InstancePath != "" {
		t.Fatalf("decode path = %q, want empty", decode.InstancePath)
	}
}

func TestTypedStageJSONShapeKeysAndArrays(t *testing.T) {
	deps := composedValidationDeps{DecodeTyped: sentinelTypedDecode}
	composed := validatePlanComposedWithObserverAndDeps(canonicalComposedPlan(), noopCompositionObserver{}, deps)
	raw, err := json.Marshal(composed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []string{"structural", "decoded", "decode_errors", "semantic_valid", "semantic_errors", "valid"}
	sort.Strings(want)
	got := make([]string, 0, len(top))
	for key := range top {
		got = append(got, key)
	}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Fatalf("top-level key count = %d, want %d (got=%v)", len(got), len(want), got)
	}
	for i, key := range want {
		if got[i] != key {
			t.Fatalf("top-level keys = %v, want %v", got, want)
		}
	}
	// No diagnostic array may be null. decode_errors has exactly one
	// entry; semantic_errors is empty (not null); structural.errors
	// is empty (not null).
	if string(top["decode_errors"]) == "null" {
		t.Fatalf("decode_errors must not be null")
	}
	if string(top["semantic_errors"]) == "null" {
		t.Fatalf("semantic_errors must not be null")
	}
	var structural PlanValidationResult
	if err := json.Unmarshal(top["structural"], &structural); err != nil {
		t.Fatalf("unmarshal structural: %v", err)
	}
	if structural.Errors == nil {
		t.Fatalf("structural.errors must not be null")
	}
	if len(structural.Errors) != 0 {
		t.Fatalf("structural.errors length = %d, want 0", len(structural.Errors))
	}
	var decodeErrors []PlanValidationError
	if err := json.Unmarshal(top["decode_errors"], &decodeErrors); err != nil {
		t.Fatalf("unmarshal decode_errors: %v", err)
	}
	if len(decodeErrors) != 1 {
		t.Fatalf("decode_errors length = %d, want 1", len(decodeErrors))
	}
}
