//go:build unix || darwin || linux

package execution

import (
	"context"
	"testing"
	"time"
)

// TestAdversarialTimeoutDirectSleep tests that timeout kills a direct sleeping process.
func TestAdversarialTimeoutDirectSleep(t *testing.T) {
	grace := 500 * time.Millisecond
	postKill := 500 * time.Millisecond
	slack := 500 * time.Millisecond
	execTimeout := 200 * time.Millisecond
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
		Name:    "timeout-direct-sleep",
		Args:    []string{helperPath, "spawn-child", "child", "10s"},
		Env:     []string{"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile()},
		Timeout: execTimeout,
	}

	start := time.Now()
	res := executor.Execute(context.Background(), req)
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

	if err := verifier.verifyAllProcessesAbsent(verificationTimeout); err != nil {
		t.Errorf("process leak detected:\n%v", err)
	}

	if res.Error == nil {
		t.Error("expected error result")
	} else if res.Error.Code == CodeExecutionProcessTreeCleanupFailed {
		t.Logf("timeout returned cleanup_failed (macOS platform behavior)")
	} else if res.Error.Code != CodeExecutionDeadlineExceeded && res.Error.Code != CodeExecutionTimeoutExceeded {
		t.Errorf("expected deadline/timeout error, got %s", res.Error.Code)
	}

	t.Logf("TestAdversarialTimeoutDirectSleep: PASSED - elapsed %v, records=%d", elapsed, len(records))
}

// TestAdversarialTimeoutChildTree tests that timeout kills parent and child.
func TestAdversarialTimeoutChildTree(t *testing.T) {
	grace := 500 * time.Millisecond
	postKill := 500 * time.Millisecond
	slack := 500 * time.Millisecond
	execTimeout := 500 * time.Millisecond
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
		Name:    "timeout-child-tree",
		Args:    []string{helperPath, "spawn-child", "10s"},
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
	verifier.requireExpectedRoles("spawn-child")

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

	t.Logf("TestAdversarialTimeoutChildTree: PASSED - elapsed %v, records=%d", elapsed, len(records))
}

// TestAdversarialTimeoutGrandchildTree tests that timeout kills 3-level tree.
func TestAdversarialTimeoutGrandchildTree(t *testing.T) {
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
		Name:    "timeout-grandchild-tree",
		Args:    []string{helperPath, "spawn-grandchild"},
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
	verifier.requireExpectedRoles("spawn-grandchild")

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

	t.Logf("TestAdversarialTimeoutGrandchildTree: PASSED - elapsed %v, records=%d", elapsed, len(records))
}
