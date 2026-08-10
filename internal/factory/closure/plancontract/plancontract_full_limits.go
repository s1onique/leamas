// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full_limits.go owns
// the canonical constants, regex patterns, and placeholder
// set for the Plan Contract v1 wire format.
//
// B2-R5 split the leaf into multiple files to keep each
// file under the LLM-friendly 400-line threshold. This
// file is the single source of truth for limits, patterns,
// and placeholder sets; the closure package references
// these via the public Max* / ContractVersionV1 exports
// (see plancontract.go) and the canonical ActIDPattern /
// ItemIDPattern / OIDPattern / EnvironmentNamePattern.
//
// ANY drift between this file and the closure package's
// mirror constants is a contract bug; the execution/
// evidence parity matrix in
// plan_contract_parity_b2r4_test.go would catch it.
package plancontract

import (
	"regexp"
	"strings"
)

// ----------------------------------------------------------------------------
// Bound constants (canonical limits)
// ----------------------------------------------------------------------------

const (
	// MaxChecks is the maximum number of checks a Plan
	// Contract v1 document may declare.
	MaxChecks = 4096

	// MaxArtifacts is the maximum number of artifacts a
	// Plan Contract v1 document may declare.
	MaxArtifacts = 4096

	// MaxArgvElements is the maximum number of argv
	// elements a single check may declare.
	MaxArgvElements = 16

	// MaxEnvironmentEntries is the maximum number of
	// environment entries a single check may declare.
	MaxEnvironmentEntries = 32

	// MaxCheckTimeoutSeconds is the inclusive upper bound
	// for the per-check timeout_seconds field.
	MaxCheckTimeoutSeconds = 600
)

// ----------------------------------------------------------------------------
// Patterns (canonical regex set)
// ----------------------------------------------------------------------------

var (
	ActIDPattern           = regexp.MustCompile(`^ACT-[A-Z0-9][A-Z0-9-]{2,199}$`)
	ItemIDPattern          = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	OIDPattern             = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
	lowercaseHex64Pattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	EnvironmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// ----------------------------------------------------------------------------
// Placeholder detection
// ----------------------------------------------------------------------------

var ExactClosurePlaceholders = map[string]struct{}{
	"TBD":            {},
	"TODO":           {},
	"UNKNOWN":        {},
	"RUNNING":        {},
	"TO BE RECORDED": {},
}

var EmbeddedClosurePlaceholders = []string{
	"(SEE GIT REV-PARSE)",
	"<COMMIT>",
	"<TREE>",
	"<HASH>",
}

// containsClosurePlaceholder mirrors the closure package's
// placeholder detection so the leaf and the closure runner
// reject the same set of values. The function is case-
// insensitive and whitespace-trimmed.
func containsClosurePlaceholder(value string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if _, found := ExactClosurePlaceholders[normalized]; found {
		return true
	}
	for _, marker := range EmbeddedClosurePlaceholders {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

// trimSpaceLower is a small helper that trims surrounding
// whitespace and lowercases the result. The execution
// mode validator uses it so the closed enum
// {"serial_fail_fast"} is matched case-insensitively and
// without leading/trailing whitespace.
func trimSpaceLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ContainsClosurePlaceholder is the canonical placeholder
// detector. The closure package aliases this function so
// callers do not duplicate the placeholder rule. A true
// return means the value carries a closure placeholder and
// the leaf MUST reject it on the wire-contract path.
//
// B2-R7 single-source rule: this is the only public entry
// point for the placeholder detector. The unexported
// containsClosurePlaceholder exists as the canonical
// implementation that the wire-contract validators use.
func ContainsClosurePlaceholder(value string) bool {
	return containsClosurePlaceholder(value)
}
