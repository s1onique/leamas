// Package exectest provides test helpers that execute external commands.
// This package wraps raw os/exec calls and is intended for use only in
// _test.go files that need to bypass the bounded execution gateway.
package exectest

import (
	"context"
	"os/exec"
)

// Request describes a command to execute.
type Request struct {
	Ctx  context.Context // Context for cancellation (nil = context.Background()).
	Dir  string          // Working directory.
	Env  []string        // Environment variables (nil = inherit current env).
	Name string          // Command name.
	Args []string        // Command arguments.
}

// ExitError represents a command that exited with a non-zero code.
type ExitError struct {
	*exec.ExitError
}

// Unwrap returns the underlying os/exec.ExitError for errors.Is/As support.
func (e *ExitError) Unwrap() error {
	return e.ExitError
}

// CombinedOutput runs the command and returns combined stdout and stderr.
// If ctx is cancelled, the subprocess is terminated.
// If the command fails to spawn, err is non-nil.
// If the command exits with a non-zero code, err is an *ExitError.
func CombinedOutput(req Request) ([]byte, error) {
	if req.Ctx == nil {
		req.Ctx = context.Background()
	}
	cmd := exec.CommandContext(req.Ctx, req.Name, req.Args...)
	cmd.Dir = req.Dir

	if req.Env != nil {
		cmd.Env = req.Env
	}

	output, err := cmd.CombinedOutput()
	if err == nil {
		return output, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return output, &ExitError{ExitError: exitErr}
	}
	return output, err
}

// Output runs the command and returns only stdout.
// If ctx is cancelled, the subprocess is terminated.
// If the command fails to spawn, err is non-nil.
// If the command exits with a non-zero code, err is an *ExitError.
func Output(req Request) ([]byte, error) {
	if req.Ctx == nil {
		req.Ctx = context.Background()
	}
	cmd := exec.CommandContext(req.Ctx, req.Name, req.Args...)
	cmd.Dir = req.Dir

	if req.Env != nil {
		cmd.Env = req.Env
	}

	output, err := cmd.Output()
	if err == nil {
		return output, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return output, &ExitError{ExitError: exitErr}
	}
	return output, err
}
