// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full_policy.go owns
// the Plan Contract v1 /policy validation. B2-R5 makes
// policy REQUIRED: every closure plan MUST carry all four
// required fields, and require_clean_before /
// require_clean_after MUST be true. This is the canonical
// authority the closure runner and the evidence package
// share.
//
// The closure package's typed validator (closure/plan_policy.go)
// was deleted in B2-R5 because the wire-contract policy
// rules now live exclusively here. The closure package's
// PlanPolicy struct remains as the wire-shape carrier.
package plancontract

import "fmt"

// policyRequiredFields is the ordered set of /policy
// sibling names that MUST be present in every Plan
// Contract v1 document. The closure package's
// PolicyFieldOrder() MUST agree with this list; drift is
// a contract bug.
var policyRequiredFields = []string{
	"require_clean_before",
	"require_clean_after",
	"forbid_tracked_full_digests",
	"require_diff_check",
}

// validatePolicyRequired enforces:
//   - /policy MUST be present as a JSON object (NOT optional,
//     NOT null).
//   - each required field MUST be present and a JSON boolean.
//   - require_clean_before MUST be true.
//   - require_clean_after MUST be true.
//   - forbid_tracked_full_digests MUST be present (true or false).
//   - require_diff_check MUST be present (true or false).
//   - any unrecognised sibling MUST be rejected.
func validatePolicyRequired(obj map[string]any) error {
	rawPolicy, ok := obj["policy"]
	if !ok || rawPolicy == nil {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "policy",
			InstancePath: "/policy",
			Message:      "policy is required",
		}
	}
	policy, ok := rawPolicy.(map[string]any)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        "policy",
			InstancePath: "/policy",
			Message:      "policy is not an object",
		}
	}
	return validatePolicyMap(policy)
}

// validatePolicyMap enforces the per-field policy rules.
func validatePolicyMap(policy map[string]any) error {
	for _, key := range policyRequiredFields {
		v, ok := policy[key]
		if !ok {
			return &DecodeError{
				Code:         "missing_field",
				Field:        "policy." + key,
				InstancePath: "/policy/" + key,
				Message:      fmt.Sprintf("policy.%s is required", key),
			}
		}
		b, ok := v.(bool)
		if !ok {
			return &DecodeError{
				Code:         "invalid_policy_constraint",
				Field:        "policy." + key,
				InstancePath: "/policy/" + key,
				Message:      fmt.Sprintf("policy.%s must be a boolean", key),
			}
		}
		if key == "require_clean_before" && !b {
			return &DecodeError{
				Code:         "clean_before_required_true",
				Field:        "policy.require_clean_before",
				InstancePath: "/policy/require_clean_before",
				Message:      "policy.require_clean_before must be true",
			}
		}
		if key == "require_clean_after" && !b {
			return &DecodeError{
				Code:         "clean_after_required_true",
				Field:        "policy.require_clean_after",
				InstancePath: "/policy/require_clean_after",
				Message:      "policy.require_clean_after must be true",
			}
		}
	}
	for key := range policy {
		found := false
		for _, req := range policyRequiredFields {
			if key == req {
				found = true
				break
			}
		}
		if !found {
			return &DecodeError{
				Code:         "invalid_policy_field",
				Field:        "policy." + key,
				InstancePath: "/policy/" + key,
				Message:      fmt.Sprintf("policy.%s is not a known policy field", key),
			}
		}
	}
	return nil
}
