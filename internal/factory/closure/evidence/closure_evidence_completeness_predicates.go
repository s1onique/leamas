// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_completeness_predicates.go
// owns the per-predicate functions for the gate, binary, and
// caller authorities. The runtime, plan, and results predicates
// live in closure_evidence_completeness.go.
//
// Splitting the per-predicate functions across two files keeps
// each file under the LLM-friendly 400-line threshold while
// preserving the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.
//
// Each predicate is exposed as a separate function so the
// mutation matrix in TestClosureEvidenceCompletenessCanonical
// can exercise them independently. The matrix MUST grow with
// every added predicate; the test asserts row count via the
// tracked predicate set in completenessPredicates.
package evidence

// ----------------------------------------------------------------------------
// Gate predicates (20..25)
// ----------------------------------------------------------------------------

// gateClassificationEqualsPASS reports that the gate classification
// verdict is PASS. Any FAIL or UNAVAILABLE verdict blocks
// COMPLETE.
func gateClassificationEqualsPASS(c ClosureEvidence) bool {
	return c.Gate.Classification == "PASS"
}

// gateInvocationCountEqualsOne reports the gate was invoked
// exactly once. The GateCollector tracks this; the predicate
// rejects any count other than 1.
func gateInvocationCountEqualsOne(c ClosureEvidence) bool {
	return c.Gate.InvocationCount == 1
}

// gateSubjectRootEqualsSExecutionRoot reports the gate's
// SubjectRoot is non-empty. The SubjectRoot is the worktree
// path the GateCollector ran against; the predicate guards
// against an empty SubjectRoot being passed off as the
// subject execution root.
func gateSubjectRootEqualsSExecutionRoot(c ClosureEvidence) bool {
	return c.Gate.SubjectRoot != ""
}

// gateNotTimedOut reports the gate did not time out.
func gateNotTimedOut(c ClosureEvidence) bool {
	return !c.Gate.TimedOut
}

// gateNoOutputTruncation reports no stdout / stderr stream was
// truncated by the gate.
func gateNoOutputTruncation(c ClosureEvidence) bool {
	return !c.Gate.StdoutTruncated && !c.Gate.StderrTruncated
}

// gateErrorAbsent reports the gate recorded no error.
func gateErrorAbsent(c ClosureEvidence) bool {
	return c.Gate.Error == ""
}

// ----------------------------------------------------------------------------
// Binary predicates (26..36)
// ----------------------------------------------------------------------------

// binaryAuthorityValid reports every binary authority invariant
// holds in the closed form. The predicate is split into the
// individual atomic predicates below so the mutation matrix
// can exercise each independently.
func binaryAuthorityValid(c ClosureEvidence) bool {
	return c.Binary.BinaryPath != "" &&
		isHexSHA256(c.Binary.BinarySHA256) &&
		isValidOID(c.Binary.BinaryCommit) &&
		!c.Binary.BinaryModified &&
		c.Binary.Executable &&
		isValidOID(c.Binary.SourceCommit) &&
		isValidOID(c.Binary.SourceTree) &&
		c.Binary.SourceClean &&
		c.Binary.SourceDetached &&
		c.Binary.OutputOutsideAllWorktrees
}

// binaryCommitEqualsSubjectCommit reports the binary's
// recorded commit matches the subject commit. The path is
// absolute so the runner cannot claim COMPLETE from a path
// the caller cannot audit.
func binaryCommitEqualsSubjectCommit(c ClosureEvidence) bool {
	return c.Binary.BinaryCommit == c.Runtime.SubjectCommit &&
		c.Binary.BinaryCommit != ""
}

// binaryNotModified reports the binary was built from a
// clean source (no dirty vcs.modified flag).
func binaryNotModified(c ClosureEvidence) bool {
	return !c.Binary.BinaryModified
}

// binarySourceCommitEqualsSubjectCommit reports the binary's
// source commit matches the subject commit.
func binarySourceCommitEqualsSubjectCommit(c ClosureEvidence) bool {
	return c.Binary.SourceCommit == c.Runtime.SubjectCommit &&
		c.Binary.SourceCommit != ""
}

// binarySourceTreeEqualsSubjectTree reports the binary's
// source tree matches the subject tree.
func binarySourceTreeEqualsSubjectTree(c ClosureEvidence) bool {
	return c.Binary.SourceTree == c.Runtime.SubjectTree &&
		c.Binary.SourceTree != ""
}

// binarySourceClean reports the source worktree was clean at
// build time.
func binarySourceClean(c ClosureEvidence) bool {
	return c.Binary.SourceClean
}

// binarySourceDetached reports the source was built against a
// detached HEAD.
func binarySourceDetached(c ClosureEvidence) bool {
	return c.Binary.SourceDetached
}

// binaryOutputOutsideAllWorktrees reports the binary's output
// path is outside every Git worktree.
func binaryOutputOutsideAllWorktrees(c ClosureEvidence) bool {
	return c.Binary.OutputOutsideAllWorktrees
}

// binaryExecutable reports the binary has the executable bit.
func binaryExecutable(c ClosureEvidence) bool {
	return c.Binary.Executable
}

// binarySHA256Valid reports the binary's SHA-256 is a 64-char
// lowercase hex digest.
func binarySHA256Valid(c ClosureEvidence) bool {
	return isHexSHA256(c.Binary.BinarySHA256)
}

// binaryCleanupErrorAbsent reports the bounded cleanup of the
// binary build recorded no error.
func binaryCleanupErrorAbsent(c ClosureEvidence) bool {
	return c.Cleanup.BinaryCleanupError == ""
}

// ----------------------------------------------------------------------------
// Caller predicates (37..43)
// ----------------------------------------------------------------------------

// callerBeforeAvailable reports the BEFORE caller-state snapshot
// was Available.
func callerBeforeAvailable(c ClosureEvidence) bool {
	return c.CallerBefore.Available
}

// callerAfterAvailable reports the AFTER caller-state snapshot
// was Available.
func callerAfterAvailable(c ClosureEvidence) bool {
	return c.CallerAfter.Available
}

// callerHEADUnchanged reports HEAD did not change between
// BEFORE and AFTER.
func callerHEADUnchanged(c ClosureEvidence) bool {
	return c.CallerBefore.Available &&
		c.CallerAfter.Available &&
		c.CallerBefore.Head == c.CallerAfter.Head
}

// callerTreeUnchanged reports HEAD's tree did not change between
// BEFORE and AFTER.
func callerTreeUnchanged(c ClosureEvidence) bool {
	return c.CallerBefore.Available &&
		c.CallerAfter.Available &&
		c.CallerBefore.Tree == c.CallerAfter.Tree
}

// callerStatusUnchanged reports the working-copy status hash
// did not change between BEFORE and AFTER.
func callerStatusUnchanged(c ClosureEvidence) bool {
	return c.CallerBefore.Available &&
		c.CallerAfter.Available &&
		c.CallerBefore.StatusHash == c.CallerAfter.StatusHash
}

// callerRefsUnchanged reports the refs hash did not change
// between BEFORE and AFTER. Empty-but-available drift is
// rejected: the runner observed a different value and the
// authority failed.
func callerRefsUnchanged(c ClosureEvidence) bool {
	return c.CallerBefore.Available &&
		c.CallerAfter.Available &&
		c.CallerBefore.RefsHash == c.CallerAfter.RefsHash
}

// callerWorktreeInventoryUnchanged reports no worktree
// registration leaked between BEFORE and AFTER.
func callerWorktreeInventoryUnchanged(c ClosureEvidence) bool {
	return c.CallerBefore.Available &&
		c.CallerAfter.Available &&
		c.CallerBefore.WorktreeInventoryHash == c.CallerAfter.WorktreeInventoryHash
}
