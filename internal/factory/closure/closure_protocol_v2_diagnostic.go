// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_diagnostic.go centralises the typed
// diagnostic codes for Closure Protocol v2. Each code maps to
// a stable snake_case identifier so machine handling never
// needs to parse message strings.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

// V2DiagnosticCode is the closed set of typed diagnostic
// identifiers emitted by the Closure Protocol v2 runner.
// The string value is the canonical snake_case token that
// downstream tooling MUST treat as the machine identifier.
// Message text is human-only and may evolve without notice.
type V2DiagnosticCode string

const (
	V2CodeUnsupportedClosureProtocolVersion V2DiagnosticCode = "unsupported_closure_protocol_version"
	V2CodeUnsupportedPlanContractVersion    V2DiagnosticCode = "unsupported_plan_contract_version"
	V2CodeUnsupportedPlanProtocolComb       V2DiagnosticCode = "unsupported_plan_protocol_combination"
	V2CodeSubjectCommitNotFound             V2DiagnosticCode = "subject_commit_not_found"
	V2CodeFreezeCommitNotFound              V2DiagnosticCode = "freeze_commit_not_found"
	V2CodeSubjectEqualsFreeze               V2DiagnosticCode = "subject_equals_freeze"
	V2CodeSubjectNotAncestorOfFreeze        V2DiagnosticCode = "subject_not_ancestor_of_freeze"
	V2CodeFreezeAncestorOfSubject           V2DiagnosticCode = "freeze_ancestor_of_subject"
	V2CodeSubjectFreezeUnrelated            V2DiagnosticCode = "subject_freeze_unrelated"
	V2CodeFrozenPlanPathMissing             V2DiagnosticCode = "frozen_plan_path_missing"
	V2CodeFrozenPlanNotBlob                 V2DiagnosticCode = "frozen_plan_not_blob"
	V2CodeFrozenPlanInvalid                 V2DiagnosticCode = "frozen_plan_invalid"
	V2CodeInvalidPlanPath                   V2DiagnosticCode = "invalid_plan_path"
	V2CodeWorkingPlanMismatch               V2DiagnosticCode = "working_plan_mismatch"
	V2CodeEvidencePathNotDetached           V2DiagnosticCode = "evidence_path_not_detached"
	V2CodeManifestPathNotDetached           V2DiagnosticCode = "manifest_path_not_detached"
	V2CodeWorkingPlanPathInvalid            V2DiagnosticCode = "working_plan_path_invalid"
	V2CodeCallerWorktreeDirty               V2DiagnosticCode = "caller_worktree_dirty"
	V2CodeExecutionTreeMismatch             V2DiagnosticCode = "execution_tree_mismatch"
	V2CodeGitOperationFailed                V2DiagnosticCode = "git_operation_failed"
	V2CodeExecutionFailed                   V2DiagnosticCode = "execution_failed"
	V2CodeCleanupFailed                     V2DiagnosticCode = "cleanup_failed"
	V2CodeManifestWriteFailed               V2DiagnosticCode = "manifest_write_failed"
	V2CodeRequestIncomplete                 V2DiagnosticCode = "request_incomplete"
	V2CodeBinaryIdentityInvalid             V2DiagnosticCode = "binary_identity_invalid"
	V2CodeManifestIdentityInvalid           V2DiagnosticCode = "manifest_identity_invalid"
	V2CodeCheckResultMappingInvalid         V2DiagnosticCode = "check_result_mapping_invalid"
	// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-LIFECYCLE-INVARIANTS01
	// adds caller-state authority codes. Each code reports a
	// specific invariant violation discovered after the
	// runner returns so the CLI can render the exact cause.
	V2CodeCallerHeadChanged          V2DiagnosticCode = "caller_head_changed"
	V2CodeCallerTreeChanged          V2DiagnosticCode = "caller_tree_changed"
	V2CodeCallerWorktreeDirtyAfter   V2DiagnosticCode = "caller_worktree_dirty_after"
	V2CodeWorktreeRegistrationLeaked V2DiagnosticCode = "worktree_registration_leaked"
	// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02
	// adds caller refs drift code. The snapshot captures
	// `git for-each-ref` output and any drift between BEFORE
	// and AFTER produces this typed diagnostic.
	V2CodeCallerRefsChanged V2DiagnosticCode = "caller_refs_changed"
	// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-LIFECYCLE-INVARIANTS01
	// adds git failure authority codes. These distinguish
	// genuine missing objects from operational Git failures
	// (timeout, cancellation, output overflow, spawn failure)
	// so the CLI can render the right remediation hint.
	V2CodeGitTimeout           V2DiagnosticCode = "git_timeout"
	V2CodeGitCancelled         V2DiagnosticCode = "git_cancelled"
	V2CodeGitOutputOverflow    V2DiagnosticCode = "git_output_overflow"
	V2CodeGitSpawnFailed       V2DiagnosticCode = "git_spawn_failed"
	V2CodeGitNotRepository     V2DiagnosticCode = "git_not_repository"
	V2CodeGitPermissionDenied  V2DiagnosticCode = "git_permission_denied"
	V2CodeGitMalformedRevision V2DiagnosticCode = "git_malformed_revision"
	// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1
	// adds fail-closed snapshot codes. Each code is emitted
	// when the corresponding Git observation cannot be
	// obtained; the runner refuses to claim success and
	// reports the exact field that failed.
	V2CodeCallerStateUnavailable       V2DiagnosticCode = "caller_state_unavailable"
	V2CodeWorktreeInventoryUnavailable V2DiagnosticCode = "worktree_inventory_unavailable"
	// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2C-R4
	// adds object-format policy codes. The verifier MUST
	// reject before any OID validation when the resolver
	// cannot produce a storage format, or when the format
	// is not "sha1". The two codes keep the failure modes
	// distinguishable: an observation failure is not the
	// same as an unsupported repository state.
	V2CodeObjectFormatUnavailable V2DiagnosticCode = "object_format_unavailable"
	V2CodeUnsupportedObjectFormat V2DiagnosticCode = "unsupported_object_format"
	// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-AUTHORITY01
	// (R6-A) adds the subject-observation authority code. Every
	// observation that can only be made while the live detached
	// subject worktree exists (HEAD, HEAD^{tree}, show-toplevel,
	// detached, status, refs, worktree inventory) routes through
	// this code on failure so the CLI can render the exact field
	// that could not be observed.
	V2CodeSubjectObservationUnavailable V2DiagnosticCode = "subject_observation_unavailable"
	// R6-A adds a registration-mismatch code for Phase 8: when
	// the live AtSubject inventory contains a row for the
	// captured worktree path but the row's HEAD does not match
	// the requested subject commit, the executor fails closed
	// with this code so downstream code cannot silently accept a
	// mismatched registration.
	V2CodeSubjectRegistrationMismatch V2DiagnosticCode = "subject_registration_mismatch"
	// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-BINARY-GATE-INTEGRATION01-CORRECTION06
	// adds the R6-B fail-closed surface codes. Each code names
	// a single authority the integration owns; downstream code
	// (CLI, B2 barrier, R6-C) routes remediation through the
	// exact code so the proof can fail-closed at the right layer.
	V2CodeR6BBinaryAuthorityInvalid        V2DiagnosticCode = "r6b_binary_authority_invalid"
	V2CodeR6BGateObservationFailed         V2DiagnosticCode = "r6b_gate_observation_failed"
	V2CodeR6BGateClassificationFailed      V2DiagnosticCode = "r6b_gate_classification_failed"
	V2CodeR6BGateClassificationUnavailable V2DiagnosticCode = "r6b_gate_classification_unavailable"
	V2CodeR6BSubjectCleanupFailed          V2DiagnosticCode = "r6b_subject_cleanup_failed"
	V2CodeR6BSubjectCleanupUnavailable     V2DiagnosticCode = "r6b_subject_cleanup_unavailable"
)

// V2Diagnostic is the structured diagnostic record emitted by
// the Closure Protocol v2 runner. The Code field is the only
// machine-identifier; Message and Detail are human-readable
// and may evolve. PropertyName is the plan-declaration
// property path or runtime-identity label that produced the
// diagnostic; it is empty when no such anchor exists.
type V2Diagnostic struct {
	Code         V2DiagnosticCode `json:"code"`
	Message      string           `json:"message"`
	PropertyName string           `json:"property_name,omitempty"`
	Detail       string           `json:"detail,omitempty"`
}

// V2Diagnostics is an ordered list of V2Diagnostic. Order is
// preserved so test failures and logs remain deterministic.
type V2Diagnostics []V2Diagnostic

// HasCode reports whether any diagnostic carries the given code.
func (d V2Diagnostics) HasCode(code V2DiagnosticCode) bool {
	for _, item := range d {
		if item.Code == code {
			return true
		}
	}
	return false
}

// Codes returns the deduplicated code list, preserving first
// occurrence order. The returned slice is safe for tests that
// assert on a closed set of failure codes.
func (d V2Diagnostics) Codes() []V2DiagnosticCode {
	seen := make(map[V2DiagnosticCode]bool)
	out := make([]V2DiagnosticCode, 0, len(d))
	for _, item := range d {
		if seen[item.Code] {
			continue
		}
		seen[item.Code] = true
		out = append(out, item.Code)
	}
	return out
}

// V2Error wraps a non-empty diagnostic list with a Go error
// so callers can choose between typed inspection (Diags) and
// the standard error interface (Error).
//
// R2C-R4: when the runner encounters a non-*V2Error failure
// from an inner authority (e.g. a plain error from the
// executor), it retains the original error as Cause so the
// wrapped error remains discoverable via errors.Is and
// errors.As through Unwrap. The Diags list carries the
// deterministic first diagnostic for the inner failure and
// any appended post-availability or drift diagnostics.
type V2Error struct {
	Diags V2Diagnostics
	// Cause is the underlying error the runner wrapped,
	// when the original failure was not already a typed
	// V2Error. It is nil when the original failure was
	// already a *V2Error. Unwrap returns Cause so the
	// standard errors.Is / errors.As helpers work.
	Cause error
}

func (e *V2Error) Error() string {
	if e == nil || len(e.Diags) == 0 {
		return "closure protocol v2 failure"
	}
	first := e.Diags[0]
	if first.Message == "" {
		return string(first.Code)
	}
	return string(first.Code) + ": " + first.Message
}

// Unwrap returns the wrapped cause so errors.Is and
// errors.As can reach the original error. It returns nil
// when there is no cause (e.g. the original failure was
// already a typed *V2Error).
func (e *V2Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewV2Error constructs a V2Error carrying a single code and
// human-readable message. PropertyName and Detail are optional.
func NewV2Error(code V2DiagnosticCode, message string) *V2Error {
	return &V2Error{Diags: V2Diagnostics{{Code: code, Message: message}}}
}

// NewV2ErrorWith builds a V2Error with optional property name
// and detail fields populated.
func NewV2ErrorWith(code V2DiagnosticCode, message, property, detail string) *V2Error {
	return &V2Error{Diags: V2Diagnostics{{
		Code:         code,
		Message:      message,
		PropertyName: property,
		Detail:       detail,
	}}}
}
