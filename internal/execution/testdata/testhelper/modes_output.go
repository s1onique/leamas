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
	"time"
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

// runHeldDescriptor implements the natural-exit retained-pipe fixture
// required by ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-
// CORRECTION06. The fixture is deliberately self-contained and
// schedules the natural exit without any test-driven cancel or
// bounded sleep:
//
//  1. parent records itself
//  2. parent spawns the descriptor-holder child with INHERITED stdout
//     and stderr so the executor-owned pipe reaches the descendant
//  3. parent blocks on the child's PID-bound descriptor-ready
//     sentinel, which carries role/pid/pgid metadata
//  4. parent publishes parent-exit-imminent (NOT parent-exited)
//     sentinel so the test can sequence the handoff but cannot
//     mistake the imminent event for actual exit evidence
//  5. parent exits successfully with status zero via os.Exit(0)
//  6. descriptor-holder child retains the inherited descriptors for
//     at least 60 s, far longer than the executor's WaitDelay
//     (TerminationGrace + PostKillWait = 1 s)
//
// The fixture MUST NOT sleep before exiting. The CORRECTION05 fixture
// introduced a 500 ms bounded sleep before os.Exit(0) which made the
// adjacent cancellation test exercise SIGTERM, not natural-exit
// WaitDelay. The CORRECTION06 fixture relies only on the natural exit
// so the executor's WaitDelay cleanup is the observable bound.
func runHeldDescriptor() {
	recordPID("parent", "held-descriptor", false)

	// Spawn the descriptor-holder child with explicit stdout/stderr
	// inheritance so the executor-owned pipe reaches the descendant.
	cmd := spawnChildWithInheritedOutputFailClosed(
		"held-descriptor", "held-descriptor-child")
	_ = cmd

	// Block on the child's descriptor-ready sentinel. The poll is
	// bounded so a stuck child cannot hang the proof; the sentinel
	// is fsynced by the child before this loop returns so the test
	// is observing a real transition.
	readyPath := filepath.Join(readyDir,
		fmt.Sprintf("descriptor-ready.wait"))
	if !waitForFile(readyPath, 10*time.Second) {
		failClosed("held-descriptor",
			"descriptor-ready sentinel not observed within 10s")
	}

	// Publish parent-exit-imminent (not parent-exited). The truthful
	// meaning is "the parent is about to call os.Exit(0)" — the
	// test MUST verify actual exit via the OS-backed parent PID
	// check, not via this sentinel.
	imminentPath := filepath.Join(readyDir,
		fmt.Sprintf("parent-exit-imminent.%d", os.Getpid()))
	f, err := os.Create(imminentPath)
	if err != nil {
		failClosed("held-descriptor",
			"failed to publish parent-exit-imminent sentinel: %v", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		failClosed("held-descriptor",
			"failed to sync parent-exit-imminent sentinel: %v", err)
	}
	_ = f.Close()

	// Exit successfully through the natural code path. The descriptor-
	// holder child still holds the inherited executor-owned stdout and
	// stderr pipes; the executor's cmd.Wait must therefore block
	// until WaitDelay releases the pipe.
	os.Exit(0)
}

// runHeldDescriptorChild inherits the parent's stdout/stderr and holds
// the inherited descriptors open for a bounded duration. It publishes
// two correlated evidence artifacts:
//
//  1. `<child-pid>.descriptor-ready.ready` — the canonical PID-bound
//     readiness sentinel. The CONTENTS bind role, pid, ppid, and
//     pgid so the test can cross-check against the manifest. The
//     filename contains the child pid so the test can match the
//     record by PID without a process search.
//  2. `descriptor-ready.wait` — the parent's poll handle. It is a
//     degenerate form (no pid in name) so the parent can sequence
//     its exit immediately after child readiness without forcing
//     a PID-discovery round trip.
//
// The child holds the inherited descriptors open by sleeping for
// at least 60 s, which is two orders of magnitude beyond the
// executor's WaitDelay (TerminationGrace + PostKillWait = 1 s by
// default). The executor MUST therefore trigger WaitDelay cleanup,
// not the request context, not a goroutine exit.
func runHeldDescriptorChild() {
	pid := os.Getpid()
	ppid := os.Getppid()
	pgid := syscall.Getpgrp()

	recordPID("child", "held-descriptor-child", false)

	// Step 1: publish the PID-bound descriptor-ready sentinel so the
	// test can cross-check the contents against the manifest. The
	// file is fsynced before close so the test observes a true
	// transition.
	pidReady := filepath.Join(readyDir,
		fmt.Sprintf("%d.descriptor-ready.ready", pid))
	f, err := os.Create(pidReady)
	if err != nil {
		failClosed("held-descriptor-child",
			"failed to publish PID-bound descriptor-ready sentinel: %v", err)
	}
	pidContent := fmt.Sprintf("role=child\npid=%d\nppid=%d\npgid=%d\n",
		pid, ppid, pgid)
	if _, err := io.WriteString(f, pidContent); err != nil {
		_ = f.Close()
		failClosed("held-descriptor-child",
			"failed to write descriptor-ready contents: %v", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		failClosed("held-descriptor-child",
			"failed to sync descriptor-ready sentinel: %v", err)
	}
	_ = f.Close()

	// Step 2: publish the parent-handle sentinel so the parent can
	// block on it without PID discovery.
	parentHandle := filepath.Join(readyDir, "descriptor-ready.wait")
	f2, err := os.Create(parentHandle)
	if err != nil {
		failClosed("held-descriptor-child",
			"failed to publish descriptor-ready.handle: %v", err)
	}
	if _, err := io.WriteString(f2, "ready"); err != nil {
		_ = f2.Close()
		failClosed("held-descriptor-child",
			"failed to write descriptor-ready handle contents: %v", err)
	}
	if err := f2.Sync(); err != nil {
		_ = f2.Close()
		failClosed("held-descriptor-child",
			"failed to sync descriptor-ready handle: %v", err)
	}
	_ = f2.Close()

	// Hold the inherited descriptors open by sleeping in 60-second
	// chunks. The executor's WaitDelay is bounded at ~1 s so the
	// child deliberately outlives every natural cleanup path.
	for {
		time.Sleep(60 * time.Second)
	}
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
