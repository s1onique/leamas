// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunCommandInDir_ExecutesOnce verifies the command runs exactly once.
func TestRunCommandInDir_ExecutesOnce(t *testing.T) {
	tmpDir := t.TempDir()
	markerPath := filepath.Join(tmpDir, "run-count.txt")
	script := `#!/bin/bash
echo "run" >> "` + markerPath + `"
exit 0
`
	scriptPath := filepath.Join(tmpDir, "counting-script.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	_, _, _ = runCommandInDir(tmpDir, scriptPath)
	content, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("failed to read marker file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected exactly 1 run, got %d", len(lines))
	}
}

// TestRunCommandInDir_CapturesSuccessOutput verifies successful command output is captured.
func TestRunCommandInDir_CapturesSuccessOutput(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "success-script.sh")
	script := `#!/bin/bash
echo "success output"
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	output, exitCode, cmdErr := runCommandInDir(tmpDir, scriptPath)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if cmdErr != nil {
		t.Errorf("expected no error on success, got %v", cmdErr)
	}
	if !strings.Contains(output, "success output") {
		t.Errorf("expected success output, got: %s", output)
	}
}

// TestRunCommandWithEnvInDir tests the environment variable handling.
func TestRunCommandWithEnvInDir(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "env-test.sh")
	script := `#!/bin/bash
echo "CUSTOM_VAR=${CUSTOM_VAR}"
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	env := []string{"CUSTOM_VAR=test-value"}
	output, exitCode, cmdErr := runCommandWithEnvInDir(tmpDir, env, scriptPath)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if cmdErr != nil {
		t.Errorf("expected no error, got %v", cmdErr)
	}
	if !strings.Contains(output, "CUSTOM_VAR=test-value") {
		t.Errorf("expected CUSTOM_VAR=test-value in output, got: %s", output)
	}
}

// TestFailureOutput_NoTemporaryFiles verifies no temp files are left behind.
func TestFailureOutput_NoTemporaryFiles(t *testing.T) {
	tmpDir := t.TempDir()
	entriesBefore := listDir(tmpDir)
	scriptPath := filepath.Join(tmpDir, "fail-script.sh")
	script := `#!/bin/bash
echo "failing"
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	runCommandInDir(tmpDir, scriptPath)
	entriesAfter := listDir(tmpDir)
	if len(entriesAfter) != len(entriesBefore)+1 {
		t.Errorf("unexpected new files created: before=%v, after=%v", entriesBefore, entriesAfter)
	}
}

// TestCombinedOutput_Interleaving verifies CombinedOutput preserves interleaving.
func TestCombinedOutput_Interleaving(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "interleave.sh")
	script := `#!/bin/bash
echo "out1"
echo "err1" >&2
echo "out2"
echo "err2" >&2
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	output, exitCode, cmdErr := runCommandInDir(tmpDir, scriptPath)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if cmdErr == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(output, "out1") {
		t.Errorf("expected 'out1' in output, got: %s", output)
	}
	if !strings.Contains(output, "out2") {
		t.Errorf("expected 'out2' in output, got: %s", output)
	}
	if !strings.Contains(output, "err1") {
		t.Errorf("expected 'err1' in output, got: %s", output)
	}
	if !strings.Contains(output, "err2") {
		t.Errorf("expected 'err2' in output, got: %s", output)
	}
	want := []string{"out1", "err1", "out2", "err2"}
	last := -1
	for _, marker := range want {
		pos := strings.Index(output, marker)
		if pos <= last {
			t.Fatalf("unexpected output order, got: %q", output)
		}
		last = pos
	}
}

// TestRunCommand_CapturesExecutionError verifies command execution errors are captured.
func TestRunCommand_CapturesExecutionError(t *testing.T) {
	cmd := exec.Command("/nonexistent/command/path/that/does/not/exist")
	_, exitCode, cmdErr := runCommand(cmd)
	if exitCode != -1 {
		t.Errorf("expected exit code -1 for execution error, got %d", exitCode)
	}
	if cmdErr == nil {
		t.Error("expected error for non-existent command, got nil")
	}
}

// TestPrintFailureOutput_ExecutionErrorOutput verifies execution errors are displayed.
func TestPrintFailureOutput_ExecutionErrorOutput(t *testing.T) {
	testErr := errors.New("executable file not found")
	var buf bytes.Buffer
	printFailureOutput(&buf, "nonexistent-command", "", -1, testErr)
	outputStr := buf.String()
	if !strings.Contains(outputStr, "execution_error:") {
		t.Errorf("expected 'execution_error:' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "executable file not found") {
		t.Errorf("expected error message in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "exit_code: -1") {
		t.Errorf("expected 'exit_code: -1' in output, got: %s", outputStr)
	}
}

// TestPrintFailureOutput_GitHubActionsExecutionError verifies execution errors in GHA mode.
func TestPrintFailureOutput_GitHubActionsExecutionError(t *testing.T) {
	oldGithubActions := os.Getenv("GITHUB_ACTIONS")
	os.Setenv("GITHUB_ACTIONS", "true")
	defer func() {
		if oldGithubActions == "" {
			os.Unsetenv("GITHUB_ACTIONS")
		} else {
			os.Setenv("GITHUB_ACTIONS", oldGithubActions)
		}
	}()
	testErr := errors.New("executable file not found")
	var buf bytes.Buffer
	printFailureOutput(&buf, "nonexistent-command", "", -1, testErr)
	outputStr := buf.String()
	if !strings.Contains(outputStr, "execution_error:") {
		t.Errorf("expected 'execution_error:' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "::error::nonexistent-command execution failed:") {
		t.Errorf("expected GHA execution error annotation in output, got: %s", outputStr)
	}
}

// TestGitCommand_ExecutesOnce verifies git commands run exactly once.
func TestGitCommand_ExecutesOnce(t *testing.T) {
	tmpDir := t.TempDir()
	markerPath := filepath.Join(tmpDir, "git-run-count.txt")
	script := `#!/bin/bash
count=$(cat "` + markerPath + `" 2>/dev/null || echo 0)
echo $((count + 1)) > "` + markerPath + `"
echo "stderr output" >&2
exit 128
`
	scriptPath := filepath.Join(tmpDir, "fake-git.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	cmd := exec.Command(scriptPath)
	cmd.Dir = tmpDir
	output, exitCode, cmdErr := runCommand(cmd)
	content, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("failed to read marker file: %v", err)
	}
	if strings.TrimSpace(string(content)) != "1" {
		t.Errorf("expected exactly 1 git invocation, got: %s", string(content))
	}
	if exitCode != 128 {
		t.Errorf("expected exit code 128, got %d", exitCode)
	}
	if cmdErr == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(output, "stderr output") {
		t.Errorf("expected 'stderr output' in output, got: %s", output)
	}
}

// TestNewStopCommandsToken_Unique verifies tokens are unique.
func TestNewStopCommandsToken_Unique(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := newStopCommandsToken()
		if err != nil {
			t.Fatalf("newStopCommandsToken failed: %v", err)
		}
		if tokens[token] {
			t.Errorf("duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}

func listDir(dir string) []string {
	entries, _ := os.ReadDir(dir)
	result := make([]string, len(entries))
	for i, e := range entries {
		result[i] = e.Name()
	}
	return result
}
