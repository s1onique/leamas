// SPDX-License-Identifier: Apache-2.0

package evidence

import (
	"context"
	"os"
	"reflect"
	"sync"
	"testing"
)

// TestGateCollectorRequestIdentityMatrix exercises the complete declared
// request identity. Only an equal request may receive the cached result.
func TestGateCollectorRequestIdentityMatrix(t *testing.T) {
	rows := []struct {
		name     string
		mutate   func(GateCaptureRequest) GateCaptureRequest
		mismatch bool
	}{
		{name: "same request"},
		{
			name: "different repository root",
			mutate: func(req GateCaptureRequest) GateCaptureRequest {
				req.RepositoryRoot = "/repository-b"
				return req
			},
			mismatch: true,
		},
		{
			name: "different subject root",
			mutate: func(req GateCaptureRequest) GateCaptureRequest {
				req.SubjectRoot = "/subject-b"
				return req
			},
			mismatch: true,
		},
		{
			name: "different evidence directory",
			mutate: func(req GateCaptureRequest) GateCaptureRequest {
				req.EvidenceDir = "/evidence-b"
				return req
			},
			mismatch: true,
		},
		{
			name: "different run id",
			mutate: func(req GateCaptureRequest) GateCaptureRequest {
				req.RunID = "run-b"
				return req
			},
			mismatch: true,
		},
		{
			name: "different executable argv length",
			mutate: func(req GateCaptureRequest) GateCaptureRequest {
				req.MakeExecutable = append(append([]string(nil), req.MakeExecutable...), "extra")
				return req
			},
			mismatch: true,
		},
		{
			name: "different executable argv zero",
			mutate: func(req GateCaptureRequest) GateCaptureRequest {
				req.MakeExecutable = append([]string(nil), req.MakeExecutable...)
				req.MakeExecutable[0] = "/binary-b"
				return req
			},
			mismatch: true,
		},
		{
			name: "different executable argv argument",
			mutate: func(req GateCaptureRequest) GateCaptureRequest {
				req.MakeExecutable = append([]string(nil), req.MakeExecutable...)
				req.MakeExecutable[3] = "--lane=slow"
				return req
			},
			mismatch: true,
		},
		{
			name: "different executable argv order",
			mutate: func(req GateCaptureRequest) GateCaptureRequest {
				req.MakeExecutable = append([]string(nil), req.MakeExecutable...)
				req.MakeExecutable[2], req.MakeExecutable[3] =
					req.MakeExecutable[3], req.MakeExecutable[2]
				return req
			},
			mismatch: true,
		},
	}

	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			runner := &identityRunner{}
			collector := NewGateCollector(runner)
			first := identityRequest(t)
			want, err := collector.Capture(context.Background(), first)
			if err != nil {
				t.Fatalf("first capture: %v", err)
			}
			second := first
			if row.mutate != nil {
				second = row.mutate(second)
			}
			got, err := collector.Capture(context.Background(), second)
			if row.mismatch {
				if !CollectorRequestMismatch(err) {
					t.Fatalf("second error = %v, want request mismatch", err)
				}
			} else {
				if err != nil {
					t.Fatalf("same-request error: %v", err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("cached capture changed: got=%+v want=%+v", got, want)
				}
			}
			assertOneRunnerCall(t, runner, collector)
		})
	}
}

// TestGateCollectorRequestIdentityAfterCachedFailure proves a failed first
// execution remains bound to request A, while request B still gets the typed
// identity mismatch instead of A's cached execution error.
func TestGateCollectorRequestIdentityAfterCachedFailure(t *testing.T) {
	runner := &identityRunner{}
	first := identityRequest(t)
	runner.afterRun = func() { _ = os.RemoveAll(first.EvidenceDir) }
	collector := NewGateCollector(runner)

	_, firstErr := collector.Capture(context.Background(), first)
	if firstErr == nil {
		t.Fatal("first capture must fail after the runner removes the evidence directory")
	}
	_, sameErr := collector.Capture(context.Background(), first)
	if sameErr != firstErr {
		t.Fatalf("same-request error = %v, want cached error %v", sameErr, firstErr)
	}
	second := first
	second.SubjectRoot = "/subject-b"
	_, mismatchErr := collector.Capture(context.Background(), second)
	if !CollectorRequestMismatch(mismatchErr) {
		t.Fatalf("mismatched request error = %v, want request mismatch", mismatchErr)
	}
	assertOneRunnerCall(t, runner, collector)
}

// TestGateCollectorInvalidRequestDoesNotPinIdentity proves validation happens
// before request identity is established.
func TestGateCollectorInvalidRequestDoesNotPinIdentity(t *testing.T) {
	for _, field := range []string{"subject root", "evidence directory"} {
		t.Run(field, func(t *testing.T) {
			runner := &identityRunner{}
			collector := NewGateCollector(runner)
			invalid := identityRequest(t)
			if field == "subject root" {
				invalid.SubjectRoot = ""
			} else {
				invalid.EvidenceDir = ""
			}
			if _, err := collector.Capture(context.Background(), invalid); err == nil {
				t.Fatalf("invalid %s must fail", field)
			}
			if got := runner.Calls(); got != 0 {
				t.Fatalf("runner calls after validation failure = %d, want 0", got)
			}
			if got := collector.Calls(); got != 0 {
				t.Fatalf("collector calls after validation failure = %d, want 0", got)
			}

			valid := identityRequest(t)
			if _, err := collector.Capture(context.Background(), valid); err != nil {
				t.Fatalf("valid capture after invalid request: %v", err)
			}
			assertOneRunnerCall(t, runner, collector)
		})
	}
}

// TestGateCollectorConcurrentSameRequest proves concurrent equivalent calls
// share one execution and one immutable cached result.
func TestGateCollectorConcurrentSameRequest(t *testing.T) {
	runner := &identityRunner{}
	collector := NewGateCollector(runner)
	req := identityRequest(t)
	const callers = 32
	start := make(chan struct{})
	results := make(chan identityCaptureResult, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			<-start
			capture, err := collector.Capture(context.Background(), req)
			results <- identityCaptureResult{capture: capture, err: err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var want GateCapture
	i := 0
	for result := range results {
		if result.err != nil {
			t.Fatalf("caller %d: %v", i, result.err)
		}
		if i == 0 {
			want = result.capture
			i++
			continue
		}
		if !reflect.DeepEqual(result.capture, want) {
			t.Fatalf("caller %d received a different cached capture", i)
		}
		i++
	}
	assertOneRunnerCall(t, runner, collector)
}

// TestGateCollectorConcurrentDifferentRequests proves exactly one identity
// wins the first-request race. Calls using that identity may share the cached
// result; all calls using the other identity must be mismatches.
func TestGateCollectorConcurrentDifferentRequests(t *testing.T) {
	runner := &identityRunner{}
	collector := NewGateCollector(runner)
	first := identityRequest(t)
	second := identityRequest(t)
	const perIdentity = 16
	start := make(chan struct{})
	results := make(chan identityLabeledResult, perIdentity*2)
	var wg sync.WaitGroup
	for label, req := range map[string]GateCaptureRequest{"a": first, "b": second} {
		for i := 0; i < perIdentity; i++ {
			label, req := label, req
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				capture, err := collector.Capture(context.Background(), req)
				results <- identityLabeledResult{label: label, capture: capture, err: err}
			}()
		}
	}
	close(start)
	wg.Wait()
	close(results)

	wins := map[string]int{}
	mismatches := map[string]int{}
	for result := range results {
		switch {
		case result.err == nil:
			wins[result.label]++
		case CollectorRequestMismatch(result.err):
			mismatches[result.label]++
		default:
			t.Fatalf("request %s returned unexpected error: %v", result.label, result.err)
		}
	}
	if len(wins) != 1 {
		t.Fatalf("winning identities = %v, want exactly one", wins)
	}
	winner := "a"
	if wins[winner] == 0 {
		winner = "b"
	}
	loser := "a"
	if winner == loser {
		loser = "b"
	}
	if wins[winner] == 0 || mismatches[loser] != perIdentity {
		t.Fatalf("winner=%s wins=%v mismatches=%v, want winner and %d loser mismatches", winner, wins, mismatches, perIdentity)
	}
	assertOneRunnerCall(t, runner, collector)
}

type identityCaptureResult struct {
	capture GateCapture
	err     error
}

type identityLabeledResult struct {
	label   string
	capture GateCapture
	err     error
}

type identityRunner struct {
	mu       sync.Mutex
	calls    int
	afterRun func()
}

func (r *identityRunner) Run(
	context.Context,
	string,
	[]string,
	string,
	[]string,
) CommandResult {
	r.mu.Lock()
	r.calls++
	afterRun := r.afterRun
	r.mu.Unlock()
	if afterRun != nil {
		afterRun()
	}
	return CommandResult{
		ExitCode: 0,
		Stdout:   []byte("EXEC_GATE_OBSERVED_STATUS:OK\n"),
	}
}

func (r *identityRunner) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func identityRequest(t *testing.T) GateCaptureRequest {
	t.Helper()
	return GateCaptureRequest{
		RepositoryRoot: "/repository-a",
		SubjectRoot:    "/subject-a",
		EvidenceDir:    t.TempDir(),
		RunID:          "run-a",
		MakeExecutable: []string{"/binary-a", "factory", "gate", "--lane=fast"},
	}
}

func assertOneRunnerCall(t *testing.T, runner *identityRunner, collector *GateCollector) {
	t.Helper()
	if got := runner.Calls(); got != 1 {
		t.Fatalf("underlying runner calls = %d, want 1", got)
	}
	if got := collector.Calls(); got != 1 {
		t.Fatalf("collector calls = %d, want 1", got)
	}
}
