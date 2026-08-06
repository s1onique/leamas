// SPDX-License-Identifier: Apache-2.0

package closure

// r2cr4_diagnostic_order_test.go covers the exact diagnostic
// cardinality and order required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2C-R4.
//
// Required order matches the production `Diff` contract:
//   1. inner execution failure diagnostic
//   2. HEAD commit changed
//   3. HEAD tree changed
//   4. caller worktree dirty
//   5. linked-worktree registration leaked

import (
	"context"
	"testing"
)

// TestR2CRMultiDriftPinsCanonicalOrder proves that a
// multi-drift fixture produces diagnostics in the canonical
// Diff order: head_commit, head_tree, status, worktree.
func TestR2CRMultiDriftPinsCanonicalOrder(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	before := snapshotCallerStateForFixture(t, req.RepositoryRoot)
	afterFn := func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
		if phase == V2SnapshotPhaseBefore {
			return before
		}
		after := before
		mutateR2CR2SnapshotState(&after.State, "head_commit")
		mutateR2CR2SnapshotState(&after.State, "head_tree")
		mutateR2CR2SnapshotState(&after.State, "status")
		mutateR2CR2SnapshotState(&after.State, "worktree_leaked")
		return after
	}
	deps := r2cr2InnerFailureDeps(t, req,
		V2CodeExecutionFailed, "multi-drift inner failure",
		afterFn, &countingCandidateObserver{})
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("runner must surface a V2Error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	if len(v2err.Diags) != 5 {
		t.Fatalf("expected exactly 5 diagnostics (inner + 4 drifts), got %d: %+v",
			len(v2err.Diags), v2err.Diags)
	}
	wantOrder := []V2DiagnosticCode{
		V2CodeExecutionFailed,
		V2CodeCallerHeadChanged,
		V2CodeCallerTreeChanged,
		V2CodeCallerWorktreeDirtyAfter,
		V2CodeWorktreeRegistrationLeaked,
	}
	for i, want := range wantOrder {
		if v2err.Diags[i].Code != want {
			t.Fatalf("diag[%d].Code: got=%s want=%s", i, v2err.Diags[i].Code, want)
		}
	}
}

// TestR2CRTypedInnerErrorExactCardinality proves a typed
// *V2Error inner failure surfaces exactly two diagnostics.
func TestR2CRTypedInnerErrorExactCardinality(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	before := snapshotCallerStateForFixture(t, req.RepositoryRoot)
	afterFn := driftSnapshotAfterFn(before, "head_commit")
	deps := r2cr2InnerFailureDeps(t, req,
		V2CodeExecutionFailed, "typed inner failure for exact cardinality",
		afterFn, &countingCandidateObserver{})
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("runner must surface a V2Error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	if len(v2err.Diags) != 2 {
		t.Fatalf("expected exactly 2 diagnostics, got %d: %+v",
			len(v2err.Diags), v2err.Diags)
	}
	if v2err.Diags[0].Code != V2CodeExecutionFailed {
		t.Fatalf("diag[0].Code: got=%s want=%s",
			v2err.Diags[0].Code, V2CodeExecutionFailed)
	}
	if !containsDiagsMessage(v2err.Diags[0].Message, "typed inner failure for exact cardinality") {
		t.Fatalf("diag[0].Message must embed inner message: %q", v2err.Diags[0].Message)
	}
	if v2err.Diags[1].Code != V2CodeCallerHeadChanged {
		t.Fatalf("diag[1].Code: got=%s want=%s",
			v2err.Diags[1].Code, V2CodeCallerHeadChanged)
	}
}
