// SPDX-License-Identifier: Apache-2.0

package closure

// closure_runtime_context_umbrellas_test.go provides the
// required umbrellas for
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02-A-R1-FINAL.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution"
	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// worktreeObservingGit is a tiny package-private test seam
// that wraps the production gitClient and records the
// actual worktree authority observed during the production
// executor's run. The seam exists so
// TestClosureSubjectWorktreeAuthority can prove the worktree
// authority against the ACTUAL path the production
// executor created, not a path the test synthesised from a
// separate `git worktree add` call.
//
// The production executor removes the worktree before
// returning, so the test cannot re-run `git rev-parse HEAD`
// or `git symbolic-ref -q HEAD` against the captured path
// after the executor returns. The wrapper therefore
// captures the worktree HEAD, HEAD^{tree}, and symbolic-ref
// state through the production executor's own git command
// stream plus a one-shot capture invoked just before the
// executor's `git worktree remove --force` cleanup matches
// the captured path. The capture uses the inner git client
// so the assertions remain grounded in real git output.
type worktreeObservingGit struct {
	inner gitClient

	mu                sync.Mutex
	detachedAddPath   string
	headAt            string
	headTreeAt        string
	symbolicRefExit   int
	symbolicRefStdout string
	worktreeListSeen  bool
}

func newWorktreeObservingGit(inner gitClient) *worktreeObservingGit {
	if inner == nil {
		inner = RealGit{}
	}
	return &worktreeObservingGit{inner: inner}
}

func (g *worktreeObservingGit) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	// Capture the worktree authority BEFORE delegating to
	// the actual git invocation. The capture runs only
	// when the executor is about to remove the captured
	// worktree, which is the last authoritative observable
	// moment before the path is gone.
	g.mu.Lock()
	capturePath := g.detachedAddPath
	g.mu.Unlock()
	isRemoveCall := len(args) >= 4 && args[0] == "worktree" && args[1] == "remove" && args[2] == "--force"
	if isRemoveCall && capturePath != "" && args[3] == capturePath {
		g.captureWorktreeAuthority(ctx, capturePath)
	}
	res := g.inner.Run(ctx, directory, args...)
	g.mu.Lock()
	defer g.mu.Unlock()
	// The production executor's call is exactly:
	//   git worktree add --detach <worktreePath> <subjectCommit>
	if len(args) >= 4 && args[0] == "worktree" && args[1] == "add" && args[2] == "--detach" {
		g.detachedAddPath = args[3]
	}
	// Bound subsequent git invocations on the captured
	// worktree path so the test can assert the executor
	// saw the worktree as the production authority.
	if g.detachedAddPath != "" && directory == g.detachedAddPath {
		if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "HEAD" {
			g.headAt = strings.TrimSpace(string(res.Stdout))
		}
		if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "HEAD^{tree}" {
			g.headTreeAt = strings.TrimSpace(string(res.Stdout))
		}
		if len(args) >= 3 && args[0] == "symbolic-ref" && args[1] == "-q" && args[2] == "HEAD" {
			g.symbolicRefExit = res.ExitCode
			g.symbolicRefStdout = string(res.Stdout)
		}
	}
	if len(args) >= 1 && args[0] == "worktree" && len(args) >= 2 && args[1] == "list" {
		g.worktreeListSeen = true
	}
	return res
}

// captureWorktreeAuthority runs the three bounded-git
// authority checks against the captured worktree path. The
// capture is invoked once, just before the production
// executor's `git worktree remove --force` cleanup matches
// the captured path, so the assertions remain grounded in
// real git output observed against the path the production
// executor actually created.
func (g *worktreeObservingGit) captureWorktreeAuthority(ctx context.Context, worktreePath string) {
	headRes := g.inner.Run(ctx, worktreePath, "rev-parse", "HEAD")
	g.mu.Lock()
	g.headAt = strings.TrimSpace(string(headRes.Stdout))
	g.mu.Unlock()
	treeRes := g.inner.Run(ctx, worktreePath, "rev-parse", "HEAD^{tree}")
	g.mu.Lock()
	g.headTreeAt = strings.TrimSpace(string(treeRes.Stdout))
	g.mu.Unlock()
	symRes := g.inner.Run(ctx, worktreePath, "symbolic-ref", "-q", "HEAD")
	g.mu.Lock()
	g.symbolicRefExit = symRes.ExitCode
	g.symbolicRefStdout = string(symRes.Stdout)
	g.mu.Unlock()
}

func (g *worktreeObservingGit) RunWithStdin(ctx context.Context, directory, stdin string, args ...string) gitCommandResult {
	return g.inner.RunWithStdin(ctx, directory, stdin, args...)
}

func (g *worktreeObservingGit) RunWithEnv(ctx context.Context, directory string, env []string, args ...string) gitCommandResult {
	return g.inner.RunWithEnv(ctx, directory, env, args...)
}

func (g *worktreeObservingGit) RunWithStdinAndEnv(ctx context.Context, directory, stdin string, env []string, args ...string) gitCommandResult {
	return g.inner.RunWithStdinAndEnv(ctx, directory, stdin, env, args...)
}

func (g *worktreeObservingGit) snapshot() (worktreeRoot, headAt, headTreeAt string, symRefExit int, symRefStdout string, worktreeListSeen bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.detachedAddPath, g.headAt, g.headTreeAt, g.symbolicRefExit, g.symbolicRefStdout, g.worktreeListSeen
}

func TestClosureSubjectWorktreeAuthority(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject-only\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	// Capture the ACTUAL worktree authority the production
	// executor sees during its run. The wrapper records
	// the worktree path, the HEAD and HEAD^{tree} outputs,
	// and the symbolic-ref exit before the executor cleans
	// up the worktree.
	obs := newWorktreeObservingGit(RealGit{})
	executor := NewGitV2SubjectExecutor(obs)
	req := V2ExecuteRequest{
		RepositoryRoot: dir,
		SubjectCommit:  subject,
		SubjectTree:    subjectTree,
		EvidenceDir:    t.TempDir(),
		Checks: []PlanCheck{
			{ID: "noop", Mode: "run", Argv: []string{"true"}, WorkingDirectory: ".", TimeoutSeconds: 5},
		},
		CommandExecutor: boundedCommandExecutor{},
	}
	res, err := executor.ExecuteSubjectChecks(context.Background(), req)
	if err != nil {
		t.Fatalf("subject worktree execution failed: %v", err)
	}
	if res.ObservedTree != subjectTree {
		t.Fatalf("execution tree %s != subject tree %s", res.ObservedTree, subjectTree)
	}
	worktreeRoot, headAt, headTreeAt, symRefExit, symRefStdout, _ := obs.snapshot()
	// 1. The production executor must have observed exactly
	// one `git worktree add --detach` call whose target
	// path is the captured worktree root.
	if worktreeRoot == "" {
		t.Fatalf("production executor did not invoke `git worktree add --detach`")
	}
	if !filepath.IsAbs(worktreeRoot) {
		t.Fatalf("worktree root must be absolute, got %q", worktreeRoot)
	}
	// 2. The captured worktree root must be OUTSIDE the
	// caller repository so the runner never mutates the
	// caller's checkout.
	if strings.HasPrefix(worktreeRoot, dir+string(filepath.Separator)) || worktreeRoot == dir {
		t.Fatalf("worktree root %q must be outside caller repository %q", worktreeRoot, dir)
	}
	// 3. The captured worktree's HEAD must equal the
	// subject commit and its HEAD^{tree} must equal the
	// subject tree. The values come from the git commands
	// the production executor ran on the captured path,
	// not from a separate worktree add issued by the test.
	if headAt != subject {
		t.Fatalf("worktree HEAD %s != subject commit %s", headAt, subject)
	}
	if headTreeAt != subjectTree {
		t.Fatalf("worktree HEAD^{tree} %s != subject tree %s", headTreeAt, subjectTree)
	}
	// 4. The captured worktree must be detached: a
	// `git symbolic-ref -q HEAD` invocation must exit
	// with status 1 exactly. Git documents that
	// symbolic-ref -q returns 1 when HEAD is not a
	// symbolic ref (the detached-HEAD case). Any other
	// exit code (including 128) indicates a different
	// failure mode, so the assertion is exact.
	if symRefExit != 1 {
		t.Fatalf("worktree HEAD must be detached; symbolic-ref exit=%d stdout=%q",
			symRefExit, symRefStdout)
	}
	// 5. After ExecuteSubjectChecks returns, the captured
	// worktree must no longer appear in the porcelain
	// inventory. The execution path is the only authority
	// that may write to the worktree list, and the runner
	// must have removed its registration. Git documents
	// `git worktree list --porcelain -z` as the stable
	// machine-readable format.
	listed := mustRunGit(t, dir, "worktree", "list", "--porcelain", "-z")
	if bytes.Contains([]byte(listed), []byte(worktreeRoot)) {
		t.Fatalf("subject worktree %q leaked after cleanup; porcelain=%q", worktreeRoot, listed)
	}
}

func TestClosureCheckResultBijection(t *testing.T) {
	t.Parallel()
	// Build a fully valid candidate so DeriveClosureEvidenceCompleteness
	// returns EvidenceComplete. The B2 canonical predicate is a closed
	// AND of all 43 authorities; the test mutates only the plan/result
	// bijection invariants.
	planBytes := []byte("{\"contract_version\":1,\"checks\":[]}")
	planSum := sha256.Sum256(planBytes)
	planSHA := hex.EncodeToString(planSum[:])
	subjectCommit := "3333333333333333333333333333333333333333"
	subjectTree := "4444444444444444444444444444444444444444"
	build := func(results []evidence.CheckResult) evidence.ClosureEvidence {
		return evidence.ClosureEvidence{
			SchemaVersion: evidence.ClosureEvidenceSchemaVersion,
			Protocol:      evidence.ClosureProtocolVersion,
			Runtime: evidence.RuntimeAuthority{
				RepositoryRoot:      "/repo",
				FreezeCommit:        "1111111111111111111111111111111111111111",
				FreezeTree:          "2222222222222222222222222222222222222222",
				SubjectCommit:       subjectCommit,
				SubjectTree:         subjectTree,
				ExecutionTree:       subjectTree,
				PlanPath:            "plan",
				PlanBlob:            "5555555555555555555555555555555555555555",
				PlanSHA256:          planSHA,
				PlanBytes:           planBytes,
				FAncestorOfSVerified: true,
			},
			Plan: evidence.PlanAuthority{
				ExpectedChecks: []evidence.PlanCheckSpec{
					{ID: "c1", Mode: "run"},
					{ID: "c2", Mode: "run"},
					{ID: "c3", Mode: "exclude"},
				},
			},
			Results: results,
			Gate: evidence.GateAuthority{
				ObservedStatus:  "OK",
				Classification:  "PASS",
				InvocationCount: 1,
				RepositoryRoot:  "/repo",
				SubjectRoot:     "/subject",
			},
			Binary: evidence.BinaryAuthority{
				BinaryPath:                "/tmp/leamas",
				BinarySHA256:              planSHA,
				BinaryCommit:              subjectCommit,
				BinaryModified:            false,
				SourceCommit:              subjectCommit,
				SourceTree:                subjectTree,
				SourceClean:               true,
				SourceDetached:            true,
				OutputOutsideAllWorktrees: true,
				Executable:                true,
			},
			CallerBefore: evidence.CallerStateSnapshot{
				Available:             true,
				Head:                  subjectCommit,
				Tree:                  subjectTree,
				StatusHash:            "status",
				RefsHash:              "refs",
				WorktreeInventoryHash: "wt",
			},
			CallerAfter: evidence.CallerStateSnapshot{
				Available:             true,
				Head:                  subjectCommit,
				Tree:                  subjectTree,
				StatusHash:            "status",
				RefsHash:              "refs",
				WorktreeInventoryHash: "wt",
			},
		}
	}
	good := []evidence.CheckResult{
		{CheckID: "c1", Mode: "run", Outcome: "pass"},
		{CheckID: "c2", Mode: "run", Outcome: "pass"},
		{CheckID: "c3", Mode: "exclude", Outcome: "excluded"},
	}
	env := build(good)
	if got := evidence.DeriveClosureEvidenceCompleteness(env); got != evidence.EvidenceComplete {
		t.Fatalf("valid bijection must pass: %+v", env)
	}
	badOrder := build([]evidence.CheckResult{
		{CheckID: "c2", Mode: "run", Outcome: "pass"},
		{CheckID: "c1", Mode: "run", Outcome: "pass"},
		{CheckID: "c3", Mode: "exclude", Outcome: "excluded"},
	})
	if got := evidence.DeriveClosureEvidenceCompleteness(badOrder); got != evidence.EvidenceIncomplete {
		t.Fatalf("out-of-order results must reject")
	}
	badUnknown := build([]evidence.CheckResult{
		{CheckID: "c1", Mode: "run", Outcome: "pass"},
		{CheckID: "c2", Mode: "run", Outcome: "pass"},
		{CheckID: "ghost", Mode: "exclude", Outcome: "excluded"},
	})
	if got := evidence.DeriveClosureEvidenceCompleteness(badUnknown); got != evidence.EvidenceIncomplete {
		t.Fatalf("unknown ID must reject")
	}
	badDup := build([]evidence.CheckResult{
		{CheckID: "c1", Mode: "run", Outcome: "pass"},
		{CheckID: "c1", Mode: "run", Outcome: "pass"},
		{CheckID: "c3", Mode: "exclude", Outcome: "excluded"},
	})
	if got := evidence.DeriveClosureEvidenceCompleteness(badDup); got != evidence.EvidenceIncomplete {
		t.Fatalf("duplicate ID must reject")
	}
	// Empty plan: build a candidate whose only mutation is the missing
	// expected-checks list. The closed-AND predicate must reject.
	empty := build(good)
	empty.Plan.ExpectedChecks = nil
	if got := evidence.DeriveClosureEvidenceCompleteness(empty); got != evidence.EvidenceIncomplete {
		t.Fatalf("empty plan must reject")
	}
}

func TestClosureBoundedExecutionMatrix(t *testing.T) {
	t.Parallel()
	exec := boundedCommandExecutor{}
	r := exec.Execute(context.Background(), &execution.Request{
		Name: "true", Args: []string{"true"}, Timeout: 5 * time.Second, OutputCap: 1024,
	})
	if r.Error != nil || r.ExitCode != 0 || r.OutputTruncated {
		t.Fatalf("success: err=%v exit=%d trunc=%v", r.Error, r.ExitCode, r.OutputTruncated)
	}
	r = exec.Execute(context.Background(), &execution.Request{
		Name: "false", Args: []string{"false"}, Timeout: 5 * time.Second, OutputCap: 1024,
	})
	if r.ExitCode == 0 || r.Error != nil {
		t.Fatalf("nonzero: err=%v exit=%d", r.Error, r.ExitCode)
	}
	cap := int64(1024)
	long := strings.Repeat("a", int(cap))
	r = exec.Execute(context.Background(), &execution.Request{
		Name: "printf", Args: []string{"printf", long}, Timeout: 5 * time.Second, OutputCap: cap,
	})
	if int(r.OutputBytesRetained) != int(cap) || r.OutputTruncated {
		t.Fatalf("stdout cap: retained=%d truncated=%v", r.OutputBytesRetained, r.OutputTruncated)
	}
	tooLong := strings.Repeat("a", int(cap)+1)
	r = exec.Execute(context.Background(), &execution.Request{
		Name: "printf", Args: []string{"printf", tooLong}, Timeout: 5 * time.Second, OutputCap: cap,
	})
	if !r.OutputTruncated {
		t.Fatalf("stdout cap+1 must be truncated")
	}
	stderr := strings.Repeat("e", int(cap))
	r = exec.Execute(context.Background(), &execution.Request{
		Name: "sh", Args: []string{"sh", "-c", "printf \"$1\" >&2", "sh", stderr}, Timeout: 5 * time.Second, OutputCap: cap,
	})
	if int(r.OutputBytesRetained) != int(cap) || r.OutputTruncated {
		t.Fatalf("stderr cap: retained=%d truncated=%v", r.OutputBytesRetained, r.OutputTruncated)
	}
	stderrTooLong := strings.Repeat("e", int(cap)+1)
	r = exec.Execute(context.Background(), &execution.Request{
		Name: "sh", Args: []string{"sh", "-c", "printf \"$1\" >&2", "sh", stderrTooLong}, Timeout: 5 * time.Second, OutputCap: cap,
	})
	if !r.OutputTruncated {
		t.Fatalf("stderr cap+1 must be truncated")
	}
	stdoutPayload := strings.Repeat("a", int(cap)+100)
	stderrPayload := strings.Repeat("e", int(cap)+100)
	r = exec.Execute(context.Background(), &execution.Request{
		Name: "sh", Args: []string{"sh", "-c",
			"printf \"$1\" & printf \"$2\" >&2 & wait", "sh", stdoutPayload, stderrPayload,
		}, Timeout: 5 * time.Second, OutputCap: cap,
	})
	if !r.OutputTruncated {
		t.Fatalf("simultaneous overflow must be truncated")
	}
	// Timeout classification. The production bounded
	// executor surfaces a *ExecutionError with the
	// CodeExecutionDeadlineExceeded code; the underlying
	// context.DeadlineExceeded is NOT exposed via Unwrap()
	// because the sentinel is constructed with a nil Cause.
	// The test asserts the production typed classification
	// rather than reaching through an opaque wrapper.
	r = exec.Execute(context.Background(), &execution.Request{
		Name: "sleep", Args: []string{"sleep", "10"}, Timeout: 100 * time.Millisecond, OutputCap: 1024,
	})
	if r.Error == nil {
		t.Fatalf("timeout: error must be non-nil")
	}
	if r.Error.Code != execution.CodeExecutionDeadlineExceeded {
		t.Fatalf("timeout: code=%s, expected %s (full error: %v)",
			r.Error.Code, execution.CodeExecutionDeadlineExceeded, r.Error)
	}
	// Caller cancellation classification. As with timeout,
	// the bounded executor converts context.Canceled into a
	// typed CodeExecutionCancelled sentinel with a nil
	// Cause, so errors.Is cannot reach context.Canceled
	// through the wrapper. The test asserts the typed
	// classification.
	cancelCtx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	cancelR := exec.Execute(cancelCtx, &execution.Request{
		Name: "sleep", Args: []string{"sleep", "10"}, Timeout: 10 * time.Second, OutputCap: 1024,
	})
	if cancelR.Error == nil {
		t.Fatalf("cancel: error must be non-nil")
	}
	if cancelR.Error.Code != execution.CodeExecutionCancelled {
		t.Fatalf("cancel: code=%s, expected %s (full error: %v)",
			cancelR.Error.Code, execution.CodeExecutionCancelled, cancelR.Error)
	}
	// Timeout and cancellation must remain distinct: the
	// cancellation row must NOT be reported as a deadline
	// exceeded error, and vice versa.
	if cancelR.Error.Code == execution.CodeExecutionDeadlineExceeded {
		t.Fatalf("cancel must not be classified as a deadline-exceeded error")
	}
	if r.Error.Code == execution.CodeExecutionCancelled {
		t.Fatalf("timeout must not be classified as a cancellation error")
	}
	// Spawn failure classification. The bounded executor
	// surfaces a typed *ExecutionError with the
	// CodeExecutionCommandNotFound code when the binary
	// cannot be located on PATH.
	r = exec.Execute(context.Background(), &execution.Request{
		Name: "no-such-binary-exists-12345", Args: []string{"no-such-binary-exists-12345"}, Timeout: 5 * time.Second, OutputCap: 1024,
	})
	if r == nil {
		t.Fatalf("spawn failure: nil result")
	}
	if r.Error == nil {
		t.Fatalf("spawn failure: error must be non-nil")
	}
	if r.Error.Code != execution.CodeExecutionCommandNotFound {
		t.Fatalf("spawn failure: code=%s, expected %s",
			r.Error.Code, execution.CodeExecutionCommandNotFound)
	}
}

func TestClosureWaitDelayRetainedPipe(t *testing.T) {
	t.Parallel()
	exec := boundedCommandExecutor{}
	start := time.Now()
	r := exec.Execute(context.Background(), &execution.Request{
		Name: "sh", Args: []string{"sh", "-c",
			"/bin/sh -c 'sleep 30' & exit 0",
		}, Timeout: 30 * time.Second, OutputCap: 1024,
	})
	elapsed := time.Since(start)
	if r == nil {
		t.Fatalf("retained pipe run returned nil")
	}
	// The bounded executor preserves the osexec.ErrWaitDelay
	// identity through Unwrap because
	// ErrRetainedOutputPipe(err) sets the original Wait()
	// error as the ExecutionError Cause. The assertion
	// therefore reaches the typed sentinel via errors.Is,
	// which is the only OID-surfaced error in the bounded
	// executor that preserves its underlying identity.
	if !errors.Is(r.Error, osexec.ErrWaitDelay) {
		t.Fatalf("retained pipe: err=%v (expected osexec.ErrWaitDelay)", r.Error)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("retained pipe drain exceeded WaitDelay envelope: %v", elapsed)
	}
}

func TestClosureGateRunsAgainstSubject(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	worktreePath := filepath.Join(t.TempDir(), "subject-wt")
	mustRunGit(t, dir, "worktree", "add", "--detach", worktreePath, subject)
	defer mustRunGit(t, dir, "worktree", "remove", "--force", worktreePath)
	var seenCwd string
	runner := evidenceRecorderRunner{
		onRun: func(dir string) { seenCwd = dir },
	}
	collector := evidence.NewGateCollector(&runner)
	capture, err := collector.Capture(context.Background(), evidence.GateCaptureRequest{
		RepositoryRoot: dir,
		SubjectRoot:    worktreePath,
		EvidenceDir:    t.TempDir(),
		RunID:          "test",
	})
	if err != nil {
		t.Fatalf("gate capture: %v", err)
	}
	if collector.Calls() != 1 {
		t.Fatalf("gate invocation count must be 1, got %d", collector.Calls())
	}
	if seenCwd != worktreePath {
		t.Fatalf("gate runner cwd %q != subject worktree %q", seenCwd, worktreePath)
	}
	if capture.ExitCode != 0 {
		t.Fatalf("gate exit code: %d", capture.ExitCode)
	}
}

type evidenceRecorderRunner struct {
	onRun func(dir string)
}

func (r *evidenceRecorderRunner) Run(ctx context.Context, name string, args []string, dir string, env []string) evidence.CommandResult {
	if r.onRun != nil {
		r.onRun(dir)
	}
	return evidence.CommandResult{ExitCode: 0, Stdout: []byte("ok\n"), Stderr: []byte{}}
}

func TestClosureExecuteChecksAgainstSubjectTree_ExcludeSemantics(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	sentinelPath := filepath.Join(t.TempDir(), "exclude-command-ran")
	executor := NewGitV2SubjectExecutor(RealGit{})
	req := V2ExecuteRequest{
		RepositoryRoot: dir,
		SubjectCommit:  subject,
		SubjectTree:    subjectTree,
		EvidenceDir:    t.TempDir(),
		Checks: []PlanCheck{
			{
				ID: "exclude-must-not-run", Mode: "exclude",
				Argv:             []string{"sh", "-c", "touch \"$1\"", "sh", sentinelPath},
				WorkingDirectory: ".",
				TimeoutSeconds:   5,
			},
		},
		CommandExecutor: boundedCommandExecutor{},
	}
	res, err := executor.ExecuteSubjectChecks(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(res.CheckResults) != 1 {
		t.Fatalf("expected 1 check result, got %d", len(res.CheckResults))
	}
	if res.CheckResults[0].Status != "excluded" {
		t.Fatalf("exclude check must have status=excluded, got %q", res.CheckResults[0].Status)
	}
	if _, err := os.Stat(sentinelPath); !os.IsNotExist(err) {
		t.Fatalf("sentinel file must not exist; got err=%v", err)
	}
}
