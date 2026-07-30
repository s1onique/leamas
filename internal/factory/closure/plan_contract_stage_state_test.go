package closure

import "testing"

// TestStageStateSuccess pins the stage booleans and diagnostic
// arrays for the success path. Every reachable field has an
// expected value; any deviation fails the test.
func TestStageStateSuccess(t *testing.T) {
	composed := validatePlanComposedWithObserver(
		composedCanonicalPlanIndented(), noopCompositionObserver{})
	if !composed.Structural.Valid {
		t.Fatalf("Structural.Valid = false; errors = %v",
			composed.Structural.Errors)
	}
	if !composed.Decoded {
		t.Fatalf("Decoded = false")
	}
	if !composed.SemanticValid {
		t.Fatalf("SemanticValid = false")
	}
	if !composed.Valid {
		t.Fatalf("Valid = false")
	}
	if composed.Structural.Errors == nil {
		t.Fatalf("Structural.Errors = nil")
	}
	if len(composed.Structural.Errors) != 0 {
		t.Fatalf("Structural.Errors = %v, want []",
			composed.Structural.Errors)
	}
	if composed.DecodeErrors == nil {
		t.Fatalf("DecodeErrors = nil")
	}
	if len(composed.DecodeErrors) != 0 {
		t.Fatalf("DecodeErrors = %v, want []", composed.DecodeErrors)
	}
	if composed.SemanticErrors == nil {
		t.Fatalf("SemanticErrors = nil")
	}
	if len(composed.SemanticErrors) != 0 {
		t.Fatalf("SemanticErrors = %v, want []",
			composed.SemanticErrors)
	}
}

// TestStageStateParseFailure pins the stage booleans and
// diagnostic arrays for the parse-failure path. Structural
// diagnostics surface; decode and semantic stages do not run.
func TestStageStateParseFailure(t *testing.T) {
	composed := validatePlanComposedWithObserver(
		[]byte(`not valid json`), noopCompositionObserver{})
	if composed.Structural.Valid {
		t.Fatalf("Structural.Valid = true")
	}
	if composed.Decoded {
		t.Fatalf("Decoded = true")
	}
	if composed.SemanticValid {
		t.Fatalf("SemanticValid = true")
	}
	if composed.Valid {
		t.Fatalf("Valid = true")
	}
	if len(composed.Structural.Errors) == 0 {
		t.Fatalf("Structural.Errors empty")
	}
	if composed.DecodeErrors == nil {
		t.Fatalf("DecodeErrors = nil")
	}
	if len(composed.DecodeErrors) != 0 {
		t.Fatalf("DecodeErrors = %v, want []", composed.DecodeErrors)
	}
	if composed.SemanticErrors == nil {
		t.Fatalf("SemanticErrors = nil")
	}
	if len(composed.SemanticErrors) != 0 {
		t.Fatalf("SemanticErrors = %v, want []",
			composed.SemanticErrors)
	}
}

// TestStageStateStructuralSchemaFailure pins the stage booleans
// and diagnostic arrays for a structural-schema failure (a valid
// JSON document whose check.mode is outside the closed enum).
// The failure surfaces at the structural stage; decode and
// semantic stages do not run.
func TestStageStateStructuralSchemaFailure(t *testing.T) {
	composed := validatePlanComposedWithObserver(
		composedPlanWithInvalidEnum(), noopCompositionObserver{})
	if composed.Structural.Valid {
		t.Fatalf("Structural.Valid = true")
	}
	if composed.Decoded {
		t.Fatalf("Decoded = true")
	}
	if composed.SemanticValid {
		t.Fatalf("SemanticValid = true")
	}
	if composed.Valid {
		t.Fatalf("Valid = true")
	}
	if len(composed.Structural.Errors) == 0 {
		t.Fatalf("Structural.Errors empty")
	}
	if composed.DecodeErrors == nil {
		t.Fatalf("DecodeErrors = nil")
	}
	if len(composed.DecodeErrors) != 0 {
		t.Fatalf("DecodeErrors = %v, want []", composed.DecodeErrors)
	}
	if composed.SemanticErrors == nil {
		t.Fatalf("SemanticErrors = nil")
	}
	if len(composed.SemanticErrors) != 0 {
		t.Fatalf("SemanticErrors = %v, want []",
			composed.SemanticErrors)
	}
}

// TestStageStateSemanticFailure pins the stage booleans and
// diagnostic arrays for the semantic-failure path. Structural
// and decode stages complete successfully; only the semantic
// stage fails.
func TestStageStateSemanticFailure(t *testing.T) {
	composed := validatePlanComposedWithObserver(
		composedPlanWithDuplicateCheckID(), noopCompositionObserver{})
	if !composed.Structural.Valid {
		t.Fatalf("Structural.Valid = false; errors = %v",
			composed.Structural.Errors)
	}
	if !composed.Decoded {
		t.Fatalf("Decoded = false")
	}
	if composed.SemanticValid {
		t.Fatalf("SemanticValid = true")
	}
	if composed.Valid {
		t.Fatalf("Valid = true")
	}
	if len(composed.Structural.Errors) != 0 {
		t.Fatalf("Structural.Errors = %v, want []",
			composed.Structural.Errors)
	}
	if composed.DecodeErrors == nil {
		t.Fatalf("DecodeErrors = nil")
	}
	if len(composed.DecodeErrors) != 0 {
		t.Fatalf("DecodeErrors = %v, want []", composed.DecodeErrors)
	}
	if composed.SemanticErrors == nil {
		t.Fatalf("SemanticErrors = nil")
	}
	if len(composed.SemanticErrors) == 0 {
		t.Fatalf("SemanticErrors empty")
	}
}
