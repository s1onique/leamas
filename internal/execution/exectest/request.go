package exectest

import (
	"context"
	"errors"
	"os/exec"
)

// Request describes a command to execute.
type Request struct {
	Ctx  context.Context
	Dir  string
	Env  []string
	Name string
	Args []string
}

// CombinedOutput runs the command and returns combined stdout and stderr.
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
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return output, &ExitError{ExitError: exitErr}
	}
	return output, err
}

// Output runs the command and returns only stdout.
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
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return output, &ExitError{ExitError: exitErr}
	}
	return output, err
}
