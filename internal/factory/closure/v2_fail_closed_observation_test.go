// SPDX-License-Identifier: Apache-2.0

package closure

// v2_fail_closed_observation_test.go exercises the fail-closed
// observation invariants required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1.
//
// Each test injects a fake git client whose Run returns a
// failure for exactly one of the four observation commands
// (HEAD^{commit}, HEAD^{tree}, status --porcelain=v2,
// worktree list --porcelain). The runner must reject
// execution BEFORE running any check when the BEFORE snapshot
// fails, and must reject the AFTER snapshot before claiming
// clean success.
//
// Splitting this from v2_lifecycle_invariants_test.go keeps
// every file under the LLM-friendly 400-line threshold.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// v2FailGitClient routes Run() calls to the real client unless
// the command matches one of the configured fail hooks. Each
// hook, when matched, returns a non-zero exit and a stderr
// that classifies as a generic git_operation_failed.
//
// The fake is intentionally narrow: it only fails the exact
// command the test targets. Every other command delegates to
// the supplied RealGit so the runner can still construct a
// real hermetic repository.
// v2FailGitClient embeds RealGit so it inherits all four
// gitClient methods, then layers fail-after-argv hooks on top
// of Run / RunWithEnv. The remaining two methods (RunWithStdin
// and RunWithStdinAndEnv) are inherited verbatim from RealGit.
type v2FailGitClient struct {
	RealGit
	Real        gitClient
	FailOn      [][]string // exact git argv sequences to fail
	Err         error
	Stderr      string
	ExitCode    int
	invocations int
	failedCalls int
}

func (g *v2FailGitClient) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	g.invocations++
	for _, fail := range g.FailOn {
		if argvEquals(fail, args) {
			g.failedCalls++
			return gitCommandResult{
				Stderr:   []byte(g.Stderr),
				ExitCode: g.ExitCode,
				Err:      g.Err,
			}
		}
	}
	return g.Real.Run(ctx, directory, args...)
}

// RunWithEnv routes RunWithEnv calls through the same hooks
// so callers that go through the env-overload path still hit
// the configured failure.
func (g *v2FailGitClient) RunWithEnv(ctx context.Context, directory string, env []string, args ...string) gitCommandResult {
	g.invocations++
	for _, fail := range g.FailOn {
		if argvEquals(fail, args) {
			g.failedCalls++
			return gitCommandResult{
				Stderr:   []byte(g.Stderr),
				ExitCode: g.ExitCode,
				Err:      g.Err,
			}
		}
	}
	return g.Real.RunWithEnv(ctx, directory, env, args...)
}

func argvEquals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// v2FailClosedRunnerRequest builds the standard S < F request
// for fail-closed tests.
func v2FailClosedRunnerRequest(t *testing.T) V2Request {
	t.Helper()
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	frozenBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-FAILCLOSED",
		subject, subjectTree, v2FixtureCheck{
			ID:               "noop",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/FAILCLOSED.json": string(frozenBytes),
	})
	return V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/FAILCLOSED.json",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	}
}

// TestV2Runner_RejectBeforeHeadLookupFailure asserts the runner
// rejects execution when `git rev-parse HEAD^{commit}` fails
// in the BEFORE snapshot.
func TestV2Runner_RejectBeforeHeadLookupFailure(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	fake := &v2FailGitClient{
		Real:     RealGit{},
		FailOn:   [][]string{{"rev-parse", "HEAD^{commit}"}},
		Err:      errors.New("exit status 128"),
		Stderr:   "fatal: ambiguous HEAD",
		ExitCode: 128,
	}
	deps := DefaultV2RunnerDeps()
	deps.Git = fake
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("before HEAD failure must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerStateUnavailable) {
		t.Fatalf("expected caller_state_unavailable, got %v", v2err.Diags.Codes())
	}
	if _, err := os.Stat(req.ManifestOutput); err == nil {
		t.Fatalf("manifest must not be written on before HEAD failure")
	}
}

// TestV2Runner_RejectBeforeStatusFailure asserts the runner
// rejects execution when `git status --porcelain=v2` fails in
// the BEFORE snapshot.
func TestV2Runner_RejectBeforeStatusFailure(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	fake := &v2FailGitClient{
		Real:     RealGit{},
		FailOn:   [][]string{{"status", "--porcelain=v2", "--untracked-files=all"}},
		Err:      errors.New("exit status 128"),
		Stderr:   "fatal: bad status invocation",
		ExitCode: 128,
	}
	deps := DefaultV2RunnerDeps()
	deps.Git = fake
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("before status failure must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerStateUnavailable) {
		t.Fatalf("expected caller_state_unavailable, got %v", v2err.Diags.Codes())
	}
}

// TestV2Runner_RejectBeforeWorktreeListFailure asserts the
// runner rejects execution when `git worktree list --porcelain`
// fails in the BEFORE snapshot.
func TestV2Runner_RejectBeforeWorktreeListFailure(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	fake := &v2FailGitClient{
		Real:     RealGit{},
		FailOn:   [][]string{{"worktree", "list", "--porcelain"}},
		Err:      errors.New("exit status 128"),
		Stderr:   "fatal: cannot list worktrees",
		ExitCode: 128,
	}
	deps := DefaultV2RunnerDeps()
	deps.Git = fake
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("before worktree-list failure must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeWorktreeInventoryUnavailable) {
		t.Fatalf("expected worktree_inventory_unavailable, got %v", v2err.Diags.Codes())
	}
	if !v2err.Diags.HasCode(V2CodeCallerStateUnavailable) {
		t.Fatalf("expected caller_state_unavailable propagated, got %v", v2err.Diags.Codes())
	}
}

// countingFail is one fail-after-N-calls rule for the
// countingGitClient fake. After the Nth matching call, every
// subsequent matching call fails.
type countingFail struct {
	AfterNCalls int
	Argv        []string
	Err         error
	Stderr      string
	ExitCode    int
}

// countingGitClient is a stateful fake that fails after a
// configured number of invocations of a specific command.
// Useful for AFTER-snapshot failures: the before snapshot
// succeeds, the executor runs, then the after snapshot fails.
// countingGitClient embeds RealGit so it inherits all four
// gitClient methods, then layers a fail-after-N-calls hook on
// top of Run.
type countingGitClient struct {
	RealGit
	Real      gitClient
	FailAfter []countingFail
	OnCall    func(args []string)
	counters  map[string]int
}

func (g *countingGitClient) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	if g.OnCall != nil {
		g.OnCall(args)
	}
	if g.counters == nil {
		g.counters = map[string]int{}
	}
	key := strings.Join(args, "\x00")
	g.counters[key]++
	for _, f := range g.FailAfter {
		if argvEquals(f.Argv, args) && g.counters[key] > f.AfterNCalls {
			return gitCommandResult{
				Stderr:   []byte(f.Stderr),
				ExitCode: f.ExitCode,
				Err:      f.Err,
			}
		}
	}
	return g.Real.Run(ctx, directory, args...)
}

// TestV2Runner_RejectAfterHeadLookupFailure asserts the runner
// refuses to claim clean success when the AFTER HEAD lookup
// fails. The before snapshot succeeds; the executor runs; the
// after snapshot fails and the runner surfaces
// V2CodeCallerStateUnavailable without publishing a manifest.
func TestV2Runner_RejectAfterHeadLookupFailure(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	fake := &countingGitClient{
		Real: RealGit{},
		FailAfter: []countingFail{
			{AfterNCalls: 1, Argv: []string{"rev-parse", "HEAD^{commit}"},
				Err: errors.New("exit status 128"), Stderr: "fatal: bad head", ExitCode: 128},
		},
	}
	deps := DefaultV2RunnerDeps()
	deps.Git = fake
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after HEAD failure must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerStateUnavailable) {
		t.Fatalf("expected caller_state_unavailable, got %v", v2err.Diags.Codes())
	}
	if _, err := os.Stat(req.ManifestOutput); err == nil {
		t.Fatalf("manifest must not be written on after HEAD failure")
	}
}

// TestV2Runner_RejectAfterStatusFailure asserts the runner
// refuses to claim clean success when the AFTER status fails.
func TestV2Runner_RejectAfterStatusFailure(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	fake := &countingGitClient{
		Real: RealGit{},
		FailAfter: []countingFail{
			{AfterNCalls: 1, Argv: []string{"status", "--porcelain=v2", "--untracked-files=all"},
				Err: errors.New("exit status 128"), Stderr: "fatal: bad status", ExitCode: 128},
		},
	}
	deps := DefaultV2RunnerDeps()
	deps.Git = fake
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after status failure must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerStateUnavailable) {
		t.Fatalf("expected caller_state_unavailable, got %v", v2err.Diags.Codes())
	}
}

// TestV2Runner_RejectAfterWorktreeListFailure asserts the
// runner refuses to claim clean success when the AFTER
// worktree list fails.
func TestV2Runner_RejectAfterWorktreeListFailure(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	fake := &countingGitClient{
		Real: RealGit{},
		FailAfter: []countingFail{
			{AfterNCalls: 1, Argv: []string{"worktree", "list", "--porcelain"},
				Err: errors.New("exit status 128"), Stderr: "fatal: cannot list", ExitCode: 128},
		},
	}
	deps := DefaultV2RunnerDeps()
	deps.Git = fake
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after worktree-list failure must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerStateUnavailable) {
		t.Fatalf("expected caller_state_unavailable propagated, got %v", v2err.Diags.Codes())
	}
}
