// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Sentinel errors for git operations.
var (
	// ErrNilContext is returned when a nil context is passed to RunGit.
	ErrNilContext = errors.New("nil context not permitted: use context.Background() or provide a cancellable context")

	// ErrOutputLimit is returned when process output exceeds the configured limit.
	// This is a fail-closed error - callers cannot treat it as success.
	ErrOutputLimit = errors.New("process output limit exceeded")
)

// Default bounds for git operations.
const (
	// DefaultGitTimeout is the default deadline for git operations.
	DefaultGitTimeout = 30 * time.Second

	// DefaultOutputLimit is the maximum output allowed from git commands.
	DefaultOutputLimit = 8 * 1024 * 1024 // 8 MiB

	// DefaultGitWaitDelay bounds cleanup latency after process termination.
	DefaultGitWaitDelay = 2 * time.Second
)

// GitResult represents the result of a bounded git command.
type GitResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// RunGit runs a git command with full bounded execution.
// It requires a non-nil context and applies:
//   - context deadline (uses DefaultGitTimeout if context has no deadline)
//   - bounded stdout and stderr (DefaultOutputLimit)
//   - fail-closed on output overflow (process is terminated)
//   - WaitDelay cleanup bound (DefaultGitWaitDelay)
//   - process termination on timeout/cancellation
//   - explicit exit status preservation
func RunGit(ctx context.Context, dir string, args ...string) (GitResult, error) {
	return runCommandWithLimits(ctx, "git", dir, DefaultGitTimeout, int(DefaultOutputLimit), args...)
}

// runCommandWithLimits is the internal test seam. Production callers use RunGit.
// Tests can substitute the executable, timeout, and output limit.
func runCommandWithLimits(
	ctx context.Context,
	executable string,
	dir string,
	timeout time.Duration,
	outputLimit int,
	args ...string,
) (GitResult, error) {
	if ctx == nil {
		return GitResult{}, ErrNilContext
	}

	// Check if context has a deadline; if not, apply default timeout
	_, hasDeadline := ctx.Deadline()

	var runCtx context.Context
	var cancelRun context.CancelFunc

	if hasDeadline {
		runCtx, cancelRun = context.WithCancel(ctx)
	} else {
		runCtx, cancelRun = context.WithTimeout(ctx, timeout)
	}
	defer cancelRun()

	// Create command
	cmd := exec.CommandContext(runCtx, executable, args...)
	cmd.Dir = dir
	cmd.WaitDelay = DefaultGitWaitDelay

	// Use bytes.Buffer for concurrent draining via cmd.Stdout/cmd.Stderr
	var stdout, stderr bytes.Buffer

	// Track overflow state atomically (written from copy goroutines)
	var overflowOccurred atomicBool

	// makeOverflowHandler creates an overflow callback that fires exactly once
	// and triggers process cancellation.
	makeOverflowHandler := func() func() {
		var once sync.Once
		return func() {
			once.Do(func() {
				overflowOccurred.set(true)
				cancelRun()
			})
		}
	}

	stdoutBW := &boundedWriter{
		w:          &stdout,
		rem:        outputLimit,
		onOverflow: makeOverflowHandler(),
	}
	stderrBW := &boundedWriter{
		w:          &stderr,
		rem:        outputLimit,
		onOverflow: makeOverflowHandler(),
	}

	cmd.Stdout = stdoutBW
	cmd.Stderr = stderrBW

	// Start command
	startErr := cmd.Start()
	if startErr != nil {
		return GitResult{}, fmt.Errorf("start: %w", startErr)
	}

	// Wait for command
	waitErr := cmd.Wait()

	// Check for WaitDelay timeout
	if waitErr != nil && errors.Is(waitErr, exec.ErrWaitDelay) {
		return GitResult{
			Stdout:   stdout.Bytes(),
			Stderr:   stderr.Bytes(),
			ExitCode: -1,
		}, fmt.Errorf("cleanup timeout: %w", waitErr)
	}

	// Extract exit code
	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(interface{ ExitStatus() int }); ok {
				exitCode = ws.ExitStatus()
			} else {
				exitCode = -1
			}
		} else {
			exitCode = -1
		}
	}

	// Check for overflow first (highest priority fail-closed)
	if overflowOccurred.get() {
		// Return truncated output - callers must treat as failure
		soLen := stdout.Len()
		seLen := stderr.Len()
		sOut := stdout.Bytes()
		sErr := stderr.Bytes()
		if soLen > outputLimit {
			soLen = outputLimit
		}
		if seLen > outputLimit {
			seLen = outputLimit
		}
		return GitResult{
			Stdout:   sOut[:soLen],
			Stderr:   sErr[:seLen],
			ExitCode: -1,
		}, ErrOutputLimit
	}

	// Check context state for error classification
	select {
	case <-runCtx.Done():
		if runCtx.Err() == context.DeadlineExceeded {
			return GitResult{
				Stdout:   stdout.Bytes(),
				Stderr:   stderr.Bytes(),
				ExitCode: exitCode,
			}, context.DeadlineExceeded
		}
		// Cancelled by caller
		return GitResult{
			Stdout:   stdout.Bytes(),
			Stderr:   stderr.Bytes(),
			ExitCode: exitCode,
		}, context.Canceled
	default:
		// Context not cancelled
		return GitResult{
			Stdout:   stdout.Bytes(),
			Stderr:   stderr.Bytes(),
			ExitCode: exitCode,
		}, waitErr
	}
}

// atomicBool provides atomic boolean operations without sync/atomic dep.
type atomicBool struct {
	mu sync.Mutex
	v  bool
}

func (a *atomicBool) set(v bool) {
	a.mu.Lock()
	a.v = v
	a.mu.Unlock()
}

func (a *atomicBool) get() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.v
}

// boundedWriter wraps a writer and enforces a byte limit.
// When the limit is exceeded, it triggers overflow callback exactly once.
type boundedWriter struct {
	w          io.Writer
	rem        int
	overflow   bool
	onOverflow func()
}

// Write implements io.Writer with byte limit enforcement.
// The first byte beyond the limit triggers onOverflow exactly once.
func (bw *boundedWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if bw.overflow {
		// Already in overflow state; no more writes accepted
		return 0, ErrOutputLimit
	}
	if bw.rem == 0 {
		// Limit was exactly reached previously; any further write overflows
		bw.overflow = true
		if bw.onOverflow != nil {
			bw.onOverflow()
		}
		return 0, ErrOutputLimit
	}
	if len(p) > bw.rem {
		// Partial write up to limit
		n, err := bw.w.Write(p[:bw.rem])
		bw.rem -= n
		bw.overflow = true
		if bw.onOverflow != nil {
			bw.onOverflow()
		}
		if err != nil {
			return n, err
		}
		return n, ErrOutputLimit
	}
	n, err := bw.w.Write(p)
	bw.rem -= n
	return n, err
}
