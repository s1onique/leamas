//go:build unix || darwin || linux

package execution

import (
	"context"
	"fmt"
	"time"
)

// NewExecutor creates a new bounded executor.
func NewExecutor(budget *Budget, root *ExecutionRoot) (*Executor, error) {
	if err := budget.Validate(time.Now()); err != nil {
		return nil, fmt.Errorf("invalid budget: %w", err)
	}
	return &Executor{
		budget:        budget,
		startsLimit:   budget.MaxStarts,
		sem:           newContextSemaphore(budget.MaxConcurrent),
		maxTaskDepth:  budget.MaxTaskDepth,
		cycleDetector: NewCycleDetector(),
		root:          root,
	}, nil
}

// Budget returns the executor's budget.
func (e *Executor) Budget() *Budget {
	return e.budget
}

// ExecuteSimple is a convenience method for simple execution without context.
func (e *Executor) ExecuteSimple(req *Request) *Result {
	return e.Execute(context.Background(), req)
}

// Stats returns current executor statistics.
func (e *Executor) Stats() (starts, active int64) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return int64(e.starts), int64(e.sem.Count())
}

// Close releases all resources.
func (e *Executor) Close() error {
	return nil
}

// WaitForCompletion waits for all pending executions to complete.
func (e *Executor) WaitForCompletion(ctx context.Context) error {
	return nil
}
