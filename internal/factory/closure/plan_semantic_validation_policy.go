package closure

import "fmt"

// plan_semantic_validation_policy.go contains typed error constructors
// for policy validation.

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
