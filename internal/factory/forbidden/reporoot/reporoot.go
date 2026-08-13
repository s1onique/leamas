// SPDX-License-Identifier: Apache-2.0

// Package reporoot provides canonical Git repository root resolution.
package reporoot

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Errors for root resolution failures.
var (
	ErrEmptyInput      = errors.New("root resolver: empty input")
	ErrNotAbsolute     = errors.New("root resolver: path is not absolute")
	ErrNotDirectory    = errors.New("root resolver: path is not a directory")
	ErrNotInRepository = errors.New("root resolver: path is not within a Git repository")
	ErrGitRootFailed   = errors.New("root resolver: failed to discover Git root")
	ErrSymlinkFailed   = errors.New("root resolver: failed to resolve symlink")
	ErrUnreadableRoot  = errors.New("root resolver: root directory is not readable")
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
// The function canonicalizes absPath through the same
// EvalSymlinks-based path the production root uses; this is the
// matching authority for canonical-path comparisons.
func (r *RootResolver) SplitRepoPath(absPath string) (repoRoot string, relPath string, err error) {
	root, err := r.Resolve(absPath)
	if err != nil {
		return "", "", err
	}

	canonicalAbs, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// absPath does not exist on disk. The repo root
			// exists (Resolve succeeded), so absPath is below
			// root and never materialized. Walk upward to the
			// largest existing prefix and re-append the missing
			// suffix so the relative path is still meaningful.
			canonicalAbs, err = canonicalizeExistingPrefix(absPath)
			if err != nil {
				return root, "", nil
			}
		} else {
			return root, "", nil
		}
	}

	rel, err := filepath.Rel(root, canonicalAbs)
	if err != nil {
		return root, "", nil
	}

	return root, rel, nil
}

// canonicalizeExistingPrefix returns the canonical form of the
// largest existing prefix of path with the missing suffix
// re-appended lexically. When the full path exists, this is
// identical to filepath.EvalSymlinks. When the path does not
// exist, this avoids the resolver's reflexive refusal of the
// nil-exact identity case in SplitRepoPath (the repo root still
// exists; only the leaf is missing).
func canonicalizeExistingPrefix(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	tail := ""
	cursor := abs
	for {
		if _, err := os.Lstat(cursor); err == nil {
			canonicalCursor, err := filepath.EvalSymlinks(cursor)
			if err != nil {
				return "", err
			}
			return filepath.Join(canonicalCursor, tail), nil
		}
		parent := filepath.Dir(cursor)
		if parent == cursor {
			return abs, nil
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
