// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPrintFailureOutput_StandardMode verifies the standard output format on failure.
func TestPrintFailureOutput_StandardMode(t *testing.T) {
	// Ensure we run in plain mode, not GitHub Actions mode.
	t.Setenv("GITHUB_ACTIONS", "false")

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-failing-command.sh")
	script := `#!/bin/bash
echo "stdout-sentinel"
echo "stderr-sentinel" >&2
exit 23
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	output, exitCode, cmdErr := runCommandInDir(tmpDir, scriptPath)
	if exitCode != 23 {
		t.Errorf("expected exit code 23, got %d", exitCode)
	}
	if cmdErr == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(output, "stdout-sentinel") {
		t.Errorf("expected stdout-sentinel in output, got: %s", output)
	}
	if !strings.Contains(output, "stderr-sentinel") {
		t.Errorf("expected stderr-sentinel in output, got: %s", output)
	}
	var buf bytes.Buffer
	printFailureOutput(&buf, scriptPath, output, exitCode, cmdErr)
	outputStr := buf.String()
	if !strings.Contains(outputStr, "--- failure output:") {
		t.Errorf("expected '--- failure output:' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "stdout-sentinel") {
		t.Errorf("expected 'stdout-sentinel' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "stderr-sentinel") {
		t.Errorf("expected 'stderr-sentinel' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "--- end failure output ---") {
		t.Errorf("expected '--- end failure output ---' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "exit_code: 23") {
		t.Errorf("expected 'exit_code: 23' in output, got: %s", outputStr)
	}
}

// TestPrintFailureOutput_GitHubActionsMode verifies the GitHub Actions output format.
func TestPrintFailureOutput_GitHubActionsMode(t *testing.T) {
	oldGithubActions := os.Getenv("GITHUB_ACTIONS")
	os.Setenv("GITHUB_ACTIONS", "true")
	defer func() {
		if oldGithubActions == "" {
			os.Unsetenv("GITHUB_ACTIONS")
		} else {
			os.Setenv("GITHUB_ACTIONS", oldGithubActions)
		}
	}()
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-failing-command.sh")
	script := `#!/bin/bash
echo "stdout-sentinel"
echo "stderr-sentinel" >&2
exit 42
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	output, exitCode, cmdErr := runCommandInDir(tmpDir, scriptPath)
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}
	if cmdErr == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(output, "stdout-sentinel") {
		t.Errorf("expected stdout-sentinel in output, got: %s", output)
	}
	if !strings.Contains(output, "stderr-sentinel") {
		t.Errorf("expected stderr-sentinel in output, got: %s", output)
	}
	var buf bytes.Buffer
	printFailureOutput(&buf, scriptPath, output, exitCode, cmdErr)
	outputStr := buf.String()
	if !strings.Contains(outputStr, "::group::failure output:") {
		t.Errorf("expected '::group::failure output:' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "stdout-sentinel") {
		t.Errorf("expected 'stdout-sentinel' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "stderr-sentinel") {
		t.Errorf("expected 'stderr-sentinel' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "::endgroup::") {
		t.Errorf("expected '::endgroup::' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "exit_code: 42") {
		t.Errorf("expected 'exit_code: 42' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "::error::") {
		t.Errorf("expected '::error::' annotation in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "::stop-commands::") {
		t.Errorf("expected '::stop-commands::' protection in output, got: %s", outputStr)
	}
}

// TestPrintFailureOutput_GHA_Protocol verifies GHA protocol structure is correct.
func TestPrintFailureOutput_GHA_Protocol(t *testing.T) {
	oldGithubActions := os.Getenv("GITHUB_ACTIONS")
	os.Setenv("GITHUB_ACTIONS", "true")
	defer func() {
		if oldGithubActions == "" {
			os.Unsetenv("GITHUB_ACTIONS")
		} else {
			os.Setenv("GITHUB_ACTIONS", oldGithubActions)
		}
	}()
	var buf bytes.Buffer
	testOutput := "::error::must-not-be-interpreted\n::endgroup::should-not-close\n"
	printFailureOutput(&buf, "test-command", testOutput, 1, nil)
	outputStr := buf.String()
	stopMatch := regexp.MustCompile(`::stop-commands::([^\n]+)\n`).FindStringSubmatch(outputStr)
	if len(stopMatch) < 2 {
		t.Fatalf("expected ::stop-commands:: token in output, got: %s", outputStr)
	}
	token := stopMatch[1]
	expectedResume := "::" + token + "::"
	if !strings.Contains(outputStr, expectedResume) {
		t.Errorf("expected '%s' resume marker in output, got: %s", expectedResume, outputStr)
	}
	required := []string{"::group::", "::stop-commands::", expectedResume, "command:", "exit_code:"}
	for _, marker := range required {
		if !strings.Contains(outputStr, marker) {
			t.Errorf("missing required marker '%s' in output", marker)
		}
	}
	groupIdx := strings.Index(outputStr, "::group::")
	stopIdx := strings.Index(outputStr, "::stop-commands::")
	resumeIdx := strings.Index(outputStr, expectedResume)
	commandIdx := strings.Index(outputStr, "command: ")
	exitIdx := strings.Index(outputStr, "exit_code: ")
	if !(groupIdx < stopIdx && stopIdx < resumeIdx && resumeIdx < commandIdx && commandIdx < exitIdx) {
		t.Errorf("incorrect GHA marker ordering in output: %s", outputStr)
	}
	outputIdx := strings.Index(outputStr, testOutput)
	if outputIdx < 0 {
		t.Fatalf("missing raw output: %q", testOutput)
	}
	stopClose := stopIdx + len("::stop-commands::"+token+"\n")
	if !(stopClose <= outputIdx && outputIdx < resumeIdx) {
		t.Fatalf("raw output outside protected region")
	}
}
