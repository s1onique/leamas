// SPDX-License-Identifier: Apache-2.0

// simple_entrypoint_test.go is the executable-contract-first
// suite for the simplified closure entrypoint. The tests
// exercise the production-default helpers through a fake
// gitClient; the architecture is bound to real production
// paths, not stubs.

package closure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// simpleFakeGitClient records every command and serves canned
// responses keyed by command argv. Tests use it as the
// production-binding witness for resolveSubjectTree,
// boundedPush, and (optionally) discoverFrozenPlanForAct.
// simpleFakeGitClient records every command and serves canned
// responses keyed by command argv. The fixtures map is keyed
// by the joined argv and stores a FIFO slice of responses so
// multiple calls with the same argv can return different
// results (e.g., pre-push vs post-push ls-remote).
type simpleFakeGitClient struct {
	mu       sync.Mutex
	calls    []gitCommandCall
	fixtures map[string][]gitCommandResult
}

type gitCommandCall struct {
	Directory string
	Args      []string
	Stdin     string
}

// Run records every command and serves canned responses.
// Fixtures are consumed FIFO so multiple calls with the same
// argv can return different results (e.g., pre-push vs
// post-push ls-remote).
//
// FAIL-CLOSED: unmatched invocations return a typed failure
// (ExitCode 127 + non-nil Err + descriptive Stderr). The previous
// behaviour of silently returning success on unmatched commands
// caused vacuous GREEN reports; this prevents that class of
// false-positive.
func (f *simpleFakeGitClient) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := strings.Join(args, "\x00")
	f.calls = append(f.calls, gitCommandCall{Directory: directory, Args: append([]string(nil), args...)})
	if f.fixtures == nil {
		return unexpectedGit(args, nil)
	}
	if responses, ok := f.fixtures[key]; ok && len(responses) > 0 {
		// Pop FIFO so multiple calls with the same argv can
		// return different results.
		resp := responses[0]
		f.fixtures[key] = responses[1:]
		return resp
	}
	if _, ok := f.fixtures[key]; ok {
		// Empty slice — fixture was exhausted. Return zero
		// (treat as no match; fail closed below).
	}
	return unexpectedGit(args, nil)
}

// RunWithStdin records the stdin alongside the argv so the
// fixture can match the operation's authoritative input (e.g.,
// `git update-ref --stdin` where the body is on stdin).
func (f *simpleFakeGitClient) RunWithStdin(ctx context.Context, directory, stdin string, args ...string) gitCommandResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	argvKey := strings.Join(args, "\x00")
	f.calls = append(f.calls, gitCommandCall{Directory: directory, Args: append([]string(nil), args...), Stdin: stdin})
	if f.fixtures == nil {
		return unexpectedGit(args, &stdin)
	}
	stdinKey := argvKey + "\nstdin=\n" + strings.TrimRight(stdin, "\r\n")
	if responses, ok := f.fixtures[stdinKey]; ok && len(responses) > 0 {
		resp := responses[0]
		f.fixtures[stdinKey] = responses[1:]
		return resp
	}
	// Fallback: try argv-only match.
	if responses, ok := f.fixtures[argvKey]; ok && len(responses) > 0 {
		resp := responses[0]
		f.fixtures[argvKey] = responses[1:]
		return resp
	}
	return unexpectedGit(args, &stdin)
}

func (f *simpleFakeGitClient) RunWithEnv(ctx context.Context, directory string, env []string, args ...string) gitCommandResult {
	return f.Run(ctx, directory, args...)
}

func (f *simpleFakeGitClient) RunWithStdinAndEnv(ctx context.Context, directory, stdin string, env []string, args ...string) gitCommandResult {
	return f.RunWithStdin(ctx, directory, stdin, args...)
}

// unexpectedGit constructs a fail-closed response for an
// invocation that no test fixture anticipated. Production
// commands must be matched by tests or the test is wrong.
func unexpectedGit(args []string, stdin *string) gitCommandResult {
	var sb strings.Builder
	sb.WriteString("unexpected git invocation: ")
	for i, a := range args {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(a)
	}
	if stdin != nil {
		sb.WriteString(" (stdin=")
		sb.WriteString(*stdin)
		sb.WriteString(")")
	}
	return gitCommandResult{
		Err:      errors.New("fake git: " + sb.String()),
		Stderr:   []byte(sb.String()),
		ExitCode: 127,
	}
}

// hasCommand reports whether any recorded call's argv starts
// with the given command (e.g., "push", "fetch").
func (f *simpleFakeGitClient) hasCommand(cmd string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if len(c.Args) > 0 && c.Args[0] == cmd {
			return true
		}
	}
	return false
}

// hasForceOrForceWithLease reports whether any recorded push
// call used --force / --force-with-lease. The test fails
// (security regression) if true.
func (f *simpleFakeGitClient) hasForceOrForceWithLease() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if len(c.Args) > 0 && c.Args[0] == "push" {
			for _, a := range c.Args {
				if a == "--force" || a == "--force-with-lease" {
					return true
				}
			}
		}
	}
	return false
}

// fakeTransactionRunner records the request it received and
// returns a canned TransactionResult.
type fakeTransactionRunner struct {
	calls  int
	result *TransactionResult
	err    error
}

func (f *fakeTransactionRunner) RunClosureV2(ctx context.Context, req RunV2Options) (*TransactionResult, error) {
	f.calls++
	return f.result, f.err
}

func (f *fakeTransactionRunner) Calls() int { return f.calls }

// makeFakeTransactionResult returns a non-fixed-point result.
func makeFakeTransactionResult(closureCommit, closureTree string) *TransactionResult {
	return &TransactionResult{
		ActID:            "ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01",
		FreezeCommit:     "F-oid",
		SubjectCommit:    "S-oid",
		ClosureCommit:    closureCommit,
		ClosureTree:      closureTree,
		Verdict:          "PASS",
		TransactionState: v2StateNew,
	}
}

// fixedPointTransactionResult returns a TransactionResult
// whose TransactionState is v2StateVerified.
func fixedPointTransactionResult(closureCommit, closureTree string) *TransactionResult {
	return &TransactionResult{
		ActID:            "ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01",
		FreezeCommit:     "F-oid",
		SubjectCommit:    "S-oid",
		ClosureCommit:    closureCommit,
		ClosureTree:      closureTree,
		Verdict:          "PASS",
		TransactionState: v2StateVerified,
	}
}

// newSimpleGitSingle constructs a simpleFakeGitClient from a
// map of single responses (the most common test shape). Tests
// that need FIFO responses across multiple identical-argv
// calls use newSimpleGitFIFO directly.
func newSimpleGitSingle(fixtures map[string]gitCommandResult) *simpleFakeGitClient {
	fifo := make(map[string][]gitCommandResult, len(fixtures))
	for k, v := range fixtures {
		fifo[k] = []gitCommandResult{v}
	}
	return &simpleFakeGitClient{fixtures: fifo}
}

// newSimpleGit constructs a simpleFakeGitClient from a map of
// single responses (convenience wrapper for newSimpleGitSingle).
func newSimpleGit(fixtures map[string]gitCommandResult) *simpleFakeGitClient {
	return newSimpleGitSingle(fixtures)
}

// newSimpleGitFIFO constructs a simpleFakeGitClient from a map
// of FIFO response slices (for tests that need multiple
// different responses per argv).
func newSimpleGitFIFO(fixtures map[string][]gitCommandResult) *simpleFakeGitClient {
	return &simpleFakeGitClient{fixtures: fixtures}
}

// writePlanFile materialises a freeze-plan file at
// <repoRoot>/<planPath> (creating intermediate directories)
// and returns repoRoot. The planPath is interpreted relative
// to the repository root; the production discoverFrozenPlanForAct
// looks at repoRoot/docs/closure-plans/<ACT-ID>.json by
// default, but the test may override planPath.
func writePlanFile(t *testing.T, repoRoot, planPath, content string) string {
	t.Helper()
	if planPath == "" {
		return repoRoot
	}
	full := filepath.Join(repoRoot, planPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("writePlanFile mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("writePlanFile write: %v", err)
	}
	return repoRoot
}

// TestSimpleCloseContract is the table-driven suite for the
// simplified closure entrypoint. The contract cases exercise
// the production-default helpers through a fake gitClient.
func TestSimpleCloseContract(t *testing.T) {
	type want struct {
		state         string
		verdict       string
		rerunRequired bool
		published     bool
		reasonCode    string
		runnerCalls   int
		expectError   bool
	}
	cases := []struct {
		name     string
		req      SimpleCloseRequest
		git      *simpleFakeGitClient
		planPath string
		planJSON string
		runner   *fakeTransactionRunner
		want     want
	}{
		{
			name: "act_and_subject_only_required",
			req: SimpleCloseRequest{
				ActID:   "ACT-PROD",
				Subject: "0123456789abcdef0123456789abcdef01234567",
				Lane:    "fast",
			},
			git: newSimpleGit(map[string]gitCommandResult{
				"rev-parse\x00--verify\x00--end-of-options\x000123456789abcdef0123456789abcdef01234567^{tree}": {Stdout: []byte("S-tree-oid"), ExitCode: 0},
			}),
			planPath: "docs/closure-plans/ACT-PROD.json",
			planJSON: `{"freeze_commit":"F-oid","plan_path":"docs/closure-plans/ACT-PROD.json"}`,
			runner:   &fakeTransactionRunner{result: makeFakeTransactionResult("C", "C-tree")},
			want: want{
				state: "fixed_point", verdict: "PASS", rerunRequired: false,
				published: false, runnerCalls: 1,
			},
		},
		{
			name: "unsupported_lane_rejected",
			req: SimpleCloseRequest{
				ActID:   "ACT-PROD",
				Subject: "0123456789abcdef0123456789abcdef01234567",
				Lane:    "slow",
			},
			git:    newSimpleGit(nil),
			runner: &fakeTransactionRunner{result: makeFakeTransactionResult("C", "C-tree")},
			want: want{
				state: "rerun_required", verdict: "FAIL", rerunRequired: true,
				reasonCode: "unsupported_lane", expectError: true,
			},
		},
		{
			name: "already_closed_returns_fixed_point",
			req: SimpleCloseRequest{
				ActID:   "ACT-PROD",
				Subject: "0123456789abcdef0123456789abcdef01234567",
				Lane:    "fast",
			},
			git: newSimpleGit(map[string]gitCommandResult{
				"rev-parse\x00--verify\x00--end-of-options\x000123456789abcdef0123456789abcdef01234567^{tree}": {Stdout: []byte("S-tree-oid"), ExitCode: 0},
			}),
			planPath: "docs/closure-plans/ACT-PROD.json",
			planJSON: `{"freeze_commit":"F-oid","plan_path":"docs/closure-plans/ACT-PROD.json"}`,
			runner:   &fakeTransactionRunner{result: fixedPointTransactionResult("existing-C", "existing-C-tree")},
			want: want{
				state: "fixed_point", verdict: "PASS", rerunRequired: false,
				runnerCalls: 1,
			},
		},
		{
			name: "unclosed_subject_reaches_orchestrator",
			req: SimpleCloseRequest{
				ActID:   "ACT-PROD",
				Subject: "0123456789abcdef0123456789abcdef01234567",
				Lane:    "fast",
			},
			git: newSimpleGit(map[string]gitCommandResult{
				"rev-parse\x00--verify\x00--end-of-options\x000123456789abcdef0123456789abcdef01234567^{tree}": {Stdout: []byte("S-tree-oid"), ExitCode: 0},
			}),
			planPath: "docs/closure-plans/ACT-PROD.json",
			planJSON: `{"freeze_commit":"F-oid","plan_path":"docs/closure-plans/ACT-PROD.json"}`,
			runner:   &fakeTransactionRunner{result: makeFakeTransactionResult("new-C", "new-C-tree")},
			want: want{
				state: "fixed_point", verdict: "PASS", rerunRequired: false,
				runnerCalls: 1,
			},
		},
		{
			name: "publish_false_zero_remote_mutation",
			req: SimpleCloseRequest{
				ActID:   "ACT-PROD",
				Subject: "0123456789abcdef0123456789abcdef01234567",
				Lane:    "fast",
			},
			git: newSimpleGit(map[string]gitCommandResult{
				"rev-parse\x00--verify\x00--end-of-options\x000123456789abcdef0123456789abcdef01234567^{tree}": {Stdout: []byte("S-tree-oid"), ExitCode: 0},
			}),
			planPath: "docs/closure-plans/ACT-PROD.json",
			planJSON: `{"freeze_commit":"F-oid","plan_path":"docs/closure-plans/ACT-PROD.json"}`,
			runner:   &fakeTransactionRunner{result: makeFakeTransactionResult("new-C", "new-C-tree")},
			want: want{
				state: "fixed_point", verdict: "PASS", rerunRequired: false,
				published: false, runnerCalls: 1,
			},
		},
		{
			name: "partial_result_survives_runner_error",
			req: SimpleCloseRequest{
				ActID:   "ACT-PROD",
				Subject: "0123456789abcdef0123456789abcdef01234567",
				Lane:    "fast",
			},
			git: newSimpleGit(map[string]gitCommandResult{
				"rev-parse\x00--verify\x00--end-of-options\x000123456789abcdef0123456789abcdef01234567^{tree}": {Stdout: []byte("S-tree-oid"), ExitCode: 0},
			}),
			planPath: "docs/closure-plans/ACT-PROD.json",
			planJSON: `{"freeze_commit":"F-oid","plan_path":"docs/closure-plans/ACT-PROD.json"}`,
			runner: &fakeTransactionRunner{
				result: makeFakeTransactionResult("partial-C", "partial-C-tree"),
				err:    errors.New("cleanup_only_failure"),
			},
			want: want{
				state: "rerun_required", verdict: "PASS", rerunRequired: true,
				reasonCode: "transaction_failed", runnerCalls: 1, expectError: true,
			},
		},
	}

	// fakeFrozenPlanLoader returns a canned FrozenPlan for the
	// close tests. Phase 6D canonical design: F identifies the
	// plan; the plan schema carries no self-F reference. The
	// loader just returns a stable F + plan-path so SimpleClose
	// gets authoritative inputs.
	fakeFrozenPlanLoader := func(ctx context.Context, g gitClient, repoRoot, actID string) (FrozenPlan, error) {
		return FrozenPlan{
			FreezeCommit: "F-oid",
			PlanPath:     "docs/closure-plans/" + actID + ".json",
		}, nil
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := writePlanFile(t, t.TempDir(), tc.planPath, tc.planJSON)
			deps := SimpleCloseDeps{
				FrozenPlanLoader:  fakeFrozenPlanLoader,
				TransactionRunner: tc.runner.RunClosureV2,
				Git:               tc.git,
				RepositoryRoot:    repoRoot,
				Remote:            "origin",
				Now:               func() time.Time { return time.Unix(1700000000, 0) },
			}
			result, err := SimpleClose(context.Background(), tc.req, deps)

			if tc.want.expectError && err == nil {
				t.Fatalf("expected error, got nil (result=%+v)", result)
			}
			if !tc.want.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := result.State; got != tc.want.state {
				t.Errorf("state = %q, want %q", got, tc.want.state)
			}
			if got := result.Verdict; got != tc.want.verdict {
				t.Errorf("verdict = %q, want %q", got, tc.want.verdict)
			}
			if got := result.RerunRequired; got != tc.want.rerunRequired {
				t.Errorf("rerun_required = %v, want %v", got, tc.want.rerunRequired)
			}
			if got := result.Published; got != tc.want.published {
				t.Errorf("published = %v, want %v", got, tc.want.published)
			}
			if tc.want.reasonCode != "" && result.ReasonCode != tc.want.reasonCode {
				t.Errorf("reason_code = %q, want %q", result.ReasonCode, tc.want.reasonCode)
			}
			if got := tc.runner.Calls(); got != tc.want.runnerCalls {
				t.Errorf("runner calls = %d, want %d", got, tc.want.runnerCalls)
			}
		})
	}
}

// TestResolveSubjectTreeProduction exercises the production
// resolveSubjectTree against a fake gitClient. The fake
// returns canned output for `git rev-parse --verify
// --end-of-options <S>^{tree}` and the test asserts the
// production helper returns the exact tree OID.
func TestResolveSubjectTreeProduction(t *testing.T) {
	t.Run("valid_committed_subject_returns_tree", func(t *testing.T) {
		git := newSimpleGit(map[string]gitCommandResult{
			"rev-parse\x00--verify\x00--end-of-options\x00S^{tree}": {Stdout: []byte("S-tree-oid"), ExitCode: 0},
		})
		tree, err := resolveSubjectTree(context.Background(), git, "/repo", "S")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tree != "S-tree-oid" {
			t.Errorf("tree = %q, want %q", tree, "S-tree-oid")
		}
	})
	t.Run("unknown_subject_returns_subject_tree_unavailable", func(t *testing.T) {
		git := newSimpleGit(map[string]gitCommandResult{
			"rev-parse\x00--verify\x00--end-of-options\x00S^{tree}": {Stderr: []byte("fatal: not a tree"), ExitCode: 128},
		})
		_, err := resolveSubjectTree(context.Background(), git, "/repo", "S")
		if err == nil {
			t.Fatalf("expected subject_tree_unavailable error, got nil")
		}
		if !strings.Contains(err.Error(), "subject_tree_unavailable") {
			t.Errorf("error %q does not contain subject_tree_unavailable", err.Error())
		}
	})
}

// TestDiscoverFrozenPlanForActProduction exercises the production
// discoverFrozenPlanForAct. The function reads F from the
// canonical sideband ref refs/factory/freeze/<ACT-ID> — NOT
// from the on-disk plan file (the canonical authority model
// is: F identifies the plan; the plan does NOT identify F).
func TestDiscoverFrozenPlanForActProduction(t *testing.T) {
	t.Run("existing_freeze_ref_returns_freeze_metadata", func(t *testing.T) {
		fOID := strings.Repeat("f", 40)
		git := newSimpleGit(map[string]gitCommandResult{
			"rev-parse\x00--verify\x00--end-of-options\x00refs/factory/freeze/ACT-PROD": {
				Stdout: []byte(fOID), ExitCode: 0,
			},
		})
		repoRoot := t.TempDir()
		frozen, err := discoverFrozenPlanForAct(context.Background(), git, repoRoot, "ACT-PROD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if frozen.FreezeCommit != fOID {
			t.Errorf("FreezeCommit = %q, want %q", frozen.FreezeCommit, fOID)
		}
		if frozen.PlanPath != "docs/closure-plans/ACT-PROD.json" {
			t.Errorf("PlanPath = %q, want %q", frozen.PlanPath, "docs/closure-plans/ACT-PROD.json")
		}
	})
	t.Run("missing_freeze_ref_returns_act_not_frozen", func(t *testing.T) {
		git := newSimpleGit(map[string]gitCommandResult{
			"rev-parse\x00--verify\x00--end-of-options\x00refs/factory/freeze/ACT-NEW": {
				Stderr: []byte("fatal: not a ref"), ExitCode: 128,
			},
		})
		repoRoot := t.TempDir()
		_, err := discoverFrozenPlanForAct(context.Background(), git, repoRoot, "ACT-NEW")
		if err == nil {
			t.Fatalf("expected act_not_frozen error, got nil")
		}
		if !strings.Contains(err.Error(), "act_not_frozen") {
			t.Errorf("error %q does not contain act_not_frozen", err.Error())
		}
	})
}

// TestBeginActProduction exercises the production BeginAct
// against a fake gitClient. The fake simulates the bounded
// Phase 6D sequence: status (clean worktree), rev-parse HEAD,
// hash-object -w --stdin, read-tree/update-index/write-tree
// against a TEMP INDEX, commit-tree, ONE ref transaction
// (`git update-ref --stdin`) covering BOTH the freeze ref and
// HEAD CAS moves, read-tree against the live index, plan write.
func TestBeginActProduction(t *testing.T) {
	headCommit := strings.Repeat("a", 40)
	blobOID := strings.Repeat("b", 40)
	treeOID := strings.Repeat("c", 40)
	fOID := strings.Repeat("d", 40)

	// Construct the ref transaction exactly once. This is the
	// single authority primitive: Git acquires locks for all
	// queued refs atomically; either both become authoritative
	// or neither does.
	txn := "start\n" +
		"update refs/factory/freeze/ACT-BEGIN " + fOID + " " + strings.Repeat("0", 40) + "\n" +
		"update HEAD " + fOID + " " + headCommit + "\n" +
		"prepare\n" +
		"commit\n"
	txnKey := "update-ref\x00--stdin\nstdin=\n" + strings.TrimRight(txn, "\r\n")

	t.Run("begin_act_creates_freeze_commit_and_plan_file", func(t *testing.T) {
		git := newSimpleGitFIFO(map[string][]gitCommandResult{
			"status\x00--porcelain=v1\x00--untracked-files=all":                                                {{Stdout: nil, ExitCode: 0}},
			"rev-parse\x00--verify\x00refs/factory/freeze/ACT-BEGIN":                                           {{ExitCode: 128, Stderr: []byte("fatal: not a ref")}},
			"rev-parse\x00--verify\x00--end-of-options\x00HEAD^{commit}":                                       {{Stdout: []byte(headCommit), ExitCode: 0}},
			"rev-parse\x00--verify\x00--end-of-options\x00HEAD^{tree}":                                         {{Stdout: []byte(headCommit), ExitCode: 0}},
			"hash-object\x00-w\x00--stdin":                                                                     {{Stdout: []byte(blobOID), ExitCode: 0}},
			"update-index\x00--add\x00--cacheinfo\x00100644," + blobOID + ",docs/closure-plans/ACT-BEGIN.json": {{ExitCode: 0}},
			"write-tree": {{Stdout: []byte(treeOID), ExitCode: 0}},
			"commit-tree\x00" + treeOID + "\x00-p\x00" + headCommit + "\x00-m\x00factory: freeze ACT ACT-BEGIN": {{Stdout: []byte(fOID), ExitCode: 0}},
			// Two FIFO read-tree results: temp index (GIT_INDEX_FILE)
			// then live index.
			"read-tree\x00HEAD": {{ExitCode: 0}, {ExitCode: 0}},
		})
		git.fixtures[txnKey] = []gitCommandResult{{ExitCode: 0}}

		repoRoot := t.TempDir()
		deps := SimpleCloseDeps{
			Git:            git,
			RepositoryRoot: repoRoot,
			Now:            func() time.Time { return time.Unix(1700000000, 0) },
		}
		plan, err := BeginAct(context.Background(), deps, "ACT-BEGIN")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.FreezeCommit != fOID {
			t.Errorf("FreezeCommit = %q, want %q", plan.FreezeCommit, fOID)
		}
		if plan.PlanPath != "docs/closure-plans/ACT-BEGIN.json" {
			t.Errorf("PlanPath = %q, want %q", plan.PlanPath, "docs/closure-plans/ACT-BEGIN.json")
		}

		// Path invariant: worktree plan path is the SINGLE
		// canonical repository-relative path
		// docs/closure-plans/<ACT>.json (NOT a doubled
		// docs/closure-plans/docs/closure-plans/...).
		planPath := filepath.Join(repoRoot, "docs", "closure-plans", "ACT-BEGIN.json")
		data, err := os.ReadFile(planPath)
		if err != nil {
			t.Fatalf("read canonical plan file: %v", err)
		}
		if strings.Contains(string(data), "F-pending") {
			t.Errorf("plan file still has F-pending placeholder: %s", data)
		}
		if strings.Contains(string(data), "freeze_commit") {
			t.Errorf("plan file should NOT contain freeze_commit (canonical authority model): %s", data)
		}
		if !strings.Contains(string(data), `"act_id":"ACT-BEGIN"`) {
			t.Errorf("plan file missing act_id: %s", data)
		}

		// The duplicated-path bug must NOT manifest.
		dupPath := filepath.Join(repoRoot, "docs", "closure-plans", "docs", "closure-plans", "ACT-BEGIN.json")
		if _, err := os.Stat(dupPath); err == nil {
			t.Errorf("duplicate-path plan file leaked: %s", dupPath)
		}

		// Mandatory fake assertions: prove the production
		// authority primitive is the ONE ref transaction and
		// that no standalone update-ref invocations exist.
		updateRefStdinCalls := 0
		standaloneFreezeRefCalls := 0
		standaloneHEADCalls := 0
		for _, c := range git.calls {
			if len(c.Args) < 2 || c.Args[0] != "update-ref" {
				continue
			}
			if c.Args[1] == "--stdin" {
				updateRefStdinCalls++
				continue
			}
			if c.Args[1] == "HEAD" {
				standaloneHEADCalls++
			}
			if strings.HasPrefix(c.Args[1], "refs/factory/freeze/") {
				standaloneFreezeRefCalls++
			}
		}
		if updateRefStdinCalls != 1 {
			t.Errorf("update-ref --stdin calls = %d, want 1", updateRefStdinCalls)
		}
		if standaloneFreezeRefCalls != 0 {
			t.Errorf("standalone update-ref refs/factory/freeze/... = %d, want 0", standaloneFreezeRefCalls)
		}
		if standaloneHEADCalls != 0 {
			t.Errorf("standalone update-ref HEAD ... = %d, want 0", standaloneHEADCalls)
		}
	})

	t.Run("begin_act_requires_git_client", func(t *testing.T) {
		repoRoot := t.TempDir()
		deps := SimpleCloseDeps{
			Git:            nil,
			RepositoryRoot: repoRoot,
		}
		_, err := BeginAct(context.Background(), deps, "ACT-PROD")
		if err == nil {
			t.Fatalf("expected git-client-required error, got nil")
		}
		if !strings.Contains(err.Error(), "git client is required") {
			t.Errorf("error %q does not contain expected message", err.Error())
		}
	})

	t.Run("begin_act_rejects_dirty_worktree", func(t *testing.T) {
		git := newSimpleGit(map[string]gitCommandResult{
			"status\x00--porcelain=v1\x00--untracked-files=all": {Stdout: []byte(" M foo.txt\n"), ExitCode: 0},
		})
		repoRoot := t.TempDir()
		deps := SimpleCloseDeps{
			Git:            git,
			RepositoryRoot: repoRoot,
		}
		_, err := BeginAct(context.Background(), deps, "ACT-PROD")
		if err == nil {
			t.Fatalf("expected dirty-worktree error, got nil")
		}
		if !strings.Contains(err.Error(), "caller worktree is dirty") {
			t.Errorf("error %q does not contain expected message", err.Error())
		}
	})
}

// TestSimpleCloseFreezeAuthorityMatch covers the happy path of
// the freeze-authority invariant: sideband F observed by the
// discoverer agrees with tx F observed by the transaction.
// SimpleClose must return fixed_point, PASS, no rerun.
func TestSimpleCloseFreezeAuthorityMatch(t *testing.T) {
	frozenF := strings.Repeat("f", 40)
	txF := frozenF // sideband agrees with tx
	req := SimpleCloseRequest{
		ActID:   "ACT-PROD",
		Subject: "0123456789abcdef0123456789abcdef01234567",
		Lane:    "fast",
	}
	git := newSimpleGit(map[string]gitCommandResult{
		"rev-parse\x00--verify\x00--end-of-options\x000123456789abcdef0123456789abcdef01234567^{tree}": {Stdout: []byte("S-tree-oid"), ExitCode: 0},
	})
	repoRoot := writePlanFile(t, t.TempDir(), "docs/closure-plans/ACT-PROD.json", "{}")
	loader := func(ctx context.Context, g gitClient, root, actID string) (FrozenPlan, error) {
		return FrozenPlan{FreezeCommit: frozenF, PlanPath: "docs/closure-plans/" + actID + ".json"}, nil
	}
	runner := &fakeTransactionRunner{result: &TransactionResult{
		ActID:            "ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01",
		FreezeCommit:     txF,
		SubjectCommit:    "S-oid",
		ClosureCommit:    "C-oid",
		ClosureTree:      "C-tree",
		Verdict:          "PASS",
		TransactionState: v2StateNew,
	}}
	deps := SimpleCloseDeps{
		FrozenPlanLoader:  loader,
		TransactionRunner: runner.RunClosureV2,
		Git:               git,
		RepositoryRoot:    repoRoot,
		Remote:            "origin",
		Now:               func() time.Time { return time.Unix(1700000000, 0) },
	}
	res, err := SimpleClose(context.Background(), req, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.FreezeCommit != frozenF {
		t.Fatalf("FreezeCommit = %q, want %q", res.FreezeCommit, frozenF)
	}
	if res.State != "fixed_point" {
		t.Fatalf("State = %q, want fixed_point", res.State)
	}
	if res.RerunRequired {
		t.Fatalf("RerunRequired = true, want false")
	}
	if res.Verdict != "PASS" {
		t.Fatalf("Verdict = %q, want PASS", res.Verdict)
	}
}

// TestSimpleCloseFreezeAuthorityMismatchFails asserts that
// when the sideband F (discovered) disagrees with tx F
// (transaction-derived), SimpleClose fails closed with
// freeze_authority_mismatch. The envelope preserves both
// values for the operator and returns rerun_required.
func TestSimpleCloseFreezeAuthorityMismatchFails(t *testing.T) {
	sidebandF := strings.Repeat("a", 40)
	txF := strings.Repeat("b", 40) // sideband disagrees with tx
	req := SimpleCloseRequest{
		ActID:   "ACT-PROD",
		Subject: "0123456789abcdef0123456789abcdef01234567",
		Lane:    "fast",
	}
	git := newSimpleGit(map[string]gitCommandResult{
		"rev-parse\x00--verify\x00--end-of-options\x000123456789abcdef0123456789abcdef01234567^{tree}": {Stdout: []byte("S-tree-oid"), ExitCode: 0},
	})
	repoRoot := writePlanFile(t, t.TempDir(), "docs/closure-plans/ACT-PROD.json", "{}")
	loader := func(ctx context.Context, g gitClient, root, actID string) (FrozenPlan, error) {
		return FrozenPlan{FreezeCommit: sidebandF, PlanPath: "docs/closure-plans/" + actID + ".json"}, nil
	}
	runner := &fakeTransactionRunner{result: &TransactionResult{
		ActID:            "ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01",
		FreezeCommit:     txF,
		SubjectCommit:    "S-oid",
		ClosureCommit:    "C-oid",
		ClosureTree:      "C-tree",
		Verdict:          "PASS",
		TransactionState: v2StateNew,
	}}
	deps := SimpleCloseDeps{
		FrozenPlanLoader:  loader,
		TransactionRunner: runner.RunClosureV2,
		Git:               git,
		RepositoryRoot:    repoRoot,
		Remote:            "origin",
		Now:               func() time.Time { return time.Unix(1700000000, 0) },
	}
	res, err := SimpleClose(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("expected freeze_authority_mismatch error, got nil (result=%+v)", res)
	}
	if !strings.Contains(err.Error(), "freeze_authority_mismatch") {
		t.Fatalf("error %q does not contain freeze_authority_mismatch", err.Error())
	}
	if res.State != "rerun_required" {
		t.Fatalf("State = %q, want rerun_required", res.State)
	}
	if !res.RerunRequired {
		t.Fatalf("RerunRequired = false, want true")
	}
	if res.ReasonCode != "freeze_authority_mismatch" {
		t.Fatalf("ReasonCode = %q, want freeze_authority_mismatch", res.ReasonCode)
	}
	// Envelope preserves BOTH Fs for forensic comparison.
	if res.FreezeCommit != txF {
		t.Fatalf("FreezeCommit = %q, want tx F %q", res.FreezeCommit, txF)
	}
}

// TestSimpleCloseFreezeAuthorityMissingFails asserts that when
// the sideband ref is present (non-empty F) but the underlying
// transaction observed an empty F, SimpleClose fails closed
// with freeze_authority_unavailable. We must NOT paper over
// the missing transaction-side observation.
func TestSimpleCloseFreezeAuthorityMissingFails(t *testing.T) {
	sidebandF := strings.Repeat("a", 40)
	req := SimpleCloseRequest{
		ActID:   "ACT-PROD",
		Subject: "0123456789abcdef0123456789abcdef01234567",
		Lane:    "fast",
	}
	git := newSimpleGit(map[string]gitCommandResult{
		"rev-parse\x00--verify\x00--end-of-options\x000123456789abcdef0123456789abcdef01234567^{tree}": {Stdout: []byte("S-tree-oid"), ExitCode: 0},
	})
	repoRoot := writePlanFile(t, t.TempDir(), "docs/closure-plans/ACT-PROD.json", "{}")
	loader := func(ctx context.Context, g gitClient, root, actID string) (FrozenPlan, error) {
		return FrozenPlan{FreezeCommit: sidebandF, PlanPath: "docs/closure-plans/" + actID + ".json"}, nil
	}
	runner := &fakeTransactionRunner{result: &TransactionResult{
		ActID:            "ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01",
		FreezeCommit:     "", // transaction observed no F; fail-closed
		SubjectCommit:    "S-oid",
		ClosureCommit:    "C-oid",
		ClosureTree:      "C-tree",
		Verdict:          "PASS",
		TransactionState: v2StateNew,
	}}
	deps := SimpleCloseDeps{
		FrozenPlanLoader:  loader,
		TransactionRunner: runner.RunClosureV2,
		Git:               git,
		RepositoryRoot:    repoRoot,
		Remote:            "origin",
		Now:               func() time.Time { return time.Unix(1700000000, 0) },
	}
	res, err := SimpleClose(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("expected freeze_authority_unavailable error, got nil (result=%+v)", res)
	}
	if !strings.Contains(err.Error(), "freeze_authority_unavailable") {
		t.Fatalf("error %q does not contain freeze_authority_unavailable", err.Error())
	}
	if res.State != "rerun_required" {
		t.Fatalf("State = %q, want rerun_required", res.State)
	}
	if !res.RerunRequired {
		t.Fatalf("RerunRequired = false, want true")
	}
	if res.ReasonCode != "freeze_authority_unavailable" {
		t.Fatalf("ReasonCode = %q, want freeze_authority_unavailable", res.ReasonCode)
	}
	// Result.FreezeCommit MUST remain empty (transaction was the
	// authoritative source; absence means we must NOT paper over
	// with sideband F).
	if res.FreezeCommit != "" {
		t.Fatalf("FreezeCommit = %q, want empty (do not paper over sideband F)", res.FreezeCommit)
	}
}

// TestBeginActSecondInvocationReturnsSameF asserts that a
// second BeginAct call for the same ACT ID returns the same F
// without manufacturing a second authoritative freeze commit.
// Uses the production RealGit against a real fixture.
func TestBeginActSecondInvocationReturnsSameF(t *testing.T) {
	f := newRealGitFixture(t)
	actID := "ACT-PROD"
	git := RealGit{}
	deps := SimpleCloseDeps{
		Git:            git,
		RepositoryRoot: f.repoRoot,
		Remote:         "origin",
		Now:            func() time.Time { return time.Unix(1700000000, 0) },
	}
	ctx := context.Background()
	first, err := BeginAct(ctx, deps, actID)
	if err != nil {
		t.Fatalf("first BeginAct: %v", err)
	}
	second, err := BeginAct(ctx, deps, actID)
	if err != nil {
		t.Fatalf("second BeginAct: %v", err)
	}
	if second.FreezeCommit != first.FreezeCommit {
		t.Fatalf("re-Begin F = %s, want original F = %s", second.FreezeCommit, first.FreezeCommit)
	}
	// A..HEAD must contain only F (no F2).
	countOut, err := runGitValue(ctx, git, f.repoRoot, "rev-list", "--count", f.initialOID+"..HEAD")
	if err != nil {
		t.Fatalf("rev-list A..HEAD: %v", err)
	}
	if countOut != "1" {
		t.Fatalf("A..HEAD commit count = %s, want 1 (no F2 manufacturing)", countOut)
	}
}

// TestBoundedPushProduction exercises the production boundedPush
// against a fake gitClient. The fake simulates the bounded
// sequence: rev-parse HEAD^{commit}, fetch, ls-remote,
// merge-base --is-ancestor, push, fresh ls-remote read-back.
func TestBoundedPushProduction(t *testing.T) {
	localOID := strings.Repeat("a", 40)
	remoteOID := strings.Repeat("b", 40)
	newRemoteOID := localOID // read-back equals the local we just pushed
	branch := "main"

	// Case 10: remote ancestor local → FF push succeeds.
	t.Run("remote_ancestor_local_publishes", func(t *testing.T) {
		git := newSimpleGitFIFO(map[string][]gitCommandResult{
			"rev-parse\x00--verify\x00--end-of-options\x00HEAD^{commit}": {{Stdout: []byte(localOID), ExitCode: 0}},
			"fetch\x00origin": {{ExitCode: 0}},
			"ls-remote\x00--heads\x00origin\x00main": {
				{Stdout: []byte(remoteOID + "\trefs/heads/" + branch), ExitCode: 0},
				{Stdout: []byte(newRemoteOID + "\trefs/heads/" + branch), ExitCode: 0},
			},
			"merge-base\x00--is-ancestor\x00" + remoteOID + "\x00" + localOID: {{ExitCode: 0}},
			"push\x00origin\x00HEAD:refs/heads/main":                          {{ExitCode: 0}},
		})
		repoRoot := t.TempDir()
		pub, err := boundedPush(context.Background(), git, repoRoot, "origin", "refs/heads/"+branch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pub.ReasonCode != "published" {
			t.Errorf("reason_code = %q, want published", pub.ReasonCode)
		}
		if pub.PublicationHead != localOID {
			t.Errorf("publication_head = %q, want %q", pub.PublicationHead, localOID)
		}
		if git.hasForceOrForceWithLease() {
			t.Errorf("boundedPush used --force or --force-with-lease; REFUSED")
		}
	})

	// Case 11: remote == local → idempotent (no FF check needed; first push semantics).
	t.Run("first_push_remote_unadvertised_succeeds", func(t *testing.T) {
		git := newSimpleGitFIFO(map[string][]gitCommandResult{
			"rev-parse\x00--verify\x00--end-of-options\x00HEAD^{commit}": {{Stdout: []byte(localOID), ExitCode: 0}},
			"fetch\x00origin": {{ExitCode: 0}},
			"ls-remote\x00--heads\x00origin\x00main": {
				{Stdout: []byte(""), ExitCode: 0},
				{Stdout: []byte(localOID + "\trefs/heads/" + branch), ExitCode: 0},
			},
			"push\x00origin\x00HEAD:refs/heads/main": {{ExitCode: 0}},
		})
		repoRoot := t.TempDir()
		pub, err := boundedPush(context.Background(), git, repoRoot, "origin", "refs/heads/"+branch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pub.ReasonCode != "published" {
			t.Errorf("reason_code = %q, want published", pub.ReasonCode)
		}
	})

	// Case 12: remote is NOT an ancestor of local → non_fast_forward.
	t.Run("remote_not_ancestor_returns_non_fast_forward", func(t *testing.T) {
		git := newSimpleGit(map[string]gitCommandResult{
			"rev-parse\x00--verify\x00--end-of-options\x00HEAD^{commit}": {Stdout: []byte(localOID), ExitCode: 0},
			"fetch\x00origin":                                                 {ExitCode: 0},
			"ls-remote\x00--heads\x00origin\x00main":                          {Stdout: []byte(remoteOID + "\trefs/heads/" + branch), ExitCode: 0},
			"merge-base\x00--is-ancestor\x00" + remoteOID + "\x00" + localOID: {ExitCode: 1, Stderr: []byte("not an ancestor")},
		})
		repoRoot := t.TempDir()
		pub, err := boundedPush(context.Background(), git, repoRoot, "origin", "refs/heads/"+branch)
		if err == nil {
			t.Fatalf("expected non_fast_forward error, got nil")
		}
		if pub.ReasonCode != "non_fast_forward" {
			t.Errorf("reason_code = %q, want non_fast_forward", pub.ReasonCode)
		}
		if git.hasCommand("push") {
			t.Errorf("boundedPush attempted push despite non-FF; REFUSED")
		}
	})

	// Case 13: read-back mismatch → publication_verification_failed.
	t.Run("readback_mismatch_returns_publication_verification_failed", func(t *testing.T) {
		wrongReadBack := strings.Repeat("c", 40)
		_ = wrongReadBack // used in fixture literal below
		git := newSimpleGitFIFO(map[string][]gitCommandResult{
			"rev-parse\x00--verify\x00--end-of-options\x00HEAD^{commit}": {{Stdout: []byte(localOID), ExitCode: 0}},
			"fetch\x00origin": {{ExitCode: 0}},
			"ls-remote\x00--heads\x00origin\x00main": {
				{Stdout: []byte(remoteOID + "\trefs/heads/" + branch), ExitCode: 0},
				{Stdout: []byte(wrongReadBack + "\trefs/heads/" + branch), ExitCode: 0},
			},
			"merge-base\x00--is-ancestor\x00" + remoteOID + "\x00" + localOID: {{ExitCode: 0}},
			"push\x00origin\x00HEAD:refs/heads/main":                          {{ExitCode: 0}},
		})
		repoRoot := t.TempDir()
		pub, err := boundedPush(context.Background(), git, repoRoot, "origin", "refs/heads/"+branch)
		if err == nil {
			t.Fatalf("expected publication_verification_failed error, got nil")
		}
		if pub.ReasonCode != "publication_verification_failed" {
			t.Errorf("reason_code = %q, want publication_verification_failed", pub.ReasonCode)
		}
	})

	// Case 14: fetch failure → fresh_authority_unavailable.
	t.Run("fetch_failure_returns_fresh_authority_unavailable", func(t *testing.T) {
		git := newSimpleGit(map[string]gitCommandResult{
			"rev-parse\x00--verify\x00--end-of-options\x00HEAD^{commit}": {Stdout: []byte(localOID), ExitCode: 0},
			"fetch\x00origin": {Stderr: []byte("could not fetch"), ExitCode: 128},
		})
		repoRoot := t.TempDir()
		pub, err := boundedPush(context.Background(), git, repoRoot, "origin", "refs/heads/"+branch)
		if err == nil {
			t.Fatalf("expected fresh_authority_unavailable error, got nil")
		}
		if pub.ReasonCode != "fresh_authority_unavailable" {
			t.Errorf("reason_code = %q, want fresh_authority_unavailable", pub.ReasonCode)
		}
	})

	// Case 15: NO --force / --force-with-lease — security regression guard.
	t.Run("no_force_no_force_with_lease", func(t *testing.T) {
		git := newSimpleGitFIFO(map[string][]gitCommandResult{
			"rev-parse\x00--verify\x00--end-of-options\x00HEAD^{commit}": {{Stdout: []byte(localOID), ExitCode: 0}},
			"fetch\x00origin": {{ExitCode: 0}},
			"ls-remote\x00--heads\x00origin\x00main": {
				{Stdout: []byte(""), ExitCode: 0},
				{Stdout: []byte(localOID + "\trefs/heads/" + branch), ExitCode: 0},
			},
			"push\x00origin\x00HEAD:refs/heads/main": {{ExitCode: 0}},
		})
		repoRoot := t.TempDir()
		_, _ = boundedPush(context.Background(), git, repoRoot, "origin", "refs/heads/"+branch)
		if git.hasForceOrForceWithLease() {
			t.Errorf("boundedPush used --force or --force-with-lease; SECURITY REGRESSION")
		}
	})
}
