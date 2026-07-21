//go:build unix || darwin || linux

// Package main provides a deterministic test helper for adversarial execution testing.
//
// modes_tree.go owns the multi-process tree modes (spawn-grandchild,
// sleep-grandchild family). Sleep-grandchild processes all install
// signal.Ignore(SIGTERM) so the test can prove the executor escalates to
// SIGKILL when the process group does not respond to SIGTERM.
package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

// runSpawnGrandchild records the parent and forks the grandchild-spawner
// in the background. This mode is not used by a specific adversarial test
// but is exercised through expectedRolesForMode. Uses
// waitChildExpectedSuccess because the grandchild-spawner is intentionally
// supposed to finish cleanly so the parent can exit.
func runSpawnGrandchild() {
	recordPID("parent", "spawn-grandchild", false)
	// Spawn child that spawns grandchild and exit immediately.
	cmd := spawnChildFailClosed("spawn-grandchild", "grandchild-spawner")
	waitChildExpectedSuccess("spawn-grandchild", cmd)
}

// runSleepGrandchild is the parent of a 3-level SIGTERM-immune tree. It
// installs signal.Ignore before recording or publishing readiness, then
// spawns the child and sleeps forever. The child outlives the parent so
// the executor must terminate every process in the tree.
func runSleepGrandchild() {
	signal.Ignore(syscall.SIGTERM)
	recordPID("parent", "sleep-grandchild", true)
	publishReady("sleep-grandchild")
	cmd := spawnChildFailClosed("sleep-grandchild", "sleep-grandchild-child")
	// We deliberately do NOT wait. Allow the child to outlive us.
	_ = cmd
	sleepForever()
}

// runSleepGrandchildChild spawns the third level and waits forever.
// Like its parent it ignores SIGTERM so a SIGTERM-driven cancellation
// must escalate to SIGKILL.
func runSleepGrandchildChild() {
	signal.Ignore(syscall.SIGTERM)
	recordPID("child", "sleep-grandchild-child", true)
	publishReady("sleep-grandchild-child")
	cmd := spawnChildFailClosed("sleep-grandchild-child",
		"sleep-grandchild-grandchild")
	_ = cmd
	sleepForever()
}

// runSleepGrandchildGrandchild is the deepest level. It ignores SIGTERM
// and sleeps forever.
func runSleepGrandchildGrandchild() {
	signal.Ignore(syscall.SIGTERM)
	recordPID("grandchild", "sleep-grandchild-grandchild", true)
	publishReady("sleep-grandchild-grandchild")
	sleepForever()
}

// runGrandchildSpawner records itself, forks the grandchild in the
// background, waits briefly for the grandchild to record itself, and
// then exits so the parent (output-forever-grandchild) can unblock its
// own Wait() and begin flooding output.
func runGrandchildSpawner() {
	recordPID("child", "grandchild-spawner", false)
	cmd := spawnChildFailClosed("grandchild-spawner", "grandchild", "10s")
	_ = cmd
	// Brief settling so the grandchild has time to record itself and
	// join the executor's process group before the parent exits.
	time.Sleep(50 * time.Millisecond)
	os.Exit(0)
}

// runGrandchild is the leaf of the spawn-grandchild tree. It records
// itself and sleeps for the requested duration.
func runGrandchild(args []string) {
	recordPID("grandchild", "grandchild", false)
	d := parseDuration("10s")
	if len(args) > 0 {
		d = parseDuration(args[0])
	}
	time.Sleep(d)
}
