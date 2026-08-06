// SPDX-License-Identifier: Apache-2.0

package closure

// verifier_output_path.go implements the inventory-aware output
// path canonicalization required by Phase 2 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02B.
//
// Two public entry points drive the verifier CLI:
//
//   - ValidateDetachedVerifierOutputPath(repoRoot, candidate):
//     the legacy, repo-only check used by the duplicate-flag
//     parser. It rejects an --output path inside the
//     repository root BEFORE any Git observation, so a
//     path-rejected invocation never touches the object
//     database. CORRECTION02B does NOT retire this contract;
//     it is preserved as a fast, Git-free pre-flight probe.
//
//   - resolveDetachedDestination(repoRoot, candidate, inv):
//     the inventory-aware canonicalization. CORRECTION02B
//     routes the CLI's verifier output through
//     PrepareVerifierOutput, which uses this function plus
//     the per-repository worktree inventory to confirm that
//     the destination is detached from every main and linked
//     worktree root.
//
// The resolver never falls back from a successful
// EvalSymlinks to a lexical canonicalization: when the full
// path exists, the resolver MUST use the resolved form. When
// the destination itself does not exist, the resolver walks
// upward to the nearest existing ancestor; if that ancestor
// exists, its symlinks MUST resolve. Stub / nonexistent
// ancestors that cannot be EvalSymlinks-resolved are rejected
// — the resolver refuses to publish a destination whose
// canonical containment cannot be proven, because the
// alternative is a destination planted beneath a future
// worktree root.
//
// The resolver performs no output creation and no
// observation of the verifier's Git/object database. The CLI
// calls it BEFORE the orchestrator and BEFORE the publication
// authority, so a rejected path never reaches publication.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateDetachedVerifierOutputPath returns a non-nil
// *V2VerifierError if the candidate output path is not safely
// detached from the target repository root. The function
// returns nil when the path is safe or when the caller chose
// to omit --output entirely.
//
// The check is intentionally narrow: it only inspects the
// repository root because CORRECTION02B retains a
// fast, Git-free pre-flight. The full inventory-aware check
// lives in PrepareVerifierOutput and is applied AFTER the CLI
// has observed the worktree inventory.
func ValidateDetachedVerifierOutputPath(repoRoot, candidate string) error {
	if strings.TrimSpace(candidate) == "" {
		return nil
	}
	if strings.TrimSpace(repoRoot) == "" {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      "verifier --output validation requires a non-empty --repository",
			PropertyName: "repository_root",
		})
	}
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("could not absolutize repository root %q: %s", repoRoot, err.Error()),
			PropertyName: "repository_root",
		})
	}
	root = filepath.Clean(root)
	if resolved, rerr := filepath.EvalSymlinks(root); rerr == nil {
		root = filepath.Clean(resolved)
	} else if !errors.Is(rerr, os.ErrNotExist) {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("could not resolve repository root %q: %s", repoRoot, rerr.Error()),
			PropertyName: "repository_root",
		})
	}
	resolvedCandidate, err := resolveCandidateWithAncestor(candidate)
	if err != nil {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("could not resolve verifier --output path %q: %s", candidate, err.Error()),
			PropertyName: "output_path",
		})
	}
	if resolvedCandidate == root {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output path %q equals the repository root", candidate),
			PropertyName: "output_path",
		})
	}
	if pathIsInsideOrEqual(root, resolvedCandidate) {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output path %q is inside the repository root", candidate),
			PropertyName: "output_path",
		})
	}
	if err := rejectNonRegularExistingPath(resolvedCandidate, candidate); err != nil {
		return err
	}
	return nil
}

// resolveDetachedDestination is the inventory-aware
// canonicalization used by PrepareVerifierOutput. It accepts
// the candidate CLI output path, the supplied repositoryRoot
// for binding, and the complete worktree inventory (main +
// every linked worktree), returning the canonical absolute
// destination that subsequent publication can use to open
// the parent directory descriptor.
//
// The function:
//
//  1. absolutizes and cleans the candidate;
//  2. when the destination exists, evaluates its symlinks to
//     a canonical form; the canonical form MUST then escape
//     every worktree root;
//  3. when the destination does not exist, walks upward to
//     the nearest existing ancestor whose symlinks MUST
//     resolve, re-appends the unresolved suffix, then cleans
//     the joined result; the canonical form MUST then escape
//     every worktree root;
//  4. rejects existing directories, FIFO/socket/device
//     files, symlinks pointing into the inventory, lexically
//     unclean paths, and any path the resolver cannot
//     canonicalize; binds the supplied repositoryRoot to
//     the canonical main-worktree root in the inventory.
//
// The error type is always a *V2VerifierError carrying the
// canonical V2VerifierOutputPathNotDetached code so the CLI
// can route the rejection uniformly.
func resolveDetachedDestination(repoRoot, candidate string, inv RepositoryWorktreeInventory) (string, error) {
	if strings.TrimSpace(candidate) == "" {
		return "", NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      "verifier --output destination is empty",
			PropertyName: "output_path",
		})
	}
	if inv.RootsView() == nil || len(inv.RootsView()) == 0 {
		return "", NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierWorktreeInventoryUnavailable,
			Message:      "verifier --output resolution requires at least one canonical worktree root",
			PropertyName: "worktree_inventory",
		})
	}
	if err := bindRepositoryRoot(repoRoot, inv); err != nil {
		return "", err
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("could not absolutize candidate %q: %s", candidate, err.Error()),
			PropertyName: "output_path",
		})
	}
	if filepath.Clean(absCandidate) != absCandidate {
		return "", NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output path %q is lexically unclean", candidate),
			PropertyName: "output_path",
		})
	}
	canonical, err := canonicalizeDestinationWithAncestor(absCandidate)
	if err != nil {
		return "", NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("could not resolve verifier --output path %q: %s", candidate, err.Error()),
			PropertyName: "output_path",
		})
	}
	for _, root := range inv.RootsView() {
		if pathIsInsideOrEqual(root, canonical) {
			return "", NewV2VerifierError(V2VerifierDiagnostic{
				Code:         V2VerifierOutputPathNotDetached,
				Message:      fmt.Sprintf("verifier --output path %q resolves inside worktree %q", candidate, root),
				PropertyName: "output_path",
			})
		}
	}
	if err := rejectNonRegularExistingPath(canonical, candidate); err != nil {
		return "", err
	}
	return canonical, nil
}

// bindRepositoryRoot verifies that the supplied repositoryRoot
// canonicalizes to one of the canonical worktree roots in the
// inventory. This binds the --repository argument to the
// authoritative inventory so that a fake or partial inventory
// cannot authorize an output inside the target repository.
func bindRepositoryRoot(repoRoot string, inv RepositoryWorktreeInventory) error {
	if strings.TrimSpace(repoRoot) == "" {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      "verifier --output resolution requires a non-empty --repository",
			PropertyName: "repository_root",
		})
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("could not absolutize repository root %q: %s", repoRoot, err.Error()),
			PropertyName: "repository_root",
		})
	}
	if resolved, rerr := filepath.EvalSymlinks(absRoot); rerr == nil {
		absRoot = filepath.Clean(resolved)
	} else if errors.Is(rerr, os.ErrNotExist) {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierWorktreeInventoryUnavailable,
			Message:      fmt.Sprintf("repository root %q does not exist or is unresolvable: %s", repoRoot, rerr.Error()),
			PropertyName: "repository_root",
		})
	} else {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierWorktreeInventoryUnavailable,
			Message:      fmt.Sprintf("cannot resolve repository root %q: %s", repoRoot, rerr.Error()),
			PropertyName: "repository_root",
		})
	}
	if !inv.ContainsRoot(absRoot) {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierWorktreeInventoryUnavailable,
			Message:      fmt.Sprintf("repository root %q is not in the worktree inventory; refusing to authorize", absRoot),
			PropertyName: "repository_root",
		})
	}
	return nil
}

// canonicalizeDestinationWithAncestor returns the canonical
// absolute form of an output destination. When the
// destination exists the resolver uses EvalSymlinks to a
// resolved canonical form; when it does not exist the
// resolver walks upward to the nearest existing ancestor
// whose symlinks MUST resolve, then re-appends the unresolved
// suffix. The resolver never returns a lexical fallback for
// stub or missing paths: a destination that cannot be
// canonicalized has no provable containment and the resolver
// MUST fail closed.
func canonicalizeDestinationWithAncestor(absCandidate string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(absCandidate); err == nil {
		return filepath.Clean(resolved), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	// Destination does not exist; walk upward to the first
	// existing ancestor whose canonical form we can resolve.
	suffix := ""
	cur := absCandidate
	for {
		_, lerr := os.Lstat(cur)
		if lerr == nil {
			resolved, eerr := filepath.EvalSymlinks(cur)
			if eerr != nil {
				return "", eerr
			}
			joined := filepath.Join(resolved, suffix)
			return filepath.Clean(joined), nil
		}
		if !errors.Is(lerr, os.ErrNotExist) {
			return "", lerr
		}
		parent, base := filepath.Split(cur)
		parent = filepath.Clean(parent)
		if parent == cur {
			return "", fmt.Errorf("no existing ancestor for %q", absCandidate)
		}
		if suffix == "" {
			suffix = base
		} else {
			suffix = filepath.Join(base, suffix)
		}
		cur = parent
	}
}

// resolveCandidateWithAncestor returns the canonical, symlink-
// resolved absolute path of the candidate output. The function
// accepts paths that do not yet exist by walking upward to the
// nearest existing ancestor, evaluating symlinks on that
// ancestor, and re-attaching the unresolved suffix. The
// returned form matches the canonical worktree roots emitted
// by the inventory so the containment check is comparable.
func resolveCandidateWithAncestor(candidate string) (string, error) {
	abs, absErr := filepath.Abs(candidate)
	if absErr != nil {
		return "", absErr
	}
	return canonicalizeDestinationWithAncestor(abs)
}

// rejectNonRegularExistingPath returns a typed error when the
// supplied canonical path exists and is not a regular file.
// The candidate argument is the original CLI value so the
// reported message refers to the input the user typed. The
// resolver uses Lstat to detect symlinks at the leaf so a
// caller cannot disguise an inside-the-worktree target by
// passing a symlink that resolves to inside-the-worktree via
// an external sibling.
func rejectNonRegularExistingPath(canonical, candidate string) error {
	if canonical == "" {
		return nil
	}
	lInfo, lErr := os.Lstat(canonical)
	if lErr != nil {
		if errors.Is(lErr, os.ErrNotExist) {
			return nil
		}
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output path %q is not usable: %s", candidate, lErr.Error()),
			PropertyName: "output_path",
		})
	}
	if lInfo.Mode()&os.ModeSymlink != 0 {
		// Symlink at the leaf. Resolve it and re-classify so we
		// can detect a symlink whose target sits inside a
		// worktree, even when the leaf itself appears outside.
		target, terr := os.Readlink(canonical)
		if terr != nil {
			return NewV2VerifierError(V2VerifierDiagnostic{
				Code:         V2VerifierOutputPathNotDetached,
				Message:      fmt.Sprintf("verifier --output path %q is a symlink but cannot be read: %s", candidate, terr.Error()),
				PropertyName: "output_path",
			})
		}
		// The canonical form was already passed through
		// filepath.EvalSymlinks at the level of
		// canonicalizeDestinationWithAncestor, so an unbroken
		// symlink leaf should never reach here. Surface a typed
		// diagnostic to make the condition explicit.
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output path %q is an unresolved symlink to %q", candidate, target),
			PropertyName: "output_path",
		})
	}
	if lInfo.IsDir() {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output path %q exists and is a directory", candidate),
			PropertyName: "output_path",
		})
	}
	if !lInfo.Mode().IsRegular() {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output path %q exists and is not a regular file", candidate),
			PropertyName: "output_path",
		})
	}
	return nil
}

// pathIsInsideOrEqual reports whether child is a strict
// descendant of (or equal to) parent. Both arguments are
// expected to be canonical, absolute, and Clean()-ed. Equal
// paths return true so callers can decide whether equality
// is a rejection criterion.
func pathIsInsideOrEqual(parent, child string) bool {
	if parent == "" || child == "" {
		return false
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	return true
}
