// SPDX-License-Identifier: Apache-2.0

// Package registry provides the canonical verifier registry for Factory verifiers.
package registry

import (
	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// VerifierLane classifies a verifier into a specific execution lane.
type VerifierLane string

const (
	VerifierLaneFast    VerifierLane = "fast"
	VerifierLaneDupcode VerifierLane = "dupcode"
)

// ExecutionKind classifies how a verifier executes.
type ExecutionKind string

const (
	ExecutionInProcess ExecutionKind = "in-process"
	ExecutionChild     ExecutionKind = "child-process"
)

// CacheRelevance classifies whether Go build cache affects the verifier.
type CacheRelevance string

const (
	CacheRelevant      CacheRelevance = "relevant"
	CacheNotRelevant   CacheRelevance = "not-relevant"
	CacheNotApplicable CacheRelevance = "not-applicable"
)

// TestResultCacheMode classifies whether test result cache applies.
type TestResultCacheMode string

const (
	CacheModeEnabled  TestResultCacheMode = "enabled"
	CacheModeDisabled TestResultCacheMode = "disabled"
	CacheModeNA       TestResultCacheMode = "not-applicable"
)

// ExecutionDefinition captures the authoritative execution metadata for a verifier.
type ExecutionDefinition struct {
	Kind             ExecutionKind
	ImplementationID string
	EnvVars          []string
}

// CacheSemantics captures the authoritative cache behavior for a verifier.
type CacheSemantics struct {
	GoBuildCache      CacheRelevance      `json:"go_build_cache"`
	GoTestResultCache TestResultCacheMode `json:"go_test_result_cache"`
}

// Verifier represents a Factory verifier with its authoritative metadata.
type Verifier struct {
	Name      string
	Run       func(root string) []checks.Finding
	Lane      VerifierLane
	Authority verifierauthority.ExecutionAuthority
	Execution ExecutionDefinition
	Cache     CacheSemantics
}

// Validate verifies the registry metadata for a verifier.
// Returns an error if:
// - Authority is empty
// - Authority is unknown
// - Dupcode lane is marked local-safe (contradiction)
// - Fast lane is marked remote-only
func (v *Verifier) Validate() error {
	// Check for empty authority
	if v.Authority == "" {
		return &ValidationError{
			Verifier: v.Name,
			Field:    "Authority",
			Reason:   "authority is required",
		}
	}

	// Check for unknown authority
	switch v.Authority {
	case verifierauthority.AuthorityLocalSafe,
		verifierauthority.AuthorityCIExactCheckout:
		// valid
	default:
		return &ValidationError{
			Verifier: v.Name,
			Field:    "Authority",
			Reason:   "unknown authority: " + string(v.Authority),
		}
	}

	// Check for dupcode/local_safe contradiction. The dupcode-update-baseline
	// entry is a distinct internal registry identity for baseline mutation
	// that runs only under local-safe authority; all other dupcode-lane
	// entries (notably the "dupcode" verification entry) remain CI-only.
	if v.Lane == VerifierLaneDupcode && v.Authority == verifierauthority.AuthorityLocalSafe && v.Name != "dupcode-update-baseline" {
		return &ValidationError{
			Verifier: v.Name,
			Field:    "Lane/Authority",
			Reason:   "dupcode verifier cannot be marked local_safe",
		}
	}

	// Check for fast/remote_only contradiction
	if v.Lane == VerifierLaneFast && v.Authority == verifierauthority.AuthorityCIExactCheckout {
		return &ValidationError{
			Verifier: v.Name,
			Field:    "Lane/Authority",
			Reason:   "fast verifier cannot be marked ci_exact_checkout",
		}
	}

	return nil
}

// ValidationError represents a registry validation failure.
type ValidationError struct {
	Verifier string
	Field    string
	Reason   string
}

func (e *ValidationError) Error() string {
	return "verifier " + e.Verifier + ": " + e.Field + " " + e.Reason
}
