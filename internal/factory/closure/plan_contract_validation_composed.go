package closure

import (
	"fmt"
	"strings"
)

// plan_contract_validation_composed.go contains the composed
// validation pipeline (Phase 10) and the mode-dependent
// applicability walker (Phase 9). Splitting it from
// plan_contract_validation.go keeps every file under the
// LLM-friendly 400-line threshold while preserving the single
// closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 requires.

// ComposedPlanValidationResult is the JSON-ready structured stage
// model the future CLI consumes. The result has four explicit
// stages (structural / decode / semantic / verdict) with json
// tags. Each diagnostic array is initialised non-nil so success
// JSON serialises as [] not null. The legacy exported `Semantic
// error` field is gone.
type ComposedPlanValidationResult struct {
	Structural     PlanValidationResult  `json:"structural"`
	Decoded        bool                  `json:"decoded"`
	DecodeErrors   []PlanValidationError `json:"decode_errors"`
	SemanticValid  bool                  `json:"semantic_valid"`
	SemanticErrors []PlanValidationError `json:"semantic_errors"`
	Valid          bool                  `json:"valid"`
}

// validatePlanComposedWithObserver is the single internal entry
// point that owns the composed pipeline:
//
//  1. Bounded single parse via parseBoundedClosurePlanDocument
//     (one syntactic authority; MaxPlanBytes cap; trailing
//     rejection; duplicate-key rejection).
//  2. Structural + applicability validation via
//     validatePlanStructuralFromRootWithObserver.
//  3. Typed decode via decodeTypedPlanWithObserver.
//  4. Semantic validation via ValidatePlan (called at most once).
//
// The observer is invocation-local: production passes
// noopCompositionObserver{}; tests pass a per-assertion counting
// observer. There is no package-global mutable counter.
func validatePlanComposedWithObserver(data []byte, observer compositionObserver) ComposedPlanValidationResult {
	result := ComposedPlanValidationResult{Valid: true, Decoded: true, SemanticValid: true}
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
	plan, err := decodeTypedPlanWithObserver(root, observer)
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
		result.SemanticErrors = []PlanValidationError{semanticDiagnostic(semErr)}
		result.Valid = false
	}
	return result
}

// ValidatePlanComposed is the public single internal entry point
// the future CLI invokes. It uses the noop observer so production
// has no mutable composition state.
func ValidatePlanComposed(data []byte) ComposedPlanValidationResult {
	return validatePlanComposedWithObserver(data, noopCompositionObserver{})
}

// ValidatePlanStructuralAndSemantic is a convenience wrapper
// that runs the composed pipeline and surfaces the diagnostic of
// the actual failing stage. Stage precedence: structural -> typed
// -> semantic. It never returns the generic "semantic validation
// failed" for a structural or typed-decode failure.
func ValidatePlanStructuralAndSemantic(data []byte) (PlanValidationResult, error) {
	result := validatePlanComposedWithObserver(data, noopCompositionObserver{})
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

// firstSemanticError returns the first error message of the typed
// semantic diagnostics so the legacy DecodePlan (Plan, error)
// wrapper continues to work. The structured JSON result exposes
// the full diagnostics; this helper only produces a one-line
// summary for the legacy error-returning path.
func firstSemanticError(result ComposedPlanValidationResult) error {
	if len(result.SemanticErrors) == 0 {
		return fmt.Errorf("semantic validation failed")
	}
	d := result.SemanticErrors[0]
	return fmt.Errorf("%s: %s", d.InstancePath, d.Message)
}

// typedDecodeDiagnostic wraps a typed-decode error as a stable
// semantic_constraint_failed diagnostic with the precise path the
// future CLI can render.
func typedDecodeDiagnostic(err error) PlanValidationError {
	return PlanValidationError{
		InstancePath: "",
		SchemaPath:   "",
		Code:         PlanCodeSemanticConstraintFailed,
		Keyword:      KeywordType,
		Message:      "typed decode: " + err.Error(),
	}
}

// semanticDiagnostic maps a semantic Go error to a structured
// diagnostic. The path is /act_id for ActID errors, /baseline for
// baseline errors, /execution/mode for execution mode errors, and
// the root for everything else; the field/codename is preserved in
// the message for human readers.
func semanticDiagnostic(err error) PlanValidationError {
	msg := err.Error()
	path := semanticPathFromError(msg)
	return PlanValidationError{
		InstancePath: path,
		SchemaPath:   path,
		Code:         PlanCodeSemanticConstraintFailed,
		Keyword:      KeywordType,
		Message:      msg,
	}
}

// semanticPathFromError extracts the most precise instance path
// documented in the error message. The mapping is conservative;
// unknown forms fall back to the root.
func semanticPathFromError(msg string) string {
	switch {
	case strings.HasPrefix(msg, "invalid act_id"):
		return "/act_id"
	case strings.HasPrefix(msg, "baseline.commit_oid"), strings.HasPrefix(msg, "baseline.tree_oid"):
		return "/" + strings.SplitN(msg, " ", 2)[0]
	case strings.HasPrefix(msg, "checks[") && strings.Contains(msg, "duplicate check id"):
		// duplicate check id "<id>": caller extracts <id> from the message.
		return extractDuplicateCheckIDPath(msg)
	case strings.HasPrefix(msg, "unsupported closure plan contract_version"):
		return "/contract_version"
	}
	return ""
}

func extractDuplicateCheckIDPath(msg string) string {
	const marker = "duplicate check id \""
	i := strings.Index(msg, marker)
	if i < 0 {
		return ""
	}
	rest := msg[i+len(marker):]
	j := strings.Index(rest, "\"")
	if j < 0 {
		return ""
	}
	return "/checks/" + rest[:j]
}

// ValidateModeDependentApplicability walks every check item
// and consults the descriptor's ApplicabilityRules for each
// field. The walker iterates the DESCRIPTOR's fields (not only the
// JSON members present in the document) so it can detect both
// missing-required and present-forbidden conditions
// deterministically.
//
// Presence semantics are key-existence only. A forbidden field
// is rejected whenever the JSON key is present at all (the value
// may be empty, null, an empty string, or a zero-length
// collection).
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
				_, keyPresent := check[fieldName]
				switch rule.Presence {
				case PresenceRequired:
					if !keyPresent {
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
					if keyPresent {
						diagnostics = append(diagnostics, PlanValidationError{
							InstancePath:  "/checks/" + itoa(index) + "/" + fieldName,
							SchemaPath:    "/checks/" + itoa(index) + "/" + fieldName,
							Code:          PlanCodeSemanticConstraintFailed,
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
// rule list for a field. The ApplicabilityRules slice is the sole
// authority.
func applicabilityRulesFor(field planFieldDescriptor) []fieldApplicabilityRule {
	return field.ApplicabilityRules
}

// applicabilityPresenceForMode resolves the presence rule for a
// given field under a specific mode value. The canonical example
// generator uses this helper to decide whether to emit a field in
// the run-mode fixture. The default is PresenceOptional so
// the helper is total.
func applicabilityPresenceForMode(field planFieldDescriptor, mode string) PresenceRule {
	for _, rule := range field.ApplicabilityRules {
		if rule.Sibling == "mode" && rule.Value == mode {
			return rule.Presence
		}
	}
	return PresenceOptional
}
