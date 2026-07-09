// Package runbundle provides local run bundle creation and validation for
// Leamas verification witness evidence.
package runbundle

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

// Errors for path validation.
var (
	ErrEmptyRoot        = errors.New("root directory must be non-empty")
	ErrEmptyRunID       = errors.New("run ID must be non-empty")
	ErrRunIDTooLong     = errors.New("run ID must be at most 128 characters")
	ErrRunIDNotLocal    = errors.New("run ID must be a local filename (no path separators)")
	ErrRunIDTraversal   = errors.New("run ID must not contain traversal components")
	ErrRunIDAbsolute    = errors.New("run ID must not be an absolute path")
	ErrRunIDReserved    = errors.New("run ID must not be \".\" or \"..\"")
	ErrRunIDInvalidChar = errors.New("run ID must contain only alphanumeric, dot, underscore, or hyphen characters")
	ErrRunIDNoPrefix    = errors.New("run ID must start with \"run-\"")
)

// runIDRegex validates safe run ID characters.
var runIDRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// ValidateRunID checks that the run ID is safe to use as a directory name.
// Run IDs must be lexical and platform-safe.
func ValidateRunID(id RunID) error {
	s := string(id)

	// Non-empty check
	if s == "" {
		return ErrEmptyRunID
	}

	// Length check (max 128 chars)
	if len(s) > 128 {
		return ErrRunIDTooLong
	}

	// Prefix check
	if !strings.HasPrefix(s, "run-") {
		return ErrRunIDNoPrefix
	}

	// Reserved names
	if s == "." || s == ".." {
		return ErrRunIDReserved
	}

	// Absolute path check
	if filepath.IsAbs(s) {
		return ErrRunIDAbsolute
	}

	// Local name check (no path separators)
	if strings.Contains(s, "/") || strings.Contains(s, string(filepath.Separator)) {
		return ErrRunIDNotLocal
	}

	// Traversal check
	if strings.Contains(s, "..") {
		return ErrRunIDTraversal
	}

	// Character set check
	if !runIDRegex.MatchString(s) {
		return ErrRunIDInvalidChar
	}

	// Platform IsLocal check (covers additional edge cases)
	if !filepath.IsLocal(s) {
		return ErrRunIDNotLocal
	}

	return nil
}

// BundlePath returns the full path for a run bundle given a root and run ID.
func BundlePath(root string, id RunID) (string, error) {
	if root == "" {
		return "", ErrEmptyRoot
	}
	if err := ValidateRunID(id); err != nil {
		return "", err
	}

	bundlePath := filepath.Join(root, string(id))

	// Lexical containment check: ensure the bundle path is under root.
	// Clean both paths for comparison.
	cleanRoot := filepath.Clean(root)
	cleanBundle := filepath.Clean(bundlePath)

	// Check that bundle path starts with root path prefix.
	rel, err := filepath.Rel(cleanRoot, cleanBundle)
	if err != nil {
		return "", err
	}

	// If the relative path starts with "..", the bundle is outside root.
	if strings.HasPrefix(rel, "..") {
		return "", ErrRunIDTraversal
	}

	return bundlePath, nil
}
