// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
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
//   - process termination on timeout/cancellation
//   - explicit exit status preservation
//
// The returned error directly classifies the failure:
//   - context.Canceled: caller cancelled the operation
//   - context.DeadlineExceeded: deadline expired
//   - ErrOutputLimit: output exceeded the limit (fail-closed)
//   - wrapped errors for startup/pipe failures
func RunGit(ctx context.Context, dir string, args ...string) (GitResult, error) {
	if ctx == nil {
		return GitResult{}, ErrNilContext
	}

	// Check if context has a deadline; if not, apply default
	_, hasDeadline := ctx.Deadline()

	// Create run context - will be replaced with timeout version if needed
	var runCtx context.Context
	var cancelRun context.CancelFunc

	if hasDeadline {
		runCtx, cancelRun = context.WithCancel(ctx)
	} else {
		runCtx, cancelRun = context.WithTimeout(ctx, DefaultGitTimeout)
	}
	defer cancelRun()

	// Create command
	cmd := exec.CommandContext(runCtx, "git", args...)
	cmd.Dir = dir

	// Use bytes.Buffer for concurrent draining via cmd.Stdout/cmd.Stderr
	var stdout, stderr bytes.Buffer

	// Track overflow state
	overflowOccurred := false

	// Create bounded writers with overflow cancellation
	stdoutBW := &boundedWriter{
		w:   &stdout,
		rem: int(DefaultOutputLimit),
		onOverflow: func() {
			overflowOccurred = true
			cancelRun()
		},
	}
	stderrBW := &boundedWriter{
		w:   &stderr,
		rem: int(DefaultOutputLimit),
		onOverflow: func() {
			overflowOccurred = true
			cancelRun()
		},
	}

	cmd.Stdout = stdoutBW
	cmd.Stderr = stderrBW

	// Start command
	startErr := cmd.Start()
	if startErr != nil {
		return GitResult{}, fmt.Errorf("git start: %w", startErr)
	}

	// Wait for command
	waitErr := cmd.Wait()

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
	if overflowOccurred {
		return GitResult{
			Stdout:   stdout.Bytes()[:min(stdout.Len(), int(DefaultOutputLimit))],
			Stderr:   stderr.Bytes()[:min(stderr.Len(), int(DefaultOutputLimit))],
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

// boundedWriter wraps a writer and enforces a byte limit.
// When the limit is exceeded, it triggers overflow callback and returns error.
type boundedWriter struct {
	w          io.Writer
	rem        int
	done       bool
	exceeded   bool
	onOverflow func()
}

// Write implements io.Writer with byte limit enforcement.
func (bw *boundedWriter) Write(p []byte) (int, error) {
	if bw.done {
		bw.exceeded = true
		return 0, ErrOutputLimit
	}
	if len(p) > bw.rem {
		// Write partial up to limit
		n, _ := bw.w.Write(p[:bw.rem])
		bw.done = true
		bw.exceeded = true
		// Trigger overflow cancellation
		if bw.onOverflow != nil {
			bw.onOverflow()
		}
		// Return actual bytes written
		return n, ErrOutputLimit
	}
	n, err := bw.w.Write(p)
	bw.rem -= n
	if bw.rem == 0 {
		bw.done = true
	}
	return n, err
}
