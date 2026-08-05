// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_uncorrupted_test.go adds the genuine
// topology and frozen-byte fixtures required by Phases 10 and
// 11 of ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-
// AUTHORITY01-CORRECTION01. Unlike the previous hermetic tests,
// these fixtures construct S and F from genuinely distinct root
// histories (no shared merge-base) and a D descendant that
// mutates the plan on disk to prove frozen bytes survive the
// mutation.
//
// Splitting this from closure_protocol_v2_hermetic_test.go
// keeps the file under the LLM-friendly 400-line threshold.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestV2GenuineUnrelatedTopology constructs S from the initial
// branch and F from a fresh orphan branch in the SAME
// repository so they share the object database but have no
// shared merge-base. The runner MUST report exactly
// subject_freeze_unrelated.
func TestV2GenuineUnrelatedTopology(t *testing.T) {
	dir := initRepo(t)
	// Create S on the initial branch.
	// main already exists from initRepo
	s := makeCommit(t, dir, "subject implementation", map[string]string{
		"src/lib.go": "package lib\n",
	})
	// Switch to an orphan branch with no shared history.
	mustRunGit(t, dir, "checkout", "--orphan", "unrelated")
	// Clear the working tree so the orphan commit has no
	// overlap with the original branch's tree.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}
	f := makeCommit(t, dir, "unrelated freeze", map[string]string{
		"unrelated.txt": "unrelated",
	})
	evidenceDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          s,
		FreezeCommit:           f,
		PlanPath:               "docs/closure-plans/UNRELATED-PLAN.json",
		EvidenceDirectory:      evidenceDir,
		ManifestOutput:         manifestPath,
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("expected rejection for genuinely unrelated commits")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	codes := v2err.Diags.Codes()
	if !v2err.Diags.HasCode(V2CodeSubjectFreezeUnrelated) {
		t.Fatalf("expected subject_freeze_unrelated, got %v", codes)
	}
}

// TestV2FrozenBytesSurviveDiskMutation constructs S < F < D
// where D mutates the plan at P, then asserts that the
// loader still binds the manifest to F:P bytes. The test
// validates the loader contract directly (without running
// the full pipeline) so plan-validation noise does not
// obscure the core proof.
func TestV2FrozenBytesSurviveDiskMutation(t *testing.T) {
	dir := initRepo(t)
	frozenPath := "docs/closure-plans/FROZEN-BYTES.json"
	frozenBytes := []byte(`{"contract_version": 1, "act_id": "ACT-FROZEN-BYTES-01"}`)
	f := makeCommit(t, dir, "freeze: add plan", map[string]string{
		frozenPath: string(frozenBytes),
	})
	mutated := []byte(strings.Replace(string(frozenBytes),
		"ACT-FROZEN-BYTES-01", "ACT-MUTATED-BYTES", 1))
	makeCommit(t, dir, "descendant: mutate plan", map[string]string{
		frozenPath: string(mutated),
	})
	loader := NewGitV2FrozenPlanLoader(nil)
	got, err := loader.LoadFrozenPlan(context.Background(), dir, f, frozenPath)
	if err != nil {
		t.Fatalf("LoadFrozenPlan: %v", err)
	}
	if got.SHA256 != SHA256Hex(frozenBytes) {
		t.Fatalf("frozen SHA256 mismatch: got=%s want=%s",
			got.SHA256, SHA256Hex(frozenBytes))
	}
	if got.SHA256 == SHA256Hex(mutated) {
		t.Fatalf("loader returned mutated bytes")
	}
}

// TestV2OptionalWorkingMismatchDetected verifies the working
// plan mismatch path directly. The bytes differ in length
// and SHA-256 so any single comparison fails.
func TestV2OptionalWorkingMismatchDetected(t *testing.T) {
	frozen := V2FrozenPlanBytes{
		Path:      "x",
		Bytes:     []byte("FROZEN-A"),
		SHA256:    SHA256Hex([]byte("FROZEN-A")),
		ByteCount: len("FROZEN-A"),
	}
	mismatchErr := CompareToWorkingPlan(frozen, []byte("WORKING-B"))
	if mismatchErr == nil {
		t.Fatalf("expected working_plan_mismatch")
	}
	v2err, ok := mismatchErr.(*V2Error)
	if !ok || !v2err.Diags.HasCode(V2CodeWorkingPlanMismatch) {
		t.Fatalf("expected working_plan_mismatch, got %v", mismatchErr)
	}
	if passErr := CompareToWorkingPlan(frozen, []byte("FROZEN-A")); passErr != nil {
		t.Fatalf("matching bytes should pass: %v", passErr)
	}
}

// TestV2RunnerRejectsProtocolVersion1 ensures the v2 entry
// point refuses protocol 1 even when all other inputs are
// valid. Phase 1.
func TestV2RunnerRejectsProtocolVersion1(t *testing.T) {
	dir := initRepo(t)
	// main already exists from initRepo
	s := makeCommit(t, dir, "subject", map[string]string{
		"src/lib.go": "package lib\n",
	})
	f := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/REJECT.json": `{"contract_version": 1, "act_id": "X"}`,
	})
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV1,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          s,
		FreezeCommit:           f,
		PlanPath:               "docs/closure-plans/REJECT.json",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("expected rejection for protocol 1")
	}
	v2err, ok := err.(*V2Error)
	if !ok || !v2err.Diags.HasCode(V2CodeUnsupportedClosureProtocolVersion) {
		t.Fatalf("expected unsupported_closure_protocol_version, got %v", err)
	}
}
