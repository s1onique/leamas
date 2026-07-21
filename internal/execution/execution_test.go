// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestReentryRejection tests that nested Leamas execution is rejected.
func TestReentryRejection(t *testing.T) {
	os.Setenv(EnvRootID, "test-root-id")
	os.Setenv(EnvParentPID, "12345")
	os.Setenv(EnvGeneration, "1")
	defer func() {
		os.Unsetenv(EnvRootID)
		os.Unsetenv(EnvParentPID)
		os.Unsetenv(EnvGeneration)
	}()
	_, err := NewExecutionRoot()
	if err == nil {
		t.Fatal("expected nested execution to be rejected")
	}
	if !strings.Contains(err.Error(), "Leamas cannot be started") {
		t.Errorf("expected error containing 'Leamas cannot be started', got %v", err)
	}
}

// TestReentryAllowedForTests tests that re-entry can be allowed for testing.
func TestReentryAllowedForTests(t *testing.T) {
	os.Setenv(EnvRootID, "test-root-id")
	os.Setenv(EnvParentPID, "12345")
	os.Setenv(EnvGeneration, "1")
	defer func() {
		os.Unsetenv(EnvRootID)
		os.Unsetenv(EnvParentPID)
		os.Unsetenv(EnvGeneration)
	}()
	err := checkReentry(ReentryPolicyAllow)
	if err != nil {
		t.Fatalf("expected no error with ReentryPolicyAllow, got %v", err)
	}
}

// TestCycleDetection tests that execution cycles are detected.
func TestCycleDetection(t *testing.T) {
	detector := NewCycleDetector()
	fingerprint := ComputeFingerprint("make", []string{"gate"}, ".", "factory.verify")
	err := detector.CheckAndTrack(fingerprint, "make gate")
	if err != nil {
		t.Fatalf("expected first check to succeed, got %v", err)
	}
	err = detector.CheckAndTrack(fingerprint, "make gate")
	if err == nil {
		t.Fatal("expected cycle to be detected")
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
	err := detector.CheckAndTrack(fingerprint, "go test")
	if err != nil {
		t.Fatalf("expected first check to succeed, got %v", err)
	}
	detector.Untrack(fingerprint)
	err = detector.CheckAndTrack(fingerprint, "go test")
	if err != nil {
		t.Fatalf("expected check to succeed after untrack, got %v", err)
	}
}

// TestContextSemaphoreBoundedConcurrency tests that semaphore limits concurrency.
func TestContextSemaphoreBoundedConcurrency(t *testing.T) {
	sem := newContextSemaphore(2)
	ctx := context.Background()
	ok1, _ := sem.Acquire(ctx, 1)
	ok2, _ := sem.Acquire(ctx, 1)
	if !ok1 || !ok2 {
		t.Fatal("expected both acquisitions to succeed")
	}
	if sem.Count() != 2 {
		t.Errorf("expected count to be 2, got %d", sem.Count())
	}
	done := make(chan bool)
	go func() {
		ok, _ := sem.Acquire(ctx, 1)
		done <- ok
	}()
	select {
	case <-done:
		t.Fatal("expected acquire to block")
	case <-time.After(50 * time.Millisecond):
	}
	sem.Release(1)
	select {
	case ok := <-done:
		if !ok {
			t.Fatal("expected acquire to succeed after release")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for acquire")
	}
}

// TestContextSemaphoreCancellation tests that semaphore is cancelled by context.
func TestContextSemaphoreCancellation(t *testing.T) {
	sem := newContextSemaphore(1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	ok, _ := sem.Acquire(ctx, 1)
	if !ok {
		t.Fatal("expected first acquire to succeed")
	}
	done := make(chan error)
	go func() {
		_, err := sem.Acquire(ctx, 1)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancellation error")
		}
		if err != context.Canceled && err != context.DeadlineExceeded {
			t.Fatalf("expected context error, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for acquire cancellation")
	}
}

// TestRequestCommandLine tests command line formatting.
func TestRequestCommandLine(t *testing.T) {
	req := &Request{Name: "test", Args: []string{"go", "test", "-v", "./..."}}
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
	result := NewResult(0, time.Second, nil, nil, false)
	if !result.Success() {
		t.Error("expected result to be successful")
	}
	if result.Failed() {
		t.Error("expected result to not be failed")
	}
	result = NewResult(1, time.Second, nil, nil, false)
	if result.Success() {
		t.Error("expected result to not be successful")
	}
	if !result.Failed() {
		t.Error("expected result to be failed")
	}
	result = NewErrorResult(ErrConcurrencyExhausted(4))
	if result.Success() {
		t.Error("expected error result to not be successful")
	}
	if !result.Failed() {
		t.Error("expected error result to be failed")
	}
}

// TestBudgetValidation tests budget validation.
func TestBudgetValidation(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		budget  *Budget
		wantErr bool
	}{
		{name: "nil budget", budget: nil, wantErr: true},
		{name: "valid budget", budget: &Budget{
			Deadline: now.Add(time.Hour), MaxConcurrent: 4, MaxStarts: 64,
			MaxTaskDepth: 8, MaxOutputBytes: 8 * 1024 * 1024,
			TerminationGrace: 2 * time.Second, PostKillWait: 1 * time.Second,
		}, wantErr: false},
		{name: "zero concurrency", budget: &Budget{
			Deadline: now.Add(time.Hour), MaxConcurrent: 0, MaxStarts: 64,
			MaxTaskDepth: 8, MaxOutputBytes: 8 * 1024 * 1024,
		}, wantErr: true},
		{name: "exceeds max concurrency", budget: &Budget{
			Deadline: now.Add(time.Hour), MaxConcurrent: 100, MaxStarts: 64,
			MaxTaskDepth: 8, MaxOutputBytes: 8 * 1024 * 1024,
		}, wantErr: true},
		{name: "zero starts", budget: &Budget{
			Deadline: now.Add(time.Hour), MaxConcurrent: 4, MaxStarts: 0,
			MaxTaskDepth: 8, MaxOutputBytes: 8 * 1024 * 1024,
		}, wantErr: true},
		{name: "zero task depth", budget: &Budget{
			Deadline: now.Add(time.Hour), MaxConcurrent: 4, MaxStarts: 64,
			MaxTaskDepth: 0, MaxOutputBytes: 8 * 1024 * 1024,
		}, wantErr: true},
		{name: "past deadline", budget: &Budget{
			Deadline: now.Add(-time.Hour), MaxConcurrent: 4, MaxStarts: 64,
			MaxTaskDepth: 8, MaxOutputBytes: 8 * 1024 * 1024,
		}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.budget.Validate(now)
			if (err != nil) != tt.wantErr {
				t.Errorf("Budget.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSharedOutputBuffer tests the shared output buffer.
func TestSharedOutputBuffer(t *testing.T) {
	buf := newSharedOutputBuffer(100)
	stdout := buf.StdoutWriter()
	n, _ := stdout.Write([]byte("hello"))
	if n != 5 {
		t.Errorf("expected to write 5 bytes, got %d", n)
	}
	// Write more to stdout - should truncate at 100 total
	data := make([]byte, 150)
	for i := range data {
		data[i] = 'x'
	}
	stdout.Write(data)
	if !buf.Truncated() {
		t.Error("expected truncated")
	}
	totalObserved := buf.BytesObserved()
	if totalObserved != 155 {
		t.Errorf("expected 155 bytes observed, got %d", totalObserved)
	}
	totalRetained := buf.BytesRetained()
	if totalRetained > 100 {
		t.Errorf("expected retained bytes <= 100, got %d", totalRetained)
	}
}

// TestUpdateEnv tests environment variable update.
func TestUpdateEnv(t *testing.T) {
	env := []string{"FOO=bar", "BAZ=qux"}
	env = updateEnv(env, "FOO", "updated")
	if env[0] != "FOO=updated" {
		t.Errorf("expected FOO=updated, got %s", env[0])
	}
	env = updateEnv(env, "NEW", "value")
	found := false
	for _, e := range env {
		if e == "NEW=value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find NEW=value in env")
	}
}

// TestIsESRCH tests the ESRCH error detection.
func TestIsESRCH(t *testing.T) {
	if !isESRCH(nil) {
		t.Error("expected nil to be ESRCH")
	}
}

// TestErrorCodes tests that all error codes are defined correctly.
func TestErrorCodes(t *testing.T) {
	codes := []string{
		CodeNestedLeamasExecution, CodeExecutionCycleDetected,
		CodeExecutionDeadlineExceeded, CodeExecutionConcurrencyExhausted,
		CodeExecutionStartBudgetExhausted, CodeExecutionTaskDepthExceeded,
		CodeExecutionOutputLimitExceeded, CodeExecutionRetainedOutputPipe,
		CodeExecutionProcessTreeCleanupFailed,
		CodeExecutionInvalidUnboundedTimeout, CodeExecutionCancelled,
		CodeExecutionCommandNotFound, CodeExecutionPermissionDenied,
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
