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
//   - Local-safe verify operations perform NO Git observation
//   - Mutation operations and non-local-safe verifiers trigger a full
//     observer round-trip exactly once
package verifierdispatch

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"

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
// The observer is invoked exactly once for non-trivial operations; local-safe
// verification operations that do not mutate state skip the observer entirely.
type ContextObserver interface {
	Observe(ctx context.Context, root string) verifierauthority.ExecutionContext
}

// CountingObserver wraps a ContextObserver and counts how many times
// Observe is invoked. It is used by tests that prove the cheap
// local-safe verification path.
type CountingObserver struct {
	ContextObserver
	count atomic.Int64
}

// Observe records an observation and increments the count.
func (c *CountingObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	c.count.Add(1)
	return c.ContextObserver.Observe(ctx, root)
}

// Count returns the current observation count.
func (c *CountingObserver) Count() int64 { return c.count.Load() }

// DefaultContextObserver is the production context observer using DetectExecutionContext.
type DefaultContextObserver struct{}

// Observe runs the production observation. It performs the bounded Git
// subprocess round-trip exactly once.
func (d *DefaultContextObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.DetectExecutionContext(ctx, root)
}

// CheapLocalSafeObserver is the production observer used for local-safe
// verify operations. It classifies the environment as EnvironmentLocal
// without performing any Git observation.
type CheapLocalSafeObserver struct{}

// Observe returns a NewLocalOnlyContext without invoking Git.
func (c *CheapLocalSafeObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return *verifierauthority.NewLocalOnlyContext()
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
//   - No dupcode lane with local_safe (except the dupcode-update-baseline
//     command-only identity)
//   - No fast lane with ci_exact_checkout
//   - InvocationScope is explicit on every entry
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
// Observer-call matrix:
//
//	local_safe + verify:        0 full Git observations (cheap path)
//	local_safe + update_baseline: 1 full Git observation (env-classification required)
//	ci_exact_checkout + verify:    1 full Git observation
//	ci_exact_checkout + update:    denied before runner (deterministic)
func (d *Dispatcher) Dispatch(ctx context.Context, request Request, observer ContextObserver, factory RunnerFactory) Result {
	// Step 1: Resolve verifier metadata from registry
	v := d.resolveVerifier(request.VerifierID)
	if v == nil {
		return Result{
			Error: &ErrVerifierNotFound{VerifierID: request.VerifierID},
		}
	}

	// Step 2: Determine whether the cheap local-safe verify path applies.
	// Cheap path skips the observer entirely.
	if request.Operation == verifierauthority.OperationVerify && v.Authority == verifierauthority.AuthorityLocalSafe {
		// No observer call; authority is admitted without Git observation.
		runner := factory()
		return Result{
			Findings: runner(request.Root),
		}
	}

	// Step 3: For mutations and non-local-safe verifiers, run the full
	// observer. This is the only path that performs Git observation.
	ec := observer.Observe(ctx, request.Root)

	// Step 4: Classify the environment explicitly. Any classification
	// other than EnvironmentLocal denies a local_safe mutation.
	environment := verifierauthority.ClassifyExecutionEnvironment(ec)

	// Step 5: Validate operation against authority and classified
	// environment. This is the fail-closed mutation gate.
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

	// Step 6: Validate declared authority against observed context.
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

	// Step 7: Authority passed - invoke factory
	runner := factory()
	return Result{
		Findings: runner(request.Root),
	}
}

// DispatchWithDefaultObserver is a convenience wrapper that uses the default
// production context observer. For testing, prefer Dispatch with a fake observer.
func (d *Dispatcher) DispatchWithDefaultObserver(ctx context.Context, request Request, factory RunnerFactory) Result {
	obs := selectObserver(verifierauthority.AuthorityLocalSafe, verifierauthority.OperationVerify)
	return d.Dispatch(ctx, request, obs, factory)
}

// selectObserver returns the cheap local-safe observer for verify
// operations and the full default observer otherwise. This preserves
// the cheap local-safe verify path.
func selectObserver(authority verifierauthority.ExecutionAuthority, operation verifierauthority.VerifierOperation) ContextObserver {
	if operation == verifierauthority.OperationVerify && authority == verifierauthority.AuthorityLocalSafe {
		return &CheapLocalSafeObserver{}
	}
	return &DefaultContextObserver{}
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
