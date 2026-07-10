//go:build unix || darwin || linux

// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// processGroupManager manages process group termination.
type processGroupManager struct{}

// newProcessGroupManager creates a new process group manager.
func newProcessGroupManager() *processGroupManager {
	return &processGroupManager{}
}

// killProcessGroup kills the entire process group.
// On Unix, this uses syscall.Kill with negative PID.
func (m *processGroupManager) killProcessGroup(pid int, sig syscall.Signal) error {
	// On Unix, PID 0 means the current process group
	// Negative PID means the process group of that PID
	pgid := -pid
	return syscall.Kill(pgid, sig)
}

// waitForProcessGroup waits for all processes in a group to terminate.
// Returns true if all processes have terminated.
func (m *processGroupManager) waitForProcessGroup(pid int, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Try to send signal 0 to check if process group exists
		pgid := -pid
		err := syscall.Kill(pgid, syscall.Signal(0))
		if err != nil {
			// ESRCH means no such process - group is gone
			if err == syscall.ESRCH {
				return true, nil
			}
		}

		// Wait a bit before checking again
		time.Sleep(10 * time.Millisecond)
	}

	return false, fmt.Errorf("process group %d did not terminate within %v", pid, timeout)
}

// processState holds state for a running process.
type processState struct {
	cmd     *exec.Cmd
	pid     int
	done    chan struct{}
	mu      sync.RWMutex
	cleaned bool
}

// Executor provides bounded command execution.
type Executor struct {
	budget        *Budget
	starts        uint64
	startsLimit   uint64
	concurrentSem *Semaphore
	taskDepth     uint16
	cycleDetector *CycleDetector
	root          *ExecutionRoot
	mu            sync.RWMutex
}

// NewExecutor creates a new bounded executor.
func NewExecutor(budget *Budget, root *ExecutionRoot) *Executor {
	return &Executor{
		budget:        budget,
		startsLimit:   budget.MaxStarts,
		concurrentSem: NewSemaphore(budget.MaxConcurrent),
		taskDepth:     budget.MaxTaskDepth,
		cycleDetector: NewCycleDetector(),
		root:          root,
	}
}

// Budget returns the executor's budget.
func (e *Executor) Budget() *Budget {
	return e.budget
}

// Execute runs a bounded command.
func (e *Executor) Execute(ctx context.Context, req *Request) *Result {
	start := time.Now()

	// Validate timeout
	if req.Timeout > MaxPermittedTimeout {
		return NewErrorResult(ErrUnboundedTimeout(req.Timeout.String()))
	}
	if req.Timeout == 0 {
		req.Timeout = 120 * time.Second // Default
	}

	// Validate request
	if len(req.Args) == 0 {
		return NewErrorResult(&ExecutionError{
			Code:    CodeExecutionUnknown,
			Message: "no command specified",
		})
	}

	// Check for self-execution
	if e.root != nil && e.root.IsSelfExecutable(req.Args[0]) {
		return NewErrorResult(&ExecutionError{
			Code:    CodeNestedLeamasExecution,
			Message: "refusing to execute Leamas from within Leamas",
		})
	}

	// Check fingerprint for cycles
	fingerprint := req.Fingerprint
	if fingerprint == "" {
		fingerprint = ComputeFingerprint(req.Args[0], req.Args[1:], req.Dir, req.Name)
	}
	if err := e.cycleDetector.CheckAndTrack(fingerprint, req.Name); err != nil {
		return NewErrorResult(err.(*ExecutionError))
	}
	defer e.cycleDetector.Untrack(fingerprint)

	// Check start budget - cumulative limit on total starts
	e.mu.Lock()
	if e.starts >= e.startsLimit {
		e.mu.Unlock()
		return NewErrorResult(ErrStartBudgetExhausted(e.startsLimit, e.startsLimit+1))
	}
	e.starts++ // Count this start (never decremented for cumulative budget)
	e.mu.Unlock()

	// Acquire concurrency slot
	select {
	case <-ctx.Done():
		return NewErrorResult(&ExecutionError{
			Code:    CodeExecutionCancelled,
			Message: ctx.Err().Error(),
		})
	default:
	}

	acquired, err := e.concurrentSem.Acquire(ctx, 1)
	if !acquired || err != nil {
		return NewErrorResult(ErrConcurrencyExhausted(e.budget.MaxConcurrent))
	}
	defer e.concurrentSem.Release(1, 1)

	// Build the command
	cmd := exec.Command(req.Args[0], req.Args[1:]...)
	if req.Dir != "" {
		cmd.Dir = req.Dir
	}

	// Merge environment
	if len(req.Env) > 0 {
		cmd.Env = append(os.Environ(), req.Env...)
	} else {
		cmd.Env = os.Environ()
	}

	// Set up process group on Unix
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true

	// Set up output capture
	outputCap := req.OutputCap
	if outputCap == 0 {
		outputCap = e.budget.MaxOutputBytes
	}

	stdout := NewCappedBuffer(outputCap)
	stderr := NewCappedBuffer(outputCap)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Set up context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	// Start the process
	if err := cmd.Start(); err != nil {
		return NewErrorResult(&ExecutionError{
			Code:    CodeExecutionCommandNotFound,
			Message: fmt.Sprintf("failed to start %s: %v", req.Args[0], err),
			Cause:   err,
		})
	}

	pid := cmd.Process.Pid

	// Wait for the process with cancellation
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	var exitCode int
	select {
	case <-cmdCtx.Done():
		// Timeout or cancellation
		pgm := newProcessGroupManager()

		// Send SIGTERM to process group
		pgm.killProcessGroup(pid, syscall.SIGTERM)

		// Wait with grace period
		gracePeriod := e.budget.TerminationGrace
		if gracePeriod == 0 {
			gracePeriod = 2 * time.Second
		}

		terminated, _ := pgm.waitForProcessGroup(pid, gracePeriod)
		if !terminated {
			// Send SIGKILL
			pgm.killProcessGroup(pid, syscall.SIGKILL)
			pgm.waitForProcessGroup(pid, 1*time.Second)
		}

		<-done // Wait for Wait() to complete

		if cmdCtx.Err() == context.DeadlineExceeded {
			return NewErrorResult(ErrTimeoutExceeded(req.Timeout.String()))
		}
		return NewErrorResult(&ExecutionError{
			Code:    CodeExecutionCancelled,
			Message: cmdCtx.Err().Error(),
		})
	case <-done:
		// Process completed
		if cmd.ProcessState != nil {
			if cmd.ProcessState.Exited() {
				exitCode = cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
			}
		}
	}

	duration := time.Since(start)
	return &Result{
		ExitCode:        exitCode,
		Duration:        duration,
		Stdout:          stdout.Bytes(),
		Stderr:          stderr.Bytes(),
		OutputTruncated: stdout.Truncated() || stderr.Truncated(),
	}
}

// ExecuteSimple is a convenience method for simple execution without context.
func (e *Executor) ExecuteSimple(req *Request) *Result {
	return e.Execute(context.Background(), req)
}

// Stats returns current executor statistics.
func (e *Executor) Stats() (starts, active int64) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return int64(e.starts), int64(e.concurrentSem.Count())
}

// Semaphore provides bounded concurrent access.
type Semaphore struct {
	cnt   int64
	limit int64
	mu    sync.Cond
}

// NewSemaphore creates a new semaphore with the specified limit.
func NewSemaphore(limit int64) *Semaphore {
	s := &Semaphore{limit: limit}
	s.mu.L = &sync.Mutex{}
	return s
}

// Acquire acquires n permits, blocking until available or context is done.
func (s *Semaphore) Acquire(ctx context.Context, n int64) (bool, error) {
	for {
		// First check without blocking
		s.mu.L.Lock()
		if s.cnt+n <= s.limit {
			s.cnt += n
			s.mu.L.Unlock()
			return true, nil
		}

		// Wait for permits to be released while holding the lock
		for s.cnt+n > s.limit {
			if ctx.Err() != nil {
				s.mu.L.Unlock()
				return false, ctx.Err()
			}
			// Wait releases the lock and blocks until Broadcast is called
			s.mu.Wait()
			// When we return, we hold the lock again
		}

		s.mu.L.Unlock()

		// Check context again after acquiring lock
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			// Loop and try again
		}
	}
}

// TryAcquire attempts to acquire n permits without blocking.
func (s *Semaphore) TryAcquire(n int64) bool {
	s.mu.L.Lock()
	defer s.mu.L.Unlock()
	if s.cnt+n <= s.limit {
		s.cnt += n
		return true
	}
	return false
}

// Release releases n permits.
func (s *Semaphore) Release(n int64, wakeCount int) {
	s.mu.L.Lock()
	s.cnt -= n
	if s.cnt < 0 {
		s.cnt = 0
	}
	s.mu.Broadcast()
	s.mu.L.Unlock()
}

// Count returns the current number of acquired permits.
func (s *Semaphore) Count() int64 {
	s.mu.L.Lock()
	defer s.mu.L.Unlock()
	return s.cnt
}

// Limit returns the semaphore limit.
func (s *Semaphore) Limit() int64 {
	return s.limit
}
