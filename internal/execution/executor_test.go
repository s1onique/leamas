//go:build unix || darwin || linux

package execution

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestConcurrentExecution tests concurrent command execution.
func TestConcurrentExecution(t *testing.T) {
	root := NewTestExecutionRoot()
	budget := DefaultBudget().WithMaxConcurrent(4)
	executor, err := NewExecutor(budget, root)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

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

// TestTaskDepthEnforcement tests that task depth is enforced.
func TestTaskDepthEnforcement(t *testing.T) {
	root := NewTestExecutionRoot()
	budget := &Budget{
		Deadline:         time.Now().Add(time.Minute),
		MaxConcurrent:    4,
		MaxStarts:        64,
		MaxTaskDepth:     3,
		MaxOutputBytes:   8 * 1024 * 1024,
		TerminationGrace: 2 * time.Second,
		PostKillWait:     1 * time.Second,
	}
	executor, err := NewExecutor(budget, root)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	// Request with depth at limit should succeed
	req := &Request{
		Name:      "test",
		Args:      []string{"echo", "test"},
		TaskDepth: 3,
		Timeout:   5 * time.Second,
	}
	result := executor.ExecuteSimple(req)
	if result.Error != nil {
		t.Errorf("expected depth=3 to succeed, got error: %v", result.Error)
	}

	// Request with depth over limit should fail
	req = &Request{
		Name:      "test",
		Args:      []string{"echo", "test"},
		TaskDepth: 5,
		Timeout:   5 * time.Second,
	}
	result = executor.ExecuteSimple(req)
	if result.Error == nil {
		t.Error("expected depth=5 to fail, but it succeeded")
	}
	if result.Error.Code != CodeExecutionTaskDepthExceeded {
		t.Errorf("expected error code %s, got %s", CodeExecutionTaskDepthExceeded, result.Error.Code)
	}
}

// TestExecutorTimeoutEnforcement tests that timeout is enforced by executor.
func TestExecutorTimeoutEnforcement(t *testing.T) {
	root := NewTestExecutionRoot()
	budget := DefaultBudget()
	executor, err := NewExecutor(budget, root)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	// Request with short timeout should timeout
	req := &Request{
		Name:    "sleep",
		Args:    []string{"sleep", "10"},
		Timeout: 100 * time.Millisecond,
	}

	start := time.Now()
	result := executor.ExecuteSimple(req)
	elapsed := time.Since(start)

	if result.Error == nil {
		t.Error("expected timeout/deadline error, got nil")
	} else {
		// Either deadline exceeded, timeout exceeded, or cleanup failed is acceptable
		if result.Error.Code != CodeExecutionDeadlineExceeded && result.Error.Code != CodeExecutionTimeoutExceeded && result.Error.Code != CodeExecutionProcessTreeCleanupFailed {
			t.Errorf("expected deadline/timeout error code, got %s", result.Error.Code)
		}
	}

	// Should complete within timeout + grace + some scheduler tolerance
	maxAllowed := req.Timeout + budget.TerminationGrace + 500*time.Millisecond
	if elapsed > maxAllowed {
		t.Errorf("execution took %v, expected < %v", elapsed, maxAllowed)
	}
}

// TestStartBudgetEnforcement tests that start budget is enforced.
func TestStartBudgetEnforcement(t *testing.T) {
	root := NewTestExecutionRoot()
	budget := &Budget{
		Deadline:         time.Now().Add(time.Minute),
		MaxConcurrent:    1,
		MaxStarts:        3,
		MaxTaskDepth:     8,
		MaxOutputBytes:   8 * 1024 * 1024,
		TerminationGrace: 1 * time.Second,
		PostKillWait:     500 * time.Millisecond,
	}
	executor, err := NewExecutor(budget, root)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	// Execute up to the limit
	for i := 0; i < 3; i++ {
		req := &Request{
			Name:    fmt.Sprintf("test-%d", i),
			Args:    []string{"echo", "test"},
			Timeout: 5 * time.Second,
		}
		result := executor.ExecuteSimple(req)
		if result.Error != nil {
			t.Errorf("attempt %d: unexpected error: %v", i, result.Error)
		}
	}

	// Next attempt should fail due to exhausted start budget
	req := &Request{
		Name:    "test-final",
		Args:    []string{"echo", "test"},
		Timeout: 5 * time.Second,
	}
	result := executor.ExecuteSimple(req)
	if result.Error == nil {
		t.Error("expected start budget exhaustion error, got nil")
	}
	if result.Error.Code != CodeExecutionStartBudgetExhausted {
		t.Errorf("expected error code %s, got %s", CodeExecutionStartBudgetExhausted, result.Error.Code)
	}
}

// TestExecutorOutputLimitEnforcement tests that output limit is enforced by executor.
func TestExecutorOutputLimitEnforcement(t *testing.T) {
	root := NewTestExecutionRoot()
	budget := &Budget{
		Deadline:         time.Now().Add(time.Minute),
		MaxConcurrent:    4,
		MaxStarts:        64,
		MaxTaskDepth:     8,
		MaxOutputBytes:   1024, // 1KB limit
		TerminationGrace: 1 * time.Second,
		PostKillWait:     500 * time.Millisecond,
	}
	executor, err := NewExecutor(budget, root)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	// Request that produces more output than limit
	req := &Request{
		Name:      "lots of output",
		Args:      []string{"sh", "-c", "yes | head -n 10000"},
		OutputCap: 1024,
		Timeout:   5 * time.Second,
	}

	result := executor.ExecuteSimple(req)

	// Should be truncated or failed
	if result.OutputBytesObserved > result.OutputBytesRetained {
		if !result.OutputTruncated {
			t.Error("expected output to be truncated")
		}
	}

	// Combined buffer writes to both stdout and stderr, so retained = 2x cap
	// The cap applies per-stream, not combined
	// Each stream (stdout, stderr) can retain up to OutputCap bytes
	maxRetained := req.OutputCap * 2 // both streams
	if result.OutputBytesRetained > maxRetained {
		t.Errorf("retained bytes %d exceeds max per-stream limit %d", result.OutputBytesRetained, maxRetained)
	}
}
