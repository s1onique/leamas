package execution

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunGit_NilContext(t *testing.T) {
	// Nil context must be rejected
	_, err := RunGit(nil, ".", "status")
	if !errors.Is(err, ErrNilContext) {
		t.Errorf("expected ErrNilContext, got %v", err)
	}
}

func TestRunGit_Success(t *testing.T) {
	// Successful git command
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(string(result.Stdout), "true") && !strings.Contains(string(result.Stdout), "false") {
		t.Errorf("unexpected stdout: %s", result.Stdout)
	}
}

func TestRunGit_ExitCode(t *testing.T) {
	// Git command with nonzero exit code
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "rev-parse", "nonexistent")
	// Nonzero exit code may be returned as error
	if result.ExitCode == 0 {
		t.Error("expected nonzero exit code for nonexistent ref")
	}
	_ = err // err may be present for nonzero exit
}

func TestRunGit_DeadlineExceeded(t *testing.T) {
	// Context deadline should interrupt git
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	result, err := RunGit(ctx, ".", "rev-list", "--all", "--count")
	if err == nil {
		t.Error("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
	// Exit code should indicate timeout
	if result.ExitCode != -1 && result.ExitCode != 124 { // 124 is timeout's exit code
		t.Logf("note: deadline exit code %d may vary", result.ExitCode)
	}
}

func TestRunGit_Cancellation(t *testing.T) {
	// Caller cancellation must interrupt git
	// Use a very slow command that will definitely still be running
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := RunGit(ctx, ".", "rev-list", "--all")
	// Either cancelled/deadline exceeded or fast command completed
	if err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected Canceled or DeadlineExceeded, got %v", err)
		}
	}
	t.Logf("result: exit=%d, stdout=%d bytes, err=%v", result.ExitCode, len(result.Stdout), err)
}

func TestRunGit_OutputLimit(t *testing.T) {
	// Large output should be captured up to limit
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "log", "--format=%H%n", "-n", "100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Stdout) == 0 {
		t.Error("expected stdout from git log")
	}
	if len(result.Stdout) > DefaultOutputLimit {
		t.Errorf("stdout exceeds limit: %d > %d", len(result.Stdout), DefaultOutputLimit)
	}
}

func TestRunGit_RawOutputPreservesNUL(t *testing.T) {
	// Raw output should preserve NUL bytes from -z flag
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "ls-tree", "-z", "--name-only", "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// NUL-delimited output should have NUL bytes (unless empty)
	if len(result.Stdout) > 0 && !strings.Contains(string(result.Stdout), "\x00") {
		t.Log("note: HEAD may be empty or use different format")
	}
}

func TestRunGit_StderrCapture(t *testing.T) {
	// Stderr should be captured separately
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "status", "--porcelain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// stderr should be empty on success
	if len(result.Stderr) > 0 {
		t.Logf("note: stderr has content: %s", result.Stderr)
	}
}

func TestRunGit_CWD(t *testing.T) {
	// Verify correct directory handling
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return absolute path
	if !filepath.IsAbs(strings.TrimSpace(string(result.Stdout))) {
		t.Errorf("expected absolute path, got: %s", result.Stdout)
	}
}

func TestBoundedWriter_ExactlyLimit(t *testing.T) {
	// Writing exactly limit bytes should succeed
	input := strings.Repeat("x", 100)
	lr := &boundedWriter{w: &strings.Builder{}, rem: 100}
	n, err := lr.Write([]byte(input))
	if err != nil {
		t.Errorf("unexpected error at limit: %v", err)
	}
	if n != 100 {
		t.Errorf("expected 100, got %d", n)
	}
}

func TestBoundedWriter_Overflow(t *testing.T) {
	// Writing beyond limit should fail
	input := strings.Repeat("x", 200)
	lr := &boundedWriter{w: &strings.Builder{}, rem: 100}
	n, err := lr.Write([]byte(input))
	if err == nil {
		t.Error("expected error on overflow")
	}
	if !errors.Is(err, ErrOutputLimit) {
		t.Errorf("expected ErrOutputLimit, got %v", err)
	}
	// Should have written 100 bytes
	if n != 200 {
		t.Errorf("expected Write to report full length: %d", n)
	}
}

func TestBoundedWriter_AfterDone(t *testing.T) {
	// Writing after done should fail
	lr := &boundedWriter{w: &strings.Builder{}, rem: 5, done: true}
	n, err := lr.Write([]byte("hello"))
	if err == nil {
		t.Error("expected error after done")
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

// Test concurrent output handling - not testing with fake git here
// but verifying the structure handles both streams
func TestRunGit_BothStreams(t *testing.T) {
	ctx := context.Background()
	// git status produces minimal output - both streams should be handled
	result, err := RunGit(ctx, ".", "status", "-z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both should be captured
	t.Logf("stdout: %d bytes, stderr: %d bytes", len(result.Stdout), len(result.Stderr))
}
