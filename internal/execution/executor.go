//go:build unix || darwin || linux

// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Executor provides bounded command execution on Unix.
type Executor struct {
	budget                *Budget
	starts                uint64
	startsLimit           uint64
	sem                   *contextSemaphore
	maxTaskDepth          uint16
	cycleDetector         *CycleDetector
	root                  *ExecutionRoot
	mu                    sync.RWMutex
	generation            uint32
	retainedOutputCleanup func(int, *Request) *ExecutionError
}

// Execute runs a bounded command.
func (e *Executor) Execute(ctx context.Context, req *Request) *Result {
	start := time.Now()

	// Compute effective deadline
	effectiveDeadline := e.computeEffectiveDeadline(ctx, req)
	if !effectiveDeadline.IsZero() && effectiveDeadline.Before(start) {
		return NewErrorResult(ErrDeadlineExceeded)
	}

	// Validate request
	if err := e.validateRequest(req); err != nil {
		execErr, _ := err.(*ExecutionError)
		return NewErrorResult(execErr)
	}

	// Check task depth
	if err := e.checkTaskDepth(req); err != nil {
		execErr, _ := err.(*ExecutionError)
		return NewErrorResult(execErr)
	}

	// Check for self-execution (re-entry)
	if err := e.checkSelfExecution(req); err != nil {
		execErr, _ := err.(*ExecutionError)
		return NewErrorResult(execErr)
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

	// Acquire concurrency slot with deadline
	queueStart := time.Now()
	semCtx, cancel := context.WithDeadline(ctx, effectiveDeadline)
	defer cancel()

	acquired, err := e.sem.Acquire(semCtx, 1)
	queueDuration := time.Since(queueStart)

	if !acquired {
		if err != nil {
			if semCtx.Err() == context.DeadlineExceeded {
				return NewErrorResult(ErrDeadlineExceeded)
			}
			if semCtx.Err() == context.Canceled {
				return NewErrorResult(ErrCancelled)
			}
		}
		return NewErrorResult(ErrConcurrencyExhausted(e.budget.MaxConcurrent))
	}
	if err != nil {
		if semCtx.Err() == context.Canceled {
			return NewErrorResult(ErrCancelled)
		}
		if semCtx.Err() == context.DeadlineExceeded {
			return NewErrorResult(ErrDeadlineExceeded)
		}
		return NewErrorResult(ErrConcurrencyExhausted(e.budget.MaxConcurrent))
	}
	defer e.sem.Release(1)

	// Count start attempt (after semaphore acquisition)
	e.mu.Lock()
	if e.starts >= e.startsLimit {
		e.mu.Unlock()
		return NewErrorResult(ErrStartBudgetExhausted(e.startsLimit, e.startsLimit))
	}
	e.starts++
	e.mu.Unlock()

	// Build the command with context for WaitDelay
	execCtx, execCancel := context.WithDeadline(ctx, effectiveDeadline)
	defer execCancel()

	cmd := exec.CommandContext(execCtx, req.Args[0], req.Args[1:]...)
	// Override Cancel to use process-group termination, not direct-child-only killing
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// Negative PID = process group on Unix
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		if err == nil || isESRCH(err) {
			return nil
		}
		return err
	}
	if req.Dir != "" {
		cmd.Dir = req.Dir
	}

	// Set WaitDelay to prevent blocking on inherited descriptors
	cmd.WaitDelay = e.budget.TerminationGrace + e.budget.PostKillWait

	// Merge environment with protected values last
	cmd.Env = e.buildEnv(req)

	// Set up process group on Unix
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Set up output capture with combined limit
	outputCap := req.OutputCap
	if outputCap == 0 {
		outputCap = e.budget.MaxOutputBytes
	}

	// Create combined output buffer that shares a single budget between stdout and stderr
	outputBuf := newSharedOutputBuffer(outputCap)
	cmd.Stdout = outputBuf.StdoutWriter()
	cmd.Stderr = outputBuf.StderrWriter()

	// Start the process
	if err := cmd.Start(); err != nil {
		return &Result{
			ExitCode:            -1,
			Duration:            time.Since(start),
			QueueDuration:       queueDuration,
			OutputLimit:         outputCap,
			OutputBytesObserved: outputBuf.BytesObserved(),
			OutputBytesRetained: outputBuf.BytesRetained(),
			OutputTruncated:     outputBuf.Truncated(),
			Stdout:              outputBuf.Stdout(),
			Stderr:              outputBuf.Stderr(),
			Error: NewExecutionError(CodeExecutionCommandNotFound,
				fmt.Sprintf("failed to start %s: %v", req.Args[0], err), err),
		}
	}

	pid := cmd.Process.Pid

	// Wait for the process with cancellation
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	var exitCode int
	var exitStatusKnown bool
	var outputIncomplete bool
	var triggerErr *ExecutionError
	var cleanupErr *ExecutionError

	select {
	case <-outputBuf.OverflowCh():
		// Output overflow - terminate process tree immediately
		if err := e.terminateProcessTree(pid, req); err != nil {
			cleanupErr = err
		}
		// Drain the wait channel, capturing any Wait errors
		select {
		case wErr := <-waitCh:
			// WaitDelay means copied output may be incomplete. Cleanup was
			// already attempted above, so it is not itself a cleanup failure.
			if errors.Is(wErr, exec.ErrWaitDelay) {
				outputIncomplete = true
			}
			// Other Wait errors (context.Canceled, etc.) are expected.
		case <-time.After(e.budget.TerminationGrace):
			// Timeout draining - escalate to SIGKILL
			if cleanupErr == nil {
				cleanupErr = e.escalateTermination(pid)
			} else {
				_ = e.escalateTermination(pid) // best-effort
			}
		}
		triggerErr = ErrOutputLimitExceeded(outputCap, outputBuf.BytesObserved(),
			e.rootID(), req.CommandLine())

	case <-execCtx.Done():
		// Timeout or cancellation - terminate process tree
		if err := e.terminateProcessTree(pid, req); err != nil {
			cleanupErr = err
		}
		// Drain the wait channel with timeout
		select {
		case wErr := <-waitCh:
			// WaitDelay closes retained output pipes after process cleanup;
			// it does not prove that process-group cleanup failed.
			if errors.Is(wErr, exec.ErrWaitDelay) {
				outputIncomplete = true
			}
			// Other Wait errors (context.Canceled, etc.) are expected.
		case <-time.After(e.budget.TerminationGrace):
			// Timeout draining - escalate to SIGKILL
			if cleanupErr == nil {
				cleanupErr = e.escalateTermination(pid)
			} else {
				_ = e.escalateTermination(pid) // best-effort
			}
		}

		// Report correct error type based on context state
		if execCtx.Err() == context.DeadlineExceeded {
			triggerErr = ErrDeadlineExceeded
		} else if execCtx.Err() == context.Canceled {
			triggerErr = ErrCancelled
		}

	case err := <-waitCh:
		// Process completed naturally
		if err != nil {
			var exitErr *exec.ExitError
			switch {
			case errors.Is(err, exec.ErrWaitDelay):
				// The direct process has exited, but a descendant retained an
				// output pipe. Preserve its status and clean the saved group.
				outputIncomplete = true
				if cmd.ProcessState != nil {
					if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
						exitStatusKnown = true
						if ws.Signaled() {
							exitCode = -int(ws.Signal())
						} else {
							exitCode = ws.ExitStatus()
						}
					}
				}
				cleanupErr = e.cleanupRetainedOutput(pid, req)
				if cleanupErr == nil {
					triggerErr = ErrRetainedOutputPipe(err)
				}
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				// Context terminated while waiting
				triggerErr = NewExecutionError(
					CodeExecutionUnknown,
					fmt.Sprintf("command wait failed: %v", err),
					err,
				)
			case errors.As(err, &exitErr):
				// Normal exit error - extract exit code
				if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					if ws.Signaled() {
						exitCode = -int(ws.Signal())
					} else {
						exitCode = ws.ExitStatus()
					}
				} else {
					exitCode = -1
				}
			default:
				// Unknown error
				triggerErr = NewExecutionError(
					CodeExecutionUnknown,
					fmt.Sprintf("command wait failed: %v", err),
					err,
				)
			}
		} else {
			// Normal completion - extract exit code
			if cmd.ProcessState != nil {
				if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
					if ws.Signaled() {
						exitCode = -int(ws.Signal())
					} else {
						exitCode = ws.ExitStatus()
					}
				}
			}
		}
	}

	// Context invariant: if context terminated, ensure correct triggerErr and process tree is dead.
	// This runs even if waitCh won select (Go selects pseudo-randomly when multiple cases ready).
	// In that case we override execution_unknown with the more specific deadline/cancel error.
	if execCtx.Err() != nil {
		// Classify context termination
		if execCtx.Err() == context.DeadlineExceeded {
			triggerErr = ErrDeadlineExceeded
		} else if execCtx.Err() == context.Canceled {
			triggerErr = ErrCancelled
		}
		// Ensure process group is terminated and escalated if needed
		if err := e.terminateProcessTree(pid, req); err != nil && cleanupErr == nil {
			cleanupErr = err
		}
	}

	duration := time.Since(start)
	rootID := e.rootID()
	reportedExitCode := -1
	if exitStatusKnown {
		reportedExitCode = exitCode
	}

	// Cleanup failure is highest priority - it means a process may still be running
	if cleanupErr != nil {
		cleanupErr.RootExecutionID = rootID
		cleanupErr.Command = req.CommandLine()
		return &Result{
			ExitCode:            reportedExitCode,
			Duration:            duration,
			QueueDuration:       queueDuration,
			Stdout:              outputBuf.Stdout(),
			Stderr:              outputBuf.Stderr(),
			OutputTruncated:     outputBuf.Truncated(),
			OutputIncomplete:    outputIncomplete,
			OutputBytesObserved: outputBuf.BytesObserved(),
			OutputBytesRetained: outputBuf.BytesRetained(),
			OutputLimit:         outputCap,
			Error:               cleanupErr,
		}
	}

	// Output overflow - even if waitCh won, output exceeding cap is still an error
	if outputBuf.Truncated() {
		triggerErr = ErrOutputLimitExceeded(outputCap, outputBuf.BytesObserved(),
			rootID, req.CommandLine())
	}

	// Output overflow
	if triggerErr != nil {
		triggerErr.RootExecutionID = rootID
		triggerErr.Command = req.CommandLine()
		return &Result{
			ExitCode:            reportedExitCode,
			Duration:            duration,
			QueueDuration:       queueDuration,
			Stdout:              outputBuf.Stdout(),
			Stderr:              outputBuf.Stderr(),
			OutputTruncated:     outputBuf.Truncated(),
			OutputIncomplete:    outputIncomplete,
			OutputBytesObserved: outputBuf.BytesObserved(),
			OutputBytesRetained: outputBuf.BytesRetained(),
			OutputLimit:         outputCap,
			Error:               triggerErr,
		}
	}

	return &Result{
		ExitCode:            exitCode,
		Duration:            duration,
		QueueDuration:       queueDuration,
		Stdout:              outputBuf.Stdout(),
		Stderr:              outputBuf.Stderr(),
		OutputTruncated:     outputBuf.Truncated(),
		OutputIncomplete:    outputIncomplete,
		OutputBytesObserved: outputBuf.BytesObserved(),
		OutputBytesRetained: outputBuf.BytesRetained(),
		OutputLimit:         outputCap,
	}
}
