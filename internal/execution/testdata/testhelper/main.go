//go:build unix || darwin || linux

// Package main provides a deterministic test helper for adversarial execution testing.
// This binary supports various modes to test process tree termination behavior.
//
// Build: go build -o testhelper main.go
//
// Modes:
//   - sleep <duration>: Simple sleep (default 10s)
//   - ignore-sigterm: Parent that runs a child ignoring SIGTERM, then sleeps
//   - ignore-sigterm-child: Child that ignores SIGTERM via signal.Ignore
//   - spawn-child: Parent that spawns a child process
//   - spawn-grandchild: Parent that spawns child which spawns grandchild
//   - hold-stdout-open: Parent that spawns child holding stdout/stderr open
//   - output-forever: Outputs data forever until killed
//   - exit-nonzero: Exits with code 42
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// PIDRecord records a process in the PID manifest.
type PIDRecord struct {
	Role  string `json:"role"`  // "parent", "child", "grandchild"
	Mode  string `json:"mode"`  // The mode that created this record
	PID   int    `json:"pid"`   // Process ID
	PPID  int    `json:"ppid"`  // Parent process ID
	PGID  int    `json:"pgid"`  // Process group ID
	Start int64  `json:"start"` // Unix timestamp when recorded
}

var manifestFile string

func init() {
	manifestFile = os.Getenv("LEAMAS_EXEC_TEST_PID_FILE")
}

func recordPID(role string, mode string) {
	if manifestFile == "" {
		// Must fail closed if no manifest file
		fmt.Fprintf(os.Stderr, "ERROR: LEAMAS_EXEC_TEST_PID_FILE not set\n")
		os.Exit(1)
	}

	pid := os.Getpid()
	ppid := os.Getppid()
	pgid := syscall.Getpgrp()

	record := PIDRecord{
		Role:  role,
		Mode:  mode,
		PID:   pid,
		PPID:  ppid,
		PGID:  pgid,
		Start: time.Now().Unix(),
	}

	data, err := json.Marshal(record)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to marshal PID record: %v\n", err)
		os.Exit(1)
	}

	f, err := os.OpenFile(manifestFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to open manifest: %v\n", err)
		os.Exit(1)
	}

	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write manifest: %v\n", err)
		f.Close()
		os.Exit(1)
	}

	// Sync to ensure data is flushed before process continues
	if err := f.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to sync manifest: %v\n", err)
		f.Close()
		os.Exit(1)
	}

	if err := f.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to close manifest: %v\n", err)
		os.Exit(1)
	}
}

func parseDuration(s string) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Duration(f * float64(time.Second))
	}
	return 10 * time.Second
}

func sleepForever() {
	// Use time.Sleep in a loop to block without triggering Go's deadlock detector.
	// The goroutine has work (scheduling sleeps), so it won't be detected as deadlocked.
	for {
		time.Sleep(24 * time.Hour)
	}
}

func spawnChild(mode string, args ...string) *exec.Cmd {
	helperPath := os.Args[0]
	cmd := exec.Command(helperPath, append([]string{mode}, args...)...)
	if manifestFile != "" {
		cmd.Env = append(os.Environ(), "LEAMAS_EXEC_TEST_PID_FILE="+manifestFile)
	}
	// Children join parent's process group so all can be terminated together
	// Setpgid: false (default) means child joins parent's process group
	cmd.Start()
	return cmd
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <mode> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Modes: sleep, ignore-sigterm, spawn-child, spawn-grandchild, hold-stdout-open, output-forever, exit-nonzero\n")
		os.Exit(1)
	}

	mode := os.Args[1]

	switch mode {
	case "sleep":
		recordPID("parent", mode)
		d := parseDuration("10s")
		if len(os.Args) > 2 {
			d = parseDuration(os.Args[2])
		}
		time.Sleep(d)

	case "ignore-sigterm":
		recordPID("parent", mode)
		// Spawn child that ignores SIGTERM using os/exec
		cmd := spawnChild("ignore-sigterm-child")
		cmd.Wait()
		time.Sleep(30 * time.Second) // Sleep to allow signal escalation test

	case "ignore-sigterm-child":
		recordPID("child", mode)
		// Ignore SIGTERM using Go's signal.Ignore
		signal.Ignore(syscall.SIGTERM)
		sleepForever()

	case "spawn-child":
		recordPID("parent", mode)
		duration := "10s"
		if len(os.Args) > 2 {
			duration = os.Args[2]
		}
		cmd := spawnChild("child", duration)
		cmd.Wait()
		time.Sleep(10 * time.Second)

	case "child":
		recordPID("child", mode)
		d := parseDuration("10s")
		if len(os.Args) > 2 {
			d = parseDuration(os.Args[2])
		}
		time.Sleep(d)

	case "spawn-grandchild":
		recordPID("parent", mode)
		// Spawn child that spawns grandchild and exit immediately
		_ = spawnChild("grandchild-spawner")

	case "sleep-grandchild":
		// Full 3-level tree that sleeps forever (no output) - for timeout testing
		// All processes ignore SIGTERM so they can only be killed via SIGKILL
		// Parent spawns children WITHOUT waiting so they all stay alive
		signal.Ignore(syscall.SIGTERM)
		recordPID("parent", mode)
		spawnChild("sleep-grandchild-child") // Don't wait - let children run independently
		sleepForever()

	case "sleep-grandchild-child":
		// Also ignore SIGTERM so this process doesn't exit when parent is killed
		signal.Ignore(syscall.SIGTERM)
		recordPID("child", mode)
		// Small delay to ensure child is recorded before grandchild starts
		time.Sleep(10 * time.Millisecond)
		spawnChild("sleep-grandchild-grandchild") // Don't wait
		// Small delay to allow grandchild to start and record before we sleep forever
		time.Sleep(10 * time.Millisecond)
		sleepForever()

	case "sleep-grandchild-grandchild":
		// Grandchild also ignores SIGTERM - all processes in the tree should
		// ignore SIGTERM so they can only be killed via SIGKILL
		signal.Ignore(syscall.SIGTERM)
		recordPID("grandchild", mode)
		sleepForever()

	case "grandchild-spawner":
		recordPID("child", mode)
		// Spawn grandchild in background
		grandchild := spawnChild("grandchild", "10s")
		// Wait a moment for grandchild to record, then exit
		time.Sleep(50 * time.Millisecond)
		_ = grandchild

	case "grandchild":
		recordPID("grandchild", mode)
		d := parseDuration("10s")
		if len(os.Args) > 2 {
			d = parseDuration(os.Args[2])
		}
		time.Sleep(d)

	case "hold-stdout-open":
		recordPID("parent", mode)
		cmd := spawnChild("stdout-holder")
		cmd.Wait()

	case "stdout-holder":
		recordPID("child", mode)
		// Hold stdout and stderr open by sleeping
		// The descriptors are inherited from parent
		fmt.Println("stdout-holder started")
		sleepForever()

	case "output-forever":
		recordPID("parent", mode)
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

	case "output-forever-fast":
		// Output mode that writes in small increments for faster overflow detection
		// Record PID only if manifest file is set (allow parent to produce output even without manifest)
		if manifestFile != "" {
			recordPID("parent", mode)
		}
		// Write 'x' one at a time using buffered stdout - 1 byte per write = 64 writes to overflow 64-byte buffer
		for {
			fmt.Print("x")
		}

	case "output-forever-child":
		recordPID("child", mode)
		cmd := spawnChild("output-forever")
		cmd.Wait()

	case "output-forever-fast-child":
		// Child that spawns grandchild writing 1 byte at a time (fast overflow)
		recordPID("child", mode)
		cmd := spawnChild("output-forever-fast")
		cmd.Wait()

	case "output-forever-grandchild":
		// Full 3-level tree: parent -> child -> grandchild
		// Each records its PID before the next level starts
		if manifestFile != "" {
			recordPID("parent", mode)
		}

		// Spawn child that will spawn grandchild and wait for full tree to be established
		cmd := spawnChild("grandchild-spawner")

		// Wait for child to complete spawning grandchild
		cmd.Wait()

		// Full tree is now established: parent, child, and grandchild are all recorded
		// Now begin infinite output from the parent (grandchild also outputs)
		for {
			fmt.Print("x")
		}

	case "exit-nonzero":
		recordPID("parent", mode)
		os.Exit(42)

	case "exit-nonzero-child":
		recordPID("child", mode)
		cmd := spawnChild("exit-nonzero")
		cmd.Wait()
		if cmd.ProcessState != nil {
			if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
				os.Exit(ws.ExitStatus())
			}
		}
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", mode)
		os.Exit(1)
	}
}
