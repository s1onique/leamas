// SPDX-License-Identifier: Apache-2.0

// gate_collector_lifecycle_test.go owns the real
// GateCollector start/wait matrix the CORRECTION08 ACT
// requires.
//
// Every row exercises the production GateCollector +
// production OsRunner pair so the gate's observation-
// failure / post-start outcome handling is observable
// from the GateCapture and the returned error string.
//
// The boundary under test is process lifecycle:
//   - start failure     -> Capture returns error,
//     no GateCapture authority
//   - nonzero exit      -> Capture returns GateCapture,
//     not a start failure
//   - timeout           -> Capture returns GateCapture,
//     TimedOut=true
//   - cancellation      -> Capture returns GateCapture,
//     Canceled=true
//   - WaitDelay         -> NOT a start failure
//
// Every row uses /bin/sh -c <deterministic-script> as
// the production binary so no fake runner is involved.
package evidence

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

// realGateRequest builds a GateCaptureRequest that points
// at the supplied argv. The SubjectRoot and EvidenceDir
// fields are real directories so the GateCollector
// validation passes. The helper is the focused entry
// point the matrix tests share.
func realGateRequest(t *testing.T, argv ...string) (GateCaptureRequest, string) {
	t.Helper()
	subj := t.TempDir()
	ev := t.TempDir()
	return GateCaptureRequest{
		RepositoryRoot: t.TempDir(),
		SubjectRoot:    subj,
		EvidenceDir:    ev,
		RunID:          "real-runner",
		MakeExecutable: argv,
	}, subj
}

// TestGateCollectorRealNonzeroReachesClassifier runs the
// real OsRunner through the real GateCollector with a
// deterministic executable that exits nonzero AFTER
// producing a valid FAILED gate status AND an ACT-owned
// finding on cmd/leamas/**. The test proves:
//
//   - StartErr == nil (the process started)
//   - ExitCode != 0 (nonzero wait outcome)
//   - GateCapture is returned (NOT an error)
//   - ExecGateObservedStatus == "FAILED"
//   - Findings include a path that intersects
//     cmd/leamas/** (so classifier FAIL is independently
//     justified, not relying solely on ExitCode)
//   - ClassifyACTOwnedGate is called literally and
//     returns ACTOwnedFail
//
// The MatrixRunner's argv points at /bin/sh so the test
// is hermetic and deterministic.
func TestGateCollectorRealNonzeroReachesClassifier(t *testing.T) {
	t.Parallel()
	collector := NewGateCollector(&OsRunner{})
	req, _ := realGateRequest(t,
		"/bin/sh", "-c",
		// printf produces a valid FAILED status AND
		// a deliberate new finding on cmd/leamas/main.go
		// (ACT-owned path). The subsequent exit 1 is
		// the nonzero wait outcome.
		"printf 'EXEC_GATE_OBSERVED_STATUS:FAILED\\ncmd/leamas/main.go:42:warning:rule-new:nonzero-lane finding\\n'; exit 1")
	capture, err := collector.Capture(context.Background(), req)
	if err != nil {
		t.Fatalf("Capture must succeed (nonzero is NOT a start failure); got err = %v", err)
	}
	if capture.ExitCode == 0 {
		t.Fatalf("ExitCode = 0, want nonzero")
	}
	// The capture is populated; the classifier must
	// be able to read it.
	if capture.ExecGateObservedStatus == "" {
		t.Fatalf("ExecGateObservedStatus empty, want FAILED")
	}
	if capture.ExecGateObservedStatus != "FAILED" {
		t.Fatalf("ExecGateObservedStatus = %q, want FAILED",
			capture.ExecGateObservedStatus)
	}
	// Classifier inputs are built from the actual
	// capture (NOT a synthetic payload). The ACT-owned
	// path set is cmd/leamas/** so the finding
	// intersects an ACT-owned path; the baseline
	// findings set is empty so the finding is a NEW
	// finding, not an unchanged baseline. Lane flags
	// reflect the capture's lifecycle markers.
	inputs := ClassificationInputs{
		ObservedStatus:   capture.ExecGateObservedStatus,
		ObservedFindings: capture.PreExistingFindings,
		BaselineFindings: nil,
		ACTOwnedPaths:    []string{"cmd/leamas/**"},
		LaneMissing:      capture.ExecGateObservedStatus == "",
		LaneTimedOut:     capture.TimedOut,
		LaneTruncated:    capture.StdoutTruncated || capture.StderrTruncated,
	}
	// Independent classifier verdict (NOT derived from
	// ExitCode alone). The finding's path intersects
	// cmd/leamas/** so the verdict is ACTOwnedFail.
	verdict := ClassifyACTOwnedGate(inputs)
	if verdict != ACTOwnedFail {
		t.Fatalf("ClassifyACTOwnedGate verdict = %q, want ACTOwnedFail "+
			"(status=%q findings=%d owned_paths=cmd/leamas/**)",
			verdict, capture.ExecGateObservedStatus,
			len(capture.PreExistingFindings))
	}
}

// TestGateCollectorRealTimeoutReachesClassifier runs the
// real OsRunner through the real GateCollector with a
// bounded context. The child sleeps longer than the
// deadline. The test proves:
//
//   - StartErr == nil (process started before the deadline)
//   - TimedOut == true (the bounded context fired)
//   - GateCapture is returned (NOT a start failure)
//   - Preconditions: ExecGateObservedStatus == OK,
//     StdoutTruncated == false, StderrTruncated == false
//     (so timeout — not missing status, not truncation,
//     not start failure — owns the UNAVAILABLE verdict)
//   - ClassifyACTOwnedGate is called literally and
//     returns ACTOwnedUnavailable
//
// The MatrixRunner's argv points at /bin/sh -c so the
// test is hermetic and deterministic.
func TestGateCollectorRealTimeoutReachesClassifier(t *testing.T) {
	t.Parallel()
	collector := NewGateCollector(&OsRunner{})
	req, _ := realGateRequest(t,
		"/bin/sh", "-c",
		// Emit a parseable OK status BEFORE sleeping
		// so the captured stdout has a status even on
		// the timed-out path. Then sleep long enough
		// that the bounded ctx fires. The status
		// emission MUST happen first; a missing
		// status would push the verdict to UNAVAILABLE
		// for the wrong reason (LaneMissing) and
		// would NOT prove timeout owns the verdict.
		"printf 'EXEC_GATE_OBSERVED_STATUS:OK\\n'; sleep 5")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	capture, err := collector.Capture(ctx, req)
	if err != nil {
		t.Fatalf("Capture must succeed (timeout is NOT a start failure); got err = %v", err)
	}
	if !capture.TimedOut {
		t.Fatalf("TimedOut = false, want true (bounded context fired)")
	}
	// Preconditions: the OK status was emitted BEFORE
	// the timeout fired, so the captured status is
	// parseable. No truncation, no missing status, no
	// start failure. The ONLY lane flag the classifier
	// sees as true is LaneTimedOut.
	if capture.ExecGateObservedStatus != "OK" {
		t.Fatalf("ExecGateObservedStatus = %q, want OK (must be emitted BEFORE sleep)",
			capture.ExecGateObservedStatus)
	}
	if capture.StdoutTruncated {
		t.Fatalf("StdoutTruncated = true, want false (precondition for timeout-owned verdict)")
	}
	if capture.StderrTruncated {
		t.Fatalf("StderrTruncated = true, want false (precondition for timeout-owned verdict)")
	}
	// Classifier inputs are built from the actual
	// capture. LaneTimedOut is the ONLY true lane flag;
	// the verdict must be ACTOwnedUnavailable because
	// of the timeout, not because of a missing or
	// truncated status.
	inputs := ClassificationInputs{
		ObservedStatus:   capture.ExecGateObservedStatus,
		ObservedFindings: capture.PreExistingFindings,
		BaselineFindings: nil,
		ACTOwnedPaths:    []string{"cmd/leamas/**"},
		LaneMissing:      capture.ExecGateObservedStatus == "",
		LaneTimedOut:     capture.TimedOut,
		LaneTruncated:    capture.StdoutTruncated || capture.StderrTruncated,
	}
	// Independent classifier verdict (NOT derived from
	// TimedOut alone). The timeout itself owns the
	// verdict; the captured status is OK and no
	// truncation occurred.
	verdict := ClassifyACTOwnedGate(inputs)
	if verdict != ACTOwnedUnavailable {
		t.Fatalf("ClassifyACTOwnedGate verdict = %q, want ACTOwnedUnavailable "+
			"(status=%q timed_out=%v truncated=%v)",
			verdict, capture.ExecGateObservedStatus,
			capture.TimedOut,
			capture.StdoutTruncated || capture.StderrTruncated)
	}
}

// TestGateCollectorRealCancellationNotStartFailure runs
// the real OsRunner through the real GateCollector with a
// bounded context that fires AFTER the process has
// started. The test proves:
//
//   - StartErr == nil (the process started)
//   - Canceled == true (the bounded context fired)
//   - GateCapture is returned (NOT a start failure)
//   - The error string is NOT "gate command start"
//
// The MatrixRunner's argv points at /bin/sleep 5 so the
// test is hermetic and deterministic.
func TestGateCollectorRealCancellationNotStartFailure(t *testing.T) {
	t.Parallel()
	collector := NewGateCollector(&OsRunner{})
	req, _ := realGateRequest(t,
		"/bin/sh", "-c",
		// Emit the status BEFORE sleeping so the
		// captured stdout has a parseable status
		// even on the cancelled path. Then sleep
		// long enough that the bound ctx fires.
		"printf 'EXEC_GATE_OBSERVED_STATUS:OK\\n'; sleep 5")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Cancel AFTER the process has started. The 25ms
	// delay ensures cmd.Start() returned successfully
	// before ctx is cancelled.
	go func() {
		time.Sleep(25 * time.Millisecond)
		cancel()
	}()
	capture, err := collector.Capture(ctx, req)
	if err != nil {
		if errors.Is(err, exec.ErrWaitDelay) {
			// Some platforms surface ErrWaitDelay as
			// the captured error; that's a wait
			// outcome, NOT a start failure.
		} else if err != nil && (capture.TimedOut || capture.Canceled) {
			// Post-start outcome — Capture MAY return
			// an error here for the cancelled path.
			// The key invariant: the error string MUST
			// NOT be the start-failure marker.
			if errorsContainsStartFailure(err) {
				t.Fatalf("Capture error = %v, want non-start-failure error", err)
			}
		}
	}
	if !capture.Canceled {
		t.Fatalf("Canceled = false, want true (bounded context fired)")
	}
}

// TestGateCollectorErrWaitDelayNotStartFailure drives
// the retained-pipe WaitDelay pattern through the real
// OsRunner + real GateCollector. The test proves:
//
//   - errors.Is(err, exec.ErrWaitDelay) is NOT surfaced
//     as a "gate command start" failure
//   - The GateCapture is populated when WaitDelay fires
//   - The classifier can still see the populated bytes
//
// The MatrixRunner's argv points at /bin/sh -c so the
// test is hermetic and deterministic.
func TestGateCollectorErrWaitDelayNotStartFailure(t *testing.T) {
	t.Parallel()
	collector := NewGateCollector(&OsRunner{WaitDelay: 200 * time.Millisecond})
	req, _ := realGateRequest(t,
		"/bin/sh", "-c",
		// Emit the status; spawn a detached sleeper
		// that retains an inherited pipe; then exit
		// 0. The wait can return exec.ErrWaitDelay.
		"printf 'EXEC_GATE_OBSERVED_STATUS:OK\\n'; { sleep 5 & } ; exit 0")
	capture, err := collector.Capture(context.Background(), req)
	// On Linux the wait typically succeeds despite
	// the retained pipe. On macOS / some kernels it
	// may return ErrWaitDelay. Either way the
	// GateCapture MUST be populated (NOT a start
	// failure).
	if err != nil {
		if errorsContainsStartFailure(err) {
			t.Fatalf("Capture error = %v, want non-start-failure error", err)
		}
	}
	if capture.ExecGateObservedStatus == "" {
		t.Fatalf("ExecGateObservedStatus empty; GateCapture must be populated")
	}
}

// The umbrella matrix test moved to
// gate_collector_lifecycle_matrix_test.go so this file
// stays under the LLM-friendly 400-line threshold.

// errorsContainsStartFailure reports whether err's
// message looks like the CORRECTION07 start-failure
// marker. The check is substring-only because the
// production GateCollector wraps the StartErr with a
// specific message; the assertion is that the
// substring MUST NOT appear on a post-start outcome.
func errorsContainsStartFailure(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// The CORRECTION07 marker was "evidence: gate
	// command start". The CORRECTION08 marker is the
	// same string (preserved for back-compat with the
	// test). Any error containing it is a
	// start-failure classification.
	return contains(msg, "gate command start")
}

// contains is a tiny no-import substring check so the
// test file does not need the strings package.
func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
