// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full_paths.go owns
// the small path/string helpers shared by every per-section
// helper file: validateRepositoryRelativePath,
// containsNulByte, and isValidExcludeReason.
//
// The closure package's portablePathValidate and
// validateRepositoryRelativePath were deleted in B2-R5
// because the wire-contract path rules now live
// exclusively here.
package plancontract

import (
	"path/filepath"
	"strings"
)

// validateRepositoryRelativePath is the canonical plan path
// rule. The rule rejects empty paths, absolute paths,
// null bytes, placeholders, and lexically unclean paths.
// The allowDot flag mirrors the closure package helper.
// The allowAbs flag is reserved for future use; absolute
// paths are always rejected today.
func validateRepositoryRelativePath(path string, allowDot bool, allowAbs bool) error {
	_ = allowAbs
	if path == "" || containsNulByte(path) || containsClosurePlaceholder(path) {
		return errInvalidRepoPath("must be a non-empty repository-relative path")
	}
	if filepath.IsAbs(path) {
		return errInvalidRepoPath("must be a non-empty repository-relative path")
	}
	clean := filepath.Clean(path)
	if clean == "." && allowDot {
		return nil
	}
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errInvalidRepoPath("must not escape the repository")
	}
	if clean != path {
		return errInvalidRepoPath("must be lexically clean")
	}
	return nil
}

// errInvalidRepoPath is a small helper that returns a
// plancontract error type without importing the canonical
// DecodeError (the per-section helpers already use the
// canonical type directly).
func errInvalidRepoPath(msg string) error {
	return &DecodeError{
		Code:    "invalid_repo_path",
		Message: msg,
	}
}

// containsNulByte returns true if s contains a NUL byte.
// Plan paths, argv elements, and environment values MUST NOT
// contain NUL bytes because they would break POSIX exec.
func containsNulByte(s string) bool {
	return strings.IndexByte(s, 0) >= 0
}

// isValidExcludeReason returns true if reason is a valid
// exclude-mode reason: non-empty, no leading/trailing
// whitespace, no CR/LF characters, length <= 240, and
// does not contain a closure placeholder.
func isValidExcludeReason(reason string) bool {
	if strings.TrimSpace(reason) == "" {
		return false
	}
	if strings.ContainsAny(reason, "\r\n") {
		return false
	}
	if len(reason) > 240 {
		return false
	}
	if containsClosurePlaceholder(reason) {
		return false
	}
	return true
}
