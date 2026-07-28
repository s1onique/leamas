// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"fmt"
	"sync"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
)

// BoundProfileRunner is an ID-bound verifier runner with complete metadata.
type BoundProfileRunner struct {
	Verifier registry.Verifier // complete registered metadata
	Run      func(root string) []checks.Finding
}

// ProfileRunnerFactory creates ID-bound verifier runners for the authorized profile.
// The factory receives the exact authorized verifier entries and must return
// runners with matching IDs in canonical order. The factory is ONLY invoked after
// authorization passes. The factory may perform expensive operations (like loading
// baselines or creating shared contexts) since it is only called after authorization.
type ProfileRunnerFactory func(authorized []*registry.Verifier) ([]BoundProfileRunner, error)

// ErrProfileFactoryContract is returned when the factory violates its contract.
type ErrProfileFactoryContract struct {
	Reason string
}

func (e *ErrProfileFactoryContract) Error() string {
	return "profile factory contract violated: " + e.Reason
}

// ErrProfileBindingConsumed is returned when attempting to execute an already-executed binding.
type ErrProfileBindingConsumed struct{}

func (e *ErrProfileBindingConsumed) Error() string {
	return "profile binding already consumed"
}

// ProfileBinding is the result of authorization + binding.
// It contains the authorized profile and bound runners, but does NOT execute them.
// Call Execute to run the bound verifiers exactly once.
// The binding is consumed after execution and cannot be replayed.
type ProfileBinding struct {
	profile  *AuthorizedProfile
	runners  []BoundProfileRunner
	consumed bool
	mu       sync.Mutex
}

// Execute runs each bound verifier exactly once in canonical authorized-request order.
// Returns ErrProfileBindingConsumed if called more than once.
// The findings are keyed by verifier ID in execution order.
func (b *ProfileBinding) Execute() (map[string][]checks.Finding, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.consumed {
		return nil, &ErrProfileBindingConsumed{}
	}
	b.consumed = true

	if b.profile == nil {
		return nil, fmt.Errorf("binding has nil profile")
	}

	// Execute in canonical order: profile.Requests() order
	findings := make(map[string][]checks.Finding, len(b.runners))
	for _, runner := range b.runners {
		if runner.Run != nil {
			findings[runner.Verifier.Name] = runner.Run(b.profile.RepositoryRoot())
		} else {
			findings[runner.Verifier.Name] = nil
		}
	}

	return findings, nil
}

// Profile returns the authorized profile for this binding.
func (b *ProfileBinding) Profile() *AuthorizedProfile {
	return b.profile
}

// Runners returns a copy of the bound runners in canonical order.
func (b *ProfileBinding) Runners() []BoundProfileRunner {
	if b == nil {
		return nil
	}
	result := make([]BoundProfileRunner, len(b.runners))
	copy(result, b.runners)
	return result
}

// AuthorizeAndBindProfile performs batch authorization and binds runners.
// This does NOT execute the runners - call Execute separately.
// This is the atomic entry point that binds authorization to runner creation.
func (d *Dispatcher) AuthorizeAndBindProfile(
	ctx context.Context,
	root string,
	requests []ProfileRequest,
	observer ContextObserver,
	factory ProfileRunnerFactory,
) (*ProfileBinding, error) {
	profile, err := d.AuthorizeProfile(ctx, root, requests, observer)
	if err != nil {
		return nil, err
	}

	// If authorization failed, return binding without runners
	if !profile.AuthorizationSucceeded() {
		return &ProfileBinding{profile: profile}, nil
	}

	// Build the exact authorized verifier list in canonical request order
	// Use profile.Requests() for immutable order after authorization
	canonicalOrder := profile.Requests()
	authorized := make([]*registry.Verifier, 0, len(canonicalOrder))
	authorizedSet := make(map[string]bool)
	for _, req := range canonicalOrder {
		v := d.resolveVerifier(req.VerifierID)
		if v != nil {
			authorized = append(authorized, v)
			authorizedSet[v.Name] = true
		}
	}

	// NOW invoke the factory - only after authorization passed
	runners, err := factory(authorized)
	if err != nil {
		return nil, err
	}

	// Validate exact runner set before binding
	if err := validateRunnerSet(authorizedSet, runners); err != nil {
		// Factory contract violated - no runners bound
		return &ProfileBinding{profile: profile}, err
	}

	// Canonicalize order: reorder runners to match authorized order
	canonicalized := make([]BoundProfileRunner, len(runners))
	for i, req := range canonicalOrder {
		for _, runner := range runners {
			if runner.Verifier.Name == req.VerifierID {
				canonicalized[i] = runner
				break
			}
		}
	}

	return &ProfileBinding{profile: profile, runners: canonicalized}, nil
}

// validateRunnerSet verifies the factory returned exactly the authorized runners.
func validateRunnerSet(authorizedSet map[string]bool, runners []BoundProfileRunner) error {
	// Check cardinality: must match exactly
	if len(runners) != len(authorizedSet) {
		return &ErrProfileFactoryContract{
			Reason: fmt.Sprintf("runner count mismatch: got %d, want %d", len(runners), len(authorizedSet)),
		}
	}

	// Build sets for validation
	returnedIDs := make(map[string]int)
	for i, r := range runners {
		// Use Verifier.Name as the canonical ID
		if r.Verifier.Name == "" {
			return &ErrProfileFactoryContract{
				Reason: fmt.Sprintf("runner at index %d has empty Verifier.Name", i),
			}
		}
		if r.Run == nil {
			return &ErrProfileFactoryContract{
				Reason: fmt.Sprintf("runner %s has nil Run function", r.Verifier.Name),
			}
		}
		returnedIDs[r.Verifier.Name]++
	}

	// Check no duplicates in returned set
	for id, count := range returnedIDs {
		if count > 1 {
			return &ErrProfileFactoryContract{
				Reason: fmt.Sprintf("duplicate runner ID: %s (count=%d)", id, count),
			}
		}
	}

	// Check exact set equality
	for id := range returnedIDs {
		if !authorizedSet[id] {
			return &ErrProfileFactoryContract{
				Reason: fmt.Sprintf("unexpected runner ID: %s", id),
			}
		}
	}
	for id := range authorizedSet {
		if _, ok := returnedIDs[id]; !ok {
			return &ErrProfileFactoryContract{
				Reason: fmt.Sprintf("missing runner for authorized ID: %s", id),
			}
		}
	}

	return nil
}
