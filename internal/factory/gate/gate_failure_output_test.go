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

// TestPrintFailureOutput_GitHubActionsMode verifies the GitHub Actions output
// format. The contract: the captured diagnostic content (sentinels) must be
// emitted inline at the top level of the log - NOT inside a collapsible
// ::group:: block - so the actual failure is visible immediately to a reader
// of the CI step. ::stop-commands:: protection is still required around the
// raw output, and a ::error:: annotation must still be emitted.
//
// The script intentionally emits a ::group::literal-text line as part of
// the raw subprocess output. The renderer MUST NOT emit an active
// ::group:: wrapper outside the protected region, but it MUST allow
// ::group:: literals inside the protected region (that is precisely what
// ::stop-commands:: protects against).
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
echo "::group::literal-text-not-a-wrapper"
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
	if !strings.Contains(output, "::group::literal-text-not-a-wrapper") {
		t.Errorf("expected raw ::group:: literal in output, got: %s", output)
	}
	var buf bytes.Buffer
	printFailureOutput(&buf, scriptPath, output, exitCode, cmdErr)
	outputStr := buf.String()

	// Required GHA protocol markers.
	if !strings.Contains(outputStr, "::stop-commands::") {
		t.Errorf("expected '::stop-commands::' protection in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "::error::") {
		t.Errorf("expected '::error::' annotation in output, got: %s", outputStr)
	}
	// Required summary lines (must be visible to a reader).
	if !strings.Contains(outputStr, "exit_code: 42") {
		t.Errorf("expected 'exit_code: 42' in output, got: %s", outputStr)
	}

	// The raw sentinels must appear at the top level of the rendered output.
	if !strings.Contains(outputStr, "stdout-sentinel") {
		t.Errorf("expected 'stdout-sentinel' in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "stderr-sentinel") {
		t.Errorf("expected 'stderr-sentinel' in output, got: %s", outputStr)
	}

	// The raw output must be inside the protected stop-commands/resume region.
	stopMatch := regexp.MustCompile(`::stop-commands::([^\n]+)\n`).FindStringSubmatch(outputStr)
	if len(stopMatch) < 2 {
		t.Fatalf("expected ::stop-commands:: token in output, got: %s", outputStr)
	}
	token := stopMatch[1]
	expectedResume := "::" + token + "::"
	if !strings.Contains(outputStr, expectedResume) {
		t.Fatalf("expected '%s' resume marker in output, got: %s", expectedResume, outputStr)
	}
	rawIdx := strings.Index(outputStr, "stdout-sentinel")
	stopIdx := strings.Index(outputStr, "::stop-commands::")
	stopClose := stopIdx + len("::stop-commands::"+token+"\n")
	resumeIdx := strings.Index(outputStr, expectedResume)
	if !(stopClose <= rawIdx && rawIdx < resumeIdx) {
		t.Errorf("raw output must be inside stop-commands/resume region; got: stopClose=%d rawIdx=%d resumeIdx=%d output=%q",
			stopClose, rawIdx, resumeIdx, outputStr)
	}

	// Region-aware invariant: the renderer-emitted portion (everything
	// OUTSIDE the protected region) must not contain an active workflow
	// command wrapper. The raw output region is opaque and may contain any
	// text - including the ::group:: literal emitted by the failing script.
	outsideProtected := outputStr[:stopClose] + outputStr[resumeIdx:]
	if strings.Contains(outsideProtected, "::group::failure output:") {
		t.Errorf("renderer emitted collapsible group wrapper; output=%q", outputStr)
	}
	if strings.Contains(outsideProtected, "::endgroup::") {
		t.Errorf("renderer emitted endgroup outside protected raw output; output=%q", outputStr)
	}
}

// TestPrintFailureOutput_GHA_Protocol verifies the GitHub Actions protocol
// structure around raw subprocess output. The raw captured output must be
// enclosed by a ::stop-commands::<token>/::<token>:: pair, and the concise
// summary (command: / exit_code:) must follow the resume marker.
//
// The test deliberately embeds workflow-command-shaped literals inside the
// raw subprocess output (::group::, ::error::, ::endgroup::) to prove that
// the protected region is opaque to GitHub Actions. The invariant is:
//
//	The renderer MUST NOT emit an active ::group:: wrapper (or any
//	::endgroup::) outside the protected raw-output region. Raw output
//	inside the protected region is opaque to the renderer and may contain
//	any text, including literals that look like workflow commands.
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
	// Raw subprocess output intentionally contains literal workflow-command
	// text. These literals must NOT be interpreted by the renderer.
	testOutput := strings.Join([]string{
		"::group::must-not-open",
		"::error::must-not-be-interpreted",
		"::endgroup::should-not-close",
		"",
	}, "\n")
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

	// Required markers for the corrected protocol.
	required := []string{"::stop-commands::", expectedResume, "command:", "exit_code:"}
	for _, marker := range required {
		if !strings.Contains(outputStr, marker) {
			t.Errorf("missing required marker '%s' in output", marker)
		}
	}

	// Locate the protected region and isolate the renderer-emitted portion.
	stopIdx := strings.Index(outputStr, "::stop-commands::")
	stopClose := stopIdx + len("::stop-commands::"+token+"\n")
	resumeIdx := strings.Index(outputStr, expectedResume)
	commandIdx := strings.Index(outputStr, "command: ")
	exitIdx := strings.Index(outputStr, "exit_code: ")

	// Ordering contract: stop-commands precedes raw output which precedes
	// the resume marker, which precedes the concise summary lines.
	if !(stopIdx < resumeIdx && resumeIdx < commandIdx && commandIdx < exitIdx) {
		t.Errorf("incorrect GHA marker ordering in output: stopIdx=%d resumeIdx=%d commandIdx=%d exitIdx=%d output=%q",
			stopIdx, resumeIdx, commandIdx, exitIdx, outputStr)
	}

	// Raw output must appear inside the protected stop-commands/resume
	// region (proves stop-commands encloses the data, not just bookends it).
	outputIdx := strings.Index(outputStr, testOutput)
	if outputIdx < 0 {
		t.Fatalf("missing raw output: %q", testOutput)
	}
	if !(stopClose <= outputIdx && outputIdx < resumeIdx) {
		t.Fatalf("raw output outside protected region: stopClose=%d outputIdx=%d resumeIdx=%d",
			stopClose, outputIdx, resumeIdx)
	}

	// Region-aware invariant: the renderer-emitted portion (everything
	// OUTSIDE the protected region) must not contain an active workflow
	// command wrapper. The raw output region is opaque and may contain any
	// text including literals that look like workflow commands - that is
	// precisely what ::stop-commands:: protects against.
	outsideProtected := outputStr[:stopClose] + outputStr[resumeIdx:]
	if strings.Contains(outsideProtected, "::group::failure output:") {
		t.Errorf("renderer emitted collapsible group wrapper; output=%q", outputStr)
	}
	if strings.Contains(outsideProtected, "::endgroup::") {
		t.Errorf("renderer emitted endgroup outside protected raw output; output=%q", outputStr)
	}
}
