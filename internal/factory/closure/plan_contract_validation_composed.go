package closure

// plan_contract_validation_composed.go contains the composed
// validation pipeline (Phase 10) and the mode-dependent
// applicability walker (Phase 9). Splitting it from
// plan_contract_validation.go keeps every file under the
// LLM-friendly 400-line threshold while preserving the single
// closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 requires.

// ComposedPlanValidationResult is the structured outcome of the
// composed validation pipeline. It extends PlanValidationResult
// with a typed-decoding verdict and a semantic-verdict summary so
// the future CLI can render each stage independently.
//
// PlanValidationResult.Valid is the composed verdict: the document
// passes ONLY when every stage passes. Structural failures MUST NOT
// cascade into semantic failures.
type ComposedPlanValidationResult struct {
	Structural PlanValidationResult
	Decoded    bool
	Semantic   error
	Valid      bool
}

// ValidatePlanComposed is the single internal entry point the
// future CLI invokes. It runs:
//   - single-document parsing (Phase 1)
//   - descriptor-driven structural validation (Phase 3+)
//   - typed decoding (DecodePlan)
//   - semantic validation (ValidatePlan)
//
// in that order. Structural failure short-circuits the pipeline so
// semantic rules never run on a malformed document.
func ValidatePlanComposed(data []byte) ComposedPlanValidationResult {
	result := ComposedPlanValidationResult{Valid: true, Decoded: true}
	structural := ValidatePlanStructural(data)
	result.Structural = structural
	if !structural.Valid {
		result.Decoded = false
		result.Valid = false
		return result
	}
	if _, err := DecodePlan(data); err != nil {
		result.Decoded = false
		result.Semantic = err
		result.Valid = false
		return result
	}
	result.Decoded = true
	return result
}

// ValidatePlanStructuralAndSemantic is a convenience wrapper that
// runs structural validation first and only attempts semantic
// validation when the structural pass succeeds. The semantic
// validator is the typed ValidatePlan which already enforces every
// semantic rule documented by the descriptor's SemanticRule
// fields.
func ValidatePlanStructuralAndSemantic(data []byte) (PlanValidationResult, error) {
	structural := ValidatePlanStructural(data)
	if !structural.Valid {
		return structural, nil
	}
	plan, err := DecodePlan(data)
	if err != nil {
		// Surface the structural pass so callers can still see the
		// structural diagnostics alongside the typed failure.
		return structural, err
	}
	if err := ValidatePlan(plan); err != nil {
		return structural, err
	}
	return structural, nil
}

// ValidateModeDependentApplicability walks the parsed root and
// emits required/forbidden diagnostics the descriptor's per-field
// Applicability rules declare. It runs only AFTER structural
// validation has produced a closed-object check tree.
//
// Required cases:
//   - /checks/<index>/argv when sibling mode == "run": REQUIRED
//   - /checks/<index>/reason when sibling mode == "exclude": REQUIRED
//
// Forbidden cases (Phase 9):
//   - /checks/<index>/reason when sibling mode == "run": FORBIDDEN
//   - /checks/<index>/argv when sibling mode == "exclude": FORBIDDEN
func ValidateModeDependentApplicability(root any) []PlanValidationError {
	var diagnostics []PlanValidationError
	checksRaw, ok := root.(map[string]any)["checks"]
	if !ok {
		return diagnostics
	}
	checks, ok := checksRaw.([]any)
	if !ok {
		return diagnostics
	}
	for index, item := range checks {
		checkPath := "/checks/" + itoa(index)
		check, ok := item.(map[string]any)
		if !ok {
			continue
		}
		modeRaw, present := check["mode"]
		if !present {
			continue
		}
		modeStr, ok := modeRaw.(string)
		if !ok {
			continue
		}
		for fieldName, value := range check {
			if value == nil {
				continue
			}
			fieldPath := checkPath + "/" + fieldName
			applicability := applicabilityForField("checks[]", fieldName)
			if applicability == nil {
				continue
			}
			if applicability.Sibling != "mode" {
				continue
			}
			if applicability.Required && applicability.Value == modeStr && fieldName != "mode" {
				if isAbsent(value) {
					diagnostics = append(diagnostics, PlanValidationError{
						InstancePath: fieldPath,
						SchemaPath:   fieldPath,
						Code:         PlanCodeRequiredPropertyMissing,
						Keyword:      KeywordIfThenElse,
						Message:      "property \"" + fieldName + "\" is required when mode=\"" + modeStr + "\"",
						PropertyName: fieldName,
					})
				}
			}
			if applicability.Forbidden && applicability.Value == modeStr {
				diagnostics = append(diagnostics, PlanValidationError{
					InstancePath:  fieldPath,
					SchemaPath:    fieldPath,
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordIfThenElse,
					Message:       "property \"" + fieldName + "\" is forbidden when mode=\"" + modeStr + "\"",
					PropertyName:  fieldName,
					RejectedValue: value,
				})
			}
		}
	}
	return diagnostics
}

// applicabilityForField returns the descriptor's applicability for
// the named check-item field, or nil if the field has no
// applicability. The descriptor is the single source: structural
// and mode-dependent validators cannot drift.
func applicabilityForField(parent, fieldName string) *fieldApplicability {
	contract := planContractV1()
	checksField, ok := contract.Root.Fields["checks"]
	if !ok {
		return nil
	}
	if checksField.ItemDescriptor == nil || checksField.ItemDescriptor.Children == nil {
		return nil
	}
	field, ok := checksField.ItemDescriptor.Children.Fields[fieldName]
	if !ok {
		return nil
	}
	return field.Applicability
}

// isAbsent reports whether a parsed JSON value should be treated as
// absent for required-field purposes.
func isAbsent(value any) bool {
	if value == nil {
		return true
	}
	if str, ok := value.(string); ok && str == "" {
		return true
	}
	if arr, ok := value.([]any); ok && len(arr) == 0 {
		return true
	}
	if obj, ok := value.(map[string]any); ok && len(obj) == 0 {
		return true
	}
	return false
}
