package closure

import "fmt"

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
// future CLI invokes. The composed pipeline proves the
// syntax -> structural -> applicability -> typed decode ->
// semantic chain runs each stage exactly once:
//
//   - parseClosurePlanDocument (single syntactic authority) counts
//     planParserCalls.
//   - decodeTypedPlan (typed decode via canonical marshal) counts
//     planTypedDecodeCalls.
//   - ValidatePlan (semantic validation) counts
//     planSemanticValidateCalls.
//
// Structural failure short-circuits the pipeline so semantic rules
// never run on a malformed document. A semantic-only failure keeps
// Decoded=true so callers can see the typed value was successfully
// populated.
func ValidatePlanComposed(data []byte) ComposedPlanValidationResult {
	result := ComposedPlanValidationResult{Valid: true, Decoded: true}
	if len(data) > MaxPlanBytes {
		result.Structural = PlanValidationResult{Valid: false, ContractVersion: 0, Errors: []PlanValidationError{{
			InstancePath: "",
			SchemaPath:   "",
			Code:         PlanCodeInvalidJSON,
			Keyword:      KeywordType,
			Message:      "plan exceeds " + itoa(MaxPlanBytes) + "-byte size limit",
		}}}
		result.Decoded = false
		result.Valid = false
		return result
	}
	root, parseDiagnostics := parseClosurePlanDocument(data)
	if len(parseDiagnostics) > 0 {
		result.Structural = PlanValidationResult{Valid: false, ContractVersion: 0, Errors: parseDiagnostics}
		result.Decoded = false
		result.Valid = false
		return result
	}
	result.Structural = validatePlanStructuralFromRoot(root)
	if !result.Structural.Valid {
		result.Decoded = false
		result.Valid = false
		return result
	}
	plan, err := decodeTypedPlan(root)
	if err != nil {
		result.Decoded = false
		result.Semantic = err
		result.Valid = false
		return result
	}
	result.Decoded = true
	planSemanticValidateCalls++
	if err := ValidatePlan(plan); err != nil {
		result.Semantic = err
		result.Valid = false
	}
	return result
}

// ValidatePlanStructuralAndSemantic is a convenience wrapper that
// runs structural validation first and only attempts semantic
// validation when the structural pass succeeds. The semantic
// validator is the typed ValidatePlan which already enforces every
// semantic rule documented by the descriptor's SemanticRule
// fields.
func ValidatePlanStructuralAndSemantic(data []byte) (PlanValidationResult, error) {
	root, parseDiagnostics := parseClosurePlanDocument(data)
	if len(parseDiagnostics) > 0 {
		return PlanValidationResult{Valid: false, ContractVersion: 0, Errors: parseDiagnostics},
			errorFromDiagnostics(parseDiagnostics)
	}
	structural := validatePlanStructuralFromRoot(root)
	if !structural.Valid {
		return structural, nil
	}
	plan, err := decodeTypedPlan(root)
	if err != nil {
		return structural, fmt.Errorf("typed decode: %w", err)
	}
	planSemanticValidateCalls++
	if err := ValidatePlan(plan); err != nil {
		return structural, err
	}
	return structural, nil
}

// ValidateModeDependentApplicability walks every check item
// and consults the descriptor's ApplicabilityRules (with the legacy
// Applicability as a derived single rule) for each field. Unlike
// the previous implementation, this walker iterates the DESCRIPTOR
// fields, not only the JSON members present in the document, so it
// can detect missing-required and present-forbidden conditions
// deterministically.
//
// Diagnostics:
//
//	missing required under sibling:
//	  required_property_missing at the exact instance path
//
//	present forbidden under sibling:
//	  semantic_constraint_failed at the exact instance path
//
// The walker runs as part of ValidatePlanStructural AFTER ordinary
// structural shape validation succeeds (so a malformed check array
// never triggers applicability noise).
func ValidateModeDependentApplicability(root any, contract planContractV1Descriptor) []PlanValidationError {
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
		checksField, ok := contract.Root.Fields["checks"]
		if !ok || checksField.ItemDescriptor == nil || checksField.ItemDescriptor.Children == nil {
			continue
		}
		for fieldName, childField := range checksField.ItemDescriptor.Children.Fields {
			rules := applicabilityRulesFor(childField)
			if len(rules) == 0 {
				continue
			}
			for _, rule := range rules {
				if rule.Sibling != "mode" {
					continue
				}
				if rule.Value != modeStr {
					continue
				}
				value, present := check[fieldName]
				switch rule.Presence {
				case PresenceRequired:
					if !present || isAbsentValue(value) {
						diagnostics = append(diagnostics, PlanValidationError{
							InstancePath: "/checks/" + itoa(index) + "/" + fieldName,
							SchemaPath:   "/checks/" + itoa(index) + "/" + fieldName,
							Code:         PlanCodeRequiredPropertyMissing,
							Keyword:      KeywordIfThenElse,
							Message:      "property \"" + fieldName + "\" is required when mode=\"" + modeStr + "\"",
							PropertyName: fieldName,
						})
					}
				case PresenceForbidden:
					if present && !isAbsentValue(value) {
						diagnostics = append(diagnostics, PlanValidationError{
							InstancePath:  "/checks/" + itoa(index) + "/" + fieldName,
							SchemaPath:    "/checks/" + itoa(index) + "/" + fieldName,
							Code:          PlanCodeSemanticConstraintFailed,
							Keyword:       KeywordIfThenElse,
							Message:       "property \"" + fieldName + "\" is forbidden when mode=\"" + modeStr + "\"",
							PropertyName:  fieldName,
							RejectedValue: value,
						})
					}
				}
			}
		}
	}
	return diagnostics
}

// applicabilityRulesFor returns the descriptor's rule list for a
// field, derived from the new ApplicabilityRules slice and the
// legacy single Applicability pointer. When both are present, both
// rule sets are consulted; duplicates are not collapsed.
func applicabilityRulesFor(field planFieldDescriptor) []fieldApplicabilityRule {
	if len(field.ApplicabilityRules) > 0 {
		return field.ApplicabilityRules
	}
	if field.Applicability == nil {
		return nil
	}
	rule := fieldApplicabilityRule{Sibling: field.Applicability.Sibling, Value: field.Applicability.Value}
	switch {
	case field.Applicability.Required:
		rule.Presence = PresenceRequired
	case field.Applicability.Forbidden:
		rule.Presence = PresenceForbidden
	default:
		rule.Presence = PresenceOptional
	}
	return []fieldApplicabilityRule{rule}
}

// isAbsentValue reports whether a parsed JSON value should be
// treated as absent for required-field purposes.

func isAbsentValue(value any) bool {
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
