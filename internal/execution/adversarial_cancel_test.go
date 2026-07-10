//go:build unix || darwin || linux

package execution

import (
	"context"
	"testing"
	"time"
)

// TestAdversarialCallerCancellation tests that caller cancellation kills the tree.
func TestAdversarialCallerCancellation(t *testing.T) {
	grace := 500 * time.Millisecond
	postKill := 500 * time.Millisecond
	slack := 500 * time.Millisecond
	execTimeout := 5 * time.Second
	cancelDelay := 1 * time.Second // Must be less than execTimeout to ensure running when cancelled
	maxExpected := calculateMaxTestDuration(cancelDelay, grace, postKill, slack)

	executor := buildTestExecutor(t, maxExpected+time.Second, 64*1024*1024) // Large buffer to avoid overflow
	defer executor.Close()

	verifier, cleanup := newProcessVerifier(t)
	defer cleanup()

	helperPath, err := locateHelperBinary()
	if err != nil {
		t.Skipf("cannot resolve helper path: %v", err)
	}

	// Use sleep-grandchild: parent -> child -> grandchild, all sleeping forever
	// No output is produced, so this isolates cancellation from overflow.
	req := &Request{
		Name:    "cancellation-tree",
		Args:    []string{helperPath, "sleep-grandchild"},
		Env:     []string{"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile()},
		Timeout: execTimeout,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(cancelDelay)
		cancel()
	}()

	start := time.Now()
	result := executor.Execute(ctx, req)
	elapsed := time.Since(start)

	if elapsed > maxExpected {
		t.Fatalf("Test exceeded max: elapsed=%v, maxExpected=%v", elapsed, maxExpected)
	}

	records, err := verifier.parseManifest()
	if err != nil {
		t.Fatalf("manifest parse failed: %v", err)
	}
	verifier.requireNonEmptyManifest()
	verifier.requireExpectedRoles("sleep-grandchild")

	if err := verifier.verifyAllProcessesAbsent(verificationTimeout); err != nil {
		t.Errorf("process leak detected:\n%v", err)
	}

	// Require exact cancellation error code
	if result.Error == nil {
		t.Error("expected error result")
	} else if result.Error.Code != CodeExecutionCancelled {
		t.Errorf("expected CodeExecutionCancelled, got %s", result.Error.Code)
	}

	t.Logf("TestAdversarialCallerCancellation: PASSED - elapsed %v, records=%d", elapsed, len(records))
}

// TestAdversarialNonZeroExitWithChild tests non-zero exit doesn't leak processes.
func TestAdversarialNonZeroExitWithChild(t *testing.T) {
	grace := 500 * time.Millisecond
	postKill := 500 * time.Millisecond
	slack := 500 * time.Millisecond
	execTimeout := 2 * time.Second
	maxExpected := calculateMaxTestDuration(execTimeout, grace, postKill, slack)

	executor := buildTestExecutor(t, maxExpected+time.Second, 64*1024)
	defer executor.Close()

	verifier, cleanup := newProcessVerifier(t)
	defer cleanup()

	helperPath, err := locateHelperBinary()
	if err != nil {
		t.Skipf("cannot resolve helper path: %v", err)
	}

	req := &Request{
		Name:    "exit-nonzero-child",
		Args:    []string{helperPath, "exit-nonzero-child"},
		Env:     []string{"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile()},
		Timeout: execTimeout,
	}

	start := time.Now()
	result := executor.Execute(context.Background(), req)
	elapsed := time.Since(start)

	if elapsed > maxExpected {
		t.Fatalf("Test exceeded max: elapsed=%v, maxExpected=%v", elapsed, maxExpected)
	}

	records, err := verifier.parseManifest()
	if err != nil {
		t.Fatalf("manifest parse failed: %v", err)
	}
	verifier.requireNonEmptyManifest()

	if err := verifier.verifyAllProcessesAbsent(verificationTimeout); err != nil {
		t.Errorf("process leak detected:\n%v", err)
	}

	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", result.ExitCode)
	}

	if result.Error != nil {
		t.Errorf("expected no error for non-zero exit, got %v", result.Error)
	}

	t.Logf("TestAdversarialNonZeroExitWithChild: PASSED - exit code %d, records=%d", result.ExitCode, len(records))
}
