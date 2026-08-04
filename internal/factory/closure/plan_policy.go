package closure

import (
	"fmt"
	"strings"
)

// plan_policy.go contains the policy-related validation code extracted
// from plan.go to keep files under the LLM-friendly 400-line threshold.

// validatePlanPolicy validates the policy fields.
func validatePlanPolicy(policy PlanPolicy) error {
	missing := missingPlanPolicyFields(policy)
	if len(missing) > 0 {
		return &PlanPolicyRequiredError{Missing: missing}
	}
	if !*policy.RequireCleanBefore || !*policy.RequireCleanAfter {
		return errCleanWorktreeRequired()
	}
	return nil
}

// missingPlanPolicyFields returns the ordered list of /policy
// sibling names whose value is missing. The order is read from
// the descriptor's /policy.Required set (via PolicyFieldOrder)
// so the descriptor remains the single authority; the typed
// PlanPolicy struct's pointer fields are the value source.
func missingPlanPolicyFields(policy PlanPolicy) []string {
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

// PlanPolicyRequiredError is the typed diagnostic the policy
// validator emits when one or more policy siblings are absent. The
// error is JSON-marshallable so future CLI flags can render it
// directly. The Missing slice is the ordered list of sibling names
// that were absent.
type PlanPolicyRequiredError struct {
	Missing []string
}

// Error implements the error interface. The message format lists
// every missing sibling, separated by ", ", so consumers that print
// the error see all gaps at once.
func (e *PlanPolicyRequiredError) Error() string {
	if len(e.Missing) == 0 {
		return "policy fields missing"
	}
	return fmt.Sprintf("missing required policy field(s): %s", strings.Join(e.Missing, ", "))
}

// IsPlanPolicyRequiredError reports whether err (or any wrapped
// error) is a *PlanPolicyRequiredError. The helper lets callers
// route the diagnostic through a dedicated path without exposing
// the concrete error type.
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

// PlanDiagnostics implements planDiagnosticSource. It returns one
// diagnostic for each missing policy field, each with the exact
// path /policy/<field_name>. The diagnostics are deep-copied.
func (e *PlanPolicyRequiredError) PlanDiagnostics() []PlanValidationError {
	diags := make([]PlanValidationError, len(e.Missing))
	for i, name := range e.Missing {
		diags[i] = clonePlanValidationError(PlanValidationError{
			InstancePath: "/policy/" + name,
			SchemaPath:   "/policy/" + name,
			Code:         PlanCodeRequiredPropertyMissing,
			Keyword:      KeywordRequired,
			Message:      e.Error(),
			PropertyName: name,
		})
	}
	return diags
}
