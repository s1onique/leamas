// SPDX-License-Identifier: Apache-2.0

// Package reporoot provides canonical Git repository root resolution.
package reporoot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Errors for root resolution failures.
//
// SplitRepoPath propagates every canonicalization, walk-up, and
// filepath.Rel failure rather than collapsing them into an
// empty relPath. Each path-authority failure carries a distinct
// typed error so the caller can decide whether to retry, fall
// back, or surface a diagnostic; the resolver never substitutes
// an empty string for relPath when authority cannot be
// established.
var (
	ErrEmptyInput           = errors.New("root resolver: empty input")
	ErrNotAbsolute          = errors.New("root resolver: path is not absolute")
	ErrNotDirectory         = errors.New("root resolver: path is not a directory")
	ErrNotInRepository      = errors.New("root resolver: path is not within a Git repository")
	ErrGitRootFailed        = errors.New("root resolver: failed to discover Git root")
	ErrSymlinkFailed        = errors.New("root resolver: failed to resolve symlink")
	ErrUnreadableRoot       = errors.New("root resolver: root directory is not readable")
	ErrCanonicalizeFailed   = errors.New("root resolver: canonicalize absolute path failed")
	ErrCanonicalizeNotExist = errors.New("root resolver: canonicalize reported missing path")
	ErrWalkUpLstatFailed    = errors.New("root resolver: walk-up lstat failed")
	ErrWalkUpResolveFailed  = errors.New("root resolver: walk-up resolve failed")
	ErrRelComputationFailed = errors.New("root resolver: relative path computation failed")
)

// RootResolver resolves a canonical Git repository root from various input forms.
type RootResolver struct{}

// New creates a new RootResolver.
func New() *RootResolver {
	return &RootResolver{}
}

// Resolve discovers the canonical Git repository root from an input path.
// The input may be:
//   - Repository root
//   - Repository subdirectory
//   - Symlink to repository
//
// Returns the canonical absolute path or a typed error.
func (r *RootResolver) Resolve(input string) (string, error) {
	// Handle empty input
	if input == "" {
		return "", ErrEmptyInput
	}

	// Make absolute
	absPath, err := filepath.Abs(input)
	if err != nil {
		return "", ErrNotAbsolute
	}

	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", ErrSymlinkFailed
	}

	// Check if it's a directory
	info, err := os.Stat(realPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrEmptyInput
		}
		return "", ErrUnreadableRoot
	}
	if !info.IsDir() {
		// For files, get the directory
		realPath = filepath.Dir(realPath)
	}

	// Find Git root
	gitRoot, err := r.findGitRoot(realPath)
	if err != nil {
		return "", err
	}

	// Resolve symlink of Git root too
	gitRoot, err = filepath.EvalSymlinks(gitRoot)
	if err != nil {
		return "", ErrSymlinkFailed
	}

	// Clean the path
	gitRoot = filepath.Clean(gitRoot)

	// Verify it's readable
	if !r.isReadable(gitRoot) {
		return "", ErrUnreadableRoot
	}

	return gitRoot, nil
}

// findGitRoot searches upward for a .git directory.
func (r *RootResolver) findGitRoot(start string) (string, error) {
	dir := start
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", ErrNotInRepository
		}
		dir = parent
	}
}

// isReadable checks if a directory is readable.
func (r *RootResolver) isReadable(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && entries != nil
}

// SplitRepoPath splits an absolute path into (repoRoot, relativePath).
//
// Both root and the relative-path operand MUST be in the same
// canonical form for filepath.Rel to produce a sensible answer
// (e.g. "." for the repo root itself, "internal/factory" for a
// sub-path). Without canonicalization of absPath the function
// silently returns paths full of ".." segments when the caller
// supplies a path whose ancestors include a symlink (notably on
// macOS, where /var is a symlink to /private/var, so
// /var/folders/... and /private/var/folders/... identify the
// same directory).
//
// Failure semantics: SplitRepoPath is fail-closed. Every
// canonicalization, walk-up, and filepath.Rel failure is
// surfaced as a typed error so the caller can distinguish
// "absPath does not exist yet" (a legitimate lifecycle case for
// future output leaves) from "absPath cannot be resolved" (a
// hard authority failure). The function NEVER collapses a
// failure into an empty relPath.
//
// Order of operations: canonicalize absPath first (handling
// non-existence by walking upward to the largest existing
// prefix), then Resolve the canonical form, then compute the
// relative path. Doing canonicalization up-front means Resolve
// always sees an existing canonical path; the previously
// implicit call to Resolve(absPath) could fail because absPath
// did not yet exist, which would have masked the legitimate
// "future output leaf" lifecycle.
func (r *RootResolver) SplitRepoPath(absPath string) (repoRoot string, relPath string, err error) {
	if absPath == "" {
		return "", "", ErrEmptyInput
	}

	canonicalAbs, err := canonicalizeExistingPrefix(absPath)
	if err != nil {
		return "", "", err
	}

	// Resolve requires an existing path; if canonicalAbs carries
	// a non-existent leaf (the "future output leaf" lifecycle),
	// fall back to the canonical parent for the git-root lookup.
	// The relative path is still computed against canonicalAbs so
	// the returned relPath is in canonical coordinates.
	resolveTarget := canonicalAbs
	if _, lerr := os.Lstat(resolveTarget); lerr != nil {
		if !os.IsNotExist(lerr) {
			return "", "", fmt.Errorf("%w: %s: %s", ErrCanonicalizeFailed, resolveTarget, lerr.Error())
		}
		resolveTarget = filepath.Dir(canonicalAbs)
	}

	root, err := r.Resolve(resolveTarget)
	if err != nil {
		return "", "", err
	}

	rel, err := filepath.Rel(root, canonicalAbs)
	if err != nil {
		return "", "", fmt.Errorf("%w: root=%s abs=%s: %s", ErrRelComputationFailed, root, canonicalAbs, err.Error())
	}

	return root, rel, nil
}

// canonicalizeExistingPrefix returns the canonical form of the
// largest existing prefix of path with the missing suffix
// re-appended lexically.
//
// Failure semantics: canonicalizeExistingPrefix distinguishes
// "this component does not exist yet" (IsNotExist, legitimate
// walk-up target) from "this component cannot be inspected"
// (permission, I/O, symlink-loop). The latter is propagated as a
// typed error rather than silently re-routed through the walk-up
// branch, because a permission-denied or unreadable ancestor is
// an authority failure the caller must see.
//
// When the full path exists, this is identical to
// filepath.EvalSymlinks. When the path does not exist, this
// walks upward to the first existing prefix, resolves its
// canonical form, and re-appends the missing suffix lexically.
func canonicalizeExistingPrefix(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %s", ErrCanonicalizeFailed, path, err.Error())
	}
	tail := ""
	cursor := abs
	for {
		info, lerr := os.Lstat(cursor)
		if lerr == nil {
			_ = info
			canonicalCursor, eerr := filepath.EvalSymlinks(cursor)
			if eerr != nil {
				return "", fmt.Errorf("%w: %s: %s", ErrWalkUpResolveFailed, cursor, eerr.Error())
			}
			return filepath.Join(canonicalCursor, tail), nil
		}
		if !os.IsNotExist(lerr) {
			// Permission, I/O, symlink-loop, or other failure.
			// This is NOT a legitimate walk-up case; the
			// component exists but cannot be inspected, which
			// is exactly the authority failure the resolver is
			// meant to surface.
			return "", fmt.Errorf("%w: %s: %s", ErrWalkUpLstatFailed, cursor, lerr.Error())
		}
		parent := filepath.Dir(cursor)
		if parent == cursor {
			// Walk-up reached the filesystem root without ever
			// finding an existing prefix. The supplied path is
			// therefore outside any existing filesystem subtree
			// and cannot be canonicalized.
			return "", fmt.Errorf("%w: %s", ErrCanonicalizeNotExist, abs)
		}
		base := filepath.Base(cursor)
		if tail == "" {
			tail = base
		} else {
			tail = filepath.Join(base, tail)
		}
		cursor = parent
	}
}

// ImportPathFromRelPath converts a repository-relative path to an import path.
// E.g., "internal/factory/protectedverifier" → "github.com/s1onique/leamas/internal/factory/protectedverifier"
func (r *RootResolver) ImportPathFromRelPath(relPath string, modulePath string) string {
	relPath = filepath.ToSlash(relPath)
	if relPath == "." {
		return modulePath
	}
	relPath = strings.TrimPrefix(relPath, "./")
	return modulePath + "/" + relPath
}

// IsWithinRepo checks if a path is within the given repository root.
func (r *RootResolver) IsWithinRepo(path, repoRoot string) bool {
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(repoRoot)

	if cleanPath == cleanRoot {
		return true
	}

	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..")
}
