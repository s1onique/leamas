// SPDX-License-Identifier: Apache-2.0

// binary_gate_correction06_gate_test.go owns the focused
// production-side tests for the R6-B gate authority
// fail-closed surface added by ACT-CORRECTION06. The tests
// prove that the gate runner adapter surfaces typed
// V2Error codes (V2CodeR6BGateObservationFailed,
// V2CodeR6BGateClassificationFailed,
// V2CodeR6BGateClassificationUnavailable) at the integration
// seam.
//
// The file is split from binary_gate_correction06_test.go
// so each ACT-owned file stays under the LLM-friendly
// 400-line threshold while preserving the production
// assertion contract.

package closure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// r6BCollectorMismatchingRunner is the production-shaped
// CommandRunner double TestR6BCollectorMismatchPropagates
// uses so the live integration Capture path can be
// exercised. The runner returns success so the production
// integration observes the gate collector identity
// mismatch on the SECOND capture (the first one pre-pins
// the identity on the collector).
//
// The runner maintains an actual atomic counter (not a
// constant return) so Phase 21 of the ACT can prove the
// underlying runner was invoked exactly once.
type r6BCollectorMismatchingRunner struct {
	mu    sync.Mutex
	calls int
}

func (r *r6BCollectorMismatchingRunner) Run(
	context.Context,
	string,
	[]string,
	string,
	[]string,
) evidence.CommandResult {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return evidence.CommandResult{
		ExitCode: 0,
		Stdout:   []byte("EXEC_GATE_OBSERVED_STATUS:OK\n"),
	}
}

// Calls returns the number of runner invocations. The
// counter is incremented on every Run call so tests can
// assert the production integration invoked the runner
// exactly once even when the collector rejects the second
// capture.
func (r *r6BCollectorMismatchingRunner) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// TestR6BGateObservationFailureSurfaced proves the runner
// adapter emits V2CodeR6BGateObservationFailed when the
// GateCollector cannot produce a valid observation. The
// focused test invokes the integration with a runner that
// simulates a spawn failure and confirms the adapter
// wraps it in the typed r6b_gate_observation_failed code.
func TestR6BGateObservationFailureSurfaced(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	runner := &r6BRecordingRunner{spawnFail: true}
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
			RunID:          "r6b-c06-spawn",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err == nil {
		t.Fatalf("spawn failure must surface a typed gate error")
	}
	requireV2Code(t, err, V2CodeR6BGateObservationFailed)
}

// TestR6BGateClassificationFailSurfaced proves the adapter
// surfaces V2CodeR6BGateClassificationFailed when the
// classifier returns FAIL on a successful capture. The
// status MUST be FAILED (not OK); the production classifier
// only inspects findings when the status reports failure.
func TestR6BGateClassificationFailSurfaced(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	runner := &r6BRecordingRunner{stdoutField: []byte(
		"EXEC_GATE_OBSERVED_STATUS:FAILED\n" +
			"cmd/leamas/main.go:42:warning:rule-new:extra finding\n",
	)}
	collector := evidence.NewGateCollector(runner)
	_, _, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:           r6BStubBuildFn(t),
			NewCollectorFn:    func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:     runner,
			OutputRoot:        r6BOutputRoot(t),
			OutputName:        "leamas",
			RunID:             "r6b-c06-fail",
			EvidenceDir:       r6BEvidenceDir(t),
			GateACTOwnedPaths: []string{"cmd/leamas/**"},
		})
	if err == nil {
		t.Fatalf("independent classifier FAIL must surface a typed gate classification error")
	}
	requireV2Code(t, err, V2CodeR6BGateClassificationFailed)
}

// TestR6BGateClassificationUnavailableSurfaced proves the
// adapter surfaces V2CodeR6BGateClassificationUnavailable
// when the lane is missing (no status line in stdout) so
// the classifier's LaneMissing rule fires.
func TestR6BGateClassificationUnavailableSurfaced(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	runner := &r6BRecordingRunner{stdoutField: []byte("lane-lint: OK\n")}
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
			RunID:          "r6b-c06-unavail",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err == nil {
		t.Fatalf("independent classifier UNAVAILABLE must surface a typed gate classification error")
	}
	requireV2Code(t, err, V2CodeR6BGateClassificationUnavailable)
}

// TestR6BCollectorMismatchPropagates proves the production
// adapter surfaces a GateCollector identity mismatch as the
// typed ErrCollectorRequestMismatch error after the live
// capture has run.
//
// The test is structured so the SAME pinned collector
// observes a successful first capture (with a different
// SubjectRoot) and then the live integration calls
// Capture with the executor-bound SubjectRoot. The
// collector returns ErrCollectorRequestMismatch on the
// second call, the adapter wraps it as
// V2CodeR6BGateObservationFailed (Cause is preserved), and
// the underlying runner is invoked exactly once.
func TestR6BCollectorMismatchPropagates(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	firstSubjectRoot := filepath.Join(t.TempDir(), "subj-1")
	if err := os.MkdirAll(firstSubjectRoot, 0o700); err != nil {
		t.Fatalf("mkdir first: %v", err)
	}
	collectorRunner := &r6BCollectorMismatchingRunner{}
	collector := evidence.NewGateCollector(collectorRunner)
	if _, err := collector.Capture(context.Background(), evidence.GateCaptureRequest{
		RepositoryRoot: dir,
		EvidenceDir:    filepath.Join(t.TempDir(), "ev"),
		RunID:          "pre",
		SubjectRoot:    firstSubjectRoot,
		MakeExecutable: []string{"/bin/true"},
	}); err != nil {
		t.Fatalf("first capture: %v", err)
	}
	// Now invoke the integration with a different live
	// subject root. The production adapter binds the live
	// SubjectRoot from the executor; the second call must
	// surface ErrCollectorRequestMismatch as the wrapped
	// cause and V2CodeR6BGateObservationFailed as the
	// owning R6-B typed code.
	collectorRef := collector
	runner := &r6BRecordingRunner{}
	_, _, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			BuildFn:        r6BStubBuildFn(t),
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collectorRef },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-c06-mismatch",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	if err == nil {
		t.Fatalf("identity mismatch must surface a typed error")
	}
	requireV2Code(t, err, V2CodeR6BGateObservationFailed)
	if !errors.Is(err, evidence.ErrCollectorRequestMismatch) {
		t.Fatalf("error = %v, want ErrCollectorRequestMismatch", err)
	}
	if collectorRunner.Calls() != 1 {
		t.Fatalf("collector runner calls = %d, want 1", collectorRunner.Calls())
	}
}
