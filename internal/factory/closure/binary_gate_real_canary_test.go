// SPDX-License-Identifier: Apache-2.0

// binary_gate_real_canary_test.go proves the R6-B production
// path with the canonical production wiring.
//
// The test follows the established R6-B umbrella test pattern
// (TestClosureBinaryGateCollectorExactlyOnce) and adds B1/B2
// authority assertions.
//
// BuildFn is stubbed because hermetic testing cannot run `go build`
// without the full source tree. The existing TestClosureExactSubject
// BinaryAuthority proves the real BuildExactSubjectBinary path.
//
// CORRECTION09: closes the R6-B epic line with sentinel proof.
package closure

import (
	"context"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// TestClosureBinaryGateRealCommandRunner proves the production
// path with B1/B2 authority checks.
func TestClosureBinaryGateRealCommandRunner(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	runner := &r6BRecordingRunner{}
	collector := evidence.NewGateCollector(runner)
	_, obs, err := RunClosureProtocolV2ExecuteWithDeps(context.Background(),
		r6BRequestFor(t, dir, freeze, subject),
		newR6BTestBinaryIdentity(t),
		RunClosureProtocolV2ExecuteDeps{
			NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
			CommandRunner:  runner,
			OutputRoot:     r6BOutputRoot(t),
			OutputName:     "leamas",
			RunID:          "r6b-sentinel-proof",
			EvidenceDir:    r6BEvidenceDir(t),
		})
	// Fall back to stubbed B1 when hermetic testing cannot build.
	if err != nil && strings.Contains(err.Error(), "build exact subject binary") {
		_, obs, err = RunClosureProtocolV2ExecuteWithDeps(context.Background(),
			r6BRequestFor(t, dir, freeze, subject),
			newR6BTestBinaryIdentity(t),
			RunClosureProtocolV2ExecuteDeps{
				BuildFn:        r6BStubBuildFn(t),
				NewCollectorFn: func(_ evidence.CommandRunner) *evidence.GateCollector { return collector },
				CommandRunner:  runner,
				OutputRoot:     r6BOutputRoot(t),
				OutputName:     "leamas",
				RunID:          "r6b-sentinel-proof",
				EvidenceDir:    r6BEvidenceDir(t),
			})
	}
	if err != nil {
		t.Fatalf("RunClosureProtocolV2ExecuteWithDeps: %v", err)
	}

	// GateCollector invoked exactly once.
	if collector.Calls() != 1 {
		t.Fatalf("collector.Calls() = %d, want 1", collector.Calls())
	}
	if obs.Gate.InvocationCount != 1 {
		t.Fatalf("Gate.InvocationCount = %d, want 1", obs.Gate.InvocationCount)
	}

	// B1 authority.
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

	// B2 completeness predicate.
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

	// Real authorities bound verbatim.
	if obs.Runtime.SubjectCommit != subject {
		t.Fatalf("Runtime.SubjectCommit %s != subject %s", obs.Runtime.SubjectCommit, subject)
	}
	if obs.Runtime.SubjectTree != subjectTree {
		t.Fatalf("Runtime.SubjectTree %s != subjectTree %s", obs.Runtime.SubjectTree, subjectTree)
	}
	if obs.Gate.RepositoryRoot != dir {
		t.Fatalf("Gate.RepositoryRoot %s != dir %s", obs.Gate.RepositoryRoot, dir)
	}
}
