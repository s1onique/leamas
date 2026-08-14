package closure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func canonicalPlanPath(repositoryRoot, planPath, actID string) (string, error) {
	absolute, err := filepath.Abs(planPath)
	if err != nil {
		return "", fmt.Errorf("make plan path absolute: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve plan path: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("plan path must be a non-symlink regular file")
	}
	relative, err := filepath.Rel(repositoryRoot, resolved)
	if err != nil {
		return "", fmt.Errorf("relativize plan path: %w", err)
	}
	if relative == ".." || startsWithParent(relative) {
		return "", fmt.Errorf("plan must be inside the repository")
	}
	canonical := filepath.ToSlash(relative)
	expected := "docs/closure-plans/" + actID + ".json"
	if canonical != expected {
		return "", fmt.Errorf("plan path must be canonical %q", expected)
	}
	return canonical, nil
}

// validateBaselineIdentity is the v1 legacy wrapper that validates
// baseline commit/tree binding. The v2 runner uses ValidateBaselineGitObjects
// which provides more detailed error codes.
func validateBaselineIdentity(ctx context.Context, git gitClient, root string, baseline Baseline) error {
	return ValidateBaselineGitObjects(ctx, git, root, baseline)
}

// ValidateBaselineGitObjects verifies that the baseline commit and tree
// OIDs exist in the repository and that the tree is the committed tree
// of the baseline commit. This is the repository-bound authority stage
// for v2 runner execution.
//
// Returns typed errors:
//   - baseline_commit_not_found: baseline.commit_oid does not exist
//   - baseline_tree_not_found: baseline.tree_oid does not exist
//   - baseline_tree_mismatch: baseline.commit_oid^{tree} != baseline.tree_oid
func ValidateBaselineGitObjects(ctx context.Context, git gitClient, root string, baseline Baseline) error {
	// Step 1: Verify baseline.commit_oid exists as a commit
	_, err := resolveGitObject(ctx, git, root, baseline.CommitOID+"^{commit}")
	if err != nil {
		return &BaselineGitValidationError{
			Field:   "baseline.commit_oid",
			Code:    "baseline_commit_not_found",
			Message: fmt.Sprintf("baseline commit %s not found in repository", baseline.CommitOID),
			Cause:   err,
		}
	}

	// Step 2: Verify baseline.tree_oid exists as a tree
	_, err = resolveGitObject(ctx, git, root, baseline.TreeOID+"^{tree}")
	if err != nil {
		return &BaselineGitValidationError{
			Field:   "baseline.tree_oid",
			Code:    "baseline_tree_not_found",
			Message: fmt.Sprintf("baseline tree %s not found in repository", baseline.TreeOID),
			Cause:   err,
		}
	}

	// Step 3: Verify baseline.commit_oid^{tree} == baseline.tree_oid
	actualTree, err := resolveGitObject(ctx, git, root, baseline.CommitOID+"^{tree}")
	if err != nil {
		return &BaselineGitValidationError{
			Field:   "baseline.tree_oid",
			Code:    "baseline_tree_mismatch",
			Message: fmt.Sprintf("baseline commit %s^{tree} (%s) does not match declared tree %s", baseline.CommitOID, actualTree, baseline.TreeOID),
			Cause:   err,
		}
	}
	if actualTree != baseline.TreeOID {
		return &BaselineGitValidationError{
			Field:   "baseline.tree_oid",
			Code:    "baseline_tree_mismatch",
			Message: fmt.Sprintf("baseline commit %s^{tree} (%s) does not match declared tree %s", baseline.CommitOID, actualTree, baseline.TreeOID),
		}
	}
	return nil
}

// BaselineGitValidationError is a typed error for baseline Git object validation failures.
type BaselineGitValidationError struct {
	Field   string
	Code    string
	Message string
	Cause   error
}

func (e *BaselineGitValidationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *BaselineGitValidationError) Unwrap() error {
	return e.Cause
}

// evaluateRequiredPatchHygiene is the Closure Protocol v1 (legacy
// run.go path) entry point. Its range is the historical ACT scope
// plan.baseline..HEAD, which is appropriate for the legacy single
// manifest model but is NOT what the v2 orchestrator uses. See
// PolicyRangeDecision in run_v2_policy.go for the explicit v2
// distinction: v2 patch hygiene is F..S; v1 patch hygiene is
// plan.baseline..HEAD.
func evaluateRequiredPatchHygiene(ctx context.Context, git gitClient, root string, plan Plan) (PatchHygiene, []byte) {
	if !*plan.Policy.RequireDiffCheck {
		return PatchHygiene{Status: CheckStatusPass}, nil
	}
	return evaluatePatchHygiene(ctx, git, root, plan.Baseline.CommitOID, "HEAD")
}

func evaluateRequiredClosurePolicy(ctx context.Context, git gitClient, root string, plan Plan, subject string) (ClosurePolicyResult, []byte) {
	if !*plan.Policy.ForbidTrackedFullDigests {
		return ClosurePolicyResult{TrackedFullDigestStatus: CheckStatusPass}, nil
	}
	return evaluateTrackedDigestPolicy(ctx, git, root, plan.Baseline.CommitOID, subject)
}
