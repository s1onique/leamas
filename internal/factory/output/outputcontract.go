// Package output provides the Leamas output contract for factory commands.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// ContractVerifier runs checks to verify output contract compliance.
type ContractVerifier struct {
	Commands []CommandSpec
}

// CommandSpec defines a factory command to test.
type CommandSpec struct {
	Args               []string
	ShouldPass         bool
	JSONArgs           []string // Explicit JSON args to test
	ExpectJSON         bool     // Whether to verify JSON output
	SkipMultilineCheck bool     // Skip one-line check (for JSON output which is naturally multi-line)
}

// Verify runs all contract verification checks.
func (v *ContractVerifier) Verify() []checks.Finding {
	var findings []checks.Finding

	for _, spec := range v.Commands {
		findings = append(findings, v.verifyCommand(spec)...)
	}

	return findings
}

func (v *ContractVerifier) verifyCommand(spec CommandSpec) []checks.Finding {
	var findings []checks.Finding

	// Run the command
	cmd := exec.Command("./bin/leamas", spec.Args...)
	cmd.Dir = "."
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	output := stdout.String()

	// Check 1: Success output should be one line (skip for JSON output)
	if spec.ShouldPass && !spec.SkipMultilineCheck && strings.Count(output, "\n") > 1 && output != "" {
		findings = append(findings, checks.Finding{
			Path:    strings.Join(spec.Args, " "),
			Kind:    "multiline_success",
			Message: "success output should be one line",
		})
	}

	// Check 2: Strict JSON verification when expected
	if spec.ExpectJSON {
		jsonArgs := spec.JSONArgs
		if jsonArgs == nil {
			jsonArgs = append(spec.Args, "--json")
		}
		code, jsonOut, jsonErr := v.runCommand(jsonArgs)

		// Check exit code
		if spec.ShouldPass && code != 0 {
			findings = append(findings, checks.Finding{
				Path:    strings.Join(spec.Args, " "),
				Kind:    "json_exit_code",
				Message: fmt.Sprintf("--json should exit 0, got %d: %s", code, strings.TrimSpace(jsonErr)),
			})
		}

		// Check non-empty output
		if strings.TrimSpace(jsonOut) == "" {
			findings = append(findings, checks.Finding{
				Path:    strings.Join(spec.Args, " "),
				Kind:    "missing_json",
				Message: "--json produced empty output",
			})
		}

		// Check valid JSON
		if !isValidJSON(jsonOut) {
			findings = append(findings, checks.Finding{
				Path:    strings.Join(spec.Args, " "),
				Kind:    "invalid_json",
				Message: "--json output is not valid JSON",
			})
		}
	}

	// Check 3: No ANSI codes
	if containsANSICodes(output) {
		findings = append(findings, checks.Finding{
			Path:    strings.Join(spec.Args, " "),
			Kind:    "ansi_codes",
			Message: "output contains ANSI escape codes",
		})
	}

	// Check 5: No prose in output
	if containsProse(output) {
		findings = append(findings, checks.Finding{
			Path:    strings.Join(spec.Args, " "),
			Kind:    "prose_output",
			Message: "output contains prose text",
		})
	}

	// Check 6: Exit code consistency
	if spec.ShouldPass && exitCode != 0 {
		findings = append(findings, checks.Finding{
			Path:    strings.Join(spec.Args, " "),
			Kind:    "exit_code_mismatch",
			Message: fmt.Sprintf("command should pass but got exit code %d", exitCode),
		})
	}

	return findings
}

func (v *ContractVerifier) runCommand(args []string) (int, string, string) {
	cmd := exec.Command("./bin/leamas", args...)
	cmd.Dir = "."
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}
	return exitCode, stdout.String(), stderr.String()
}

func containsANSICodes(s string) bool {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.MatchString(s)
}

func containsProse(s string) bool {
	prosePatterns := []string{
		"checking", "verified", "completed", "finished",
		"Running", "Checking", "Verifying",
		"Found", "Detected", "Analyzing",
	}
	for _, pattern := range prosePatterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}

func isValidJSON(s string) bool {
	if s == "" {
		return false
	}
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}

// DefaultVerifier returns the default contract verifier with standard commands.
// Only tests commands that use the output contract format.
func DefaultVerifier() *ContractVerifier {
	return &ContractVerifier{
		Commands: []CommandSpec{
			// Coverage: one-line success (default path, no breakdown)
			{Args: []string{"factory", "coverage", "--profile", ".factory/coverage.out", "--min-total", "0"}, ShouldPass: true},
			// Coverage thresholds: one-line success
			{Args: []string{"factory", "coverage", "--thresholds"}, ShouldPass: true},
			// Coverage with JSON: strict JSON verification
			{
				Args:               []string{"factory", "coverage", "--profile", ".factory/coverage.out", "--min-total", "0", "--no-breakdown"},
				ShouldPass:         true,
				JSONArgs:           []string{"factory", "coverage", "--profile", ".factory/coverage.out", "--min-total", "0", "--no-breakdown", "--json"},
				ExpectJSON:         true,
				SkipMultilineCheck: true,
			},
			// Digest: one-line success
			{Args: []string{"factory", "digest", "--output", "/dev/null"}, ShouldPass: true},
		},
	}
}

// RunContractCheck is the entry point for the output-contract verifier.
func RunContractCheck(root string) []checks.Finding {
	// Check if binary exists
	if _, err := os.Stat("bin/leamas"); os.IsNotExist(err) {
		return []checks.Finding{
			{
				Path:    "bin/leamas",
				Kind:    "missing_binary",
				Message: "bin/leamas not found. Run 'make build' first.",
			},
		}
	}

	verifier := DefaultVerifier()
	return verifier.Verify()
}

// ContractCheck returns findings for the output contract check.
func ContractCheck(root string) []checks.Finding {
	return RunContractCheck(root)
}
