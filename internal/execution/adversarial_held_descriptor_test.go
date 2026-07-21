//go:build unix || darwin || linux

package execution

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

const retainedOutputCodeContract = "execution_retained_output_pipe"

// TestAdversarialHeldDescriptorPipeWaitDelay is the Leamas side of the
// raw-os/exec differential. The same content-verified helper and mode must
// normalize ErrWaitDelay as retained/incomplete output, preserve exit zero,
// and remove the saved process group before returning.
func TestAdversarialHeldDescriptorPipeWaitDelay(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux /proc descriptor proof is required")
	}
	executor := buildRetainedPipeExecutor(t)
	defer executor.Close()
	verifier, _ := newProcessVerifier(t)
	helperPath, err := locateHelperBinary()
	if err != nil {
		t.Fatalf("locate content-verified helper: %v", err)
	}
	req := &Request{
		Name:    "held-descriptor-natural-pipe-waitdelay",
		Args:    []string{helperPath, retainedPipeMode},
		Env:     []string{"LEAMAS_EXEC_TEST_PID_FILE=" + verifier.ManifestFile(), "LEAMAS_EXEC_TEST_READY_DIR=" + verifier.ReadyDir()},
		Timeout: retainedPipeTimeout,
	}

	resultCh := make(chan *Result, 1)
	go func() { resultCh <- executor.Execute(context.Background(), req) }()
	emergencyUsed := false
	completed := false
	defer func() {
		if !completed {
			emergencyUsed = true
			verifier.verifyWithCleanup()
		}
	}()

	handoff := waitForRetainedPipeHandoff(t, verifier)
	lowerDeadline := time.Unix(0, handoff.Exit.UnixNano).
		Add(retainedPipeWaitDelay - retainedPipeLowerSlack)
	if remaining := time.Until(lowerDeadline); remaining > 0 {
		select {
		case result := <-resultCh:
			t.Fatalf("Execute returned before WaitDelay lower bound: result=%+v", result)
		case <-time.After(remaining):
		}
	}

	upperDeadline := time.Unix(0, handoff.Exit.UnixNano).
		Add(retainedPipeWaitDelay + retainedPipeUpperSlack)
	var result *Result
	select {
	case result = <-resultCh:
	case <-time.After(time.Until(upperDeadline)):
		t.Fatal("Execute exceeded retained-pipe upper bound")
	}
	elapsed := time.Since(time.Unix(0, handoff.Exit.UnixNano))
	var failures []string
	if elapsed < retainedPipeWaitDelay-retainedPipeLowerSlack ||
		elapsed > retainedPipeWaitDelay+retainedPipeUpperSlack {
		failures = append(failures, fmt.Sprintf("return latency %v outside [%v,%v]",
			elapsed, retainedPipeWaitDelay-retainedPipeLowerSlack,
			retainedPipeWaitDelay+retainedPipeUpperSlack))
	}
	if result == nil || result.Error == nil {
		failures = append(failures, fmt.Sprintf("missing retained-output result: %+v", result))
	} else {
		if result.Error.Code != retainedOutputCodeContract {
			failures = append(failures, fmt.Sprintf("error code=%q want=%q",
				result.Error.Code, retainedOutputCodeContract))
		}
		if result.Error.Code == CodeExecutionProcessTreeCleanupFailed {
			failures = append(failures, "successful cleanup mislabeled as cleanup failure")
		}
	}
	if result != nil {
		if result.ExitCode != 0 {
			failures = append(failures, fmt.Sprintf("exit code=%d want=0", result.ExitCode))
		}
		if !result.OutputIncomplete {
			failures = append(failures, "OutputIncomplete=false want=true")
		}
		if result.OutputTruncated {
			failures = append(failures, "retained output incorrectly marked cap-truncated")
		}
	}
	probeText := fmt.Sprintf("retained-pipe-probe pid=%d sequence=%d",
		handoff.PostProbe.PID, handoff.PostProbe.Sequence)
	if result == nil || !strings.Contains(string(result.Stderr), probeText) {
		failures = append(failures, fmt.Sprintf("stderr omits post-exit probe %q", probeText))
	}
	if err := verifier.verifyAllProcessesAbsent(time.Second); err != nil {
		failures = append(failures, "production cleanup: "+err.Error())
	}
	if len(failures) > 0 {
		verifier.verifyWithCleanup()
		emergencyUsed = true
		completed = true
		t.Fatalf("Leamas retained-pipe differential failed:\n  %s",
			strings.Join(failures, "\n  "))
	}
	completed = true
	if emergencyUsed {
		t.Fatal("passing Leamas proof used emergency cleanup")
	}
}

func buildRetainedPipeExecutor(t *testing.T) *Executor {
	t.Helper()
	budget := &Budget{
		Deadline:         time.Now().Add(retainedPipeTimeout),
		MaxConcurrent:    1,
		MaxStarts:        4,
		MaxTaskDepth:     2,
		MaxOutputBytes:   1 << 20,
		TerminationGrace: retainedPipeWaitDelay / 2,
		PostKillWait:     retainedPipeWaitDelay / 2,
	}
	executor, err := NewExecutor(budget, NewTestExecutionRoot())
	if err != nil {
		t.Fatalf("create retained-pipe executor: %v", err)
	}
	return executor
}
