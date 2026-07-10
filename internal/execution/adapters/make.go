// Package adapters provides typed adapters for the execution gateway.
package adapters

import (
	"strconv"
	"strings"

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
	args := a.clampJobs([]string{"-j" + strconv.FormatInt(a.limit, 10), "gate"})
	return &execution.Request{
		Name: "make gate",
		Args: append([]string{"make"}, args...),
		Dir:  dir,
		Env:  env,
	}
}

// FactorizeRequest creates a bounded make factorize request.
func (a *MakeAdapter) FactorizeRequest(dir string, env []string) *execution.Request {
	args := a.clampJobs([]string{"-j" + strconv.FormatInt(a.limit, 10), "factorize"})
	return &execution.Request{
		Name: "make factorize",
		Args: append([]string{"make"}, args...),
		Dir:  dir,
		Env:  env,
	}
}

func (a *MakeAdapter) clampJobs(args []string) []string {
	result := make([]string, len(args))
	copy(result, args)

	for i, arg := range result {
		if strings.HasPrefix(arg, "-j") {
			val := strings.TrimPrefix(arg, "-j")
			if val == "" {
				result[i] = "-j" + strconv.FormatInt(a.limit, 10)
			} else if val != "" {
				if n, err := strconv.ParseInt(val, 10, 64); err == nil {
					if n > a.limit || n == 0 {
						result[i] = "-j" + strconv.FormatInt(a.limit, 10)
					}
				}
			}
		}
	}

	for i := 0; i < len(result)-1; i++ {
		if result[i] == "-j" {
			val := result[i+1]
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				if n > a.limit || n == 0 {
					result[i+1] = strconv.FormatInt(a.limit, 10)
				}
			}
		}
	}

	return result
}

// HasUnboundedJobs checks if args contain unbounded -j flag.
func (a *MakeAdapter) HasUnboundedJobs(args []string) bool {
	for _, arg := range args {
		if arg == "-j" {
			return true
		}
		if strings.HasPrefix(arg, "-j") {
			val := strings.TrimPrefix(arg, "-j")
			if val == "" {
				return true
			}
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				if n == 0 {
					return true
				}
			}
		}
	}
	return false
}
