// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_candidate.go owns the
// pure candidate construction function BuildClosureEvidenceCandidate.
//
// The function consumes typed outputs from:
//   - 02-A runtime execution (RuntimeAuthority, Results, Cleanup)
//   - GateCollector (GateAuthority)
//   - B1 exact-S binary (BinaryAuthority)
//   - caller BEFORE / AFTER (CallerStateSnapshot pair)
//
// It performs NO Git, process, or filesystem observation. The
// inputs are already-authoritative values the runner obtained
// before this function is called. The candidate is the
// single canonical object the publication barrier will consume.
//
// Splitting the file at the construction boundary keeps the
// production B2 surface under the LLM-friendly 400-line
// threshold while preserving the single closure over the
// descriptor that ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01
// requires.
package evidence

// CandidateInputs is the typed bundle the runner hands to
// BuildClosureEvidenceCandidate. Every field is a fact the
// runner has already observed; the function does not perform
// any new observation.
type CandidateInputs struct {
	// Runtime is the immutable execution identity captured
	// before any check ran. FAncestorOfSVerified must reflect
	// the runner's topology-authority verdict.
	Runtime RuntimeAuthority
	// Plan is the expected check set the frozen plan declared.
	Plan PlanAuthority
	// Results is the ordered check-result list produced by the
	// immutable subject execution. Order MUST match plan order.
	Results []CheckResult
	// Gate is the captured fast-lane gate authority.
	Gate GateAuthority
	// Binary is the exact-S binary authority built by B1.
	Binary BinaryAuthority
	// CallerBefore is the BEFORE caller-state snapshot.
	CallerBefore CallerStateSnapshot
	// CallerAfter is the AFTER caller-state snapshot.
	CallerAfter CallerStateSnapshot
	// Cleanup is the bounded-cleanup outcome for the subject
	// execution and the binary build.
	Cleanup CleanupAuthority
}

// BuildClosureEvidenceCandidate is the pure construction
// function. It copies the inputs into the canonical struct
// shape, stamps the schema and protocol identifiers, and
// derives the expected check set from Runtime.PlanBytes via
// the production Plan Contract v1 decoder.
//
// B2-R1 derivation: the candidate builder no longer trusts
// the caller-supplied ExpectedChecks. The F:P bytes are the
// only source of truth; the decoder routes through
// parseBoundedClosurePlanDocument (re-exported via
// productionDecodeClosurePlan) so the derived list cannot
// diverge from the bytes the runner observed. The caller
// cannot smuggle in an alternative check set.
func BuildClosureEvidenceCandidate(in CandidateInputs) ClosureEvidence {
	derived := deriveExpectedChecksFromPlanBytes(in.Runtime.PlanBytes)
	results := append([]CheckResult(nil), in.Results...)
	return ClosureEvidence{
		SchemaVersion: ClosureEvidenceSchemaVersion,
		Protocol:      ClosureProtocolVersion,
		Runtime:       in.Runtime,
		Plan: PlanAuthority{
			ExpectedChecks: derived,
		},
		Results:      results,
		Gate:         in.Gate,
		Binary:       in.Binary,
		CallerBefore: in.CallerBefore,
		CallerAfter:  in.CallerAfter,
		Cleanup:      in.Cleanup,
	}
}

// deriveExpectedChecksFromPlanBytes lives in plan_decode.go.
// The candidate builder calls it directly so the F:P bytes
// are the only source of truth for the expected check set.

// BuildEmptyEvidence returns a zero-value evidence candidate
// stamped with the schema and protocol identifiers. Tests use
// it to start from a known-good baseline before mutation.
func BuildEmptyEvidence() ClosureEvidence {
	return ClosureEvidence{
		SchemaVersion: ClosureEvidenceSchemaVersion,
		Protocol:      ClosureProtocolVersion,
	}
}

// BinaryAuthorityFromBuild converts the B1 build observability
// (BuiltBinaryEvidence) into the canonical BinaryAuthority.
// The function is pure: it copies fields, never reads the
// filesystem or the binary header.
func BinaryAuthorityFromBuild(b BuiltBinaryEvidence, binaryCommit, sourceTree string, outputOutsideAllWorktrees bool) BinaryAuthority {
	return BinaryAuthority{
		BinaryPath:                b.BinaryPath,
		BinarySHA256:              b.BinarySHA256,
		BinaryCommit:              binaryCommit,
		BinaryModified:            b.VCSModified,
		SourceCommit:              b.SourceCommit,
		SourceTree:                sourceTree,
		SourceClean:               b.SourceClean,
		SourceDetached:            b.SourceDetached,
		OutputOutsideAllWorktrees: outputOutsideAllWorktrees,
		Executable:                b.Executable,
	}
}
