// Package claim provides typed domain models for claims and evidence
// in Leamas verification witness artifacts.
package claim

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

// Typed identifiers.

type ClaimID string
type EvidenceID string

// Errors for ID validation.

var (
	ErrEmptyID             = errors.New("ID must be non-empty")
	ErrIDTooLong           = errors.New("ID must be at most 128 characters")
	ErrIDMissingSuffix     = errors.New("ID must include a suffix after prefix")
	ErrIDNotLocal          = errors.New("ID must be a local filename (no path separators)")
	ErrIDTraversal         = errors.New("ID must not contain traversal components")
	ErrIDReserved          = errors.New("ID must not be \".\" or \"..\"")
	ErrIDInvalidChar       = errors.New("ID must contain only alphanumeric, dot, underscore, or hyphen characters")
	ErrClaimIDNoPrefix     = errors.New("claim ID must start with \"claim-\"")
	ErrEvidenceIDNoPrefix  = errors.New("evidence ID must start with \"evidence-\"")
	ErrInvalidRelativePath = errors.New("relative path must be local and must not escape bundle root")
	ErrAbsolutePath        = errors.New("relative path must not be absolute")
	ErrEmptyRelativePath   = errors.New("relative path must not be empty")
	ErrTraversalInPath     = errors.New("relative path must not contain traversal components")
)

// ID validation regex.
var idRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// ValidateClaimID checks that a claim ID is safe to use as a filename.
// Claim IDs must be lexical and platform-safe.
func ValidateClaimID(id ClaimID) error {
	return validateID(string(id), "claim-", ErrClaimIDNoPrefix)
}

// ValidateEvidenceID checks that an evidence ID is safe to use as a filename.
// Evidence IDs must be lexical and platform-safe.
func ValidateEvidenceID(id EvidenceID) error {
	return validateID(string(id), "evidence-", ErrEvidenceIDNoPrefix)
}

// validateID performs common ID validation.
func validateID(s, prefix string, prefixErr error) error {
	// Non-empty check
	if s == "" {
		return ErrEmptyID
	}

	// Prefix check
	if !strings.HasPrefix(s, prefix) {
		return prefixErr
	}

	// Must have at least one character after prefix
	if len(s) == len(prefix) {
		return ErrIDMissingSuffix
	}

	// Length check (max 128 chars)
	if len(s) > 128 {
		return ErrIDTooLong
	}

	// Reserved names
	if s == "." || s == ".." {
		return ErrIDReserved
	}

	// Local name check (no path separators)
	if strings.Contains(s, "/") || strings.Contains(s, string(filepath.Separator)) {
		return ErrIDNotLocal
	}

	// Traversal check
	if strings.Contains(s, "..") {
		return ErrIDTraversal
	}

	// Character set check
	if !idRegex.MatchString(s) {
		return ErrIDInvalidChar
	}

	// Platform IsLocal check
	if !filepath.IsLocal(s) {
		return ErrIDNotLocal
	}

	return nil
}

// ValidateRelativePath checks that a relative path is safe.
// Empty paths are allowed. Relative paths must not escape bundle root.
func ValidateRelativePath(path string) error {
	// Empty relative path is allowed
	if path == "" {
		return nil
	}

	// Absolute path check
	if filepath.IsAbs(path) {
		return ErrAbsolutePath
	}

	// Traversal check
	if strings.Contains(path, "..") {
		return ErrTraversalInPath
	}

	// Check that path is local
	if !filepath.IsLocal(path) {
		return ErrInvalidRelativePath
	}

	// Check for backslash separators (not portable)
	if strings.Contains(path, "\\") {
		return ErrInvalidRelativePath
	}

	return nil
}
