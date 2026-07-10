// Package adapters provides typed adapters for the execution gateway.
package adapters

import (
	"strconv"

	"github.com/s1onique/leamas/internal/execution"
)

// MakeAdapter provides bounded execution for Make commands.
type MakeAdapter struct {
	executor *execution.Executor
	limit    int64
}

// NewMakeAdapter creates a new Make adapter.
func NewMakeAdapter(executor *execution.Executor) *MakeAdapter {
	return &MakeAdapter{
		executor: executor,
		limit:    executor.Budget().MaxConcurrent,
	}
}

// Request creates a bounded make request.
func (a *MakeAdapter) Request(dir string, target string, args ...string) *execution.Request {
	args = a.clampJobs(args)

	req := &execution.Request{
		Name: "make " + target,
		Args: []string{"make"},
		Dir:  dir,
		Env:  a.sanitizeMakeFlags(nil),
	}

	if len(args) > 0 {
		req.Args = append(req.Args, args...)
	}

	if target != "" {
		req.Args = append(req.Args, target)
	}

	return req
}

// GateRequest creates a bounded make gate request.
func (a *MakeAdapter) GateRequest(dir string, env []string) *execution.Request {
	args := []string{"-j" + strconv.FormatInt(a.limit, 10), "gate"}
	return &execution.Request{
		Name: "make gate",
		Args: append([]string{"make"}, args...),
		Dir:  dir,
		Env:  a.sanitizeMakeFlags(env),
	}
}

// FactorizeRequest creates a bounded make factorize request.
func (a *MakeAdapter) FactorizeRequest(dir string, env []string) *execution.Request {
	args := []string{"-j" + strconv.FormatInt(a.limit, 10), "factorize"}
	return &execution.Request{
		Name: "make factorize",
		Args: append([]string{"make"}, args...),
		Dir:  dir,
		Env:  a.sanitizeMakeFlags(env),
	}
}
