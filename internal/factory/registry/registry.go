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
//
// Operations is the canonical list of operations this verifier is
// approved to handle. A verifier that lacks an entry here cannot be
// invoked through the dispatcher or profile paths. Every canonical
// definition declares exactly the operations it supports; ordinary gate
// verifiers accept only verify, while the reserved mutation identity
// accepts only update_baseline.
type Verifier struct {
	Name       string
	Run        func(root string) []checks.Finding
	Lane       VerifierLane
	Authority  verifierauthority.ExecutionAuthority
	Scope      InvocationScope
	Execution  ExecutionDefinition
	Cache      CacheSemantics
	Operations []verifierauthority.VerifierOperation
}

// reservedMutationIdentityName is the single canonical name for the
// command-only mutation identity. It is reserved by the registry: only
// an exact-match tuple may use this name, and only an exact-match
// tuple may be named this way.
const reservedMutationIdentityName = "dupcode-update-baseline"

// AllowsOperation reports whether the verifier's declared operation
// list contains the supplied operation. It is the single dispatcher
// and profile query for operation compatibility.
//
// Calling AllowsOperation with an unknown or malformed operation is
// always false; callers must still invoke
// verifierauthority.ValidateOperation to surface the canonical
// reason code.
func (v Verifier) AllowsOperation(operation verifierauthority.VerifierOperation) bool {
	for _, op := range v.Operations {
		if op == operation {
			return true
		}
	}
	return false
}

// validateOperations checks every entry of the verifier's declared
// operation list against verifierauthority.ValidateOperation. The
// helper rejects empty lists, empty strings, unknown operations,
// whitespace/case variants, and duplicates. It reports the registry
// field as "Operations" regardless of the underlying reason.
func validateOperations(verifierName string, operations []verifierauthority.VerifierOperation) error {
	if len(operations) == 0 {
		return &ValidationError{
			Verifier: verifierName,
			Field:    "Operations",
			Reason:   "verifier must declare at least one allowed operation",
		}
	}
	seen := make(map[verifierauthority.VerifierOperation]bool, len(operations))
	for _, op := range operations {
		if err := verifierauthority.ValidateOperation(op); err != nil {
			return &ValidationError{
				Verifier: verifierName,
				Field:    "Operations",
				Reason:   err.Error(),
			}
		}
		if seen[op] {
			return &ValidationError{
				Verifier: verifierName,
				Field:    "Operations",
				Reason:   "duplicate operation in allowed list: " + string(op),
			}
		}
		seen[op] = true
	}
	return nil
}

// isCanonicalDupcodeUpdateDefinition reports whether v equals the
// exact reserved mutation tuple:
//
//	Name       = reservedMutationIdentityName
//	Lane       = dupcode
//	Authority  = local_safe
//	Scope      = command_only
//	Run        = nil
//	Operations = [update_baseline]
//
// The check is biconditional: a definition matching this tuple must
// carry the canonical name, and a definition with the canonical name
// must match this tuple.
func isCanonicalDupcodeUpdateDefinition(v Verifier) bool {
	if len(v.Operations) != 1 {
		return false
	}
	if v.Operations[0] != verifierauthority.OperationUpdateBaseline {
		return false
	}
	return v.Name == reservedMutationIdentityName &&
		v.Lane == VerifierLaneDupcode &&
		v.Authority == verifierauthority.AuthorityLocalSafe &&
		v.Scope == InvocationCommandOnly &&
		v.Run == nil
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
//   - Operations is empty, contains unknown values, duplicates, or
//     whitespace / case variants
//   - The reserved mutation identity name is used without the exact
//     tuple, or a dupcode/local_safe command-only tuple is named
//     anything other than the reserved name
//   - A non-canonical verifier declares update_baseline or any
//     operation list other than exactly [verify]
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

	// Operations list is required and must be non-empty, free of
	// duplicates, and every entry must be a canonical verifier
	// operation. validateOperations reports the registry field as
	// "Operations" so callers see a stable diagnostic.
	if err := validateOperations(v.Name, v.Operations); err != nil {
		return err
	}

	// Reserved canonical mutation identity. The exact tuple is:
	//   Name       = reservedMutationIdentityName
	//   Lane       = dupcode
	//   Authority  = local_safe
	//   Scope      = command_only
	//   Run        = nil
	//   Operations = [update_baseline]
	// The biconditional check is delegated to
	// isCanonicalDupcodeUpdateDefinition so the reserved name and the
	// reserved tuple are tested by a single helper.
	if v.Name == reservedMutationIdentityName && !isCanonicalDupcodeUpdateDefinition(*v) {
		return &ValidationError{
			Verifier: v.Name,
			Field:    "Reserved",
			Reason:   "name \"dupcode-update-baseline\" is reserved and must match the canonical mutation tuple exactly",
		}
	}
	if !isCanonicalDupcodeUpdateDefinition(*v) && v.Lane == VerifierLaneDupcode && v.Authority == verifierauthority.AuthorityLocalSafe && v.Scope == InvocationCommandOnly {
		return &ValidationError{
			Verifier: v.Name,
			Field:    "Name",
			Reason:   "reserved dupcode local_safe command-only tuple must use the canonical name \"dupcode-update-baseline\"",
		}
	}
	// Only the reserved mutation identity may declare update_baseline.
	// Every other verifier is restricted to verify only.
	if v.Name != reservedMutationIdentityName {
		for _, op := range v.Operations {
			if op == verifierauthority.OperationUpdateBaseline {
				return &ValidationError{
					Verifier: v.Name,
					Field:    "Operations",
					Reason:   "update_baseline is reserved for the canonical mutation identity",
				}
			}
		}
		// Ordinary verifiers (gate-scoped or otherwise non-canonical)
		// must declare exactly [verify].
		if len(v.Operations) != 1 || v.Operations[0] != verifierauthority.OperationVerify {
			return &ValidationError{
				Verifier: v.Name,
				Field:    "Operations",
				Reason:   "ordinary verifiers must declare exactly [verify]",
			}
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
