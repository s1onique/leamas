package closure

import "testing"

// TestWrapperPrecedenceInvalidJSON proves the wrapper surfaces
// an invalid_json structural diagnostic when given malformed
// JSON. The composed pipeline never reaches the typed decoder
// or the semantic validator.
func TestWrapperPrecedenceInvalidJSON(t *testing.T) {
	result, err := ValidatePlanStructuralAndSemantic([]byte(`not valid json`))
	if err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
	if len(result.Errors) == 0 {
		t.Fatalf("structural errors empty")
	}
	if result.Errors[0].Code != PlanCodeInvalidJSON {
		t.Fatalf("structural code = %v, want %v",
			result.Errors[0].Code, PlanCodeInvalidJSON)
	}
}

// TestWrapperPrecedenceDuplicateProperty proves the wrapper
// surfaces a duplicate_property structural diagnostic when the
// bounded parser detects a duplicate root key. The composed
// pipeline short-circuits before the structural walker.
func TestWrapperPrecedenceDuplicateProperty(t *testing.T) {
	result, err := ValidatePlanStructuralAndSemantic(
		composedPlanWithDuplicateRootKey())
	if err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
	if len(result.Errors) == 0 {
		t.Fatalf("structural errors empty")
	}
	if result.Errors[0].Code != PlanCodeDuplicateProperty {
		t.Fatalf("structural code = %v, want %v",
			result.Errors[0].Code, PlanCodeDuplicateProperty)
	}
}

// TestWrapperPrecedenceInvalidEnum proves the wrapper surfaces an
// invalid_enum structural diagnostic when the document parses
// but fails the closed-enum schema check at /checks/*/mode.
func TestWrapperPrecedenceInvalidEnum(t *testing.T) {
	result, err := ValidatePlanStructuralAndSemantic(
		composedPlanWithInvalidEnum())
	if err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
	if len(result.Errors) == 0 {
		t.Fatalf("structural errors empty")
	}
	if result.Errors[0].Code != PlanCodeInvalidEnum {
		t.Fatalf("structural code = %v, want %v",
			result.Errors[0].Code, PlanCodeInvalidEnum)
	}
}

// TestWrapperPrecedenceDuplicateCheckID proves the wrapper
// surfaces a semantic_constraint_failed semantic diagnostic when
// the structural stage passes but the semantic stage rejects the
// document. The wrapper precedence rule keeps the typed-decode
// stage silent when the structural stage succeeded.
func TestWrapperPrecedenceDuplicateCheckID(t *testing.T) {
	_, err := ValidatePlanStructuralAndSemantic(
		composedPlanWithDuplicateCheckID())
	if err == nil {
		t.Fatalf("err = nil, want non-nil")
	}
	composed := validatePlanComposedWithObserver(
		composedPlanWithDuplicateCheckID(), noopCompositionObserver{})
	if len(composed.SemanticErrors) == 0 {
		t.Fatalf("semantic errors empty")
	}
	if composed.SemanticErrors[0].Code != PlanCodeSemanticConstraintFailed {
		t.Fatalf("semantic code = %v, want %v",
			composed.SemanticErrors[0].Code,
			PlanCodeSemanticConstraintFailed)
	}
}

// TestWrapperPrecedenceValidPlan proves the wrapper returns a
// nil error and a valid structural result on the canonical
// success path. ContractVersion is recovered from the parsed
// root so the future CLI can render it without re-parsing.
func TestWrapperPrecedenceValidPlan(t *testing.T) {
	result, err := ValidatePlanStructuralAndSemantic(
		composedCanonicalPlanIndented())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if !result.Valid {
		t.Fatalf("Structural.Valid = false; errors = %v",
			result.Errors)
	}
	if result.ContractVersion != 1 {
		t.Fatalf("ContractVersion = %d, want 1",
			result.ContractVersion)
	}
}
