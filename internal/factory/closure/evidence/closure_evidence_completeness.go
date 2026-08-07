// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_completeness.go extends
// ClosureEvidence with the runtime authority fields and
// implements the real DeriveClosureEvidenceCompleteness
// required by Phase 8 of
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02.
//
// COMPLETE requires every predicate below to be true:
//
//   - runtime authority valid (RuntimeAuthorityValid)
//   - exact F:P bytes valid (FrozenPlanBytesValid)
//   - subject worktree tree == S^{tree} (SubjectWorktreeMatchesTree)
//   - check/result bijection valid (CheckResultBijectionValid)
//   - all required run checks successful (AllRunChecksSuccessful)
//   - exclude checks unexecuted (ExcludeChecksUnexecuted)
//   - no timeout (NoTimeout)
//   - no cancellation (NoCancellation)
//   - no truncation (NoTruncation)
//   - no cleanup error (NoCleanupError)
//   - gate classification PASS (GateClassificationPASS)
//   - binary authority valid (BinaryAuthorityValid)
//   - BEFORE state available (BeforeStateAvailable)
//   - AFTER state available (AfterStateAvailable)
//   - caller state unchanged (CallerStateUnchanged)
//   - worktree inventory unchanged (WorktreeInventoryUnchanged)
//
// Mutation-test every predicate independently via
// TestClosureEvidenceCompletenessDerived.
//
// Splitting this from closure_evidence.go keeps both files
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
)

// RuntimeAuthorityRecord captures the runtime authority
// observations the runner must record before declaring
// COMPLETE. Every field is fail-closed: an empty value means
// the observation failed or was never performed.
type RuntimeAuthorityRecord struct {
	RepositoryRoot              string
	SubjectCommit               string
	SubjectTree                 string
	FreezeCommit                string
	FreezeTree                  string
	PlanPath                    string
	PlanBlob                    string
	PlanSHA256                  string
	PlanBytes                   []byte
	EvidenceDirectory           string
	StartedAt                   string
	SubjectWorktreeRoot         string
	SubjectWorktreeObservedTree string
	ExecutionTree               string
	CallerHEAD                  string
	CallerHEADTree              string
	CallerRefsHash              string
	PlanCheckIDs                []string
	PlanCheckModes              []string
}

// CheckResultRecord captures the typed outcome of one frozen
// plan check. Mode is "run" or "exclude". Outcome is "pass",
// "fail", "timeout", "canceled", or "excluded".
type CheckResultRecord struct {
	CheckID         string
	Mode            string
	Outcome         string
	ExitCode        int
	TimedOut        bool
	Canceled        bool
	StdoutTruncated bool
	StderrTruncated bool
	CleanupError    string
}

// GateClassificationRecord captures the gate classification
// inputs and verdict. Verdict is "PASS", "FAIL", or "UNAVAILABLE".
type GateClassificationRecord struct {
	Verdict              string
	ObservedFindings     []GateFinding
	BaselineFindings     []GateFinding
	ACTOwnedPaths        []string
	LaneMissing          []string
	LaneTimedOut         []string
	LaneTruncated        []string
	ClassificationInputs ClassificationInputs
}

// CallerStateAvailability records whether the BEFORE and
// AFTER caller-state observations were Available.
type CallerStateAvailability struct {
	BeforeAvailable bool
	AfterAvailable  bool
}

// CallerStateDrift records whether the BEFORE and AFTER
// caller-state observations were equal.
type CallerStateDrift struct {
	HEADChanged    bool
	TreeChanged    bool
	StatusChanged  bool
	RefsChanged    bool
	WorktreeLeaked bool
}

// CompletenessAuthorities bundles every observation the
// predicate derives from. A value of this type MUST be
// constructed by the runner; the evidence document alone is
// never enough to derive COMPLETE.
type CompletenessAuthorities struct {
	Runtime         RuntimeAuthorityRecord
	Checks          []CheckResultRecord
	Gate            GateClassificationRecord
	Binary          BuiltBinaryEvidence
	CallerAvailable CallerStateAvailability
	CallerDrift     CallerStateDrift
}

// ClosureEvidenceEx extends ClosureEvidence with the runtime
// authority and observation records the predicate consumes.
// SchemaVersion is bumped to 2 so consumers can detect the new
// shape.
type ClosureEvidenceEx struct {
	SchemaVersion     int                     `json:"schema_version"`
	Runtime           RuntimeContextSubset    `json:"runtime"`
	Gate              GateCapture             `json:"gate"`
	Binary            BuiltBinaryEvidence     `json:"binary"`
	Checks            []CheckEvidence         `json:"checks"`
	CallerStateBefore CallerState             `json:"caller_state_before"`
	CallerStateAfter  CallerState             `json:"caller_state_after"`
	Authorities       CompletenessAuthorities `json:"authorities"`
	Completeness      EvidenceCompleteness    `json:"completeness"`
}

// DeriveClosureEvidenceCompletenessEx derives the COMPLETE /
// INCOMPLETE verdict from the supplied authorities. Callers
// cannot force COMPLETE: the predicate is a closed AND of
// every required observation. A nil authority or missing
// observation collapses to INCOMPLETE.
//
// Every predicate is exposed as a separate function so each
// branch can be mutation-tested independently by
// TestClosureEvidenceCompletenessDerived.
func DeriveClosureEvidenceCompletenessEx(evidence ClosureEvidenceEx) EvidenceCompleteness {
	if !evidence.RuntimeAuthorityValid() {
		return EvidenceIncomplete
	}
	if !evidence.FrozenPlanBytesValid() {
		return EvidenceIncomplete
	}
	if !evidence.SubjectWorktreeMatchesTree() {
		return EvidenceIncomplete
	}
	if !evidence.CheckResultBijectionValid() {
		return EvidenceIncomplete
	}
	if !evidence.AllRunChecksSuccessful() {
		return EvidenceIncomplete
	}
	if !evidence.ExcludeChecksUnexecuted() {
		return EvidenceIncomplete
	}
	if !evidence.NoTimeout() {
		return EvidenceIncomplete
	}
	if !evidence.NoCancellation() {
		return EvidenceIncomplete
	}
	if !evidence.NoTruncation() {
		return EvidenceIncomplete
	}
	if !evidence.NoCleanupError() {
		return EvidenceIncomplete
	}
	if !evidence.GateClassificationPASS() {
		return EvidenceIncomplete
	}
	if !evidence.BinaryAuthorityValid() {
		return EvidenceIncomplete
	}
	if !evidence.BeforeStateAvailable() {
		return EvidenceIncomplete
	}
	if !evidence.AfterStateAvailable() {
		return EvidenceIncomplete
	}
	if !evidence.CallerStateUnchanged() {
		return EvidenceIncomplete
	}
	if !evidence.WorktreeInventoryUnchanged() {
		return EvidenceIncomplete
	}
	return EvidenceComplete
}

// RuntimeAuthorityValid reports whether every required runtime
// authority field is populated.
func (e ClosureEvidenceEx) RuntimeAuthorityValid() bool {
	a := e.Authorities.Runtime
	return a.RepositoryRoot != "" &&
		isValidOID(a.SubjectCommit) &&
		isValidOID(a.SubjectTree) &&
		isValidOID(a.FreezeCommit) &&
		isValidOID(a.FreezeTree) &&
		a.PlanPath != "" &&
		isValidOID(a.PlanBlob) &&
		isHexSHA256(a.PlanSHA256) &&
		a.EvidenceDirectory != "" &&
		a.SubjectWorktreeRoot != "" &&
		isValidOID(a.SubjectWorktreeObservedTree) &&
		isValidOID(a.ExecutionTree)
}

// FrozenPlanBytesValid reports whether the recorded plan
// bytes are exactly the F:P bytes and the recorded SHA-256
// matches SHA256(plan_bytes). A zero-length plan_bytes is
// rejected.
func (e ClosureEvidenceEx) FrozenPlanBytesValid() bool {
	a := e.Authorities.Runtime
	if len(a.PlanBytes) == 0 {
		return false
	}
	sum := sha256.Sum256(a.PlanBytes)
	got := hex.EncodeToString(sum[:])
	return got == a.PlanSHA256
}

// SubjectWorktreeMatchesTree reports whether the detached
// subject worktree's observed tree equals the recorded
// SubjectTree / ExecutionTree. A worktree-root that is empty
// or matches the caller repository root is rejected because
// the runner must NEVER execute in the caller checkout.
func (e ClosureEvidenceEx) SubjectWorktreeMatchesTree() bool {
	a := e.Authorities.Runtime
	if a.SubjectWorktreeRoot == "" {
		return false
	}
	if a.SubjectWorktreeRoot == a.RepositoryRoot {
		return false
	}
	if !isValidOID(a.SubjectWorktreeObservedTree) {
		return false
	}
	return a.SubjectWorktreeObservedTree == a.SubjectTree &&
		a.SubjectWorktreeObservedTree == a.ExecutionTree
}

// CheckResultBijectionValid reports whether every check in
// the recorded plan has exactly one matching result, every
// result references a check in the plan, and the result mode
// matches the plan mode. The records MUST come from the
// runner; an empty plan or empty result list collapses to
// false. A missing or unknown mode is rejected.
func (e ClosureEvidenceEx) CheckResultBijectionValid() bool {
	// planModes binds the expected check ID to the expected
	// mode. The plan authority is the only source of truth
	// for which mode each check should run in; the result
	// mode MUST match the plan mode or the bijection is
	// invalid. A plan-mode run with a result-mode exclude
	// is a silent bug and the predicate refuses to accept it.
	expected := e.Authorities.Runtime.PlanCheckIDs
	if len(expected) == 0 {
		return false
	}
	expectedModes := e.Authorities.Runtime.PlanCheckModes
	if len(expectedModes) > 0 && len(expectedModes) != len(expected) {
		return false
	}
	results := e.Authorities.Checks
	if len(results) == 0 {
		return false
	}
	if len(results) != len(expected) {
		return false
	}
	expectedSet := make(map[string]int, len(expected))
	for _, id := range expected {
		if id == "" {
			return false
		}
		expectedSet[id]++
	}
	seen := make(map[string]int, len(results))
	for i, r := range results {
		if r.CheckID == "" {
			return false
		}
		// Unknown mode is rejected. The closed set is
		// run|exclude; any other mode string is rejected.
		if r.Mode != "run" && r.Mode != "exclude" {
			return false
		}
		// Plan order must match result order so the
		// bijection is not silently reordered by an unsorted
		// runner.
		if r.CheckID != expected[i] {
			return false
		}
		// Mode binding: when the plan declares modes,
		// the result mode MUST match the plan mode.
		if len(expectedModes) > 0 && expectedModes[i] != r.Mode {
			return false
		}
		seen[r.CheckID]++
	}
	if len(seen) != len(expected) {
		return false
	}
	for id, n := range seen {
		if n != 1 {
			return false
		}
		if expectedSet[id] != 1 {
			return false
		}
	}
	return true
}
