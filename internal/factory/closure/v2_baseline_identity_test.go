// SPDX-License-Identifier: Apache-2.0

package closure

// v2_baseline_identity_test.go exercises the repository-bound
// baseline Git-object truth validation added by
// CLOSURE-V2-IDENTITY-AUTHORITY-CORRECTION01.
//
// The v2 runner must verify that:
//   - baseline.commit_oid exists as a commit in the repository
//   - baseline.tree_oid exists as a tree in the repository
//   - baseline.commit_oid^{tree} == baseline.tree_oid
//
// Each failure maps to a dedicated stable V2Code so downstream
// tooling can distinguish baseline failures from blob loading failures.
//
// Splitting this from closure_protocol_v2_hermetic_test.go keeps
// each file focused and under the LLM-friendly 400-line threshold.

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// TestV2RunnerRejectsPlaceholderTreeOID asserts that a plan with a fabricated
// baseline.tree_oid (e.g., 0000000000000000000000000000000000000000) is rejected
// by the v2 runner with the exact code baseline_tree_not_found.
//
// This is a regression test for CLOSURE-V2-IDENTITY-AUTHORITY-CORRECTION01:
// a plan with a valid commit OID but a placeholder/fabricated tree OID must
// be rejected before any executor call with the dedicated V2Code.
func TestV2RunnerRejectsPlaceholderTreeOID(t *testing.T) {
	dir := initRepo(t)
	// Create subject commit with implementation.
	subject := makeCommit(t, dir, "subject implementation", map[string]string{
		"src/lib.go": "package lib\n",
	})
	// Build a plan with a FABRICATED tree OID (40 zeroes).
	// The commit OID will be real (subject), but the tree OID will be fake.
	planBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-PLACEHOLDER-TREE-01",
		subject, strings.Repeat("0", 40), v2FixtureCheck{
			ID:               "placeholder_test",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}

	// Freeze commit includes the plan with fabricated tree OID.
	freeze := makeCommit(t, dir, "freeze: add closure plan", map[string]string{
		"docs/closure-plans/PLACEHOLDER-TREE.json": string(planBytes),
	})

	evidenceDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:   1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:          freeze,
		PlanPath:               "docs/closure-plans/PLACEHOLDER-TREE.json",
		EvidenceDirectory:      evidenceDir,
		ManifestOutput:         manifestPath,
	}

	_, err = runClosureProtocolV2ForTest(t, context.Background(), req)
	if err == nil {
		t.Fatal("expected rejection for fabricated baseline.tree_oid, got nil")
	}

	// Assert exact machine classification.
	var v2err *V2Error
	if !errors.As(err, &v2err) {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeBaselineTreeNotFound) {
		t.Errorf("expected V2CodeBaselineTreeNotFound, got codes: %v", v2err.Diags.Codes())
	}
}

// TestV2RunnerRejectsNonExistentBaselineCommit asserts that a plan referencing
// a non-existent baseline.commit_oid is rejected by the v2 runner with the
// exact code baseline_commit_not_found.
func TestV2RunnerRejectsNonExistentBaselineCommit(t *testing.T) {
	dir := initRepo(t)
	// Create subject commit.
	subject := makeCommit(t, dir, "subject implementation", map[string]string{
		"src/lib.go": "package lib\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")

	// Build plan with a NON-EXISTENT commit OID.
	planBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-NONEXISTENT-COMMIT-01",
		"deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", subjectTree, v2FixtureCheck{
			ID:               "nonexistent_commit_test",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}

	freeze := makeCommit(t, dir, "freeze: add closure plan", map[string]string{
		"docs/closure-plans/NONEXISTENT-COMMIT.json": string(planBytes),
	})

	evidenceDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:          dir,
		SubjectCommit:           subject,
		FreezeCommit:            freeze,
		PlanPath:                "docs/closure-plans/NONEXISTENT-COMMIT.json",
		EvidenceDirectory:       evidenceDir,
		ManifestOutput:          manifestPath,
	}

	_, err = runClosureProtocolV2ForTest(t, context.Background(), req)
	if err == nil {
		t.Fatal("expected rejection for non-existent baseline.commit_oid, got nil")
	}

	// Assert exact machine classification.
	var v2err *V2Error
	if !errors.As(err, &v2err) {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeBaselineCommitNotFound) {
		t.Errorf("expected V2CodeBaselineCommitNotFound, got codes: %v", v2err.Diags.Codes())
	}
}

// TestV2RunnerRejectsTreeMismatch asserts that a plan where
// baseline.commit_oid^{tree} != baseline.tree_oid is rejected with the
// exact code baseline_tree_mismatch.
func TestV2RunnerRejectsTreeMismatch(t *testing.T) {
	dir := initRepo(t)

	// Create two commits with different trees.
	commit1 := makeCommit(t, dir, "commit 1", map[string]string{"a.txt": "a"})
	tree1 := mustRunGit(t, dir, "rev-parse", commit1+"^{tree}")

	commit2 := makeCommit(t, dir, "commit 2", map[string]string{"b.txt": "b"})

	// Use commit2's tree but declare commit1's tree in the plan - mismatch.
	// The plan says baseline.commit_oid=commit2 but baseline.tree_oid=tree1.
	// Since commit2^{tree} != tree1, validation must fail.
	planBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-TREE-MISMATCH-01",
		commit2, tree1, v2FixtureCheck{
			ID:               "tree_mismatch_test",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}

	freeze := makeCommit(t, dir, "freeze: add closure plan", map[string]string{
		"docs/closure-plans/TREE-MISMATCH.json": string(planBytes),
	})

	evidenceDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:   1,
		RepositoryRoot:         dir,
		SubjectCommit:          commit2,
		FreezeCommit:          freeze,
		PlanPath:               "docs/closure-plans/TREE-MISMATCH.json",
		EvidenceDirectory:      evidenceDir,
		ManifestOutput:         manifestPath,
	}

	_, err = runClosureProtocolV2ForTest(t, context.Background(), req)
	if err == nil {
		t.Fatal("expected rejection for tree mismatch, got nil")
	}

	// Assert exact machine classification.
	var v2err *V2Error
	if !errors.As(err, &v2err) {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeBaselineTreeMismatch) {
		t.Errorf("expected V2CodeBaselineTreeMismatch, got codes: %v", v2err.Diags.Codes())
	}
}

// TestV2RunnerAcceptsValidBaselineCommitTreeBinding asserts that a plan with
// a valid baseline.commit_oid and baseline.tree_oid (where
// baseline.commit_oid^{tree} == baseline.tree_oid) passes validation and
// reaches execution. This is the mandatory positive regression for the ACT.
func TestV2RunnerAcceptsValidBaselineCommitTreeBinding(t *testing.T) {
	dir := initRepo(t)
	// Create baseline commit with implementation.
	baseline := makeCommit(t, dir, "baseline implementation", map[string]string{
		"src/lib.go": "package lib\n",
	})
	baselineTree := mustRunGit(t, dir, "rev-parse", baseline+"^{tree}")

	// Create subject commit.
	subject := makeCommit(t, dir, "subject implementation", map[string]string{
		"src/lib.go": "package lib\n// change\n",
	})

	// Build plan with valid baseline.commit_oid and baseline.tree_oid.
	planBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-VALID-BASELINE-01",
		baseline, baselineTree, v2FixtureCheck{
			ID:               "valid_baseline_test",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}

	freeze := makeCommit(t, dir, "freeze: add closure plan", map[string]string{
		"docs/closure-plans/VALID-BASELINE.json": string(planBytes),
	})

	evidenceDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:   1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:          freeze,
		PlanPath:               "docs/closure-plans/VALID-BASELINE.json",
		EvidenceDirectory:      evidenceDir,
		ManifestOutput:         manifestPath,
	}

	manifest, err := runClosureProtocolV2ForTest(t, context.Background(), req)
	if err != nil {
		t.Fatalf("expected valid baseline to pass, got error: %v", err)
	}

	// Verify the manifest was constructed (proves execution was reached).
	if manifest.SubjectCommit != subject {
		t.Errorf("subject commit mismatch: got %s want %s", manifest.SubjectCommit, subject)
	}
	if manifest.FreezeCommit != freeze {
		t.Errorf("freeze commit mismatch: got %s want %s", manifest.FreezeCommit, freeze)
	}
}
