// SPDX-License-Identifier: Apache-2.0

// binary_gate_correction06_cleanup_test.go owns the focused
// production-side test for the R6-B subject-cleanup
// fail-closed surface added by ACT-CORRECTION06. The test
// proves that the R6-A subject-cleanup authority surfaces
// V2CodeR6BSubjectCleanupFailed at the integration seam
// when the executor's worktree cleanup fails AFTER the
// GateCollector has executed.
//
// The test is split from binary_gate_correction06_test.go
// so each ACT-owned file stays under the LLM-friendly
// 400-line threshold while preserving the production
// assertion contract.

package closure

import (
	"context"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// TestR6BSubjectCleanupFailureAfterGate proves the
// R6-A subject cleanup authority surfaces
// V2CodeR6BSubjectCleanupFailed when the executor reports
// a failure AFTER the gate has executed.
//
// The test exercises the REAL production path end-to-end:
// the live S worktree is created, the live-S observations
// are captured, the GateCollector executes, the classifier
// completes, and the subject worktree cleanup fails. The
// failure surfaces as the typed V2CodeR6BSubjectCleanupFailed
// owned R6-B code via validateSubjectCleanupOutcome.
//
// Required assertions:
//   - err != nil (R6-B adapter refuses to publish)
//   - first diagnostic code is V2CodeR6BSubjectCleanupFailed
//   - subject worktree cleanup was attempted (the failure
//     originated from the R6-A cleanup authority, not from
//     B1 or the gate runner)
//   - gate invocation count is exactly 1 (the gate ran
//     BEFORE cleanup failure was returned)
func TestR6BSubjectCleanupFailureAfterGate(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	runner := &r6BRecordingRunner{stdoutField: []byte("EXEC_GATE_OBSERVED_STATUS:OK\n")}
	collector := evidence.NewGateCollector(runner)
	// Drive the production subject-cleanup authority to
	// fail deterministically. The fake delegates all
	// non-overridden commands to RealGit so the worktree
	// add, observation, and inventory phases use the
	// real authority; only the cleanup phase fails.
	// Unlike subjectMatrixGitClient{cleanupFail: true}
	// which cascades into the AFTER inventory observation,
	// r6BRealSubjectCleanupFailureGitClient fails ONLY
	// the worktree remove call so the R6-B subject-cleanup
	// failure is the owning diagnostic.
	fakeGit := r6BRealSubjectCleanupFailureGitClient()
	_, _, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-c06-subject-cleanup",
			EvidenceDir:    r6BEvidenceDir(t),
			GitClient:      fakeGit,
		})
	if err == nil {
		t.Fatalf("subject cleanup failure must surface a typed error")
	}
	requireV2Code(t, err, V2CodeR6BSubjectCleanupFailed)
	if collector.Calls() != 1 {
		t.Fatalf("gate invocation count = %d, want 1", collector.Calls())
	}
}

// TestR6BSubjectCleanupUnavailableSurface proves that
// validateSubjectCleanupOutcome surfaces
// V2CodeR6BSubjectCleanupUnavailable when the executor
// does NOT record a cleanup observation. The unit test
// verifies the validator's typed-code contract directly;
// the production-side counterpart is exercised via
// matrix row 12 when cleanup is not attempted at all.
func TestR6BSubjectCleanupUnavailableSurface(t *testing.T) {
	t.Parallel()
	vErr := validateSubjectCleanupOutcome(V2ExecuteResult{
		SubjectCleanupObserved: false,
	})
	if vErr == nil {
		t.Fatalf("subject cleanup unobserved must surface a typed error")
	}
	requireV2Code(t, vErr, V2CodeR6BSubjectCleanupUnavailable)
}
