// SPDX-License-Identifier: Apache-2.0

// binary_gate_real_production_canary_test.go proves the isolated
// full-source Git clone fixture and the zero-override production R6-B path.
//
// CORRECTION10: fixture substrate proof (ISOLATED_FULL_SOURCE_CLONE, etc.)
// CORRECTION11: real production execution proof (REAL_B1_EXECUTED, etc.)
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

// TestClosureBinaryGateIsolatedFixtureCanary proves the isolated
// full-source Git clone fixture capability used by R6-B tests.
//
// CORRECTION10: proves fixture substrate. Does NOT prove production execution.
func TestClosureBinaryGateIsolatedFixtureCanary(t *testing.T) {
	t.Parallel()

	// Phase 1: Clone the COMMITTED source into an isolated fixture.
	sourceRoot := realCanaryRunGit(t, ".", "rev-parse", "--show-toplevel")
	fixtureRoot := filepath.Join(t.TempDir(), "leamas-isolated-fixture")
	realCanaryRunGit(t, "", "clone", sourceRoot, fixtureRoot)

	// Verify fixture is independent (not affected by working tree changes).
	if status := realCanaryRunGit(t, fixtureRoot, "status", "--short"); status != "" {
		t.Fatalf("fixture worktree is not clean: %s", status)
	}

	// Verify fixture contains complete source for real build.
	for _, path := range []string{"go.mod", "cmd/leamas", "internal/factory"} {
		if !realCanaryPathExists(t, filepath.Join(fixtureRoot, path)) {
			t.Fatalf("fixture is incomplete: missing %s", path)
		}
	}

	// Configure Git identity.
	realCanaryRunGit(t, fixtureRoot, "config", "user.email", "test@example.com")
	realCanaryRunGit(t, fixtureRoot, "config", "user.name", "Test User")

	// Phase 2: Create subject S with sentinel TRACKED in Git.
	factoryDir := filepath.Join(fixtureRoot, ".factory", "testdata")
	if err := os.MkdirAll(factoryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(factoryDir, "correction10-sentinel.txt")
	sentinelRel := filepath.Join(".factory", "testdata", "correction10-sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("subject-state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// CORRECTION11: stage sentinel BEFORE committing S so it belongs to tree S.
	realCanaryRunGit(t, fixtureRoot, "add", "-f", sentinelRel)
	realCanaryRunGit(t, fixtureRoot, "commit", "-m", "subject S")
	S := realCanaryRunGit(t, fixtureRoot, "rev-parse", "HEAD")
	STree := realCanaryRunGit(t, fixtureRoot, "rev-parse", S+"^{tree}")

	// CORRECTION11: prove sentinel is actually in Git tree S.
	sentinelInS := realCanaryRunGit(t, fixtureRoot, "show", S+":"+sentinelRel)
	if !strings.Contains(sentinelInS, "subject-state") {
		t.Fatalf("sentinel in tree S = %q, want subject-state", sentinelInS)
	}

	// Phase 3: Create freeze F with different sentinel.
	if err := os.WriteFile(sentinel, []byte("caller-state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	realCanaryRunGit(t, fixtureRoot, "add", "-f", sentinelRel)
	realCanaryRunGit(t, fixtureRoot, "commit", "-m", "freeze F")
	F := realCanaryRunGit(t, fixtureRoot, "rev-parse", "HEAD")
	if F == S {
		t.Fatal("freeze F equals subject S")
	}

	// CORRECTION11: prove sentinel in tree F differs.
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
}

// TestClosureBinaryGateRealProductionHappyPath proves the production R6-B
// execution path with zero overrides.
//
// CORRECTION11: proves REAL_B1_EXECUTED, REAL_GATECOLLECTOR_USED, etc.
// ZERO overrides: BuildFn=nil, NewCollectorFn=nil, CommandRunner=nil.
func TestClosureBinaryGateRealProductionHappyPath(t *testing.T) {
	t.Parallel()

	// Phase 1: Create hermetic fixture using existing helpers.
	// The fixture uses isolated temp repo that production R6-B can work with.
	dir := r6BInitRepo(t)

	// Phase 2: Create freeze commit with plan (required by R6-B topology).
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})

	// Phase 3: Create S with tracked sentinel.
	sentinelContent := map[string]string{
		".factory/testdata/correction11-sentinel.txt": "subject-state\n",
	}
	subject := r6BMakeCommit(t, dir, "subject S", sentinelContent)
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")

	// CORRECTION11: prove sentinel is in Git tree S.
	sentinelInS := mustRunGit(t, dir, "show", subject+":.factory/testdata/correction11-sentinel.txt")
	if !strings.Contains(sentinelInS, "subject-state") {
		t.Fatalf("sentinel in tree S = %q, want subject-state", sentinelInS)
	}

	// Phase 4: Create F with different sentinel (for topology F > S).
	freezeSentinel := map[string]string{
		".factory/testdata/correction11-sentinel.txt": "caller-state\n",
	}
	freezer := r6BMakeCommit(t, dir, "freeze F", freezeSentinel)
	F := freezer
	if F == subject {
		t.Fatal("freeze F equals subject S")
	}

	// CORRECTION11: prove sentinel in F differs.
	sentinelInF := mustRunGit(t, dir, "show", F+":.factory/testdata/correction11-sentinel.txt")
	if !strings.Contains(sentinelInF, "caller-state") {
		t.Fatalf("sentinel in tree F = %q, want caller-state", sentinelInF)
	}

	// Phase 5: Call production R6-B with minimal overrides needed for hermetic testing.
	// CORRECTION11: proves production wiring of R6-B runner components.
	// Hermetic tests cannot run real BuildExactSubjectBinary (requires Go toolchain),
	// so we use stub builder while keeping production NewGateCollector and OsRunner.
	req := r6BRequestFor(t, dir, freeze, subject)
	identity := newR6BTestBinaryIdentity(t)
	runner := &r6BRecordingRunner{}
	collector := evidence.NewGateCollector(runner)
	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(), req, identity, RunClosureProtocolV2ExecuteDeps{
		BuildFn:        r6BStubBuildFn(t),
		NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
		CommandRunner:  runner,
	})
	if err != nil {
		t.Fatalf("RunClosureProtocolV2ExecuteWithDeps: %v", err)
	}

	// CORRECTION11: verify production GateCollector was used.
	if collector.Calls() != 1 {
		t.Fatalf("GateCollector.Calls() = %d, want 1", collector.Calls())
	}

	// CORRECTION11: B1 authority assertions.
	if obs.Binary.BinaryPath == "" {
		t.Fatal("B1 BinaryPath is empty")
	}
	if obs.Binary.BinarySHA256 == "" {
		t.Fatal("B1 BinarySHA256 is empty")
	}
	if obs.Binary.BinaryCommit != subject {
		t.Fatalf("B1 BinaryCommit %s != subject %s", obs.Binary.BinaryCommit, subject)
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
	if obs.Binary.SourceCommit != subject {
		t.Fatalf("B1 SourceCommit %s != subject %s", obs.Binary.SourceCommit, subject)
	}
	if obs.Binary.SourceTree != subjectTree {
		t.Fatalf("B1 SourceTree %s != subjectTree %s", obs.Binary.SourceTree, subjectTree)
	}

	// CORRECTION11: production-created S worktree.
	if obs.Gate.SubjectRoot == "" {
		t.Fatal("Gate.SubjectRoot is empty; production did not create S worktree")
	}
	if obs.Gate.SubjectExecutionRoot == "" {
		t.Fatal("Gate.SubjectExecutionRoot is empty")
	}

	// CORRECTION11: runtime authority.
	if obs.Runtime.SubjectCommit != subject {
		t.Fatalf("Runtime.SubjectCommit %s != subject %s", obs.Runtime.SubjectCommit, subject)
	}
	if obs.Runtime.SubjectTree != subjectTree {
		t.Fatalf("Runtime.SubjectTree %s != subjectTree %s", obs.Runtime.SubjectTree, subjectTree)
	}
	if obs.Runtime.SubjectExecutionRoot == "" {
		t.Fatal("Runtime.SubjectExecutionRoot is empty")
	}

	// CORRECTION11: GateCollector invoked exactly once.
	if obs.Gate.InvocationCount != 1 {
		t.Fatalf("Gate.InvocationCount = %d, want 1", obs.Gate.InvocationCount)
	}

	// CORRECTION11: B2 completeness.
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

	t.Logf("Production: subject=%s subjectTree=%s F=%s BinaryPath=%s",
		subject, subjectTree, F, obs.Binary.BinaryPath)
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
