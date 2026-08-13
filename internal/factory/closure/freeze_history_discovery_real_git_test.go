// SPDX-License-Identifier: Apache-2.0

// freeze_history_discovery_real_git_test.go is the real-Git
// proof for ACT-LEAMAS-FACTORY-FREEZE-REDISCOVERY-PORTABILITY-
// AND-REAL-DOGFOOD01.
//
// These tests run against RealGit in a fresh temporary
// repository; they do NOT use command fakes. They prove:
//
//   - F is recovered from committed history WITHOUT the
//     refs/factory/freeze/<ACT> sideband ref.
//   - A forged or mutated worktree plan has NO effect: authority
//     comes from F:P via git cat-file, never from disk.
//   - The ACT's required Section 13 invariants are upheld.

package closure

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
	"strings"
	"testing"
	"time"
)

// fixedNow returns a fixed time.Time suitable for production
// SimpleCloseDeps.Now.
func fixedNow() time.Time { return time.Unix(1700000000, 0).UTC() }

// initEmptyRepo constructs a temporary git repository with a
// single clean commit A on main. The fixture does not add a
// remote; callers wire their own.
func initEmptyRepo(t *testing.T) (repoRoot, initialOID string) {
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
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
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
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	oid, err := runGitValue(context.Background(), RealGit{}, repo,
		"rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return repo, oid
}

// gitCommitPath stages and commits a single file with the
// supplied content under the given repo-relative path and
// commit message. Returns the new HEAD commit OID.
func gitCommitPath(t *testing.T, repoRoot, relPath, content, msg string) string {
	t.Helper()
	abs := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
	for _, args := range [][]string{
		{"-C", repoRoot, "add", "--", relPath},
		{"-C", repoRoot, "commit", "-q", "-m", msg},
	} {
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	oid, err := runGitValue(context.Background(), RealGit{}, repoRoot,
		"rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return oid
}

// revTree resolves a commit's tree OID.
func revTree(t *testing.T, git gitClient, repoRoot, commit string) string {
	t.Helper()
	tree, err := runGitValue(context.Background(), git, repoRoot,
		"rev-parse", "--verify", "--end-of-options", commit+"^{tree}")
	if err != nil {
		t.Fatalf("rev-parse %s^{tree}: %v", commit, err)
	}
	return tree
}

// minimalPlanBytes constructs a Plan Contract v1 wire-shape
// JSON with baseline.commit_oid = commitOID and
// baseline.tree_oid = treeOID. Empty checks, empty artifacts,
// empty policy — sufficient for the freeze history primitive's
// structural F3 + F4 + F5 + F6 predicates.
func minimalPlanBytes(actID, commitOID, treeOID string) []byte {
	plan := map[string]any{
		"contract_version": 1,
		"act_id":           actID,
		"baseline": map[string]any{
			"commit_oid": commitOID,
			"tree_oid":   treeOID,
		},
		"execution": map[string]any{"mode": "serial_fail_fast"},
		"checks":    []any{},
		"artifacts": []any{},
		"policy":    map[string]any{},
	}
	out, err := json.Marshal(plan)
	if err != nil {
		panic(err)
	}
	return out
}

// TestFreezeHistoryDiscoveryRealGit is the Section 13 proof.
// It establishes:
//
//  1. BeginAct creates F (parented at the initial commit A).
//  2. The agent commits a harmless S on top of F.
//  3. F < S, merge-base --is-ancestor F S exits 0.
//  4. The sideband refs/factory/freeze/<ACT> is deleted.
//  5. DiscoverFrozenPlanFromHistory still recovers F.
//  6. P.act_id == ACT, P.baseline.commit_oid == A,
//     P.baseline.tree_oid == tree(A).
//  7. A FORGED worktree plan (mutated after S) does NOT affect
//     discovery; authority comes from F:P, not disk.
func TestFreezeHistoryDiscoveryRealGit(t *testing.T) {
	ctx := context.Background()
	git := RealGit{}
	repo, aOID := initEmptyRepo(t)

	actID := "ACT-PORTABILITY"

	// 1. BeginAct: creates F parented at A.
	frozen, err := BeginAct(ctx, SimpleCloseDeps{
		Git:            git,
		RepositoryRoot: repo,
		Now:            fixedNow,
		Remote:         "origin",
	}, actID)
	if err != nil {
		t.Fatalf("BeginAct: %v", err)
	}
	actF := frozen.FreezeCommit
	if actF == "" {
		t.Fatalf("BeginAct returned empty F")
	}
	// F^ == A.
	parentOID, err := runGitValue(ctx, git, repo, "rev-parse", "--verify",
		"--end-of-options", actF+"^")
	if err != nil {
		t.Fatalf("rev-parse F^: %v", err)
	}
	if parentOID != aOID {
		t.Fatalf("F^ = %s, want %s (initial commit A)", parentOID, aOID)
	}

	// 2. Commit a harmless S on top of F.
	sOID := gitCommitPath(t, repo, "src/hello.txt", "hello\n",
		"subject: harmless implementation")

	// 3. F < S (strict ancestor).
	ancRes := git.Run(ctx, repo, "merge-base", "--is-ancestor", actF, sOID)
	if ancRes.Err != nil || ancRes.ExitCode != 0 {
		t.Fatalf("merge-base --is-ancestor F=%s S=%s: exit=%d err=%v",
			actF, sOID, ancRes.ExitCode, ancRes.Err)
	}
	if actF == sOID {
		t.Fatalf("F == S (%s); expected F < S", actF)
	}

	// 4. Delete the sideband ref.
	delRes := git.Run(ctx, repo, "update-ref", "-d", freezeRefName(actID))
	if delRes.Err != nil || delRes.ExitCode != 0 {
		t.Fatalf("delete sideband ref: exit=%d err=%v stderr=%s",
			delRes.ExitCode, delRes.Err, string(delRes.Stderr))
	}
	verifyRes := git.Run(ctx, repo, "rev-parse", "--verify", "--end-of-options",
		freezeRefName(actID))
	if verifyRes.ExitCode == 0 {
		t.Fatalf("sideband ref still present after delete: %s",
			string(verifyRes.Stdout))
	}

	// 5. DiscoverFrozenPlanFromHistory recovers F from committed history.
	discovered, outcome, err := DiscoverFrozenPlanFromHistory(ctx, git, repo, actID, sOID)
	if err != nil {
		t.Fatalf("DiscoverFrozenPlanFromHistory: %v", err)
	}
	if outcome.Reason != HistoryDiscoveryDerived {
		t.Fatalf("outcome.Reason = %s, want %s (candidates=%v)",
			outcome.Reason, HistoryDiscoveryDerived, outcome.Candidates)
	}
	if discovered.FreezeCommit != actF {
		t.Fatalf("discovered F = %s, want %s", discovered.FreezeCommit, actF)
	}

	// 6. Read F:P and verify structural bindings.
	blobAtF, err := runGitValue(ctx, git, repo,
		"rev-parse", "--verify", "--end-of-options",
		actF+":"+frozenPlanPath(actID))
	if err != nil {
		t.Fatalf("rev-parse F:P: %v", err)
	}
	planBytes, err := readBlobBytesViaGit(ctx, git, repo, blobAtF)
	if err != nil {
		t.Fatalf("read F:P bytes: %v", err)
	}
	// structural decode for verification — BeginAct emits plans
	// with empty checks (a known production shape); strict
	// canonical validation is the closure runner's authority, not
	// the freeze history primitive's.
	plan, err := decodeTypedPlanForDiscovery(plancontractDecodeRoot(t, planBytes))
	if err != nil {
		t.Fatalf("structural decode F:P: %v", err)
	}
	if plan.ActID != actID {
		t.Errorf("P.act_id = %s, want %s", plan.ActID, actID)
	}
	if plan.Baseline.CommitOID != aOID {
		t.Errorf("P.baseline.commit_oid = %s, want %s", plan.Baseline.CommitOID, aOID)
	}
	if plan.Baseline.TreeOID != revTree(t, git, repo, aOID) {
		t.Errorf("P.baseline.tree_oid = %s, want %s (A^{tree})",
			plan.Baseline.TreeOID, revTree(t, git, repo, aOID))
	}

	// 7. Forge a worktree plan; discovery MUST still return the
	//    canonical F (authority is from F:P, not disk).
	worktreePath := filepath.Join(repo, frozenPlanPath(actID))
	forged := `{"contract_version":1,"act_id":"ACT-FORGED"}`
	if err := os.WriteFile(worktreePath, []byte(forged), 0o644); err != nil {
		t.Fatalf("forge worktree plan: %v", err)
	}
	discoveredAfterForge, outcomeAfterForge, err := DiscoverFrozenPlanFromHistory(ctx, git, repo, actID, sOID)
	if err != nil {
		t.Fatalf("discovery after forge: %v", err)
	}
	if outcomeAfterForge.Reason != HistoryDiscoveryDerived {
		t.Fatalf("after forge: outcome = %s, want derived", outcomeAfterForge.Reason)
	}
	if discoveredAfterForge.FreezeCommit != actF {
		t.Errorf("forge changed discovery: F = %s, want %s",
			discoveredAfterForge.FreezeCommit, actF)
	}
}

// TestFreezeHistoryDiscoveryAmbiguousRealGit covers the
// freeze_authority_ambiguous unique-authority rule: two commits
// that satisfy F1..F7 in the ancestry of S must be rejected.
func TestFreezeHistoryDiscoveryAmbiguousRealGit(t *testing.T) {
	ctx := context.Background()
	git := RealGit{}
	repo, aOID := initEmptyRepo(t)

	const actID = "ACT-AMBIGUOUS"

	// F1 = A's child; plan v1 with baseline=A.
	_ = gitCommitPath(t, repo, "docs/closure-plans/ACT-AMBIGUOUS.json",
		string(minimalPlanBytes(actID, aOID, revTree(t, git, repo, aOID))),
		"factory: freeze ACT-AMBIGUOUS (v1)")
	// Intermediate X introduces an unrelated file.
	xOID := gitCommitPath(t, repo, "src/intermediate.txt", "mid\n",
		"intermediate")
	// F2 = X's child; plan v2 with baseline=X.
	_ = gitCommitPath(t, repo, "docs/closure-plans/ACT-AMBIGUOUS.json",
		string(minimalPlanBytes(actID, xOID, revTree(t, git, repo, xOID))),
		"factory: freeze ACT-AMBIGUOUS (v2)")
	// S = F2's child, harmless subject.
	sOID := gitCommitPath(t, repo, "src/subj.txt", "ok\n", "subject: ambiguous")

	_, outcome, err := DiscoverFrozenPlanFromHistory(ctx, git, repo, actID, sOID)
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	if outcome.Reason != HistoryDiscoveryAmbiguous {
		t.Fatalf("reason = %s, want %s (candidates=%v)",
			outcome.Reason, HistoryDiscoveryAmbiguous, outcome.Candidates)
	}
	if len(outcome.Candidates) < 2 {
		t.Fatalf("expected >=2 candidates in ambiguous case, got %d (%v)",
			len(outcome.Candidates), outcome.Candidates)
	}
}

// TestFreezeHistoryDiscoveryNotFrozenRealGit covers the
// act_not_frozen case: zero commits in the ancestry of S
// introduce the canonical plan.
func TestFreezeHistoryDiscoveryNotFrozenRealGit(t *testing.T) {
	ctx := context.Background()
	git := RealGit{}
	repo, _ := initEmptyRepo(t)
	sOID := gitCommitPath(t, repo, "src/x.txt", "x\n", "subject: never frozen")
	_, outcome, err := DiscoverFrozenPlanFromHistory(ctx, git, repo, "ACT-NONE", sOID)
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	if outcome.Reason != HistoryDiscoveryNotFrozen {
		t.Fatalf("reason = %s, want %s", outcome.Reason, HistoryDiscoveryNotFrozen)
	}
}

// TestFreezeHistoryDiscoveryNoSidebandRefRealGit covers the
// ACT's NO_HIDDEN_SIDEBAND_DEPENDENCY invariant: deleting the
// sideband ref between BeginAct and discoverFrozenPlanForAct
// MUST NOT change the discovery outcome.
func TestFreezeHistoryDiscoveryNoSidebandRefRealGit(t *testing.T) {
	ctx := context.Background()
	git := RealGit{}
	repo, _ := initEmptyRepo(t)
	const actID = "ACT-NO-SIDEBAND"
	if _, err := BeginAct(ctx, SimpleCloseDeps{
		Git:            git,
		RepositoryRoot: repo,
		Now:            fixedNow,
		Remote:         "origin",
	}, actID); err != nil {
		t.Fatalf("BeginAct: %v", err)
	}
	// Add a harmless subject commit so HEAD != F and the history
	// discovery has a strict ancestor F < S to traverse.
	_ = gitCommitPath(t, repo, "src/subj.txt", "ok\n", "subject")
	// Capture F BEFORE deleting the sideband.
	withCache, err := discoverFrozenPlanForAct(ctx, git, repo, actID)
	if err != nil {
		t.Fatalf("discover with cache: %v", err)
	}
	delRes := git.Run(ctx, repo, "update-ref", "-d", freezeRefName(actID))
	if delRes.Err != nil || delRes.ExitCode != 0 {
		t.Fatalf("delete sideband: %v %s", delRes.Err, string(delRes.Stderr))
	}
	withoutCache, err := discoverFrozenPlanForAct(ctx, git, repo, actID)
	if err != nil {
		t.Fatalf("discover without cache: %v", err)
	}
	if withoutCache.FreezeCommit != withCache.FreezeCommit {
		t.Fatalf("cache vs nocache: F = %s vs %s",
			withCache.FreezeCommit, withoutCache.FreezeCommit)
	}
}

// plancontractDecodeRoot is a small test helper that structural-
// decodes Plan Contract v1 bytes without applying semantic
// validation. Used to verify F:P contents without requiring
// empty-checks plans (BeginAct's canonical shape) to pass full
// semantic validation.
func plancontractDecodeRoot(t *testing.T, planBytes []byte) map[string]any {
	t.Helper()
	root, err := plancontract.DecodeBytes(planBytes)
	if err != nil {
		t.Fatalf("plancontract.DecodeBytes: %v", err)
	}
	rootMap, ok := root.(map[string]any)
	if !ok {
		t.Fatalf("plan root is %T, not map[string]any", root)
	}
	return rootMap
}

// avoid unused-import warnings if helpers go unused.
var _ = strings.TrimSpace
