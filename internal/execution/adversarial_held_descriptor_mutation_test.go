//go:build unix || darwin || linux

package execution

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestAdversarialHeldDescriptorCleanupMutationRejected proves that replacing
// natural-exit process-group cleanup with a no-op leaves the absence contract
// red. The test then performs explicit reference cleanup and does not pass via
// an emergency cleanup path.
func TestAdversarialHeldDescriptorCleanupMutationRejected(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux /proc descriptor proof is required")
	}
	executor := buildRetainedPipeExecutor(t)
	defer executor.Close()
	executor.retainedOutputCleanup = func(int, *Request) *ExecutionError {
		return nil
	}
	verifier, _ := newProcessVerifier(t)
	helperPath, err := locateHelperBinary()
	if err != nil {
		t.Fatalf("locate helper: %v", err)
	}
	req := &Request{
		Name:    "held-descriptor-cleanup-mutation",
		Args:    []string{helperPath, retainedPipeMode},
		Env:     []string{"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile(), "LEAMAS_EXEC_TEST_READY_DIR=" + verifier.ReadyDir()},
		Timeout: retainedPipeTimeout,
	}
	resultCh := make(chan *Result, 1)
	go func() { resultCh <- executor.Execute(context.Background(), req) }()
	handoff := waitForRetainedPipeHandoff(t, verifier)
	guard := &retainedProcessGroupGuard{}
	guard.arm(handoff.Parent.PGID)
	defer guard.emergencyCleanup()

	var result *Result
	select {
	case result = <-resultCh:
	case <-time.After(retainedPipeWaitDelay + retainedPipeUpperSlack):
		t.Fatal("mutated Execute did not return")
	}
	if result == nil || result.Error == nil ||
		result.Error.Code != CodeExecutionRetainedOutputPipe ||
		!result.OutputIncomplete || result.ExitCode != 0 {
		t.Fatalf("mutation did not reach retained-output boundary: %+v", result)
	}
	absenceErr := verifier.verifyAllProcessesAbsent(100 * time.Millisecond)
	if absenceErr == nil {
		t.Fatal("no-op cleanup mutation escaped the process-absence contract")
	}
	if !strings.Contains(absenceErr.Error(), "still alive") {
		t.Fatalf("mutation failed for unexpected reason: %v", absenceErr)
	}

	if err := cleanupRetainedProcessGroup(handoff.Parent.PGID,
		50*time.Millisecond, time.Second); err != nil {
		t.Fatalf("explicit mutation cleanup: %v", err)
	}
	if err := verifier.verifyAllProcessesAbsent(time.Second); err != nil {
		t.Fatalf("mutation cleanup left processes: %v", err)
	}
	guard.disarm()
	if guard.emergencyUsed {
		t.Fatal("mutation proof used emergency cleanup")
	}
}
