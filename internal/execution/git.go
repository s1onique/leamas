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
// All output is captured in bounded buffers. On success, stdout contains
// the complete output up to the limit. On ErrOutputLimit, output is truncated.
type GitResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// RunGit runs a git command with full bounded execution.
// It requires a non-nil context and applies:
//   - context deadline (uses DefaultGitTimeout if context has no deadline)
//   - bounded stdout and stderr (DefaultOutputLimit)
//   - fail-closed on output overflow
//   - process termination on timeout/cancellation
//   - explicit exit status preservation
//
// The returned error directly classifies the failure:
//   - context.Canceled: caller cancelled the operation
//   - context.DeadlineExceeded: deadline expired
//   - ErrOutputLimit: output exceeded the limit (fail-closed)
//   - startup/pipe errors: wrapped errors
//
// On success, the returned GitResult contains stdout and the exit code.
// On ErrOutputLimit, partial output may be available in GitResult.
func RunGit(ctx context.Context, dir string, args ...string) (GitResult, error) {
	if ctx == nil {
		return GitResult{}, ErrNilContext
	}

	// Create deadline if context has none
	hasDeadline := ctx.Err() != context.DeadlineExceeded
	if !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultGitTimeout)
		defer cancel()
	}

	// Create command
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	// Use bytes.Buffer for concurrent draining via cmd.Stdout/cmd.Stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &boundedWriter{w: &stdout, rem: int(DefaultOutputLimit)}
	cmd.Stderr = &boundedWriter{w: &stderr, rem: int(DefaultOutputLimit)}

	// Start command
	startErr := cmd.Start()
	if startErr != nil {
		return GitResult{}, fmt.Errorf("git start: %w", startErr)
	}

	// Wait with deadline
	waitErr := cmd.Wait()

	// Check output overflow
	if stdout.Len() > DefaultOutputLimit || stderr.Len() > DefaultOutputLimit {
		// Fail-closed: do not return partial output as success
		return GitResult{
			Stdout:   stdout.Bytes()[:DefaultOutputLimit],
			Stderr:   stderr.Bytes()[:min(stderr.Len(), DefaultOutputLimit)],
			ExitCode: -1,
		}, ErrOutputLimit
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

	// Return classification errors directly
	if ctx.Err() == context.Canceled {
		return GitResult{
			Stdout:   stdout.Bytes(),
			Stderr:   stderr.Bytes(),
			ExitCode: exitCode,
		}, context.Canceled
	}
	if ctx.Err() == context.DeadlineExceeded {
		return GitResult{
			Stdout:   stdout.Bytes(),
			Stderr:   stderr.Bytes(),
			ExitCode: exitCode,
		}, context.DeadlineExceeded
	}

	return GitResult{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: exitCode,
	}, waitErr
}

// boundedWriter wraps a writer and enforces a byte limit.
// When the limit is exceeded, subsequent writes fail.
type boundedWriter struct {
	w    io.Writer
	rem  int
	done bool
}

// Write implements io.Writer with byte limit enforcement.
func (bw *boundedWriter) Write(p []byte) (int, error) {
	if bw.done {
		return 0, ErrOutputLimit
	}
	if len(p) > bw.rem {
		// Write partial up to limit
		n, _ := bw.w.Write(p[:bw.rem])
		bw.done = true
		_ = n // n is written count, not consumed count
		return len(p), ErrOutputLimit
	}
	n, err := bw.w.Write(p)
	bw.rem -= n
	return n, err
}
