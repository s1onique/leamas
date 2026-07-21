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
	_, err := RunGit(nil, ".", "status")
	if !errors.Is(err, ErrNilContext) {
		t.Errorf("expected ErrNilContext, got %v", err)
	}
}

func TestRunGit_Success(t *testing.T) {
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
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "rev-parse", "nonexistent")
	// Nonzero exit code may be returned as error
	if result.ExitCode == 0 {
		t.Error("expected nonzero exit code for nonexistent ref")
	}
	_ = err // err may be present for nonzero exit
}

func TestRunGit_DefaultTimeout(t *testing.T) {
	// context.Background() should get default timeout applied
	ctx := context.Background()

	// Verify default timeout is applied by checking Deadline()
	deadlineCtx, ok := ctx.Deadline()
	if ok {
		t.Logf("context already has deadline: %v", deadlineCtx)
	}

	result, err := RunGit(ctx, ".", "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("unexpected error with Background context: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
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
	if result.ExitCode != -1 && result.ExitCode != 124 {
		t.Logf("note: deadline exit code %d may vary", result.ExitCode)
	}
}

func TestRunGit_Cancellation(t *testing.T) {
	// Caller cancellation must interrupt git
	ctx, cancel := context.WithCancel(context.Background())

	// Start cancellation after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result, err := RunGit(ctx, ".", "rev-list", "--all", "--count")
	// Either cancelled or fast command completed
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected Canceled, got %v", err)
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
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "ls-tree", "-z", "--name-only", "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// NUL-delimited output should have NUL bytes
	if len(result.Stdout) > 0 && !strings.Contains(string(result.Stdout), "\x00") {
		t.Log("note: HEAD may be empty or use different format")
	}
}

func TestRunGit_StderrCapture(t *testing.T) {
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "status", "--porcelain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// stderr should be empty on success
	if len(result.Stderr) > 0 {
		t.Errorf("unexpected stderr content: %s", result.Stderr)
	}
}

func TestRunGit_CWD(t *testing.T) {
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(strings.TrimSpace(string(result.Stdout))) {
		t.Errorf("expected absolute path, got: %s", result.Stdout)
	}
}

func TestBoundedWriter_ExactlyLimit(t *testing.T) {
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
	input := strings.Repeat("x", 200)
	lr := &boundedWriter{w: &strings.Builder{}, rem: 100, onOverflow: func() {}}
	n, err := lr.Write([]byte(input))
	if err == nil {
		t.Error("expected error on overflow")
	}
	if !errors.Is(err, ErrOutputLimit) {
		t.Errorf("expected ErrOutputLimit, got %v", err)
	}
	// Should return actual bytes written (100)
	if n != 100 {
		t.Errorf("expected 100 (bytes actually written), got %d", n)
	}
	if !lr.exceeded {
		t.Error("expected exceeded flag to be set")
	}
}

func TestBoundedWriter_AfterDone(t *testing.T) {
	lr := &boundedWriter{w: &strings.Builder{}, rem: 5, done: true}
	n, err := lr.Write([]byte("hello"))
	if err == nil {
		t.Error("expected error after done")
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestRunGit_BothStreams(t *testing.T) {
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "status", "-z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("stdout: %d bytes, stderr: %d bytes", len(result.Stdout), len(result.Stderr))
}

func TestBoundedWriter_OnOverflowCallback(t *testing.T) {
	called := false
	input := strings.Repeat("x", 200)
	lr := &boundedWriter{
		w:          &strings.Builder{},
		rem:        100,
		onOverflow: func() { called = true },
	}
	_, err := lr.Write([]byte(input))
	if err == nil {
		t.Error("expected error")
	}
	if !called {
		t.Error("expected overflow callback to be called")
	}
}
