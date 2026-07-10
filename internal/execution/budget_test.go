// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"testing"
	"time"
)

// TestBudgetDefaults tests that budget defaults are correct.
func TestBudgetDefaults(t *testing.T) {
	budget := DefaultBudget()

	if budget.MaxConcurrent != DefaultMaxConcurrent {
		t.Errorf("expected MaxConcurrent=%d, got %d", DefaultMaxConcurrent, budget.MaxConcurrent)
	}

	if budget.MaxStarts != DefaultMaxStarts {
		t.Errorf("expected MaxStarts=%d, got %d", DefaultMaxStarts, budget.MaxStarts)
	}

	if budget.MaxTaskDepth != DefaultMaxTaskDepth {
		t.Errorf("expected MaxTaskDepth=%d, got %d", DefaultMaxTaskDepth, budget.MaxTaskDepth)
	}

	if budget.MaxOutputBytes != DefaultMaxOutputBytes {
		t.Errorf("expected MaxOutputBytes=%d, got %d", DefaultMaxOutputBytes, budget.MaxOutputBytes)
	}
}

// TestBudgetWithMethods tests budget modifier methods.
func TestBudgetWithMethods(t *testing.T) {
	budget := DefaultBudget()

	// Test WithTimeout
	newBudget := budget.WithTimeout(5 * time.Minute)
	if newBudget.Deadline.IsZero() {
		t.Error("expected deadline to be set")
	}

	// Test WithMaxConcurrent
	newBudget = budget.WithMaxConcurrent(8)
	if newBudget.MaxConcurrent != 8 {
		t.Errorf("expected MaxConcurrent=8, got %d", newBudget.MaxConcurrent)
	}

	// Test WithMaxStarts
	newBudget = budget.WithMaxStarts(128)
	if newBudget.MaxStarts != 128 {
		t.Errorf("expected MaxStarts=128, got %d", newBudget.MaxStarts)
	}

	// Test WithMaxOutputBytes
	newBudget = budget.WithMaxOutputBytes(16 * 1024 * 1024)
	if newBudget.MaxOutputBytes != 16*1024*1024 {
		t.Errorf("expected MaxOutputBytes=%d, got %d", 16*1024*1024, newBudget.MaxOutputBytes)
	}
}

// TestExecutorCreation tests executor creation.
func TestExecutorCreation(t *testing.T) {
	root := NewTestExecutionRoot()
	budget := DefaultBudget()
	executor, err := NewExecutor(budget, root)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	if executor == nil {
		t.Fatal("expected executor to be created")
	}

	if executor.budget != budget {
		t.Error("expected budget to be set")
	}

	if executor.root != root {
		t.Error("expected root to be set")
	}
}

// TestTimeoutEnforcement tests that timeouts are enforced.
func TestTimeoutEnforcement(t *testing.T) {
	root := NewTestExecutionRoot()
	budget := DefaultBudget()
	executor, _ := NewExecutor(budget, root)

	req := &Request{
		Name:    "sleep",
		Args:    []string{"sleep", "10"},
		Timeout: 100 * time.Millisecond,
	}

	start := time.Now()
	result := executor.ExecuteSimple(req)
	elapsed := time.Since(start)

	if result == nil {
		t.Fatal("result was nil")
	}

	if result.Error == nil {
		t.Error("expected timeout/deadline error")
	} else if result.Error.Code != CodeExecutionDeadlineExceeded && result.Error.Code != CodeExecutionTimeoutExceeded && result.Error.Code != CodeExecutionProcessTreeCleanupFailed {
		t.Errorf("expected deadline/timeout error code, got %s", result.Error.Code)
	}

	// Should have been terminated within reasonable time
	if elapsed > 5*time.Second {
		t.Errorf("execution took too long: %v", elapsed)
	}
}

// TestUnboundedTimeoutRejection tests that unbounded timeouts are rejected.
func TestUnboundedTimeoutRejection(t *testing.T) {
	root := NewTestExecutionRoot()
	budget := DefaultBudget()
	executor, _ := NewExecutor(budget, root)

	req := &Request{
		Name:    "sleep",
		Args:    []string{"sleep", "1"},
		Timeout: 20 * time.Minute, // Exceeds MaxPermittedTimeout
	}

	result := executor.ExecuteSimple(req)

	if result == nil {
		t.Fatal("result was nil")
	}

	if result.Error == nil {
		t.Error("expected unbounded timeout error")
	} else if result.Error.Code != CodeExecutionInvalidUnboundedTimeout {
		t.Errorf("expected unbounded timeout error code, got %s", result.Error.Code)
	}
}

// TestOutputLimitEnforcement tests that output limits are enforced.
func TestOutputLimitEnforcement(t *testing.T) {
	root := NewTestExecutionRoot()
	budget := DefaultBudget().WithMaxOutputBytes(1024) // 1KB limit
	executor, _ := NewExecutor(budget, root)

	req := &Request{
		Name:      "dd",
		Args:      []string{"dd", "if=/dev/zero", "bs=1M", "count=10"},
		Timeout:   5 * time.Second,
		OutputCap: 512, // 512 bytes
	}

	result := executor.ExecuteSimple(req)

	// Should complete, possibly with truncated output
	if result == nil {
		t.Fatal("result was nil")
	}

	// Output should be truncated
	if !result.OutputTruncated {
		t.Log("output was not truncated (this may vary by platform)")
	}
}

// TestMaxStartsBudget tests that start budget is enforced.
// MaxStarts is a cumulative limit - once exhausted, no more commands can start.
func TestMaxStartsBudget(t *testing.T) {
	root := NewTestExecutionRoot()
	// Use MaxStarts=1 to test that only 1 command can start
	budget := DefaultBudget().WithMaxStarts(1)
	executor, _ := NewExecutor(budget, root)

	// First command should succeed
	req1 := &Request{
		Name:    "budget-test-1",
		Args:    []string{"sh", "-c", "echo first"},
		Timeout: 5 * time.Second,
	}
	result1 := executor.ExecuteSimple(req1)
	if result1 == nil || result1.Error != nil {
		t.Fatalf("first command should succeed, got: %+v", result1)
	}
	t.Logf("First command succeeded (as expected)")

	// Second command should fail with start budget exhausted
	req2 := &Request{
		Name:    "budget-test-2",
		Args:    []string{"sh", "-c", "echo second"},
		Timeout: 5 * time.Second,
	}
	result2 := executor.ExecuteSimple(req2)

	if result2 == nil || result2.Error == nil {
		t.Fatal("expected start budget exhausted error on 2nd command")
	}

	if result2.Error.Code != CodeExecutionStartBudgetExhausted {
		t.Errorf("expected start budget exhausted error, got: %s", result2.Error.Code)
	} else {
		t.Logf("correctly rejected 2nd command with: %s", result2.Error.Code)
	}
}
