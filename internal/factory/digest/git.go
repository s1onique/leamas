// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunGit executes a git command in the repository root and returns its output.
func RunGit(repoRoot string, args []string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), fmt.Errorf("git %s failed: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return string(output), fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}

// RunGitWithExitCode executes a git command and returns output and exit code.
// Returns exitCode=-1 if command execution failed (not just non-zero exit).
func RunGitWithExitCode(repoRoot string, args []string) (string, int) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), exitErr.ExitCode()
		}
		// Command execution failed entirely
		return string(output), -1
	}
	return string(output), 0
}

// DetectRepoRoot finds the root of the current Git repository.
func DetectRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to detect git repo root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// IsTracked checks if a file is tracked by git.
func IsTracked(repoRoot, path string) bool {
	_, err := RunGit(repoRoot, []string{"ls-files", "--error-unmatch", path})
	return err == nil
}
