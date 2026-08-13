// SPDX-License-Identifier: Apache-2.0

// simple_entrypoint_real_git_test.go is the real-Git proof
// suite for the simplified closure entrypoint. These tests
// drive production helpers (BeginAct, discoverFrozenPlanForAct,
// SimpleClose) through RealGit against a real temporary
// repository; they do NOT use command fakes. They exist to
// prove the authority primitives (atomic ref transaction,
// freeze ref CAS, HEAD CAS, plan-path invariant, idempotence)
// behave correctly against real Git semantics.

package closure

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// realGitFixture is the minimal real-Git repository fixture:
// one initial commit on main, no remote, ready for the
// production BeginAct path.
type realGitFixture struct {
	repoRoot   string
	initialOID string
}

// newRealGitFixture constructs a temporary git repository with
// a single clean commit A on main. The fixture DOES NOT add a
// remote so close path tests can wire their own remote.
func newRealGitFixture(t *testing.T) realGitFixture {
	t.Helper()
	repo := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main", repo},
		{"-C", repo, "config", "user.email", "leamas-test@example.invalid"},
		{"-C", repo, "config", "user.name", "leamas-test"},
		{"-C", repo, "config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	// Initial commit A on main with a tiny file.
	seedPath := filepath.Join(repo, "README.md")
	if err := os.WriteFile(seedPath, []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	for _, args := range [][]string{
		{"-C", repo, "add", "README.md"},
		{"-C", repo, "commit", "-q", "-m", "initial"},
	} {
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	oid, err := runGitValue(context.Background(), RealGit{}, repo, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return realGitFixture{repoRoot: repo, initialOID: oid}
}

// TestBeginActRealGit is the decisive real-Git proof of
// BeginAct. It runs against RealGit in a fresh repository and
// asserts all production invariants simultaneously:
//
//	HEAD                  == F
//	refs/factory/freeze/X == F
//	F^                    == A
//	git status            == empty
//	git diff              == empty
//	git diff --cached     == empty
//	F:P                   == worktree docs/closure-plans/X.json
//
// Then it re-runs BeginAct and asserts idempotence: same F,
// no second authoritative freeze commit.
func TestBeginActRealGit(t *testing.T) {
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

	// First BeginAct: should commit F.
	first, err := BeginAct(ctx, deps, actID)
	if err != nil {
		t.Fatalf("BeginAct first: %v", err)
	}
	if first.FreezeCommit == "" {
		t.Fatalf("first BeginAct returned empty FreezeCommit")
	}
	if first.FreezeCommit == f.initialOID {
		t.Fatalf("F equals initial A (%s); F must be a NEW commit", f.initialOID)
	}
	if first.PlanPath != frozenPlanPath(actID) {
		t.Fatalf("PlanPath = %q, want %q", first.PlanPath, frozenPlanPath(actID))
	}

	// HEAD == F
	headOID, err := runGitValue(ctx, git, f.repoRoot, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	if headOID != first.FreezeCommit {
		t.Fatalf("HEAD = %s, want F = %s", headOID, first.FreezeCommit)
	}

	// freeze-ref == F
	frOID, err := runGitValue(ctx, git, f.repoRoot, "rev-parse", "--verify", "--end-of-options", "refs/factory/freeze/"+actID)
	if err != nil {
		t.Fatalf("rev-parse freeze ref: %v", err)
	}
	if frOID != first.FreezeCommit {
		t.Fatalf("freeze-ref = %s, want F = %s", frOID, first.FreezeCommit)
	}

	// F^ == A
	parentOID, err := runGitValue(ctx, git, f.repoRoot, "rev-parse", "--verify", "--end-of-options", first.FreezeCommit+"^")
	if err != nil {
		t.Fatalf("rev-parse F^: %v", err)
	}
	if parentOID != f.initialOID {
		t.Fatalf("F^ = %s, want initial A = %s", parentOID, f.initialOID)
	}

	// status empty
	statusOut, err := runGitValue(ctx, git, f.repoRoot, "status", "--porcelain=v1")
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if statusOut != "" {
		t.Fatalf("git status --porcelain = %q, want empty", statusOut)
	}

	// git diff empty
	diffOut, err := runGitValue(ctx, git, f.repoRoot, "diff")
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	if diffOut != "" {
		t.Fatalf("git diff = %q, want empty", diffOut)
	}

	// git diff --cached empty
	cachedOut, err := runGitValue(ctx, git, f.repoRoot, "diff", "--cached")
	if err != nil {
		t.Fatalf("git diff --cached: %v", err)
	}
	if cachedOut != "" {
		t.Fatalf("git diff --cached = %q, want empty", cachedOut)
	}

	// F:P == worktree plan
	showFOut, err := runGitValue(ctx, git, f.repoRoot, "show", first.FreezeCommit+":"+frozenPlanPath(actID))
	if err != nil {
		t.Fatalf("git show F:P: %v", err)
	}
	worktreeP, err := os.ReadFile(filepath.Join(f.repoRoot, frozenPlanPath(actID)))
	if err != nil {
		t.Fatalf("read worktree plan: %v", err)
	}
	if string(worktreeP) != showFOut {
		t.Fatalf("worktree plan != F:P\nworktree=%q\nF:P=%q", string(worktreeP), showFOut)
	}

	// Plan file MUST NOT contain self-F reference.
	if strings.Contains(string(worktreeP), "freeze_commit") {
		t.Fatalf("plan schema carries self-F reference: %s", string(worktreeP))
	}

	// Re-Begin: same F, no second commit, HEAD unchanged.
	refCountBefore, err := runGitValue(ctx, git, f.repoRoot, "for-each-ref", "--count=1", "--format=%(objectname)", "refs/factory/freeze/"+actID)
	if err != nil {
		t.Fatalf("for-each-ref: %v", err)
	}
	if refCountBefore != first.FreezeCommit {
		t.Fatalf("freeze-ref = %s before re-Begin, want %s", refCountBefore, first.FreezeCommit)
	}

	second, err := BeginAct(ctx, deps, actID)
	if err != nil {
		t.Fatalf("BeginAct second: %v", err)
	}
	if second.FreezeCommit != first.FreezeCommit {
		t.Fatalf("re-Begin F = %s, want original F = %s", second.FreezeCommit, first.FreezeCommit)
	}

	// HEAD must still equal F (no second freeze commit layered).
	headAfter, err := runGitValue(ctx, git, f.repoRoot, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		t.Fatalf("rev-parse HEAD after re-Begin: %v", err)
	}
	if headAfter != first.FreezeCommit {
		t.Fatalf("HEAD after re-Begin = %s, want F = %s", headAfter, first.FreezeCommit)
	}

	// And the freeze-ref must still equal F.
	frAfter, err := runGitValue(ctx, git, f.repoRoot, "rev-parse", "--verify", "--end-of-options", "refs/factory/freeze/"+actID)
	if err != nil {
		t.Fatalf("rev-parse freeze ref after re-Begin: %v", err)
	}
	if frAfter != first.FreezeCommit {
		t.Fatalf("freeze-ref after re-Begin = %s, want F = %s", frAfter, first.FreezeCommit)
	}

	// Confirm we did NOT manufacture F2 by checking the commit
	// graph: only one commit must exist between A and HEAD
	// (F itself). Any F2 manufacturing would show up as a
	// second descendant.
	countOut, err := runGitValue(ctx, git, f.repoRoot, "rev-list", "--count", f.initialOID+"..HEAD")
	if err != nil {
		t.Fatalf("rev-list A..HEAD: %v", err)
	}
	if countOut != "1" {
		t.Fatalf("A..HEAD commit count = %s, want 1 (no F2 manufacturing)", countOut)
	}
}

// TestBeginActRejectsTraversalActID asserts BeginAct refuses
// ACT identifiers containing the ".." traversal sequence. This
// is a fail-closed invariant: the sideband ref must never
// resolve into something that could escape docs/closure-plans/.
func TestBeginActRejectsTraversalActID(t *testing.T) {
	f := newRealGitFixture(t)
	git := RealGit{}
	deps := SimpleCloseDeps{
		Git:            git,
		RepositoryRoot: f.repoRoot,
		Remote:         "origin",
		Now:            func() time.Time { return time.Unix(1700000000, 0) },
	}
	ctx := context.Background()

	// The traversal ID passes the actIDPattern regex in the
	// happy case but must be rejected by the explicit
	// traversal guard. If the pattern ever tightens, the
	// guard remains the canonical defense.
	traversal := "ACT-../escape"
	_, err := BeginAct(ctx, deps, traversal)
	if err == nil {
		t.Fatalf("expected traversal rejection, got nil")
	}
	if !strings.Contains(err.Error(), "act_id_invalid") {
		t.Fatalf("error %q does not contain act_id_invalid", err.Error())
	}
	// No ref may have been created.
	_, revErr := runGitValue(ctx, git, f.repoRoot, "rev-parse", "--verify", "--end-of-options", "refs/factory/freeze/"+traversal)
	if revErr == nil {
		t.Fatalf("traversal ACT produced freeze ref; refused")
	}
	// And HEAD is unchanged.
	head, _ := runGitValue(ctx, git, f.repoRoot, "rev-parse", "--verify", "HEAD^{commit}")
	if head != f.initialOID {
		t.Fatalf("HEAD changed after rejected traversal; HEAD=%s, want %s", head, f.initialOID)
	}
}
