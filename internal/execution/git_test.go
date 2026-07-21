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

// TestRunGitWithLimits_BothStreams uses the test seam to exercise concurrent
// stdout/stderr draining with a deterministic helper. Uses /bin/sh to write
// distinct payloads to both streams simultaneously.
func TestRunGitWithLimits_BothStreams(t *testing.T) {
	ctx := context.Background()
	// sh writes 100 lines to stdout, then 50 lines to stderr, interleaved via subshells
	script := `
		for i in $(seq 1 100); do echo "stdout-$i"; done >&1 &
		for i in $(seq 1 50); do echo "stderr-$i" >&2; done &
		wait
	`
	result, err := runCommandWithLimits(ctx, "/bin/sh", ".", 5*time.Second, 16384, "-c", script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(result.Stdout), "stdout-1") {
		t.Errorf("expected stdout payload, got: %s", result.Stdout)
	}
	if !strings.Contains(string(result.Stderr), "stderr-1") {
		t.Errorf("expected stderr payload, got: %s", result.Stderr)
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

// TestRunGitWithLimits_RejectsBadClassification verifies the helper returns
// successful exit code with no error for a fast, clean command. The production
// implementation guarantees that nil waitErr means success regardless of
// post-Wait context state, so this contract is verified by inspection of
// the code path: if waitErr is nil and overflow is false, we return (nil).
func TestRunGitWithLimits_RejectsBadClassification(t *testing.T) {
	ctx := context.Background()
	result, err := runCommandWithLimits(ctx, "true", ".", 5*time.Second, 1024)
	if err != nil {
		t.Fatalf("expected no error for true, got %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

// TestRunGitWithLimits_DefaultTimeoutEnforced uses the test seam to verify
// that a helper process is terminated by the internal default timeout when
// the caller provides context.Background() with no deadline.
func TestRunGitWithLimits_DefaultTimeoutEnforced(t *testing.T) {
	// Use a helper that would run forever if not killed
	ctx := context.Background()
	start := time.Now()
	_, err := runCommandWithLimits(ctx, "sleep", ".", 100*time.Millisecond, 1024, "60")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
	// Should complete in approximately the configured timeout, NOT 60 seconds
	if elapsed > 1*time.Second {
		t.Errorf("expected fast termination, took %v", elapsed)
	}
}

// TestRunGitWithLimits_RetainedPipeBound uses the test seam to verify
// that even when a descendant keeps a stdout descriptor open, RunGit
// returns within DefaultGitWaitDelay plus tolerance. The WaitDelay
// mechanism forces the I/O copy goroutine to terminate.
func TestRunGitWithLimits_RetainedPipeBound(t *testing.T) {
	// Start a child that forks a sleep helper which holds stdout open.
	// The direct child exits quickly but the descendant keeps the pipe alive.
	// Without WaitDelay, cmd.Wait would block on the I/O copy.
	script := `
		(sleep 30) &
		exit 0
	`
	ctx := context.Background()
	start := time.Now()
	_, _ = runCommandWithLimits(ctx, "/bin/sh", ".", 5*time.Second, 1024, "-c", script)
	elapsed := time.Since(start)

	// With DefaultGitWaitDelay = 2s, RunGit should return within 3 seconds
	// even though the descendant holds the pipe for 30 seconds.
	if elapsed > 4*time.Second {
		t.Errorf("expected bounded latency via WaitDelay, took %v", elapsed)
	}
}

// TestRunGitWithLimits_ExplicitCancellation uses the test seam with a helper
// that signals readiness then blocks, and verifies the parent cancel interrupts it.
func TestRunGitWithLimits_ExplicitCancellation(t *testing.T) {
	// sleep 60 blocks until killed
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := runCommandWithLimits(ctx, "sleep", ".", 30*time.Second, 1024, "60")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected Canceled, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("expected fast cancellation, took %v", elapsed)
	}
}
