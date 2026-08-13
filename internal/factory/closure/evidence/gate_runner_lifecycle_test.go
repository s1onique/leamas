// SPDX-License-Identifier: Apache-2.0

// gate_runner_lifecycle_test.go owns the real OsRunner
// start/wait matrix the CORRECTION08 ACT requires.
//
// Every row exercises the production OsRunner (no fake
// CommandRunner). The matrix proves the lifecycle split:
//
//   - StartErr is the authoritative command-start signal
//   - Err is the post-start wait outcome (nonzero exit,
//     timeout, cancellation, ErrWaitDelay)
//   - TimedOut / Canceled reflect the bounded context
//     state observed by OsRunner
//
// The GateCollector behaviour is exercised in
// gate_collector_lifecycle_test.go. The split keeps each
// file focused and under the LLM-friendly 400-line
// threshold.
package evidence

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

// TestGateOsRunnerStartFailureContract exercises the
// production OsRunner with a guaranteed spawn failure
// (non-existent binary path) and asserts the lifecycle
// fields Phase 2 of CORRECTION08 freezes:
//
//   - StartErr != nil (process never reached cmd.Wait)
//   - Err == nil (no wait outcome)
//   - ExitCode == 127
//   - TimedOut == false
//   - Canceled == false
//   - Stdout empty
//   - Stderr contains a bounded diagnostic
//   - StdoutTrunc == false, StderrTrunc == false
//
// The test is the canonical contract for "command-start
// failure" — GateCollector reads StartErr as the
// observation-failure signal.
func TestGateOsRunnerStartFailureContract(t *testing.T) {
	runner := &OsRunner{}
	ctx := context.Background()
	bogusName := "/this/path/does/not/exist/leamas-fake-binary"
	result := runner.Run(ctx, bogusName, []string{"factory", "gate", "--lane=fast"}, "/tmp", nil)
	if result.StartErr == nil {
		t.Fatalf("StartErr = nil, want non-nil for spawn failure")
	}
	if result.Err != nil {
		t.Fatalf("Err = %v, want nil (StartErr is the canonical signal)", result.Err)
	}
	if result.ExitCode != 127 {
		t.Fatalf("ExitCode = %d, want 127", result.ExitCode)
	}
	if result.TimedOut {
		t.Fatalf("TimedOut = true, want false")
	}
	if result.Canceled {
		t.Fatalf("Canceled = true, want false")
	}
	if len(result.Stdout) != 0 {
		t.Fatalf("Stdout = %q, want empty", string(result.Stdout))
	}
	if len(result.Stderr) == 0 {
		t.Fatalf("Stderr empty, want bounded diagnostic")
	}
	if result.StdoutTrunc {
		t.Fatalf("StdoutTrunc = true, want false")
	}
	if result.StderrTrunc {
		t.Fatalf("StderrTrunc = true, want false")
	}
}

// TestGateOsRunnerStartWaitContract exercises every
// post-start lifecycle outcome the production OsRunner
// surfaces. The matrix is the source of truth for the
// Phase 24 contract:
//
//	start_failure     -> StartErr != nil, Err == nil
//	success           -> StartErr == nil, Err == nil,
//	                    ExitCode == 0
//	nonzero_exit      -> StartErr == nil, Err != nil
//	                    (exec.ExitError-shaped wait err),
//	                    ExitCode != 0
//	timeout           -> StartErr == nil, TimedOut == true
//	cancellation      -> StartErr == nil, Canceled == true
//	retained_pipe_waitdelay
//	                  -> StartErr == nil,
//	                     errors.Is(Err, exec.ErrWaitDelay)
//
// No string-only classification. Every assertion uses
// the typed lifecycle fields.
func TestGateOsRunnerStartWaitContract(t *testing.T) {
	t.Parallel()
	rows := []struct {
		name   string
		invoke func(t *testing.T, ctx context.Context) CommandResult
		check  func(t *testing.T, got CommandResult)
	}{
		{
			name: "start_failure",
			invoke: func(t *testing.T, ctx context.Context) CommandResult {
				runner := &OsRunner{}
				return runner.Run(ctx,
					"/this/path/does/not/exist/leamas-fake-binary",
					[]string{"factory", "gate", "--lane=fast"},
					"/tmp", nil)
			},
			check: func(t *testing.T, got CommandResult) {
				if got.StartErr == nil {
					t.Fatalf("StartErr nil, want non-nil")
				}
				if got.Err != nil {
					t.Fatalf("Err = %v, want nil", got.Err)
				}
				if got.ExitCode != 127 {
					t.Fatalf("ExitCode = %d, want 127", got.ExitCode)
				}
				if got.TimedOut || got.Canceled {
					t.Fatalf("TimedOut=%v Canceled=%v, want both false",
						got.TimedOut, got.Canceled)
				}
			},
		},
		{
			name: "success",
			invoke: func(t *testing.T, ctx context.Context) CommandResult {
				runner := &OsRunner{}
				// CORRECTION07: use LookPath to find the utility
				// from PATH, avoiding hardcoded /bin assumptions.
				truePath, err := exec.LookPath("true")
				if err != nil {
					t.Skipf("true not in PATH: %v", err)
				}
				return runner.Run(ctx, truePath,
					[]string{}, "/tmp", nil)
			},
			check: func(t *testing.T, got CommandResult) {
				if got.StartErr != nil {
					t.Fatalf("StartErr = %v, want nil", got.StartErr)
				}
				if got.Err != nil {
					t.Fatalf("Err = %v, want nil", got.Err)
				}
				if got.ExitCode != 0 {
					t.Fatalf("ExitCode = %d, want 0", got.ExitCode)
				}
				if got.TimedOut || got.Canceled {
					t.Fatalf("TimedOut=%v Canceled=%v, want both false",
						got.TimedOut, got.Canceled)
				}
			},
		},
		{
			name: "nonzero_exit",
			invoke: func(t *testing.T, ctx context.Context) CommandResult {
				runner := &OsRunner{}
				// CORRECTION07: use LookPath to find the utility
				// from PATH, avoiding hardcoded /bin assumptions.
				falsePath, err := exec.LookPath("false")
				if err != nil {
					t.Skipf("false not in PATH: %v", err)
				}
				return runner.Run(ctx, falsePath,
					[]string{}, "/tmp", nil)
			},
			check: func(t *testing.T, got CommandResult) {
				if got.StartErr != nil {
					t.Fatalf("StartErr = %v, want nil", got.StartErr)
				}
				if got.Err == nil {
					t.Fatalf("Err = nil, want non-nil wait error")
				}
				if got.ExitCode == 0 {
					t.Fatalf("ExitCode = 0, want nonzero")
				}
				// exec.ExitError is the canonical wait
				// error shape for a normal nonzero exit.
				var exitErr *exec.ExitError
				if !errors.As(got.Err, &exitErr) {
					t.Fatalf("Err = %v, want *exec.ExitError", got.Err)
				}
			},
		},
		{
			name: "timeout",
			invoke: func(t *testing.T, ctx context.Context) CommandResult {
				// Bounded context. The child sleeps
				// longer than the deadline so cmd.Wait
				// returns after the ctx fires. We
				// construct the context with a tight
				// deadline to keep the test fast.
				ctx, cancel := context.WithTimeout(ctx, 25*time.Millisecond)
				defer cancel()
				runner := &OsRunner{}
				// CORRECTION07: use LookPath to find the utility
				// from PATH, avoiding hardcoded /bin assumptions.
				sleepPath, err := exec.LookPath("sleep")
				if err != nil {
					t.Skipf("sleep not in PATH: %v", err)
				}
				return runner.Run(ctx, sleepPath,
					[]string{"5"}, "/tmp", nil)
			},
			check: func(t *testing.T, got CommandResult) {
				if got.StartErr != nil {
					t.Fatalf("StartErr = %v, want nil", got.StartErr)
				}
				if !got.TimedOut {
					t.Fatalf("TimedOut = false, want true")
				}
				if got.Canceled {
					t.Fatalf("Canceled = true, want false")
				}
			},
		},
		{
			name: "cancellation",
			invoke: func(t *testing.T, ctx context.Context) CommandResult {
				// The child process MUST reach cmd.Start()
				// (so StartErr == nil) and only THEN be
				// cancelled by the bounded context. We
				// schedule the cancellation ~25ms after
				// start; the child sleeps for 5s, so
				// the bound ctx is the proximate cause
				// of process exit. The lifecycle marker
				// is Canceled=true.
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				go func() {
					time.Sleep(25 * time.Millisecond)
					cancel()
				}()
				runner := &OsRunner{}
				// CORRECTION07: use LookPath to find the utility
				// from PATH, avoiding hardcoded /bin assumptions.
				sleepPath, err := exec.LookPath("sleep")
				if err != nil {
					t.Skipf("sleep not in PATH: %v", err)
				}
				return runner.Run(ctx, sleepPath,
					[]string{"5"}, "/tmp", nil)
			},
			check: func(t *testing.T, got CommandResult) {
				if got.StartErr != nil {
					t.Fatalf("StartErr = %v, want nil", got.StartErr)
				}
				if !got.Canceled {
					t.Fatalf("Canceled = false, want true")
				}
				if got.TimedOut {
					t.Fatalf("TimedOut = true, want false")
				}
			},
		},
		{
			name: "retained_pipe_waitdelay",
			invoke: func(t *testing.T, ctx context.Context) CommandResult {
				// The retained-pipe pattern: a parent
				// process exits while a descendant
				// holds one of its output pipes.
				//
				// CORRECTION09: use the proven
				// retained-pipe fixture from
				// TestClosureWaitDelayRetainedPipe so
				// the wait deterministically surfaces
				// exec.ErrWaitDelay on supported CI
				// platforms. The descendant
				// (/bin/sh -c 'sleep 30') inherits the
				// parent's stdout/stderr pipes and
				// outlives the parent's exit 0 by
				// far more than the WaitDelay envelope.
				// The parent starts successfully,
				// exits successfully, and the wait
				// reports a delay outcome via
				// exec.ErrWaitDelay.
				runner := &OsRunner{WaitDelay: 200 * time.Millisecond}
				// CORRECTION07: use LookPath to find the utility
				// from PATH, avoiding hardcoded /bin assumptions.
				shPath, err := exec.LookPath("sh")
				if err != nil {
					t.Skipf("sh not in PATH: %v", err)
				}
				sleepPath, err := exec.LookPath("sleep")
				if err != nil {
					t.Skipf("sleep not in PATH: %v", err)
				}
				return runner.Run(ctx, shPath,
					[]string{"-c",
						shPath + " -c '" + sleepPath + " 30' & exit 0"},
					"/tmp", nil)
			},
			check: func(t *testing.T, got CommandResult) {
				// CORRECTION09 frozen contract for the
				// dedicated OsRunner retained-pipe test:
				//
				//   StartErr == nil
				//   Err     != nil
				//   errors.Is(Err, exec.ErrWaitDelay)
				//
				// The previous implementation accepted a
				// substring "wait" as an alternative
				// authority, which is forbidden: the
				// typed sentinel is the only contract.
				// The previous check also dereferenced
				// got.Err before checking it was non-nil,
				// which is a nil-deref crash on a clean
				// wait (Err == nil). The new contract
				// checks got.Err != nil FIRST so the
				// typed sentinel assertion is reached
				// only when Err is guaranteed non-nil.
				if got.StartErr != nil {
					t.Fatalf("StartErr = %v, want nil", got.StartErr)
				}
				if got.Err == nil {
					t.Fatalf("Err = nil, want wait-delay outcome (retained pipe fixture)")
				}
				if !errors.Is(got.Err, exec.ErrWaitDelay) {
					t.Fatalf("Err = %v, want errors.Is(exec.ErrWaitDelay)", got.Err)
				}
				// The key invariant: StartErr == nil.
				// WaitDelay is a WAIT outcome, NOT a
				// start failure. This is the property
				// the test exists to prove.
			},
		},
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			got := row.invoke(t, context.Background())
			row.check(t, got)
		})
	}
}
