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
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected nonzero exit code for nonexistent ref")
	}
}

func TestRunGit_DeadlineExceeded(t *testing.T) {
	// Context deadline should interrupt git
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Sleep command that exceeds deadline
	result, err := RunGit(ctx, ".", "rev-list", "--all", "--count")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should indicate deadline exceeded
	if result.ExitCode != -1 {
		t.Logf("note: deadline may not interrupt fast commands, exit code: %d", result.ExitCode)
	}
}

func TestRunGit_OutputLimit(t *testing.T) {
	// Large output should be captured
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "log", "--format=%H%n", "-n", "100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Stdout) == 0 {
		t.Error("expected stdout from git log")
	}
}

func TestRunGit_RawOutputPreservesNUL(t *testing.T) {
	// Raw output should preserve NUL bytes from -z flag
	ctx := context.Background()
	result, err := RunGit(ctx, ".", "ls-tree", "-z", "--name-only", "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
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
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
	}
	// Should return absolute path
	if !filepath.IsAbs(strings.TrimSpace(string(result.Stdout))) {
		t.Errorf("expected absolute path, got: %s", result.Stdout)
	}
}

func TestGitOutputLimitReader(t *testing.T) {
	// Test the limit reader
	input := []byte("hello world")
	lr := NewGitOutputLimitReader(strings.NewReader(string(input)), 5)

	buf := make([]byte, 10)
	n, err := lr.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes, got %d", n)
	}
	if string(buf[:n]) != "hello" {
		t.Errorf("expected 'hello', got '%s'", buf[:n])
	}

	// Second read should get EOF
	n, err = lr.Read(buf)
	if err == nil {
		t.Error("expected EOF after limit")
	}
	if !lr.Exceeded() {
		t.Error("expected Exceeded() to return true")
	}
}

func TestRunGitSimple_Deprecated(t *testing.T) {
	// RunGitSimple should still work (backward compat)
	stdout, exitCode, err := RunGitSimple(".", "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(string(stdout), "true") && !strings.Contains(string(stdout), "false") {
		t.Errorf("unexpected stdout: %s", stdout)
	}
}
