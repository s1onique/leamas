// SPDX-License-Identifier: Apache-2.0

package closure

// v2_path_authority.go implements the canonical path resolver
// and forbidden-root enforcement required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-PATH-AUTHORITY01.
//
// The resolver handles paths whose final component may not
// exist yet (the runner writes the manifest and creates evidence
// directories during execution). For such paths it:
//
//   1. makes the path absolute
//   2. finds the deepest existing ancestor
//   3. resolves that ancestor's symlinks via EvalSymlinks
//   4. appends the nonexistent suffix lexically
//   5. cleans the result
//
// It does NOT fall back to an unresolved full path merely
// because the leaf does not exist; a symlink-parent that
// points into the repository MUST still be detected.
//
// The forbidden roots enforced here are:
//
//   - the target repository worktree
//   - git rev-parse --git-dir (the repository's .git)
//   - git rev-parse --git-common-dir (an external common dir)
//   - the temporary subject worktree created by the runner
//
// All evidence / manifest / working-plan paths MUST be
// detached from every forbidden root.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// canonicalisePathDetached is the ACT 2 path resolver. It
// locates the deepest existing ancestor of p, resolves its
// symlinks, and appends the nonexistent suffix lexically so
// the caller cannot escape the resolver via a parent
// symlink.
//
// The function returns the canonical absolute path. Paths that
// fail to make absolute or whose deepest ancestor cannot be
// resolved surface as errors.
func canonicalisePathDetached(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	if strings.ContainsRune(p, 0) {
		return "", fmt.Errorf("path %q contains NUL byte", p)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("make absolute: %w", err)
	}
	// Walk upward until we find an existing directory or file.
	ancestor := abs
	suffix := ""
	for {
		if _, err := os.Lstat(ancestor); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("lstat %s: %w", ancestor, err)
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			// Reached filesystem root without finding an
			// existing ancestor; fall back to the absolute
			// path so the caller still gets a usable string.
			return filepath.Clean(abs), nil
		}
		// Build the suffix in reverse so we can join cleanly.
		suffix = filepath.Join(filepath.Base(ancestor), suffix)
		ancestor = parent
	}
	resolved, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks for %s: %w", ancestor, err)
	}
	if suffix == "" {
		return filepath.Clean(resolved), nil
	}
	return filepath.Clean(filepath.Join(resolved, suffix)), nil
}

// forbiddenRoots collects every root that the v2 runner MUST
// reject a candidate path for being contained in.
//
// roots is the list of canonical absolute roots; subjectWorktree
// may be empty when the runner is operating without a
// temporary worktree (unit tests).
type forbiddenRoots struct {
	targetRepoWorktree string
	gitDir             string
	gitCommonDir       string
	subjectWorktree    string
}

// resolveForbiddenRoots queries the supplied Git client for the
// canonical forbidden roots for the supplied repository.
//
// Each field is the canonical absolute path; an empty field
// means the root was not resolved (e.g. external common dir
// equals git dir) or the runner has no temporary subject
// worktree.
func resolveForbiddenRoots(ctx context.Context, git gitClient, repoRoot, subjectWorktree string) (forbiddenRoots, error) {
	fr := forbiddenRoots{}
	if repoRoot == "" {
		return fr, nil
	}
	canonRepo, err := canonicalisePathDetached(repoRoot)
	if err != nil {
		return fr, fmt.Errorf("canonicalise repository_root: %w", err)
	}
	fr.targetRepoWorktree = canonRepo
	if git == nil {
		return fr, nil
	}
	if out, err := runGitValue(ctx, git, canonRepo, "rev-parse", "--git-dir"); err == nil {
		if abs, absErr := filepath.Abs(filepath.Join(canonRepo, out)); absErr == nil {
			if c, err := canonicalisePathDetached(abs); err == nil {
				fr.gitDir = c
			}
		}
	}
	if out, err := runGitValue(ctx, git, canonRepo, "rev-parse", "--git-common-dir"); err == nil {
		abs, absErr := filepath.Abs(out)
		if absErr == nil {
			if c, err := canonicalisePathDetached(abs); err == nil {
				fr.gitCommonDir = c
			}
		}
		// Git returns a relative path when common-dir equals
		// git-dir; treat that case as equivalent so we don't
		// produce a duplicate root.
		if fr.gitCommonDir == "" || fr.gitCommonDir == fr.gitDir {
			fr.gitCommonDir = ""
		}
	}
	if subjectWorktree != "" {
		if c, err := canonicalisePathDetached(subjectWorktree); err == nil {
			fr.subjectWorktree = c
		}
	}
	return fr, nil
}

// pathIsUnder reports whether the canonical candidate path is
// the same as, or contained inside, the canonical root.
//
// Both arguments must already be canonical (no symlinks, no
// trailing separator, no relative components).
func pathIsUnder(candidate, root string) bool {
	if root == "" || candidate == "" {
		return false
	}
	if candidate == root {
		return true
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	if filepath.IsAbs(rel) {
		return false
	}
	return true
}

// ensureDetachedFromAny rejects the canonical target path when
// it is contained in any forbidden root. The diagnostic code
// is property-specific so the CLI can render the right
// message:
//
//   - evidence_directory -> V2CodeEvidencePathNotDetached
//   - manifest_output    -> V2CodeManifestPathNotDetached
//   - working_plan_assertion -> V2CodeWorkingPlanPathInvalid
//   - other properties    -> V2CodeInvalidPlanPath
func ensureDetachedFromAny(property, target string, roots forbiddenRoots) error {
	canonTarget, err := canonicalisePathDetached(target)
	if err != nil {
		code := V2CodeInvalidPlanPath
		switch property {
		case "evidence_directory":
			code = V2CodeEvidencePathNotDetached
		case "manifest_output":
			code = V2CodeManifestPathNotDetached
		case "working_plan_assertion":
			code = V2CodeWorkingPlanPathInvalid
		}
		return NewV2ErrorWith(code,
			fmt.Sprintf("%s canonicalise: %s", property, err.Error()),
			property, target)
	}
	if pathIsUnder(canonTarget, roots.targetRepoWorktree) {
		return newDetachedError(property, "repository_root", target, canonTarget)
	}
	if pathIsUnder(canonTarget, roots.gitDir) {
		return newDetachedError(property, "git_dir", target, canonTarget)
	}
	if pathIsUnder(canonTarget, roots.gitCommonDir) {
		return newDetachedError(property, "git_common_dir", target, canonTarget)
	}
	if pathIsUnder(canonTarget, roots.subjectWorktree) {
		return newDetachedError(property, "subject_worktree", target, canonTarget)
	}
	return nil
}

// newDetachedError returns the property-specific V2Error for a
// detached-path violation.
func newDetachedError(property, rootKind, target, canonTarget string) error {
	code := V2CodeInvalidPlanPath
	switch property {
	case "evidence_directory":
		code = V2CodeEvidencePathNotDetached
	case "manifest_output":
		code = V2CodeManifestPathNotDetached
	case "working_plan_assertion":
		code = V2CodeWorkingPlanPathInvalid
	}
	return NewV2ErrorWith(code,
		fmt.Sprintf("%s %q resolves inside %s (%s)",
			property, target, rootKind, canonTarget),
		property, canonTarget)
}

// enforceWorkingPlanAssertion reads the bytes at workingPath and
// compares them against the frozen bytes loaded from F:P.
//
// The comparison is byte-exact:
//   - canonical absolute path resolution
//   - exact-length comparison
//   - SHA-256 comparison
//   - byte-exact comparison
//
// A mismatch emits V2CodeWorkingPlanMismatch so the runner
// refuses to execute. An unreadable path emits
// V2CodeWorkingPlanPathInvalid.
func enforceWorkingPlanAssertion(workingPath string, frozen V2FrozenPlanBytes) error {
	canon, err := canonicalisePathDetached(workingPath)
	if err != nil {
		return NewV2ErrorWith(V2CodeWorkingPlanPathInvalid,
			fmt.Sprintf("working_plan_assertion canonicalise: %s", err.Error()),
			"working_plan_assertion", workingPath)
	}
	data, err := os.ReadFile(canon)
	if err != nil {
		return NewV2ErrorWith(V2CodeWorkingPlanPathInvalid,
			fmt.Sprintf("read working_plan_assertion: %s", err.Error()),
			"working_plan_assertion", canon)
	}
	return CompareToWorkingPlan(frozen, data)
}
