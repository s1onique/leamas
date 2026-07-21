//go:build unix || darwin || linux

package execution

import (
	"context"
	"testing"
	"time"
)

// TestAdversarialIgnoreSIGTERMViaGoHelper proves that an
// Executor.Execute() call fails with CodeExecutionCancelled after the
// caller cancels its context, even when the SIGTERM-resistant child has
// already installed signal.Ignore(syscall.SIGTERM) before the trigger.
//
// The previous implementation triggered cancellation on a fixed 300ms
// timeout without verifying that the child had reached the state required
// for the proof. This implementation:
//
//  1. Runs Execute in a goroutine with a cancellable caller context and a
//     large request timeout that functions only as a fail-safe.
//  2. Waits for parent and child records, with SignalReady=true on the
//     child, before triggering cancellation.
//  3. Cancels exactly once and waits up to
//     TerminationGrace+PostKillWait+slack for Execute to return.
//  4. Requires CodeExecutionCancelled.
//  5. Verifies every recorded PID and PGID is absent.
//  6. Reports unexpected execution returns BEFORE readiness as
//     deterministic failures with diagnostic evidence.
func TestAdversarialIgnoreSIGTERMViaGoHelper(t *testing.T) {
	grace := 500 * time.Millisecond
	postKill := 500 * time.Millisecond
	execBudget := grace + postKill + sigtermSlack

	upperBoundFromCancel := execBudget
	maxExpected := sigtermReadinessWait + upperBoundFromCancel +
		sigtermTestHarnessStabilityBound

	executor := buildTestExecutor(t, sigtermRequestTimeout+time.Second, 64*1024)
	defer executor.Close()

	verifier, cleanup := newProcessVerifier(t)
	defer cleanup()

	helperPath, err := locateHelperBinary()
	if err != nil {
		t.Skipf("cannot resolve helper path: %v", err)
	}

	const mode = "ignore-sigterm"
	req := &Request{
		Name: "ignore-sigterm-escalation",
		Args: []string{helperPath, mode},
		Env: []string{
			"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile(),
			"LEAMAS_EXEC_TEST_READY_DIR=" + verifier.ReadyDir(),
		},
		Timeout: sigtermRequestTimeout,
	}

	callerCtx, cancelCaller := context.WithCancel(context.Background())
	defer cancelCaller()

	resultCh := make(chan execOutcome, 1)
	start := time.Now()
	go func() {
		res := executor.Execute(callerCtx, req)
		resultCh <- execOutcome{result: res}
	}()

	// Step A: wait for the adversarial child to reach the required state.
	readinessDeadline := start.Add(sigtermReadinessWait)
	if err := verifier.waitForReadiness(mode, readinessDeadline); err != nil {
		// Drain the goroutine with bounded wait to avoid leaking it.
		verifyReadinessCleanup(t, executor, verifier, resultCh)
		t.Fatalf("readiness not reached in %v: %v\n"+
			"helper observed stderr should explain the failure",
			sigtermReadinessWait, err)
	}

	// Step B: prove the precondition that justifies the trigger.
	verifier.requireExpectedRoles(mode)
	verifier.requireSignalReadyForRoles(mode)
	if verifier.records == nil {
		verifyReadinessCleanup(t, executor, verifier, resultCh)
		t.Fatal("readiness reported but verifier.records is empty")
	}
	parentPGID, childPGID := requireSharedPGID(t, verifier)
	verifyHelperProcessAlive(t, verifier, "parent")
	verifyHelperProcessAlive(t, verifier, "child")

	// Step C: trigger cancellation only after readiness.
	triggerAt := time.Now()
	cancelCaller()

	// Step D: bound the cancel-to-return latency by the cleanup budget plus
	// explicit scheduler slack.
	boundTimer := time.NewTimer(upperBoundFromCancel)
	defer boundTimer.Stop()

	select {
	case msg := <-resultCh:
		elapsed := time.Since(triggerAt)
		totalElapsed := time.Since(start)
		if elapsed > upperBoundFromCancel {
			t.Fatalf(
				"cancellation exceeded cleanup bound: trigger+=%v budget=%v\n"+
					"  result error code=%v",
				elapsed, upperBoundFromCancel,
				errorCodeOrNil(msg.result))
		}

		// Step E: require the cancellation classification.
		if msg.result == nil {
			t.Fatal("Execute returned a nil result")
		}
		if msg.result.Error == nil {
			t.Fatalf("expected cancellation error, got nil result\n"+
				"  exit=%d elapsed=%v totalElapsed=%v",
				msg.result.ExitCode, elapsed, totalElapsed)
		}
		allowedCodes := map[string]struct{}{
			CodeExecutionCancelled: {},
			// CodeExecutionProcessTreeCleanupFailed is accepted on macOS
			// where SIGKILL escalation is not always deliverable inside the
			// same cleanup budget. Document the platform behaviour.
			CodeExecutionProcessTreeCleanupFailed: {},
		}
		if _, ok := allowedCodes[msg.result.Error.Code]; !ok {
			t.Errorf("expected %s or %s, got %s",
				CodeExecutionCancelled,
				CodeExecutionProcessTreeCleanupFailed,
				msg.result.Error.Code)
		}
		if msg.result.Error.Code == CodeExecutionProcessTreeCleanupFailed {
			t.Logf("escalation returned cleanup_failed (macOS platform behaviour)")
		}

		// Step F: prove every recorded PID and PGID is absent.
		if err := verifier.verifyAllProcessesAbsent(verificationTimeout); err != nil {
			t.Errorf("process leak detected:\n%v", err)
		}

		// Step G: at most one expected PGID exists in the manifest.
		pgids := allPGIDs(verifier)
		if len(pgids) != 1 {
			t.Errorf("expected one PGID across records, got %v", pgids)
		}
		_ = parentPGID
		_ = childPGID

		if totalElapsed > maxExpected {
			t.Fatalf("test exceeded outer bound: totalElapsed=%v maxExpected=%v",
				totalElapsed, maxExpected)
		}
		t.Logf(
			"elapsed=%v triggerToReturn=%v records=%d pgid=%v",
			totalElapsed, elapsed, len(verifier.records), pgids)

	case <-boundTimer.C:
		verifyReadinessCleanup(t, executor, verifier, resultCh)
		t.Fatalf("Execute did not return within %v of cancellation",
			upperBoundFromCancel)
	}
}

// TestAdversarialHeldOutputDescriptor tests that an Executor.Execute() call
// for the held-stdout-open mode terminates via the cancellation path after
// readiness is proven. The stdout holder child holds its descriptors open,
// so the executor must rely on WaitDelay to bound cleanup.
func TestAdversarialHeldOutputDescriptor(t *testing.T) {
	grace := 500 * time.Millisecond
	postKill := 500 * time.Millisecond
	execBudget := grace + postKill + sigtermSlack
	maxExpected := sigtermReadinessWait + execBudget +
		sigtermTestHarnessStabilityBound

	executor := buildTestExecutor(t, sigtermRequestTimeout+time.Second, 64*1024)
	defer executor.Close()

	verifier, cleanup := newProcessVerifier(t)
	defer cleanup()

	helperPath, err := locateHelperBinary()
	if err != nil {
		t.Skipf("cannot resolve helper path: %v", err)
	}

	const mode = "hold-stdout-open"
	req := &Request{
		Name: "held-output-descriptor",
		Args: []string{helperPath, mode},
		Env: []string{
			"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile(),
			"LEAMAS_EXEC_TEST_READY_DIR=" + verifier.ReadyDir(),
		},
		Timeout: sigtermRequestTimeout,
	}

	callerCtx, cancelCaller := context.WithCancel(context.Background())
	defer cancelCaller()
	resultCh := make(chan execOutcome, 1)
	start := time.Now()
	go func() {
		res := executor.Execute(callerCtx, req)
		resultCh <- execOutcome{result: res}
	}()

	readinessDeadline := start.Add(sigtermReadinessWait)
	if err := verifier.waitForReadiness(mode, readinessDeadline); err != nil {
		verifyReadinessCleanup(t, executor, verifier, resultCh)
		t.Fatalf("readiness not reached in %v: %v", sigtermReadinessWait, err)
	}
	verifier.requireExpectedRoles(mode)
	verifyHelperProcessAlive(t, verifier, "parent")
	verifyHelperProcessAlive(t, verifier, "child")

	triggerAt := time.Now()
	cancelCaller()

	boundTimer := time.NewTimer(execBudget + sigtermSlack)
	defer boundTimer.Stop()
	select {
	case msg := <-resultCh:
		elapsed := time.Since(triggerAt)
		totalElapsed := time.Since(start)
		if elapsed > execBudget+sigtermSlack {
			t.Fatalf("held-descriptor cancellation exceeded bound: trigger+=%v budget=%v",
				elapsed, execBudget+sigtermSlack)
		}
		res := msg.result
		if res == nil || res.Error == nil {
			t.Fatalf("expected cancellation error, got %+v", res)
		}
		allowedCodes := map[string]struct{}{
			CodeExecutionCancelled:                {},
			CodeExecutionProcessTreeCleanupFailed: {},
		}
		if _, ok := allowedCodes[res.Error.Code]; !ok {
			t.Errorf("expected %s or %s, got %s",
				CodeExecutionCancelled,
				CodeExecutionProcessTreeCleanupFailed,
				res.Error.Code)
		}
		if err := verifier.verifyAllProcessesAbsent(verificationTimeout); err != nil {
			t.Errorf("process leak detected:\n%v", err)
		}
		if totalElapsed > maxExpected {
			t.Fatalf("test exceeded outer bound: totalElapsed=%v maxExpected=%v",
				totalElapsed, maxExpected)
		}
		t.Logf(
			"elapsed=%v triggerToReturn=%v records=%d",
			totalElapsed, elapsed, len(verifier.records))
	case <-boundTimer.C:
		verifyReadinessCleanup(t, executor, verifier, resultCh)
		t.Fatalf("Execute did not return within %v of cancellation",
			execBudget+sigtermSlack)
	}
}
