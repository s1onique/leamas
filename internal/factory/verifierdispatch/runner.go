// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"fmt"

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

// ProfileBinding is the result of authorization + binding.
// It contains the authorized profile and bound runners, but does NOT execute them.
// Call ExecuteBoundRunners to execute.
type ProfileBinding struct {
	Profile *AuthorizedProfile
	Runners []BoundProfileRunner // in canonical authorized-request order
}

// ExecuteBoundRunners executes the bound runners with the given executor.
// This separates authorization from execution, ensuring exactly one execution pass.
func (b *ProfileBinding) ExecuteBoundRunners(
	executor func(profile *AuthorizedProfile, runners []BoundProfileRunner) error,
) error {
	if b == nil || b.Profile == nil {
		return fmt.Errorf("binding is nil")
	}
	return executor(b.Profile, b.Runners)
}

// ProfileResult represents the outcome of profile-authorized execution.
type ProfileResult struct {
	Profile  *AuthorizedProfile
	Findings map[string][]checks.Finding // keyed by verifier ID
	AllRun   bool                        // true only if all authorized runners executed
}

// AuthorizeAndBindProfile performs batch authorization and binds runners.
// This does NOT execute the runners - call ExecuteBoundRunners separately.
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

	// If authorization failed, return early - factory was NOT called
	if !profile.AuthorizationSucceeded() {
		return &ProfileBinding{Profile: profile}, nil
	}

	// Build the exact authorized verifier list in canonical request order
	authorized := make([]*registry.Verifier, 0, len(profile.VerifierIDs()))
	seenIDs := make(map[string]bool)
	for _, id := range profile.VerifierIDs() {
		seenIDs[id] = true
	}
	for _, req := range requests {
		if seenIDs[req.VerifierID] {
			v := d.resolveVerifier(req.VerifierID)
			if v != nil {
				authorized = append(authorized, v)
			}
		}
	}

	// NOW invoke the factory - only after authorization passed
	runners, err := factory(authorized)
	if err != nil {
		return nil, err
	}

	// Validate exact runner set before executing anything
	if err := validateRunnerSet(authorized, runners); err != nil {
		// Factory contract violated - no runners execute
		return &ProfileBinding{Profile: profile}, err
	}

	// Runners are already in canonical order from factory (validated above)
	return &ProfileBinding{Profile: profile, Runners: runners}, nil
}

// validateRunnerSet verifies the factory returned exactly the authorized runners.
func validateRunnerSet(authorized []*registry.Verifier, runners []BoundProfileRunner) error {
	// Check cardinality: must match exactly
	if len(runners) != len(authorized) {
		return &ErrProfileFactoryContract{
			Reason: fmt.Sprintf("runner count mismatch: got %d, want %d", len(runners), len(authorized)),
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
	authorizedSet := make(map[string]bool)
	for _, v := range authorized {
		authorizedSet[v.Name] = true
	}

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
