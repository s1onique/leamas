// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_completeness_predicates_runtime.go
// owns the runtime predicates (1..7) and the plan/result /
// results predicates (8..19) of the canonical completeness
// matrix.
//
// The gate, binary, and caller predicates live in
// closure_evidence_completeness_predicates.go. Splitting the
// per-predicate functions across multiple files keeps each
// file under the LLM-friendly 400-line threshold while
// preserving the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
)

// ----------------------------------------------------------------------------
// Runtime predicates (1..7)
// ----------------------------------------------------------------------------

// runtimeIdentitiesStructurallyValid reports whether every
// required runtime identity field is present and well-formed.
func runtimeIdentitiesStructurallyValid(c ClosureEvidence) bool {
	r := c.Runtime
	return r.RepositoryRoot != "" &&
		isValidOID(r.FreezeCommit) &&
		isValidOID(r.FreezeTree) &&
		isValidOID(r.SubjectCommit) &&
		isValidOID(r.SubjectTree) &&
		isValidOID(r.ExecutionTree) &&
		r.PlanPath != "" &&
		isValidOID(r.PlanBlob) &&
		isHexSHA256(r.PlanSHA256)
}

// runtimeFreezeDifferentFromSubject reports F != S at the commit
// level. The verdict is INCOMPLETE when both commits are equal
// because the immutable subject execution authority requires
// F is a strict ancestor of S.
func runtimeFreezeDifferentFromSubject(c ClosureEvidence) bool {
	return c.Runtime.FreezeCommit != c.Runtime.SubjectCommit
}

// runtimeFAncestorOfSVerified reports whether the runner topology
// authority has verified freeze_commit is an ancestor of
// subject_commit. The candidate builder is the only writer.
func runtimeFAncestorOfSVerified(c ClosureEvidence) bool {
	return c.Runtime.FAncestorOfSVerified
}

// runtimeExecutionTreeEqualsSubjectTree reports that the
// observed execution tree equals the recorded subject tree.
func runtimeExecutionTreeEqualsSubjectTree(c ClosureEvidence) bool {
	return c.Runtime.ExecutionTree == c.Runtime.SubjectTree
}

// runtimePlanBlobValid reports the plan blob is a 40-char hex OID.
func runtimePlanBlobValid(c ClosureEvidence) bool {
	return isValidOID(c.Runtime.PlanBlob)
}

// runtimePlanSHA256Valid reports the plan SHA-256 is a 64-char
// hex digest.
func runtimePlanSHA256Valid(c ClosureEvidence) bool {
	return isHexSHA256(c.Runtime.PlanSHA256)
}

// runtimePlanBytesParseSuccessfully runs the production Plan
// Contract v1 decoder on c.Runtime.PlanBytes. The previous
// B2 implementation only compared bytes to the recorded
// SHA-256; arbitrary bytes with a matching SHA-256 could
// satisfy that predicate. The B2-R1 production decoder
// (parseBoundedClosurePlanDocument, mirrored in the
// evidence package's productionParseClosurePlanDocument
// to avoid a test-only import cycle) enforces the
// MaxPlanBytes cap, rejects malformed JSON, and refuses
// trailing values so the bytes MUST be a real Plan
// Contract document.
//
// The predicate also verifies the SHA-256 still matches the
// recorded PlanSHA256 because the runtime identity is
// authoritative, and rejects any well-formed Plan whose
// declared SHA-256 differs from the recorded one.
func runtimePlanBytesParseSuccessfully(c ClosureEvidence) bool {
	if len(c.Runtime.PlanBytes) == 0 {
		return false
	}
	sum := sha256.Sum256(c.Runtime.PlanBytes)
	if hex.EncodeToString(sum[:]) != c.Runtime.PlanSHA256 {
		return false
	}
	_, err := productionDecodeClosurePlan(c.Runtime.PlanBytes)
	return err == nil
}

// runtimeExpectedChecksDerivedFromPlanBytes reports that the
// candidate's Plan.ExpectedChecks agree with the production
// Plan Contract decoder applied to Runtime.PlanBytes. The
// previous B2 implementation accepted the supplied
// ExpectedChecks independently; the B2-R1 contract derives
// the expected set from the production decode so the candidate
// cannot smuggle in an alternative check set.
func runtimeExpectedChecksDerivedFromPlanBytes(c ClosureEvidence) bool {
	if len(c.Runtime.PlanBytes) == 0 {
		return false
	}
	derived, err := productionDecodeClosurePlan(c.Runtime.PlanBytes)
	if err != nil {
		return false
	}
	if len(derived) != len(c.Plan.ExpectedChecks) {
		return false
	}
	for i, want := range derived {
		got := c.Plan.ExpectedChecks[i]
		if got.ID != want.ID || got.Mode != want.Mode {
			return false
		}
	}
	return true
}

// ----------------------------------------------------------------------------
// Plan / result bijection predicates (8..12)
// ----------------------------------------------------------------------------

// planResultCardinalityEqual reports plan and result lists have
// the same length.
func planResultCardinalityEqual(c ClosureEvidence) bool {
	return len(c.Plan.ExpectedChecks) == len(c.Results)
}

// planResultIDsBijective reports the result ID set is exactly
// the plan ID set, with no duplicates and no gaps.
func planResultIDsBijective(c ClosureEvidence) bool {
	if len(c.Plan.ExpectedChecks) == 0 {
		return false
	}
	expected := make(map[string]int, len(c.Plan.ExpectedChecks))
	for _, p := range c.Plan.ExpectedChecks {
		expected[p.ID]++
	}
	if len(c.Results) == 0 {
		return false
	}
	seen := make(map[string]int, len(c.Results))
	for _, r := range c.Results {
		seen[r.CheckID]++
	}
	if len(seen) != len(expected) {
		return false
	}
	for id, n := range seen {
		if n != 1 {
			return false
		}
		if expected[id] != 1 {
			return false
		}
	}
	return true
}

// planResultOrderMatchesPlan reports the result order equals
// the plan order (no silent reordering).
func planResultOrderMatchesPlan(c ClosureEvidence) bool {
	if len(c.Plan.ExpectedChecks) != len(c.Results) {
		return false
	}
	for i, p := range c.Plan.ExpectedChecks {
		if c.Results[i].CheckID != p.ID {
			return false
		}
	}
	return true
}

// planResultModeMatchesPlanMode reports every result mode
// matches the corresponding plan mode.
func planResultModeMatchesPlanMode(c ClosureEvidence) bool {
	if len(c.Plan.ExpectedChecks) != len(c.Results) {
		return false
	}
	for i, p := range c.Plan.ExpectedChecks {
		if c.Results[i].Mode != p.Mode {
			return false
		}
	}
	return true
}

// planNoUnknownCheckMode rejects any check whose mode is not
// "run" or "exclude".
func planNoUnknownCheckMode(c ClosureEvidence) bool {
	for _, p := range c.Plan.ExpectedChecks {
		if p.Mode != "run" && p.Mode != "exclude" {
			return false
		}
	}
	for _, r := range c.Results {
		if r.Mode != "run" && r.Mode != "exclude" {
			return false
		}
	}
	return true
}

// ----------------------------------------------------------------------------
// Results predicates (13..19)
// ----------------------------------------------------------------------------

// resultsEveryRunCheckSuccessful reports every "run" check
// produced a pass outcome with no timeout / cancel / cleanup
// error.
func resultsEveryRunCheckSuccessful(c ClosureEvidence) bool {
	for _, r := range c.Results {
		if r.Mode != "run" {
			continue
		}
		if r.Outcome != "pass" {
			return false
		}
		if r.TimedOut || r.Canceled || r.CleanupError != "" {
			return false
		}
	}
	return true
}

// resultsEveryExcludeCheckExcluded reports every "exclude"
// check produced an "excluded" outcome with no exit / cleanup.
func resultsEveryExcludeCheckExcluded(c ClosureEvidence) bool {
	for _, r := range c.Results {
		if r.Mode != "exclude" {
			continue
		}
		if r.Outcome != "excluded" {
			return false
		}
		if r.ExitCode != 0 || r.TimedOut || r.Canceled || r.CleanupError != "" {
			return false
		}
	}
	return true
}

// resultsNoTimeout reports no check timed out.
func resultsNoTimeout(c ClosureEvidence) bool {
	for _, r := range c.Results {
		if r.TimedOut {
			return false
		}
	}
	return true
}

// resultsNoCancellation reports no check was cancelled.
func resultsNoCancellation(c ClosureEvidence) bool {
	for _, r := range c.Results {
		if r.Canceled {
			return false
		}
	}
	return true
}

// resultsNoStdoutTruncation reports no check stdout was truncated.
func resultsNoStdoutTruncation(c ClosureEvidence) bool {
	for _, r := range c.Results {
		if r.StdoutTruncated {
			return false
		}
	}
	return true
}

// resultsNoStderrTruncation reports no check stderr was truncated.
func resultsNoStderrTruncation(c ClosureEvidence) bool {
	for _, r := range c.Results {
		if r.StderrTruncated {
			return false
		}
	}
	return true
}

// resultsNoExecutionCleanupError reports the subject execution
// cleanup reported no error.
func resultsNoExecutionCleanupError(c ClosureEvidence) bool {
	return c.Cleanup.SubjectCleanupError == ""
}
