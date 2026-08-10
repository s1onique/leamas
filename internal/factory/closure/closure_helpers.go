// SPDX-License-Identifier: Apache-2.0

// Package closure - closure_helpers.go owns small,
// non-Plan-Contract helpers the closure package uses for
// manifest and runtime identity validation. These helpers
// are NOT duplicates of any plancontract semantic rule:
// they live outside the Plan Contract v1 wire contract
// and exist solely to support the closure runtime's own
// identity-binding work.
//
// B2-R7 single-authority rule: this file MUST NOT carry
// any Plan Contract v1 wire rule. Every such rule lives
// in the plancontract leaf; the closure package's
// adapters (plan_adapter.go, runner_authority.go,
// plan_patterns.go) reference the leaf by import. Any
// drift between this file and the leaf is a contract bug.
package closure

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateOID validates any OID field using the generic
// string-based dispatch. This is used by manifest and
// runtime identity validation where field identity is
// implicit from context (for example, a Git CLI call
// returning an OID the caller then sanity-checks). The
// function delegates the regex match to the canonical
// plancontract pattern; the placeholder check is also
// delegated so the closure package does not maintain its
// own placeholder set.
//
// B2-R7: this helper carries no Plan Contract rule. It
// exists only to give runtime callers a small, single-
// argument OID-shape probe. Plan Contract callers MUST
// go through plancontract.Validate* so the leaf is the
// single semantic authority.
func validateOID(field, value string) error {
	if containsClosurePlaceholder(value) {
		return fmt.Errorf("%s contains a closure placeholder", field)
	}
	if !oidPattern.MatchString(value) {
		return fmt.Errorf("%s is not a valid 40- or 64-character lowercase hex OID", field)
	}
	return nil
}

// validateRepositoryRelativePath validates a non-Plan-
// Contract repository-relative path. This is used by
// manifest path validation, not by Plan Contract
// artifact validation (which goes through
// plancontract.ValidateArtifactMap).
//
// B2-R7: the canonical Plan Contract path policy lives
// in plancontract (see plancontract_full_paths.go).
// This helper exists only for the closure-side manifest
// code paths that operate outside the Plan Contract
// wire contract. The shape rules (NUL, control chars,
// backslash, parent traversal, lexical cleanliness)
// mirror the canonical leaf's policy so the two systems
// cannot disagree on what a valid path looks like.
func validateRepositoryRelativePath(path string, allowDot bool) error {
	if path == "" || filepath.IsAbs(path) || strings.ContainsRune(path, 0) || containsClosurePlaceholder(path) {
		return fmt.Errorf("must be a non-empty repository-relative path")
	}
	clean := filepath.Clean(path)
	if clean == "." && allowDot {
		return nil
	}
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("must not escape the repository")
	}
	if clean != path {
		return fmt.Errorf("must be lexically clean")
	}
	return nil
}
