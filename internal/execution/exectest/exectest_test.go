package exectest

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

// TestOutputPreservesStdoutOnNonZeroExit verifies that stdout is captured
// even when the command exits with a non-zero status code.
func TestOutputPreservesStdoutOnNonZeroExit(t *testing.T) {
	req := Request{
		Name: "sh",
		Args: []string{"-c", "echo hello && exit 1"},
	}
	output, err := Output(req)

	if err == nil {
		t.Fatal("expected non-nil error for exit code 1")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *ExitError, got %T: %v", err, err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	if string(output) != "hello\n" {
		t.Errorf("expected stdout 'hello\\n', got %q", string(output))
	}
}

// TestExitErrorUnwrap verifies that both exectest.ExitError and os/exec.ExitError
// can be extracted from the wrapped error.
func TestExitErrorUnwrap(t *testing.T) {
	req := Request{
		Name: "sh",
		Args: []string{"-c", "exit 42"},
	}
	_, err := Output(req)

	// Check exectest.ExitError via errors.As
	var exectestErr *ExitError
	if !errors.As(err, &exectestErr) {
		t.Fatalf("expected *ExitError via errors.As, got %T", err)
	}

	// Check os/exec.ExitError via errors.As (Unwrap chain)
	var execErr *exec.ExitError
	if !errors.As(err, &execErr) {
		t.Fatalf("expected *exec.ExitError via Unwrap chain, got %T", err)
	}

	if exectestErr.ExitCode() != 42 {
		t.Errorf("expected exit code 42, got %d", exectestErr.ExitCode())
	}
}

// TestCombinedOutputPreservesBothStreams verifies that CombinedOutput captures
// both stdout and stderr.
func TestCombinedOutputPreservesBothStreams(t *testing.T) {
	req := Request{
		Name: "sh",
		Args: []string{"-c", "echo stdout && echo stderr >&2"},
	}
	output, err := CombinedOutput(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Combined output should contain both stdout and stderr
	outputStr := string(output)
	if outputStr != "stdout\nstderr\n" {
		t.Errorf("expected 'stdout\\nstderr\\n', got %q", outputStr)
	}
}

// TestExitErrorExitCodeSurvivesWrapping verifies that ExitCode() method
// is accessible on the wrapped error.
func TestExitErrorExitCodeSurvivesWrapping(t *testing.T) {
	req := Request{
		Name: "sh",
		Args: []string{"-c", "exit 42"},
	}
	_, err := Output(req)

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *ExitError, got %T: %v", err, err)
	}

	if exitErr.ExitCode() != 42 {
		t.Errorf("expected exit code 42, got %d", exitErr.ExitCode())
	}

	// Verify the underlying exec.ExitError is accessible
	if exitErr.ExitError == nil {
		t.Error("expected non-nil underlying ExitError")
	}
}

// TestSpawnFailureDistinguishableFromCommandFailure verifies that spawn
// failures (command not found) are distinguishable from command failures
// (command found but exited non-zero).
func TestSpawnFailureDistinguishableFromCommandFailure(t *testing.T) {
	req := Request{
		Name: "nonexistent_command_xyz",
		Args: []string{},
	}
	_, err := Output(req)

	// A spawn failure should not be an ExitError
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		t.Error("spawn failure should not be wrapped as ExitError")
	}

	// It should be some other error (likely *exec.Error)
	if err == nil {
		t.Error("expected non-nil error for spawn failure")
	}
}

// TestDirSemanticsPreserved verifies that the Dir field is applied correctly.
func TestDirSemanticsPreserved(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()

	req := Request{
		Name: "pwd",
		Args: []string{},
		Dir:  tmpDir,
	}
	output, err := Output(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// pwd should output the current directory
	if string(output) != tmpDir+"\n" {
		t.Errorf("expected pwd to return %q, got %q", tmpDir+"\n", string(output))
	}
}

// TestEnvSemanticsPreserved verifies environment variable handling.
// When Env is nil, the current environment is inherited.
// When Env is non-nil, it replaces the environment.
func TestEnvSemanticsPreserved(t *testing.T) {
	// Use repository-specific variable to avoid host contamination
	t.Setenv("LEAMAS_EXECTEST_VALUE", "inherited")

	output, err := Output(Request{
		Name: "sh",
		Args: []string{"-c", `printf '%s' "$LEAMAS_EXECTEST_VALUE"`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(output); got != "inherited" {
		t.Fatalf("inherited environment: got %q", got)
	}

	// Test with explicit Env (replacement)
	output, err = Output(Request{
		Name: "sh",
		Args: []string{"-c", `printf '%s' "$LEAMAS_EXECTEST_VALUE"`},
		Env:  []string{"LEAMAS_EXECTEST_VALUE=replaced"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(output); got != "replaced" {
		t.Fatalf("replacement environment: got %q", got)
	}
}

// TestContextDeadlineTerminatesProcess verifies that context deadline
// terminates the subprocess promptly.
func TestContextDeadlineTerminatesProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := Output(Request{
		Ctx:  ctx,
		Name: "sleep",
		Args: []string{"10"}, // Sleep for 10 seconds
	})
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("expected command failure after context deadline")
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("context error = %v, want deadline exceeded", ctx.Err())
	}
	// Process should be terminated promptly (within 2 seconds)
	if elapsed > 2*time.Second {
		t.Fatalf("command was not terminated promptly: %s", elapsed)
	}
}
