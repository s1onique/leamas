//go:build unix || darwin || linux

// Package main provides a deterministic test helper for adversarial execution testing.
//
// pid_manifest.go owns the PID manifest writer and readiness publishing used
// by every test mode. It is kept separate from main() to keep that file
// focused on the mode-dispatch switch.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// PIDRecord records a process in the PID manifest.
//
// SignalReady is true ONLY when the recording process has already installed
// every signal behavior the relevant test mode requires. The test harness
// treats a record with SignalReady=false as proof that the process has not
// yet reached the required state and must not publish readiness.
type PIDRecord struct {
	Role        string         `json:"role"` // "parent", "child", "grandchild"
	Mode        string         `json:"mode"` // The mode that created this record
	PID         int            `json:"pid"`  // Process ID
	PPID        int            `json:"ppid"` // Parent process ID
	PGID        int            `json:"pgid"` // Process group ID
	Start       int64          `json:"start"`
	SignalReady bool           `json:"signal_ready"` // True iff required signal handlers already installed
	Descriptors *descriptorSet `json:"descriptors,omitempty"`
}

var (
	manifestFile string
	readyDir     string
)

// init reads the manifest path and readiness directory from the environment.
// If either is missing for a mode that requires it, the helper exits non-zero
// before any child is spawned.
func init() {
	manifestFile = os.Getenv("LEAMAS_EXEC_TEST_PID_FILE")
	readyDir = os.Getenv("LEAMAS_EXEC_TEST_READY_DIR")
}

// publishReady writes a per-process ready sentinel under readyDir/<pid>.ready
// with fsync, then closes the file. The file is opened with O_CREATE|O_EXCL
// so duplicate publication is observable.
//
// The ready sentinel is auxiliary diagnostic evidence. The authoritative
// readiness signal is the SignalReady=true flag in the manifest record
// itself. publishReady is therefore a no-op when LEAMAS_EXEC_TEST_READY_DIR
// is unset so legacy tests can run unchanged.
func publishReady(role string) {
	if readyDir == "" {
		return
	}
	pid := os.Getpid()
	readyPath := filepath.Join(readyDir, fmt.Sprintf("%d.ready", pid))
	f, err := os.OpenFile(readyPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"ERROR: %s failed to create ready file %s: %v\n",
			role, readyPath, err)
		os.Exit(1)
	}
	if _, err := io.WriteString(f, role); err != nil {
		_ = f.Close()
		fmt.Fprintf(os.Stderr,
			"ERROR: %s failed to write ready file %s: %v\n",
			role, readyPath, err)
		os.Exit(1)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		fmt.Fprintf(os.Stderr,
			"ERROR: %s failed to sync ready file %s: %v\n",
			role, readyPath, err)
		os.Exit(1)
	}
	if err := f.Close(); err != nil {
		fmt.Fprintf(os.Stderr,
			"ERROR: %s failed to close ready file %s: %v\n",
			role, readyPath, err)
		os.Exit(1)
	}
}

// recordPID writes a single PIDRecord line to manifestFile. When signalReady
// is true, the record carries the signal_ready flag so the verifier can
// distinguish "process exists, behavior installed" from "process exists,
// signal still pending".
//
// All underlying write failures exit the helper with a non-zero status so
// the test harness treats a silently truncated manifest as a helper failure.
func recordPID(role string, mode string, signalReady bool) {
	recordPIDWithEvidence(role, mode, signalReady, nil)
}

func recordPIDWithDescriptors(role string, mode string, signalReady bool) descriptorSet {
	descriptors, err := captureDescriptorSet()
	if err != nil {
		failClosed(mode, "capture stdout/stderr descriptors: %v", err)
	}
	recordPIDWithEvidence(role, mode, signalReady, &descriptors)
	return descriptors
}

func recordPIDWithEvidence(role string, mode string, signalReady bool,
	descriptors *descriptorSet,
) {
	if manifestFile == "" {
		// Must fail closed if no manifest file
		fmt.Fprintf(os.Stderr, "ERROR: LEAMAS_EXEC_TEST_PID_FILE not set\n")
		os.Exit(1)
	}

	pid := os.Getpid()
	ppid := os.Getppid()
	pgid := syscall.Getpgrp()

	record := PIDRecord{
		Role:        role,
		Mode:        mode,
		PID:         pid,
		PPID:        ppid,
		PGID:        pgid,
		Start:       time.Now().Unix(),
		SignalReady: signalReady,
		Descriptors: descriptors,
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
		_ = f.Close()
		fmt.Fprintf(os.Stderr, "ERROR: failed to write manifest: %v\n", err)
		os.Exit(1)
	}

	// Sync to ensure data is flushed before process continues
	if err := f.Sync(); err != nil {
		_ = f.Close()
		fmt.Fprintf(os.Stderr, "ERROR: failed to sync manifest: %v\n", err)
		os.Exit(1)
	}

	if err := f.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to close manifest: %v\n", err)
		os.Exit(1)
	}
}
