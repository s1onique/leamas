//go:build windows

// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"context"
	"fmt"
	"time"
)

// Executor represents a bounded command executor on Windows.
// Windows execution is not supported - this stub returns errors.
type Executor struct {
	budget *Budget
	root   *ExecutionRoot
}

// NewExecutor creates a new bounded executor.
func NewExecutor(budget *Budget, root *ExecutionRoot) (*Executor, error) {
	if err := budget.Validate(time.Now()); err != nil {
		return nil, fmt.Errorf("invalid budget: %w", err)
	}
	return &Executor{
		budget: budget,
		root:   root,
	}, nil
}

// Budget returns the execution budget.
func (e *Executor) Budget() *Budget {
	return e.budget
}

// Execute runs a bounded command and returns an error.
// Windows execution is not supported.
func (e *Executor) Execute(_ context.Context, req *Request) *Result {
	err := NewExecutionError(
		CodeExecutionUnknown,
		"execution is not supported on Windows",
		nil,
	)
	if req != nil && len(req.Args) > 0 {
		err.Command = req.CommandLine()
	}
	if e.root != nil {
		err.RootExecutionID = e.root.ID
	}
	return NewErrorResult(err)
}

// ExecuteSimple is a convenience method for simple execution without context.
func (e *Executor) ExecuteSimple(req *Request) *Result {
	return e.Execute(context.Background(), req)
}

// WaitForCompletion waits for all pending executions to complete.
func (e *Executor) WaitForCompletion(_ context.Context) error {
	return NewExecutionError(CodeExecutionUnknown, "execution is not supported on Windows", nil)
}

// Close releases all resources.
func (e *Executor) Close() error {
	return nil
}

// Stats returns current executor statistics.
func (e *Executor) Stats() (starts, active int64) {
	return 0, 0
}
