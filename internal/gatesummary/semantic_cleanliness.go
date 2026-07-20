package gatesummary

// validateCleanliness checks that when scope status is CLOSED, both cleanliness
// booleans must be true. Returns diagnostics for each violation.
func validateCleanliness(scopeStatus LifecycleStatus, cleanBefore, cleanAfter bool) []Diagnostic {
	if scopeStatus != LifecycleClosed {
		return nil
	}
	var diags []Diagnostic
	if !cleanBefore {
		diags = append(diags, Diagnostic{
			Code:     CodeScopeClosedDirtyWorktree,
			Path:     "/worktree_clean_before",
			Expected: "true",
			Observed: "false",
			Message:  "closed scope requires clean worktree before",
		})
	}
	if !cleanAfter {
		diags = append(diags, Diagnostic{
			Code:     CodeScopeClosedDirtyWorktree,
			Path:     "/worktree_clean_after",
			Expected: "true",
			Observed: "false",
			Message:  "closed scope requires clean worktree after",
		})
	}
	return diags
}
