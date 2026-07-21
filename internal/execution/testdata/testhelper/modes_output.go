//go:build unix || darwin || linux

// Package main provides a deterministic test helper for adversarial execution testing.
//
// modes_output.go owns the held-descriptor, output-flood, and exit-code
// modes. The wait-on-child semantics chosen here determine whether the
// executor observes a stable parent/child tree (waitChildOrFail) or a
// deterministic exit-status propagation (waitChildAndPropagate).
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// runHoldStdoutOpen waits for the stdout-holder child to exit and surfaces
// the child's exit status. The child holds its descriptors open so the
// executor must rely on WaitDelay to bound cleanup.
func runHoldStdoutOpen() {
	recordPID("parent", "hold-stdout-open", false)
	// Parents must wait for the stdout holder child so the executor
	// observes a stable parent/child tree before timing out.
	cmd := spawnChildFailClosed("hold-stdout-open", "stdout-holder")
	// Don't fail-closed on this wait. Hold the parent open until the
	// child returns or signals; if the child unexpectedly exits we
	// surface the exit code so the test framework sees it.
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

// runStdoutHolder holds stdout/stderr open by sleeping. The descriptors
// are inherited from the parent so the executor's WaitDelay path must
// fire to release them.
func runStdoutHolder() {
	recordPID("child", "stdout-holder", false)
	fmt.Println("stdout-holder started")
	sleepForever()
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
// (e.g. 64 bytes) trips within tens of milliseconds. The recordPID happens
// unconditionally because the auxiliary diagnostic mode is purely about
// overflow detection.
func runOutputForeverFast() {
	recordPID("parent", "output-forever-fast", false)
	for {
		fmt.Print("x")
	}
}

// runOutputForeverChild uses waitChildOrFail because the child is expected
// to flood output until the executor kills the entire group.
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

// runOutputForeverGrandchild waits for grandchild-spawner (which exits
// after 50 ms once the grandchild has recorded itself) before flooding
// output. This guarantees all three roles are recorded before overflow
// can be triggered.
func runOutputForeverGrandchild() {
	recordPID("parent", "output-forever-grandchild", false)

	// Spawn child that will spawn grandchild and wait for the full tree
	// to be established.
	cmd := spawnChildFailClosed("output-forever-grandchild",
		"grandchild-spawner")
	waitChildOrFail("output-forever-grandchild", cmd)

	// Full tree is now established: parent, child, and grandchild are
	// all recorded. Begin infinite output so the executor's output
	// overflow detection fires.
	for {
		fmt.Print("x")
	}
}

// runExitNonzero records the parent and exits with code 42 so callers
// observing the process exit code can verify the executor surfaces it
// through Result.ExitCode.
func runExitNonzero() {
	recordPID("parent", "exit-nonzero", false)
	os.Exit(42)
}

// runExitNonzeroChild records itself, spawns an exit-nonzero child, and
// propagates its exit status so callers observing the parent exit code
// see 42.
func runExitNonzeroChild() {
	recordPID("child", "exit-nonzero-child", false)
	cmd := spawnChildFailClosed("exit-nonzero-child", "exit-nonzero")
	waitChildAndPropagate(cmd)
}
