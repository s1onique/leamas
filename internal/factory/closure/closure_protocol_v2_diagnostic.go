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
	V2CodeEvidencePathNotDetached            V2DiagnosticCode = "evidence_path_not_detached"
	V2CodeManifestPathNotDetached            V2DiagnosticCode = "manifest_path_not_detached"
	V2CodeWorkingPlanPathInvalid             V2DiagnosticCode = "working_plan_path_invalid"
	V2CodeCallerWorktreeDirty               V2DiagnosticCode = "caller_worktree_dirty"
	V2CodeExecutionTreeMismatch             V2DiagnosticCode = "execution_tree_mismatch"
	V2CodeGitOperationFailed                V2DiagnosticCode = "git_operation_failed"
	V2CodeExecutionFailed                   V2DiagnosticCode = "execution_failed"
	V2CodeCleanupFailed                     V2DiagnosticCode = "cleanup_failed"
	V2CodeManifestWriteFailed               V2DiagnosticCode = "manifest_write_failed"
	V2CodeRequestIncomplete                 V2DiagnosticCode = "request_incomplete"
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
type V2Error struct {
	Diags V2Diagnostics
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
