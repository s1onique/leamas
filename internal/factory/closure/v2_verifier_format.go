// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_format.go implements the SHA-1 object-format
// policy for the v2 closure verifier foundation.
//
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01
// requires the verifier to reject:
//
//   - sha1: accepted
//   - sha256 / sha256d / any other format: rejected
//   - empty format: rejected
//   - resolver failure: rejected
//
// The format check MUST occur BEFORE any OID-length validation
// so a sha256 repository never accepts a misleading 64-char
// OID that would otherwise pass a length-based filter.
//
// The policy is layered on top of the production runner's
// existing EnforceSHA1ObjectFormat so both code paths converge
// on a single source of truth for SHA-1 enforcement. The
// wrapper exists because the verifier interface
// (V2ClosureGitAuthority) is structurally distinct from the
// runner's GitObjectResolver interface.

import "context"

// EnforceV2VerifierObjectFormatPolicy applies the SHA-1 object
// format policy to the supplied v2 closure verifier authority.
// The check is fail-closed and runs BEFORE any OID validation
// or commit resolution.
//
// On success, the function returns nil. On failure, the
// returned error is a *V2VerifierError carrying a stable code:
//
//   - object_format_unavailable: resolver failed or returned
//     empty
//   - unsupported_object_format: resolver returned a format
//     other than "sha1"
//
// The function never accepts a sha256 repository regardless
// of OID length; the typed diagnostic is the authoritative
// verdict.
func EnforceV2VerifierObjectFormatPolicy(authority V2ClosureGitAuthority) error {
	if authority == nil {
		return NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierObjectFormatUnavailable,
			"git authority is required for object-format policy",
		))
	}
	// The foundation ACT only implements the ObjectFormat
	// method on the new authority interface, so we
	// delegate to the legacy GitObjectResolver adapter to
	// reuse the production policy logic.
	adapter := v2AuthorityFormatAdapter{authority: authority}
	if v2err := EnforceSHA1ObjectFormat(adapter); v2err != nil {
		// Translate *V2Error -> *V2VerifierError so
		// callers see a stable verifier-code namespace.
		v2vErr, ok := v2err.(*V2Error)
		if !ok {
			return WrapV2VerifierError(
				NewV2VerifierDiagnostic(V2VerifierObjectFormatUnavailable, v2err.Error()),
				v2err,
			)
		}
		var diag V2VerifierDiagnostic
		if len(v2vErr.Diags) > 0 {
			first := v2vErr.Diags[0]
			diag = V2VerifierDiagnostic{
				Code:         V2VerifierCode(first.Code),
				Message:      first.Message,
				PropertyName: first.PropertyName,
				Detail:       first.Detail,
			}
		} else {
			diag = NewV2VerifierDiagnostic(V2VerifierObjectFormatUnavailable, "object format policy rejected")
		}
		return NewV2VerifierError(diag)
	}
	return nil
}

// v2AuthorityFormatAdapter bridges the v2 closure verifier
// authority interface to the production runner's
// GitObjectResolver interface so the existing SHA-1 policy
// helper applies unchanged.
//
// The adapter implements only the methods required by
// EnforceSHA1ObjectFormat (ObjectFormat). CatFile is
// supplied as a panic-on-call stub because the format policy
// MUST short-circuit before any blob read; if a regression
// causes CatFile to run, the panic surfaces the defect
// immediately rather than silently producing a misleading
// success.
type v2AuthorityFormatAdapter struct {
	authority V2ClosureGitAuthority
}

// ObjectFormat delegates to the bound v2 closure verifier
// authority. The function never uses the process CWD; the
// authority is bound to a single repository root at
// construction time.
func (a v2AuthorityFormatAdapter) ObjectFormat() (string, error) {
	return a.authority.ObjectFormat(context.Background())
}

// CatFile panics if invoked. The SHA-1 format policy MUST
// reject sha256 repositories before any OID validation, and
// the format-policy helper does not read blobs. If a future
// regression causes CatFile to run, the panic surfaces the
// defect immediately.
func (a v2AuthorityFormatAdapter) CatFile(oid string) ([]byte, error) {
	panic("v2AuthorityFormatAdapter.CatFile must not be invoked; " +
		"EnforceSHA1ObjectFormat MUST reject unsupported formats before any blob read")
}
