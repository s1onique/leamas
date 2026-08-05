package closure

// plan_contract_validation_composed.go contains the composed
// validation pipeline (Phase 10) and the mode-dependent
// applicability walker (Phase 9). Splitting it from
// plan_contract_validation.go keeps every file under the
// LLM-friendly 400-line threshold while preserving the single
// closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 requires.

// ComposedPlanValidationResult is the JSON-ready structured stage
// model the future CLI consumes.
type ComposedPlanValidationResult struct {
	Structural     PlanValidationResult  `json:"structural"`
	Decoded        bool                  `json:"decoded"`
	DecodeErrors   []PlanValidationError `json:"decode_errors"`
	SemanticValid  bool                  `json:"semantic_valid"`
	SemanticErrors []PlanValidationError `json:"semantic_errors"`
	Valid          bool                  `json:"valid"`
}

type composedValidationDeps struct {
	DecodeTyped func(root any, observer compositionObserver) (Plan, error)
}

func defaultComposedValidationDeps() composedValidationDeps {
	return composedValidationDeps{
		DecodeTyped: decodeTypedPlanWithObserver,
	}
}

type typedDecodeMissingDependencyError struct{}

func (typedDecodeMissingDependencyError) Error() string {
	return "composed validation dependency bundle has no DecodeTyped binding"
}

var errTypedDecodeMissingDependency typedDecodeMissingDependencyError

func validatePlanComposedWithObserver(data []byte, observer compositionObserver) ComposedPlanValidationResult {
	return validatePlanComposedWithObserverAndDeps(data, observer, defaultComposedValidationDeps())
}

func validatePlanComposedWithObserverAndDeps(data []byte, observer compositionObserver, deps composedValidationDeps) ComposedPlanValidationResult {
	result := ComposedPlanValidationResult{
		Structural:     PlanValidationResult{Errors: []PlanValidationError{}},
		DecodeErrors:   []PlanValidationError{},
		SemanticErrors: []PlanValidationError{},
	}
	root, parseDiagnostics := parseBoundedClosurePlanDocument(data)
	observer.Parsed()
	if len(parseDiagnostics) > 0 {
		result.Structural = PlanValidationResult{Valid: false, ContractVersion: 0, Errors: parseDiagnostics}
		result.Decoded = false
		result.SemanticValid = false
		result.Valid = false
		return result
	}
	result.Structural = validatePlanStructuralFromRootWithObserver(root, observer)
	if !result.Structural.Valid {
		result.Decoded = false
		result.SemanticValid = false
		result.Valid = false
		return result
	}
	if deps.DecodeTyped == nil {
		result.Decoded = false
		result.DecodeErrors = []PlanValidationError{typedDecodeDiagnostic(errTypedDecodeMissingDependency)}
		result.SemanticValid = false
		result.Valid = false
		return result
	}
	plan, err := deps.DecodeTyped(root, observer)
	if err != nil {
		result.Decoded = false
		result.DecodeErrors = []PlanValidationError{typedDecodeDiagnostic(err)}
		result.SemanticValid = false
		result.Valid = false
		return result
	}
	result.Decoded = true
	semErr := ValidatePlan(plan)
	observer.SemanticValidated()
	if semErr != nil {
		result.SemanticValid = false
		result.SemanticErrors = semanticDiagnostics(semErr)
		result.Valid = false
		return result
	}
	result.SemanticValid = true
	result.Valid = true
	return result
}

func ValidatePlanComposed(data []byte) ComposedPlanValidationResult {
	return validatePlanComposedWithObserver(data, noopCompositionObserver{})
}

func validatePlanStructuralAndSemanticWith(data []byte, observer compositionObserver, deps composedValidationDeps) (PlanValidationResult, error) {
	result := validatePlanComposedWithObserverAndDeps(data, observer, deps)
	if !result.Structural.Valid {
		return result.Structural, errorFromDiagnostics(result.Structural.Errors)
	}
	if !result.Decoded {
		return result.Structural, errorFromDiagnostics(result.DecodeErrors)
	}
	if !result.SemanticValid {
		return result.Structural, errorFromDiagnostics(result.SemanticErrors)
	}
	return result.Structural, nil
}

func ValidatePlanStructuralAndSemantic(data []byte) (PlanValidationResult, error) {
	return validatePlanStructuralAndSemanticWith(data, noopCompositionObserver{}, defaultComposedValidationDeps())
}

// ValidateModeDependentApplicability walks every check item and
// consults the descriptor's ApplicabilityRules for each field. The
// walker iterates the DESCRIPTOR's fields (not only the JSON
// members present in the document) so it can detect both
// missing-required and present-forbidden conditions
// deterministically.
//
// Diagnostic taxonomy:
//
//	required under sibling:
//	  required_property_missing at the exact instance path
//	forbidden under sibling:
//	  forbidden_presence at the exact instance path
//
// The walker emits forbidden_presence regardless of the supplied
// value so a forbidden field is reported consistently. The
// structural pipeline (validatePlanStructuralFromRootWithObserver)
// post-processes the diagnostic stream to drop any structural
// value-constraint diagnostics for forbidden fields; the
// applicability classification always wins for forbidden fields.
func ValidateModeDependentApplicability(root any, contract planContractV1Descriptor) []PlanValidationError {
	return validateModeDependentApplicabilityWithObserver(root, contract, noopDescriptorValidationObserver{})
}

func validateModeDependentApplicabilityWithObserver(root any, contract planContractV1Descriptor, observer descriptorValidationObserver) []PlanValidationError {
	identityDiagnostics := validateDescriptorApplicabilityIdentity(contract)
	duplicatePaths := duplicateApplicabilityFieldPaths(identityDiagnostics)
	snapshot := make([]PlanValidationError, len(identityDiagnostics))
	copy(snapshot, identityDiagnostics)
	observer.DescriptorIdentityValidated(snapshot)
	var diagnostics []PlanValidationError
	diagnostics = append(diagnostics, identityDiagnostics...)
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
			fieldPath := "/checks/" + itoa(index) + "/" + fieldName
			descriptorFieldPath := "/checks/" + fieldName
			if _, suppressed := duplicatePaths[descriptorFieldPath]; suppressed {
				continue
			}
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
				_, keyPresent := check[fieldName]
				switch rule.Presence {
				case PresenceRequired:
					if !keyPresent {
						diagnostics = append(diagnostics, PlanValidationError{
							InstancePath: fieldPath,
							SchemaPath:   fieldPath,
							Code:         PlanCodeRequiredPropertyMissing,
							Keyword:      KeywordRequired,
							Message:      "required property \"" + fieldName + "\" is missing",
							PropertyName: fieldName,
						})
					}
				case PresenceForbidden:
					if keyPresent {
						diagnostics = append(diagnostics, PlanValidationError{
							InstancePath:  fieldPath,
							SchemaPath:    fieldPath,
							Code:          PlanCodeForbiddenPresence,
							Keyword:       KeywordIfThenElse,
							Message:       "property \"" + fieldName + "\" is forbidden when mode=\"" + modeStr + "\"",
							PropertyName:  fieldName,
							RejectedValue: check[fieldName],
						})
					}
				}
			}
		}
	}
	return diagnostics
}

// applicabilityRulesFor returns the descriptor's authoritative
// rule list for a field.
func applicabilityRulesFor(field planFieldDescriptor) []fieldApplicabilityRule {
	return field.ApplicabilityRules
}

func applicabilityPresenceForMode(field planFieldDescriptor, mode string) PresenceRule {
	for _, rule := range field.ApplicabilityRules {
		if rule.Sibling == "mode" && rule.Value == mode {
			return rule.Presence
		}
	}
	return PresenceOptional
}

// filterStructuralByApplicability drops structural diagnostics for
// any (instance_path, property_name) pair that the applicability
// walker classified as forbidden_presence. The applicability
// classification wins so a forbidden field reports the same
// deterministic category regardless of the supplied value.
//
// The function is total: when the applicability stream is empty or
// carries no forbidden_presence codes, the structural stream is
// returned unchanged.
func filterStructuralByApplicability(structural []PlanValidationError, applicability []PlanValidationError) []PlanValidationError {
	if len(applicability) == 0 || len(structural) == 0 {
		return structural
	}
	forbidden := make(map[string]struct{}, len(applicability))
	for _, d := range applicability {
		if d.Code != PlanCodeForbiddenPresence {
			continue
		}
		key := d.InstancePath + "\x00" + d.PropertyName
		forbidden[key] = struct{}{}
	}
	if len(forbidden) == 0 {
		return structural
	}
	out := make([]PlanValidationError, 0, len(structural))
	for _, d := range structural {
		key := d.InstancePath + "\x00" + d.PropertyName
		if _, drop := forbidden[key]; drop {
			continue
		}
		out = append(out, d)
	}
	return out
}
