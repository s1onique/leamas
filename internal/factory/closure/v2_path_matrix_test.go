// SPDX-License-Identifier: Apache-2.0

package closure

// v2_path_matrix_test.go enumerates the authoritative path
// matrix required by Phase 5 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-PATH-AUTHORITY01.
//
// Every case constructs a real repository in a temp
// directory and asserts the runner / path authority returns
// the expected verdict (accept or reject with the typed
// code).

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// newRepoForPath builds a tiny repository with a subject
// commit and a freeze commit, and returns the repo dir,
// subject commit OID, subject tree OID, and freeze commit OID.
func newRepoForPath(t *testing.T) (dir, subject, subjectTree, freeze string) {
	t.Helper()
	dir = initRepo(t)
	subject = makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject\n",
	})
	subjectTree = mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	freeze = makeCommit(t, dir, "freeze", map[string]string{"freeze-only.txt": "freeze\n"})
	return dir, subject, subjectTree, freeze
}

// buildValidPlanAndCommit builds a valid Plan Contract v1
// document for the supplied repo and commits it under a fresh
// freeze commit; returns the plan path under the freeze tree,
// the plan bytes, and the freeze OID.
func buildValidPlanAndCommit(t *testing.T, dir, subject, subjectTree string) (planPath string, planBytes []byte, freeze string) {
	t.Helper()
	planBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-PATH-AUTHORITY", subject, subjectTree, v2FixtureCheck{
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
	freeze = makeCommit(t, dir, "freeze: add plan", map[string]string{
		"docs/closure-plans/PATH.json": string(planBytes),
	})
	return "docs/closure-plans/PATH.json", planBytes, freeze
}

// writeWorkingPlan writes the plan bytes to a temp location
// outside the repo and returns the path.
func writeWorkingPlan(t *testing.T, dir string, bytes []byte) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "working-plan.json")
	if err := os.WriteFile(out, bytes, 0o644); err != nil {
		t.Fatalf("write working plan: %v", err)
	}
	return out
}

// TestV2PathMatrix_AbsoluteDetachedAccepted asserts a normal
// absolute detached path passes the detached-path check.
func TestV2PathMatrix_AbsoluteDetachedAccepted(t *testing.T) {
	dir, subject, subjectTree, _ := newRepoForPath(t)
	_, planBytes, freeze := buildValidPlanAndCommit(t, dir, subject, subjectTree)
	evidence := t.TempDir()
	manifest := filepath.Join(t.TempDir(), "manifest.json")
	working := writeWorkingPlan(t, dir, planBytes)
	req := V2Request{
		ClosureProtocolVersion:       ClosureProtocolV2,
		PlanContractVersion:          1,
		RepositoryRoot:               dir,
		SubjectCommit:                subject,
		FreezeCommit:                 freeze,
		PlanPath:                     "docs/closure-plans/PATH.json",
		EvidenceDirectory:            evidence,
		ManifestOutput:               manifest,
		OptionalWorkingPlanAssertion: working,
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err != nil {
		t.Fatalf("absolute detached path must pass: %v", err)
	}
}

// TestV2PathMatrix_InsideRepoRejected asserts an evidence
// path inside the repo is rejected with
// V2CodeEvidencePathNotDetached.
func TestV2PathMatrix_InsideRepoRejected(t *testing.T) {
	dir, subject, subjectTree, _ := newRepoForPath(t)
	_, _, freeze := buildValidPlanAndCommit(t, dir, subject, subjectTree)
	evidence := filepath.Join(dir, "inside-repo")
	if err := os.MkdirAll(evidence, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/PATH.json",
		EvidenceDirectory:      evidence,
		ManifestOutput:         manifest,
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("inside-repo path must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok || !v2err.Diags.HasCode(V2CodeEvidencePathNotDetached) {
		t.Fatalf("expected evidence_path_not_detached, got %v", err)
	}
}

// TestV2PathMatrix_RelativeRepoRootRejected asserts a
// relative repository root is rejected.
func TestV2PathMatrix_RelativeRepoRootRejected(t *testing.T) {
	dir, subject, subjectTree, _ := newRepoForPath(t)
	_, _, freeze := buildValidPlanAndCommit(t, dir, subject, subjectTree)
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         "relative/path",
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/PATH.json",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("relative repository_root must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeInvalidPlanPath) && !v2err.Diags.HasCode(V2CodeRequestIncomplete) {
		t.Fatalf("expected invalid_plan_path or request_incomplete, got %v", v2err.Diags.Codes())
	}
}

// TestV2PathMatrix_NonexistentLeafSymlinkIntoRepoRejected
// asserts that an evidence directory under a nonexistent
// path whose deepest existing ancestor is a symlink into the
// repository is rejected. The runner must resolve the
// symlink and reject the path as inside-repo.
func TestV2PathMatrix_NonexistentLeafSymlinkIntoRepoRejected(t *testing.T) {
	dir, subject, subjectTree, _ := newRepoForPath(t)
	_, _, freeze := buildValidPlanAndCommit(t, dir, subject, subjectTree)
	// Create a symlink pointing INTO the repository, then
	// place the evidence directory under a nonexistent leaf
	// below it.
	external := t.TempDir()
	link := filepath.Join(external, "into_repo")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	evidence := filepath.Join(link, "evidence")
	manifest := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/PATH.json",
		EvidenceDirectory:      evidence,
		ManifestOutput:         manifest,
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("evidence under symlink-into-repo must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok || !v2err.Diags.HasCode(V2CodeEvidencePathNotDetached) {
		t.Fatalf("expected evidence_path_not_detached, got %v", err)
	}
}

// TestV2PathMatrix_BenignSymlinkOutsideRepoAccepted asserts
// an evidence directory under a symlink that points outside
// the repository (and outside its .git) is accepted.
func TestV2PathMatrix_BenignSymlinkOutsideRepoAccepted(t *testing.T) {
	dir, subject, subjectTree, _ := newRepoForPath(t)
	_, planBytes, freeze := buildValidPlanAndCommit(t, dir, subject, subjectTree)
	external := t.TempDir()
	target := filepath.Join(external, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	link := filepath.Join(external, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	evidence := filepath.Join(link, "evidence")
	manifest := filepath.Join(t.TempDir(), "manifest.json")
	working := writeWorkingPlan(t, dir, planBytes)
	req := V2Request{
		ClosureProtocolVersion:       ClosureProtocolV2,
		PlanContractVersion:          1,
		RepositoryRoot:               dir,
		SubjectCommit:                subject,
		FreezeCommit:                 freeze,
		PlanPath:                     "docs/closure-plans/PATH.json",
		EvidenceDirectory:            evidence,
		ManifestOutput:               manifest,
		OptionalWorkingPlanAssertion: working,
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err != nil {
		t.Fatalf("benign symlink outside repo must pass: %v", err)
	}
}

// TestV2PathMatrix_WorkingPlanMatchingFP asserts a working
// plan whose bytes match F:P passes the assertion.
func TestV2PathMatrix_WorkingPlanMatchingFP(t *testing.T) {
	dir, subject, subjectTree, _ := newRepoForPath(t)
	planPath, planBytes, freeze := buildValidPlanAndCommit(t, dir, subject, subjectTree)
	working := writeWorkingPlan(t, dir, planBytes)
	req := V2Request{
		ClosureProtocolVersion:       ClosureProtocolV2,
		PlanContractVersion:          1,
		RepositoryRoot:               dir,
		SubjectCommit:                subject,
		FreezeCommit:                 freeze,
		PlanPath:                     planPath,
		EvidenceDirectory:            t.TempDir(),
		ManifestOutput:               filepath.Join(t.TempDir(), "manifest.json"),
		OptionalWorkingPlanAssertion: working,
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err != nil {
		t.Fatalf("matching working plan must pass: %v", err)
	}
}

// TestV2PathMatrix_WorkingPlanDifferingFromFP asserts a
// working plan whose bytes differ from F:P is rejected with
// V2CodeWorkingPlanMismatch.
func TestV2PathMatrix_WorkingPlanDifferingFromFP(t *testing.T) {
	dir, subject, subjectTree, _ := newRepoForPath(t)
	planPath, _, freeze := buildValidPlanAndCommit(t, dir, subject, subjectTree)
	working := writeWorkingPlan(t, dir, []byte("different-bytes"))
	req := V2Request{
		ClosureProtocolVersion:       ClosureProtocolV2,
		PlanContractVersion:          1,
		RepositoryRoot:               dir,
		SubjectCommit:                subject,
		FreezeCommit:                 freeze,
		PlanPath:                     planPath,
		EvidenceDirectory:            t.TempDir(),
		ManifestOutput:               filepath.Join(t.TempDir(), "manifest.json"),
		OptionalWorkingPlanAssertion: working,
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("differing working plan must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok || !v2err.Diags.HasCode(V2CodeWorkingPlanMismatch) {
		t.Fatalf("expected working_plan_mismatch, got %v", err)
	}
}

// TestV2PathMatrix_WorkingPlanInsideRepoRejected asserts a
// working plan path inside the repo is rejected with
// V2CodeWorkingPlanPathInvalid (the detached check fires
// before the byte compare).
func TestV2PathMatrix_WorkingPlanInsideRepoRejected(t *testing.T) {
	dir, subject, subjectTree, _ := newRepoForPath(t)
	_, planBytes, freeze := buildValidPlanAndCommit(t, dir, subject, subjectTree)
	working := filepath.Join(dir, "docs", "closure-plans", "WORKING.json")
	if err := os.MkdirAll(filepath.Dir(working), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(working, planBytes, 0o644); err != nil {
		t.Fatalf("write: %v", err)
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
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("working plan inside repo must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok || !v2err.Diags.HasCode(V2CodeWorkingPlanPathInvalid) {
		t.Fatalf("expected working_plan_path_invalid, got %v", err)
	}
}

// TestV2PathMatrix_ManifestInsideRepoRejected asserts the
// manifest output inside the repository is rejected with
// V2CodeManifestPathNotDetached.
func TestV2PathMatrix_ManifestInsideRepoRejected(t *testing.T) {
	dir, subject, subjectTree, _ := newRepoForPath(t)
	_, _, freeze := buildValidPlanAndCommit(t, dir, subject, subjectTree)
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/PATH.json",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(dir, "manifest.json"),
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("manifest inside repo must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok || !v2err.Diags.HasCode(V2CodeManifestPathNotDetached) {
		t.Fatalf("expected manifest_path_not_detached, got %v", err)
	}
}

// TestV2PathMatrix_NULByteRejected asserts a path containing
// a NUL byte is rejected with V2CodeInvalidPlanPath.
func TestV2PathMatrix_NULByteRejected(t *testing.T) {
	dir, subject, subjectTree, _ := newRepoForPath(t)
	_, _, freeze := buildValidPlanAndCommit(t, dir, subject, subjectTree)
	evidence := t.TempDir() + "\x00bad"
	_, err := RunClosureProtocolV2(context.Background(), V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/PATH.json",
		EvidenceDirectory:      evidence,
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	})
	if err == nil {
		t.Fatalf("NUL byte path must reject")
	}
}

// TestV2PathResolution_DeepestExistingAncestor verifies the
// canonical resolver finds the deepest existing ancestor and
// resolves symlinks for nonexistent leaf paths.
