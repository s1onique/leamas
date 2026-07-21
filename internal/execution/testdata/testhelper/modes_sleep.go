//go:build unix || darwin || linux

// Package main provides a deterministic test helper for adversarial execution testing.
//
// modes_sleep.go owns the lifecycle and cancellation-focused modes. The
// child processes here either ignore SIGTERM (ignore-sigterm-child), sleep
// for a bounded duration, or block until explicitly killed.
package main

import (
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// parseDuration accepts an integer/decimal second string OR any time.ParseDuration string.
// An empty or invalid argument falls back to the 10-second default which
// matches the historical exit-nonzero-based expected bounded lifetime.
func parseDuration(s string) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Duration(f * float64(time.Second))
	}
	return 10 * time.Second
}

// sleepForever blocks the helper without triggering Go's deadlock detector
// by emitting long sleeps in a tight loop.
func sleepForever() {
	for {
		time.Sleep(24 * time.Hour)
	}
}

// runSleep implements the `sleep` mode. The helper records itself, then
// sleeps for the requested duration (or 10s by default).
func runSleep(mode string, args []string) {
	recordPID("parent", mode, false)
	d := parseDuration("10s")
	if len(args) > 0 {
		d = parseDuration(args[0])
	}
	time.Sleep(d)
}

// runChild implements the `child` mode. The helper records itself, then
// sleeps for the requested duration. Used as a spawned child of
// spawn-child and spawn-grandchild.
func runChild(args []string) {
	recordPID("child", "child", false)
	d := parseDuration("10s")
	if len(args) > 0 {
		d = parseDuration(args[0])
	}
	time.Sleep(d)
}

// runIgnoreSigterm is the parent side of the SIGTERM escalation proof. It
// records itself, then spawns the SIGTERM-ignore child and refuses to
// continue if the child exits before the test trigger fires.
func runIgnoreSigterm() {
	recordPID("parent", "ignore-sigterm", false)
	cmd := spawnChildFailClosed("ignore-sigterm", "ignore-sigterm-child")
	// The child is required by the test contract. If it exits before
	// the test signals, the helper fails closed instead of masking the
	// unexpected exit with a long sleep.
	waitChildOrFail("ignore-sigterm", cmd)
}

// runIgnoreSigtermChild is the child side of the SIGTERM escalation proof.
// CRITICAL ORDERING: signal.Ignore(SIGTERM) MUST be installed BEFORE the
// PID record is written and BEFORE readiness is published. The test
// harness only considers the child ready when its manifest record carries
// signal_ready=true after the handler has been installed.
func runIgnoreSigtermChild() {
	signal.Ignore(syscall.SIGTERM)
	recordPID("child", "ignore-sigterm-child", true)
	publishReady("ignore-sigterm-child")
	sleepForever()
}

// runSpawnChild waits for the spawned child to exit and propagates its
// exit status. The test relies on the parent staying alive long enough
// for executor timeout to terminate the tree.
func runSpawnChild(args []string) {
	recordPID("parent", "spawn-child", false)
	duration := "10s"
	if len(args) > 0 {
		duration = args[0]
	}
	cmd := spawnChildFailClosed("spawn-child", "child", duration)
	waitChildAndPropagate(cmd)
}
