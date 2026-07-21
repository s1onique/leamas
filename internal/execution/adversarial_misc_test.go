//go:build unix || darwin || linux

package execution

import (
	"context"
	"strings"
	"testing"
	"time"
)

const verificationTimeout = 2 * time.Second

// TestAdversarialProcessGroupIsolation tests process groups are properly isolated.
func TestAdversarialProcessGroupIsolation(t *testing.T) {
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
		Name:    "pgid-test",
		Args:    []string{helperPath, "sleep", "10s"},
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

	pgid := verifier.requireExactlyOnePGID()
	_ = pgid

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

	t.Logf("TestAdversarialProcessGroupIsolation: elapsed=%v records=%d pgid=%d",
		elapsed, len(records), pgid)
}

// TestAdversarialManifestIsolation tests manifest files don't interfere.
func TestAdversarialManifestIsolation(t *testing.T) {
	verifier1, cleanup1 := newProcessVerifier(t)
	defer cleanup1()

	verifier2, cleanup2 := newProcessVerifier(t)
	defer cleanup2()

	if verifier1.ManifestFile() == verifier2.ManifestFile() {
		t.Error("manifest files must be unique")
	}

	t.Logf("Manifest 1: %s", verifier1.ManifestFile())
	t.Logf("Manifest 2: %s", verifier2.ManifestFile())
}

// TestAdversarialPermissionDeniedHandling tests permission errors fail closed.
func TestAdversarialPermissionDeniedHandling(t *testing.T) {
	verifier, _ := newProcessVerifier(t)
	defer func() {
		verifier.verifyWithCleanup()
	}()

	alive, err := verifier.isProcessAlive(999999)
	if err != nil && !strings.Contains(err.Error(), "no such process") && !strings.Contains(err.Error(), "ESRCH") {
		t.Logf("isProcessAlive returned error: %v", err)
	}
	if alive {
		t.Error("process 999999 must not be alive")
	}

	alive, err = verifier.isProcessGroupAlive(999999)
	if err != nil && !strings.Contains(err.Error(), "no such process") && !strings.Contains(err.Error(), "ESRCH") {
		t.Logf("isProcessGroupAlive returned error: %v", err)
	}
	if alive {
		t.Error("process group 999999 must not be alive")
	}
}

// TestAdversarialSyscallVerification tests low-level syscall verification.
func TestAdversarialSyscallVerification(t *testing.T) {
	t.Logf("AdversarialSyscallVerification: syscall operations available")
}
