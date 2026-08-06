// SPDX-License-Identifier: Apache-2.0

package closure

// v2_publication_barrier_r2cr2_inner_failure_test.go proves the
// R2C-R2 inner-error drift matrix. Each test:
//
//  1. makes the inner runner fail with a deterministic typed
//     diagnostic (via a failing executor),
//  2. mutates the after-state snapshot so the runner reports
//     a drift diagnostic alongside the original inner
//     diagnostic,
//  3. asserts the runner returned the typed V2Error,
//  4. asserts the inner diagnostic appears FIRST in the
//     diagnostic slice,
//  5. asserts the drift diagnostic appears exactly once,
//  6. asserts no manifest bytes were published.
//
// The tests use a phase-aware snapshot seam so the snapshot
// phase can be targeted exactly without command-count
// approximations.

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestR2CR2Runner_InnerFailure_HeadDrift_SurfacesBothDiagnostics(t *testing.T) {
	runInnerFailureWithDrift(t, "head_commit", V2CodeCallerHeadChanged)
}

func TestR2CR2Runner_InnerFailure_HeadTreeDrift_SurfacesBothDiagnostics(t *testing.T) {
	runInnerFailureWithDrift(t, "head_tree", V2CodeCallerTreeChanged)
}

func TestR2CR2Runner_InnerFailure_StatusDrift_SurfacesBothDiagnostics(t *testing.T) {
	runInnerFailureWithDrift(t, "status", V2CodeCallerWorktreeDirtyAfter)
}

func TestR2CR2Runner_InnerFailure_WorktreeLeak_SurfacesBothDiagnostics(t *testing.T) {
	runInnerFailureWithDrift(t, "worktree_leaked", V2CodeWorktreeRegistrationLeaked)
}

// runInnerFailureWithDrift runs the runner with a fake
// executor that fails the inner execution with a known typed
// diagnostic, plus a phase-aware snapshot function that
// reports the requested drift on the after phase. The test
// asserts:
//
//  1. the runner returns a *V2Error,
//  2. the inner diagnostic appears first in the diagnostic
//     slice (so the CLI renders the root cause),
//  3. the drift diagnostic appears exactly once,
//  4. no manifest bytes were published.
func runInnerFailureWithDrift(t *testing.T, driftKind string, expectedDriftCode V2DiagnosticCode) {
	t.Helper()
	req := v2FailClosedRunnerRequest(t)
	const innerCode = V2CodeExecutionFailed
	const innerMessage = "inner execution failed (synthetic)"
	observer := &countingCandidateObserver{}

	before := snapshotCallerStateForFixture(t, req.RepositoryRoot)
	afterFn := driftSnapshotAfterFn(before, driftKind)

	deps := r2cr2InnerFailureDeps(t, req, innerCode, innerMessage, afterFn, observer)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("inner failure must surface a V2Error")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}

	if len(v2err.Diags) < 2 {
		t.Fatalf("expected at least 2 diagnostics (inner + drift); got %d: %+v",
			len(v2err.Diags), v2err.Diags)
	}
	if v2err.Diags[0].Code != innerCode {
		t.Fatalf("first diagnostic must be the inner %s; got %s message=%q",
			innerCode, v2err.Diags[0].Code, v2err.Diags[0].Message)
	}
	if !strings.Contains(v2err.Diags[0].Message, innerMessage) {
		t.Fatalf("inner diagnostic message lost; got %q", v2err.Diags[0].Message)
	}
	driftCount := 0
	for _, d := range v2err.Diags {
		if d.Code == expectedDriftCode {
			driftCount++
		}
	}
	if driftCount != 1 {
		t.Fatalf("expected drift code %s exactly once; got count=%d diags=%+v",
			expectedDriftCode, driftCount, v2err.Diags)
	}

	if observer.Calls() != 0 {
		t.Fatalf("candidate observer must not fire on inner failure; got %d calls", observer.Calls())
	}

	// The manifest path must remain absent.
	if _, statErr := os.Stat(req.ManifestOutput); statErr == nil {
		t.Fatalf("manifest must not be published on inner failure")
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("unexpected manifest stat error: %v", statErr)
	}
}

// snapshotCallerStateForFixture returns a synthetic before-state
// snapshot suitable for the inner-failure matrix. It mirrors
// the structure produced by the production snapshotCallerState.
func snapshotCallerStateForFixture(t *testing.T, repoRoot string) v2CallerStateSnapshot {
	t.Helper()
	subject := gitForClosureTestHelperR2CR2(t, repoRoot, "rev-parse", "HEAD")
	tree := gitForClosureTestHelperR2CR2(t, repoRoot, "rev-parse", subject+"^{tree}")
	return v2CallerStateSnapshot{
		Available: true,
		State: v2CallerState{
			HEADCommit:            subject,
			HEADTree:              tree,
			StatusPorcelain:       "",
			WorktreeRegistrations: v2WorktreeRegistrationSet{},
		},
	}
}

// driftSnapshotAfterFn returns a V2RunnerSnapshotFunc that
// returns a real before-state snapshot at the before phase and
// a drifted snapshot at the after phase.
func driftSnapshotAfterFn(before v2CallerStateSnapshot, kind string) V2RunnerSnapshotFunc {
	return func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
		if phase == V2SnapshotPhaseBefore {
			return before
		}
		after := before
		mutateR2CR2SnapshotState(&after.State, kind)
		return after
	}
}

// mutateR2CR2SnapshotState mutates the supplied caller-state
// in place to inject the requested drift.
func mutateR2CR2SnapshotState(s *v2CallerState, kind string) {
	switch kind {
	case "head_commit":
		s.HEADCommit = r2cr2MutateCommit(s.HEADCommit)
	case "head_tree":
		s.HEADTree = r2cr2MutateCommit(s.HEADTree)
	case "status":
		s.StatusPorcelain = "?? leaked-untracked\n"
	case "worktree_leaked":
		s.WorktreeRegistrations = append(s.WorktreeRegistrations,
			v2WorktreeRegistration{Path: "/tmp/leaked-worktree", Hash: s.HEADCommit})
	default:
		panic("unknown drift kind: " + kind)
	}
}

// r2cr2MutateCommit flips the last byte of a 40-char commit hex
// to produce a syntactically valid but distinct OID.
func r2cr2MutateCommit(oid string) string {
	if len(oid) == 0 {
		return ""
	}
	b := []byte(oid)
	if b[len(b)-1] == '0' {
		b[len(b)-1] = '1'
	} else {
		b[len(b)-1] = '0'
	}
	return string(b)
}

// r2cr2InnerFailureDeps assembles a V2RunnerDeps configured to
// fail the inner execution with the supplied code/message and
// to use the supplied snapshot function.
func r2cr2InnerFailureDeps(t *testing.T, req V2Request, innerCode V2DiagnosticCode, innerMessage string, afterFn V2RunnerSnapshotFunc, observer V2CandidateObserver) V2RunnerDeps {
	t.Helper()
	deps := DefaultV2RunnerDeps()
	deps.BinaryIdentity = newV2TestBinaryIdentity(t)
	deps.SnapshotFn = afterFn
	deps.CandidateObserver = observer
	deps.Executor = &r2cr2FailingExecutor{code: innerCode, message: innerMessage}
	return deps
}

// r2cr2FailingExecutor is a V2SubjectExecutor that always fails
// the inner execution with a known typed diagnostic.
type r2cr2FailingExecutor struct {
	code    V2DiagnosticCode
	message string
}

func (e *r2cr2FailingExecutor) ExecuteSubjectChecks(ctx context.Context, req V2ExecuteRequest) (V2ExecuteResult, error) {
	return V2ExecuteResult{}, NewV2ErrorWith(e.code, e.message, "inner_evidence", e.message)
}

// gitForClosureTestHelperR2CR2 runs an arbitrary git command
// in repoRoot and returns the trimmed stdout.
func gitForClosureTestHelperR2CR2(t *testing.T, repoRoot string, args ...string) string {
	t.Helper()
	out, err := runGitValue(context.Background(), RealGit{}, repoRoot, args...)
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return out
}

// r2cr2InnerErrorSentinel is a sentinel error used by the
// inner-failure matrix. It satisfies the V2Error contract so
// the runner can append diagnostics to it.
var r2cr2InnerErrorSentinel = errors.New("r2cr2 inner failure sentinel")
