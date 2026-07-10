// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// runCommand captures stdout and stderr together using CombinedOutput.
// This preserves original interleaving and ensures diagnostics are available
// on failure without rerunning.
func runCommand(cmd *exec.Cmd) (output string, exitCode int, cmdErr error) {
	out, err := cmd.CombinedOutput()
	output = string(out)
	if err == nil {
		return output, 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return output, exitErr.ExitCode(), err
	}
	// Command execution failed (not just non-zero exit)
	return output, -1, err
}

// runCommandInDir runs a command in the specified directory.
func runCommandInDir(dir, name string, args ...string) (output string, exitCode int, cmdErr error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return runCommand(cmd)
}

// runCommandWithEnvInDir runs a command with custom environment variables.
func runCommandWithEnvInDir(dir string, env []string, name string, args ...string) (output string, exitCode int, cmdErr error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	return runCommand(cmd)
}

// newStopCommandsToken generates a unique random token for GHA stop-commands.
// Uses crypto/rand for secure token generation.
func newStopCommandsToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate stop-commands token: %w", err)
	}
	return "leamas-" + hex.EncodeToString(raw[:]), nil
}

// printFailureOutput prints captured command output on failure.
// In GitHub Actions, this wraps output in ::group:: for collapsible logs.
// Uses ::stop-commands:: to protect raw output from workflow-command interpretation.
// Writes to w (defaults to os.Stdout if nil).
func printFailureOutput(w io.Writer, command string, output string, exitCode int, cmdErr error) {
	if w == nil {
		w = os.Stdout
	}

	// Check if we're in GitHub Actions environment
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		// Generate unique token to prevent GHA from interpreting raw output
		// Fail closed if token generation fails (no fallback to predictable value)
		token, err := newStopCommandsToken()
		if err != nil {
			// Cannot safely format output without a valid token
			fmt.Fprintf(w, "command: %s\n", command)
			fmt.Fprintf(w, "exit_code: %d\n", exitCode)
			fmt.Fprintf(w, "GHA output formatting failed: %v\n", err)
			return
		}
		fmt.Fprintf(w, "::group::failure output: %s\n", command)
		fmt.Fprintf(w, "::stop-commands::%s\n", token)
		fmt.Fprint(w, output)
		if output != "" && !strings.HasSuffix(output, "\n") {
			fmt.Fprintln(w)
		}
		// Resume command processing - correct syntax has two leading colons
		fmt.Fprintf(w, "::%s::\n", token)
		fmt.Fprintf(w, "command: %s\n", command)
		fmt.Fprintf(w, "exit_code: %d\n", exitCode)
		if cmdErr != nil && exitCode == -1 {
			fmt.Fprintf(w, "execution_error: %v\n", cmdErr)
		}
		fmt.Fprintln(w, "::endgroup::")
		// Optional concise annotation
		if cmdErr != nil && exitCode == -1 {
			fmt.Fprintf(w, "::error::%s execution failed: %v\n", command, cmdErr)
		} else {
			fmt.Fprintf(w, "::error::%s failed with exit code %d\n", command, exitCode)
		}
	} else {
		// Standard output format
		if output != "" {
			fmt.Fprintf(w, "--- failure output: %s ---\n", command)
			fmt.Fprint(w, output)
			if !strings.HasSuffix(output, "\n") {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "--- end failure output ---\n")
		}
		fmt.Fprintf(w, "command: %s\n", command)
		fmt.Fprintf(w, "exit_code: %d\n", exitCode)
		if cmdErr != nil && exitCode == -1 {
			fmt.Fprintf(w, "execution_error: %v\n", cmdErr)
		}
	}
}

// runToolchainChecks runs all Go toolchain checks and reports failures.
func runToolchainChecks(root string, failed *bool) {
	fmt.Printf("\n--- Go toolchain ---\n")

	// go mod tidy
	fmt.Printf("  go mod tidy...")
	output, exitCode, cmdErr := runCommandInDir(root, "go", "mod", "tidy")
	if exitCode != 0 || cmdErr != nil {
		fmt.Printf(" FAILED\n")
		printFailureOutput(nil, "go mod tidy", output, exitCode, cmdErr)
		*failed = true
	} else {
		fmt.Printf(" OK\n")
	}

	// Check go.mod/go.sum didn't change - run exactly once
	if checks.FileExists(filepath.Join(root, "go.sum")) {
		cmd := exec.Command("git", "diff", "--quiet", "--", "go.mod", "go.sum")
		cmd.Dir = root
		output, exitCode, cmdErr := runCommand(cmd)

		switch exitCode {
		case 0:
			// Clean, no changes
		case 1:
			fmt.Printf("  go.mod/go.sum changed after tidy\n")
			*failed = true
		default:
			// Infrastructure failure (git not found, invalid dir, etc.)
			fmt.Printf("  git diff failed unexpectedly\n")
			printFailureOutput(nil, "git diff --quiet -- go.mod go.sum", output, exitCode, cmdErr)
			*failed = true
		}
	} else {
		cmd := exec.Command("git", "diff", "--quiet", "--", "go.mod")
		cmd.Dir = root
		output, exitCode, cmdErr := runCommand(cmd)

		switch exitCode {
		case 0:
			// Clean, no changes
		case 1:
			fmt.Printf("  go.mod changed after tidy\n")
			*failed = true
		default:
			// Infrastructure failure
			fmt.Printf("  git diff failed unexpectedly\n")
			printFailureOutput(nil, "git diff --quiet -- go.mod", output, exitCode, cmdErr)
			*failed = true
		}
	}

	// gofmt check - capture output for diagnostics
	fmt.Printf("  gofmt...")
	cmd := exec.Command("gofmt", "-l", ".")
	cmd.Dir = root
	output, exitCode, cmdErr = runCommand(cmd)
	if cmdErr != nil {
		// gofmt failed to execute
		fmt.Printf(" FAILED\n")
		printFailureOutput(nil, "gofmt", output, exitCode, cmdErr)
		*failed = true
	} else if exitCode != 0 {
		// gofmt returned non-zero even without execution error
		fmt.Printf(" FAILED\n")
		printFailureOutput(nil, "gofmt", output, exitCode, cmdErr)
		*failed = true
	} else if len(strings.TrimSpace(output)) > 0 {
		// Exit 0 but files listed = formatting issues
		fmt.Printf(" FAILED\n")
		fmt.Printf("    Unformatted files:\n")
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, f := range lines {
			if f != "" {
				fmt.Printf("    - %s\n", f)
			}
		}
		*failed = true
	} else {
		fmt.Printf(" OK\n")
	}

	// go vet
	fmt.Printf("  go vet ./...")
	output, exitCode, cmdErr = runCommandInDir(root, "go", "vet", "./...")
	if exitCode != 0 || cmdErr != nil {
		fmt.Printf(" FAILED\n")
		printFailureOutput(nil, "go vet ./...", output, exitCode, cmdErr)
		*failed = true
	} else {
		fmt.Printf(" OK\n")
	}

	// go test
	fmt.Printf("  go test ./...")
	output, exitCode, cmdErr = runCommandInDir(root, "go", "test", "./...")
	if exitCode != 0 || cmdErr != nil {
		fmt.Printf(" FAILED\n")
		printFailureOutput(nil, "go test ./...", output, exitCode, cmdErr)
		*failed = true
	} else {
		fmt.Printf(" OK\n")
	}

	// CGO_ENABLED=0 build
	fmt.Printf("  static build...")
	env := append(os.Environ(), "CGO_ENABLED=0")
	output, exitCode, cmdErr = runCommandWithEnvInDir(root, env, "go", "build", "-trimpath", "-o", "bin/leamas", "./cmd/leamas")
	if exitCode != 0 || cmdErr != nil {
		fmt.Printf(" FAILED\n")
		printFailureOutput(nil, "static build", output, exitCode, cmdErr)
		*failed = true
	} else {
		fmt.Printf(" OK\n")
	}
}
