//go:build unix || darwin || linux

package execution

import (
	"sort"
	"testing"
	"time"
)

// execOutcome captures the result of an asynchronous Executor.Execute call.
// The outcome is delivered through a buffered channel so callers can drain
// it from readiness-failure paths without blocking on a stuck helper.
type execOutcome struct {
	result *Result
	err    error
}

// readiness-config constants for the SIGTERM escalation proof. The triggers
// are derived from budget values plus explicit scheduler slack so the test
// cannot rely on scheduler timing luck.
const (
	// sigtermRequestTimeout is the request timeout used purely as a fail-safe.
	// It is intentionally larger than the entire expected test lifetime so the
	// signal-driven cancellation is the only intended trigger.
	sigtermRequestTimeout = 30 * time.Second

	// sigtermReadinessWait is the upper bound on waiting for the adversarial
	// child to reach the required state (PID recorded and SignalReady=true).
	// A readiness deadline of 5 seconds comfortably covers process startup
	// jitter on a heavily loaded CI runner while still being small enough to
	// fail fast when readiness is broken.
	sigtermReadinessWait = 5 * time.Second

	// sigtermSlack accounts for scheduler jitter and the SIGKILL-to-Wait
	// return latency the executor guarantees via WaitDelay. It is added on
	// top of TerminationGrace+PostKillWait to compute the post-cancel
	// upper bound.
	sigtermSlack = 500 * time.Millisecond

	// sigtermTestHarnessStabilityBound is an outer fence against a hung
	// executor that has failed to honour cancellation. It is intentionally
	// generous but bounded.
	sigtermTestHarnessStabilityBound = 10 * time.Second
)

// verifyReadinessCleanup cancels the caller context, drains the goroutine
// with a bounded wait, and forces leaked process cleanup. This is invoked
// from readiness-failure paths so a stuck goroutine cannot hide a test
// pass behind a leaked helper process.
//
// cancelCallerFirst MUST run before executor.Close(): the caller's
// cancellation drives the executor's execCtx.Done() select case which
// in turn fires the post-select termination branch that signals the
// descendant's process group. Closing the executor or killing
// recorded processes before the cancel propagates lets the cmd.Wait
// case win the select race and skip termination.
//
// Every cleanup wait remains bounded via a 2-second timer.
func verifyReadinessCleanup(t *testing.T, executor *Executor,
	verifier *processVerifier, resultCh <-chan execOutcome,
	cancelCallerFirst func(),
) {
	t.Helper()

	if cancelCallerFirst != nil {
		cancelCallerFirst()
	}
	// Best-effort: release any semaphore slot the executor still holds.
	if executor != nil {
		_ = executor.Close()
	}
	// Force cleanup of leaked processes from the (possibly partial)
	// manifest.
	if verifier != nil {
		verifier.verifyWithCleanup()
	}
	// Drain the goroutine with a bounded wait so a t.Fatal after a readiness
	// timeout does not leak the helper even if cancellation is unreachable.
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-resultCh:
	case <-timer.C:
	}
}

// requireSharedPGID requires all recorded PIDs to share a single PGID and
// returns the parent and child PGIDs for caller logging.
func requireSharedPGID(t *testing.T, v *processVerifier) (int, int) {
	t.Helper()
	if len(v.records) == 0 {
		t.Fatal("no records to check PGID")
	}
	pgid := v.records[0].PGID
	for _, rec := range v.records[1:] {
		if rec.PGID != pgid {
			t.Errorf("PGID mismatch: first=%d, got=%d (role=%s)",
				pgid, rec.PGID, rec.Role)
		}
	}
	parent := -1
	child := -1
	for _, rec := range v.records {
		switch rec.Role {
		case "parent":
			parent = rec.PGID
		case "child":
			child = rec.PGID
		}
	}
	return parent, child
}

// verifyHelperProcessAlive asserts the most recent record with role is
// alive. It uses kill(pid, signal(0)) so the test does not race with the
// signal-driven cancellation.
func verifyHelperProcessAlive(t *testing.T, v *processVerifier, role string) {
	t.Helper()
	var pid int
	for _, rec := range v.records {
		if rec.Role == role {
			pid = rec.PID
		}
	}
	if pid == 0 {
		t.Errorf("no record with role %q", role)
		return
	}
	alive, err := v.isProcessAlive(pid)
	if err != nil {
		t.Fatalf("isProcessAlive(%d) failed: %v", pid, err)
	}
	if !alive {
		t.Errorf("expected %s (pid=%d) to be alive", role, pid)
	}
}

// allPGIDs returns the sorted list of distinct PGIDs in v.records.
func allPGIDs(v *processVerifier) []int {
	seen := make(map[int]bool)
	for _, rec := range v.records {
		seen[rec.PGID] = true
	}
	out := make([]int, 0, len(seen))
	for pgid := range seen {
		out = append(out, pgid)
	}
	sort.Ints(out)
	return out
}

// errorCodeOrNil returns the ExecutionError.Code or "<nil>" so fatal
// messages remain readable when the result has no error.
func errorCodeOrNil(r *Result) string {
	if r == nil || r.Error == nil {
		return "<nil>"
	}
	return r.Error.Code
}
