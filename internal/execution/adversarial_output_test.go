//go:build unix || darwin || linux

package execution

import (
	"context"
	"testing"
	"time"
)

// TestAdversarialOutputOverflowWithDescendants tests output overflow terminates descendant tree.
// Contract: OutputBytesRetained <= OutputLimit, OutputBytesObserved > OutputBytesRetained,
// OutputTruncated == true, Error.Code == execution_output_limit_exceeded
func TestAdversarialOutputOverflowWithDescendants(t *testing.T) {
	grace := 500 * time.Millisecond
	postKill := 500 * time.Millisecond
	slack := 500 * time.Millisecond
	execTimeout := 10 * time.Second
	// Use small cap (64 bytes) so fast 1-byte writes overflow quickly
	outputCap := int64(64)
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
		Name: "output-overflow-tree",
		// Use output-forever-grandchild: parent spawns child which spawns grandchild
		// that writes 1 byte at a time, creating a multi-level descendant tree
		Args:      []string{helperPath, "output-forever-grandchild"},
		Env:       []string{"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile()},
		OutputCap: outputCap,
		Timeout:   execTimeout,
	}

	start := time.Now()
	result := executor.Execute(context.Background(), req)
	elapsed := time.Since(start)

	// Strict contract assertions
	if result.Error == nil {
		t.Fatal("expected error result, got nil")
	}

	if !result.OutputTruncated {
		t.Fatal("expected output to be truncated")
	}

	if result.OutputBytesRetained > result.OutputLimit {
		t.Fatalf("OutputBytesRetained (%d) exceeds OutputLimit (%d)",
			result.OutputBytesRetained, result.OutputLimit)
	}

	if result.OutputBytesObserved <= result.OutputBytesRetained {
		t.Fatalf("expected OutputBytesObserved (%d) > OutputBytesRetained (%d)",
			result.OutputBytesObserved, result.OutputBytesRetained)
	}

	if elapsed >= execTimeout {
		t.Fatalf("expected elapsed (%v) < execTimeout (%v)", elapsed, execTimeout)
	}

	// Require exact output limit exceeded error code
	if result.Error.Code != CodeExecutionOutputLimitExceeded {
		t.Fatalf("expected error code %s, got %s", CodeExecutionOutputLimitExceeded, result.Error.Code)
	}

	records, err := verifier.parseManifest()
	if err != nil {
		t.Fatalf("manifest parse failed: %v", err)
	}
	verifier.requireNonEmptyManifest()

	// Require full 3-level tree (parent, child, grandchild)
	verifier.requireExpectedRoles("output-forever-grandchild")

	// Verify all descendant PIDs and PGIDs are absent
	if err := verifier.verifyAllProcessesAbsent(verificationTimeout); err != nil {
		t.Errorf("process leak detected:\n%v", err)
	}

	// Rely on Go's normal PASS reporting; do not emit an unconditional
	// PASSED log line.
	t.Logf("TestAdversarialOutputOverflowWithDescendants: elapsed=%v retained=%d limit=%d observed=%d records=%d",
		elapsed, result.OutputBytesRetained, result.OutputLimit, result.OutputBytesObserved, len(records))
}
