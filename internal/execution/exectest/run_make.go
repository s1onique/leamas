package exectest

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

// DefaultWaitDelay is the default WaitDelay for retained descriptor cleanup.
const DefaultWaitDelay = 2 * time.Second

// DefaultTimeout is the default runtime limit (5 minutes).
const DefaultTimeout = 5 * time.Minute

// Result captures command execution outcome with separate stdout/stderr.
type Result struct {
	Outcome   Outcome
	Stdout    []byte
	Stderr    []byte
	ExitCode  int
	Overflow  *OutputLimitExceeded
	SpawnErr  *SpawnError
	WaitDelay bool
}

// Run executes a command with bounded resources and separate stdout/stderr.
// The name parameter is the executable to run (e.g., "make", "/nonexistent").
func Run(ctx context.Context, dir string, env []string, name string, args ...string) *Result {
	if ctx == nil {
		ctx = context.Background()
	}

	// Apply default timeout if context has no deadline
	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}

	outputLimit := DefaultOutputLimit

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = mergeEnv(env)
	cmd.WaitDelay = DefaultWaitDelay

	rs := newRunState(outputLimit, outputLimit)

	// Use Cancel callback to record cancellation cause
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		err := cmd.Process.Kill()
		// Only mark cancellation when kill was actually applied (nil means killed)
		// os.ErrProcessDone means process already completed naturally
		if err == nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				rs.markTimeout()
			} else {
				rs.markCancelled()
			}
			return nil
		}
		return err
	}

	cmd.Stdout = &boundedWriter{rs: rs, isStderr: false}
	cmd.Stderr = &boundedWriter{rs: rs, isStderr: true}

	if err := cmd.Start(); err != nil {
		return &Result{
			Outcome:  OutcomeSpawnFailure,
			Stderr:   []byte("spawn failed: " + err.Error()),
			SpawnErr: &SpawnError{Cause: err},
		}
	}

	waitErr := cmd.Wait()

	// Check for context termination first (from Cancel callback)
	if rs.isTimeout() {
		return &Result{
			Outcome: OutcomeTimeout,
			Stdout:  rs.stdoutBytes(),
			Stderr:  rs.stderrBytes(),
		}
	}
	if rs.isCancelled() {
		return &Result{
			Outcome: OutcomeCancelled,
			Stdout:  rs.stdoutBytes(),
			Stderr:  rs.stderrBytes(),
		}
	}

	// Check for output overflow
	if rs.hadOverflow() {
		return &Result{
			Outcome: OutcomeOutputOverflow,
			Stdout:  rs.stdoutBytes(),
			Stderr:  rs.stderrBytes(),
			Overflow: &OutputLimitExceeded{
				Limit:    outputLimit,
				Observed: rs.observedBytes(),
			},
		}
	}

	// Classify wait result
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return &Result{
				Outcome:  OutcomeExitFailure,
				Stdout:   rs.stdoutBytes(),
				Stderr:   rs.stderrBytes(),
				ExitCode: exitErr.ExitCode(),
			}
		}
		if errors.Is(waitErr, exec.ErrWaitDelay) {
			return &Result{
				Outcome:   OutcomeWaitDelay,
				Stdout:    rs.stdoutBytes(),
				Stderr:    rs.stderrBytes(),
				WaitDelay: true,
			}
		}
		return &Result{
			Outcome: OutcomeExecutionError,
			Stdout:  rs.stdoutBytes(),
			Stderr:  rs.stderrBytes(),
		}
	}

	return &Result{
		Outcome:  OutcomeSuccess,
		Stdout:   rs.stdoutBytes(),
		Stderr:   rs.stderrBytes(),
		ExitCode: 0,
	}
}

// RunMake executes a make command with bounded resources and separate stdout/stderr.
func RunMake(ctx context.Context, dir string, env []string, target string, args ...string) *Result {
	makeArgs := append([]string{}, args...)
	makeArgs = append(makeArgs, target)
	return Run(ctx, dir, env, "make", makeArgs...)
}
