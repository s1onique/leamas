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
// silently swallows a Start error.
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
