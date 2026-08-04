// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"fmt"
)

// PolicyProfileError represents errors in policy profile validation.
type PolicyProfileError struct {
	ProfileName string // The policy_profile value that caused the error
	CheckID     string // For missing_check errors, the ID of the missing check
	Kind        PolicyProfileErrorKind
	Cause       error
}

// PolicyProfileErrorKind categorizes the specific validation failure.
type PolicyProfileErrorKind int

const (
	// PolicyProfileErrorUnknown indicates the profile name is not in the known profiles map.
	PolicyProfileErrorUnknown PolicyProfileErrorKind = iota
	// PolicyProfileErrorUnimplemented indicates the profile exists but is not yet enabled.
	PolicyProfileErrorUnimplemented
	// PolicyProfileErrorMissingCheck indicates a required check for the profile is absent or mismatched.
	PolicyProfileErrorMissingCheck
)

// Error implements the error interface.
func (e *PolicyProfileError) Error() string {
	switch e.Kind {
	case PolicyProfileErrorUnknown:
		return fmt.Sprintf("policy_profile %q is unknown", e.ProfileName)
	case PolicyProfileErrorUnimplemented:
		return fmt.Sprintf("policy_profile %q is not yet implemented for this repository", e.ProfileName)
	case PolicyProfileErrorMissingCheck:
		return fmt.Sprintf("plan does not satisfy policy profile %q: missing or non-matching check %q", e.ProfileName, e.CheckID)
	default:
		return "unknown policy profile error"
	}
}

// Unwrap returns the underlying cause for errors.Is/errors.As.
func (e *PolicyProfileError) Unwrap() error {
	return e.Cause
}

// PlanDiagnostics implements planDiagnosticSource. It returns a diagnostic
// with exact identity: InstancePath or PropertyName set appropriately.
func (e *PolicyProfileError) PlanDiagnostics() []PlanValidationError {
	var diag PlanValidationError

	switch e.Kind {
	case PolicyProfileErrorUnknown, PolicyProfileErrorUnimplemented:
		// These are about the policy_profile value selection - use InstancePath
		diag = PlanValidationError{
			InstancePath:  "/policy_profile",
			SchemaPath:    "/policy_profile",
			Code:          PlanCodeSemanticConstraintFailed,
			Keyword:       KeywordEnum,
			Message:       e.Error(),
			RejectedValue: e.ProfileName,
		}
	case PolicyProfileErrorMissingCheck:
		// Missing check is about the checks array not containing the required check
		diag = PlanValidationError{
			InstancePath: "/checks",
			SchemaPath:   "/checks",
			Code:         PlanCodeSemanticConstraintFailed,
			Keyword:      KeywordMinItems,
			Message:      e.Error(),
			PropertyName: e.CheckID,
		}
	}

	return []PlanValidationError{clonePlanValidationError(diag)}
}
