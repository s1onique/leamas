// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"time"
)

// Request represents an execution request.
type Request struct {
	Name        string        // Human-readable name for the command
	Args        []string      // Command arguments (first element is the executable)
	Dir         string        // Working directory
	Env         []string      // Environment variables (appended to inherited env)
	Timeout     time.Duration // Command timeout (must be <= MaxPermittedTimeout)
	OutputCap   int64         // Output capture limit (0 = use budget default)
	Fingerprint string        // Logical fingerprint for cycle detection
}

// Result represents the outcome of an execution.
type Result struct {
	ExitCode        int             // Process exit code
	Duration        time.Duration   // Execution duration
	Stdout          []byte          // Captured stdout (may be truncated)
	Stderr          []byte          // Captured stderr (may be truncated)
	OutputTruncated bool            // True if output was truncated
	Error           *ExecutionError // Execution error, if any
}

// Success returns true if the command completed successfully.
func (r *Result) Success() bool {
	return r.Error == nil && r.ExitCode == 0
}

// Failed returns true if the command failed.
func (r *Result) Failed() bool {
	return r.Error != nil || r.ExitCode != 0
}

// NewResult creates a successful result.
func NewResult(exitCode int, duration time.Duration, stdout, stderr []byte, truncated bool) *Result {
	return &Result{
		ExitCode:        exitCode,
		Duration:        duration,
		Stdout:          stdout,
		Stderr:          stderr,
		OutputTruncated: truncated,
	}
}

// NewErrorResult creates an error result.
func NewErrorResult(err *ExecutionError) *Result {
	return &Result{
		ExitCode: -1,
		Error:    err,
	}
}

// CommandName returns the command name (first argument) or empty string.
func (r *Request) CommandName() string {
	if len(r.Args) == 0 {
		return ""
	}
	return r.Args[0]
}

// CommandLine returns the full command line as a string.
func (r *Request) CommandLine() string {
	if len(r.Args) == 0 {
		return ""
	}
	result := r.Args[0]
	for _, arg := range r.Args[1:] {
		result += " " + arg
	}
	return result
}
