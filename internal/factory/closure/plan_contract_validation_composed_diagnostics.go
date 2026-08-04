package closure

// plan_contract_validation_composed_diagnostics.go centralises
// the diagnostic-mapping helpers the composed pipeline uses.
// Splitting them out keeps plan_contract_validation_composed.go
// under the LLM-friendly 400-line threshold while every helper
// remains reviewable in one place.

// typedDecodeDiagnostic wraps a typed-decode error as a
// decode-stage diagnostic. The typed-decode failure lives in the
// decode_errors stage array of the composed result; precise paths
// are deferred to a later correction.
func typedDecodeDiagnostic(err error) PlanValidationError {
	return PlanValidationError{
		InstancePath: "",
		SchemaPath:   "",
		Code:         PlanCodeInvalidType,
		Keyword:      KeywordType,
		Message:      "typed decode: " + err.Error(),
	}
}
