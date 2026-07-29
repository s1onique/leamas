// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"slices"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// ProfileRequest represents a single verifier authorization request.
type ProfileRequest struct {
	VerifierID string
	Operation  verifierauthority.VerifierOperation
}

// AuthorizedProfile represents the result of batch authorization.
// Authorization is all-or-nothing: either ALL verifiers are authorized,
// or ALL denials are returned. There is no partial authorization.
type AuthorizedProfile struct {
	// RepositoryRoot is the root that was authorized.
	root string
	// Requests are the original authorization requests (defensive copy).
	requests []ProfileRequest
	// VerifierIDs are the IDs of the authorized verifiers (defensive copy).
	verifierIDs []string
	// registryDigest is the SHA-256 identity digest of the authorized verifier set.
	registryDigest [32]byte
	// context is the observed execution context (cloned).
	context *verifierauthority.ExecutionContext
	// denials contains all authorization denials if authorization failed.
	denials []ProfileDenial
	// authorizationSucceeded is true if all verifiers were authorized.
	authorizationSucceeded bool
}

// ProfileDenial represents an authorization denial for a single verifier.
type ProfileDenial struct {
	VerifierID string
	Findings   []checks.Finding
}

// NewAuthorizedProfile creates a new authorized profile. Use this constructor
// for defensive field initialization.
func NewAuthorizedProfile() *AuthorizedProfile {
	return &AuthorizedProfile{
		requests:    make([]ProfileRequest, 0),
		verifierIDs: make([]string, 0),
		denials:     make([]ProfileDenial, 0),
	}
}

// Getters for immutable access

func (p *AuthorizedProfile) RepositoryRoot() string { return p.root }

func (p *AuthorizedProfile) Requests() []ProfileRequest {
	if p == nil {
		return nil
	}
	return slices.Clone(p.requests)
}

func (p *AuthorizedProfile) VerifierIDs() []string {
	if p == nil {
		return nil
	}
	return slices.Clone(p.verifierIDs)
}

func (p *AuthorizedProfile) RegistryDigest() [32]byte {
	if p == nil {
		return [32]byte{}
	}
	return p.registryDigest
}

func (p *AuthorizedProfile) Context() *verifierauthority.ExecutionContext {
	if p == nil || p.context == nil {
		return nil
	}
	ec := *p.context
	return &ec
}

func (p *AuthorizedProfile) Denials() []ProfileDenial {
	if p == nil {
		return nil
	}
	// Deep clone to prevent mutation of inner Findings slices
	return cloneDenials(p.denials)
}

// cloneDenials creates a deep clone of the denials slice.
func cloneDenials(input []ProfileDenial) []ProfileDenial {
	output := make([]ProfileDenial, len(input))
	for i := range input {
		output[i] = input[i]
		output[i].Findings = slices.Clone(input[i].Findings)
	}
	return output
}

func (p *AuthorizedProfile) AuthorizationSucceeded() bool {
	if p == nil {
		return false
	}
	return p.authorizationSucceeded
}

// AuthorizeProfile performs batch authorization for a set of verifier requests.
// This is the correct entry point for factorize, which must authorize ALL dupcode
// verifiers BEFORE creating the shared analysis context.
//
// Authorization is ALL-OR-NOTHING:
//   - If ANY verifier is denied, the profile returns with AuthorizationSucceeded=false
//   - Only when ALL verifiers are authorized does AuthorizationSucceeded=true
//
// Input hardening:
//   - Rejects nil observer when remote authority is requested
//   - Rejects duplicate verifier requests
//   - Defensive copies of all mutable inputs
func (d *Dispatcher) AuthorizeProfile(
	ctx context.Context,
	root string,
	requests []ProfileRequest,
	observer ContextObserver,
) (*AuthorizedProfile, error) {
	// Input validation
	if root == "" {
		return nil, errors.New("root cannot be empty")
	}
	if len(requests) == 0 {
		return nil, errors.New("requests cannot be empty")
	}

	// Defensive copy of requests
	requests = slices.Clone(requests)

	// Check for duplicate requests in the copy
	seenIDs := make(map[string]bool)
	for _, req := range requests {
		if seenIDs[req.VerifierID] {
			return nil, fmt.Errorf("duplicate verifier request: %s", req.VerifierID)
		}
		seenIDs[req.VerifierID] = true
	}

	profile := &AuthorizedProfile{
		root:                   root,
		requests:               requests,
		verifierIDs:            make([]string, 0, len(requests)),
		denials:                make([]ProfileDenial, 0),
		authorizationSucceeded: true,
	}

	// Phase 1: Resolve all verifier metadata
	resolved := make([]*registry.Verifier, len(requests))
	for i, req := range requests {
		v := d.resolveVerifier(req.VerifierID)
		if v == nil {
			profile.authorizationSucceeded = false
			profile.denials = append(profile.denials, ProfileDenial{
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

	// Phase 2: Check if remote observation is needed and validate observer
	needsObservation := false
	for _, v := range resolved {
		if v != nil && v.Authority != verifierauthority.AuthorityLocalSafe {
			needsObservation = true
			break
		}
	}

	if needsObservation && observer == nil {
		return nil, errors.New("observer cannot be nil for remote authority")
	}

	if needsObservation {
		ec := observer.Observe(ctx, root)
		// Clone the context to prevent mutation
		cloned := ec
		profile.context = &cloned
	}

	// Phase 3: Validate authorization for each verifier
	for i, v := range resolved {
		if v == nil {
			continue
		}

		req := requests[i]
		var ec verifierauthority.ExecutionContext
		if v.Authority == verifierauthority.AuthorityLocalSafe {
			ec = *verifierauthority.NewLocalOnlyContext()
		} else if profile.context != nil {
			ec = *profile.context
		} else {
			ec = *verifierauthority.NewLocalOnlyContext()
		}

		// Classify the environment explicitly for fail-closed mutation
		// gating. local_safe verifiers with the trusted local observer
		// produce EnvironmentLocal; everything else falls through to a
		// deny-kind classification.
		environment := verifierauthority.ClassifyExecutionEnvironment(ec)

		// Validate operation against declared authority and classified
		// environment. This replaces the legacy validateOperation check
		// with the fail-closed mutation gate.
		if err := verifierauthority.ValidateOperationInContext(v.Authority, req.Operation, environment); err != nil {
			profile.authorizationSucceeded = false
			profile.denials = append(profile.denials, ProfileDenial{
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
			profile.authorizationSucceeded = false
			profile.denials = append(profile.denials, ProfileDenial{
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

		profile.verifierIDs = append(profile.verifierIDs, v.Name)
	}

	// All-or-nothing: clear VerifierIDs if authorization failed
	if !profile.authorizationSucceeded {
		profile.verifierIDs = nil
	}

	// Deep clone denials for storage
	if len(profile.denials) > 0 {
		profile.denials = cloneDenials(profile.denials)
	}

	// Compute registry digest if authorization succeeded
	if profile.authorizationSucceeded {
		profile.registryDigest = d.computeRegistryDigest(profile.requests, resolved)
	}

	return profile, nil
}

// computeRegistryDigest computes a canonical SHA-256 digest of the authorized verifier set.
// Uses streaming hash to avoid fixed-buffer panic risk.
// The digest includes: verifier IDs, lanes, authorities, operations, and execution metadata.
// Ordering is canonical (sorted by verifier ID).
func (d *Dispatcher) computeRegistryDigest(requests []ProfileRequest, resolved []*registry.Verifier) [32]byte {
	// Build sorted slice of resolved verifiers for canonical ordering
	type entry struct {
		id  string
		v   *registry.Verifier
		req ProfileRequest
	}
	entries := make([]entry, 0, len(resolved))
	for i, v := range resolved {
		if v != nil {
			entries = append(entries, entry{
				id:  v.Name,
				v:   v,
				req: requests[i],
			})
		}
	}

	// Sort by verifier ID for canonical ordering
	slices.SortFunc(entries, func(left, right entry) int {
		return cmpString(left.id, right.id)
	})

	// Use streaming hash to avoid fixed-buffer overflow
	h := sha256.New()

	for _, e := range entries {
		// Write verifier name
		writeString(h, e.v.Name)

		// Write lane
		writeString(h, string(e.v.Lane))

		// Write authority
		writeString(h, string(e.v.Authority))

		// Write operation
		writeString(h, string(e.req.Operation))

		// Write execution kind
		writeString(h, string(e.v.Execution.Kind))

		// Write implementation ID
		writeString(h, e.v.Execution.ImplementationID)
	}

	// Get the hash sum
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return digest
}

// writeString writes a length-prefixed string to the hash using streaming writes.
func writeString(h hash.Hash, s string) {
	var length [4]byte
	binary.LittleEndian.PutUint32(length[:], uint32(len(s)))
	h.Write(length[:])
	h.Write([]byte(s))
}

// cmpString compares two strings lexicographically.
func cmpString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
