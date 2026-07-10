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
// Returns (false, error) for any uncertainty - fail-closed.
func (m *processGroupManager) waitForProcessGroup(pid int, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check if process group exists using signal 0
		pgid := -pid
		err := syscall.Kill(pgid, syscall.Signal(0))
		if err == nil {
			// Group exists - wait and retry
			time.Sleep(10 * time.Millisecond)
			continue
		}
		// ESRCH is the only confirmed-absent signal for kill(2)
		if err == syscall.ESRCH {
			return true, nil
		}
		// EINVAL means invalid signal - unexpected for signal 0
		// EPERM means no permission - group may still exist
		// Any other error means uncertainty
		// All non-ESRCH errors are fail-closed
		return false, err
	}

	// Timeout: group still exists
	return false, nil
}

// waitForProcessExit checks if a process has exited using Wait4 with WNOHANG.
// Returns (true, nil) only when the process is confirmed absent.
// Returns (false, error) for any uncertainty - fail-closed.
func (m *processGroupManager) waitForProcessExit(pid int, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		var status syscall.WaitStatus

		// Wait on the specific PID with WNOHANG
		reaped, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
		if err != nil {
			if err == syscall.ECHILD {
				// ECHILD: process is not our child or already reaped.
				// Only ESRCH on kill(0) proves the process is gone.
				if killErr := syscall.Kill(pid, syscall.Signal(0)); killErr == syscall.ESRCH {
					return true, nil // Confirmed absent
				}
				// PID still exists - fail-closed
				return false, err
			}
			if err == syscall.EINTR {
				continue // Retry
			}
			// Other errors mean uncertainty - fail-closed
			return false, err
		}

		if reaped > 0 {
			// Process was reaped - confirmed terminated
			return true, nil
		}

		// reaped == 0: process still running, retry
		time.Sleep(5 * time.Millisecond)
	}

	// Timeout: process still running
	return false, nil
}
