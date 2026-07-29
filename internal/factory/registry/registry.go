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

// InvocationScope classifies whether a verifier participates in ordinary
// gate / factorize selection or is reachable only through typed command
// dispatch.
//
// The canonical registry model is the single source of truth for both
// scopes. Command-only definitions are absent from gate / factorize
// selection and from any registry.Run execution; they are resolved by
// the typed dispatcher only.
type InvocationScope string

const (
	// InvocationGate selects the verifier for ordinary gate / factorize
	// execution. It requires a non-nil Run function; the typed dispatcher
	// may also resolve it.
	InvocationGate InvocationScope = "gate"

	// InvocationCommandOnly excludes the verifier from gate / factorize
	// selection. A command-only definition must NOT supply a Run
	// function (the typed binder is the only execution path). This
	// prevents a fake no-op verifier from polluting the gate registry.
	InvocationCommandOnly InvocationScope = "command_only"
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
//
// Scope selects between ordinary gate / factorize selection (InvocationGate)
// and typed command dispatch only (InvocationCommandOnly). The Scope field
// is the single source of truth for both registration and selection; there
// is no parallel manual registry to maintain.
type Verifier struct {
	Name      string
	Run       func(root string) []checks.Finding
	Lane      VerifierLane
	Authority verifierauthority.ExecutionAuthority
	Scope     InvocationScope
	Execution ExecutionDefinition
	Cache     CacheSemantics
}

// Validate verifies the registry metadata for a verifier.
// Returns an error if:
//   - Authority is empty
//   - Authority is unknown
//   - Dupcode lane is marked local-safe (contradiction)
//   - Fast lane is marked ci_exact_checkout
//   - Scope is empty (must be explicit)
//   - Gate-scoped verifier has nil Run
//   - Command-only verifier has non-nil Run
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

	// Scope is required and must be explicit. There is no implicit default.
	if v.Scope == "" {
		return &ValidationError{
			Verifier: v.Name,
			Field:    "Scope",
			Reason:   "invocation scope is required (use InvocationGate or InvocationCommandOnly)",
		}
	}

	// Gate-scoped verifiers MUST have a Run function; command-only
	// verifiers MUST NOT (the typed binder is the only execution path).
	switch v.Scope {
	case InvocationGate:
		if v.Run == nil {
			return &ValidationError{
				Verifier: v.Name,
				Field:    "Run",
				Reason:   "gate-scoped verifier requires a Run function",
			}
		}
	case InvocationCommandOnly:
		if v.Run != nil {
			return &ValidationError{
				Verifier: v.Name,
				Field:    "Run",
				Reason:   "command-only verifier must not supply a Run function (typed binder is the only execution path)",
			}
		}
	default:
		return &ValidationError{
			Verifier: v.Name,
			Field:    "Scope",
			Reason:   "unknown invocation scope: " + string(v.Scope),
		}
	}

	// Check for dupcode/local_safe contradiction. A local-safe dupcode
	// definition is valid ONLY when it is the command-only mutation
	// identity (the dupcode-update-baseline entry). Any other
	// combination is rejected by construction.
	switch {
	case v.Lane == VerifierLaneDupcode && v.Authority == verifierauthority.AuthorityLocalSafe && v.Scope != InvocationCommandOnly:
		return &ValidationError{
			Verifier: v.Name,
			Field:    "Lane/Authority/Scope",
			Reason:   "dupcode local_safe verifier must be command_only",
		}
	case v.Lane == VerifierLaneDupcode && v.Authority == verifierauthority.AuthorityLocalSafe && v.Run != nil:
		return &ValidationError{
			Verifier: v.Name,
			Field:    "Lane/Authority/Run",
			Reason:   "dupcode command-only mutation identity must have nil Run (typed binder is the only execution path)",
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
