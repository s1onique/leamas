// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
)

// ProfileRunnerFactory creates verifier runners bound to the authorized profile.
// The factory receives the exact authorized verifier entries and must return
// runners in the same order. The factory is ONLY invoked after authorization passes.
type ProfileRunnerFactory func(authorized []*registry.Verifier) []func(root string) []checks.Finding

// ProfileResult represents the outcome of profile-authorized execution.
type ProfileResult struct {
	Profile  *AuthorizedProfile
	Results  [][]checks.Finding
	Errors   []error
	AllFound bool // true if all verifiers produced results
}

// AuthorizeAndRunProfile performs batch authorization and executes ONLY the authorized verifiers.
// This is the atomic entry point that binds authorization to execution.
//
// Ordering guarantee:
//   - resolve exact registry entries
//   - observe once when needed
//   - authorize all requests
//   - on any denial: factory count remains zero, results are empty
//   - invoke factory with the exact resolved entries
//   - run only those entries
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
		Results:  nil,
		Errors:   nil,
		AllFound: false,
	}

	// If authorization failed, return early with empty results
	// Factory was NOT called - this is the key invariant
	if !profile.AuthorizationSucceeded() {
		return result, nil
	}

	// Build the exact authorized verifier list
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
	runners := factory(authorized)

	// Execute only the authorized verifiers
	result.Results = make([][]checks.Finding, len(runners))
	result.Errors = make([]error, len(runners))
	result.AllFound = true

	for i, runner := range runners {
		if runner != nil {
			result.Results[i] = runner(profile.RepositoryRoot())
		} else {
			result.Results[i] = nil
			result.AllFound = false
		}
	}

	return result, nil
}
