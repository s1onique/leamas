//go:build windows

// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"context"
	"sync"
)

// Executor represents a bounded command executor (Windows stub).
// All operations return an error indicating Windows is unsupported.
type Executor struct {
	budget *Budget
	root   *ExecutionRoot
	mu     sync.Mutex
}

// NewExecutor creates a new bounded executor.
func NewExecutor(budget *Budget, root *ExecutionRoot) *Executor {
	return &Executor{
		budget: budget,
		root:   root,
	}
}

// Budget returns the execution budget.
func (e *Executor) Budget() *Budget {
	return e.budget
}

// Execute runs a bounded command and returns the result.
func (e *Executor) Execute(_ context.Context, req *Request) *Result {
	err := NewExecutionError(
		CodeExecutionUnknown,
		"execution is unsupported on Windows",
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
	return NewExecutionError(CodeExecutionUnknown, "execution is unsupported on Windows", nil)
}

// Close releases all resources.
func (e *Executor) Close() error {
	return nil
}

// Stats returns current executor statistics.
func (e *Executor) Stats() (starts, active int64) {
	return 0, 0
}
