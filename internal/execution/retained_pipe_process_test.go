//go:build unix || darwin || linux

package execution

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func waitForPIDBoundReady(dir, role string, timeout time.Duration) string {
	path, err := waitForSinglePath(filepath.Join(dir, "*."+role+".ready"),
		time.Now().Add(timeout))
	if err != nil {
		return ""
	}
	return path
}

func waitForSinglePath(pattern string, deadline time.Time) (string, error) {
	for time.Now().Before(deadline) {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return "", err
		}
		switch len(matches) {
		case 0:
			time.Sleep(readinessPollInterval)
		case 1:
			return matches[0], nil
		default:
			return "", fmt.Errorf("pattern %q matched multiple paths: %v", pattern, matches)
		}
	}
	return "", fmt.Errorf("deadline exceeded waiting for %q", pattern)
}

func waitForProcessAbsent(verifier *processVerifier, pid int,
	deadline time.Time,
) error {
	for time.Now().Before(deadline) {
		alive, err := verifier.isProcessAlive(pid)
		if err != nil {
			return fmt.Errorf("check pid %d: %w", pid, err)
		}
		if !alive {
			return nil
		}
		time.Sleep(readinessPollInterval)
	}
	return fmt.Errorf("pid %d remained present", pid)
}

func requireNonZombieProcess(pid int) error {
	if err := syscall.Kill(pid, 0); err != nil {
		return fmt.Errorf("signal-zero check: %w", err)
	}
	if runtime.GOOS != "linux" {
		return nil
	}
	contents, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return fmt.Errorf("read proc stat: %w", err)
	}
	state, err := parseLinuxProcState(string(contents))
	if err != nil {
		return err
	}
	if state == 'Z' {
		return fmt.Errorf("pid %d is a zombie", pid)
	}
	return nil
}

func parseLinuxProcState(contents string) (byte, error) {
	closeParen := strings.LastIndexByte(contents, ')')
	if closeParen < 0 || closeParen+1 >= len(contents) {
		return 0, fmt.Errorf("malformed /proc stat")
	}
	fields := strings.Fields(contents[closeParen+1:])
	if len(fields) == 0 || len(fields[0]) != 1 {
		return 0, fmt.Errorf("missing /proc process state")
	}
	return fields[0][0], nil
}

type retainedProcessGroupGuard struct {
	pgid          int
	armed         bool
	emergencyUsed bool
}

func (g *retainedProcessGroupGuard) arm(pgid int) {
	g.pgid = pgid
	g.armed = true
}

func (g *retainedProcessGroupGuard) disarm() {
	g.armed = false
}

func (g *retainedProcessGroupGuard) emergencyCleanup() {
	if !g.armed || g.pgid <= 0 {
		return
	}
	g.emergencyUsed = true
	_ = syscall.Kill(-g.pgid, syscall.SIGKILL)
	_, _ = newProcessGroupManager().waitForProcessGroup(g.pgid, time.Second)
	g.armed = false
}

func cleanupRetainedProcessGroup(pgid int, grace, postKill time.Duration) error {
	manager := newProcessGroupManager()
	if err := manager.killProcessGroup(pgid, syscall.SIGTERM); err != nil &&
		!isESRCH(err) {
		return fmt.Errorf("signal retained group with SIGTERM: %w", err)
	}
	absent, err := manager.waitForProcessGroup(pgid, grace)
	if err != nil {
		return fmt.Errorf("wait after SIGTERM: %w", err)
	}
	if absent {
		return nil
	}
	if err := manager.killProcessGroup(pgid, syscall.SIGKILL); err != nil &&
		!isESRCH(err) {
		return fmt.Errorf("signal retained group with SIGKILL: %w", err)
	}
	absent, err = manager.waitForProcessGroup(pgid, postKill)
	if err != nil {
		return fmt.Errorf("wait after SIGKILL: %w", err)
	}
	if !absent {
		return fmt.Errorf("process group %d survived bounded cleanup", pgid)
	}
	return nil
}
