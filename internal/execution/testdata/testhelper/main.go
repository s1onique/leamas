//go:build unix || darwin || linux

// Package main provides a deterministic test helper for adversarial execution testing.
//
// Modes:
//   - sleep <duration>: Simple sleep (default 10s)
//   - ignore-sigterm: Parent that runs a child ignoring SIGTERM, then sleeps
//   - ignore-sigterm-child: Child that ignores SIGTERM via signal.Ignore before
//     publishing readiness evidence, then sleeps
//   - spawn-child: Parent that spawns a child process
//   - spawn-grandchild: Parent that spawns child which spawns grandchild
//   - hold-stdout-open: Parent that spawns child holding stdout/stderr open
//   - output-forever: Outputs data forever until killed
//   - exit-nonzero: Exits with code 42
//
// The helper is composed of multiple files:
//
//   - main.go         : argv validation + mode dispatch switch.
//   - pid_manifest.go : PIDRecord type, recordPID, publishReady, init.
//   - proc_runtime.go : spawnChildFailClosed, waitChildOrFail, helpers.
//   - modes_sleep.go  : sleep, child, spawn-child, sleep/parseDuration.
//   - modes_tree.go   : spawn-grandchild and sleep-grandchild subtree.
//   - modes_output.go : hold-stdout-open and output-flood family.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr,
			"Usage: %s <mode> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr,
			"Modes: sleep, ignore-sigterm, spawn-child, spawn-grandchild,"+
				" hold-stdout-open, output-forever, exit-nonzero\n")
		os.Exit(1)
	}

	mode := os.Args[1]

	switch mode {
	// Lifecycle and cancellation modes.
	case "sleep":
		runSleep(mode, os.Args[2:])
	case "ignore-sigterm":
		runIgnoreSigterm()
	case "ignore-sigterm-child":
		runIgnoreSigtermChild()

	// Single-child and grandchild subtree modes.
	case "spawn-child":
		runSpawnChild(os.Args[2:])
	case "child":
		runChild(os.Args[2:])
	case "spawn-grandchild":
		runSpawnGrandchild()
	case "sleep-grandchild":
		runSleepGrandchild()
	case "sleep-grandchild-child":
		runSleepGrandchildChild()
	case "sleep-grandchild-grandchild":
		runSleepGrandchildGrandchild()
	case "grandchild-spawner":
		runGrandchildSpawner()
	case "grandchild":
		runGrandchild(os.Args[2:])

	// Output, hold, and exit modes.
	case "hold-stdout-open":
		runHoldStdoutOpen()
	case "stdout-holder":
		runStdoutHolder()
	case "held-descriptor":
		runHeldDescriptor()
	case "held-descriptor-child":
		runHeldDescriptorChild()
	case "negative-overflow-fail":
		runNegativeOutputProvenFail()
	case "output-forever":
		runOutputForever()
	case "output-forever-fast":
		runOutputForeverFast()
	case "output-forever-child":
		runOutputForeverChild()
	case "output-forever-fast-child":
		runOutputForeverFastChild()
	case "output-forever-grandchild":
		runOutputForeverGrandchild()
	case "exit-nonzero":
		runExitNonzero()
	case "exit-nonzero-child":
		runExitNonzeroChild()

	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", mode)
		os.Exit(1)
	}
}
