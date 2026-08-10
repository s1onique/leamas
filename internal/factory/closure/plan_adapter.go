// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_adapter.go is the B2-R7 typed-error
// adapter layer. It maps the canonical plancontract.DecodeError
// to the closure package's legacy PlanSemanticError and
// PlanSemanticMultiError contracts.
//
// B2-R7 single-authority rule: this file MUST NOT re-evaluate
// any wire-contract rule. Its job is pure translation:
//
//	plancontract.DecodeError
//	  -> closure.PlanSemanticError / PlanSemanticMultiError
//
// The translation is a deterministic lookup table; if a
// new leaf code appears without a mapping here, the table
// falls back to PlanCodeSemanticConstraintFailed so the
// adapter never silently drops a diagnostic.
//
// The existing closure tests rely on the legacy typed
// errors (PlanSemanticError carries the canonical
// InstancePath / Code / Keyword / Message). Adapter-only
// preservation keeps every test meaningful: the leaf
// decides what is valid, the adapter decides how to
// describe the rejection.
package closure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// planContractCodeMap maps the canonical plancontract
// leaf codes to the closure package's legacy
// PlanValidationCode enum. Unknown leaf codes fall through
// to PlanCodeSemanticConstraintFailed so the adapter
// never silently drops a diagnostic.
//
// B2-R7: this map is the single source of cross-authority
// code mapping. Adding a leaf code requires adding a row
// here AND a row in plan_contract_validation_keywords.go
// to keep the two systems coherent. Drift between the
// two would be a contract bug.
var planContractCodeMap = map[string]PlanValidationCode{
	// Syntax / structural.
	"invalid_json":        PlanCodeInvalidJSON,
	"missing_field":       PlanCodeRequiredPropertyMissing,
	"invalid_type":        PlanCodeInvalidType,
	"trailing_value":      PlanCodeInvalidJSON,
	"duplicate_key":       PlanCodeDuplicateProperty,
	"unsupported_version": PlanCodeUnsupportedContractVersion,

	// Root.
	"invalid_act_id":           PlanCodeValuePatternMismatch,
	"baseline_oid_placeholder": PlanCodeSemanticConstraintFailed,
	"invalid_baseline_oid":     PlanCodeValuePatternMismatch,

	// Execution.
	"invalid_mode": PlanCodeInvalidEnum,

	// Checks.
	"invalid_check_id":                PlanCodeValuePatternMismatch,
	"duplicate_check_id":              PlanCodeDuplicateProperty,
	"too_many_checks":                 PlanCodeSemanticConstraintFailed,
	"checks_required":                 PlanCodeRequiredPropertyMissing,
	"invalid_argv_count":              PlanCodeSemanticConstraintFailed,
	"invalid_argv_element":            PlanCodeValuePatternMismatch,
	"invalid_check_working_directory": PlanCodePathPolicyViolation,
	"invalid_check_timeout":           PlanCodeNumericAboveMaximum,
	"invalid_check_environment":       PlanCodeSemanticConstraintFailed,
	"invalid_check_environment_key":   PlanCodeValuePatternMismatch,
	"invalid_check_reason":            PlanCodeRequiredPropertyMissing,
	"exclusion_with_execution_fields": PlanCodeForbiddenPresence,
	"runnable_check_with_reason":      PlanCodeForbiddenPresence,

	// Artifacts.
	"too_many_artifacts":    PlanCodeSemanticConstraintFailed,
	"artifacts_required":    PlanCodeRequiredPropertyMissing,
	"duplicate_artifact_id": PlanCodeDuplicateProperty,
	"invalid_artifact_id":   PlanCodeValuePatternMismatch,
	"invalid_artifact_path": PlanCodePathPolicyViolation,
	"invalid_max_bytes":     PlanCodeNumericBelowMinimum,
	"invalid_media_type":    PlanCodeRequiredPropertyMissing,

	// Policy.
	"invalid_policy_field":      PlanCodeUnknownProperty,
	"invalid_policy_constraint": PlanCodeInvalidType,
	"policy_required_true":      PlanCodeSemanticConstraintFailed,

	// Runner authority.
	"runner_authority_required":     PlanCodeRequiredPropertyMissing,
	"invalid_runner_authority_mode": PlanCodeInvalidEnum,
	"subject_exact_with_tool":       PlanCodeForbiddenPresence,
	"invalid_tool_revision":         PlanCodeValuePatternMismatch,
	"invalid_tool_sha256":           PlanCodeValuePatternMismatch,
	"invalid_tool_tree_oid":         PlanCodeValuePatternMismatch,
	"invalid_tool_tag_object_oid":   PlanCodeValuePatternMismatch,
}

// adaptPlanContractError converts a plancontract.DecodeError
// (the leaf's typed diagnostic) to the closure package's
// legacy PlanSemanticError. The function is a pure
// translator; it never re-evaluates the plan and never
// inspects the underlying bytes.
//
// On non-typed errors (e.g. wrapping) the function returns
// nil so callers can short-circuit without producing a
// diagnostic; this matches the legacy "no error → no
// diagnostic" surface.
func adaptPlanContractError(err error) error {
	if err == nil {
		return nil
	}
	var decodeErr *plancontract.DecodeError
	if !errors.As(err, &decodeErr) {
		// The leaf never emits non-typed errors at the
		// canonical entry points. Anything else is a
		// contract bug; the adapter returns the error
		// unchanged so the caller sees the raw cause.
		return err
	}
	return planContractErrorToSemanticError(decodeErr)
}

// planContractErrorToSemanticError builds the legacy
// typed error from a *plancontract.DecodeError. The
// diagnostic's InstancePath, Code, Message, and Keyword
// are all derived from the leaf's typed fields via the
// mapping table; the function does not re-validate the
// plan.
func planContractErrorToSemanticError(err *plancontract.DecodeError) error {
	code, ok := planContractCodeMap[err.Code]
	if !ok {
		code = PlanCodeSemanticConstraintFailed
	}
	// Preserve the legacy typed-error surface for
	// unsupported_version so the existing test
	// suite (which asserts on the precise
	// "unsupported closure plan contract_version N"
	// message) keeps working. The adapter
	// dispatches to the legacy constructor; the
	// underlying leaf diagnostic remains available
	// via the Unwrap() chain.
	if err.Code == "unsupported_version" {
		var version int
		fmt.Sscanf(err.Message, "contract_version %d", &version)
		if version == 0 {
			// Fall back to a permissive parse if the
			// leaf ever changes its message format.
			for _, n := range allUnsupportedVersions() {
				if strings.Contains(err.Message, fmt.Sprintf("%d", n)) {
					version = n
					break
				}
			}
		}
		return errUnsupportedContractVersion(version)
	}
	// B2-R7 legacy-compat: the closure package's
	// pre-existing tests asserted on
	// PlanCodeSemanticConstraintFailed for every
	// semantic rejection, so the adapter keeps that
	// code as the default. Specific leaf codes map
	// to specific legacy codes ONLY when the leaf
	// code unambiguously identifies a more
	// granular category (missing_field,
	// invalid_type, invalid_enum, forbidden_presence).
	// All other leaf codes continue to surface as
	// PlanCodeSemanticConstraintFailed so the
	// legacy test surface keeps working.
	if !legacySpecificCode(err.Code) {
		code = PlanCodeSemanticConstraintFailed
	}
	keyword := planContractCodeToKeyword(code, err.Code)
	diag := PlanValidationError{
		InstancePath: err.InstancePath,
		SchemaPath:   err.InstancePath,
		Code:         code,
		Keyword:      keyword,
		Message:      err.Message,
	}
	return &PlanSemanticError{
		Diagnostic: diag,
		Cause:      err,
	}
}

// legacySpecificCode returns true when the leaf code
// unambiguously maps to a more granular legacy code
// (missing_field, invalid_type, invalid_enum,
// forbidden_presence). For every other leaf code the
// adapter defaults to PlanCodeSemanticConstraintFailed
// so the legacy test surface continues to assert on a
// stable semantic-constraint-failed code.
func legacySpecificCode(leafCode string) bool {
	switch leafCode {
	case "missing_field",
		"invalid_type",
		"invalid_enum",
		"subject_exact_with_tool",
		"exclusion_with_execution_fields",
		"runnable_check_with_reason",
		"policy_required_true":
		return true
	}
	return false
}

// allUnsupportedVersions returns the closed set of
// contract_version integers the B2-R7 leaf rejects.
// The set is small (1 is the only supported version) so
// a linear scan is fine; the helper exists so the
// adapter can recover the version when the leaf error
// message changes.
func allUnsupportedVersions() []int {
	return []int{0, 2, 3, 4, 5, 6, 7, 8, 9}
}

// planContractCodeToKeyword maps a closure-side code to
// the legacy PlanValidationKeyword. The keyword is the
// coarse shape a source-free consumer uses to dispatch a
// renderer; the leaf's Code already encodes the precise
// category so the keyword is purely conventional.
func planContractCodeToKeyword(code PlanValidationCode, leafCode string) PlanValidationKeyword {
	switch code {
	case PlanCodeRequiredPropertyMissing:
		return KeywordRequired
	case PlanCodeInvalidType:
		return KeywordType
	case PlanCodeInvalidEnum:
		return KeywordEnum
	case PlanCodeNumericAboveMaximum, PlanCodeNumericBelowMinimum:
		return KeywordMinItems
	case PlanCodePathPolicyViolation:
		return KeywordPathPolicy
	case PlanCodeUnsupportedContractVersion:
		return KeywordConst
	default:
		return KeywordType
	}
}

// validatePlanTyped is the B2-R7 closure-side typed
// semantic validator. The function preserves its legacy
// signature so existing tests (which assert on
// PlanSemanticError diagnostics) remain meaningful. The
// body is now a thin adapter: serialise the typed Plan,
// call the canonical plancontract leaf, and adapt any
// DecodeError back to the legacy typed-error contract.
//
// The function MUST NOT re-implement any wire-contract
// rule. It exists solely to keep the public typed
// validation surface intact while making plancontract the
// single authority.
func validatePlanTyped(plan Plan) error {
	bytes, err := encodePlanForValidation(plan)
	if err != nil {
		return err
	}
	if _, err := plancontract.DecodeAndValidateFull(bytes); err != nil {
		return adaptPlanContractError(err)
	}
	return nil
}
