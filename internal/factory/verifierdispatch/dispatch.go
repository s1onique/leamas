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
// The registry is deeply copied; caller may modify original after construction.
// Each verifier's EnvVars slice is cloned to prevent mutation.
func NewDispatcher(verifiers []registry.Verifier) (*Dispatcher, error) {
	if len(verifiers) == 0 {
		return nil, errors.New("verifiers slice is empty")
	}

	// Deep copy: clone outer slice and each verifier's EnvVars
	copied := make([]registry.Verifier, len(verifiers))
	for i, v := range verifiers {
		copied[i] = cloneRegistryVerifier(v)
	}

	seen := make(map[string]bool)

	for i, v := range copied {
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

	return &Dispatcher{verifiers: copied}, nil
}

// cloneRegistryVerifier creates a deep copy of a registry verifier.
func cloneRegistryVerifier(v registry.Verifier) registry.Verifier {
	clone := v
	clone.Execution.EnvVars = slices.Clone(v.Execution.EnvVars)
	return clone
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
// Authorization is fail-closed: the explicit execution-environment
// classification drives every mutation decision. The ContextObserver is
// always invoked (cheap zero-cost observer for local_safe verifiers is the
// trust boundary that sets the LocalTrust sentinel).
func (d *Dispatcher) Dispatch(ctx context.Context, request Request, observer ContextObserver, factory RunnerFactory) Result {
	// Step 1: Resolve verifier metadata from registry
	v := d.resolveVerifier(request.VerifierID)
	if v == nil {
		return Result{
			Error: &ErrVerifierNotFound{VerifierID: request.VerifierID},
		}
	}

	// Step 2: Get execution context from the observer. The observer is the
	// only trust boundary permitted to set LocalTrust.
	ec := observer.Observe(ctx, request.Root)

	// Step 3: Classify the environment explicitly. Any classification other
	// than EnvironmentLocal denies a local_safe mutation, regardless of how
	// permissive the declared authority is.
	environment := verifierauthority.ClassifyExecutionEnvironment(ec)

	// Step 4: Validate the operation against BOTH the declared authority
	// and the classified environment. This is the fail-closed mutation
	// gate: a local_safe update is admitted only when the environment is
	// EnvironmentLocal; any CI / GitHub Actions / unknown environment
	// denies the mutation.
	if err := verifierauthority.ValidateOperationInContext(v.Authority, request.Operation, environment); err != nil {
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

	// Step 5: Validate declared authority against the observed context.
	// This is the historical per-context acceptance check (CI exact
	// checkout markers, SHA validity, etc.).
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

	// Step 6: Authority passed - invoke factory
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

// resolveVerifier looks up a verifier by ID in the registry.
func (d *Dispatcher) resolveVerifier(id string) *registry.Verifier {
	for i := range d.verifiers {
		if d.verifiers[i].Name == id {
			return &d.verifiers[i]
		}
	}
	return nil
}

// LookupVerifierMetadata returns non-executable metadata for a verifier by ID.
// This does NOT expose the Run function; use Dispatch for execution.
func (d *Dispatcher) LookupVerifierMetadata(id string) (VerifierMetadata, error) {
	v := d.resolveVerifier(id)
	if v == nil {
		return VerifierMetadata{}, &ErrVerifierNotFound{VerifierID: id}
	}
	return VerifierMetadata{
		Name:      v.Name,
		Lane:      v.Lane,
		Authority: v.Authority,
		Kind:      v.Execution.Kind,
		ImplID:    v.Execution.ImplementationID,
		EnvVars:   slices.Clone(v.Execution.EnvVars),
		Cache:     v.Cache,
	}, nil
}

// GetVerifierMetadata returns non-executable metadata for all verifiers.
// This does NOT expose Run functions; use Dispatch for execution.
func (d *Dispatcher) GetVerifierMetadata() []VerifierMetadata {
	result := make([]VerifierMetadata, len(d.verifiers))
	for i, v := range d.verifiers {
		result[i] = VerifierMetadata{
			Name:      v.Name,
			Lane:      v.Lane,
			Authority: v.Authority,
			Kind:      v.Execution.Kind,
			ImplID:    v.Execution.ImplementationID,
			EnvVars:   slices.Clone(v.Execution.EnvVars),
			Cache:     v.Cache,
		}
	}
	return result
}
