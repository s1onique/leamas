// SPDX-License-Identifier: Apache-2.0

package closure

// r2cr4_inner_cause_test.go covers the inner-cause
// preservation contract required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2C-R4.
//
// When the inner runner returns a plain (non-*V2Error)
// sentinel error, the outer runner MUST:
//   - retain the original error as Cause (errors.Is/
//     errors.As work via Unwrap);
//   - emit a deterministic first diagnostic representing
//     the inner failure;
//   - append drift / availability diagnostics in
//     canonical order;
//   - never return a drift-only error.
//
// Required tests:
//   - plain sentinel + no drift
//   - plain sentinel + HEAD drift
//   - plain sentinel + HEAD-tree drift
//   - plain sentinel + status drift
//   - plain sentinel + worktree leak
//   - plain sentinel + unavailable after-snapshot

import (
	"context"
	"errors"
	"os"
	"testing"
)

// TestR2CRNonV2InnerErrorPreservesCause_NoDrift proves the
// outer runner preserves the original error via errors.Is
// when the inner runner returns a plain (non-*V2Error)
// sentinel and there is no drift.
func TestR2CRNonV2InnerErrorPreservesCause_NoDrift(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel")
	runR2CRInnerCauseMatrix(t, sentinel, "")
}

// TestR2CRNonV2InnerErrorPreservesCause_HeadDrift covers the
// HEAD drift scenario.
func TestR2CRNonV2InnerErrorPreservesCause_HeadDrift(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel for head drift")
	runR2CRInnerCauseMatrix(t, sentinel, "head_commit")
}

// TestR2CRNonV2InnerErrorPreservesCause_TreeDrift covers the
// HEAD-tree drift scenario.
func TestR2CRNonV2InnerErrorPreservesCause_TreeDrift(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel for tree drift")
	runR2CRInnerCauseMatrix(t, sentinel, "head_tree")
}

// TestR2CRNonV2InnerErrorPreservesCause_StatusDrift covers
// the status drift scenario.
func TestR2CRNonV2InnerErrorPreservesCause_StatusDrift(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel for status drift")
	runR2CRInnerCauseMatrix(t, sentinel, "status")
}

// TestR2CRNonV2InnerErrorPreservesCause_WorktreeLeak covers
// the worktree-leak scenario.
func TestR2CRNonV2InnerErrorPreservesCause_WorktreeLeak(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel for worktree leak")
	runR2CRInnerCauseMatrix(t, sentinel, "worktree_leaked")
}

// TestR2CRNonV2InnerErrorPreservesCause_AfterUnavailable
// covers the after-snapshot unavailability scenario.
func TestR2CRNonV2InnerErrorPreservesCause_AfterUnavailable(t *testing.T) {
	sentinel := errors.New("synthetic inner sentinel for after unavailability")
	runR2CRInnerCauseAfterUnavailableMatrix(t, sentinel)
}

// runR2CRInnerCauseMatrix exercises the runner with a
// non-*V2Error sentinel and mutates the after-snapshot per
// `driftKind`. It asserts:
//   - the wrapped error is a *V2Error,
//   - errors.Is(result, sentinel) is true (Cause preserved),
//   - the diagnostic list begins with the inner fallback
//     diagnostic (V2CodeExecutionFailed),
//   - the drift diagnostic (if any) follows in canonical
//     order,
//   - no manifest bytes were published.
func runR2CRInnerCauseMatrix(t *testing.T, sentinel error, driftKind string) {
	t.Helper()
	req := v2FailClosedRunnerRequest(t)
	before := snapshotCallerStateForFixture(t, req.RepositoryRoot)
	var afterFn V2RunnerSnapshotFunc
	if driftKind == "" {
		afterFn = func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
			return before
		}
	} else {
		afterFn = driftSnapshotAfterFn(before, driftKind)
	}
	deps := nonV2CauseDeps(t, req, sentinel, afterFn)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("runner must surface a V2Error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("errors.Is must reach sentinel: v2err.Cause=%v", v2err.Cause)
	}
	if len(v2err.Diags) == 0 {
		t.Fatalf("expected at least one diagnostic, got none")
	}
	if v2err.Diags[0].Code != V2CodeExecutionFailed {
		t.Fatalf("first diagnostic must be execution_failed, got %s", v2err.Diags[0].Code)
	}
	if !containsDiagsMessage(v2err.Diags[0].Message, sentinel.Error()) {
		t.Fatalf("inner diagnostic message must embed sentinel text: %q", v2err.Diags[0].Message)
	}
	if driftKind != "" {
		driftCode := r2cr4DriftCode(driftKind)
		idx := -1
		for i, d := range v2err.Diags {
			if d.Code == driftCode {
				idx = i
				break
			}
		}
		if idx < 1 {
			t.Fatalf("drift code %s must follow inner diagnostic, idx=%d diags=%+v",
				driftCode, idx, v2err.Diags)
		}
	}
	if _, statErr := os.Stat(req.ManifestOutput); statErr == nil {
		t.Fatalf("manifest must not be published on inner failure")
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("unexpected manifest stat error: %v", statErr)
	}
}

// runR2CRInnerCauseAfterUnavailableMatrix exercises the
// runner with a sentinel error and an unavailable
// after-snapshot. The inner diagnostic must remain FIRST.
func runR2CRInnerCauseAfterUnavailableMatrix(t *testing.T, sentinel error) {
	t.Helper()
	req := v2FailClosedRunnerRequest(t)
	before := snapshotCallerStateForFixture(t, req.RepositoryRoot)
	afterFn := func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
		if phase == V2SnapshotPhaseBefore {
			return before
		}
		return v2CallerStateSnapshot{
			Available: false,
			Diagnostics: V2Diagnostics{{
				Code:         V2CodeCallerStateUnavailable,
				Message:      "synthetic after-snapshot unavailable",
				PropertyName: "caller_state",
			}},
		}
	}
	deps := nonV2CauseDeps(t, req, sentinel, afterFn)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("runner must surface a V2Error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("errors.Is must reach sentinel: v2err.Cause=%v", v2err.Cause)
	}
	if len(v2err.Diags) == 0 {
		t.Fatalf("expected at least one diagnostic")
	}
	if v2err.Diags[0].Code != V2CodeExecutionFailed {
		t.Fatalf("first diagnostic must be execution_failed, got %s", v2err.Diags[0].Code)
	}
	hasUnavailable := false
	for _, d := range v2err.Diags {
		if d.Code == V2CodeCallerStateUnavailable {
			hasUnavailable = true
		}
	}
	if !hasUnavailable {
		t.Fatalf("after-availability diagnostic missing: %+v", v2err.Diags)
	}
}

// nonV2CauseDeps returns a V2RunnerDeps configured to fail
// the inner execution with the supplied plain sentinel error
// and to use the supplied snapshot function.
func nonV2CauseDeps(t *testing.T, req V2Request, sentinel error, afterFn V2RunnerSnapshotFunc) V2RunnerDeps {
	t.Helper()
	deps := DefaultV2RunnerDeps()
	deps.BinaryIdentity = newV2TestBinaryIdentity(t)
	deps.SnapshotFn = afterFn
	deps.Executor = &r2cr4SentinelExecutor{sentinel: sentinel}
	return deps
}

// r2cr4SentinelExecutor returns the supplied plain error so
// the outer runner must exercise the inner-cause preservation
// path.
type r2cr4SentinelExecutor struct {
	sentinel error
}

func (e *r2cr4SentinelExecutor) ExecuteSubjectChecks(ctx context.Context, req V2ExecuteRequest) (V2ExecuteResult, error) {
	return V2ExecuteResult{}, e.sentinel
}

// r2cr4DriftCode maps a drift-kind label to the matching
// V2DiagnosticCode.
func r2cr4DriftCode(kind string) V2DiagnosticCode {
	switch kind {
	case "head_commit":
		return V2CodeCallerHeadChanged
	case "head_tree":
		return V2CodeCallerTreeChanged
	case "status":
		return V2CodeCallerWorktreeDirtyAfter
	case "worktree_leaked":
		return V2CodeWorktreeRegistrationLeaked
	default:
		return ""
	}
}

// containsDiagsMessage reports whether needle appears in hay.
func containsDiagsMessage(hay, needle string) bool {
	if needle == "" {
		return true
	}
	if hay == "" {
		return false
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
