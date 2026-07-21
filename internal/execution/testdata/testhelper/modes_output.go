//go:build unix || darwin || linux

// Package main provides a deterministic test helper for adversarial execution testing.
//
// modes_output.go owns the held-descriptor, output-flood, and exit-code
// modes. The wait-on-child semantics chosen here determine whether the
// executor observes a stable parent/child tree (waitChildExpectedSuccess,
// waitChildOrFail) or a deterministic exit-status propagation
// (waitChildAndPropagate).
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// publishReadyInDir writes a per-process ready sentinel under
// <dir>/<pid>.ready with fsync then close. Used by the retained-pipe
// fixture to publish explicit per-stage ready evidence. Returns the
// sentinel path on success and exits non-zero on any I/O failure.
func publishReadyInDir(role, dir string) string {
	pid := os.Getpid()
	readyPath := filepath.Join(dir, fmt.Sprintf("%d.%s.ready", pid, role))
	f, err := os.OpenFile(readyPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		failClosed(role,
			"failed to open ready sentinel %s: %v", readyPath, err)
	}
	if _, err := f.WriteString(role); err != nil {
		_ = f.Close()
		failClosed(role,
			"failed to write ready sentinel %s: %v", readyPath, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		failClosed(role,
			"failed to sync ready sentinel %s: %v", readyPath, err)
	}
	if err := f.Close(); err != nil {
		failClosed(role,
			"failed to close ready sentinel %s: %v", readyPath, err)
	}
	return readyPath
}

// outputFloodReadyPath returns the canonical path the parent uses to
// publish output-flood-ready evidence. The path is
// <readyDir>/<pid>.output-flood-ready, matching the per-pid sentinel
// pattern. The helper pollutes readyDir with this auxiliary sentinel so
// the verifier can observe the producer reaching the output-producing
// state.
func outputFloodReadyPath() string {
	return filepath.Join(readyDir,
		fmt.Sprintf("%d.output-flood-ready", os.Getpid()))
}

// publishOutputFloodReady writes the per-pid output-flood-ready sentinel
// at the canonical path. The file is fsynced before close so the
// verifier observes the producer having reached the output-producing
// state. Returns the canonical path on success and exits non-zero on
// any I/O failure.
func publishOutputFloodReady(dir string) string {
	readyPath := outputFloodReadyPath()
	f, err := os.Create(readyPath)
	if err != nil {
		failClosed("output-flood-ready",
			"failed to create %s: %v", readyPath, err)
	}
	if _, err := io.WriteString(f, "ready"); err != nil {
		_ = f.Close()
		failClosed("output-flood-ready",
			"failed to write %s: %v", readyPath, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		failClosed("output-flood-ready",
			"failed to sync %s: %v", readyPath, err)
	}
	if err := f.Close(); err != nil {
		failClosed("output-flood-ready",
			"failed to close %s: %v", readyPath, err)
	}
	return readyPath
}

// runHoldStdoutOpen is the legacy cancellation-only mode. It is retained
// so non-corrected-05 callers can still exercise a parent/child tree
// where the parent waits for the child. The CORRECTION05 genuine
// retained-pipe fixture (runHeldDescriptor, mode "held-descriptor")
// replaces this for new retention proofs.
func runHoldStdoutOpen() {
	recordPID("parent", "hold-stdout-open", false)
	cmd := spawnChildFailClosed("hold-stdout-open", "stdout-holder")
	// Surface the child's exit status so the test framework sees it.
	err := cmd.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(ws.ExitStatus())
			}
		}
		failClosed("hold-stdout-open",
			"stdout-holder child wait failed: %v", err)
	}
}

// runStdoutHolder holds stdout/stderr open by sleeping. Used by the
// legacy hold-stdout-open mode.
func runStdoutHolder() {
	recordPID("child", "stdout-holder", false)
	fmt.Println("stdout-holder started")
	sleepForever()
}

// runNegativeOutputProvenFail is the negative control for the descendant
// overflow test. It deliberately exits with a small diagnostic so the
// test can prove that a helper error message does NOT itself satisfy a
// 64-byte output overflow contract.
func runNegativeOutputProvenFail() {
	recordPID("parent", "negative-overflow-fail", false)
	// Print exactly 12 bytes; this is well under the 64-byte overflow
	// cap, so the test can observe the parent exiting cleanly without
	// producing overflow provenance.
	os.Stdout.WriteString("ok-no-flood")
	os.Exit(0)
}

// runOutputForever floods stdout with 4 KiB chunks until SIGKILL arrives.
// The test verifies the executor's bounded output policy catches the
// overflow and triggers cancellation.
func runOutputForever() {
	recordPID("parent", "output-forever", false)
	buf := make([]byte, 4096)
	for i := range buf {
		buf[i] = 'x'
	}
	for i := 0; ; i++ {
		os.Stdout.Write(buf)
		if i > 10000 {
			i = 0
		}
	}
}

// runOutputForeverFast writes one byte at a time so a small output cap
// (e.g. 64 bytes) trips within tens of milliseconds.
func runOutputForeverFast() {
	recordPID("parent", "output-forever-fast", false)
	for {
		fmt.Print("x")
	}
}

// runOutputForeverChild uses waitChildOrFail because the child is
// expected to flood output until the executor kills the entire group.
func runOutputForeverChild() {
	recordPID("child", "output-forever-child", false)
	cmd := spawnChildFailClosed("output-forever-child", "output-forever")
	waitChildOrFail("output-forever-child", cmd)
}

// runOutputForeverFastChild forwards 1-byte-per-write overflow detection
// through a child so the executor must observe a child pid in its tree.
func runOutputForeverFastChild() {
	recordPID("child", "output-forever-fast-child", false)
	cmd := spawnChildFailClosed("output-forever-fast-child",
		"output-forever-fast")
	waitChildOrFail("output-forever-fast-child", cmd)
}

// runOutputForeverGrandchild uses waitChildExpectedSuccess because the
// grandchild-spawner is intentionally supposed to exit cleanly. After
// success, the parent publishes an output-flood-ready sentinel so the
// test can observe the parent reaching the output-producing state, then
// begins the flood.
func runOutputForeverGrandchild() {
	recordPID("parent", "output-forever-grandchild", false)

	// Spawn the grandchild-spawner and expect it to exit cleanly. This
	// is the CORRECTION05 fix: the prior waitChildOrFail emitted an
	// 84-byte diagnostic that itself satisfied the 64-byte output cap.
	cmd := spawnChildFailClosed("output-forever-grandchild",
		"grandchild-spawner")
	waitChildExpectedSuccess("output-forever-grandchild", cmd)

	// Full tree is now established: parent, child, and grandchild are
	// all recorded. Publish output-flood-ready evidence so the test
	// can observe the parent reaching the output-producing state.
	ready := publishOutputFloodReady(readyDir)
	if ready == "" {
		failClosed("output-forever-grandchild",
			"output-flood-ready sentinel could not be published")
	}

	// Begin infinite output so the executor's overflow detection
	// fires.
	for {
		fmt.Print("x")
	}
}

// runExitNonzero records the parent and exits with code 42.
func runExitNonzero() {
	recordPID("parent", "exit-nonzero", false)
	os.Exit(42)
}

// runExitNonzeroChild records itself, spawns an exit-nonzero child, and
// propagates its exit status.
func runExitNonzeroChild() {
	recordPID("child", "exit-nonzero-child", false)
	cmd := spawnChildFailClosed("exit-nonzero-child", "exit-nonzero")
	waitChildAndPropagate(cmd)
}
