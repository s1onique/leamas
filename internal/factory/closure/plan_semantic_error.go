package closure

// plan_semantic_error.go defines the typed semantic error types
// that carry exact JSON Pointer paths for every semantic validation
// failure reachable from ValidatePlan.

// planDiagnosticSource is the interface all typed semantic errors
// implement. The composed pipeline uses errors.As to extract
// structured diagnostics without error-message parsing.
type planDiagnosticSource interface {
	PlanDiagnostics() []PlanValidationError
}

// PlanSemanticError is a single semantic validation error with an
// exact RFC 6901 JSON Pointer path.
type PlanSemanticError struct {
	Diagnostic PlanValidationError
	Cause      error
}

// PlanDiagnostics implements planDiagnosticSource. The diagnostic is
// deep-copied so callers can mutate the returned AcceptedValues
// without affecting the error's internal state.
func (e *PlanSemanticError) PlanDiagnostics() []PlanValidationError {
	return []PlanValidationError{clonePlanValidationError(e.Diagnostic)}
}

// Error implements the error interface.
func (e *PlanSemanticError) Error() string {
	return e.Diagnostic.Message
}

// Unwrap returns the underlying cause for errors.Is/errors.As.
func (e *PlanSemanticError) Unwrap() error {
	return e.Cause
}

// PlanSemanticMultiError aggregates multiple diagnostics for errors
// that need to report more than one problem.
type PlanSemanticMultiError struct {
	Diagnostics []PlanValidationError
	Cause       error
}

// PlanDiagnostics implements planDiagnosticSource. The diagnostics are
// deep-copied so callers can mutate the returned AcceptedValues
// without affecting the error's internal state.
func (e *PlanSemanticMultiError) PlanDiagnostics() []PlanValidationError {
	return clonePlanValidationErrors(e.Diagnostics)
}

// Error implements the error interface.
func (e *PlanSemanticMultiError) Error() string {
	if len(e.Diagnostics) == 0 {
		return "multiple semantic validation errors"
	}
	return e.Diagnostics[0].Message
}

// Unwrap returns the underlying cause for errors.Is/errors.As.
func (e *PlanSemanticMultiError) Unwrap() error {
	return e.Cause
}

// newSemanticError constructs a PlanSemanticError with exact paths.
func newSemanticError(instancePath string, code PlanValidationCode, keyword PlanValidationKeyword, message string, cause error) *PlanSemanticError {
	return &PlanSemanticError{
		Diagnostic: PlanValidationError{
			InstancePath: instancePath,
			SchemaPath:   instancePath,
			Code:         code,
			Keyword:      keyword,
			Message:      message,
		},
		Cause: cause,
	}
}

// newSemanticMultiError constructs a PlanSemanticMultiError.
func newSemanticMultiError(diagnostics []PlanValidationError, cause error) *PlanSemanticMultiError {
	return &PlanSemanticMultiError{
		Diagnostics: diagnostics,
		Cause:       cause,
	}
}
