// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_diagnostic.go defines the typed diagnostic
// taxonomy for the Closure Protocol v2 verifier foundation.
//
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01
// requires the foundation to publish a stable, extensible
// verifier-specific diagnostic family that later verifier ACTs
// (topology-objects, manifest-results, CLI-tag-state,
// mac-handoff) can rely on without inventing new codes.
//
// The codes live alongside the existing V2DiagnosticCode set
// (defined in closure_protocol_v2_diagnostic.go) so v2 tooling
// keeps a single code registry. New codes follow the existing
// snake_case convention. The Code field is the only stable
// machine identifier; Message text is human-only.

// V2VerifierCode is the typed diagnostic code emitted by the
// Closure Protocol v2 verifier foundation. The string value
// is the canonical snake_case token downstream tooling MUST
// treat as the machine identifier.
type V2VerifierCode string

const (
	// Version-validation codes. Emitted before any Git
	// observation so a malformed request never reaches the
	// resolver.
	V2VerifierUnsupportedClosureProtocolVersion V2VerifierCode = "unsupported_closure_protocol_version"
	V2VerifierUnsupportedPlanContractVersion    V2VerifierCode = "unsupported_plan_contract_version"
	V2VerifierInvalidVersionCombination         V2VerifierCode = "invalid_version_combination"

	// Repository availability code. Emitted when the
	// supplied RepositoryRoot does not name a usable Git
	// repository at the time the resolver is constructed.
	V2VerifierRepositoryUnavailable V2VerifierCode = "repository_unavailable"

	// Path-policy codes. Emitted by request validation when
	// PlanPath or ManifestPath fail the repository-relative
	// path policy.
	V2VerifierPlanPathInvalid     V2VerifierCode = "plan_path_invalid"
	V2VerifierManifestPathInvalid V2VerifierCode = "manifest_path_invalid"

	// Topology codes. Emitted by the topology ACT (ACT 2).
	// Defined here so the foundation ACT publishes the full
	// stable family up front.
	V2VerifierSubjectUnresolved            V2VerifierCode = "subject_unresolved"
	V2VerifierFreezeUnresolved             V2VerifierCode = "freeze_unresolved"
	V2VerifierClosureUnresolved            V2VerifierCode = "closure_unresolved"
	V2VerifierSubjectFreezeEqual           V2VerifierCode = "subject_freeze_equal"
	V2VerifierFreezeClosureEqual           V2VerifierCode = "freeze_closure_equal"
	V2VerifierSubjectClosureEqual          V2VerifierCode = "subject_closure_equal"
	V2VerifierSubjectNotAncestorFreeze     V2VerifierCode = "subject_not_ancestor_of_freeze"
	V2VerifierFreezeNotAncestorClosure     V2VerifierCode = "freeze_not_ancestor_of_closure"
	V2VerifierSubjectFreezeUnrelated       V2VerifierCode = "subject_freeze_unrelated"
	V2VerifierFreezeClosureUnrelated       V2VerifierCode = "freeze_closure_unrelated"
	V2VerifierReverseSubjectFreezeTopology V2VerifierCode = "reverse_subject_freeze_topology"
	V2VerifierReverseFreezeClosureTopology V2VerifierCode = "reverse_freeze_closure_topology"
	V2VerifierTopologyObservationFailed    V2VerifierCode = "topology_observation_failed"

	// Frozen plan and committed manifest authority codes.
	// Emitted when F:P or C:M resolution / read fails.
	V2VerifierFrozenPlanMissing              V2VerifierCode = "frozen_plan_missing"
	V2VerifierFrozenPlanNotBlob              V2VerifierCode = "frozen_plan_not_blob"
	V2VerifierFrozenPlanReadFailed           V2VerifierCode = "frozen_plan_read_failed"
	V2VerifierClosureManifestMissing         V2VerifierCode = "closure_manifest_missing"
	V2VerifierClosureManifestNotBlob         V2VerifierCode = "closure_manifest_not_blob"
	V2VerifierClosureManifestReadFailed      V2VerifierCode = "closure_manifest_read_failed"
	V2VerifierClosureManifestInvalidJSON     V2VerifierCode = "closure_manifest_invalid_json"
	V2VerifierClosureManifestContractInvalid V2VerifierCode = "closure_manifest_contract_invalid"
	// Closure-manifest assertion mismatch. Emitted by the
	// optional mutable-manifest assertion path when the
	// supplied bytes do not match C:M bytes. The assertion
	// is non-authoritative: a mismatch NEVER replaces the
	// C:M binding.
	V2VerifierClosureManifestAssertionMismatch V2VerifierCode = "closure_manifest_assertion_mismatch"

	// Object-format policy codes. Reuses the runner's
	// existing V2CodeObjectFormatUnavailable /
	// V2CodeUnsupportedObjectFormat, but the verifier
	// exposes aliases here so future ACT 2/3 tests can
	// reference a stable verifier namespace without
	// depending on the runner diagnostic file.
	V2VerifierObjectFormatUnavailable V2VerifierCode = "object_format_unavailable"
	V2VerifierUnsupportedObjectFormat V2VerifierCode = "unsupported_object_format"

	// Manifest identity codes. Emitted by
	// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MANIFEST-RESULTS01
	// when the committed manifest at C:M disagrees with the
	// independently computed Git authority or with the frozen
	// plan at F:P. Each code names exactly the manifest field
	// that failed to bind so callers can route the rejection
	// without re-parsing the manifest.
	V2VerifierManifestProtocolVersionMismatch     V2VerifierCode = "manifest_protocol_version_mismatch"
	V2VerifierManifestPlanContractVersionMismatch V2VerifierCode = "manifest_plan_contract_version_mismatch"
	V2VerifierManifestSubjectMismatch             V2VerifierCode = "manifest_subject_mismatch"
	V2VerifierManifestSubjectTreeMismatch         V2VerifierCode = "manifest_subject_tree_mismatch"
	V2VerifierManifestFreezeMismatch              V2VerifierCode = "manifest_freeze_mismatch"
	V2VerifierManifestFreezeTreeMismatch          V2VerifierCode = "manifest_freeze_tree_mismatch"
	V2VerifierManifestExecutionTreeMismatch       V2VerifierCode = "manifest_execution_tree_mismatch"
	V2VerifierManifestPlanPathMismatch            V2VerifierCode = "manifest_plan_path_mismatch"
	V2VerifierManifestPlanBlobMismatch            V2VerifierCode = "manifest_plan_blob_mismatch"
	V2VerifierManifestPlanSHA256Mismatch          V2VerifierCode = "manifest_plan_sha256_mismatch"

	// Binary-identity and result-bijection codes. Emitted by
	// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MANIFEST-RESULTS01
	// when the committed manifest's binary identity is
	// structurally invalid, when the per-check result set is
	// not a bijection against the frozen-plan checks, or when
	// the run success / cleanup state is internally
	// inconsistent.
	V2VerifierManifestBinaryIdentityInvalid      V2VerifierCode = "manifest_binary_identity_invalid"
	V2VerifierManifestCheckResultBijectionFailed V2VerifierCode = "manifest_check_result_bijection_failed"
	V2VerifierManifestUnsuccessfulRun            V2VerifierCode = "manifest_unsuccessful_run"
	V2VerifierManifestCheckResultsInvalid        V2VerifierCode = "manifest_check_results_invalid"

	// Frozen-plan check inventory codes. Emitted when the
	// committed manifest references checks that are not in
	// the frozen plan at F:P, or when the frozen-plan
	// document fails the production parser before the
	// manifest is consulted.
	V2VerifierFrozenPlanInvalid      V2VerifierCode = "frozen_plan_invalid"
	V2VerifierManifestUnknownCheckID V2VerifierCode = "manifest_unknown_check_id"
)

// V2VerifierDiagnostic is the structured diagnostic record
// emitted by the Closure Protocol v2 verifier. The Code field
// is the only stable machine identifier; the remaining fields
// are human-readable context that may evolve.
//
// All fields except Code are optional. PropertyName is the
// request field path that produced the diagnostic (for
// request-validation codes). ObjectOID / ObjectPath are the
// Git object anchor that produced the diagnostic (for
// topology and authority codes). Expected / Observed report
// asserted vs actual values for diagnostic-by-comparison
// codes. Detail is a free-form human description.
type V2VerifierDiagnostic struct {
	Code         V2VerifierCode `json:"code"`
	Message      string         `json:"message"`
	PropertyName string         `json:"property_name,omitempty"`
	Expected     string         `json:"expected,omitempty"`
	Observed     string         `json:"observed,omitempty"`
	ObjectOID    string         `json:"object_oid,omitempty"`
	ObjectPath   string         `json:"object_path,omitempty"`
	Detail       string         `json:"detail,omitempty"`
}

// V2VerifierDiagnostics is an ordered list of
// V2VerifierDiagnostic. Order is preserved so tests and audit
// logs remain deterministic.
type V2VerifierDiagnostics []V2VerifierDiagnostic

// HasCode reports whether any diagnostic carries the given
// code. The check is order-independent and case-sensitive.
func (d V2VerifierDiagnostics) HasCode(code V2VerifierCode) bool {
	for _, item := range d {
		if item.Code == code {
			return true
		}
	}
	return false
}

// Codes returns the deduplicated code list, preserving first
// occurrence order. Tests assert on a closed set of failure
// codes; the helper exists so they can compare slices without
// re-implementing dedup.
func (d V2VerifierDiagnostics) Codes() []V2VerifierCode {
	seen := make(map[V2VerifierCode]bool)
	out := make([]V2VerifierCode, 0, len(d))
	for _, item := range d {
		if seen[item.Code] {
			continue
		}
		seen[item.Code] = true
		out = append(out, item.Code)
	}
	return out
}

// NewV2VerifierDiagnostic constructs a diagnostic with code
// and human-readable message. PropertyName, Expected, Observed,
// ObjectOID, ObjectPath, and Detail are optional and may be
// left empty.
func NewV2VerifierDiagnostic(code V2VerifierCode, message string) V2VerifierDiagnostic {
	return V2VerifierDiagnostic{Code: code, Message: message}
}

// V2VerifierError wraps a non-empty diagnostic list with the
// Go error interface so callers can choose between typed
// inspection (Diags) and the standard error interface
// (Error).
type V2VerifierError struct {
	Diags V2VerifierDiagnostics
	Cause error
}

// Error returns a stable "<code>: <message>" rendering for the
// first diagnostic, falling back to a fixed label when no
// diagnostic is present.
func (e *V2VerifierError) Error() string {
	if e == nil || len(e.Diags) == 0 {
		return "v2 closure verifier failure"
	}
	first := e.Diags[0]
	if first.Message == "" {
		return string(first.Code)
	}
	return string(first.Code) + ": " + first.Message
}

// Unwrap returns the wrapped cause so errors.Is and
// errors.As can reach the original error. Returns nil when no
// cause was recorded.
func (e *V2VerifierError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewV2VerifierError constructs a V2VerifierError carrying a
// single diagnostic. Multi-diagnostic failures are built up by
// appending to the returned Diags slice and re-wrapping with
// WrapV2VerifierError.
func NewV2VerifierError(diag V2VerifierDiagnostic) *V2VerifierError {
	return &V2VerifierError{Diags: V2VerifierDiagnostics{diag}}
}

// WrapV2VerifierError attaches the supplied cause to a fresh
// V2VerifierError. Used when the verifier wraps an underlying
// non-verifier error (for example a *V2Error from the runner
// or a plain error from the Git process authority).
func WrapV2VerifierError(diag V2VerifierDiagnostic, cause error) *V2VerifierError {
	return &V2VerifierError{Diags: V2VerifierDiagnostics{diag}, Cause: cause}
}
