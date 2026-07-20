// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
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

	// Run synchronously - wait for completion before returning
	err := cmd.Run()
	if err == nil {
		return &RunTestLongResult{ExitCode: 0}
	}
	if ctx.Err() != nil {
		return &RunTestLongResult{ExitCode: -1, Error: ctx.Err()}
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return &RunTestLongResult{ExitCode: exitErr.ExitCode(), Error: err}
	}
	return &RunTestLongResult{ExitCode: 1, Error: err}
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

	// Run synchronously - wait for completion before returning
	err := cmd.Run()
	if ctx.Err() != nil {
		return &BuildResult{Error: ctx.Err()}
	}
	return &BuildResult{Error: err}
}

// RunGoTestResult holds the result of running a go test.
type RunGoTestResult struct {
	ExitCode int
	Error    error
}

// RunGoTest runs "go test" with the given arguments in the specified directory.
// The context timeout (runnerDeadline) is separate from the -timeout flag (goTimeout)
// to give Go its own diagnostic window.
func RunGoTest(ctx context.Context, dir string, goTimeout time.Duration, args ...string) *RunGoTestResult {
	// Build the go test command with -timeout flag
	// args should be like: ["-count=1", "-run=<pattern>", "<package>"]
	timeoutArg := fmt.Sprintf("-timeout=%s", goTimeout)
	testArgs := append([]string{"test", timeoutArg}, args...)

	cmd := exec.CommandContext(ctx, "go", testArgs...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run synchronously - wait for completion before returning
	err := cmd.Run()
	if err == nil {
		return &RunGoTestResult{ExitCode: 0}
	}
	if ctx.Err() != nil {
		return &RunGoTestResult{ExitCode: -1, Error: ctx.Err()}
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return &RunGoTestResult{ExitCode: exitErr.ExitCode(), Error: err}
	}
	return &RunGoTestResult{ExitCode: 1, Error: err}
}
