// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full_profile.go owns the
// Plan Contract v1 /policy_profile authority. The leaf is the
// single canonical semantic authority for the optional
// policy_profile field; the closure package's
// validatePlanAuthority helper is preserved as a compat shim
// that re-exports the same answer.
//
// B2-R7-CORRECTION01 motivation: prior to this commit, the
// closure package's ValidatePlan accepted any policy_profile
// value because the plancontract leaf had no per-value
// validator for the field. The closure package's typed
// validator (closure/plan_authority.go::validatePlanAuthority)
// existed but was dead code — the canonical entry point
// validatePlanTyped routed through plancontract.DecodeAndValidateFull,
// and plancontract had no policy_profile rule. A plan claiming
// `policy_profile = "mystery-profile"` therefore passed
// validation, which made the profile claim non-authoritative.
//
// B2-R7-CORRECTION01 makes plancontract the single authority
// for /policy_profile. The leaf rejects:
//   - policy_profile whose value is not in the closed profiles
//     table ("unknown_policy_profile" code).
//   - policy_profile whose profile is recorded but disabled
//     ("disabled_policy_profile" code).
//   - enabled policy_profile whose required checks are missing
//     or whose argv does not exactly match the profile's
//     required argv ("missing_policy_profile_check" code).
//
// The closure package's adapter converts these codes back to
// the legacy PolicyProfileError so existing typed-error
// assertions (TestClosurePlanAuthorityRejects*) continue to
// pass.
package plancontract

import (
	"encoding/json"
	"fmt"
)

// PolicyProfileLeamasActV1 is the only currently-enabled
// policy profile. Future profiles will be added here as
// separate const values so the descriptor and the leaf
// cannot drift.
const (
	PolicyProfileLeamasActV1 = "leamas-act-v1"
	PolicyProfileUnsupported = "unsupported"
)

// RequiredCheck is the canonical declaration of one
// check the policy-profile authority requires. Argv
// compares element-wise against the candidate Plan's
// /checks/<id>/argv field.
type RequiredCheck struct {
	ID   string
	Argv []string
}

// policyProfile is the canonical declaration of a single
// policy profile. Name MUST be unique across the table;
// Enabled=false profiles are still part of the closed
// authority but the leaf rejects them.
type policyProfile struct {
	Name           string
	RequiredChecks []RequiredCheck
	Enabled        bool
}

// policyProfiles is the canonical profile authority. The
// closure package's typed validator MUST consume the same
// table; in B2-R7-CORRECTION01 the closure package's table
// is removed and only this canonical declaration survives.
// Two profiles are reserved for staged rollouts
// (indeep-act-v1, circus-act-v1) so their names are still
// rejected as "disabled" instead of "unknown".
var policyProfiles = map[string]policyProfile{
	PolicyProfileLeamasActV1: {
		Name:    PolicyProfileLeamasActV1,
		Enabled: true,
		RequiredChecks: []RequiredCheck{
			{ID: "focused-count-1", Argv: []string{"go", "test", "-count=1", "./internal/factory/closure/...", "./cmd/leamas/..."}},
			{ID: "focused-count-20", Argv: []string{"go", "test", "-count=20", "./internal/factory/closure/...", "./cmd/leamas/..."}},
			{ID: "focused-race-5", Argv: []string{"go", "test", "-race", "-count=5", "./internal/factory/closure/...", "./cmd/leamas/..."}},
			{ID: "vet", Argv: []string{"go", "vet", "./internal/factory/closure/...", "./cmd/leamas/..."}},
			{ID: "build", Argv: []string{"go", "build", "-buildvcs=true", "-trimpath", "-o", "/tmp/leamas-closure-protocol-v1-self", "./cmd/leamas"}},
			{ID: "gate-fast", Argv: []string{"make", "gate-fast"}},
			{ID: "diff-check", Argv: []string{"git", "diff", "--check"}},
		},
	},
	PolicyProfileUnsupported: {Name: PolicyProfileUnsupported, Enabled: false},
	"indeep-act-v1":          {Name: "indeep-act-v1", Enabled: false},
	"circus-act-v1":          {Name: "circus-act-v1", Enabled: false},
	"clinemm-act-v1":         {Name: "clinemm-act-v1", Enabled: false},
}

// validatePolicyProfileOptional enforces the policy_profile
// authority rules on the parsed root object. The leaf is
// the single source of truth; the closure package's
// adapter maps the resulting DecodeError to its legacy
// PolicyProfileError typed-error contract.
//
//	absent           -> nil
//	non-string       -> DecodeError "invalid_policy_profile_type"
//	unknown value    -> DecodeError "unknown_policy_profile"
//	disabled profile -> DecodeError "disabled_policy_profile"
//	missing check    -> DecodeError "missing_policy_profile_check"
//
// The check runs AFTER validateChecksRequired so the leaf
// guarantees the candidate Plan's /checks array is itself
// structurally valid (non-empty, well-formed entries).
// Profile-required-check matching is the leaf's own argv
// comparison; it does NOT delegate to the typed-Plan
// decoder because the leaf sees the parsed JSON object,
// not the typed struct.
func validatePolicyProfileOptional(obj map[string]any) error {
	raw, ok := obj["policy_profile"]
	if !ok || raw == nil {
		return nil
	}
	profile, ok := raw.(string)
	if !ok {
		return &DecodeError{
			Code:         "invalid_policy_profile_type",
			Field:        "policy_profile",
			InstancePath: "/policy_profile",
			Message:      "policy_profile must be a string",
		}
	}
	declared, ok := policyProfiles[profile]
	if !ok {
		return &DecodeError{
			Code:         "unknown_policy_profile",
			Field:        "policy_profile",
			InstancePath: "/policy_profile",
			Message:      fmt.Sprintf("policy_profile %q is unknown", profile),
		}
	}
	if !declared.Enabled {
		return &DecodeError{
			Code:         "disabled_policy_profile",
			Field:        "policy_profile",
			InstancePath: "/policy_profile",
			Message:      fmt.Sprintf("policy_profile %q is not yet implemented for this repository", profile),
		}
	}
	checks, ok := obj["checks"].([]any)
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "checks",
			InstancePath: "/checks",
			Message:      "checks is required",
		}
	}
	for _, required := range declared.RequiredChecks {
		if !profileCheckSatisfied(checks, required) {
			return &DecodeError{
				Code:         "missing_policy_profile_check",
				Field:        "checks",
				InstancePath: "/checks",
				Message: fmt.Sprintf(
					"plan does not satisfy policy profile %q: missing or non-matching check %q",
					profile, required.ID),
			}
		}
	}
	return nil
}

// profileCheckSatisfied returns true iff the candidate
// /checks array contains exactly one entry whose id matches
// required.ID AND whose argv matches required.Argv
// element-wise. An empty required.ID is rejected so the
// profile authority cannot accidentally match every check.
func profileCheckSatisfied(checks []any, required RequiredCheck) bool {
	if required.ID == "" {
		return false
	}
	for _, raw := range checks {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := entry["id"].(string)
		if id != required.ID {
			continue
		}
		if !argvMatchesString(entry["argv"], required.Argv) {
			return false
		}
		return true
	}
	return false
}

// argvMatchesString compares a candidate /argv JSON value
// against the required argv. Accepts []any (canonical
// shape produced by DecodeBytes) and []string for tests.
func argvMatchesString(candidate any, required []string) bool {
	switch v := candidate.(type) {
	case []string:
		return sliceEqual(v, required)
	case []any:
		if len(v) != len(required) {
			return false
		}
		for i, want := range required {
			got, ok := v[i].(string)
			if !ok || got != want {
				return false
			}
		}
		return true
	}
	return false
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// IsEnabledPolicyProfile reports whether name is a known,
// currently-enabled policy profile.
func IsEnabledPolicyProfile(name string) bool {
	profile, ok := policyProfiles[name]
	return ok && profile.Enabled
}

// EnabledPolicyProfileNames returns the sorted set of
// currently-enabled policy profile names.
func EnabledPolicyProfileNames() []string {
	out := make([]string, 0, len(policyProfiles))
	for name, profile := range policyProfiles {
		if profile.Enabled {
			out = append(out, name)
		}
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// ArgvIsPolicyProfileRequiredCheck is exposed for
// validator-internal callers. External callers should
// consume the typed-Plan API instead.
func ArgvIsPolicyProfileRequiredCheck(argv []string) []byte {
	out, err := json.Marshal(argv)
	if err != nil {
		return nil
	}
	return out
}
