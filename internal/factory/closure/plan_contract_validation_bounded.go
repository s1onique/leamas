package closure

// plan_contract_validation_bounded.go centralises the size-bound
// parser. Splitting it from plan_contract_validation.go keeps every
// file under the LLM-friendly 400-line threshold. The
// compositionObserver type and noopCompositionObserver live in
// plan_contract_validation.go alongside the structural validator.

// parseBoundedClosurePlanDocument enforces the documented
// MaxPlanBytes cap and then runs the single strict syntactic
// authority. Every byte-entry validation API in this package
// MUST go through this helper; no exported entry point may bypass
// the bound.
func parseBoundedClosurePlanDocument(data []byte) (any, []PlanValidationError) {
	if len(data) > MaxPlanBytes {
		return nil, []PlanValidationError{{
			InstancePath: "",
			SchemaPath:   "",
			Code:         PlanCodeInvalidJSON,
			Keyword:      KeywordType,
			Message:      "plan exceeds " + itoa(MaxPlanBytes) + "-byte size limit",
		}}
	}
	return parseClosurePlanDocument(data)
}
