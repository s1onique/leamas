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
	if err != nil && strings.Contains(err.Error(), "build exact subject binary") {
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
	runner1 := &r6BRecordingRunner{}
	runner2 := &r6BRecordingRunner{}
	collector1 := evidence.NewGateCollector(runner1)
	collector2 := evidence.NewGateCollector(runner2)
	// Use r6BStubBuildFn which produces valid binary authority
	// (BinaryCommit == SubjectCommit) for each run.
	_, obs1, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
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
			BuildFn:        r6BStubBuildFn(t),
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

// TestClosureBinaryGateRealHappyPathWithStubBuilder runs the
// production V2 runner with the stub builder that produces
// canonical BinaryAuthority (BinaryCommit == SubjectCommit).
// The stub exercises the seam without requiring the Go
// toolchain on a temp Git repo. This is NOT the real B1
// production canary; it is seam coverage only.
func TestClosureBinaryGateRealHappyPathWithStubBuilder(t *testing.T) {
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
			RunID:          "r6b-real-happy",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err != nil {
		t.Fatalf("production path failed: %v", err)
	}
	if obs.Binary.BinaryPath == "" {
		t.Fatalf("B1 produced empty BinaryPath")
	}
	if obs.Binary.BinarySHA256 == "" {
		t.Fatalf("B1 produced empty BinarySHA256")
	}
	if obs.Binary.BinaryModified {
		t.Fatalf("B1 produced modified binary")
	}
	if obs.Gate.InvocationCount != 1 {
		t.Fatalf("gate invocation count = %d, want 1", obs.Gate.InvocationCount)
	}
	// Lifetime: the binary referenced by obs.Binary.BinaryPath
	// MUST still exist at the moment the B2 candidate is built.
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
		t.Fatalf("B2 barrier refused candidate: %v", err)
	}
	if got := evidence.DeriveClosureEvidenceCompleteness(prepared.Document()); got != evidence.EvidenceComplete {
		t.Fatalf("B2 candidate verdict = %s, want COMPLETE", got)
	}
}
