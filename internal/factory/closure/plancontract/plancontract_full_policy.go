// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full_policy.go owns
// the Plan Contract v1 /policy validation. B2-R6 requires
// every closure plan to carry all four required policy
// siblings as JSON booleans equal to true. This is the
// canonical authority the closure runner and the evidence
// package share.
//
// B2-R5 made policy REQUIRED and required require_clean_before
// and require_clean_after to be true. B2-R6 extends the
// "must equal true" rule to ALL four siblings so the leaf
// agrees with the canonical descriptor that the closure
// package's typed validator historically enforced.
//
// The closure package's typed validator (closure/plan_policy.go)
// is preserved for backward compatibility but its rule set
// now mirrors the leaf exactly; it is no longer an
// independent authority.
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
//   - ALL four required booleans MUST equal true.
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
// B2-R6: ALL four required siblings MUST equal true, not
// just require_clean_before/after. The leaf's diagnostic
// is the canonical authority.
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
		if !b {
			return &DecodeError{
				Code:         "policy_required_true",
				Field:        "policy." + key,
				InstancePath: "/policy/" + key,
				Message:      fmt.Sprintf("policy.%s must be true", key),
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
