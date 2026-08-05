// SPDX-License-Identifier: Apache-2.0

package closure

// v2_working_plan_full_runner_test.go exercises the working-plan
// assertion through the full runner against a meaningful
// S < F < D repository, as required by Phase 6 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-PATH-AUTHORITY01.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestV2WorkingPlan_FullRunnerMismatch constructs:
//
//	S: subject implementation (subject-only.txt)
//	F: child of S, valid plan added at docs/closure-plans/PATH.json
//	D: child of F, mutated plan at docs/closure-plans/PATH.json
//	HEAD = D
//
// With assertion = D:P, the runner must reject with
// working_plan_mismatch, must NOT invoke the executor, and must
// NOT write the manifest.
//
// Without the assertion, the runner must use F:P and complete.
func TestV2WorkingPlan_FullRunnerMismatch(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	frozenBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-WORKING-MISMATCH",
		subject, subjectTree, v2FixtureCheck{
			ID:               "subject_only_present",
			Mode:             "run",
			Argv:             []string{"test", "-f", "subject-only.txt"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}
	freeze := makeCommit(t, dir, "freeze: add plan", map[string]string{
		"docs/closure-plans/PATH.json": string(frozenBytes),
	})
	// D: child of F with a DIFFERENT plan bytes.
	mutatedBytes := []byte(`{"contract_version": 1, "act_id": "ACT-MUTATED"}`)
	d := makeCommit(t, dir, "descendant: mutate plan", map[string]string{
		"docs/closure-plans/PATH.json": string(mutatedBytes),
	})
	// Place the mutated bytes at the working-plan location
	// (outside the repo). The runner compares them against
	// the frozen F:P bytes.
	working := filepath.Join(t.TempDir(), "WORKING.json")
	if err := os.WriteFile(working, mutatedBytes, 0o644); err != nil {
		t.Fatalf("write working: %v", err)
	}
	req := V2Request{
		ClosureProtocolVersion:       ClosureProtocolV2,
		PlanContractVersion:          1,
		RepositoryRoot:               dir,
		SubjectCommit:                subject,
		FreezeCommit:                 freeze,
		PlanPath:                     "docs/closure-plans/PATH.json",
		EvidenceDirectory:            t.TempDir(),
		ManifestOutput:               filepath.Join(t.TempDir(), "manifest.json"),
		OptionalWorkingPlanAssertion: working,
	}
	_, err = RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("working plan mismatch must reject (HEAD=%s, D=descendant of F)", d)
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	// The detached-path check fires first if working is
	// inside the repo; this test places it outside so the
	// mismatch path is the one that triggers.
	if !v2err.Diags.HasCode(V2CodeWorkingPlanMismatch) {
		t.Fatalf("expected working_plan_mismatch, got %v", v2err.Diags.Codes())
	}
	if _, statErr := os.Stat(req.ManifestOutput); statErr == nil {
		t.Fatalf("manifest must not be written on working plan mismatch")
	}
}

// TestV2WorkingPlan_FullRunnerMatch asserts that a working
// plan whose bytes match F:P passes the assertion and the
// runner completes (using F:P regardless of HEAD = D).
func TestV2WorkingPlan_FullRunnerMatch(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	frozenBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-WORKING-MATCH",
		subject, subjectTree, v2FixtureCheck{
			ID:               "subject_only_present",
			Mode:             "run",
			Argv:             []string{"test", "-f", "subject-only.txt"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}
	freeze := makeCommit(t, dir, "freeze: add plan", map[string]string{
		"docs/closure-plans/PATH.json": string(frozenBytes),
	})
	makeCommit(t, dir, "descendant: mutate plan", map[string]string{
		"docs/closure-plans/PATH.json": `{"contract_version": 1, "act_id": "ACT-MUTATED"}`,
	})
	// Working-plan bytes match F:P, so the assertion passes.
	working := filepath.Join(t.TempDir(), "WORKING.json")
	if err := os.WriteFile(working, frozenBytes, 0o644); err != nil {
		t.Fatalf("write working: %v", err)
	}
	req := V2Request{
		ClosureProtocolVersion:       ClosureProtocolV2,
		PlanContractVersion:          1,
		RepositoryRoot:               dir,
		SubjectCommit:                subject,
		FreezeCommit:                 freeze,
		PlanPath:                     "docs/closure-plans/PATH.json",
		EvidenceDirectory:            t.TempDir(),
		ManifestOutput:               filepath.Join(t.TempDir(), "manifest.json"),
		OptionalWorkingPlanAssertion: working,
	}
	manifest, err := RunClosureProtocolV2(context.Background(), req)
	if err != nil {
		t.Fatalf("matching working plan must pass: %v", err)
	}
	// Manifest must bind F:P bytes, not D:P bytes.
	if manifest.PlanSHA256 == "" {
		t.Fatalf("plan SHA-256 missing")
	}
	wantBlob := mustRunGit(t, dir, "rev-parse", freeze+":docs/closure-plans/PATH.json")
	if manifest.PlanBlob != wantBlob {
		t.Fatalf("plan blob mismatch: got=%s want=%s", manifest.PlanBlob, wantBlob)
	}
}
