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
	"fmt"

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

// Dispatcher is the central production dispatcher for verifier execution.
type Dispatcher struct {
	verifiers []registry.Verifier
}

// NewDispatcher creates a new dispatcher with the canonical verifier registry.
func NewDispatcher(verifiers []registry.Verifier) *Dispatcher {
	return &Dispatcher{verifiers: verifiers}
}

// Dispatch routes a verifier execution request through authority validation.
// It returns the verifier's findings if authority permits, or a denial finding
// if authority is denied. The RunnerFactory is never invoked unless authority
// validation passes.
func (d *Dispatcher) Dispatch(ctx context.Context, request Request, factory RunnerFactory) Result {
	// Step 1: Resolve verifier metadata from registry
	v := d.resolveVerifier(request.VerifierID)
	if v == nil {
		return Result{
			Error: fmt.Errorf("verifier not found: %s", request.VerifierID),
		}
	}

	// Step 2: Validate authority
	// For local_safe verifiers, skip expensive Git observation
	var ec *verifierauthority.ExecutionContext
	var err error
	if v.Authority == verifierauthority.AuthorityLocalSafe {
		ec, err = verifierauthority.NewLocalOnlyContext(), nil
	} else {
		observed := verifierauthority.DetectExecutionContext(ctx, request.Root)
		ec = &observed
	}
	if err != nil {
		return Result{
			Error: err,
		}
	}
	if err := verifierauthority.ValidateAuthority(*ec, v.Authority, request.Operation); err != nil {
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

	// Step 3: Authority passed - invoke factory
	runner := factory()
	return Result{
		Findings: runner(request.Root),
	}
}

// DispatchGuarded is a convenience wrapper that resolves the execution context
// and applies authority validation. Use this when you cannot supply a pre-built context.
func (d *Dispatcher) DispatchGuarded(ctx context.Context, request Request, runner func(root string) []checks.Finding) Result {
	// Step 1: Resolve verifier metadata from registry
	v := d.resolveVerifier(request.VerifierID)
	if v == nil {
		return Result{
			Error: fmt.Errorf("verifier not found: %s", request.VerifierID),
		}
	}

	// Step 2: Validate authority
	ec := verifierauthority.DetectExecutionContext(ctx, request.Root)
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

	// Step 3: Authority passed - run the verifier
	return Result{
		Findings: runner(request.Root),
	}
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

// ErrVerifierNotFound is returned when a verifier ID is not in the registry.
type ErrVerifierNotFound struct {
	VerifierID string
}

func (e *ErrVerifierNotFound) Error() string {
	return fmt.Sprintf("verifier not found: %s", e.VerifierID)
}

// LookupVerifier returns a verifier by ID or an error if not found.
func (d *Dispatcher) LookupVerifier(id string) (*registry.Verifier, error) {
	v := d.resolveVerifier(id)
	if v == nil {
		return nil, &ErrVerifierNotFound{VerifierID: id}
	}
	return v, nil
}
