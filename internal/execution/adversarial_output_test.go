//go:build unix || darwin || linux

package execution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAdversarialOutputOverflowWithDescendants tests output overflow terminates descendant tree.
// Contract: OutputBytesRetained <= OutputLimit, OutputBytesObserved > OutputBytesRetained,
// OutputTruncated == true, Error.Code == execution_output_limit_exceeded.
//
// CORRECTION05 hardening:
//
//   - The helper must publish an output-flood-ready sentinel AFTER the
//     tree is established so the test cannot be satisfied by an
//     unobserved producer.
//   - The retained output must contain no "ERROR:" helper diagnostic.
//     The pre-CORRECTION05 waitChildOrFail semantic emitted an 84-byte
//     "child exited cleanly before expected test trigger" line that
//     itself satisfied the 64-byte cap and produced a false-positive
//     overflow proof.
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
		Args: []string{helperPath, "output-forever-grandchild"},
		Env: []string{
			"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile(),
			"LEAMAS_EXEC_TEST_READY_DIR=" + verifier.ReadyDir(),
		},
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
		t.Fatalf("expected error code %s, got %s",
			CodeExecutionOutputLimitExceeded, result.Error.Code)
	}

	records, err := verifier.parseManifest()
	if err != nil {
		t.Fatalf("manifest parse failed: %v", err)
	}
	verifier.requireNonEmptyManifest()

	// Require full 3-level tree (parent, child, grandchild).
	verifier.requireExpectedRoles("output-forever-grandchild")

	// CORRECTION05: require the output-flood-ready sentinel so the
	// test cannot be satisfied by a parent error message that itself
	// overflows the cap (the prior runChildOrFail bug). The helper
	// emits "<pid>.output-flood-ready" after the tree is established.
	parentPID := pidForRole(records, "parent")
	if parentPID == 0 {
		t.Fatal("no parent pid recorded")
	}
	readySentinel := filepath.Join(verifier.ReadyDir(),
		fmt.Sprintf("%d.output-flood-ready", parentPID))
	if _, err := os.Stat(readySentinel); err != nil {
		t.Errorf("output-flood-ready sentinel not observed at %s: %v",
			readySentinel, err)
	}

	// Require no ERROR: helper diagnostic in the retained output.
	// The pre-CORRECTION05 waitChildOrFail semantic emitted an
	// 84-byte "child exited cleanly" line that itself satisfied the
	// 64-byte cap; that failure must never recur.
	for _, line := range strings.Split(string(result.Stdout), "\n") {
		if strings.HasPrefix(line, "ERROR:") {
			t.Errorf("helper ERROR: diagnostic leaked into retained output: %q",
				line)
		}
	}

	// Verify all descendant PIDs and PGIDs are absent.
	if err := verifier.verifyAllProcessesAbsent(verificationTimeout); err != nil {
		t.Errorf("process leak detected:\n%v", err)
	}

	// Rely on Go's normal PASS reporting; do not emit an unconditional
	// PASSED log line.
	t.Logf("TestAdversarialOutputOverflowWithDescendants: elapsed=%v retained=%d limit=%d observed=%d records=%d",
		elapsed, result.OutputBytesRetained, result.OutputLimit, result.OutputBytesObserved, len(records))
}

// pidForRole returns the PID of the most recent record with the given
// role. Returns 0 if no record exists for the role.
func pidForRole(records []PIDRecord, role string) int {
	var pid int
	for _, rec := range records {
		if rec.Role == role {
			pid = rec.PID
		}
	}
	return pid
}
