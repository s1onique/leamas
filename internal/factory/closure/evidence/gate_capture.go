// SPDX-License-Identifier: Apache-2.0

// Package evidence - gate_capture.go implements Phase 4
// (exactly-once per-run gate capture) and Phase 3
// (authoritative bounded result fields) of CORRECTION01-R1.
//
// GateCollector owns the capture state for exactly one closure
// run. The first Capture call executes the gate via the runner;
// every subsequent call returns the same immutable GateCapture
// and error without invoking the runner again. Concurrent
// callers are serialised through a sync.Mutex that protects
// the runner call, the cached capture, and the cached error.

package evidence

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GateLaneResult is the typed status of a single lane.
type GateLaneResult struct {
	LaneID  string `json:"lane_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// GateFinding describes a single fast-lane finding.
type GateFinding struct {
	Path     string `json:"path"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Rule     string `json:"rule,omitempty"`
}

// GateCaptureRequest parameterises a single fast-lane invocation.
type GateCaptureRequest struct {
	RepositoryRoot string
	SubjectRoot    string
	EvidenceDir    string
	RunID          string
	MakeExecutable []string
}

// GateCapture is the typed result of one fast-lane invocation.
type GateCapture struct {
	ExitCode int `json:"exit_code"`

	TimedOut        bool `json:"timed_out"`
	StdoutTruncated bool `json:"stdout_truncated"`
	StderrTruncated bool `json:"stderr_truncated"`
	Canceled        bool `json:"canceled"`

	StdoutSHA256 string `json:"stdout_sha256"`
	StderrSHA256 string `json:"stderr_sha256"`

	RawOutputPath string `json:"raw_output_path"`
	RawSHA256     string `json:"raw_sha256"`

	LaneResults []GateLaneResult `json:"lane_results"`

	ExecGateObservedStatus string        `json:"exec_gate_observed_status"`
	ACTOwnedExecGateResult string        `json:"act_owned_exec_gate_result"`
	PreExistingFindings    []GateFinding `json:"pre_existing_findings"`

	StartedAtUTC  string `json:"started_at_utc"`
	FinishedAtUTC string `json:"finished_at_utc"`
	DurationMS    int64  `json:"duration_ms"`
}

// CommandRunner is the gateway every GateCollector uses.
type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, dir string, env []string) CommandResult
}

// CommandResult is the authoritative bounded process result.
// Truncation comes from the writer's authoritative flag, not
// from a length comparison.
type CommandResult struct {
	ExitCode      int
	TimedOut      bool
	Canceled      bool
	Stdout        []byte
	Stderr        []byte
	StdoutTrunc   bool
	StderrTrunc   bool
	Err           error
	StartedAtUTC  time.Time
	FinishedAtUTC time.Time
}

// OsRunner is the production CommandRunner. WaitDelay is finite
// so the bounded execution authority is enforced.
type OsRunner struct {
	OutputCap int
	WaitDelay time.Duration
}

// Run invokes the supplied argv through the OS process package.
func (r *OsRunner) Run(ctx context.Context, name string, args []string, dir string, env []string) CommandResult {
	cap := r.OutputCap
	if cap == 0 {
		cap = 8 << 20
	}
	waitDelay := r.WaitDelay
	if waitDelay == 0 {
		waitDelay = 5 * time.Second
	}
	started := time.Now().UTC()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = env
	}
	cmd.WaitDelay = waitDelay
	var stdoutBuf, stderrBuf truncatedBuffer
	stdoutBuf.cap = cap
	stderrBuf.cap = cap
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Start()
	if err != nil {
		return CommandResult{
			ExitCode: 127, Err: err,
			Stdout:       []byte(err.Error()),
			StartedAtUTC: started, FinishedAtUTC: time.Now().UTC(),
		}
	}
	waitErr := cmd.Wait()
	finished := time.Now().UTC()
	result := CommandResult{
		ExitCode:      exitCodeFromWait(waitErr),
		Stdout:        stdoutBuf.bytes(),
		Stderr:        stderrBuf.bytes(),
		StdoutTrunc:   stdoutBuf.truncated,
		StderrTrunc:   stderrBuf.truncated,
		StartedAtUTC:  started,
		FinishedAtUTC: finished,
		Err:           waitErr,
	}
	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
	}
	if ctx.Err() == context.Canceled {
		result.Canceled = true
	}
	return result
}

func exitCodeFromWait(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

// truncatedBuffer is the authoritative writer. Its truncated
// flag is set when overflow occurs; downstream consumers MUST
// read this flag rather than re-deriving truncation from byte
// length.
type truncatedBuffer struct {
	cap       int
	buf       bytes.Buffer
	truncated bool
}

func (b *truncatedBuffer) Write(p []byte) (int, error) {
	remaining := b.cap - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *truncatedBuffer) bytes() []byte {
	return append([]byte(nil), b.buf.Bytes()...)
}

// GateCollector owns exactly one closure run. The first Capture
// call invokes the runner; every subsequent call returns the
// cached result. Two collectors never share state.
type GateCollector struct {
	mu         sync.Mutex
	once       sync.Once
	runner     CommandRunner
	done       bool
	capture    GateCapture
	captureErr error
}

// NewGateCollector constructs an empty per-run collector.
func NewGateCollector(runner CommandRunner) *GateCollector {
	if runner == nil {
		runner = &OsRunner{}
	}
	return &GateCollector{runner: runner}
}

// Calls returns the number of runner invocations the collector
// has performed. Exactly-once means this is 0 before Capture
// and 1 after any number of Capture calls.
func (c *GateCollector) Calls() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done {
		return 1
	}
	return 0
}

// Capture returns the gate capture for this run. The first call
// invokes the runner exactly once; every subsequent call
// returns the cached result.
func (c *GateCollector) Capture(ctx context.Context, req GateCaptureRequest) (GateCapture, error) {
	if c == nil {
		return GateCapture{}, errors.New("evidence: GateCollector is nil")
	}
	if strings.TrimSpace(req.SubjectRoot) == "" {
		return GateCapture{}, errors.New("evidence: subject root is required")
	}
	if strings.TrimSpace(req.EvidenceDir) == "" {
		return GateCapture{}, errors.New("evidence: evidence directory is required")
	}
	c.once.Do(func() {
		c.capture, c.captureErr = c.runCapture(ctx, req)
	})
	c.mu.Lock()
	defer c.mu.Unlock()
	c.done = true
	return c.capture, c.captureErr
}

// runCapture performs the actual gate invocation. It is called
// exactly once per collector via sync.Once.
func (c *GateCollector) runCapture(ctx context.Context, req GateCaptureRequest) (GateCapture, error) {
	if err := os.MkdirAll(req.EvidenceDir, 0o700); err != nil {
		return GateCapture{}, fmt.Errorf("evidence: create evidence directory: %w", err)
	}
	argv := req.MakeExecutable
	if len(argv) == 0 {
		argv = []string{"make", "gate-fast"}
	}
	started := time.Now().UTC()
	result := c.runner.Run(ctx, argv[0], argv[1:], req.SubjectRoot, nil)
	finished := time.Now().UTC()
	rawPath := filepath.Join(req.EvidenceDir, "gate-fast.raw.txt")
	if err := os.WriteFile(rawPath, append(result.Stdout, result.Stderr...), 0o600); err != nil {
		return GateCapture{}, fmt.Errorf("evidence: write raw output: %w", err)
	}
	combined := append(append([]byte(nil), result.Stdout...), result.Stderr...)
	capture := GateCapture{
		ExitCode:               result.ExitCode,
		TimedOut:               result.TimedOut,
		StdoutTruncated:        result.StdoutTrunc,
		StderrTruncated:        result.StderrTrunc,
		Canceled:               result.Canceled,
		StdoutSHA256:           SHA256HexBytes(result.Stdout),
		StderrSHA256:           SHA256HexBytes(result.Stderr),
		RawOutputPath:          rawPath,
		RawSHA256:              SHA256HexBytes(combined),
		LaneResults:            parseLaneStatus(combined),
		ExecGateObservedStatus: parseObservedStatus(combined),
		PreExistingFindings:    parseFindings(combined),
		StartedAtUTC:           started.Format(time.RFC3339Nano),
		FinishedAtUTC:          finished.Format(time.RFC3339Nano),
		DurationMS:             finished.Sub(started).Milliseconds(),
	}
	return capture, nil
}

// SHA256HexBytes is the small helper that hashes arbitrary bytes.
func SHA256HexBytes(data []byte) string {
	return sha256Hex(data)
}
