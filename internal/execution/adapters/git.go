// Package adapters provides typed adapters for the execution gateway.
package adapters

import (
	"github.com/s1onique/leamas/internal/execution"
)

// GitAdapter provides bounded execution for Git commands.
type GitAdapter struct {
	executor *execution.Executor
}

// NewGitAdapter creates a new Git adapter.
func NewGitAdapter(executor *execution.Executor) *GitAdapter {
	return &GitAdapter{
		executor: executor,
	}
}

// RunGit creates a bounded git request.
func (a *GitAdapter) RunGit(dir string, args []string) *execution.Request {
	name := "git"
	if len(args) > 0 {
		name += " " + args[0]
	}

	return &execution.Request{
		Name: name,
		Args: append([]string{"git"}, args...),
		Dir:  dir,
	}
}

// Status creates a bounded git status request.
func (a *GitAdapter) Status(dir string) *execution.Request {
	return a.RunGit(dir, []string{"status"})
}

// Diff creates a bounded git diff request.
func (a *GitAdapter) Diff(dir string, args ...string) *execution.Request {
	fullArgs := []string{"diff"}
	fullArgs = append(fullArgs, args...)
	return a.RunGit(dir, fullArgs)
}

// RevParse creates a bounded git rev-parse request.
func (a *GitAdapter) RevParse(dir string, args ...string) *execution.Request {
	fullArgs := []string{"rev-parse"}
	fullArgs = append(fullArgs, args...)
	return a.RunGit(dir, fullArgs)
}

// ShowTopLevel creates a bounded git rev-parse --show-toplevel request.
func (a *GitAdapter) ShowTopLevel(dir string) *execution.Request {
	return a.RevParse(dir, "--show-toplevel")
}

// LsFiles creates a bounded git ls-files request.
func (a *GitAdapter) LsFiles(dir string, args ...string) *execution.Request {
	fullArgs := []string{"ls-files"}
	fullArgs = append(fullArgs, args...)
	return a.RunGit(dir, fullArgs)
}

// IsTracked creates a bounded git ls-files --error-unmatch request.
func (a *GitAdapter) IsTracked(dir, path string) *execution.Request {
	return a.LsFiles(dir, "--error-unmatch", path)
}
