package closure

import "errors"

// plan_semantic_composed.go contains the semantic diagnostic extraction
// helpers the composed pipeline uses. Every semantic error reachable
// from ValidatePlan returns a typed diagnostic source. No error-message
// parsing occurs in this file.

// semanticDiagnostics maps a semantic Go error to a structured diagnostic
// stream. The function is the single entry point the composed pipeline
// uses to extract diagnostics from semantic validation errors.
//
// Behavior:
//  1. nil error → non-nil empty slice.
//  2. errors.As(err, planDiagnosticSource) → defensive copy of typed diagnostics.
//  3. Wrapped typed error → typed diagnostics preserved.
//  4. Unknown untyped error → one deterministic root fallback.
func semanticDiagnostics(err error) []PlanValidationError {
	if err == nil {
		return []PlanValidationError{}
	}

	// Check for planDiagnosticSource (typed semantic error).
	var source planDiagnosticSource
	if errors.As(err, &source) {
		return clonePlanValidationErrors(source.PlanDiagnostics())
	}

	// Unknown untyped error → deterministic root fallback.
	return []PlanValidationError{{
		InstancePath: "",
		SchemaPath:   "",
		Code:         PlanCodeSemanticConstraintFailed,
		Keyword:      KeywordType,
		Message:      err.Error(),
	}}
}
