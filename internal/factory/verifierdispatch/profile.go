// SPDX-License-Identifier: Apache-2.0

package verifierdispatch

import (
	"context"
	"fmt"

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
	RepositoryRoot string
	// Requests are the original authorization requests.
	Requests []ProfileRequest
	// VerifierIDs are the IDs of the authorized verifiers.
	VerifierIDs []string
	// RegistryDigest is the digest of the authorized registry.
	RegistryDigest string
	// Context is the observed execution context (only for CI authority).
	Context *verifierauthority.ExecutionContext
	// Denials contains all authorization denials if authorization failed.
	Denials []ProfileDenial
	// AuthorizationSucceeded is true if all verifiers were authorized.
	AuthorizationSucceeded bool
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
// Authorization is ALL-OR-NOTHING:
//   - If ANY verifier is denied, the profile returns with AuthorizationSucceeded=false
//   - Only when ALL verifiers are authorized does AuthorizationSucceeded=true
func (d *Dispatcher) AuthorizeProfile(
	ctx context.Context,
	root string,
	requests []ProfileRequest,
	observer ContextObserver,
) (*AuthorizedProfile, error) {
	profile := &AuthorizedProfile{
		RepositoryRoot:         root,
		Requests:               requests,
		VerifierIDs:            make([]string, 0, len(requests)),
		Denials:                make([]ProfileDenial, 0),
		AuthorizationSucceeded: true,
	}

	// Phase 1: Resolve all verifier metadata
	resolved := make([]*registry.Verifier, len(requests))
	for i, req := range requests {
		v := d.resolveVerifier(req.VerifierID)
		if v == nil {
			profile.AuthorizationSucceeded = false
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

	// Phase 2: Collect Git observation if needed
	needsObservation := false
	for _, v := range resolved {
		if v != nil && v.Authority != verifierauthority.AuthorityLocalSafe {
			needsObservation = true
			break
		}
	}

	if needsObservation {
		ec := observer.Observe(ctx, root)
		profile.Context = &ec
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
		} else if profile.Context != nil {
			ec = *profile.Context
		} else {
			ec = *verifierauthority.NewLocalOnlyContext()
		}

		// Validate operation
		if err := validateOperation(v.Authority, req.Operation); err != nil {
			profile.AuthorizationSucceeded = false
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
			profile.AuthorizationSucceeded = false
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

		profile.VerifierIDs = append(profile.VerifierIDs, v.Name)
	}

	// All-or-nothing: clear VerifierIDs if authorization failed
	if !profile.AuthorizationSucceeded {
		profile.VerifierIDs = nil
	}

	// Compute registry digest if authorization succeeded
	if profile.AuthorizationSucceeded {
		profile.RegistryDigest = d.computeRegistryDigest(profile.VerifierIDs)
	}

	return profile, nil
}

// computeRegistryDigest computes a deterministic digest of the authorized verifier IDs.
func (d *Dispatcher) computeRegistryDigest(authorizedIDs []string) string {
	h := 0
	for _, id := range authorizedIDs {
		h += len(id) * 31
	}
	return fmt.Sprintf("v1:%d:%d", len(authorizedIDs), h)
}
