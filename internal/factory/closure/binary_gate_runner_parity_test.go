// SPDX-License-Identifier: Apache-2.0

// binary_gate_runner_parity_test.go owns the
// TestR6BFakeRunnerMatchesProcessLifecycleContract
// assertion the CORRECTION08 ACT requires (Phase 27).
//
// The fake/production split was a real defect in
// CORRECTION07: the r6BRecordingRunner reported a
// "spawn failure" by setting Err=context.DeadlineExceeded
// while OsRunner reports it via StartErr=nil. CORRECTION08
// aligns the fake with the production lifecycle so future
// matrix rows cannot regress.
//
// The test is package-local (closure) because it asserts
// on the r6BRecordingRunner type, which is a
// package-private test seam.
package closure

import (
	"context"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// TestR6BFakeRunnerMatchesProcessLifecycleContract asserts
// every r6BRecordingRunner failure-mode flag aligns with
// the production OsRunner lifecycle contract:
//
//	spawnFail    -> StartErr != nil, Err == nil
//	timeOut      -> StartErr == nil, TimedOut == true,
//	                Err == nil
//	nonZero      -> StartErr == nil, Err != nil (wait err)
//	stdoutTrunc  -> StartErr == nil
//	stderrTrunc  -> StartErr == nil
//
// The test is the regression guard Phase 27 requires: if a
// future ACT reintroduces the fake/production split, the
// matrix will fail before any production code change
// causes a real-world failure.
func TestR6BFakeRunnerMatchesProcessLifecycleContract(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := "/tmp"
	env := []string(nil)

	t.Run("spawnFail", func(t *testing.T) {
		runner := &r6BRecordingRunner{spawnFail: true}
		got := runner.Run(ctx, "/bin/sh", []string{"-c", "echo"}, dir, env)
		if got.StartErr == nil {
			t.Fatalf("spawnFail: StartErr = nil, want non-nil")
		}
		if got.Err != nil {
			t.Fatalf("spawnFail: Err = %v, want nil", got.Err)
		}
		if got.ExitCode != 127 {
			t.Fatalf("spawnFail: ExitCode = %d, want 127", got.ExitCode)
		}
	})

	t.Run("nonZero", func(t *testing.T) {
		runner := &r6BRecordingRunner{nonZero: true}
		got := runner.Run(ctx, "/bin/sh", []string{"-c", "echo"}, dir, env)
		if got.StartErr != nil {
			t.Fatalf("nonZero: StartErr = %v, want nil", got.StartErr)
		}
		if got.Err == nil {
			t.Fatalf("nonZero: Err = nil, want non-nil wait error")
		}
		if got.ExitCode == 0 {
			t.Fatalf("nonZero: ExitCode = 0, want nonzero")
		}
	})

	t.Run("timeOut", func(t *testing.T) {
		runner := &r6BRecordingRunner{timeOut: true}
		got := runner.Run(ctx, "/bin/sh", []string{"-c", "echo"}, dir, env)
		if got.StartErr != nil {
			t.Fatalf("timeOut: StartErr = %v, want nil", got.StartErr)
		}
		// Phase 10 forbids context.DeadlineExceeded as a
		// fake command-start error. The fake must NOT
		// surface a wait error; it sets TimedOut=true
		// via the StartErr==nil lifecycle.
		if got.Err != nil {
			t.Fatalf("timeOut: Err = %v, want nil (TimedOut is the lifecycle signal)", got.Err)
		}
		if !got.TimedOut {
			t.Fatalf("timeOut: TimedOut = false, want true")
		}
	})

	t.Run("stdoutTrunc", func(t *testing.T) {
		runner := &r6BRecordingRunner{stdoutTrunc: true}
		got := runner.Run(ctx, "/bin/sh", []string{"-c", "echo"}, dir, env)
		if got.StartErr != nil {
			t.Fatalf("stdoutTrunc: StartErr = %v, want nil", got.StartErr)
		}
	})

	t.Run("stderrTrunc", func(t *testing.T) {
		runner := &r6BRecordingRunner{stderrTrunc: true}
		got := runner.Run(ctx, "/bin/sh", []string{"-c", "echo"}, dir, env)
		if got.StartErr != nil {
			t.Fatalf("stderrTrunc: StartErr = %v, want nil", got.StartErr)
		}
	})
}

// Compile-time guard: the r6BRecordingRunner must remain
// a CommandRunner. The fake/production split would be
// detectable at compile time if a future ACT renames
// the type or removes the interface.
var _ evidence.CommandRunner = (*r6BRecordingRunner)(nil)
