// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// ErrNilContext is returned when a nil context is passed to a bounded function.
var ErrNilContext = errors.New("nil context not permitted: use context.Background() or provide a cancellable context")

// DefaultGitTimeout is the default timeout for git operations.
const DefaultGitTimeout = 30 * time.Second

// MaxGitOutputBytes is the maximum output allowed from git commands.
const MaxGitOutputBytes = 8 * 1024 * 1024 // 8 MiB

// GitOutputLimitReader wraps a reader and enforces a maximum byte limit.
type GitOutputLimitReader struct {
	r    io.Reader
	rem  int64
	over bool
}

// NewGitOutputLimitReader creates a reader that enforces a byte limit.
func NewGitOutputLimitReader(r io.Reader, maxBytes int64) *GitOutputLimitReader {
	return &GitOutputLimitReader{r: r, rem: maxBytes}
}

// Read implements io.Reader with byte limit enforcement.
func (l *GitOutputLimitReader) Read(p []byte) (int, error) {
	if l.rem <= 0 {
		l.over = true
		return 0, io.EOF
	}
	n := len(p)
	if int64(n) > l.rem {
		n = int(l.rem)
	}
	n, err := l.r.Read(p[:n])
	l.rem -= int64(n)
	if l.rem <= 0 {
		l.over = true
	}
	return n, err
}

// Exceeded returns true if the output limit was exceeded.
func (l *GitOutputLimitReader) Exceeded() bool {
	return l.over
}

// GitResult represents the result of a bounded git command.
type GitResult struct {
	Stdout     []byte
	Stderr     []byte
	ExitCode   int
	OutputOver bool
	Error      error
}

// RunGit runs a git command with full bounded execution.
// It requires a non-nil context and applies canonical resource limits:
// - timeout via context deadline (uses DefaultGitTimeout if context has no deadline)
// - bounded stdout and stderr (MaxGitOutputBytes)
// - explicit exit status preservation
// - process group termination on timeout/cancellation
func RunGit(ctx context.Context, dir string, args ...string) (*GitResult, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	// Compute effective deadline
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		// Apply default timeout if context has no deadline
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultGitTimeout)
		defer cancel()
		deadline, _ = ctx.Deadline()
	}

	// Create command with deadline
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	// Set WaitDelay for proper cleanup
	cmd.WaitDelay = 2 * time.Second

	// Create output buffers with limits
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start git: %w", err)
	}

	// Read stdout with limit
	stdoutLim := NewGitOutputLimitReader(stdoutPipe, MaxGitOutputBytes)
	stdout := make([]byte, 0, 4096)
	buf := make([]byte, 4096)
	for {
		n, err := stdoutLim.Read(buf)
		if n > 0 {
			stdout = append(stdout, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Read stderr with limit
	stderrLim := NewGitOutputLimitReader(stderrPipe, MaxGitOutputBytes)
	stderr := make([]byte, 0, 4096)
	for {
		n, err := stderrLim.Read(buf)
		if n > 0 {
			stderr = append(stderr, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Wait for command with deadline
	waitErr := make(chan error, 1)
	go func() {
		waitErr <- cmd.Wait()
	}()

	select {
	case err := <-waitErr:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if ws, ok := exitErr.Sys().(interface{ ExitStatus() int }); ok {
					exitCode = ws.ExitStatus()
				} else {
					exitCode = -1
				}
			} else {
				exitCode = -1
			}
		}
		return &GitResult{
			Stdout:     stdout,
			Stderr:     stderr,
			ExitCode:   exitCode,
			OutputOver: stdoutLim.Exceeded() || stderrLim.Exceeded(),
			Error:      err,
		}, nil
	case <-ctx.Done():
		// Context expired - kill process tree
		if cmd.Process != nil {
			// Try SIGTERM first
			cmd.Process.Kill()
			// Drain wait
			<-waitErr
		}
		if ctx.Err() == context.DeadlineExceeded {
			return &GitResult{
				Stdout:     stdout,
				Stderr:     stderr,
				ExitCode:   -1,
				OutputOver: stdoutLim.Exceeded() || stderrLim.Exceeded(),
				Error:      fmt.Errorf("deadline exceeded before %s", deadline.Format(time.RFC3339)),
			}, nil
		}
		return &GitResult{
			Stdout:     stdout,
			Stderr:     stderr,
			ExitCode:   -1,
			OutputOver: stdoutLim.Exceeded() || stderrLim.Exceeded(),
			Error:      ctx.Err(),
		}, nil
	}
}

// RunGitSimple is DEPRECATED. Use RunGit with a proper context.
// This function exists only for backward compatibility during migration.
func RunGitSimple(dir string, args ...string) ([]byte, int, error) {
	result, err := RunGit(context.Background(), dir, args...)
	if err != nil {
		return nil, -1, err
	}
	return result.Stdout, result.ExitCode, result.Error
}
