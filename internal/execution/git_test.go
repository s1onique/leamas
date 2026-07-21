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
	if result.ExitCode == 0 {
		t.Error("expected nonzero exit code for nonexistent ref")
	}
	_ = err
}

func TestRunGit_DefaultTimeout(t *testing.T) {
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("unexpected error with Background context: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestRunGit_DeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	result, err := RunGit(ctx, ".", "rev-list", "--all", "--count")
	if err == nil {
		t.Error("expected deadline exceeded error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
	if result.ExitCode != -1 && result.ExitCode != 124 {
		t.Logf("note: deadline exit code %d may vary", result.ExitCode)
	}
}

func TestRunGit_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	result, err := RunGit(ctx, ".", "rev-list", "--all", "--count")
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected Canceled, got %v", err)
		}
	}
	t.Logf("result: exit=%d, stdout=%d bytes, err=%v", result.ExitCode, len(result.Stdout), err)
}

func TestRunGit_OutputLimit(t *testing.T) {
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
	if len(result.Stdout) > 0 && !strings.Contains(string(result.Stdout), "\x00") {
		t.Errorf("expected NUL-delimited output, got: %q", result.Stdout)
	}
}

func TestRunGit_StderrCapture(t *testing.T) {
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "status", "--porcelain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	called := 0
	input := strings.Repeat("x", 200)
	lr := &boundedWriter{w: &strings.Builder{}, rem: 100, onOverflow: func() { called++ }}
	n, err := lr.Write([]byte(input))
	if err == nil {
		t.Error("expected error on overflow")
	}
	if !errors.Is(err, ErrOutputLimit) {
		t.Errorf("expected ErrOutputLimit, got %v", err)
	}
	if n != 100 {
		t.Errorf("expected 100 (bytes actually written), got %d", n)
	}
	if called != 1 {
		t.Errorf("expected overflow callback called once, got %d", called)
	}
}

func TestBoundedWriter_AfterOverflow(t *testing.T) {
	called := 0
	lr := &boundedWriter{w: &strings.Builder{}, rem: 5, onOverflow: func() { called++ }}
	// First write of 10 bytes overflows immediately
	_, err := lr.Write([]byte("0123456789"))
	if err == nil {
		t.Error("expected error")
	}
	// Second write should also fail and NOT call overflow again
	_, err = lr.Write([]byte("more"))
	if err == nil {
		t.Error("expected error on second write")
	}
	if called != 1 {
		t.Errorf("expected overflow callback called exactly once, got %d", called)
	}
}

func TestBoundedWriter_ExactLimitThenOneByte(t *testing.T) {
	// Regression test for exact-limit split writes bypassing cancellation
	called := 0
	lr := &boundedWriter{w: &strings.Builder{}, rem: 10, onOverflow: func() { called++ }}

	// First write: exactly 10 bytes - should succeed
	n1, err1 := lr.Write([]byte("0123456789"))
	if err1 != nil {
		t.Errorf("first write should succeed, got error: %v", err1)
	}
	if n1 != 10 {
		t.Errorf("expected 10, got %d", n1)
	}
	if called != 0 {
		t.Errorf("callback should not fire on exact limit, got %d", called)
	}

	// Second write: 1 byte - should trigger overflow
	n2, err2 := lr.Write([]byte("X"))
	if err2 == nil {
		t.Error("expected error on overflow after exact limit")
	}
	if !errors.Is(err2, ErrOutputLimit) {
		t.Errorf("expected ErrOutputLimit, got %v", err2)
	}
	if n2 != 0 {
		t.Errorf("expected 0 bytes written, got %d", n2)
	}
	if called != 1 {
		t.Errorf("expected callback to fire exactly once, got %d", called)
	}
}

func TestRunGit_BothStreams(t *testing.T) {
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "status", "-z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Stdout) == 0 {
		t.Error("expected non-empty stdout")
	}
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

// TestRunGitWithLimits exercises the test seam with deterministic limits.
func TestRunGitWithLimits_Success(t *testing.T) {
	ctx := context.Background()
	result, err := runCommandWithLimits(ctx, "true", ".", 5*time.Second, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestRunGitWithLimits_Overflow(t *testing.T) {
	// Use /bin/sh -c 'yes' to overflow the output limit
	ctx := context.Background()
	_, err := runCommandWithLimits(ctx, "/bin/sh", ".", 5*time.Second, 100, "-c", "yes | head -c 1000")
	if err == nil {
		t.Error("expected ErrOutputLimit")
	}
	if !errors.Is(err, ErrOutputLimit) {
		t.Errorf("expected ErrOutputLimit, got %v", err)
	}
}

func TestAtomicBool(t *testing.T) {
	var ab atomicBool
	if ab.get() {
		t.Error("expected initial false")
	}
	ab.set(true)
	if !ab.get() {
		t.Error("expected true after set")
	}
	ab.set(false)
	if ab.get() {
		t.Error("expected false after reset")
	}
}
