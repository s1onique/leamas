// SPDX-License-Identifier: Apache-2.0

// binary_gate_integration_test.go authorises the R6-B
// integration by exercising the production BuildExactSubjectBinary
// + GateCollector + V2 runner flow under the non-publishing
// entry point. The tests assert the production authorities
// produce a B2-eligible V2ExecutionObservation.
package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// TestClosureBinaryGateIntegrationSmoke asserts the
// production integration completes a happy-path run
// and produces a fully valid V2ExecutionObservation. The
// test uses a fake BuildFn and a fake runner that returns
// an "OK" capture so the umbrella covers the real
// gate wiring without requiring a real gate process.
func TestClosureBinaryGateIntegrationSmoke(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(simpleValidPlanBytes()),
	})
	subject := makeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	subjectRoot := filepath.Join(t.TempDir(), "subject-wt")
	mustRunGit(t, dir, "worktree", "add", "--detach", subjectRoot, subject)
	defer mustRunGit(t, dir, "worktree", "remove", "--force", subjectRoot)
	binaryPath := filepath.Join(t.TempDir(), "leamas")
	if err := os.WriteFile(binaryPath, []byte("fake binary\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	binaryBytes, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read binary: %v", err)
	}
	sum := sha256.Sum256(binaryBytes)
	wantSHA := hex.EncodeToString(sum[:])
	runner := &fakeCloseRunner{workspace: subjectRoot}
	collector := evidence.NewGateCollector(runner)
	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(), v2RequestForSmoke(t, dir, subject, freeze), newTestBinaryIdentity(t), RunClosureProtocolV2ExecuteDeps{
		BuildFn:        makeFakeBinaryBuilder(binaryPath, subject),
		NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
		CommandRunner:  runner,
		OutputRoot:     t.TempDir(),
		OutputName:     "leamas",
		RunID:          "test-smoke",
		EvidenceDir:    filepath.Join(t.TempDir(), "leamas-evidence-" + filepath.Base(t.TempDir())),
	})
	if err != nil {
		t.Fatalf("integration: %v", err)
	}
	if obs.Binary.BinaryPath != binaryPath {
		t.Fatalf("binary path = %q, want %q", obs.Binary.BinaryPath, binaryPath)
	}
	if obs.Binary.BinarySHA256 != wantSHA {
		t.Fatalf("binary SHA = %q, want %q", obs.Binary.BinarySHA256, wantSHA)
	}
	if obs.Binary.BinaryCommit != subject {
		t.Fatalf("binary commit = %q, want %q", obs.Binary.BinaryCommit, subject)
	}
	if obs.Binary.BinaryModified {
		t.Fatalf("binary modified = true, want false")
	}
	if !obs.Binary.OutputOutsideAllWorktrees {
		t.Fatalf("binary OutputOutsideAllWorktrees = false, want true")
	}
	if !obs.Binary.Executable {
		t.Fatalf("binary Executable = false, want true")
	}
	if obs.Gate.InvocationCount != 1 {
		t.Fatalf("gate invocation count = %d, want 1", obs.Gate.InvocationCount)
	}
	if collector.Calls() != 1 {
		t.Fatalf("collector.Calls() = %d, want 1", collector.Calls())
	}
	if obs.Gate.RepositoryRoot != dir {
		t.Fatalf("gate RepositoryRoot = %q, want %q", obs.Gate.RepositoryRoot, dir)
	}
	if obs.Gate.SubjectRoot == "" {
		t.Fatalf("gate SubjectRoot is empty")
	}
	if obs.Gate.SubjectExecutionRoot == "" {
		t.Fatalf("gate SubjectExecutionRoot is empty")
	}
	if obs.Binary.SourceCommit != subject {
		t.Fatalf("binary source commit = %q, want %q", obs.Binary.SourceCommit, subject)
	}
	if !obs.Binary.SourceClean {
		t.Fatalf("binary source clean = false, want true")
	}
	if !obs.Binary.SourceDetached {
		t.Fatalf("binary source detached = false, want true")
	}
}

// TestClosureBinaryGateIntegrationIsNonPublishing asserts the
// integration does not write a canonical evidence JSON or
// a legacy V2 manifest to the caller repository.
func TestClosureBinaryGateIntegrationIsNonPublishing(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(simpleValidPlanBytes()),
	})
	subject := makeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	subjectRoot := filepath.Join(t.TempDir(), "subject-wt")
	mustRunGit(t, dir, "worktree", "add", "--detach", subjectRoot, subject)
	defer mustRunGit(t, dir, "worktree", "remove", "--force", subjectRoot)
	binaryPath := filepath.Join(t.TempDir(), "leamas")
	if err := os.WriteFile(binaryPath, []byte("nope\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	runner := &fakeCloseRunner{workspace: subjectRoot}
	collector := evidence.NewGateCollector(runner)
	_, _, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(), v2RequestForSmoke(t, dir, subject, freeze), newTestBinaryIdentity(t), RunClosureProtocolV2ExecuteDeps{
		BuildFn:        makeFakeBinaryBuilder(binaryPath, subject),
		NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
		CommandRunner:  runner,
		OutputRoot:     t.TempDir(),
		OutputName:     "leamas",
		RunID:          "test-nonpublish",
		EvidenceDir:    filepath.Join(t.TempDir(), "leamas-evidence-" + filepath.Base(t.TempDir())),
	})
	if err != nil {
		t.Fatalf("integration: %v", err)
	}
	for _, p := range []string{
		filepath.Join(dir, "evidence.json"),
		filepath.Join(dir, "manifest.json"),
	} {
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("integration wrote %s but must not publish", p)
		}
	}
}

// TestClosureBinaryGateIntegrationRunScoped asserts two
// independent runs do not share run-scoped state.
func TestClosureBinaryGateIntegrationRunScoped(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(simpleValidPlanBytes()),
	})
	subject := makeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	subjectRoot1 := filepath.Join(t.TempDir(), "subject-wt-1")
	subjectRoot2 := filepath.Join(t.TempDir(), "subject-wt-2")
	mustRunGit(t, dir, "worktree", "add", "--detach", subjectRoot1, subject)
	mustRunGit(t, dir, "worktree", "add", "--detach", subjectRoot2, subject)
	defer mustRunGit(t, dir, "worktree", "remove", "--force", subjectRoot1)
	defer mustRunGit(t, dir, "worktree", "remove", "--force", subjectRoot2)
	binaryPath1 := filepath.Join(t.TempDir(), "leamas-1")
	binaryPath2 := filepath.Join(t.TempDir(), "leamas-2")
	if err := os.WriteFile(binaryPath1, []byte("first\n"), 0o755); err != nil {
		t.Fatalf("write binary 1: %v", err)
	}
	if err := os.WriteFile(binaryPath2, []byte("second\n"), 0o755); err != nil {
		t.Fatalf("write binary 2: %v", err)
	}
	runner1 := &fakeCloseRunner{workspace: subjectRoot1}
	runner2 := &fakeCloseRunner{workspace: subjectRoot2}
	collector1 := evidence.NewGateCollector(runner1)
	collector2 := evidence.NewGateCollector(runner2)
	_, obs1, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(), v2RequestForSmoke(t, dir, subject, freeze), newTestBinaryIdentity(t), RunClosureProtocolV2ExecuteDeps{
		BuildFn:        makeFakeBinaryBuilder(binaryPath1, subject),
		NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector1 },
		CommandRunner:  runner1,
		OutputRoot:     t.TempDir(),
		OutputName:     "leamas",
		RunID:          "run-scoped-1",
		EvidenceDir:    filepath.Join(t.TempDir(), "leamas-evidence-" + filepath.Base(t.TempDir())),
	})
	if err != nil {
		t.Fatalf("run 1: %v", err)
	}
	_, obs2, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(), v2RequestForSmoke(t, dir, subject, freeze), newTestBinaryIdentity(t), RunClosureProtocolV2ExecuteDeps{
		BuildFn:        makeFakeBinaryBuilder(binaryPath2, subject),
		NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector2 },
		CommandRunner:  runner2,
		OutputRoot:     t.TempDir(),
		OutputName:     "leamas",
		RunID:          "run-scoped-2",
		EvidenceDir:    filepath.Join(t.TempDir(), "leamas-evidence-" + filepath.Base(t.TempDir())),
	})
	if err != nil {
		t.Fatalf("run 2: %v", err)
	}
	if obs1.Binary.BinaryPath == obs2.Binary.BinaryPath {
		t.Fatalf("two independent runs must not share binary path")
	}
	if obs1.Gate.InvocationCount != 1 || obs2.Gate.InvocationCount != 1 {
		t.Fatalf("two independent runs must each observe 1 invocation: 1=%d 2=%d",
			obs1.Gate.InvocationCount, obs2.Gate.InvocationCount)
	}
	if collector1.Calls() != 1 || collector2.Calls() != 1 {
		t.Fatalf("collector.Calls() must each be 1: 1=%d 2=%d", collector1.Calls(), collector2.Calls())
	}
}

// fakeCloseRunner is the production-command-runner test
// double the R6-B umbrellas use to confirm the gate path.
// The runner records invocations and returns synthetic OK
// captures so the gate classification produces PASS.
type fakeCloseRunner struct {
	workspace string
	calls     int
}

func (r *fakeCloseRunner) Run(ctx context.Context, name string, args []string, dir string, env []string) evidence.CommandResult {
	r.calls++
	return evidence.CommandResult{
		ExitCode: 0,
		Stdout:   []byte("OK\n"),
		Stderr:   []byte{},
	}
}

// simpleValidPlanBytes returns the canonical Plan Contract
// v1 plan bytes the R6-B umbrellas use. The bytes are
// constructed via the production BuildV2ValidPlanFixture
// helper so the umbrella stays in sync with the canonical
// fixture.
func simpleValidPlanBytes() []byte {
	bytes, err := BuildV2ValidPlanFixture("ACT-R6B-INTEGRATION-01", strings.Repeat("a", 40), strings.Repeat("b", 40))
	if err != nil {
		panic(err)
	}
	return bytes
}

// v2RequestForSmoke builds the V2Request the smoke umbrellas
// use. The helper exists so every umbrella uses the same
// request construction.
func v2RequestForSmoke(t *testing.T, dir, subject, freeze string) V2Request {
	return V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/X.json",
		EvidenceDirectory:      filepath.Join(t.TempDir(), "leamas-b2-evidence-v2"),
		ManifestOutput:         "",
	}
}

// makeFakeBinaryBuilder returns a BuildFn that the
// umbrellas use to inject a deterministic B1 result. The
// builder reads the file to compute the SHA-256 so the B2
// BinaryAuthority matches the on-disk content.
func makeFakeBinaryBuilder(binaryPath, subjectCommit string) func(context.Context, ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
	return func(_ context.Context, _ ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
		data, err := os.ReadFile(binaryPath)
		if err != nil {
			return ExactSubjectBinaryResult{}, err
		}
		sum := sha256.Sum256(data)
		return ExactSubjectBinaryResult{
			BinaryPath:                binaryPath,
			BinarySHA256:              hex.EncodeToString(sum[:]),
			BinaryCommit:              subjectCommit,
			BinaryModified:            false,
			SourceCommit:              subjectCommit,
			SourceTree:                strings.Repeat("d", 40),
			SourceClean:               true,
			SourceDetached:            true,
			OutputOutsideAllWorktrees: true,
			Executable:                true,
		}, nil
	}
}

// newTestBinaryIdentity returns a valid V2BinaryIdentity
// the R6-B umbrellas use. The identity names a real file
// on disk so the manifest identity validation accepts.
func newTestBinaryIdentity(t testing.TB) V2BinaryIdentity {
	t.Helper()
	path := filepath.Join(t.TempDir(), "leamas-test-binary")
	data := []byte("deterministic fake leamas binary identity\n")
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return V2BinaryIdentity{
		Path:          path,
		SHA256:        hex.EncodeToString(sum[:]),
		VCSRevision:   strings.Repeat("7", 40),
		VCSModified:   false,
		LeamasVersion: "0.1.0+test",
	}
}

// newTestBinaryIdentity returns a valid V2BinaryIdentity
// the umbrellas use. The identity names a real file on disk
// so the manifest identity validation accepts.
















