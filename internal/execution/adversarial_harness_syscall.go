//go:build unix || darwin || linux

package execution

import (
	"fmt"
	"strings"
	"syscall"
	"time"
)

// isProcessAlive checks if a process is alive.
func (v *processVerifier) isProcessAlive(pid int) (bool, error) {
	err := syscall.Kill(pid, syscall.Signal(0))
	if err == nil {
		return true, nil
	}
	if err == syscall.ESRCH {
		return false, nil
	}
	if err == syscall.EPERM {
		return true, fmt.Errorf("EPERM: cannot verify process %d", pid)
	}
	return false, err
}

// isProcessGroupAlive checks if a process group is alive.
func (v *processVerifier) isProcessGroupAlive(pgid int) (bool, error) {
	err := syscall.Kill(-pgid, syscall.Signal(0))
	if err == nil {
		return true, nil
	}
	if err == syscall.ESRCH {
		return false, nil
	}
	if err == syscall.EINVAL {
		return true, fmt.Errorf("EINVAL: cannot verify process group %d", pgid)
	}
	if err == syscall.EPERM {
		return true, fmt.Errorf("EPERM: cannot verify process group %d", pgid)
	}
	return false, err
}

// verifyAllProcessesAbsent verifies all recorded PIDs and process groups are gone.
func (v *processVerifier) verifyAllProcessesAbsent(verificationTimeout time.Duration) error {
	v.t.Helper()

	records, err := v.parseManifest()
	if err != nil {
		return fmt.Errorf("manifest parse failed: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("empty manifest: cannot verify absence without evidence")
	}

	deadline := time.Now().Add(verificationTimeout)
	pollInterval := 10 * time.Millisecond

	var failedChecks []string

	for time.Now().Before(deadline) {
		failedChecks = nil

		for _, rec := range records {
			alive, err := v.isProcessAlive(rec.PID)
			if err != nil {
				failedChecks = append(failedChecks,
					fmt.Sprintf("PID %d (%s): verification error: %v", rec.PID, rec.Role, err))
				continue
			}
			if alive {
				failedChecks = append(failedChecks,
					fmt.Sprintf("PID %d (%s): still alive", rec.PID, rec.Role))
			}
		}

		groups := v.getProcessGroups()
		for pgid := range groups {
			alive, err := v.isProcessGroupAlive(pgid)
			if err != nil {
				failedChecks = append(failedChecks,
					fmt.Sprintf("PGID %d: verification error: %v", pgid, err))
				continue
			}
			if alive {
				failedChecks = append(failedChecks,
					fmt.Sprintf("PGID %d: still alive", pgid))
			}
		}

		if len(failedChecks) == 0 {
			return nil
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("verification timeout after %v:\n  %s",
		verificationTimeout, strings.Join(failedChecks, "\n  "))
}

// verifyWithCleanup kills leaked processes and reports them.
func (v *processVerifier) verifyWithCleanup() {
	v.t.Helper()

	records, err := v.parseManifest()
	if err != nil {
		v.t.Logf("cleanup: manifest parse failed: %v", err)
		return
	}

	var leaked []string
	for _, rec := range records {
		alive, _ := v.isProcessAlive(rec.PID)
		if alive {
			_ = syscall.Kill(rec.PID, syscall.SIGKILL)
			leaked = append(leaked, fmt.Sprintf("PID %d (%s)", rec.PID, rec.Role))
		}
	}

	for pgid := range v.getProcessGroups() {
		alive, _ := v.isProcessGroupAlive(pgid)
		if alive {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
			leaked = append(leaked, fmt.Sprintf("PGID %d", pgid))
		}
	}

	if len(leaked) > 0 {
		v.t.Logf("WARNING: forcibly killed leaked processes: %v", leaked)
	}
}
