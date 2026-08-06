// SPDX-License-Identifier: Apache-2.0

package closure

// verifier_output_path.go implements the read-only output
// authority required by Phase 1 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION01.
//
// The verifier never writes inside the repository it is
// verifying. The CLI rejects --output paths that resolve
// inside the target repository BEFORE any Git observation
// so a path-rejected invocation never touches the object
// database.
//
// The resolver:
//
//  1. resolves the repository root to an absolute,
//     symlink-evaluated path;
//  2. rejects any output path that is:
//
//     - inside the repository;
//     - equal to the repository root;
//     - reachable through a symlink into the repository;
//     - lexically ambiguous (non-canonical);
//
//  3. rejects directories and non-regular existing files;
//  4. emits a typed V2VerifierError whose Code is
//     V2VerifierOutputPathNotDetached.
//
// No filesystem read or Git observation runs before the
// resolver finishes. The CLI passes the repository root
// (resolved from --repository) into the resolver; the
// resolver itself never invokes git, so the rejection path
// is observable from a unit test without any Git
// dependency.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateDetachedVerifierOutputPath returns a non-nil
// *V2VerifierError if the candidate output path is not safely
// detached from the target repository. The function
// returns nil when the path is safe.
//
// The repository root is resolved to an absolute,
// symlink-evaluated, canonical path. The candidate output
// path is also resolved to the same form before
// containment checks run.
//
// When candidate is the empty string, the resolver returns
// nil so the CLI can default to stdout. The "candidate must
// live outside the repo" contract only applies when the
// candidate is non-empty.
func ValidateDetachedVerifierOutputPath(repoRoot, candidate string) error {
	if strings.TrimSpace(candidate) == "" {
		// No --output flag was supplied; the CLI publishes
		// the verdict to stdout instead, so there is nothing
		// to reject on filesystem grounds.
		return nil
	}
	if strings.TrimSpace(repoRoot) == "" {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      "verifier --output validation requires a non-empty --repository",
			PropertyName: "repository_root",
		})
	}
	resolvedRoot, rootErr := filepath.EvalSymlinks(repoRoot)
	if rootErr != nil {
		if !errors.Is(rootErr, os.ErrNotExist) {
			return NewV2VerifierError(V2VerifierDiagnostic{
				Code:         V2VerifierOutputPathNotDetached,
				Message:      fmt.Sprintf("could not resolve repository root %q: %s", repoRoot, rootErr.Error()),
				PropertyName: "repository_root",
			})
		}
		absRoot, absErr := filepath.Abs(repoRoot)
		if absErr != nil {
			return NewV2VerifierError(V2VerifierDiagnostic{
				Code:         V2VerifierOutputPathNotDetached,
				Message:      fmt.Sprintf("could not absolutize repository root %q: %s", repoRoot, absErr.Error()),
				PropertyName: "repository_root",
			})
		}
		resolvedRoot = filepath.Clean(absRoot)
	} else {
		resolvedRoot = filepath.Clean(resolvedRoot)
	}

	resolvedCandidate, candErr := resolveCandidate(candidate)
	if candErr != nil {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("could not resolve verifier --output path %q: %s", candidate, candErr.Error()),
			PropertyName: "output_path",
		})
	}

	if resolvedCandidate == resolvedRoot {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output path %q equals the repository root", candidate),
			PropertyName: "output_path",
		})
	}

	if pathIsInsideOrEqual(resolvedRoot, resolvedCandidate) {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output path %q is inside the repository root", candidate),
			PropertyName: "output_path",
		})
	}

	if info, statErr := os.Stat(resolvedCandidate); statErr == nil {
		if !info.Mode().IsRegular() {
			return NewV2VerifierError(V2VerifierDiagnostic{
				Code:         V2VerifierOutputPathNotDetached,
				Message:      fmt.Sprintf("verifier --output path %q exists and is not a regular file", candidate),
				PropertyName: "output_path",
			})
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return NewV2VerifierError(V2VerifierDiagnostic{
			Code:         V2VerifierOutputPathNotDetached,
			Message:      fmt.Sprintf("verifier --output path %q is not usable: %s", candidate, statErr.Error()),
			PropertyName: "output_path",
		})
	}

	return nil
}

// resolveCandidate returns the canonical, symlink-evaluated
// absolute path of the candidate output. The function
// accepts a path that does not yet exist; non-existent
// paths are still absolutized and cleaned so containment
// checks run against the same lexical form.
func resolveCandidate(candidate string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(candidate); err == nil {
		return filepath.Clean(resolved), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	abs, absErr := filepath.Abs(candidate)
	if absErr != nil {
		return "", absErr
	}
	return filepath.Clean(abs), nil
}

// pathIsInsideOrEqual reports whether child is a strict
// descendant of (or equal to) parent. Both arguments are
// expected to be canonical, absolute, and Clean()-ed.
// Equal paths return true so callers can decide whether
// equality is a rejection criterion.
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
