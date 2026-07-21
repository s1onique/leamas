//go:build unix || darwin || linux

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const retainedPipeProbeInterval = 10 * time.Millisecond

type retainedPipeProbe struct {
	PID      int    `json:"pid"`
	Sequence uint64 `json:"sequence"`
	UnixNano int64  `json:"unix_nano"`
	Bytes    int    `json:"bytes"`
	Error    string `json:"error,omitempty"`
}

// runHeldDescriptor exits naturally after a descriptor-verified child has
// inherited fd 1 and fd 2 and successfully probed stderr.
func runHeldDescriptor() {
	if readyDir == "" {
		failClosed("held-descriptor", "LEAMAS_EXEC_TEST_READY_DIR not set")
	}
	recordPIDWithDescriptors("parent", "held-descriptor", false)
	cmd := spawnChildWithInheritedOutputFailClosed(
		"held-descriptor", "held-descriptor-child")
	_ = cmd

	readyPath := filepath.Join(readyDir, "descriptor-ready.wait")
	if !waitForFile(readyPath, 10*time.Second) {
		failClosed("held-descriptor",
			"descriptor-ready sentinel not observed within 10s")
	}

	pid := os.Getpid()
	imminentPath := filepath.Join(readyDir,
		fmt.Sprintf("parent-exit-imminent.%d", pid))
	exitNanos := time.Now().UnixNano()
	contents := fmt.Sprintf("pid=%d\nparent_exit_imminent_unix_nano=%d\n",
		pid, exitNanos)
	if err := writeSyncedExclusive(imminentPath, contents); err != nil {
		failClosed("held-descriptor", "publish parent exit evidence: %v", err)
	}
	os.Exit(0)
}

// runHeldDescriptorChild ignores SIGPIPE so it can record the write failure
// caused by os/exec closing its read pipe, then remains alive for explicit
// process-group cleanup. It also ignores SIGTERM so the cleanup path must
// exercise the bounded SIGKILL escalation.
func runHeldDescriptorChild() {
	if readyDir == "" {
		failClosed("held-descriptor-child",
			"LEAMAS_EXEC_TEST_READY_DIR not set")
	}
	signal.Ignore(syscall.SIGPIPE, syscall.SIGTERM)
	pid := os.Getpid()
	ppid := os.Getppid()
	pgid := syscall.Getpgrp()
	descriptors := recordPIDWithDescriptors(
		"child", "held-descriptor-child", true)
	probePath := filepath.Join(readyDir,
		fmt.Sprintf("%d.retained-pipe-probes.jsonl", pid))

	sequence := uint64(1)
	if err := emitRetainedPipeProbe(probePath, pid, sequence); err != nil {
		failClosed("held-descriptor-child", "initial stderr probe: %v", err)
	}
	readyContents := fmt.Sprintf("role=child\npid=%d\nppid=%d\npgid=%d\n%s",
		pid, ppid, pgid, formatDescriptorEvidence(descriptors))
	pidReady := filepath.Join(readyDir,
		fmt.Sprintf("%d.descriptor-ready.ready", pid))
	if err := writeSyncedExclusive(pidReady, readyContents); err != nil {
		failClosed("held-descriptor-child",
			"publish PID-bound descriptor readiness: %v", err)
	}
	parentHandle := filepath.Join(readyDir, "descriptor-ready.wait")
	if err := writeSyncedExclusive(parentHandle, "ready\n"); err != nil {
		failClosed("held-descriptor-child",
			"publish parent descriptor handle: %v", err)
	}

	for {
		time.Sleep(retainedPipeProbeInterval)
		sequence++
		if err := emitRetainedPipeProbe(probePath, pid, sequence); err != nil {
			// The final write error is persisted outside the retained pipe.
			// Stay alive so the executor, rather than SIGPIPE, owns cleanup.
			sleepForever()
		}
	}
}

func emitRetainedPipeProbe(path string, pid int, sequence uint64) error {
	line := fmt.Sprintf("retained-pipe-probe pid=%d sequence=%d\n", pid, sequence)
	written, writeErr := io.WriteString(os.Stderr, line)
	if writeErr == nil && written != len(line) {
		writeErr = io.ErrShortWrite
	}
	probe := retainedPipeProbe{
		PID:      pid,
		Sequence: sequence,
		UnixNano: time.Now().UnixNano(),
		Bytes:    written,
	}
	if writeErr != nil {
		probe.Error = writeErr.Error()
	}
	if err := appendProbeEvidence(path, probe); err != nil {
		return fmt.Errorf("record probe evidence: %w", err)
	}
	return writeErr
}

func appendProbeEvidence(path string, probe retainedPipeProbe) error {
	data, err := json.Marshal(probe)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writeSyncedExclusive(path, contents string) error {
	file, err := os.OpenFile(path,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(file, contents); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
