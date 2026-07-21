// Package exectest provides test helpers that execute external commands.
package exectest

import (
	"os/exec"
	"time"
)

// Outcome classifies the result of a command execution.
type Outcome int

const (
	OutcomeSuccess Outcome = iota
	OutcomeExitFailure
	OutcomeSpawnFailure
	OutcomeTimeout
	OutcomeCancelled
	OutcomeOutputOverflow
	OutcomeWaitDelay
	OutcomeExecutionError
)

func (o Outcome) String() string {
	switch o {
	case OutcomeSuccess:
		return "success"
	case OutcomeExitFailure:
		return "exit_failure"
	case OutcomeSpawnFailure:
		return "spawn_failure"
	case OutcomeTimeout:
		return "timeout"
	case OutcomeCancelled:
		return "cancelled"
	case OutcomeOutputOverflow:
		return "output_overflow"
	case OutcomeWaitDelay:
		return "wait_delay"
	case OutcomeExecutionError:
		return "execution_error"
	default:
		return "unknown"
	}
}

// OutputLimitExceeded indicates output exceeded the limit.
type OutputLimitExceeded struct {
	Limit    int64
	Observed int64
}

func (e *OutputLimitExceeded) Error() string {
	return "output limit exceeded"
}

// TimeoutExceeded indicates timeout was exceeded.
type TimeoutExceeded struct {
	Timeout time.Duration
}

func (e *TimeoutExceeded) Error() string {
	return "timeout exceeded"
}

// SpawnError indicates the command failed to start.
type SpawnError struct {
	Cause error
}

func (e *SpawnError) Error() string {
	return "command failed to spawn: " + e.Cause.Error()
}

func (e *SpawnError) Unwrap() error {
	return e.Cause
}

// ExitError wraps exec.ExitError for stable API access.
type ExitError struct {
	*exec.ExitError
}

func (e *ExitError) Unwrap() error {
	return e.ExitError
}
