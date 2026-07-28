// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// VerifierMetadata is non-executable metadata for an authorized verifier.
// It contains all information needed for display, metrics, and reporting,
// but deliberately excludes any executable function.
type VerifierMetadata struct {
	Name      string
	Lane      registry.VerifierLane
	Authority verifierauthority.ExecutionAuthority
	Kind      registry.ExecutionKind
	ImplID    string
	EnvVars   []string
	Cache     registry.CacheSemantics
}

// cloneVerifierMetadata creates a deep copy of verifier metadata.
func cloneVerifierMetadata(m VerifierMetadata) VerifierMetadata {
	clone := m
	clone.EnvVars = slices.Clone(m.EnvVars)
	return clone
}

// cloneVerifier creates a deep copy of a verifier's metadata portion.
func cloneVerifier(v registry.Verifier) registry.Verifier {
	clone := v
	clone.Execution.EnvVars = slices.Clone(v.Execution.EnvVars)
	return clone
}

// FactoryRunner is what the factory returns: just the verifier ID and the executable function.
// The dispatcher attaches authoritative metadata from its registry.
type FactoryRunner struct {
	VerifierID string
	Run        func(root string) []checks.Finding
}

// ProfileRunnerFactory creates ID-bound verifier runners for the authorized profile.
// The factory receives non-executable authorized verifier metadata and must return
// runners with matching IDs in canonical order. The factory is ONLY invoked after
// authorization passes. The factory may perform expensive operations (like loading
// baselines or creating shared contexts) since it is only called after authorization.
type ProfileRunnerFactory func(authorized []VerifierMetadata) ([]FactoryRunner, error)

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

// ErrProfileNotAuthorized is returned when attempting to execute a binding with a denied profile.
type ErrProfileNotAuthorized struct{}

func (e *ErrProfileNotAuthorized) Error() string {
	return "profile not authorized"
}

// ResourceSnapshot represents resource usage at a point in time.
type ResourceSnapshot struct {
	UserCPU   time.Duration
	SystemCPU time.Duration
	MaxRSSKB  int64
}

// ResourceSampler samples resource usage.
type ResourceSampler interface {
	Sample() (ResourceSnapshot, error)
}

// ExecutionRecord represents the outcome of executing a single verifier.
type ExecutionRecord struct {
	Metadata VerifierMetadata
	Findings []checks.Finding
	Duration time.Duration
	Before   ResourceSnapshot
	After    ResourceSnapshot
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

// BoundProfileRunner is the internal bound runner with authoritative metadata.
type BoundProfileRunner struct {
	Metadata VerifierMetadata
	Run      func(root string) []checks.Finding
}

// Execute runs each bound verifier exactly once in canonical authorized-request order.
// Returns ErrProfileBindingConsumed if called more than once.
// Returns ErrProfileNotAuthorized if the profile was not successfully authorized.
// Measures real wall-clock duration and resource usage for each verifier.
func (b *ProfileBinding) Execute(sampler ResourceSampler) ([]ExecutionRecord, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.consumed {
		return nil, &ErrProfileBindingConsumed{}
	}
	b.consumed = true

	if b.profile == nil {
		return nil, fmt.Errorf("binding has nil profile")
	}

	// Check authorization succeeded
	if !b.profile.AuthorizationSucceeded() {
		return nil, &ErrProfileNotAuthorized{}
	}

	// Execute in canonical order: profile.Requests() order
	records := make([]ExecutionRecord, 0, len(b.runners))
	for _, runner := range b.runners {
		// Sample before execution
		var before ResourceSnapshot
		if sampler != nil {
			var err error
			before, err = sampler.Sample()
			if err != nil {
				return nil, fmt.Errorf("resource sample before %s: %w", runner.Metadata.Name, err)
			}
		}

		started := time.Now()
		var findings []checks.Finding
		if runner.Run != nil {
			findings = runner.Run(b.profile.RepositoryRoot())
		}
		elapsed := time.Since(started)

		// Sample after execution
		var after ResourceSnapshot
		if sampler != nil {
			var err error
			after, err = sampler.Sample()
			if err != nil {
				return nil, fmt.Errorf("resource sample after %s: %w", runner.Metadata.Name, err)
			}
		}

		records = append(records, ExecutionRecord{
			Metadata: cloneVerifierMetadata(runner.Metadata),
			Findings: findings,
			Duration: elapsed,
			Before:   before,
			After:    after,
		})
	}

	return records, nil
}

// Profile returns the authorized profile for this binding.
func (b *ProfileBinding) Profile() *AuthorizedProfile {
	return b.profile
}

// Runners returns metadata-only copies of the bound runners.
// This does NOT expose executable functions; use Execute() for running verifiers.
func (b *ProfileBinding) Runners() []VerifierMetadata {
	if b == nil {
		return nil
	}
	result := make([]VerifierMetadata, len(b.runners))
	for i, r := range b.runners {
		result[i] = cloneVerifierMetadata(r.Metadata)
	}
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

	// Build the exact authorized verifier metadata list in canonical request order
	// Use profile.Requests() for immutable order after authorization
	canonicalOrder := profile.Requests()
	authorized := make([]VerifierMetadata, 0, len(canonicalOrder))
	authorizedSet := make(map[string]bool)
	for _, req := range canonicalOrder {
		v := d.resolveVerifier(req.VerifierID)
		if v != nil {
			// Deep copy for factory input
			metadata := cloneVerifierMetadata(VerifierMetadata{
				Name:      v.Name,
				Lane:      v.Lane,
				Authority: v.Authority,
				Kind:      v.Execution.Kind,
				ImplID:    v.Execution.ImplementationID,
				EnvVars:   v.Execution.EnvVars,
				Cache:     v.Cache,
			})
			authorized = append(authorized, metadata)
			authorizedSet[v.Name] = true
		}
	}

	// NOW invoke the factory - only after authorization passed
	// Factory receives non-executable metadata and returns ID + Run only
	factoryRunners, err := factory(authorized)
	if err != nil {
		return nil, err
	}

	// Validate exact runner set before binding
	if err := validateRunnerSet(authorizedSet, factoryRunners); err != nil {
		// Factory contract violated - no runners bound
		return &ProfileBinding{profile: profile}, err
	}

	// Canonicalize order: reorder runners to match authorized order
	// Attach authoritative metadata from the registry (NOT from factory)
	canonicalized := make([]BoundProfileRunner, len(factoryRunners))
	for i, req := range canonicalOrder {
		for _, fr := range factoryRunners {
			if fr.VerifierID == req.VerifierID {
				// Get authoritative metadata from registry
				v := d.resolveVerifier(fr.VerifierID)
				if v != nil {
					canonicalized[i] = BoundProfileRunner{
						Metadata: cloneVerifierMetadata(VerifierMetadata{
							Name:      v.Name,
							Lane:      v.Lane,
							Authority: v.Authority,
							Kind:      v.Execution.Kind,
							ImplID:    v.Execution.ImplementationID,
							EnvVars:   v.Execution.EnvVars,
							Cache:     v.Cache,
						}),
						Run: fr.Run,
					}
				}
				break
			}
		}
	}

	return &ProfileBinding{profile: profile, runners: canonicalized}, nil
}

// validateRunnerSet verifies the factory returned exactly the authorized runners.
func validateRunnerSet(authorizedSet map[string]bool, runners []FactoryRunner) error {
	// Check cardinality: must match exactly
	if len(runners) != len(authorizedSet) {
		return &ErrProfileFactoryContract{
			Reason: fmt.Sprintf("runner count mismatch: got %d, want %d", len(runners), len(authorizedSet)),
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
