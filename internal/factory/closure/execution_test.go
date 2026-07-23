package closure

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

type recordingExecutor struct {
	mu        sync.Mutex
	results   []*execution.Result
	argv      [][]string
	calls     int
	active    int
	maxActive int
}

func (r *recordingExecutor) Execute(_ context.Context, request *execution.Request) *execution.Result {
	r.mu.Lock()
	r.calls++
	r.argv = append(r.argv, append([]string(nil), request.Args...))
	r.active++
	if r.active > r.maxActive {
		r.maxActive = r.active
	}
	index := r.calls - 1
	result := r.results[index]
	r.active--
	r.mu.Unlock()
	return result
}

func runnableChecks(ids ...string) []PlanCheck {
	checks := make([]PlanCheck, 0, len(ids))
	for _, id := range ids {
		checks = append(checks, PlanCheck{ID: id, Mode: CheckModeRun, Argv: []string{"tool", id}, WorkingDirectory: ".", TimeoutSeconds: 10, Environment: map[string]string{}})
	}
	return checks
}

func successExecution(stdout, stderr string) *execution.Result {
	return &execution.Result{ExitCode: 0, Duration: time.Second, Stdout: []byte(stdout), Stderr: []byte(stderr), OutputBytesObserved: int64(len(stdout) + len(stderr)), OutputBytesRetained: int64(len(stdout) + len(stderr))}
}

func executeForTest(t *testing.T, checks []PlanCheck, executor commandExecutor) ([]CheckResult, []EvidenceRecord) {
	t.Helper()
	now := time.Date(2026, 7, 23, 7, 0, 0, 0, time.UTC)
	results, evidence, err := executeChecks(context.Background(), checkExecutionRequest{
		RepositoryRoot:    t.TempDir(),
		EvidenceDirectory: t.TempDir(),
		SubjectTreeOID:    fullTreeOID,
		Checks:            checks,
		Now: func() time.Time {
			now = now.Add(time.Second)
			return now
		},
	}, executor)
	if err != nil {
		t.Fatalf("executeChecks() error = %v", err)
	}
	return results, evidence
}

func TestClosureRunExecutesChecksInPlanOrder(t *testing.T) {
	executor := &recordingExecutor{results: []*execution.Result{successExecution("", ""), successExecution("", "")}}
	results, _ := executeForTest(t, runnableChecks("first", "second"), executor)
	if got := []string{results[0].CheckID, results[1].CheckID}; !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Fatalf("result order = %v", got)
	}
	if got := []string{executor.argv[0][1], executor.argv[1][1]}; !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Fatalf("execution order = %v", got)
	}
}

func TestClosureRunDoesNotRunChecksInParallel(t *testing.T) {
	executor := &recordingExecutor{results: []*execution.Result{successExecution("", ""), successExecution("", "")}}
	executeForTest(t, runnableChecks("first", "second"), executor)
	if executor.maxActive != 1 {
		t.Fatalf("maximum concurrent executions = %d, want 1", executor.maxActive)
	}
}

func TestClosureRunStopsAfterRequiredFailure(t *testing.T) {
	executor := &recordingExecutor{results: []*execution.Result{{ExitCode: 3, Duration: time.Millisecond}}}
	results, _ := executeForTest(t, runnableChecks("fails", "later"), executor)
	if executor.calls != 1 || results[0].Status != CheckStatusFail {
		t.Fatalf("calls=%d results=%+v", executor.calls, results)
	}
}

func TestClosureRunMarksRemainingChecksNotRun(t *testing.T) {
	executor := &recordingExecutor{results: []*execution.Result{{ExitCode: 1}}}
	results, _ := executeForTest(t, runnableChecks("fails", "later"), executor)
	if results[1].Status != CheckStatusNotRun || results[1].CleanupStatus != CleanupNotRequired {
		t.Fatalf("later result = %+v", results[1])
	}
}

func TestClosureRunDoesNotRetry(t *testing.T) {
	executor := &recordingExecutor{results: []*execution.Result{{ExitCode: 1}}}
	executeForTest(t, runnableChecks("fails"), executor)
	if executor.calls != 1 {
		t.Fatalf("calls = %d, want 1", executor.calls)
	}
}

func TestClosureRunPreservesStdoutAndStderrSeparation(t *testing.T) {
	executor := &recordingExecutor{results: []*execution.Result{successExecution("out", "err")}}
	results, evidence := executeForTest(t, runnableChecks("separate"), executor)
	if results[0].StdoutSHA256 == results[0].StderrSHA256 || len(evidence) != 2 {
		t.Fatalf("results=%+v evidence=%+v", results[0], evidence)
	}
}

func TestClosureRunRecordsTruncatedAndIncompleteSeparately(t *testing.T) {
	executor := &recordingExecutor{results: []*execution.Result{{ExitCode: -1, OutputTruncated: true, OutputIncomplete: false, Error: execution.ErrOutputLimitExceeded(1, 2, "", "tool")}}}
	results, _ := executeForTest(t, runnableChecks("bounded"), executor)
	if !results[0].OutputTruncated || results[0].OutputIncomplete {
		t.Fatalf("result = %+v", results[0])
	}
}

func TestClosureRunPropagatesProcessCleanupFailure(t *testing.T) {
	executor := &recordingExecutor{results: []*execution.Result{{ExitCode: -1, Error: execution.ErrProcessTreeCleanupFailed(123, "still alive")}}}
	results, _ := executeForTest(t, runnableChecks("cleanup"), executor)
	if results[0].CleanupStatus != CleanupFailed || results[0].ExecutionErrorCode != execution.CodeExecutionProcessTreeCleanupFailed {
		t.Fatalf("result = %+v", results[0])
	}
}

func TestClosureDetachedEvidenceFilesContainCapturedBytes(t *testing.T) {
	root := t.TempDir()
	evidenceDir := t.TempDir()
	executor := &recordingExecutor{results: []*execution.Result{successExecution("stdout", "stderr")}}
	_, _, err := executeChecks(context.Background(), checkExecutionRequest{RepositoryRoot: root, EvidenceDirectory: evidenceDir, SubjectTreeOID: fullTreeOID, Checks: runnableChecks("proof"), Now: time.Now}, executor)
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := os.ReadFile(filepath.Join(evidenceDir, "proof.stdout"))
	if err != nil || string(stdout) != "stdout" {
		t.Fatalf("stdout file = %q, %v", stdout, err)
	}
	stderr, err := os.ReadFile(filepath.Join(evidenceDir, "proof.stderr"))
	if err != nil || string(stderr) != "stderr" {
		t.Fatalf("stderr file = %q, %v", stderr, err)
	}
}
