//go:build unix || darwin || linux

package execution

import (
	"context"
	"testing"
	"time"
)

// TestAdversarialOutputOverflowWithDescendants tests output overflow terminates tree.
func TestAdversarialOutputOverflowWithDescendants(t *testing.T) {
	grace := 500 * time.Millisecond
	postKill := 500 * time.Millisecond
	slack := 500 * time.Millisecond
	execTimeout := 10 * time.Second
	outputCap := int64(512)
	maxExpected := calculateMaxTestDuration(execTimeout, grace, postKill, slack)

	executor := buildTestExecutor(t, maxExpected+time.Second, outputCap)
	defer executor.Close()

	verifier, cleanup := newProcessVerifier(t)
	defer cleanup()

	helperPath, err := locateHelperBinary()
	if err != nil {
		t.Skipf("cannot resolve helper path: %v", err)
	}

	req := &Request{
		Name:      "output-overflow-tree",
		Args:      []string{helperPath, "output-forever-child"},
		Env:       []string{"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile()},
		OutputCap: outputCap,
		Timeout:   execTimeout,
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

	if err := verifier.verifyAllProcessesAbsent(verificationTimeout); err != nil {
		t.Errorf("process leak detected:\n%v", err)
	}

	if result.OutputTruncated && result.OutputBytesRetained > result.OutputLimit {
		t.Errorf("OutputBytesRetained (%d) exceeds OutputLimit (%d)",
			result.OutputBytesRetained, result.OutputLimit)
	}

	if result.Error == nil {
		t.Error("expected error result")
	} else if result.Error.Code != CodeExecutionOutputLimitExceeded &&
		result.Error.Code != CodeExecutionDeadlineExceeded {
		t.Errorf("expected output_limit_exceeded or deadline_exceeded, got %s", result.Error.Code)
	} else {
		t.Logf("overflow test got error: %s", result.Error.Code)
	}

	t.Logf("TestAdversarialOutputOverflowWithDescendants: PASSED - elapsed=%v, retained=%d, limit=%d, records=%d",
		elapsed, result.OutputBytesRetained, result.OutputLimit, len(records))
}
