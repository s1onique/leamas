// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence.go defines the canonical
// closure evidence record for ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-
// CONTEXT-AND-EXECUTE01-CORRECTION02-B2.
//
// The file is the single source of truth for the evidence shape.
// There is exactly one canonical type (ClosureEvidence), one
// canonical completeness predicate (DeriveClosureEvidenceCompleteness)
// and one canonical validation entry point (ValidateClosureEvidence).
// The parallel `Ex` model that previously co-existed with
// ClosureEvidence has been removed: an authority cannot be carried
// in a struct that is not the canonical evidence record.
//
// Completeness is never stored on the struct. The single
// canonical predicate re-derives COMPLETE / INCOMPLETE from the
// authorities every time. Callers cannot force COMPLETE by
// assigning a field.
//
// Splitting the file at the type boundary keeps each file under
// the LLM-friendly 400-line threshold while preserving the single
// closure over the descriptor required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.
package evidence

import (
	"errors"
	"fmt"
)

// ClosureEvidenceSchemaVersion is the schema identifier. The
// version is bumped when the canonical struct shape changes.
// B2-R1 bumped the version from 3 to 4: GateAuthority gained
// the SubjectExecutionRoot field, and the strict evidence
// decoder rejects unknown fields.
const ClosureEvidenceSchemaVersion = 4

// ClosureProtocolVersion is the published protocol identifier
// that pairs with the schema version. The string value is the
// canonical JSON token, so machine handling never has to parse
// the message text.
const ClosureProtocolVersion = "v2"

// EvidenceCompleteness is the derived verdict returned by the
// canonical predicate. The string values are the only valid
// tokens; the struct itself never carries a Completeness field.
type EvidenceCompleteness string

const (
	// EvidenceIncomplete is the verdict when ANY predicate in
	// the closed AND is false. The evidence MUST NOT cross the
	// publication barrier.
	EvidenceIncomplete EvidenceCompleteness = "INCOMPLETE"
	// EvidenceComplete is the verdict when every predicate in
	// the closed AND is true. Only after the predicate returns
	// EvidenceComplete may the barrier produce publication bytes.
	EvidenceComplete EvidenceCompleteness = "COMPLETE"
)

// RuntimeAuthority captures the immutable execution identity.
// Every field is a fact observed by the runner; an empty value
// means the observation failed or was never performed.
//
// Path vs OID: SubjectExecutionRoot is the absolute filesystem
// path of the detached subject worktree (where the gate ran
// and the checks executed). SubjectTree is the Git tree OID of
// the same worktree (what immutable content was in scope).
// ExecutionTree is the runner's observed tree OID (which must
// equal SubjectTree). The namespaces are deliberately
// separate: a path is "where" and an OID is "what".
type RuntimeAuthority struct {
	RepositoryRoot string `json:"repository_root"`
	FreezeCommit   string `json:"freeze_commit"`
	FreezeTree     string `json:"freeze_tree"`
	SubjectCommit  string `json:"subject_commit"`
	SubjectTree    string `json:"subject_tree"`
	// SubjectExecutionRoot is the absolute filesystem path of
	// the detached subject worktree. The GateAuthority compares
	// its SubjectRoot and SubjectExecutionRoot fields to this
	// path; comparing a path to a Git OID is a type error that
	// B2-R1 hid with a test fixture that used the OID in both
	// fields. B2-R2 separates the two namespaces.
	SubjectExecutionRoot string `json:"subject_execution_root"`
	ExecutionTree        string `json:"execution_tree"`
	PlanPath             string `json:"plan_path"`
	PlanBlob             string `json:"plan_blob"`
	PlanSHA256           string `json:"plan_sha256"`
	PlanBytes            []byte `json:"plan_bytes,omitempty"`
	// FAncestorOfSVerified records that the runner topology
	// authority has proven freeze_commit is an ancestor of
	// subject_commit. The runtime identity fields alone cannot
	// prove ancestorship; the runner must observe it. The
	// candidate builder is the only writer.
	FAncestorOfSVerified bool `json:"f_ancestor_of_s_verified"`
}

// PlanAuthority captures the expected check set the frozen plan
// declared. Each entry fixes the check ID and the mode the
// runner MUST execute it in.
type PlanAuthority struct {
	ExpectedChecks []PlanCheckSpec `json:"expected_checks"`
}

// PlanCheckSpec is one immutable expected check.
type PlanCheckSpec struct {
	ID   string `json:"id"`
	Mode string `json:"mode"`
}

// CheckResult captures the typed outcome of one executed check.
// Mode is "run" or "exclude". Outcome is "pass", "fail",
// "timeout", "canceled", or "excluded".
type CheckResult struct {
	CheckID         string `json:"check_id"`
	Mode            string `json:"mode"`
	Outcome         string `json:"outcome"`
	ExitCode        int    `json:"exit_code"`
	TimedOut        bool   `json:"timed_out"`
	Canceled        bool   `json:"canceled"`
	StdoutTruncated bool   `json:"stdout_truncated"`
	StderrTruncated bool   `json:"stderr_truncated"`
	CleanupError    string `json:"cleanup_error,omitempty"`
}

// GateAuthority captures the gate (GateCollector) authority. The
// invocation_count is the collector's CallsCount; the predicate
// accepts exactly one.
//
// SubjectRoot is the worktree path the gate ran against. The
// B2-R1 matrix binds the gate to the actual subject execution
// root by recording that root in SubjectExecutionRoot and
// requiring the two are equal. Without
// SubjectExecutionRoot, a caller could populate SubjectRoot
// with any path and the predicate would accept it.
// GateAuthority fields are aligned for the widest tag.
type GateAuthority struct {
	ObservedStatus       string `json:"observed_status"`
	Classification       string `json:"classification"`
	InvocationCount      int    `json:"invocation_count"`
	RepositoryRoot       string `json:"repository_root"`
	SubjectRoot          string `json:"subject_root"`
	SubjectExecutionRoot string `json:"subject_execution_root"`
	TimedOut             bool   `json:"timed_out"`
	StdoutTruncated      bool   `json:"stdout_truncated"`
	StderrTruncated      bool   `json:"stderr_truncated"`
	Error                string `json:"error,omitempty"`
}

// BinaryAuthority captures the exact-S binary authority. The
// fields map 1:1 to the B2 specification. The runner is the
// only writer; the candidate builder converts from the build
// observability (VCSRevision / VCSModified) into the canonical
// authority (BinaryCommit / BinaryModified).
type BinaryAuthority struct {
	BinaryPath                string `json:"binary_path"`
	BinarySHA256              string `json:"binary_sha256"`
	BinaryCommit              string `json:"binary_commit"`
	BinaryModified            bool   `json:"binary_modified"`
	SourceCommit              string `json:"source_commit"`
	SourceTree                string `json:"source_tree"`
	SourceClean               bool   `json:"source_clean"`
	SourceDetached            bool   `json:"source_detached"`
	OutputOutsideAllWorktrees bool   `json:"output_outside_all_worktrees"`
	Executable                bool   `json:"executable"`
}

// CallerStateSnapshot captures one observable snapshot of the
// caller worktree. The BEFORE / AFTER pair diffs across the
// runner. Available distinguishes "the snapshot was taken"
// from "the snapshot was empty".
type CallerStateSnapshot struct {
	Available             bool   `json:"available"`
	Head                  string `json:"head,omitempty"`
	Tree                  string `json:"tree,omitempty"`
	StatusHash            string `json:"status_hash,omitempty"`
	RefsHash              string `json:"refs_hash,omitempty"`
	WorktreeInventoryHash string `json:"worktree_inventory_hash,omitempty"`
}

// CleanupAuthority captures the bounded-cleanup outcome for the
// subject execution and the binary build. A non-empty field is
// the typeless error string the runner observed.
type CleanupAuthority struct {
	SubjectCleanupError string `json:"subject_cleanup_error,omitempty"`
	BinaryCleanupError  string `json:"binary_cleanup_error,omitempty"`
}

// ClosureEvidence is the canonical publication record. The
// struct intentionally has NO Completeness field: the verdict
// is always derived from the authorities, never stored. The
// JSON serialization is deterministic by Go's struct field
// declaration order.
type ClosureEvidence struct {
	SchemaVersion int                 `json:"schema_version"`
	Protocol      string              `json:"protocol"`
	Runtime       RuntimeAuthority    `json:"runtime"`
	Plan          PlanAuthority       `json:"plan"`
	Results       []CheckResult       `json:"results"`
	Gate          GateAuthority       `json:"gate"`
	Binary        BinaryAuthority     `json:"binary"`
	CallerBefore  CallerStateSnapshot `json:"caller_before"`
	CallerAfter   CallerStateSnapshot `json:"caller_after"`
	Cleanup       CleanupAuthority    `json:"cleanup"`
}

// ValidateClosureEvidence is the structural validator. It
// rejects documents whose schema is unsupported and whose
// fields are not well-formed. The function does NOT derive
// completeness; that is the responsibility of the canonical
// predicate alone. The barrier calls both: validate first
// (cheap), then derive.
func ValidateClosureEvidence(candidate ClosureEvidence) error {
	if candidate.SchemaVersion != ClosureEvidenceSchemaVersion {
		return fmt.Errorf("evidence: schema_version %d is not supported (expected %d)",
			candidate.SchemaVersion, ClosureEvidenceSchemaVersion)
	}
	if candidate.Protocol != ClosureProtocolVersion {
		return fmt.Errorf("evidence: protocol %q is not supported (expected %q)",
			candidate.Protocol, ClosureProtocolVersion)
	}
	if stringsOrEmpty(candidate.Runtime.RepositoryRoot) == "" {
		return errors.New("evidence: runtime.repository_root is empty")
	}
	if !isValidOID(candidate.Runtime.FreezeCommit) {
		return errors.New("evidence: runtime.freeze_commit is not a 40-char hex OID")
	}
	if !isValidOID(candidate.Runtime.FreezeTree) {
		return errors.New("evidence: runtime.freeze_tree is not a 40-char hex OID")
	}
	if !isValidOID(candidate.Runtime.SubjectCommit) {
		return errors.New("evidence: runtime.subject_commit is not a 40-char hex OID")
	}
	if !isValidOID(candidate.Runtime.SubjectTree) {
		return errors.New("evidence: runtime.subject_tree is not a 40-char hex OID")
	}
	if !isValidOID(candidate.Runtime.ExecutionTree) {
		return errors.New("evidence: runtime.execution_tree is not a 40-char hex OID")
	}
	if candidate.Runtime.PlanPath == "" {
		return errors.New("evidence: runtime.plan_path is empty")
	}
	if !isValidOID(candidate.Runtime.PlanBlob) {
		return errors.New("evidence: runtime.plan_blob is not a 40-char hex OID")
	}
	if !isHexSHA256(candidate.Runtime.PlanSHA256) {
		return errors.New("evidence: runtime.plan_sha256 is not a 64-char hex digest")
	}
	if len(candidate.Plan.ExpectedChecks) == 0 {
		return errors.New("evidence: plan.expected_checks is empty")
	}
	for i, c := range candidate.Plan.ExpectedChecks {
		if c.ID == "" {
			return fmt.Errorf("evidence: plan.expected_checks[%d].id is empty", i)
		}
		if c.Mode != "run" && c.Mode != "exclude" {
			return fmt.Errorf("evidence: plan.expected_checks[%d].mode is %q (expected run|exclude)", i, c.Mode)
		}
	}
	if candidate.Binary.BinaryPath == "" {
		return errors.New("evidence: binary.binary_path is empty")
	}
	if !isHexSHA256(candidate.Binary.BinarySHA256) {
		return errors.New("evidence: binary.binary_sha256 is not a 64-char hex digest")
	}
	if !isValidOID(candidate.Binary.BinaryCommit) {
		return errors.New("evidence: binary.binary_commit is not a 40-char hex OID")
	}
	if !isValidOID(candidate.Binary.SourceCommit) {
		return errors.New("evidence: binary.source_commit is not a 40-char hex OID")
	}
	if !isValidOID(candidate.Binary.SourceTree) {
		return errors.New("evidence: binary.source_tree is not a 40-char hex OID")
	}
	if candidate.Gate.RepositoryRoot == "" {
		return errors.New("evidence: gate.repository_root is empty")
	}
	if candidate.Gate.SubjectRoot == "" {
		return errors.New("evidence: gate.subject_root is empty")
	}
	if candidate.Gate.SubjectExecutionRoot == "" {
		return errors.New("evidence: gate.subject_execution_root is empty")
	}
	if candidate.Gate.Classification == "" {
		return errors.New("evidence: gate.classification is empty")
	}
	return nil
}

// stringsOrEmpty is the small helper that returns the input
// string verbatim. It exists so the file can be split later
// without changing call sites.
func stringsOrEmpty(s string) string {
	return s
}
