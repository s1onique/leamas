// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"fmt"
)

// Error codes for execution failures.
const (
	CodeNestedLeamasExecution             = "nested_leamas_execution"
	CodeExecutionCycleDetected            = "execution_cycle_detected"
	CodeExecutionCancelled                = "execution_cancelled"
	CodeExecutionTimeoutExceeded          = "execution_timeout_exceeded"
	CodeExecutionDeadlineExceeded         = "execution_deadline_exceeded"
	CodeExecutionConcurrencyExhausted     = "execution_concurrency_exhausted"
	CodeExecutionStartBudgetExhausted     = "execution_start_budget_exhausted"
	CodeExecutionTaskDepthExceeded        = "execution_task_depth_exceeded"
	CodeExecutionOutputLimitExceeded      = "execution_output_limit_exceeded"
	CodeExecutionProcessTreeCleanupFailed = "execution_process_tree_cleanup_failed"
	CodeExecutionInvalidUnboundedTimeout  = "execution_invalid_unbounded_timeout"
	CodeExecutionInvalidRequest           = "execution_invalid_request"
	CodeExecutionCommandNotFound          = "execution_command_not_found"
	CodeExecutionPermissionDenied         = "execution_permission_denied"
	CodeExecutionUnknown                  = "execution_unknown"
)

// ExecutionError represents a structured execution failure.
type ExecutionError struct {
	Code            string      `json:"code"`
	Dimension       string      `json:"dimension,omitempty"`
	Limit           interface{} `json:"limit,omitempty"`
	Observed        interface{} `json:"observed,omitempty"`
	RootExecutionID string      `json:"root_execution_id,omitempty"`
	Command         string      `json:"command,omitempty"`
	ElapsedMs       int64       `json:"elapsed_ms,omitempty"`
	Message         string      `json:"message,omitempty"`
	Cause           error       `json:"-"`
}

func (e *ExecutionError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Code, e.Cause)
}

func (e *ExecutionError) Unwrap() error {
	return e.Cause
}

// NewExecutionError creates a new ExecutionError.
func NewExecutionError(code, msg string, cause error) *ExecutionError {
	return &ExecutionError{
		Code:    code,
		Message: msg,
		Cause:   cause,
	}
}

// WithDimension adds dimension info to the error.
func (e *ExecutionError) WithDimension(dim string, limit, observed interface{}) *ExecutionError {
	e.Dimension = dim
	e.Limit = limit
	e.Observed = observed
	return e
}

// WithContext adds execution context to the error.
func (e *ExecutionError) WithContext(rootID, command string, elapsedMs int64) *ExecutionError {
	e.RootExecutionID = rootID
	e.Command = command
	e.ElapsedMs = elapsedMs
	return e
}

// ErrNestedLeamasExecution is returned when Leamas detects it is running inside another Leamas execution.
var ErrNestedLeamasExecution = &ExecutionError{
	Code:    CodeNestedLeamasExecution,
	Message: "Leamas cannot be started from within a Leamas execution",
}

// ErrExecutionCycleDetected is returned when an execution cycle is detected.
var ErrExecutionCycleDetected = &ExecutionError{
	Code:    CodeExecutionCycleDetected,
	Message: "execution cycle detected",
}

// ErrStartBudgetExhausted is returned when the total starts budget is exhausted.
func ErrStartBudgetExhausted(limit, observed uint64) *ExecutionError {
	err := &ExecutionError{
		Code:    CodeExecutionStartBudgetExhausted,
		Message: fmt.Sprintf("start budget exhausted: limit=%d, observed=%d", limit, observed),
	}
	err.Dimension = "total_starts"
	err.Limit = limit
	err.Observed = observed
	return err
}

// ErrConcurrencyExhausted is returned when concurrent execution slots are exhausted.
func ErrConcurrencyExhausted(limit int64) *ExecutionError {
	err := &ExecutionError{
		Code:    CodeExecutionConcurrencyExhausted,
		Message: fmt.Sprintf("concurrency limit reached: %d", limit),
	}
	err.Dimension = "concurrent"
	err.Limit = limit
	err.Observed = limit
	return err
}

// ErrTaskDepthExceeded is returned when the task depth exceeds the configured limit.
func ErrTaskDepthExceeded(limit uint16, observed uint16) *ExecutionError {
	err := &ExecutionError{
		Code:    CodeExecutionTaskDepthExceeded,
		Message: fmt.Sprintf("task depth exceeded: limit=%d, observed=%d", limit, observed),
	}
	err.Dimension = "task_depth"
	err.Limit = limit
	err.Observed = observed
	return err
}

// ErrOutputLimitExceeded is returned when command output exceeds the limit.
func ErrOutputLimitExceeded(limit int64, observed int64, rootID, command string) *ExecutionError {
	err := &ExecutionError{
		Code:    CodeExecutionOutputLimitExceeded,
		Message: fmt.Sprintf("output limit exceeded: limit=%d bytes, observed=%d bytes", limit, observed),
	}
	err.Dimension = "output_bytes"
	err.Limit = limit
	err.Observed = observed
	err.RootExecutionID = rootID
	err.Command = command
	return err
}

// ErrDeadlineExceeded is returned when the deadline is exceeded.
var ErrDeadlineExceeded = &ExecutionError{
	Code:    CodeExecutionDeadlineExceeded,
	Message: "execution deadline exceeded",
}

// ErrTimeoutExceeded is returned when the timeout is exceeded.
func ErrTimeoutExceeded(timeout string) *ExecutionError {
	return &ExecutionError{
		Code:    CodeExecutionTimeoutExceeded,
		Message: fmt.Sprintf("command timeout exceeded: %s", timeout),
	}
}

// ErrUnboundedTimeout is returned when a timeout exceeds the maximum permitted.
func ErrUnboundedTimeout(timeout string) *ExecutionError {
	return &ExecutionError{
		Code:    CodeExecutionInvalidUnboundedTimeout,
		Message: fmt.Sprintf("timeout exceeds maximum permitted (%s)", timeout),
	}
}

// ErrProcessTreeCleanupFailed is returned when process tree cleanup fails.
func ErrProcessTreeCleanupFailed(pid int, detail string) *ExecutionError {
	return &ExecutionError{
		Code:    CodeExecutionProcessTreeCleanupFailed,
		Message: fmt.Sprintf("failed to cleanup process tree (pid=%d): %s", pid, detail),
	}
}

// ErrCancelled is returned when the context is cancelled.
var ErrCancelled = &ExecutionError{
	Code:    CodeExecutionCancelled,
	Message: "execution was cancelled",
}

// ErrInvalidRequest is returned when a request is invalid.
func ErrInvalidRequest(msg string) *ExecutionError {
	return &ExecutionError{
		Code:    CodeExecutionInvalidRequest,
		Message: msg,
	}
}
