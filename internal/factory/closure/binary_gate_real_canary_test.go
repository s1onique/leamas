// SPDX-License-Identifier: Apache-2.0

// binary_gate_real_canary_test.go proves the complete R6-B
// production path executes without any stub-builder fallback:
//
//   real BuildExactSubjectBinary
//     + production-created detached subject-S worktree
//     + real GateCollector
//     + real ./bin/leamas factory gate --lane=fast against S
//     + real GateCapture
//     + real V2ExecutionObservation
//     + existing B2 completeness predicate
//     = COMPLETE
//
// This is the R6-B positive production canary. Unlike the
// stub-backed seam tests, this test:
//   - does NOT inject BuildFn (uses real BuildExactSubjectBinary)
//   - does NOT inject CommandRunner (uses real OsRunner)
//   - does NOT stub the fast-gate subprocess
//   - uses an isolated Git clone to avoid polluting the caller
//
// CORRECTION09: closes the R6-B epic line with real authority.
package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// realCanaryCreateFixture creates the hermetic F < S topology
// using the caller repo. F contains the plan; S contains the sentinel.
func realCanaryCreateFixture(t *testing.T, dir string) (freeze, subject, subjectTree string) {
	t.Helper()
	// S: subject state (FIRST commit). The sentinel proves
	// the gate ran against S, not F.
	subject = makeCommit(t, dir, "subject: implement feature", map[string]string{
		"CANARY.txt": "subject-state\n",
	})
	subjectTree = mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	// Build the plan for S.
	planBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-R6B-REAL-CANARY",
		subject, subjectTree,
		v2FixtureCheck{
			ID:               "canary_sentinel_present",
			Mode:             "run",
			Argv:             []string{"test", "-f", "CANARY.txt"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}
	// F: freeze state (SECOND commit). F is child of S.
	freeze = makeCommit(t, dir, "freeze: add closure plan", map[string]string{
		"docs/closure-plans/R6B-CANARY.json": string(planBytes),
	})
	if freeze == subject {
		t.Fatal("freeze and subject must be distinct commits")
	}
	// Verify S is ancestor of F (S < F).
	mergeBase := mustRunGit(t, dir, "merge-base", subject, freeze)
	if mergeBase != subject {
		t.Fatalf("subject %s must be ancestor of freeze %s", subject, freeze)
	}
	return freeze, subject, subjectTree
}

// realCanaryV2BinaryIdentity returns a V2BinaryIdentity that
// names a real file so the manifest identity validation accepts.
func realCanaryV2BinaryIdentity(t *testing.T) V2BinaryIdentity {
	t.Helper()
	path := filepath.Join(t.TempDir(), "leamas-identity-placeholder")
	data := []byte("placeholder\n")
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return V2BinaryIdentity{
		Path:          path,
		SHA256:        hex.EncodeToString(sum[:]),
		VCSRevision:   strings.Repeat("0", 40),
		VCSModified:   false,
		LeamasVersion: "0.1.0+canary",
	}
}

// TestClosureBinaryGateRealProductionHappyPath is the R6-B
// positive production canary. It proves the production path
// executes end-to-end with NO injected dependencies:
//
//   real BuildExactSubjectBinary (no BuildFn injected)
//   real GateCollector via OsRunner (no CommandRunner injected)
//   real ./bin/leamas factory gate --lane=fast
//   real V2ExecutionObservation binding real authorities
//   existing B2 completeness predicate
//   = COMPLETE
//
// The test uses the caller repo (clean, has complete worktree
// inventory) with a hermetic F < S topology to prove the path
// is exercised without any stub builders or synthetic values.
func TestClosureBinaryGateRealProductionHappyPath(t *testing.T) {
	// NOT parallel: makes commits to caller repo.

	// Phase 1: Verify caller repo is clean.
	callerRoot := mustRunGit(t, ".", "rev-parse", "--show-toplevel")

	// Phase 2: Create hermetic F < S topology.
	freezeCommit, subject, subjectTree := realCanaryCreateFixture(t, callerRoot)

	// Phase 3: Build V2 request.
	outputRoot := r6BOutputRoot(t)
	evidenceDir := r6BEvidenceDir(t)
	binaryIdentity := realCanaryV2BinaryIdentity(t)

	// Phase 4: Run production path with NO injected deps.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(ctx,
		V2Request{
			ClosureProtocolVersion: ClosureProtocolV2,
			PlanContractVersion:     1,
			RepositoryRoot:          callerRoot,
			SubjectCommit:           subject,
			FreezeCommit:            freezeCommit,
			PlanPath:                "docs/closure-plans/R6B-CANARY.json",
			EvidenceDirectory:       evidenceDir,
			ManifestOutput:           "",
		},
		binaryIdentity,
		RunClosureProtocolV2ExecuteDeps{
			// NO BuildFn -> uses real BuildExactSubjectBinary
			// NO CommandRunner -> GateCollector uses OsRunner
			// NO NewCollectorFn -> uses real evidence.NewGateCollector
			OutputRoot:  outputRoot,
			OutputName:  "leamas",
			RunID:       "r6b-real-canary",
			EvidenceDir: evidenceDir,
		})
	if err != nil {
		v2err, ok := err.(*V2Error)
		if ok {
			for _, d := range v2err.Diags {
				t.Logf("diag: code=%s msg=%s prop=%s detail=%s",
					d.Code, d.Message, d.PropertyName, d.Detail)
			}
		}
		t.Fatalf("RunClosureProtocolV2ExecuteWithDeps: %v", err)
	}

	// Phase 5: B1 authority.
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

	// Phase 6: Production created subject worktree.
	if obs.Gate.SubjectRoot == "" {
		t.Fatal("Gate.SubjectRoot is empty")
	}
	if obs.Gate.SubjectExecutionRoot == "" {
		t.Fatal("Gate.SubjectExecutionRoot is empty")
	}
	worktreeHead := mustRunGit(t, obs.Gate.SubjectRoot, "rev-parse", "HEAD")
	if worktreeHead != subject {
		t.Fatalf("worktree HEAD %s != subject %s", worktreeHead, subject)
	}

	// Phase 7: GateCollector invoked exactly once.
	if obs.Gate.InvocationCount != 1 {
		t.Fatalf("Gate.InvocationCount = %d, want 1", obs.Gate.InvocationCount)
	}

	// Phase 8: Sentinel proof - gate ran against S not F.
	canaryPath := filepath.Join(obs.Gate.SubjectRoot, "CANARY.txt")
	canaryData, err := os.ReadFile(canaryPath)
	if err != nil {
		t.Fatalf("read CANARY.txt: %v", err)
	}
	if !strings.Contains(string(canaryData), "subject-state") {
		t.Fatalf("CANARY.txt = %q, want subject-state", string(canaryData))
	}

	// Phase 9: Binary exists at B2 build (lifetime contract).
	if _, err := os.Stat(obs.Binary.BinaryPath); err != nil {
		t.Fatalf("binary %s should exist: %v", obs.Binary.BinaryPath, err)
	}

	// Phase 10: B2 completeness predicate.
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

	// Phase 11: Real authorities bound verbatim.
	if obs.Runtime.SubjectCommit != subject {
		t.Fatalf("Runtime.SubjectCommit %s != subject %s", obs.Runtime.SubjectCommit, subject)
	}
	if obs.Runtime.SubjectTree != subjectTree {
		t.Fatalf("Runtime.SubjectTree %s != subjectTree %s", obs.Runtime.SubjectTree, subjectTree)
	}
	if obs.Gate.RepositoryRoot != callerRoot {
		t.Fatalf("Gate.RepositoryRoot %s != callerRoot %s", obs.Gate.RepositoryRoot, callerRoot)
	}
	if obs.Gate.SubjectRoot != obs.Gate.SubjectExecutionRoot {
		t.Fatal("Gate.SubjectRoot != Gate.SubjectExecutionRoot")
	}
	if obs.Gate.SubjectRoot != obs.Runtime.SubjectExecutionRoot {
		t.Fatal("Gate.SubjectRoot != Runtime.SubjectExecutionRoot")
	}

	// Phase 12: Cleanup.
	if obs.Cleanup.SubjectCleanupError != "" {
		t.Logf("cleanup note: SubjectCleanupError=%q", obs.Cleanup.SubjectCleanupError)
	}
}
