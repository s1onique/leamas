// SPDX-License-Identifier: Apache-2.0

// Package evidence - gate_capture.go implements Phase 4 (per-run
// gate capture) of CORRECTION01. GateCollector owns the capture
// state for a single ExecuteClose invocation; there is no
// process-global cache.

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

// CommandResult is the typed outcome of a single process
// invocation.
type CommandResult struct {
	ExitCode      int
	TimedOut      bool
	Stdout        []byte
	Stderr        []byte
	StartedAtUTC  time.Time
	FinishedAtUTC time.Time
}

// OsRunner is the production CommandRunner.
type OsRunner struct {
	OutputCap int
}

// Run invokes the supplied argv through the OS process package.
func (r *OsRunner) Run(ctx context.Context, name string, args []string, dir string, env []string) CommandResult {
	cap := r.OutputCap
	if cap == 0 {
		cap = 8 << 20
	}
	started := time.Now().UTC()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = env
	}
	var stdoutBuf, stderrBuf truncatedBuffer
	stdoutBuf.cap = cap
	stderrBuf.cap = cap
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Start()
	if err != nil {
		return CommandResult{ExitCode: 127, Stderr: []byte(err.Error()), StartedAtUTC: started, FinishedAtUTC: time.Now().UTC()}
	}
	waitErr := cmd.Wait()
	finished := time.Now().UTC()
	result := CommandResult{
		ExitCode:      exitCodeFromWait(waitErr),
		Stdout:        stdoutBuf.bytes(),
		Stderr:        stderrBuf.bytes(),
		StartedAtUTC:  started,
		FinishedAtUTC: finished,
	}
	if waitErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
		}
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

// truncatedBuffer caps the captured bytes and records overflow.
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

// GateCollector is the per-run owner of gate capture state.
// A new collector MUST be constructed for every ExecuteClose
// invocation so two independent runs do not share data.
type GateCollector struct {
	runner CommandRunner
	calls  int
}

// NewGateCollector constructs an empty per-run collector.
func NewGateCollector(runner CommandRunner) *GateCollector {
	if runner == nil {
		runner = &OsRunner{}
	}
	return &GateCollector{runner: runner}
}

// Calls returns the number of gate processes invoked so far.
func (c *GateCollector) Calls() int {
	if c == nil {
		return 0
	}
	return c.calls
}

// Capture runs the fast lane once via the collector's runner
// and records the typed result. Two collectors never share
// state.
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
	c.calls++
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
		StdoutTruncated:        stdoutOverflowed(result.Stdout),
		StderrTruncated:        stderrOverflowed(result.Stderr),
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

// stdoutOverflowed reports whether the supplied bytes were
// truncated by the writer.
func stdoutOverflowed(stdout []byte) bool {
	return len(stdout) >= 8<<20
}

// stderrOverflowed reports whether the supplied bytes were
// truncated by the writer.
func stderrOverflowed(stderr []byte) bool {
	return len(stderr) >= 8<<20
}

// SHA256HexBytes is the small helper that hashes arbitrary bytes.
func SHA256HexBytes(data []byte) string {
	return sha256Hex(data)
}
