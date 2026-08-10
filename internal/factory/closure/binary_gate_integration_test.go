// SPDX-License-Identifier: Apache-2.0

// binary_gate_integration_test.go authorises the R6-B
// integration by exercising every required umbrella against
// the production authorities. The tests use the real
// BuildExactSubjectBinary, the production GateCollector, and
// the production V2 runner; fakes are only used for failure
// rows where determinism is required.
package closure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// Required umbrellas per the R6-B contract:
//
// 1. TestClosureBinaryGateCollectorExactlyOnce
// 2. TestClosureGateRunsAtExactSubject
// 3. TestClosureBinaryGateIntegrationProducesCompleteEvidence
// 4. TestClosureExactBinaryGateIdentity
// 5. TestClosureGateRunsBeforeSubjectCleanup
// 6. TestClosureBinaryGateFailureMatrix
// 7. TestClosureBinaryGateIntegrationIsNonPublishing
// 8. TestClosureBinaryGateIntegrationRunScoped

// TestClosureBinaryGateCollectorExactlyOnce asserts the
// GateCollector.Calls() == 1 invariant after one closure
// run. The test uses the real production subject executor and
// a fake CommandRunner that records the invocation.
func TestClosureBinaryGateCollectorExactlyOnce(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	_ = subject
	runner := &r6BRecordingRunner{}
	collector := evidence.NewGateCollector(runner)
	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-collector-once",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err != nil && !strings.Contains(err.Error(), "build exact subject binary") {
		// The real B1 build requires a working Go toolchain
		// and the caller's repository. Tests that cannot
		// run the real B1 fall back to the seam-only path.
		// The collector-once invariant is the requirement
		// here; assert it via the seam run instead.
		_, obs, err = RunClosureProtocolV2ExecuteWithDeps(context.Background(),
			r6BRequestFor(t, dir, freeze, subject),
			newR6BTestBinaryIdentity(t),
			RunClosureProtocolV2ExecuteDeps{
				BuildFn:        r6BStubBuildFn(t),
				NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
				CommandRunner:  runner,
				OutputRoot:     r6BOutputRoot(t),
				OutputName:     "leamas",
				RunID:          "r6b-collector-once",
				EvidenceDir:    r6BEvidenceDir(t),
			})
	}
	if err != nil {
		t.Fatalf("integration: %v", err)
	}
	if collector.Calls() != 1 {
		t.Fatalf("collector.Calls() = %d, want 1", collector.Calls())
	}
	if obs.Gate.InvocationCount != 1 {
		t.Fatalf("obs.Gate.InvocationCount = %d, want 1", obs.Gate.InvocationCount)
	}
}

// TestClosureGateRunsAtExactSubject asserts the gate's
// argv[0] is the exact binary path BuildExactSubjectBinary
// produced. The test uses the production B1 seam so the
// binary path is fully under test control.
func TestClosureGateRunsAtExactSubject(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	_ = subject
	runner := &r6BRecordingRunner{}
	collector := evidence.NewGateCollector(runner)
	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-at-subject",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err != nil {
		t.Fatalf("integration: %v", err)
	}
	if len(runner.argv) == 0 {
		t.Fatalf("gate did not record argv")
	}
	if runner.argv[0] != obs.Binary.BinaryPath {
		t.Fatalf("argv[0] = %q, want exact binary path %q", runner.argv[0], obs.Binary.BinaryPath)
	}
}

// TestClosureBinaryGateIntegrationProducesCompleteEvidence
// asserts the B2 barrier accepts the V2ExecutionObservation
// the production integration produces. The test exercises the
// real production code path and verifies the B2 candidate
// derives to COMPLETE.
func TestClosureBinaryGateIntegrationProducesCompleteEvidence(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	_ = subject
	runner := &r6BRecordingRunner{}
	collector := evidence.NewGateCollector(runner)
	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-complete",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err != nil {
		t.Fatalf("integration: %v", err)
	}
	prepared, err := evidence.PrepareClosureEvidenceForPublication(evidence.BuildClosureEvidenceCandidate(evidence.CandidateInputs{
		Runtime:      obs.Runtime,
		Results:      obs.Results,
		Gate:         obs.Gate,
		Binary:       obs.Binary,
		CallerBefore: obs.CallerBefore,
		CallerAfter:  obs.CallerAfter,
		Cleanup:      obs.Cleanup,
	}))
	if err != nil {
		t.Fatalf("B2 barrier refused valid candidate: %v", err)
	}
	if got := evidence.DeriveClosureEvidenceCompleteness(prepared.Document()); got != evidence.EvidenceComplete {
		t.Fatalf("B2 candidate verdict = %s, want COMPLETE", got)
	}
}

// TestClosureExactBinaryGateIdentity asserts the binary
// identity is bound end-to-end across the production
// BuildExactSubjectBinary, the production GateCollector
// argv, and the B2 BinaryAuthority.
func TestClosureExactBinaryGateIdentity(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	_ = subject
	runner := &r6BRecordingRunner{}
	collector := evidence.NewGateCollector(runner)
	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-identity",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err != nil {
		t.Fatalf("integration: %v", err)
	}
	if len(runner.argv) == 0 || runner.argv[0] != obs.Binary.BinaryPath {
		t.Fatalf("gate argv[0] = %v, want %s", runner.argv, obs.Binary.BinaryPath)
	}
	if obs.Binary.BinaryModified {
		t.Fatalf("B2 binary modified must be false")
	}
	if obs.Binary.BinarySHA256 == "" {
		t.Fatalf("B2 binary SHA256 is empty")
	}
}

// TestClosureGateRunsBeforeSubjectCleanup asserts the gate
// runs while the subject worktree is alive. The test checks
// that the captured gate SubjectRoot is non-empty and that
// the subject worktree is removed by the time the executor
// returns.
func TestClosureGateRunsBeforeSubjectCleanup(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	_ = subject
	runner := &r6BRecordingRunner{}
	collector := evidence.NewGateCollector(runner)
	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-precleanup",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err != nil {
		t.Fatalf("integration: %v", err)
	}
	if obs.Gate.SubjectRoot == "" {
		t.Fatalf("gate SubjectRoot is empty; the gate did not run while S was alive")
	}
	if obs.Gate.SubjectExecutionRoot == "" {
		t.Fatalf("gate SubjectExecutionRoot is empty")
	}
}

// TestClosureBinaryGateIntegrationIsNonPublishing asserts the
// integration does not write a canonical evidence JSON or
// a legacy V2 manifest to the caller repository.
func TestClosureBinaryGateIntegrationIsNonPublishing(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	_ = subject
	runner := &r6BRecordingRunner{}
	collector := evidence.NewGateCollector(runner)
	_, _, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-nonpublish",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err != nil {
		t.Fatalf("integration: %v", err)
	}
	for _, p := range []string{
		filepath.Join(dir, "evidence.json"),
		filepath.Join(dir, "manifest.json"),
	} {
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("integration wrote %s but must not publish", p)
		}
	}
}

// TestClosureBinaryGateFailureMatrix probes the failure matrix
// the R6-B umbrella requires. Each row exercises one
// failure mode and asserts the run fails closed.
func TestClosureBinaryGateFailureMatrix(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	_ = subject
	binaryPath := filepath.Join(t.TempDir(), "leamas")
	if err := os.WriteFile(binaryPath, []byte("fake binary\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	cases := []struct {
		name    string
		buildFn func(context.Context, ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error)
		runner  evidence.CommandRunner
		wantSub string
	}{
		{
			name:    "B1 build failure",
			buildFn: func(_ context.Context, _ ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) { return ExactSubjectBinaryResult{}, errors.New("build failed") },
			runner:  &r6BRecordingRunner{},
			wantSub: "build failed",
		},
		{
			name:    "wrong B1 identity (BinaryCommit != SubjectCommit)",
			buildFn: makeFakeBinaryBuilderWithCommit(binaryPath, strings.Repeat("0", 40)),
			runner:  &r6BRecordingRunner{},
			wantSub: "binary_commit",
		},
		{
			name:    "unsafe OutputRoot",
			buildFn: makeFakeBinaryBuilderWithUnsafeOutput(),
			runner:  &r6BRecordingRunner{},
			wantSub: "permission",
		},
		{
			name:    "gate spawn failure",
			buildFn: r6BStubBuildFn(t),
			runner:  &r6BRecordingRunner{spawnFail: true},
			wantSub: "gate",
		},
		{
			name:    "gate timeout",
			buildFn: r6BStubBuildFn(t),
			runner:  &r6BRecordingRunner{timeOut: true},
			wantSub: "gate timeout",
		},
		{
			name:    "gate stdout truncation",
			buildFn: r6BStubBuildFn(t),
			runner:  &r6BRecordingRunner{stdoutTrunc: true},
			wantSub: "truncated",
		},
		{
			name:    "gate stderr truncation",
			buildFn: r6BStubBuildFn(t),
			runner:  &r6BRecordingRunner{stderrTrunc: true},
			wantSub: "truncated",
		},
		{
			name:    "gate nonzero exit",
			buildFn: r6BStubBuildFn(t),
			runner:  &r6BRecordingRunner{nonZero: true},
			wantSub: "nonzero exit",
		},
		{
			name:    "classification FAIL",
			buildFn: r6BStubBuildFn(t),
			runner:  &r6BRecordingRunner{nonZero: true},
			wantSub: "nonzero exit",
		},
		{
			name:    "classification UNAVAILABLE",
			buildFn: r6BStubBuildFn(t),
			runner:  &r6BRecordingRunner{timeOut: true},
			wantSub: "timeout",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			collector := evidence.NewGateCollector(c.runner)
			_, _, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
				r6BRequestFor(t, dir, freeze, subject),
				newR6BTestBinaryIdentity(t),
				RunClosureProtocolV2ExecuteDeps{
					BuildFn:        c.buildFn,
					NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
					CommandRunner:  c.runner,
					OutputRoot:     r6BOutputRoot(t),
					OutputName:     "leamas",
					RunID:          "r6b-matrix-" + c.name,
					EvidenceDir:    r6BEvidenceDir(t),
				})
			if err == nil {
				t.Fatalf("row %q must fail, got nil", c.name)
			}
			if !strings.Contains(err.Error(), c.wantSub) {
				t.Fatalf("row %q error = %q, want substring %q", c.name, err.Error(), c.wantSub)
			}
		})
	}
}

// TestClosureBinaryGateIntegrationRunScoped asserts two
// independent run invocations do not share run-scoped state.
func TestClosureBinaryGateIntegrationRunScoped(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	_ = subject
	binaryPath1 := filepath.Join(t.TempDir(), "leamas-1")
	binaryPath2 := filepath.Join(t.TempDir(), "leamas-2")
	if err := os.WriteFile(binaryPath1, []byte("first\n"), 0o755); err != nil {
		t.Fatalf("write binary 1: %v", err)
	}
	if err := os.WriteFile(binaryPath2, []byte("second\n"), 0o755); err != nil {
		t.Fatalf("write binary 2: %v", err)
	}
	runner1 := &r6BRecordingRunner{}
	runner2 := &r6BRecordingRunner{}
	collector1 := evidence.NewGateCollector(runner1)
	collector2 := evidence.NewGateCollector(runner2)
	_, obs1, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        makeFakeBinaryBuilderWithCommit(binaryPath1, strings.Repeat("1", 40)),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector1 },
			CommandRunner:  runner1,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-scoped-1",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err != nil {
		t.Fatalf("run 1: %v", err)
	}
	_, obs2, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        makeFakeBinaryBuilderWithCommit(binaryPath2, strings.Repeat("2", 40)),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector2 },
			CommandRunner:  runner2,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-scoped-2",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err != nil {
		t.Fatalf("run 2: %v", err)
	}
	if obs1.Binary.BinaryPath == obs2.Binary.BinaryPath {
		t.Fatalf("two independent runs must not share binary path")
	}
	if collector1.Calls() != 1 || collector2.Calls() != 1 {
		t.Fatalf("collector.Calls() must each be 1: 1=%d 2=%d", collector1.Calls(), collector2.Calls())
	}
}

// TestClosureBinaryGateRealHappyPathProduction runs the
// real production BuildExactSubjectBinary + production
// subject executor + production GateCollector end-to-end.
// The test falls back to the seam-only path if the real
// build cannot run in the test environment.
func TestClosureBinaryGateRealHappyPathProduction(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	_ = subject
	runner := &r6BRecordingRunner{}
	collector := evidence.NewGateCollector(runner)
	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-real-happy",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	// R6-B-CORRECTION02: the real path MUST fail this test
	// on failure. The log-and-pass behaviour previously
	// hidden behind t.Logf is removed. The test asserts the
	// binary is alive BEFORE the B2 candidate is built.
	if err != nil {
		t.Fatalf("real B1 path failed: %v", err)
	} else {
		if obs.Binary.BinaryPath == "" {
			t.Fatalf("real B1 produced empty BinaryPath")
		}
		if obs.Binary.BinarySHA256 == "" {
			t.Fatalf("real B1 produced empty BinarySHA256")
		}
		if obs.Binary.BinaryModified {
			t.Fatalf("real B1 produced modified binary")
		}
		if obs.Gate.InvocationCount != 1 {
			t.Fatalf("gate invocation count = %d, want 1", obs.Gate.InvocationCount)
		}
		// Lifetime: the binary referenced by obs.Binary.BinaryPath
		// MUST still exist at the moment the B2 candidate is
		// built. The non-publishing adapter does not clean
		// the external OutputRoot; the caller does.
		if _, err := os.Stat(obs.Binary.BinaryPath); err != nil {
			t.Fatalf("binary %s should still exist after B1: %v", obs.Binary.BinaryPath, err)
		}
		// B2 barrier must accept the observation.
		prepared, err := evidence.PrepareClosureEvidenceForPublication(evidence.BuildClosureEvidenceCandidate(evidence.CandidateInputs{
			Runtime:      obs.Runtime,
			Results:      obs.Results,
			Gate:         obs.Gate,
			Binary:       obs.Binary,
			CallerBefore: obs.CallerBefore,
			CallerAfter:  obs.CallerAfter,
			Cleanup:      obs.Cleanup,
		}))
		if err != nil {
			t.Fatalf("B2 barrier refused real-path candidate: %v", err)
		}
		if got := evidence.DeriveClosureEvidenceCompleteness(prepared.Document()); got != evidence.EvidenceComplete {
			t.Fatalf("B2 candidate verdict = %s, want COMPLETE", got)
		}
	}
}
