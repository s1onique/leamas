//go:build unix || darwin || linux

// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"syscall"
	"time"
)

// processGroupManager manages process group termination on Unix.
type processGroupManager struct{}

// newProcessGroupManager creates a new process group manager.
func newProcessGroupManager() *processGroupManager {
	return &processGroupManager{}
}

// killProcessGroup kills the entire process group.
// On Unix, this uses syscall.Kill with negative PID.
func (m *processGroupManager) killProcessGroup(pid int, sig syscall.Signal) error {
	// On Unix, PID 0 means the current process group
	// Negative PID means the process group of that PID
	pgid := -pid
	return syscall.Kill(pgid, sig)
}

// waitForProcessGroup waits for all processes in a group to terminate.
// Returns true if all processes have terminated.
func (m *processGroupManager) waitForProcessGroup(pid int, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Try to send signal 0 to check if process group exists
		pgid := -pid
		err := syscall.Kill(pgid, syscall.Signal(0))
		if err != nil {
			// ESRCH means no such process - group is gone
			if err == syscall.ESRCH {
				return true, nil
			}
			// EINVAL does not prove process group is absent - fail closed
			// Only ESRCH confirms absence per POSIX
			if err == syscall.EINVAL {
				return false, err
			}
			return false, err
		}

		// Wait a bit before checking again
		time.Sleep(10 * time.Millisecond)
	}

	return false, nil
}
