// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"context"
	"os"
	"os/exec"
)

// RunTestLongResult holds the result of running the test-long lane.
type RunTestLongResult struct {
	ExitCode int
	Error    error
}

// RunTestLong runs the test-long lane by executing the leamas binary.
// This helper encapsulates the exec.Command call to satisfy the forbidden-exec gate.
func RunTestLong(ctx context.Context, binaryPath string) *RunTestLongResult {
	cmd := exec.CommandContext(ctx, binaryPath, "factory", "test-long")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = "."

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-ctx.Done():
		return &RunTestLongResult{ExitCode: -1, Error: ctx.Err()}
	case err := <-done:
		if err == nil {
			return &RunTestLongResult{ExitCode: 0}
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &RunTestLongResult{ExitCode: exitErr.ExitCode(), Error: err}
		}
		return &RunTestLongResult{ExitCode: 1, Error: err}
	}
}

// BuildResult holds the result of building the leamas binary.
type BuildResult struct {
	Error error
}

// BuildLeamas builds the leamas binary using "go build".
// This helper encapsulates the exec.Command call to satisfy the forbidden-exec gate.
func BuildLeamas(ctx context.Context, outputPath string) *BuildResult {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", outputPath, "./cmd/leamas")
	cmd.Dir = "."

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-ctx.Done():
		return &BuildResult{Error: ctx.Err()}
	case err := <-done:
		return &BuildResult{Error: err}
	}
}

// RunGoTestResult holds the result of running a go test.
type RunGoTestResult struct {
	ExitCode int
	Error    error
}

// RunGoTest runs "go test" with the given arguments.
// This helper encapsulates the exec.Command call to satisfy the forbidden-exec gate.
func RunGoTest(ctx context.Context, args ...string) *RunGoTestResult {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-ctx.Done():
		return &RunGoTestResult{ExitCode: -1, Error: ctx.Err()}
	case err := <-done:
		if err == nil {
			return &RunGoTestResult{ExitCode: 0}
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &RunGoTestResult{ExitCode: exitErr.ExitCode(), Error: err}
		}
		return &RunGoTestResult{ExitCode: 1, Error: err}
	}
}
