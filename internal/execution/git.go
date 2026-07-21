// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"context"
	"os/exec"
	"strings"
)

// RunGit runs a git command in the specified directory and returns the output.
// It is a simple, production-friendly wrapper that uses exec.CommandContext.
// For bounded execution with resource limits, use Executor.Execute with GitAdapter.
func RunGit(ctx context.Context, dir string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// RunGitSimple runs a git command without context cancellation support.
func RunGitSimple(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}
