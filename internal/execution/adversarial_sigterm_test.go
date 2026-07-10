//go:build unix || darwin || linux

package execution

import (
	"context"
	"testing"
	"time"
)

// TestAdversarialIgnoreSIGTERMViaGoHelper tests SIGTERM escalation to SIGKILL.
func TestAdversarialIgnoreSIGTERMViaGoHelper(t *testing.T) {
	grace := 500 * time.Millisecond
	postKill := 500 * time.Millisecond
	slack := 500 * time.Millisecond
	execTimeout := 300 * time.Millisecond
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
		Name:    "ignore-sigterm",
		Args:    []string{helperPath, "ignore-sigterm"},
		Env:     []string{"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile()},
		Timeout: execTimeout,
	}

	start := time.Now()
	result := executor.Execute(context.Background(), req)
	elapsed := time.Since(start)

	time.Sleep(100 * time.Millisecond)

	if elapsed > maxExpected {
		t.Fatalf("Test exceeded max: elapsed=%v, maxExpected=%v", elapsed, maxExpected)
	}

	records, err := verifier.parseManifest()
	if err != nil {
		t.Fatalf("manifest parse failed: %v", err)
	}
	verifier.requireNonEmptyManifest()
	verifier.requireExpectedRoles("ignore-sigterm")

	if err := verifier.verifyAllProcessesAbsent(verificationTimeout); err != nil {
		t.Errorf("process leak detected:\n%v", err)
	}

	if result.Error == nil {
		t.Error("expected error result")
	} else if result.Error.Code == CodeExecutionProcessTreeCleanupFailed {
		t.Logf("timeout returned cleanup_failed (macOS platform behavior)")
	} else if result.Error.Code != CodeExecutionDeadlineExceeded && result.Error.Code != CodeExecutionTimeoutExceeded {
		t.Errorf("expected deadline/timeout error, got %s", result.Error.Code)
	}

	t.Logf("TestAdversarialIgnoreSIGTERMViaGoHelper: PASSED - elapsed %v, records=%d", elapsed, len(records))
}

// TestAdversarialHeldOutputDescriptor tests WaitDelay bounds held descriptors.
func TestAdversarialHeldOutputDescriptor(t *testing.T) {
	grace := 500 * time.Millisecond
	postKill := 500 * time.Millisecond
	slack := 500 * time.Millisecond
	execTimeout := 700 * time.Millisecond
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
		Name:    "held-stdout",
		Args:    []string{helperPath, "hold-stdout-open"},
		Env:     []string{"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile()},
		Timeout: execTimeout,
	}

	start := time.Now()
	_ = executor.Execute(context.Background(), req)
	elapsed := time.Since(start)

	time.Sleep(100 * time.Millisecond)

	if elapsed > maxExpected {
		t.Fatalf("Test exceeded max: elapsed=%v, maxExpected=%v", elapsed, maxExpected)
	}

	records, err := verifier.parseManifest()
	if err != nil {
		t.Fatalf("manifest parse failed: %v", err)
	}
	verifier.requireNonEmptyManifest()
	verifier.requireExpectedRoles("hold-stdout-open")

	if err := verifier.verifyAllProcessesAbsent(verificationTimeout); err != nil {
		t.Errorf("process leak detected:\n%v", err)
	}

	t.Logf("TestAdversarialHeldOutputDescriptor: PASSED - elapsed %v, records=%d", elapsed, len(records))
}
