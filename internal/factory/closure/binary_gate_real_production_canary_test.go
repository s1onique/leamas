// SPDX-License-Identifier: Apache-2.0

// binary_gate_real_production_canary_test.go proves the isolated
// full-source Git clone fixture and the zero-override production R6-B path.
//
// CORRECTION10: fixture substrate proof (ISOLATED_FULL_SOURCE_CLONE, etc.)
// CORRECTION11: real production execution proof (REAL_B1_EXECUTED, etc.)
// CORRECTION12: compose full-source fixture with zero-override production R6-B.
package closure

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// realR6BFixture is the authoritative fixture the real production
// canary uses. It contains a full-source isolated clone that can
// run the real BuildExactSubjectBinary.
type realR6BFixture struct {
	Root        string // RepositoryRoot for the request
	Subject     string // Literal S commit OID
	SubjectTree string // Literal S^{tree} OID
	Freeze      string // Freeze commit (F > S)
	SentinelRel string // Relative sentinel path for assertions
}

// newRealR6BFixture creates a full-source isolated clone with tracked
// sentinel commits S and F. The fixture is suitable for REAL_B1_EXECUTED
// because the clone contains the complete Go source for real builds.
//
// CORRECTION12: extracted from Test A so Test B can compose
// the full-source fixture with zero-override production execution.
func newRealR6BFixture(t *testing.T) *realR6BFixture {
	t.Helper()

	// Phase 1: Clone the COMMITTED source into an isolated fixture.
	sourceRoot := realCanaryRunGit(t, ".", "rev-parse", "--show-toplevel")
	fixtureRoot := filepath.Join(t.TempDir(), "leamas-real-fixture")
	realCanaryRunGit(t, "", "clone", sourceRoot, fixtureRoot)

	// Verify fixture is independent.
	if status := realCanaryRunGit(t, fixtureRoot, "status", "--short"); status != "" {
		t.Fatalf("fixture worktree is not clean: %s", status)
	}

	// Verify fixture contains complete source for real build.
	for _, path := range []string{"go.mod", "cmd/leamas", "internal/factory"} {
		if !realCanaryPathExists(t, filepath.Join(fixtureRoot, path)) {
			t.Fatalf("fixture is incomplete: missing %s", path)
		}
	}

	// Configure Git identity for commits.
	realCanaryRunGit(t, fixtureRoot, "config", "user.email", "test@example.com")
	realCanaryRunGit(t, fixtureRoot, "config", "user.name", "Test User")

	// Phase 2: Create S with tracked sentinel.
	factoryDir := filepath.Join(fixtureRoot, ".factory", "testdata")
	if err := os.MkdirAll(factoryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(factoryDir, "correction12-sentinel.txt")
	sentinelRel := ".factory/testdata/correction12-sentinel.txt"
	if err := os.WriteFile(sentinel, []byte("subject-state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Stage sentinel BEFORE committing S so it belongs to tree S.
	realCanaryRunGit(t, fixtureRoot, "add", "-f", sentinelRel)
	realCanaryRunGit(t, fixtureRoot, "commit", "-m", "subject S")
	S := realCanaryRunGit(t, fixtureRoot, "rev-parse", "HEAD")
	STree := realCanaryRunGit(t, fixtureRoot, "rev-parse", S+"^{tree}")

	// Prove sentinel is in Git tree S.
	sentinelInS := realCanaryRunGit(t, fixtureRoot, "show", S+":"+sentinelRel)
	if !strings.Contains(sentinelInS, "subject-state") {
		t.Fatalf("sentinel in tree S = %q, want subject-state", sentinelInS)
	}

	// Phase 3: Create F with different sentinel.
	if err := os.WriteFile(sentinel, []byte("caller-state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	realCanaryRunGit(t, fixtureRoot, "add", "-f", sentinelRel)
	realCanaryRunGit(t, fixtureRoot, "commit", "-m", "freeze F")
	F := realCanaryRunGit(t, fixtureRoot, "rev-parse", "HEAD")
	if F == S {
		t.Fatal("freeze F equals subject S")
	}

	// Prove sentinel in F differs.
	sentinelInF := realCanaryRunGit(t, fixtureRoot, "show", F+":"+sentinelRel)
	if !strings.Contains(sentinelInF, "caller-state") {
		t.Fatalf("sentinel in tree F = %q, want caller-state", sentinelInF)
	}

	// Assertions.
	if S == "" || F == "" {
		t.Fatal("commit hashes are empty")
	}
	if STree == "" {
		t.Fatal("tree hash is empty")
	}

	t.Logf("Fixture: root=%s S=%s STree=%s F=%s", fixtureRoot, S, STree, F)
	return &realR6BFixture{
		Root:        fixtureRoot,
		Subject:     S,
		SubjectTree: STree,
		Freeze:      F,
		SentinelRel: sentinelRel,
	}
}

// TestClosureBinaryGateIsolatedFixtureCanary proves the isolated
// full-source Git clone fixture capability used by R6-B tests.
//
// CORRECTION10/CORRECTION12: proves fixture substrate.
// Does NOT prove production execution.
func TestClosureBinaryGateIsolatedFixtureCanary(t *testing.T) {
	t.Parallel()

	fixture := newRealR6BFixture(t)
	t.Logf("Fixture: root=%s S=%s STree=%s F=%s",
		fixture.Root, fixture.Subject, fixture.SubjectTree, fixture.Freeze)

	// Assertions.
	if fixture.Subject == "" || fixture.Freeze == "" {
		t.Fatal("commit hashes are empty")
	}
	if fixture.SubjectTree == "" {
		t.Fatal("tree hash is empty")
	}
}

// TestClosureBinaryGateRealProductionHappyPath proves the production R6-B
// execution path with ZERO execution overrides.
//
// CORRECTION12: COMPOSES the full-source fixture with zero-override
// production R6-B. This is the authoritative real production canary.
//
// Key assertions:
// - BuildFn is NOT injected (production BuildExactSubjectBinary runs)
// - CommandRunner is NOT injected (production OsRunner runs)
// - NewCollectorFn is NOT injected (production GateCollector runs)
func TestClosureBinaryGateRealProductionHappyPath(t *testing.T) {
	t.Parallel()

	// Phase 1: Create full-source fixture using the extracted helper.
	// CORRECTION12: This is the critical composition. The full-source
	// fixture is the same one Test A proves, now passed to production R6-B.
	fixture := newRealR6BFixture(t)

	// Phase 2: Create freeze commit with plan (required by R6-B topology).
	// The plan must be committed so the production loader can read it.
	freezePlanDir := filepath.Join(fixture.Root, "docs", "closure-plans")
	if err := os.MkdirAll(freezePlanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	planBytes := r6BValidPlanBytes()
	planPath := "docs/closure-plans/X.json"
	if err := os.WriteFile(filepath.Join(fixture.Root, planPath), planBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	realCanaryRunGit(t, fixture.Root, "add", "-f", planPath)
	realCanaryRunGit(t, fixture.Root, "commit", "-m", "freeze plan")
	freezeCommit := realCanaryRunGit(t, fixture.Root, "rev-parse", "HEAD")

	// Phase 3: Verify topology F > S.
	if freezeCommit == fixture.Subject {
		t.Fatal("freeze commit equals subject S")
	}

	// Phase 4: Build the request for production execution.
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         fixture.Root,
		SubjectCommit:          fixture.Subject,
		FreezeCommit:           freezeCommit,
		PlanPath:               planPath,
		EvidenceDirectory:      filepath.Join(t.TempDir(), "leamas-gate-evidence"),
		ManifestOutput:         "",
	}
	identity := newR6BTestBinaryIdentity(t)

	// Phase 5: CORRECTION12 - Call production R6-B with ZERO overrides.
	//
	// RunClosureProtocolV2Execute uses empty deps, which means:
	// - BuildFn = nil        -> production BuildExactSubjectBinary runs
	// - CommandRunner = nil  -> production OsRunner runs
	// - NewCollectorFn = nil -> production evidence.NewGateCollector runs
	//
	// This is the ONLY test that proves real B1 execution.
	_, obs, err := RunClosureProtocolV2Execute(context.Background(), req, identity)
	if err != nil {
		t.Fatalf("RunClosureProtocolV2Execute: %v", err)
	}

	// Phase 6: CORRECTION12 - B1 authority assertions.
	// These prove the binary came from REAL BuildExactSubjectBinary.
	if obs.Binary.BinaryPath == "" {
		t.Fatal("B1 BinaryPath is empty")
	}
	if obs.Binary.BinarySHA256 == "" {
		t.Fatal("B1 BinarySHA256 is empty")
	}
	if obs.Binary.BinaryCommit != fixture.Subject {
		t.Fatalf("B1 BinaryCommit %s != S %s", obs.Binary.BinaryCommit, fixture.Subject)
	}
	if obs.Binary.BinaryModified {
		t.Fatal("B1 BinaryModified is true")
	}
	if !obs.Binary.Executable {
		t.Fatal("B1 Executable is false")
	}
	if !obs.Binary.SourceClean {
		t.Fatal("B1 SourceClean is false")
	}
	if !obs.Binary.SourceDetached {
		t.Fatal("B1 SourceDetached is false")
	}
	if !obs.Binary.OutputOutsideAllWorktrees {
		t.Fatal("B1 OutputOutsideAllWorktrees is false")
	}
	if obs.Binary.SourceCommit != fixture.Subject {
		t.Fatalf("B1 SourceCommit %s != S %s", obs.Binary.SourceCommit, fixture.Subject)
	}
	if obs.Binary.SourceTree != fixture.SubjectTree {
		t.Fatalf("B1 SourceTree %s != STree %s", obs.Binary.SourceTree, fixture.SubjectTree)
	}

	// Phase 7: CORRECTION12 - Production-created S worktree.
	// The production executor creates a detached worktree at S.
	if obs.Gate.SubjectRoot == "" {
		t.Fatal("Gate.SubjectRoot is empty; production did not create S worktree")
	}
	if obs.Gate.SubjectExecutionRoot == "" {
		t.Fatal("Gate.SubjectExecutionRoot is empty")
	}
	if obs.Runtime.SubjectExecutionRoot == "" {
		t.Fatal("Runtime.SubjectExecutionRoot is empty")
	}

	// Phase 8: CORRECTION12 - Prove worktree is S.
	// Read HEAD and TREE from the production-created worktree.
	worktreeHead := realCanaryRunGit(t, obs.Gate.SubjectRoot, "rev-parse", "HEAD")
	if worktreeHead != fixture.Subject {
		t.Fatalf("worktree HEAD %s != S %s", worktreeHead, fixture.Subject)
	}
	worktreeTree := realCanaryRunGit(t, obs.Gate.SubjectRoot, "rev-parse", "HEAD^{tree}")
	if worktreeTree != fixture.SubjectTree {
		t.Fatalf("worktree TREE %s != STree %s", worktreeTree, fixture.SubjectTree)
	}

	// Phase 9: CORRECTION12 - Prove detached state.
	// HEAD must be a detached OID, not a branch ref.
	branch := realCanaryRunGit(t, obs.Gate.SubjectRoot, "branch", "--show-current")
	if branch != "" {
		t.Fatalf("worktree is not detached; branch=%q", branch)
	}

	// Phase 10: CORRECTION12 - Prove sentinel at real subject root.
	// The production worktree should have the S sentinel content.
	sentinelInWorktree := realCanaryRunGit(t, obs.Gate.SubjectRoot, "show", "HEAD:"+fixture.SentinelRel)
	if !strings.Contains(sentinelInWorktree, "subject-state") {
		t.Fatalf("sentinel at worktree = %q, want subject-state", sentinelInWorktree)
	}

	// Phase 11: CORRECTION12 - Prove caller F sentinel differs.
	sentinelInCaller := realCanaryRunGit(t, fixture.Root, "show", fixture.Freeze+":"+fixture.SentinelRel)
	if !strings.Contains(sentinelInCaller, "caller-state") {
		t.Fatalf("caller sentinel = %q, want caller-state", sentinelInCaller)
	}

	// Phase 12: CORRECTION12 - Root bindings.
	if obs.Gate.SubjectRoot != obs.Runtime.SubjectExecutionRoot {
		t.Fatalf("Gate.SubjectRoot %s != Runtime.SubjectExecutionRoot %s",
			obs.Gate.SubjectRoot, obs.Runtime.SubjectExecutionRoot)
	}
	if obs.Gate.SubjectExecutionRoot != obs.Runtime.SubjectExecutionRoot {
		t.Fatalf("Gate.SubjectExecutionRoot %s != Runtime.SubjectExecutionRoot %s",
			obs.Gate.SubjectExecutionRoot, obs.Runtime.SubjectExecutionRoot)
	}
	if obs.Gate.SubjectRoot == fixture.Root {
		t.Fatal("Gate.SubjectRoot equals caller root; gate must use production S worktree")
	}

	// Phase 13: CORRECTION12 - Runtime authority.
	if obs.Runtime.SubjectCommit != fixture.Subject {
		t.Fatalf("Runtime.SubjectCommit %s != S %s", obs.Runtime.SubjectCommit, fixture.Subject)
	}
	if obs.Runtime.SubjectTree != fixture.SubjectTree {
		t.Fatalf("Runtime.SubjectTree %s != STree %s", obs.Runtime.SubjectTree, fixture.SubjectTree)
	}

	// Phase 14: CORRECTION12 - GateCollector invoked exactly once.
	if obs.Gate.InvocationCount != 1 {
		t.Fatalf("Gate.InvocationCount = %d, want 1", obs.Gate.InvocationCount)
	}

	// Phase 15: CORRECTION12 - B2 completeness from real observation.
	prepared, err := evidence.PrepareClosureEvidenceForPublication(
		evidence.BuildClosureEvidenceCandidate(evidence.CandidateInputs{
			Runtime:      obs.Runtime,
			Results:      obs.Results,
			Gate:         obs.Gate,
			Binary:       obs.Binary,
			CallerBefore: obs.CallerBefore,
			CallerAfter:  obs.CallerAfter,
			Cleanup:      obs.Cleanup,
		}))
	if err != nil {
		t.Fatalf("B2 barrier refused valid candidate: %v", err)
	}
	completeness := evidence.DeriveClosureEvidenceCompleteness(prepared.Document())
	if completeness != evidence.EvidenceComplete {
		t.Fatalf("B2 verdict = %s, want COMPLETE", completeness)
	}

	t.Logf("CORRECTION12: REAL production PASS")
	t.Logf("  S=%s STree=%s F=%s", fixture.Subject, fixture.SubjectTree, fixture.Freeze)
	t.Logf("  WorktreeRoot=%s", obs.Gate.SubjectRoot)
	t.Logf("  BinaryPath=%s SHA256=%s", obs.Binary.BinaryPath, obs.Binary.BinarySHA256)
}

// Helper functions.

func realCanaryRunGit(t *testing.T, dir, command string, args ...string) string {
	t.Helper()
	ctx := context.Background()
	var c *exec.Cmd
	if dir == "" {
		c = exec.CommandContext(ctx, "git", append([]string{command}, args...)...)
	} else {
		c = exec.CommandContext(ctx, "git", append([]string{command}, args...)...)
		c.Dir = dir
	}
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s %v: %v\n%s", command, args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func realCanaryReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func realCanaryPathExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}
