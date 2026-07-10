//go:build unix || darwin || linux

package execution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
)

// computeEffectiveDeadline calculates the earliest deadline.
func (e *Executor) computeEffectiveDeadline(ctx context.Context, req *Request) time.Time {
	now := time.Now()
	var deadline time.Time

	// Start with request timeout
	if req.Timeout > 0 {
		deadline = now.Add(req.Timeout)
	} else {
		deadline = now.Add(DefaultTimeout)
	}

	// Check Budget.Deadline - use budget deadline if it's sooner (more restrictive)
	if !e.budget.Deadline.IsZero() && e.budget.Deadline.Before(deadline) {
		deadline = e.budget.Deadline
	}

	// Check parent context deadline
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	return deadline
}

// validateRequest checks request validity.
func (e *Executor) validateRequest(req *Request) error {
	if len(req.Args) == 0 {
		return ErrInvalidRequest("no command specified")
	}

	// Check timeout bounds
	if req.Timeout < 0 {
		return ErrInvalidRequest("negative timeout is invalid")
	}
	if req.Timeout > MaxPermittedTimeout {
		return ErrUnboundedTimeout(req.Timeout.String())
	}

	// Check output cap bounds
	if req.OutputCap < 0 {
		return ErrInvalidRequest("negative output cap is invalid")
	}
	if req.OutputCap > MaxPermittedMaxOutputBytes {
		return ErrInvalidRequest("output cap exceeds hard maximum")
	}

	return nil
}

// checkTaskDepth verifies task depth is within limits.
func (e *Executor) checkTaskDepth(req *Request) error {
	depth := req.TaskDepth
	if depth == 0 {
		depth = 1
	}
	if depth > e.maxTaskDepth {
		return ErrTaskDepthExceeded(e.maxTaskDepth, depth)
	}
	return nil
}

// checkSelfExecution checks for nested Leamas execution.
func (e *Executor) checkSelfExecution(req *Request) error {
	if e.root != nil && e.root.IsSelfExecutable(req.Args[0]) {
		return ErrNestedLeamasExecution
	}
	return nil
}

// buildEnv constructs the environment for the command.
// Request environment is merged first, then protected values are set last
// to prevent caller override of re-entry metadata.
func (e *Executor) buildEnv(req *Request) []string {
	// Start with base environment
	env := os.Environ()

	// Merge request environment first (will be overridden by protected values)
	if len(req.Env) > 0 {
		env = mergeEnvironment(env, req.Env)
	}

	// Add/update protected re-entry metadata LAST to prevent override
	if e.root != nil {
		env = updateEnv(env, EnvRootID, e.root.ID)
		env = updateEnv(env, EnvParentPID, fmt.Sprintf("%d", os.Getpid()))
		env = updateEnv(env, EnvGeneration, fmt.Sprintf("%d", e.root.Generation+1))
	}

	return env
}

// mergeEnvironment merges two environment slices, with later values taking precedence.
// Duplicate keys are resolved by keeping the value from the second slice.
func mergeEnvironment(base, overlay []string) []string {
	// Build a map from base
	result := make(map[string]string)
	for _, e := range base {
		if idx := strings.Index(e, "="); idx >= 0 {
			result[e[:idx]] = e
		}
	}

	// Apply overlay
	for _, e := range overlay {
		if idx := strings.Index(e, "="); idx >= 0 {
			result[e[:idx]] = e
		}
	}

	// Convert back to slice
	env := make([]string, 0, len(result))
	for _, v := range result {
		env = append(env, v)
	}
	return env
}

// terminateProcessTree gracefully terminates the process group.
func (e *Executor) terminateProcessTree(pid int, req *Request) *ExecutionError {
	pgm := newProcessGroupManager()

	// Send SIGTERM to process group
	if err := pgm.killProcessGroup(pid, syscall.SIGTERM); err != nil {
		// Any error from Kill means we can't signal the process group.
		// This is benign - either process is gone (ESRCH), zombie (EPERM),
		// or orphaned to init. Either way, no cleanup needed.
		// Just return nil; the waitForProcessGroup will confirm termination.
	}

	// Wait with grace period
	terminated, err := pgm.waitForProcessGroup(pid, e.budget.TerminationGrace)
	// Any error means we can't determine if process is gone.
	// Treat as not terminated and escalate to SIGKILL.
	if err != nil || !terminated {
		// Process still alive after grace period - escalate to SIGKILL.
		// escalateTermination returns nil even if process doesn't die within PostKillWait
		// (that's expected for stubborn processes). Only unexpected errors are returned.
		return e.escalateTermination(pid)
	}

	return nil
}

// escalateTermination escalates termination to SIGKILL.
func (e *Executor) escalateTermination(pid int) *ExecutionError {
	pgm := newProcessGroupManager()

	// Send SIGKILL to process group (not just leader)
	if err := pgm.killProcessGroup(pid, syscall.SIGKILL); err != nil {
		// Only ESRCH means the process group is confirmed absent
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		// EPERM means we lack permission to signal the group - it may still exist
		// EINVAL means invalid signal - unexpected for SIGKILL
		// Any other error means we can't determine if group is gone
		return ErrProcessTreeCleanupFailed(pid, fmt.Sprintf("SIGKILL: %v", err))
	}

	// Wait with post-kill period and verify process group is gone
	terminated, err := pgm.waitForProcessGroup(pid, e.budget.PostKillWait)
	if err != nil {
		// Any error during wait means we couldn't verify absence
		return ErrProcessTreeCleanupFailed(pid, fmt.Sprintf("post-kill wait: %v", err))
	}
	if !terminated {
		// Process group survived SIGKILL and post-kill wait - cleanup failed
		return ErrProcessTreeCleanupFailed(pid, "process group survived SIGKILL")
	}

	return nil
}

// rootID returns the root execution ID.
func (e *Executor) rootID() string {
	if e.root != nil {
		return e.root.ID
	}
	return ""
}
