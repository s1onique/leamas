//go:build unix || darwin || linux

package execution

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildTestExecutor creates an executor for testing with the given budget.
func buildTestExecutor(t *testing.T, timeout time.Duration, outputCap int64) *Executor {
	t.Helper()
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

// runHelper executes the test helper with the given mode and manifest.
func runHelper(mode string, manifestFile string) *Result {
	helperPath, err := filepath.Abs(testHelperBinary)
	if err != nil {
		return &Result{Error: &ExecutionError{
			Code:    CodeExecutionCommandNotFound,
			Message: fmt.Sprintf("failed to resolve helper path: %v", err),
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
