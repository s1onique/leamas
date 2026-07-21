//go:build unix || darwin || linux

// Package main provides a deterministic test helper for adversarial execution testing.
//
// proc_runtime.go owns the fail-closed child process helpers used by every
// mode whose test contract depends on the child existing for the duration
// of the trigger. Helpers here MUST never silently swallow a cmd.Start
// error: any error is a deterministic test-harness failure.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// errChildStartup is the sentinel wrapping helpers attach to start errors
// so callers can still classify them via errors.Is.
var errChildStartup = errors.New("child process failed to start")

// failClosed prints the diagnostic to stderr and exits non-zero. The helper
// uses this when a precondition that the test depends on is not satisfied.
// Exiting non-zero lets the Go test framework report the helper failure.
func failClosed(context, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+context+": "+format+"\n", args...)
	os.Exit(1)
}

// startChild configures the helper-invoking exec.Cmd but does NOT start it.
// Mode callers MUST inspect the returned error before invoking cmd.Wait().
// The mode argument is prepended to args because the helper binary uses
// its own argv as the mode selector.
func startChild(mode string, args ...string) (*exec.Cmd, error) {
	helperPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("%w: cannot resolve helper path: %v",
			errChildStartup, err)
	}
	allArgs := append([]string{mode}, args...)
	cmd := exec.Command(helperPath, allArgs...)

	// Children must inherit LEAMAS_EXEC_TEST_PID_FILE and READY_DIR so they
	// can record their own PID and publish readiness in lockstep with the
	// parent. Without these the child silently fails to contribute manifest
	// rows and the test proof collapses.
	if manifestFile != "" {
		cmd.Env = append(os.Environ(),
			"LEAMAS_EXEC_TEST_PID_FILE="+manifestFile)
	}
	if readyDir != "" {
		cmd.Env = append(cmd.Env,
			"LEAMAS_EXEC_TEST_READY_DIR="+readyDir)
	}
	return cmd, nil
}

// spawnChildFailClosed starts the child and refuses to continue when Start
// fails. This is the fail-closed variant for every mode whose test contract
// depends on the child existing. Returns the live cmd on success and never
// silently swallows a Start error. The child's Stdout and Stderr remain
// nil so Go connects them to the null device - this is correct for
// commands whose output we do not care about.
func spawnChildFailClosed(context, mode string, args ...string) *exec.Cmd {
	cmd, err := startChild(mode, args...)
	if err != nil {
		failClosed(context, "%v", err)
	}
	if err := cmd.Start(); err != nil {
		failClosed(context, "cmd.Start failed: %v", err)
	}
	return cmd
}

// spawnChildWithInheritedOutputFailClosed behaves like
// spawnChildFailClosed but explicitly wires cmd.Stdout and cmd.Stderr to
// the parent's os.Stdout / os.Stderr so the child inherits the parent's
// file descriptors. Use this for retained-output modes whose test contract
// requires the descendant to actually hold the executor-owned pipe.
//
// Modes whose semantics do NOT require the child to inherit the executor's
// stdout/stderr must continue using spawnChildFailClosed; the explicit
// choice prevents silent descriptor inheritance regressions.
func spawnChildWithInheritedOutputFailClosed(context, mode string, args ...string) *exec.Cmd {
	cmd, err := startChild(mode, args...)
	if err != nil {
		failClosed(context, "%v", err)
	}
	// These assignments must precede Start. Assigning them afterward leaves
	// the already-started child connected to the null device.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		failClosed(context, "cmd.Start failed: %v", err)
	}
	return cmd
}

// errExpectedFailure is used by the helper-internal "negative control"
// probe to surface proof that the adversary halts cleanly without an
// attached child. It is intentionally not exported: it is a private
// mechanism for the CORRECTION05 test-internal failure path.
var errExpectedFailure = errors.New("expected failure path reached")

// waitForFile polls until path exists or the deadline elapses. It is
// used by the retained-pipe fixture so the parent can sequence its
// exit after the child has published the descriptor-ready event.
//
// Returns true when the file is observed, false on deadline expiry.
func waitForFile(path string, deadline time.Duration) bool {
	poll := 5 * time.Millisecond
	end := time.Now().Add(deadline)
	for {
		if _, err := os.Stat(path); err == nil {
			return true
		} else if !os.IsNotExist(err) {
			failClosed("waitForFile",
				"unexpected stat error for %s: %v", path, err)
		}
		if time.Now().After(end) {
			return false
		}
		time.Sleep(poll)
	}
}

// waitChildExpectedSuccess waits for the child to exit with status zero
// and returns normally on success. Any non-zero exit, signal termination,
// or wait error fails closed with a single-line diagnostic.
//
// Use this helper for setup children whose contract is to finish cleanly:
//   - grandchild-spawner that records itself, spawns the grandchild in
//     background, and exits so the test can observe the spawned tree.
//   - spawn-grandchild mode that exits once its tree is recorded.
//
// IMPORTANT: this helper emits NO diagnostic on the success path. Earlier
// versions emitted an 84-byte "child exited cleanly" line that satisfied
// a 64-byte output cap and produced a false-positive overflow proof.
func waitChildExpectedSuccess(context string, cmd *exec.Cmd) {
	err := cmd.Wait()
	if err == nil {
		return
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		ws, ok := exitErr.Sys().(syscall.WaitStatus)
		if ok {
			if ws.Signaled() {
				failClosed(context,
					"expected child was signalled (signal=%d)",
					ws.Signal())
			}
			failClosed(context,
				"expected child exited with status=%d",
				exitErr.ExitCode())
		}
	}
	failClosed(context, "expected child wait failed: %v", err)
}

// waitChildOrFail waits for the child to exit. If the child exited before
// the test could observe it (i.e. before the readiness gate expired) we
// report whether the child exited successfully, non-zero, or was signalled.
// Returning exits the helper with non-zero so the test framework reports the
// problem instead of misleadingly recording a successful run.
//
// Use this helper only for modes whose proof depends on the child staying
// alive past the test trigger (e.g. ignore-sigterm). For modes that expect
// a fast, deterministic child exit (e.g. exit-nonzero-child), the parent
// should propagate the child's exit status via waitChildAndPropagate.
//
// DO NOT use this helper for an intentionally successful setup child.
// Use waitChildExpectedSuccess instead. Treating a successful exit as a
// helper failure produced a false-positive output-overflow proof that
// CORRECTION05 must eliminate.
func waitChildOrFail(context string, cmd *exec.Cmd) {
	err := cmd.Wait()
	if err == nil {
		failClosed(context,
			"child exited cleanly before expected test trigger")
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		ws, ok := exitErr.Sys().(syscall.WaitStatus)
		if ok && ws.Signaled() {
			failClosed(context,
				"child was signalled (signal=%d) before expected test trigger",
				ws.Signal())
		}
		failClosed(context,
			"child exited with status=%d before expected test trigger",
			exitErr.ExitCode())
	}
	failClosed(context, "child wait failed: %v", err)
}

// waitChildAndPropagate waits for the child to exit and re-exits the parent
// with the child's exit status (or the signal that terminated it). This
// preserves the historical exit-status propagation contract for modes such
// as exit-nonzero-child.
func waitChildAndPropagate(cmd *exec.Cmd) {
	err := cmd.Wait()
	if err == nil {
		// Clean child exit: re-emit status 0 so callers observe the same
		// exit code they would have observed if exec had been inline.
		os.Exit(0)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		ws, ok := exitErr.Sys().(syscall.WaitStatus)
		if ok {
			if ws.Signaled() {
				// Propagate the signal so the executor reports the right
				// exit code via cmd.ProcessState.Sys().Signal().
				os.Exit(128 + int(ws.Signal()))
			}
			os.Exit(ws.ExitStatus())
		}
	}
	os.Exit(1)
}
