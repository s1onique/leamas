// SPDX-License-Identifier: Apache-2.0

// binary_gate_validation.go owns the pure R6-B fail-closed
// validators. Each validator returns a typed *V2Error on the
// first failure it finds; the runner adapter wraps the
// returned error so the published surface is the typed
// V2DiagnosticCode.
//
// Splitting this from closure_evidence_publication_runner_adapter.go
// keeps the adapter under the LLM-friendly 400-line threshold
// while preserving the single closure over the descriptor
// that ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.
//
// The validators do NOT recompute any production authority.
// They are consumer-side contract checks; the authorities
// remain BuildExactSubjectBinary, GateCapture, the
// ClassifyACTOwnedGate verdict, and the R6-A
// SubjectCleanupObserved/SubjectCleanupError fields.

package closure

import (
	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// validateExactSubjectBinaryResult checks that the B1 result
// corresponds to the requested authority. The check is a
// consumer-side contract assertion; it never duplicates B1
// implementation internals.
//
// Returns nil when the result is structurally sound for the
// request. Returns a typed *V2Error with V2CodeR6BBinaryAuthorityInvalid
// when any field disagrees with the request.
//
// The expected literal S / S^{tree} OIDs are passed
// explicitly so the validator cannot be fooled by empty
// B1-result fields. CORRECTION06 requires each identity
// field to be NON-EMPTY and to match the resolved OID; the
// previous version allowed empty values, which masked a
// real production correctness issue.
func validateExactSubjectBinaryResult(
	expectedCommitOID, expectedTreeOID string,
	res ExactSubjectBinaryResult,
) *V2Error {
	if res.BinaryPath == "" {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary result has empty BinaryPath",
			"binary_path", "")
	}
	if !validBinarySHA256(res.BinarySHA256) {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary result has invalid BinarySHA256",
			"binary_sha256", res.BinarySHA256)
	}
	if res.BinaryCommit == "" {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary result has empty BinaryCommit",
			"binary_commit", "")
	}
	if res.BinaryCommit != expectedCommitOID {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary BinaryCommit does not match the resolved subject commit",
			"binary_commit",
			"binary_commit="+res.BinaryCommit+" subject_commit="+expectedCommitOID)
	}
	if res.BinaryModified {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary result has BinaryModified=true",
			"binary_modified", "")
	}
	if !res.SourceClean {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary result has SourceClean=false",
			"source_clean", "")
	}
	if !res.SourceDetached {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary result has SourceDetached=false",
			"source_detached", "")
	}
	if !res.OutputOutsideAllWorktrees {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary result has OutputOutsideAllWorktrees=false",
			"output_outside_all_worktrees", "")
	}
	if !res.Executable {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary result has Executable=false",
			"executable", "")
	}
	if res.SourceCommit == "" {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary result has empty SourceCommit",
			"source_commit", "")
	}
	if res.SourceCommit != expectedCommitOID {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary SourceCommit does not match the resolved subject commit",
			"source_commit",
			"source_commit="+res.SourceCommit+" subject_commit="+expectedCommitOID)
	}
	if res.SourceTree == "" {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary result has empty SourceTree",
			"source_tree", "")
	}
	if res.SourceTree != expectedTreeOID {
		return NewV2ErrorWith(V2CodeR6BBinaryAuthorityInvalid,
			"BuildExactSubjectBinary SourceTree does not match the resolved subject tree",
			"source_tree",
			"source_tree="+res.SourceTree+" subject_tree="+expectedTreeOID)
	}
	return nil
}

// classifyCapturedGate routes the captured GateCapture through
// the canonical ClassifyACTOwnedGate verdict. The function is
// a thin package-private seam so tests can replace the
// classifier with a counting wrapper.
//
// Returns (verdict, classifiedInputs) for diagnostics. The
// production classifier is the only authority for the verdict.
func classifyCapturedGate(capture evidence.GateCapture,
	ownedPaths []string, baseline []evidence.GateFinding,
	classifierFn func(evidence.ClassificationInputs) evidence.ACTOwnedClassification,
) (evidence.ACTOwnedClassification, evidence.ClassificationInputs) {
	if classifierFn == nil {
		classifierFn = evidence.ClassifyACTOwnedGate
	}
	inputs := evidence.ClassificationInputs{
		ObservedStatus:   capture.ExecGateObservedStatus,
		ObservedFindings: capture.PreExistingFindings,
		BaselineFindings: baseline,
		ACTOwnedPaths:    ownedPaths,
		LaneMissing:      capture.ExecGateObservedStatus == "",
		LaneTimedOut:     capture.TimedOut,
		LaneTruncated:    capture.StdoutTruncated || capture.StderrTruncated,
	}
	return classifierFn(inputs), inputs
}

// validateSubjectCleanupOutcome inspects the R6-A subject
// cleanup fields the executor captured and converts the
// observation status into a typed V2Error when the cleanup
// authority is incomplete or failed.
//
// The fields are the canonical R6-A subject cleanup authority.
// The integration MUST surface a typed error when the
// executor either did not attempt cleanup or reported a
// failure, so callers can distinguish the two states.
func validateSubjectCleanupOutcome(result V2ExecuteResult) *V2Error {
	if !result.SubjectCleanupObserved {
		return NewV2ErrorWith(V2CodeR6BSubjectCleanupUnavailable,
			"subject cleanup was not observed by the executor",
			"subject_cleanup_observed", "")
	}
	if result.SubjectCleanupError != "" {
		return NewV2ErrorWith(V2CodeR6BSubjectCleanupFailed,
			"subject cleanup reported an error after the gate had run",
			"subject_cleanup_error", result.SubjectCleanupError)
	}
	return nil
}

// validBinarySHA256 reports whether s is a 64-character
// lowercase hex string. SHA-256 digests are exactly 64 hex
// characters; anything else is a malformed B1 result.
func validBinarySHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}
