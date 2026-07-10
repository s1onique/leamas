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
	args = a.clampParallelism(args, "-p")

	req := &execution.Request{
		Name: "go build",
		Args: append([]string{"go", "build"}, args...),
		Dir:  dir,
		Env:  a.AddEnvGOMAXPROCS(nil),
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
		Env:  a.AddEnvGOMAXPROCS(nil),
	}
	if len(packages) > 0 {
		req.Args = append(req.Args, packages...)
	}

	return req
}

// VetRequest creates a bounded go vet request.
func (a *GoAdapter) VetRequest(dir string, packages []string, args ...string) *execution.Request {
	args = a.clampParallelism(args, "-p")

	req := &execution.Request{
		Name: "go vet",
		Args: append([]string{"go", "vet"}, args...),
		Dir:  dir,
		Env:  a.AddEnvGOMAXPROCS(nil),
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
		Env:  a.AddEnvGOMAXPROCS(nil),
	}
}

// ListRequest creates a bounded go list request.
func (a *GoAdapter) ListRequest(dir string, pattern string, args ...string) *execution.Request {
	args = a.clampParallelism(args, "-p")

	req := &execution.Request{
		Name: "go list",
		Args: []string{"go", "list"},
		Dir:  dir,
		Env:  a.AddEnvGOMAXPROCS(nil),
	}
	if len(args) > 0 {
		req.Args = append(req.Args, args...)
	}
	if pattern != "" {
		req.Args = append(req.Args, pattern)
	}

	return req
}

// clampParallelism ensures -p flag is bounded and not duplicated.
func (a *GoAdapter) clampParallelism(args []string, flag string) []string {
	result := make([]string, len(args))
	copy(result, args)

	hasFlag := false

	for i := 0; i < len(result); i++ {
		arg := result[i]

		// Handle -p=value or -p value
		if strings.HasPrefix(arg, flag+"=") {
			if hasFlag {
				// Remove duplicate
				result = append(result[:i], result[i+1:]...)
				i--
			} else {
				hasFlag = true
				val := strings.TrimPrefix(arg, flag+"=")
				if n, err := strconv.ParseInt(val, 10, 64); err == nil {
					if n <= 0 || n > a.limit {
						result[i] = flag + "=" + strconv.FormatInt(a.limit, 10)
					}
				}
			}
		} else if arg == flag {
			if i+1 < len(result) {
				val := result[i+1]
				if hasFlag {
					// Remove this flag and value
					result = append(result[:i], result[i+2:]...)
					i--
				} else {
					hasFlag = true
					if n, err := strconv.ParseInt(val, 10, 64); err == nil {
						if n <= 0 || n > a.limit {
							result[i+1] = strconv.FormatInt(a.limit, 10)
						}
					}
				}
			}
		}
	}

	// If no valid -p found, add one
	if !hasFlag {
		result = append(result, flag, strconv.FormatInt(a.limit, 10))
	}

	return result
}

// clampTestFlags ensures test parallelism is bounded and timeout is present.
func (a *GoAdapter) clampTestFlags(args []string) []string {
	result := make([]string, len(args))
	copy(result, args)

	hasP := false
	hasParallel := false
	hasTimeout := false

	for i := 0; i < len(result); i++ {
		arg := result[i]

		// Handle -p=value
		if strings.HasPrefix(arg, "-p=") {
			if hasP {
				// Remove duplicate
				result = append(result[:i], result[i+1:]...)
				i--
			} else {
				hasP = true
				val := strings.TrimPrefix(arg, "-p=")
				if n, err := strconv.ParseInt(val, 10, 64); err == nil {
					if n <= 0 || n > a.limit {
						result[i] = "-p=" + strconv.FormatInt(a.limit, 10)
					}
				}
			}
		} else if arg == "-p" && i+1 < len(result) {
			if hasP {
				result = append(result[:i], result[i+2:]...)
				i--
			} else {
				hasP = true
				val := result[i+1]
				if n, err := strconv.ParseInt(val, 10, 64); err == nil {
					if n <= 0 || n > a.limit {
						result[i+1] = strconv.FormatInt(a.limit, 10)
					}
				}
			}
		}

		// Handle -parallel=value
		if strings.HasPrefix(arg, "-parallel=") {
			if hasParallel {
				result = append(result[:i], result[i+1:]...)
				i--
			} else {
				hasParallel = true
				val := strings.TrimPrefix(arg, "-parallel=")
				if n, err := strconv.ParseInt(val, 10, 64); err == nil {
					if n <= 0 || n > a.limit {
						result[i] = "-parallel=" + strconv.FormatInt(a.limit, 10)
					}
				}
			}
		} else if arg == "-parallel" && i+1 < len(result) {
			if hasParallel {
				result = append(result[:i], result[i+2:]...)
				i--
			} else {
				hasParallel = true
				val := result[i+1]
				if n, err := strconv.ParseInt(val, 10, 64); err == nil {
					if n <= 0 || n > a.limit {
						result[i+1] = strconv.FormatInt(a.limit, 10)
					}
				}
			}
		}

		// Handle -timeout=value
		if strings.HasPrefix(arg, "-timeout=") {
			hasTimeout = true
			val := strings.TrimPrefix(arg, "-timeout=")
			// Replace zero timeout with default
			if val == "0" {
				result[i] = "-timeout=120s"
			}
		} else if arg == "-timeout" && i+1 < len(result) {
			hasTimeout = true
			// Replace zero timeout with default
			if result[i+1] == "0" {
				result[i+1] = "120s"
			}
		}
	}

	// Add missing -p
	if !hasP {
		result = append(result, "-p", strconv.FormatInt(a.limit, 10))
	}

	// Add missing -parallel
	if !hasParallel {
		result = append(result, "-parallel", strconv.FormatInt(a.limit, 10))
	}

	// Add missing -timeout
	if !hasTimeout {
		result = append(result, "-timeout", "120s")
	}

	return result
}

// ensureGOMAXPROCS adds GOMAXPROCS to environment if not present.
func (a *GoAdapter) ensureGOMAXPROCS(args []string) []string {
	// GOMAXPROCS is added to environment, not args
	// This is a no-op here; the caller should use ClampEnvForTest
	return args
}

// ClampEnvForTest returns environment variables clamped for testing.
func (a *GoAdapter) ClampEnvForTest(env []string) []string {
	result := make([]string, len(env))
	copy(result, env)

	hasGOMAXPROCS := false

	for i, e := range result {
		if strings.HasPrefix(e, "GOMAXPROCS=") {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				hasGOMAXPROCS = true
				if val, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					if val <= 0 || val > a.limit {
						result[i] = "GOMAXPROCS=" + strconv.FormatInt(a.limit, 10)
					}
				}
			}
		}
	}

	if !hasGOMAXPROCS {
		result = append(result, "GOMAXPROCS="+strconv.FormatInt(a.limit, 10))
	}

	return result
}

// AddEnvGOMAXPROCS returns a copy of env with GOMAXPROCS added or clamped.
func (a *GoAdapter) AddEnvGOMAXPROCS(env []string) []string {
	return a.ClampEnvForTest(env)
}
