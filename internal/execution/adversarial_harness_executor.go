//go:build unix || darwin || linux

package execution

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// testHelperSource is the source path for the test helper.
const testHelperSource = "internal/execution/testdata/testhelper/main.go"

// testHelperBinary is the compiled helper path.
const testHelperBinary = "internal/execution/testdata/testhelper/main"

var (
	helperBuildOnce sync.Once
	helperBuildErr  error
	helperPath      string
)

// findRepoRoot finds the repository root by looking for go.mod.
func findRepoRoot() (string, error) {
	// Start from the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk up the directory tree looking for go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found in any parent directory")
}

// ensureHelperBuilt builds the test helper from source if not already built.
func ensureHelperBuilt() error {
	helperBuildOnce.Do(func() {
		repoRoot, err := findRepoRoot()
		if err != nil {
			helperBuildErr = err
			return
		}

		sourcePath := filepath.Join(repoRoot, testHelperSource)
		outputPath := filepath.Join(repoRoot, testHelperBinary)

		// Check if already built and source hasn't changed
		if info, err := os.Stat(outputPath); err == nil {
			if srcInfo, err := os.Stat(sourcePath); err == nil {
				if info.ModTime().After(srcInfo.ModTime()) {
					helperPath = outputPath
					return // Already built and up-to-date
				}
			}
		}

		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			helperBuildErr = fmt.Errorf("failed to create helper directory: %w", err)
			return
		}

		// Build the helper
		cmd := exec.Command("go", "build", "-o", outputPath, sourcePath)
		cmd.Dir = repoRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			helperBuildErr = fmt.Errorf("failed to build test helper: %w, output: %s", err, output)
			return
		}

		helperPath = outputPath
	})
	return helperBuildErr
}

// getHelperPath returns the path to the built helper.
func getHelperPath() (string, error) {
	if err := ensureHelperBuilt(); err != nil {
		return "", err
	}
	return helperPath, nil
}

// buildTestExecutor creates an executor for testing with the given budget.
func buildTestExecutor(t *testing.T, timeout time.Duration, outputCap int64) *Executor {
	t.Helper()

	// Ensure test helper is built from source
	if _, err := getHelperPath(); err != nil {
		t.Fatalf("failed to build test helper: %v", err)
	}

	budget := &Budget{
		Deadline:         time.Now().Add(timeout),
		MaxConcurrent:    4,
		MaxStarts:        64,
		MaxTaskDepth:     8,
		MaxOutputBytes:   outputCap,
		TerminationGrace: 500 * time.Millisecond,
		PostKillWait:     500 * time.Millisecond,
	}
	executor, err := NewExecutor(budget, NewTestExecutionRoot())
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	return executor
}

// calculateMaxTestDuration calculates the maximum expected test duration.
func calculateMaxTestDuration(timeout, grace, postKill, slack time.Duration) time.Duration {
	return timeout + grace + postKill + slack
}

// locateHelperBinary returns the path to the test helper binary.
func locateHelperBinary() (string, error) {
	return getHelperPath()
}

// runHelper executes the test helper with the given mode and manifest.
func runHelper(mode string, manifestFile string) *Result {
	helperPath, err := getHelperPath()
	if err != nil {
		return &Result{Error: &ExecutionError{
			Code:    CodeExecutionCommandNotFound,
			Message: fmt.Sprintf("failed to build helper: %v", err),
		}}
	}

	cmd := newTestExecCommand("/bin/sh", "-c",
		fmt.Sprintf("LEAMAS_EXEC_TEST_PID_FILE=%s %s %s", manifestFile, helperPath, mode))

	return executeTestCommand(cmd)
}

// newTestExecCommand creates a test execution command.
func newTestExecCommand(name string, args ...string) *testCmd {
	return &testCmd{name: name, args: args}
}

// testCmd is a test command for the harness.
type testCmd struct {
	name string
	args []string
}

func (c *testCmd) String() string {
	return c.name + " " + strings.Join(c.args, " ")
}

// executeTestCommand executes a test command and returns the result.
func executeTestCommand(cmd *testCmd) *Result {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	execCmd := newTestExecCmdContext(ctx, cmd.name, cmd.args...)
	execCmd.Stdout = io.Discard
	execCmd.Stderr = io.Discard

	err := execCmd.Run()
	if err == nil {
		return &Result{ExitCode: 0}
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return &Result{ExitCode: ee.ExitCode()}
	}
	return &Result{
		ExitCode: -1,
		Error: &ExecutionError{
			Code:    CodeExecutionUnknown,
			Message: err.Error(),
		},
	}
}

// newTestExecCmdContext creates an exec.Cmd with the given context.
func newTestExecCmdContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
