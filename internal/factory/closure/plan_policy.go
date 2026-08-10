// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_policy.go is the B2-R7 closure-
// side adapter for /policy validation. The function
// validatePlanPolicy preserves its legacy typed-input
// signature so existing tests (which assert on
// PlanPolicyRequiredError) remain meaningful. The body
// is a thin adapter: serialise the typed policy, call
// the canonical plancontract.ValidatePolicyBytes, and
// adapt any DecodeError back to the legacy typed-error
// contract.
//
// B2-R7 single-authority rule: this file MUST NOT carry
// any wire-contract rule. Every such rule lives in the
// plancontract leaf; the closure package's adapters
// (plan_adapter.go, plan_policy.go, runner_authority.go,
// plan_patterns.go) reference the leaf by import.
package closure

import (
	"encoding/json"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// validatePlanPolicy is the B2-R7 closure-side adapter
// for /policy. The function preserves its legacy
// typed-input signature so existing tests (which assert
// on the legacy typed-error contract) remain meaningful.
// The body serialises the typed policy to JSON, calls
// the canonical plancontract.ValidatePolicyBytes, and
// adapts any DecodeError back to the legacy typed-error
// contract.
//
// The adapter MUST NOT re-implement any wire-contract
// rule. The canonical leaf owns every rule; this file
// only translates the leaf's typed diagnostic into the
// closure package's PlanPolicyRequiredError shape.
func validatePlanPolicy(policy PlanPolicy) error {
	data, err := json.Marshal(map[string]any{"policy": policyToWire(policy)})
	if err != nil {
		return &PlanPolicyRequiredError{Missing: missingPolicyFields(policy)}
	}
	if err := plancontract.ValidatePolicyBytes(data); err != nil {
		// Preserve the legacy typed-error contract: a
		// policy missing-field failure surfaces as a
		// PlanPolicyRequiredError so the existing test
		// surface (which asserts on Missing) keeps
		// working.
		if isPolicyMissingField(err) {
			return &PlanPolicyRequiredError{Missing: missingPolicyFields(policy)}
		}
		return adaptPlanContractError(err)
	}
	return nil
}

// missingPolicyFields returns the ordered list of
// /policy sibling names whose value is missing in the
// supplied typed policy. The order is read from the
// descriptor so the descriptor remains the single
// authority.
func missingPolicyFields(policy PlanPolicy) []string {
	values := map[string]*bool{
		"require_clean_before":        policy.RequireCleanBefore,
		"require_clean_after":         policy.RequireCleanAfter,
		"forbid_tracked_full_digests": policy.ForbidTrackedFullDigests,
		"require_diff_check":          policy.RequireDiffCheck,
	}
	order := PolicyFieldOrder()
	missing := make([]string, 0, len(order))
	for _, name := range order {
		if values[name] == nil {
			missing = append(missing, name)
		}
	}
	return missing
}

// isPolicyMissingField reports whether the leaf error
// is a /policy/* missing-field rejection. The check is
// name-based: the leaf sets Code to "missing_field" and
// Field to a /policy/* path for every policy-sibling
// absence.
func isPolicyMissingField(err error) bool {
	var decodeErr *plancontract.DecodeError
	if !errorsAsDecodeError(err, &decodeErr) {
		return false
	}
	return decodeErr.Code == "missing_field" &&
		len(decodeErr.Field) >= len("policy.") &&
		decodeErr.Field[:len("policy.")] == "policy."
}

// errorsAsDecodeError is a tiny errors.As wrapper so
// this file does not need to import "errors" directly.
func errorsAsDecodeError(err error, target **plancontract.DecodeError) bool {
	t, ok := err.(*plancontract.DecodeError)
	if ok {
		*target = t
		return true
	}
	return false
}

// policyToWire converts the typed PlanPolicy into a
// JSON-friendly shape. The function does NOT re-validate;
// it merely copies fields so json.Marshal can emit them.
func policyToWire(policy PlanPolicy) map[string]any {
	out := map[string]any{}
	if policy.RequireCleanBefore != nil {
		out["require_clean_before"] = *policy.RequireCleanBefore
	}
	if policy.RequireCleanAfter != nil {
		out["require_clean_after"] = *policy.RequireCleanAfter
	}
	if policy.ForbidTrackedFullDigests != nil {
		out["forbid_tracked_full_digests"] = *policy.ForbidTrackedFullDigests
	}
	if policy.RequireDiffCheck != nil {
		out["require_diff_check"] = *policy.RequireDiffCheck
	}
	return out
}

// PlanPolicyRequiredError is the legacy typed diagnostic
// the closure package emits when one or more policy
// siblings are absent. B2-R7 preserves the type for
// backward compatibility; the adapter populates it from
// the canonical plancontract.DecodeError.
type PlanPolicyRequiredError struct {
	Missing []string
}

// Error implements the error interface.
func (e *PlanPolicyRequiredError) Error() string {
	if len(e.Missing) == 0 {
		return "policy fields missing"
	}
	return "missing required policy field(s): " + joinStrings(e.Missing, ", ")
}

// IsPlanPolicyRequiredError reports whether err (or any
// wrapped error) is a *PlanPolicyRequiredError.
func IsPlanPolicyRequiredError(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(*PlanPolicyRequiredError); ok {
		return true
	}
	if unwrap, ok := err.(interface{ Unwrap() error }); ok {
		return IsPlanPolicyRequiredError(unwrap.Unwrap())
	}
	return false
}

// joinStrings joins the strings in s with sep. The
// helper avoids importing strings just for one
// concatenation site.
func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	out := s[0]
	for _, v := range s[1:] {
		out += sep + v
	}
	return out
}

// PlanDiagnostics implements planDiagnosticSource. It
// returns:
//   - Empty Missing: non-nil empty slice
//   - Known fields: deduplicated, ordered by PolicyFieldOrder
//   - Unknown fields: sorted alphabetically before rendering
//
// B2-R7: this is preserved as the legacy diagnostic
// surface; the adapter populates it from the canonical
// plancontract DecodeError when the closure runner
// surfaces a policy-related failure.
func (e *PlanPolicyRequiredError) PlanDiagnostics() []PlanValidationError {
	order := PolicyFieldOrder()
	orderSet := make(map[string]bool, len(order))
	for _, name := range order {
		orderSet[name] = true
	}

	knownMissing := make(map[string]bool)
	var unknownFields []string
	seen := make(map[string]bool)
	for _, name := range e.Missing {
		if seen[name] {
			continue
		}
		seen[name] = true
		if orderSet[name] {
			knownMissing[name] = true
		} else {
			unknownFields = append(unknownFields, name)
		}
	}

	diags := make([]PlanValidationError, 0)

	if len(e.Missing) == 0 {
		return diags
	}

	for _, name := range order {
		if knownMissing[name] {
			diags = append(diags, clonePlanValidationError(PlanValidationError{
				InstancePath: "/policy/" + name,
				SchemaPath:   "/policy/" + name,
				Code:         PlanCodeRequiredPropertyMissing,
				Keyword:      KeywordRequired,
				Message:      e.Error(),
				PropertyName: name,
			}))
		}
	}

	if len(unknownFields) > 0 {
		diags = append(diags, clonePlanValidationError(PlanValidationError{
			Code:         PlanCodeSemanticConstraintFailed,
			Keyword:      KeywordType,
			Message:      "unknown policy field(s): " + joinStrings(unknownFields, ", "),
			PropertyName: joinStrings(unknownFields, ","),
		}))
	}

	return diags
}
