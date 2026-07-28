// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"fmt"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
)

// BoundProfileRunner is an ID-bound verifier runner.
type BoundProfileRunner struct {
	VerifierID string
	Run        func(root string) []checks.Finding
}

// ProfileRunnerFactory creates ID-bound verifier runners for the authorized profile.
// The factory receives the exact authorized verifier entries and must return
// runners with matching IDs in canonical order. The factory is ONLY invoked after
// authorization passes.
type ProfileRunnerFactory func(authorized []*registry.Verifier) ([]BoundProfileRunner, error)

// ErrProfileFactoryContract is returned when the factory violates its contract.
type ErrProfileFactoryContract struct {
	Reason string
}

func (e *ErrProfileFactoryContract) Error() string {
	return "profile factory contract violated: " + e.Reason
}

// ProfileResult represents the outcome of profile-authorized execution.
type ProfileResult struct {
	Profile  *AuthorizedProfile
	Findings map[string][]checks.Finding // keyed by verifier ID
	AllRun   bool                        // true only if all authorized runners executed
}

// AuthorizeAndRunProfile performs batch authorization and executes ONLY the authorized verifiers.
// This is the atomic entry point that binds authorization to execution.
//
// Ordering guarantee:
//   - resolve exact registry entries
//   - observe once when needed
//   - authorize all requests
//   - on any denial: factory is not called, results are empty
//   - invoke factory with the exact resolved entries
//   - validate returned runners match exactly
//   - execute in canonical authorized-request order
//
// This ensures RunFactorize cannot authorize inventory A and execute inventory B.
func (d *Dispatcher) AuthorizeAndRunProfile(
	ctx context.Context,
	root string,
	requests []ProfileRequest,
	observer ContextObserver,
	factory ProfileRunnerFactory,
) (*ProfileResult, error) {
	profile, err := d.AuthorizeProfile(ctx, root, requests, observer)
	if err != nil {
		return nil, err
	}

	result := &ProfileResult{
		Profile:  profile,
		Findings: nil,
		AllRun:   false,
	}

	// If authorization failed, return early - factory was NOT called
	if !profile.AuthorizationSucceeded() {
		return result, nil
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
		return result, err
	}

	// Execute in canonical authorized-request order
	result.Findings = make(map[string][]checks.Finding, len(runners))
	for _, runner := range runners {
		if runner.Run != nil {
			result.Findings[runner.VerifierID] = runner.Run(profile.RepositoryRoot())
		} else {
			result.Findings[runner.VerifierID] = nil
		}
	}
	result.AllRun = true

	return result, nil
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
		if r.VerifierID == "" {
			return &ErrProfileFactoryContract{
				Reason: fmt.Sprintf("runner at index %d has empty VerifierID", i),
			}
		}
		if r.Run == nil {
			return &ErrProfileFactoryContract{
				Reason: fmt.Sprintf("runner %s has nil Run function", r.VerifierID),
			}
		}
		returnedIDs[r.VerifierID]++
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
