//go:build unix || darwin || linux

package execution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestRawExecNaturalExitRetainedPipeWaitDelay is the standard-library control
// for the retained-pipe fixture. It must pass before Executor production code
// is modified for this ACT.
func TestRawExecNaturalExitRetainedPipeWaitDelay(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux /proc descriptor proof is required")
	}
	helperPath, err := locateHelperBinary()
	if err != nil {
		t.Fatalf("locate content-verified helper: %v", err)
	}
	verifier, _ := newProcessVerifier(t)
	output := newSharedOutputBuffer(1 << 20)

	cmd := exec.CommandContext(context.Background(), helperPath, retainedPipeMode)
	cmd.Stdout = output.StdoutWriter()
	cmd.Stderr = output.StderrWriter()
	cmd.WaitDelay = retainedPipeWaitDelay
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = updateEnv(os.Environ(), "LEAMAS_EXEC_TEST_PID_FILE", verifier.ManifestFile())
	cmd.Env = updateEnv(cmd.Env, "LEAMAS_EXEC_TEST_READY_DIR", verifier.ReadyDir())
	if err := cmd.Start(); err != nil {
		t.Fatalf("start raw os/exec control: %v", err)
	}

	guard := &retainedProcessGroupGuard{}
	guard.arm(cmd.Process.Pid)
	defer guard.emergencyCleanup()
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	handoff := waitForRetainedPipeHandoff(t, verifier)
	if handoff.Parent.PID != cmd.Process.Pid {
		t.Fatalf("raw pid=%d, fixture parent pid=%d", cmd.Process.Pid, handoff.Parent.PID)
	}

	upperDeadline := time.Unix(0, handoff.Exit.UnixNano).
		Add(retainedPipeWaitDelay + retainedPipeUpperSlack)
	var waitErr error
	select {
	case waitErr = <-waitCh:
	case <-time.After(time.Until(upperDeadline)):
		t.Fatalf("raw Wait exceeded WaitDelay upper bound")
	}
	elapsed := time.Since(time.Unix(0, handoff.Exit.UnixNano))
	if !errors.Is(waitErr, exec.ErrWaitDelay) {
		t.Fatalf("raw Wait error=%v, want errors.Is(exec.ErrWaitDelay); elapsed=%v",
			waitErr, elapsed)
	}
	if elapsed < retainedPipeWaitDelay-retainedPipeLowerSlack {
		t.Fatalf("raw Wait returned too early: %v < %v", elapsed,
			retainedPipeWaitDelay-retainedPipeLowerSlack)
	}
	if elapsed > retainedPipeWaitDelay+retainedPipeUpperSlack {
		t.Fatalf("raw Wait returned too late: %v > %v", elapsed,
			retainedPipeWaitDelay+retainedPipeUpperSlack)
	}
	if elapsed >= retainedPipeTimeout/2 {
		t.Fatalf("raw Wait approached request bound: elapsed=%v timeout=%v",
			elapsed, retainedPipeTimeout)
	}

	probeText := fmt.Sprintf("retained-pipe-probe pid=%d sequence=%d",
		handoff.PostProbe.PID, handoff.PostProbe.Sequence)
	if !strings.Contains(string(output.Stderr()), probeText) {
		t.Fatalf("captured stderr omits post-exit probe %q", probeText)
	}
	probePath := filepath.Join(verifier.ReadyDir(),
		fmt.Sprintf("%d.retained-pipe-probes.jsonl", handoff.Child.PID))
	if _, err := waitForProbeError(probePath, time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := requireNonZombieProcess(handoff.Child.PID); err != nil {
		t.Fatalf("holder did not survive os/exec pipe closure for explicit cleanup: %v", err)
	}

	if err := cleanupRetainedProcessGroup(handoff.Parent.PGID,
		100*time.Millisecond, time.Second); err != nil {
		t.Fatalf("clean raw retained process group: %v", err)
	}
	if err := verifier.verifyAllProcessesAbsent(time.Second); err != nil {
		t.Fatalf("raw control leaked fixture processes: %v", err)
	}
	guard.disarm()
	if guard.emergencyUsed {
		t.Fatal("passing raw control used emergency cleanup")
	}
}
