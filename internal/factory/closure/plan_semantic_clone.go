package closure

// plan_semantic_clone.go contains canonical deep-copy helpers for
// PlanValidationError to ensure mutation isolation between
// extraction calls.

// clonePlanValidationError returns a deep copy of the diagnostic.
// The AcceptedValues slice is cloned so callers can mutate the
// returned AcceptedValues without affecting the original.
func clonePlanValidationError(diagnostic PlanValidationError) PlanValidationError {
	out := diagnostic
	if len(diagnostic.AcceptedValues) > 0 {
		accepted := make([]string, len(diagnostic.AcceptedValues))
		copy(accepted, diagnostic.AcceptedValues)
		out.AcceptedValues = accepted
	}
	// RejectedValue is left as-is since it is an interface{}
	// and copying interface values preserves the underlying value.
	// Callers that need deep-copy of RejectedValue must handle
	// that separately based on their specific needs.
	return out
}

// clonePlanValidationErrors returns a deep copy of the diagnostics slice.
// Each diagnostic is cloned so mutations to individual diagnostics do not
// affect the original slice.
func clonePlanValidationErrors(diagnostics []PlanValidationError) []PlanValidationError {
	if diagnostics == nil {
		return nil
	}
	out := make([]PlanValidationError, len(diagnostics))
	for i := range diagnostics {
		out[i] = clonePlanValidationError(diagnostics[i])
	}
	return out
}
