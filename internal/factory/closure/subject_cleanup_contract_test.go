// SPDX-License-Identifier: Apache-2.0

// subject_cleanup_contract_test.go owns the direct
// R6-A subject-executor cleanup-error contract tests
// the CORRECTION08 ACT requires (Phase 11-13).
//
// CORRECTION07 weakened the lower-level executor: a
// cleanup-only failure returned (V2ExecuteResult{}, nil)
// and relied on R6-B to reinterpret the result. That
// changed the R6-A direct-caller contract.
//
// CORRECTION08 restores the contract:
//
//	successful execution + successful cleanup
//	  -> result, nil
//
//	successful execution + cleanup failure
//	  -> populated result
//	  -> non-nil V2CodeCleanupFailed
//
// The test exercises the production
// GitV2SubjectExecutor.ExecuteSubjectChecks directly
// with the cleanup-failure Git seam so the assertion
// does NOT route through the R6-B adapter.
package closure

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestSubjectExecutorCleanupFailureStillErrors proves the
// R6-A direct caller contract preserved by CORRECTION08:
//
//	result.SubjectCleanupObserved == true
//	result.SubjectCleanupError   != ""
//	err != nil
//	first diagnostic code includes V2CodeCleanupFailed
//
// The test calls the production
// GitV2SubjectExecutor.ExecuteSubjectChecks directly so
// the assertion does NOT route through the R6-B adapter.
// The seam is r6BRealSubjectCleanupFailureGitClient (the
// row-12 seam).
func TestSubjectExecutorCleanupFailureStillErrors(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	exec := NewGitV2SubjectExecutor(r6BRealSubjectCleanupFailureGitClient())
	plan := r6BValidPlanBytes()
	// Decode the canonical plan to feed the executor.
	planObj, _, err := parsePlanBytes(plan)
	if err != nil {
		t.Fatalf("parse plan: %v", err)
	}
	result, execErr := exec.ExecuteSubjectChecks(context.Background(), V2ExecuteRequest{
		RepositoryRoot: dir,
		SubjectCommit:  subject,
		SubjectTree:    strings.Repeat("b", 40),
		EvidenceDir:    r6BEvidenceDir(t),
		Checks:         planObj.Checks,
	})
	if execErr == nil {
		t.Fatalf("cleanup failure must produce a non-nil error")
	}
	if !result.SubjectCleanupObserved {
		t.Fatalf("SubjectCleanupObserved = false, want true")
	}
	if result.SubjectCleanupError == "" {
		t.Fatalf("SubjectCleanupError empty, want populated")
	}
	// The first/contained diagnostic code must include
	// V2CodeCleanupFailed so direct R6-A callers see
	// the owning family.
	if !v2ErrorContainsCode(execErr, V2CodeCleanupFailed) {
		t.Fatalf("err = %v, want first/contained code %q",
			execErr, V2CodeCleanupFailed)
	}
}

// TestSubjectExecutorPrimaryAndCleanupFailurePreserved
// proves the primary-error-preserved-on-cleanup-failure
// contract: when execution has already failed AND cleanup
// also fails, both failures remain observable.
//
// The test seeds a primary failure by asking the executor
// to run against a subject_tree that does not match the
// observed tree (forces V2CodeExecutionTreeMismatch) and
// a cleanup-failure Git seam. The result must contain
// BOTH diagnostics: the primary V2CodeExecutionTreeMismatch
// and the cleanup V2CodeCleanupFailed.
func TestSubjectExecutorPrimaryAndCleanupFailurePreserved(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	exec := NewGitV2SubjectExecutor(r6BRealSubjectCleanupFailureGitClient())
	plan := r6BValidPlanBytes()
	planObj, _, err := parsePlanBytes(plan)
	if err != nil {
		t.Fatalf("parse plan: %v", err)
	}
	// Use a subject_tree that does NOT match the
	// resolved subject tree so the executor's
	// observed-tree mismatch diagnostic fires BEFORE
	// cleanup. The cleanup seam then adds a second
	// failure.
	mismatchedTree := strings.Repeat("0", 40)
	result, execErr := exec.ExecuteSubjectChecks(context.Background(), V2ExecuteRequest{
		RepositoryRoot: dir,
		SubjectCommit:  subject,
		SubjectTree:    mismatchedTree,
		EvidenceDir:    r6BEvidenceDir(t),
		Checks:         planObj.Checks,
	})
	if execErr == nil {
		t.Fatalf("primary + cleanup failure must produce a non-nil error")
	}
	if !result.SubjectCleanupObserved {
		t.Fatalf("SubjectCleanupObserved = false, want true")
	}
	if result.SubjectCleanupError == "" {
		t.Fatalf("SubjectCleanupError empty, want populated")
	}
	// The error MUST contain BOTH the primary and the
	// cleanup codes. The primary is preserved; cleanup
	// is appended/preserved; nothing is silently
	// replaced.
	if !v2ErrorContainsCode(execErr, V2CodeExecutionTreeMismatch) {
		t.Fatalf("err = %v, want primary code %q",
			execErr, V2CodeExecutionTreeMismatch)
	}
	if !v2ErrorContainsCode(execErr, V2CodeCleanupFailed) {
		t.Fatalf("err = %v, want cleanup code %q",
			execErr, V2CodeCleanupFailed)
	}
}

// v2ErrorContainsCode walks the cause chain looking for
// any *V2Error whose Diags contains the given code.
// The helper is the focused cause-inspection that the
// direct R6-A cleanup contract assertions need.
func v2ErrorContainsCode(err error, want V2DiagnosticCode) bool {
	for err != nil {
		if v2err, ok := err.(*V2Error); ok {
			for _, d := range v2err.Diags {
				if d.Code == want {
					return true
				}
			}
		}
		err = errors.Unwrap(err)
	}
	return false
}
