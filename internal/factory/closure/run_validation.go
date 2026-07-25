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

func validateBaselineIdentity(ctx context.Context, git gitClient, root string, baseline Baseline) error {
	commit, err := resolveGitObject(ctx, git, root, baseline.CommitOID+"^{commit}")
	if err != nil {
		return fmt.Errorf("resolve baseline commit: %w", err)
	}
	tree, err := resolveGitObject(ctx, git, root, baseline.CommitOID+"^{tree}")
	if err != nil {
		return fmt.Errorf("resolve baseline tree: %w", err)
	}
	if commit != baseline.CommitOID || tree != baseline.TreeOID {
		return fmt.Errorf("baseline commit/tree binding mismatch")
	}
	return nil
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
