// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestReentryRejection tests that nested Leamas execution is rejected.
func TestReentryRejection(t *testing.T) {
	// Set up environment variables to simulate nested execution
	os.Setenv(EnvRootID, "test-root-id")
	os.Setenv(EnvParentPID, "12345")
	os.Setenv(EnvGeneration, "1")

	defer func() {
		os.Unsetenv(EnvRootID)
		os.Unsetenv(EnvParentPID)
		os.Unsetenv(EnvGeneration)
	}()

	// Try to create a new execution root - should fail
	_, err := NewExecutionRoot()
	if err == nil {
		t.Fatal("expected nested execution to be rejected, but it was allowed")
	}

	// Check that error contains the nested execution code
	if !strings.Contains(err.Error(), "Leamas cannot be started") {
		t.Errorf("expected error containing 'Leamas cannot be started', got %v", err)
	}
}

// TestReentryAllowedForTests tests that re-entry can be allowed for testing.
func TestReentryAllowedForTests(t *testing.T) {
	// Set up environment variables to simulate nested execution
	os.Setenv(EnvRootID, "test-root-id")
	os.Setenv(EnvParentPID, "12345")
	os.Setenv(EnvGeneration, "1")

	defer func() {
		os.Unsetenv(EnvRootID)
		os.Unsetenv(EnvParentPID)
		os.Unsetenv(EnvGeneration)
	}()

	// With ReentryPolicyAllow, should succeed
	err := checkReentry(ReentryPolicyAllow)
	if err != nil {
		t.Fatalf("expected no error with ReentryPolicyAllow, got %v", err)
	}
}

// TestCycleDetection tests that execution cycles are detected.
func TestCycleDetection(t *testing.T) {
	detector := NewCycleDetector()
	fingerprint := ComputeFingerprint("make", []string{"gate"}, ".", "factory.verify")

	// First check should succeed
	err := detector.CheckAndTrack(fingerprint, "make gate")
	if err != nil {
		t.Fatalf("expected first check to succeed, got %v", err)
	}

	// Second check with same fingerprint should fail (cycle detected)
	err = detector.CheckAndTrack(fingerprint, "make gate")
	if err == nil {
		t.Fatal("expected cycle to be detected, but it was not")
	}

	execErr, ok := err.(*ExecutionError)
	if !ok {
		t.Fatalf("expected ExecutionError, got %T", err)
	}

	if execErr.Code != CodeExecutionCycleDetected {
		t.Errorf("expected error code %s, got %s", CodeExecutionCycleDetected, execErr.Code)
	}
}

// TestCycleDetectionUntrack tests that fingerprints can be untracked.
func TestCycleDetectionUntrack(t *testing.T) {
	detector := NewCycleDetector()
	fingerprint := ComputeFingerprint("go", []string{"test"}, ".", "unit.test")

	// Track and untrack
	err := detector.CheckAndTrack(fingerprint, "go test")
	if err != nil {
		t.Fatalf("expected first check to succeed, got %v", err)
	}

	detector.Untrack(fingerprint)

	// Should be able to track again after untracking
	err = detector.CheckAndTrack(fingerprint, "go test")
	if err != nil {
		t.Fatalf("expected check to succeed after untrack, got %v", err)
	}
}

// TestSemaphoreBoundedConcurrency tests that semaphore limits concurrency.
func TestSemaphoreBoundedConcurrency(t *testing.T) {
	sem := NewSemaphore(2)
	ctx := context.Background()

	// Acquire 2 permits
	ok1, _ := sem.Acquire(ctx, 1)
	ok2, _ := sem.Acquire(ctx, 1)
	if !ok1 || !ok2 {
		t.Fatal("expected both acquisitions to succeed")
	}

	if sem.Count() != 2 {
		t.Errorf("expected count to be 2, got %d", sem.Count())
	}

	// Third acquire should block (but we use a short timeout in a goroutine)
	done := make(chan bool)
	go func() {
		ok, _ := sem.Acquire(ctx, 1)
		done <- ok
	}()

	select {
	case <-done:
		t.Fatal("expected acquire to block, but it succeeded")
	case <-time.After(50 * time.Millisecond):
		// Expected: blocked
	}

	// Release one permit
	sem.Release(1, 1)

	// Now the third acquire should succeed
	select {
	case ok := <-done:
		if !ok {
			t.Fatal("expected acquire to succeed after release")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for acquire")
	}
}

// TestRequestCommandLine tests command line formatting.
func TestRequestCommandLine(t *testing.T) {
	req := &Request{
		Name: "test",
		Args: []string{"go", "test", "-v", "./..."},
	}

	if req.CommandName() != "go" {
		t.Errorf("expected command name 'go', got '%s'", req.CommandName())
	}

	expected := "go test -v ./..."
	if req.CommandLine() != expected {
		t.Errorf("expected command line '%s', got '%s'", expected, req.CommandLine())
	}
}

// TestResultSuccessFailure tests result status methods.
func TestResultSuccessFailure(t *testing.T) {
	// Successful result
	result := NewResult(0, time.Second, nil, nil, false)
	if !result.Success() {
		t.Error("expected result to be successful")
	}
	if result.Failed() {
		t.Error("expected result to not be failed")
	}

	// Failed result with non-zero exit
	result = NewResult(1, time.Second, nil, nil, false)
	if result.Success() {
		t.Error("expected result to not be successful")
	}
	if !result.Failed() {
		t.Error("expected result to be failed")
	}

	// Error result
	result = NewErrorResult(ErrConcurrencyExhausted(4))
	if result.Success() {
		t.Error("expected error result to not be successful")
	}
	if !result.Failed() {
		t.Error("expected error result to be failed")
	}
}

// TestCappedBuffer tests bounded output buffer.
func TestCappedBuffer(t *testing.T) {
	buf := NewCappedBuffer(100)

	// Write less than limit
	n, _ := buf.Write([]byte("hello"))
	if n != 5 {
		t.Errorf("expected to write 5 bytes, got %d", n)
	}

	if buf.Len() != 5 {
		t.Errorf("expected length 5, got %d", buf.Len())
	}

	if buf.Truncated() {
		t.Error("expected not truncated")
	}

	// Write enough to exceed limit
	data := make([]byte, 150)
	for i := range data {
		data[i] = 'x'
	}
	buf.Write(data)

	if !buf.Truncated() {
		t.Error("expected truncated")
	}

	if buf.Len() != 100 {
		t.Errorf("expected length 100, got %d", buf.Len())
	}
}

// TestConcurrentExecution tests concurrent command execution.
func TestConcurrentExecution(t *testing.T) {
	root := NewTestExecutionRoot()
	budget := DefaultBudget().WithMaxConcurrent(4)
	executor := NewExecutor(budget, root)

	var wg sync.WaitGroup
	results := make([]*Result, 4)

	// Use unique commands to avoid cycle detection collisions
	commands := []string{"echo 1", "echo 2", "echo 3", "echo 4"}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := &Request{
				Name:    fmt.Sprintf("concurrent-%d", idx),
				Args:    []string{"sh", "-c", commands[idx]},
				Timeout: 5 * time.Second,
			}
			results[idx] = executor.ExecuteSimple(req)
		}(i)
	}

	wg.Wait()

	// All should succeed
	for i, result := range results {
		if result == nil {
			t.Errorf("result %d was nil", i)
			continue
		}
		if result.Failed() && result.Error != nil {
			t.Errorf("result %d failed: %v", i, result.Error)
		}
	}
}

// TestErrorCodes tests that all error codes are defined correctly.
func TestErrorCodes(t *testing.T) {
	codes := []string{
		CodeNestedLeamasExecution,
		CodeExecutionCycleDetected,
		CodeExecutionDeadlineExceeded,
		CodeExecutionConcurrencyExhausted,
		CodeExecutionStartBudgetExhausted,
		CodeExecutionOutputLimitExceeded,
		CodeExecutionProcessTreeCleanupFailed,
		CodeExecutionInvalidUnboundedTimeout,
		CodeExecutionTimeoutExceeded,
		CodeExecutionCancelled,
		CodeExecutionCommandNotFound,
		CodeExecutionPermissionDenied,
		CodeExecutionUnknown,
	}

	for _, code := range codes {
		if code == "" {
			t.Error("found empty error code")
		}
	}
}

// TestFingerprintConsistency tests that fingerprints are consistent.
func TestFingerprintConsistency(t *testing.T) {
	fp1 := ComputeFingerprint("go", []string{"test", "-v"}, "/home/user/project", "unit.test")
	fp2 := ComputeFingerprint("go", []string{"test", "-v"}, "/home/user/project", "unit.test")

	if fp1 != fp2 {
		t.Error("expected fingerprints to be equal for same inputs")
	}

	fp3 := ComputeFingerprint("go", []string{"test"}, "/home/user/project", "unit.test")
	if fp1 == fp3 {
		t.Error("expected different fingerprints for different inputs")
	}
}
