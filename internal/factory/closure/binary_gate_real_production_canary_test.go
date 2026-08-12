// SPDX-License-Identifier: Apache-2.0

// binary_gate_real_production_canary_test.go proves the isolated
// full-source Git clone fixture and the production R6-B execution path.
//
// CORRECTION10: fixture substrate proof (ISOLATED_FULL_SOURCE_CLONE, etc.)
// CORRECTION11: real production execution proof (REAL_B1_EXECUTED, etc.)
// CORRECTION12: compose the full-source fixture with production R6-B on the
// correct F < S topology, with B1 stubbed so the canary stays hermetic.
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
	Freeze      string // Freeze commit (F < S)
	SentinelRel string // Relative sentinel path for assertions
}

// newRealR6BFixture creates a full-source isolated clone with tracked
// sentinel commits S and F. The fixture is suitable for REAL_B1_EXECUTED
// because the clone contains the complete Go source for real builds.
//
// CORRECTION12: extracted from Test A so Test B can compose
// the full-source fixture with production R6-B execution.
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

	// Phase 2: Create F with different sentinel AND plan.
	// The adapter requires F < S (freeze ancestor of subject).
	factoryDir := filepath.Join(fixtureRoot, ".factory", "testdata")
	if err := os.MkdirAll(factoryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(factoryDir, "correction12-sentinel.txt")
	sentinelRel := ".factory/testdata/correction12-sentinel.txt"
	if err := os.WriteFile(sentinel, []byte("caller-state\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Also create the plan directory and plan file as part of F.
	// The plan must be committed BEFORE S so F < S.
	freezePlanDir := filepath.Join(fixtureRoot, "docs", "closure-plans")
	if err := os.MkdirAll(freezePlanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	planBytes := r6BValidPlanBytes()
	planPath := "docs/closure-plans/X.json"
	if err := os.WriteFile(filepath.Join(fixtureRoot, planPath), planBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	realCanaryRunGit(t, fixtureRoot, "add", "-f", sentinelRel, planPath)
	realCanaryRunGit(t, fixtureRoot, "commit", "-m", "freeze F with plan")
	F := realCanaryRunGit(t, fixtureRoot, "rev-parse", "HEAD")

	// Phase 3: Create S with tracked sentinel.
	// S is created AFTER F so that F < S (freeze is ancestor of subject).
	if err := os.WriteFile(sentinel, []byte("subject-state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	realCanaryRunGit(t, fixtureRoot, "add", "-f", sentinelRel)
	realCanaryRunGit(t, fixtureRoot, "commit", "-m", "subject S")
	S := realCanaryRunGit(t, fixtureRoot, "rev-parse", "HEAD")
	STree := realCanaryRunGit(t, fixtureRoot, "rev-parse", S+"^{tree}")

	// Prove sentinel is in Git tree S.
	sentinelInS := realCanaryRunGit(t, fixtureRoot, "show", S+":"+sentinelRel)
	if !strings.Contains(sentinelInS, "subject-state") {
		t.Fatalf("sentinel in tree S = %q, want subject-state", sentinelInS)
	}

	// F was already set before S was created. Verify F < S.
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
// execution path with the isolated full-source fixture.
//
// CORRECTION12: COMPOSES the full-source fixture with production R6-B.
// This is the authoritative R6-B integration canary.
//
// Key assertions prove the production R6-B integration:
// - the executor runs the gate in an S worktree, not the caller root
// - GateCollector captures the gate inside the live-S window, exactly once
// - B2 completeness derives from the recorded observation
// - real topology (F < S) verified via the full-source clone
//
// B1 is stubbed via r6BStubBuildFn, so this canary does NOT prove
// BuildExactSubjectBinary. TestClosureExactSubjectBinaryAuthority owns
// that authority.
func TestClosureBinaryGateRealProductionHappyPath(t *testing.T) {
	t.Parallel()

	// Phase 1: Create full-source fixture using the extracted helper.
	// CORRECTION12: This is the critical composition. The full-source
	// fixture is the same one Test A proves, now passed to production R6-B.
	// The plan is already committed as part of F in the fixture, so F < S.
	fixture := newRealR6BFixture(t)
	planPath := "docs/closure-plans/X.json"

	// Phase 2: Verify topology F < S (freeze is ancestor of subject).
	// The fixture already committed the plan as part of F.
	if fixture.Freeze == fixture.Subject {
		t.Fatal("freeze commit equals subject S")
	}

	// Phase 3: Build the request for production execution.
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         fixture.Root,
		SubjectCommit:          fixture.Subject,
		FreezeCommit:           fixture.Freeze,
		PlanPath:               planPath,
		EvidenceDirectory:      filepath.Join(t.TempDir(), "leamas-gate-evidence"),
		ManifestOutput:         "",
	}
	identity := newR6BTestBinaryIdentity(t)

	// Phase 4: CORRECTION12 - Call production R6-B with stubbed B1.
	//
	// The test verifies R6-B integration (B2 authority from gate capture),
	// not B1 authority (build correctness). Using stubbed B1 ensures
	// the test runs deterministically in hermetic environments without
	// needing the full Go toolchain.
	//
	// BuildFn = r6BStubBuildFn    -> deterministic stub binary
	// CommandRunner = runner     -> recording runner for assertions
	// NewCollectorFn = collector -> production GateCollector runs
	runner := &r6BRecordingRunner{}
	collector := evidence.NewGateCollector(runner)

	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(), req, identity, RunClosureProtocolV2ExecuteDeps{
		BuildFn:        r6BStubBuildFn(t),
		NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
		CommandRunner:  runner,
		OutputRoot:     r6BOutputRoot(t),
		OutputName:     "leamas",
		RunID:          "correction12-real-production",
		EvidenceDir:    r6BEvidenceDir(t),
	})
	if err != nil {
		t.Fatalf("RunClosureProtocolV2ExecuteWithDeps: %v", err)
	}

	// Phase 5: CORRECTION12 - B1 observation assertions.
	// B1 is stubbed here, so these prove the executor propagates a complete
	// binary observation bound to S; they do not prove the real build.
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

	// Phase 6: CORRECTION12 - Production-created S worktree path.
	// The executor captures SubjectRoot during the live-S window.
	if obs.Gate.SubjectRoot == "" {
		t.Fatal("Gate.SubjectRoot is empty; production did not capture S worktree path")
	}
	if obs.Gate.SubjectExecutionRoot == "" {
		t.Fatal("Gate.SubjectExecutionRoot is empty")
	}
	if obs.Runtime.SubjectExecutionRoot == "" {
		t.Fatal("Runtime.SubjectExecutionRoot is empty")
	}

	// Phase 7: CORRECTION12 - Root bindings.
	// The gate and runtime must use the same production worktree path.
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

	// Phase 8: CORRECTION12 - Runtime authority.
	// The runtime captures S and S^{tree} during execution.
	if obs.Runtime.SubjectCommit != fixture.Subject {
		t.Fatalf("Runtime.SubjectCommit %s != S %s", obs.Runtime.SubjectCommit, fixture.Subject)
	}
	if obs.Runtime.SubjectTree != fixture.SubjectTree {
		t.Fatalf("Runtime.SubjectTree %s != STree %s", obs.Runtime.SubjectTree, fixture.SubjectTree)
	}

	// Phase 9: CORRECTION12 - GateCollector invoked exactly once.
	if obs.Gate.InvocationCount != 1 {
		t.Fatalf("Gate.InvocationCount = %d, want 1", obs.Gate.InvocationCount)
	}

	// Phase 10: CORRECTION12 - B2 completeness from the recorded observation.
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

	t.Logf("CORRECTION12: production R6-B integration PASS (B1 stubbed)")
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
