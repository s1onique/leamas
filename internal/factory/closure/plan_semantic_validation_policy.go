package closure

import "fmt"

// plan_semantic_validation_policy.go contains typed error constructors
// for policy validation and PlanPolicyRequiredError diagnostics.

// errMissingPolicyField returns a typed error for a missing policy field.
func errMissingPolicyField(fieldName string) *PlanSemanticError {
	path := "/policy/" + fieldName
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordRequired,
		fmt.Sprintf("missing required policy field: %s", fieldName),
		fmt.Errorf("missing required policy field: %s", fieldName),
	)
}

// errCleanWorktreeRequired returns a typed error for clean worktree requirement.
func errCleanWorktreeRequired() *PlanSemanticError {
	return newSemanticError(
		"/policy",
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		"closure v1 requires clean worktree before and after",
		fmt.Errorf("closure v1 requires clean worktree before and after"),
	)
}

// PlanDiagnostics implements planDiagnosticSource for PlanPolicyRequiredError.
// It emits one diagnostic per missing policy field in canonical field order.
func (e *PlanPolicyRequiredError) PlanDiagnostics() []PlanValidationError {
	if len(e.Missing) == 0 {
		return []PlanValidationError{}
	}
	order := PolicyFieldOrder()
	diagnostics := make([]PlanValidationError, 0, len(e.Missing))
	missingSet := make(map[string]bool)
	for _, m := range e.Missing {
		missingSet[m] = true
	}
	// Emit diagnostics in canonical policy field order.
	for _, name := range order {
		if missingSet[name] {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath: "/policy/" + name,
				SchemaPath:   "/policy/" + name,
				Code:         PlanCodeSemanticConstraintFailed,
				Keyword:      KeywordRequired,
				Message:      fmt.Sprintf("missing required policy field: %s", name),
				PropertyName: name,
			})
		}
	}
	return diagnostics
}
