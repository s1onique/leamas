// Package adapters provides typed adapters for the execution gateway.
package adapters

import (
	"strconv"
	"strings"

	"github.com/s1onique/leamas/internal/execution"
)

// GoAdapter provides bounded execution for Go toolchain commands.
type GoAdapter struct {
	executor *execution.Executor
	limit    int64
}

// NewGoAdapter creates a new Go adapter.
func NewGoAdapter(executor *execution.Executor) *GoAdapter {
	return &GoAdapter{
		executor: executor,
		limit:    executor.Budget().MaxConcurrent,
	}
}

// BuildRequest creates a bounded go build request.
func (a *GoAdapter) BuildRequest(dir string, output string, packages []string, args ...string) *execution.Request {
	args = a.clampParallelism(args)

	req := &execution.Request{
		Name: "go build",
		Args: append([]string{"go", "build"}, args...),
		Dir:  dir,
	}

	if output != "" {
		req.Args = append(req.Args, "-o", output)
	}
	if len(packages) > 0 {
		req.Args = append(req.Args, packages...)
	}

	return req
}

// TestRequest creates a bounded go test request.
func (a *GoAdapter) TestRequest(dir string, packages []string, args ...string) *execution.Request {
	args = a.clampTestFlags(args)

	req := &execution.Request{
		Name: "go test",
		Args: append([]string{"go", "test"}, args...),
		Dir:  dir,
	}
	if len(packages) > 0 {
		req.Args = append(req.Args, packages...)
	}

	return req
}

// VetRequest creates a bounded go vet request.
func (a *GoAdapter) VetRequest(dir string, packages []string, args ...string) *execution.Request {
	args = a.clampParallelism(args)

	req := &execution.Request{
		Name: "go vet",
		Args: append([]string{"go", "vet"}, args...),
		Dir:  dir,
	}
	if len(packages) > 0 {
		req.Args = append(req.Args, packages...)
	}

	return req
}

// ModTidyRequest creates a go mod tidy request.
func (a *GoAdapter) ModTidyRequest(dir string) *execution.Request {
	return &execution.Request{
		Name: "go mod tidy",
		Args: []string{"go", "mod", "tidy"},
		Dir:  dir,
	}
}

// ListRequest creates a bounded go list request.
func (a *GoAdapter) ListRequest(dir string, pattern string, args ...string) *execution.Request {
	args = a.clampParallelism(args)

	req := &execution.Request{
		Name: "go list",
		Args: []string{"go", "list"},
		Dir:  dir,
	}
	if len(args) > 0 {
		req.Args = append(req.Args, args...)
	}
	if pattern != "" {
		req.Args = append(req.Args, pattern)
	}

	return req
}

func (a *GoAdapter) clampParallelism(args []string) []string {
	result := make([]string, len(args))
	copy(result, args)

	for i, arg := range result {
		if arg == "-p" && i+1 < len(result) {
			if p, err := strconv.ParseInt(result[i+1], 10, 64); err == nil {
				if p > a.limit {
					result[i+1] = strconv.FormatInt(a.limit, 10)
				}
			}
		}
	}

	return result
}

func (a *GoAdapter) clampTestFlags(args []string) []string {
	result := make([]string, len(args))
	copy(result, args)

	hasTimeout := false

	for i, arg := range result {
		switch arg {
		case "-p":
			if i+1 < len(result) {
				if p, err := strconv.ParseInt(result[i+1], 10, 64); err == nil {
					if p > a.limit {
						result[i+1] = strconv.FormatInt(a.limit, 10)
					}
				}
			}
		case "-parallel":
			if i+1 < len(result) {
				if p, err := strconv.ParseInt(result[i+1], 10, 64); err == nil {
					if p > a.limit {
						result[i+1] = strconv.FormatInt(a.limit, 10)
					}
				}
			}
		case "-timeout":
			hasTimeout = true
		}
	}

	if !hasTimeout {
		result = append(result, "-timeout", "120s")
	}

	return result
}

// ClampEnvForTest returns environment variables clamped for testing.
func (a *GoAdapter) ClampEnvForTest(env []string) []string {
	result := env

	for i, e := range result {
		if strings.HasPrefix(e, "GOMAXPROCS=") {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				if val, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					if val > a.limit {
						result[i] = "GOMAXPROCS=" + strconv.FormatInt(a.limit, 10)
					}
				}
			}
		}
	}

	hasGOMAXPROCS := false
	for _, e := range result {
		if strings.HasPrefix(e, "GOMAXPROCS=") {
			hasGOMAXPROCS = true
			break
		}
	}
	if !hasGOMAXPROCS {
		result = append(result, "GOMAXPROCS="+strconv.FormatInt(a.limit, 10))
	}

	return result
}
