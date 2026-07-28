// SPDX-License-Identifier: Apache-2.0

// Package verifierdispatch provides the central production dispatcher for verifier execution.
//
// The dispatcher is the single point of authority enforcement for all Factory verifier
// execution. It:
//   - Resolves verifier metadata from the canonical registry
//   - Validates execution authority before any expensive initialization
//   - Invokes the runner factory only after authority passes
//   - Propagates the real verifier result without changing its semantics
//
// Key invariants:
//   - Authority is validated before RunnerFactory is invoked
//   - No production entry point directly starts dupcode analysis
//   - Git observations use the bounded execution.RunGit gateway
package verifierdispatch

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// Request represents a verifier execution request.
type Request struct {
	VerifierID string
	Operation  verifierauthority.VerifierOperation
	Root       string
}

// RunnerFactory creates a verifier runner on demand.
// The factory is invoked only after authority validation passes.
type RunnerFactory func() func(root string) []checks.Finding

// Result represents the outcome of dispatcher processing.
type Result struct {
	Findings []checks.Finding
	Error    error
}

// ErrVerifierNotFound is returned when a verifier ID is not in the registry.
type ErrVerifierNotFound struct {
	VerifierID string
}

func (e *ErrVerifierNotFound) Error() string {
	return fmt.Sprintf("verifier not found: %s", e.VerifierID)
}

// ContextObserver observes execution context when required by authority.
// For local_safe verifiers, the observer is never called.
// For ci_exact_checkout verifiers, the observer is called exactly once
// before authority validation.
type ContextObserver interface {
	Observe(ctx context.Context, root string) verifierauthority.ExecutionContext
}

// DefaultContextObserver is the production context observer using DetectExecutionContext.
type DefaultContextObserver struct{}

func (d *DefaultContextObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.DetectExecutionContext(ctx, root)
}

// Dispatcher is the central production dispatcher for verifier execution.
type Dispatcher struct {
	verifiers []registry.Verifier
}

// NewDispatcher creates a new dispatcher with the canonical verifier registry.
// The registry is validated for structural correctness:
//   - No empty verifier IDs
//   - No duplicate verifier IDs
//   - No empty authority
//   - All authorities must be known
//   - No dupcode lane with local_safe
//   - No fast lane with ci_exact_checkout
//
// The registry is copied defensively; caller may modify original after construction.
func NewDispatcher(verifiers []registry.Verifier) (*Dispatcher, error) {
	if len(verifiers) == 0 {
		return nil, errors.New("verifiers slice is empty")
	}

	// Copy to prevent caller mutation
	verifiers = slices.Clone(verifiers)

	seen := make(map[string]bool)

	for i, v := range verifiers {
		// Check for empty verifier ID
		if v.Name == "" {
			return nil, &RegistryValidationError{
				Verifier: fmt.Sprintf("[%d]", i),
				Field:    "Name",
				Reason:   "verifier ID cannot be empty",
			}
		}

		// Check for duplicate verifier IDs
		if seen[v.Name] {
			return nil, &RegistryValidationError{
				Verifier: v.Name,
				Field:    "Name",
				Reason:   "duplicate verifier ID",
			}
		}
		seen[v.Name] = true

		// Delegate to registry's own validation for authority and lane compatibility
		if err := v.Validate(); err != nil {
			var validationErr *registry.ValidationError
			if errors.As(err, &validationErr) {
				return nil, &RegistryValidationError{
					Verifier: v.Name,
					Field:    validationErr.Field,
					Reason:   validationErr.Reason,
				}
			}
			return nil, &RegistryValidationError{
				Verifier: v.Name,
				Field:    "Authority",
				Reason:   err.Error(),
			}
		}
	}

	return &Dispatcher{verifiers: verifiers}, nil
}

// RegistryValidationError represents a dispatcher construction validation failure.
type RegistryValidationError struct {
	Verifier string
	Field    string
	Reason   string
}

func (e *RegistryValidationError) Error() string {
	return fmt.Sprintf("verifier %s: %s %s", e.Verifier, e.Field, e.Reason)
}

// Dispatch routes a verifier execution request through authority validation.
// It returns the verifier's findings if authority permits, or a denial finding
// if authority is denied. The RunnerFactory is never invoked unless authority
// validation passes.
//
// For local_safe verifiers, the ContextObserver is NOT called.
// For ci_exact_checkout verifiers, the ContextObserver is called exactly once.
func (d *Dispatcher) Dispatch(ctx context.Context, request Request, observer ContextObserver, factory RunnerFactory) Result {
	// Step 1: Resolve verifier metadata from registry
	v := d.resolveVerifier(request.VerifierID)
	if v == nil {
		return Result{
			Error: &ErrVerifierNotFound{VerifierID: request.VerifierID},
		}
	}

	// Step 2: Get execution context (observer NOT called for local_safe)
	var ec verifierauthority.ExecutionContext
	if v.Authority == verifierauthority.AuthorityLocalSafe {
		// Local-safe: no Git observation needed
		ec = *verifierauthority.NewLocalOnlyContext()
	} else {
		// CI authority: observer is called exactly once, even if operation is denied
		ec = observer.Observe(ctx, request.Root)
	}

	// Step 3: Validate operation against authority policy
	if err := validateOperation(v.Authority, request.Operation); err != nil {
		findings := []checks.Finding{
			{
				Path:     v.Name,
				Kind:     "verifier_execution_authority_denied",
				Message:  err.Error(),
				Severity: checks.SeverityError,
			},
		}
		return Result{
			Findings: findings,
			Error:    err,
		}
	}

	// Step 4: Validate authority
	if err := verifierauthority.ValidateAuthority(ec, v.Authority, request.Operation); err != nil {
		findings := []checks.Finding{
			{
				Path:     v.Name,
				Kind:     "verifier_execution_authority_denied",
				Message:  err.Error(),
				Severity: checks.SeverityError,
			},
		}
		return Result{
			Findings: findings,
			Error:    err,
		}
	}

	// Step 5: Authority passed - invoke factory
	runner := factory()
	return Result{
		Findings: runner(request.Root),
	}
}

// DispatchWithDefaultObserver is a convenience wrapper that uses the default
// production context observer. For testing, prefer Dispatch with a fake observer.
func (d *Dispatcher) DispatchWithDefaultObserver(ctx context.Context, request Request, factory RunnerFactory) Result {
	return d.Dispatch(ctx, request, &DefaultContextObserver{}, factory)
}

// validateOperation checks if the operation is allowed for the authority.
// Returns error if the operation is not permitted.
func validateOperation(authority verifierauthority.ExecutionAuthority, operation verifierauthority.VerifierOperation) error {
	// OperationUpdateBaseline is denied for ci_exact_checkout
	if authority == verifierauthority.AuthorityCIExactCheckout && operation == verifierauthority.OperationUpdateBaseline {
		return &verifierauthority.AuthorityError{
			RequiredAuthority: authority,
			Operation:         operation,
			ReasonCode:        verifierauthority.ReasonCodeOperationDenied,
			Message:           "update_baseline operation is not permitted under ci_exact_checkout authority",
		}
	}
	return nil
}

// resolveVerifier looks up a verifier by ID in the registry.
func (d *Dispatcher) resolveVerifier(id string) *registry.Verifier {
	for i := range d.verifiers {
		if d.verifiers[i].Name == id {
			return &d.verifiers[i]
		}
	}
	return nil
}

// LookupVerifier returns a verifier by ID or an error if not found.
func (d *Dispatcher) LookupVerifier(id string) (*registry.Verifier, error) {
	v := d.resolveVerifier(id)
	if v == nil {
		return nil, &ErrVerifierNotFound{VerifierID: id}
	}
	return v, nil
}

// GetVerifiers returns a copy of the verifier registry.
func (d *Dispatcher) GetVerifiers() []registry.Verifier {
	return slices.Clone(d.verifiers)
}

// ProfileRequest represents a single verifier authorization request.
type ProfileRequest struct {
	VerifierID string
	Operation  verifierauthority.VerifierOperation
}

// AuthorizedProfile represents the result of batch authorization.
type AuthorizedProfile struct {
	Verifiers []registry.Verifier
	Denials   []ProfileDenial
}

// ProfileDenial represents an authorization denial for a single verifier.
type ProfileDenial struct {
	VerifierID string
	Findings   []checks.Finding
}

// AuthorizeProfile performs batch authorization for a set of verifier requests.
// This is the correct entry point for factorize, which must authorize ALL dupcode
// verifiers BEFORE creating the shared analysis context.
//
// The ordering guarantee:
//   - Git observation happens first (if needed)
//   - Authority validation happens for all verifiers
//   - Only after ALL authorization passes should the shared context be created
//
// If any verifier is denied, the returned AuthorizedProfile contains only the
// denied verifiers. The caller should NOT create the shared context.
func (d *Dispatcher) AuthorizeProfile(
	ctx context.Context,
	root string,
	requests []ProfileRequest,
	observer ContextObserver,
) (*AuthorizedProfile, error) {
	profile := &AuthorizedProfile{
		Verifiers: make([]registry.Verifier, 0, len(requests)),
		Denials:   make([]ProfileDenial, 0),
	}

	// Phase 1: Resolve all verifier metadata
	resolved := make([]*registry.Verifier, len(requests))
	for i, req := range requests {
		v := d.resolveVerifier(req.VerifierID)
		if v == nil {
			profile.Denials = append(profile.Denials, ProfileDenial{
				VerifierID: req.VerifierID,
				Findings: []checks.Finding{
					{
						Path:     req.VerifierID,
						Kind:     "verifier_not_found",
						Message:  fmt.Sprintf("verifier not found: %s", req.VerifierID),
						Severity: checks.SeverityError,
					},
				},
			})
			continue
		}
		resolved[i] = v
	}

	// Phase 2: Collect Git observation if needed (single observation for all ci_exact_checkout)
	var observedCtx *verifierauthority.ExecutionContext
	needsObservation := false
	for _, v := range resolved {
		if v != nil && v.Authority != verifierauthority.AuthorityLocalSafe {
			needsObservation = true
			break
		}
	}

	if needsObservation {
		ec := observer.Observe(ctx, root)
		observedCtx = &ec
	}

	// Phase 3: Validate authorization for each verifier
	for i, v := range resolved {
		if v == nil {
			continue // Already recorded as denial
		}

		req := requests[i]

		// Get execution context
		var ec verifierauthority.ExecutionContext
		if v.Authority == verifierauthority.AuthorityLocalSafe {
			ec = *verifierauthority.NewLocalOnlyContext()
		} else if observedCtx != nil {
			ec = *observedCtx
		} else {
			// Should not happen, but default to local-only
			ec = *verifierauthority.NewLocalOnlyContext()
		}

		// Validate operation
		if err := validateOperation(v.Authority, req.Operation); err != nil {
			profile.Denials = append(profile.Denials, ProfileDenial{
				VerifierID: v.Name,
				Findings: []checks.Finding{
					{
						Path:     v.Name,
						Kind:     "verifier_execution_authority_denied",
						Message:  err.Error(),
						Severity: checks.SeverityError,
					},
				},
			})
			continue
		}

		// Validate authority
		if err := verifierauthority.ValidateAuthority(ec, v.Authority, req.Operation); err != nil {
			profile.Denials = append(profile.Denials, ProfileDenial{
				VerifierID: v.Name,
				Findings: []checks.Finding{
					{
						Path:     v.Name,
						Kind:     "verifier_execution_authority_denied",
						Message:  err.Error(),
						Severity: checks.SeverityError,
					},
				},
			})
			continue
		}

		// Authorized
		profile.Verifiers = append(profile.Verifiers, *v)
	}

	return profile, nil
}
