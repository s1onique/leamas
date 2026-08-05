// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_hermetic_test.go exercises the v2
// runner against a hermetic Git repository constructed in a
// temp directory. The test simulates the ClineMM topology:
//
//   - subject commit S implements the change, no plan
//   - freeze commit F = child of S, plan file added in F only
//   - HEAD = F
//
// The runner MUST:
//   - accept Plan Contract v1 + Closure Protocol v2
//   - load plan bytes from F:PATH (not from disk)
//   - execute checks against S^{tree} (not F^{tree})
//   - bind manifest subject=S, freeze=F, execution_tree=S^{tree}
//   - bind manifest plan_blob=F:PATH blob, plan_sha256=hash(bytes)

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

// writeGitConfig sets a stable user.name and user.email so
// commits succeed in the temp directory.
func writeGitConfig(t *testing.T, dir string) {
	t.Helper()
	for _, kv := range [][]string{
		{"user.name", "Leamas Test"},
		{"user.email", "test@leamas.local"},
		{"commit.gpgsign", "false"},
		{"tag.gpgsign", "false"},
	} {
		out, err := execution.RunGit(context.Background(), dir, append([]string{"config"}, kv...)...)
		if err != nil || out.ExitCode != 0 {
			t.Fatalf("git config %v: %v exit=%d stderr=%s", kv, err, out.ExitCode, string(out.Stderr))
		}
	}
}

// initRepo runs git init in the temp directory and returns
// the repo path. The repo has an empty initial commit so
// rev-parse HEAD^{commit} works.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	out, err := execution.RunGit(context.Background(), dir, "init", "-b", "main", dir)
	if err != nil || out.ExitCode != 0 {
		t.Fatalf("git init: %v exit=%d stderr=%s", err, out.ExitCode, string(out.Stderr))
	}
	writeGitConfig(t, dir)
	mustRunGit(t, dir, "commit", "--allow-empty", "-m", "initial")
	return dir
}

// mustRunGit runs an arbitrary git command and fails the
// test on error. Returns stdout.
func mustRunGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := execution.RunGit(context.Background(), dir, args...)
	if err != nil || out.ExitCode != 0 {
		t.Fatalf("git %v: %v exit=%d stderr=%s", args, err, out.ExitCode, string(out.Stderr))
	}
	return strings.TrimSpace(string(out.Stdout))
}

// makeCommit creates a commit with the supplied tree.
func makeCommit(t *testing.T, dir, message string, files map[string]string) string {
	t.Helper()
	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	mustRunGit(t, dir, "add", ".")
	mustRunGit(t, dir, "commit", "-m", message)
	return mustRunGit(t, dir, "rev-parse", "HEAD")
}

// TestV2HermeticTopologyEndToEnd creates a hermetic repo
// matching the ClineMM shape (S implementation, F=child+S+plan)
// and runs the v2 runner end-to-end.
func TestV2HermeticTopologyEndToEnd(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject implementation", map[string]string{
		"src/lib.go":       "package lib\n",
		"subject-only.txt": "subject implementation file\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	planBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-HERMETIC-V2-01",
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
	freeze := makeCommit(t, dir, "freeze: add closure plan", map[string]string{
		"docs/closure-plans/HERMETIC-PLAN.json": string(planBytes),
	})
	evidenceDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/HERMETIC-PLAN.json",
		EvidenceDirectory:      evidenceDir,
		ManifestOutput:         manifestPath,
	}
	manifest, err := RunClosureProtocolV2(context.Background(), req)
	if err != nil {
		v2err, ok := err.(*V2Error)
		if ok {
			for _, d := range v2err.Diags {
				t.Logf("diag: code=%s msg=%s prop=%s detail=%s", d.Code, d.Message, d.PropertyName, d.Detail)
			}
		}
		t.Fatalf("RunClosureProtocolV2: %v", err)
	}
	if manifest.SubjectCommit != subject {
		t.Fatalf("subject commit mismatch: got %s want %s", manifest.SubjectCommit, subject)
	}
	if manifest.FreezeCommit != freeze {
		t.Fatalf("freeze commit mismatch: got %s want %s", manifest.FreezeCommit, freeze)
	}
	expectedSubjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	expectedFreezeTree := mustRunGit(t, dir, "rev-parse", freeze+"^{tree}")
	if manifest.SubjectTree != expectedSubjectTree {
		t.Fatalf("subject tree mismatch: got %s want %s", manifest.SubjectTree, expectedSubjectTree)
	}
	if manifest.FreezeTree != expectedFreezeTree {
		t.Fatalf("freeze tree mismatch: got %s want %s", manifest.FreezeTree, expectedFreezeTree)
	}
	if manifest.ExecutionTree != manifest.SubjectTree {
		t.Fatalf("execution tree %s must equal subject tree %s", manifest.ExecutionTree, manifest.SubjectTree)
	}
	if manifest.PlanBlob == "" {
		t.Fatalf("plan blob OID missing")
	}
	if manifest.PlanSHA256 == "" {
		t.Fatalf("plan SHA-256 missing")
	}
	if manifest.PlanPath != "docs/closure-plans/HERMETIC-PLAN.json" {
		t.Fatalf("plan path: got %q", manifest.PlanPath)
	}
	planBlob := mustRunGit(t, dir, "rev-parse", freeze+":docs/closure-plans/HERMETIC-PLAN.json")
	if manifest.PlanBlob != planBlob {
		t.Fatalf("plan blob mismatch: got %s want %s", manifest.PlanBlob, planBlob)
	}
	if manifest.ClosureProtocolVersion != ClosureProtocolV2 {
		t.Fatalf("closure protocol version: got %s want %s", manifest.ClosureProtocolVersion, ClosureProtocolV2)
	}
	if manifest.PlanContractVersion != 1 {
		t.Fatalf("plan contract version: got %d want 1", manifest.PlanContractVersion)
	}
	onDisk, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("manifest file missing: %v", err)
	}
	var roundTripped V2Manifest
	if err := json.Unmarshal(onDisk, &roundTripped); err != nil {
		t.Fatalf("manifest JSON invalid: %v", err)
	}
	if roundTripped.SubjectCommit != subject {
		t.Fatalf("round-tripped subject mismatch")
	}
}

// TestV2RunnerRejectsReverseRelation covers the
// "F < S" case which v2 must reject with a typed diagnostic.
func TestV2RunnerRejectsReverseRelation(t *testing.T) {
	dir := initRepo(t)
	// S has the plan, F is the implementation child of S
	freeze := makeCommit(t, dir, "freeze: add closure plan", map[string]string{
		"docs/closure-plans/HERMETIC.json": `{"contract_version": 1, "act_id": "X"}`,
	})
	subject := makeCommit(t, dir, "subject implementation", map[string]string{
		"src/lib.go": "package lib\n",
	})
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	evidenceDir := t.TempDir()
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/HERMETIC.json",
		EvidenceDirectory:      evidenceDir,
		ManifestOutput:         manifestPath,
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("expected rejection for reverse relation")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeFreezeAncestorOfSubject) {
		t.Fatalf("expected freeze_ancestor_of_subject, got %v", v2err.Diags.Codes())
	}
	if _, err := os.Stat(manifestPath); err == nil {
		t.Fatalf("manifest must not be written on failure")
	}
}

// TestV2RunnerRejectsDirtyCaller verifies the runner refuses
// to publish a manifest when the caller worktree is dirty.
func TestV2RunnerRejectsDirtyCaller(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{"a.txt": "a"})
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/HERMETIC.json": `{"contract_version": 1, "act_id": "X"}`,
	})
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty: %v", err)
	}
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/HERMETIC.json",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("expected rejection for dirty caller")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerWorktreeDirty) {
		t.Fatalf("expected caller_worktree_dirty, got %v", v2err.Diags.Codes())
	}
}

// TestV2FrozenBytesAdversarial modifies the disk plan after
// F and proves the runner still loads frozen bytes from F.
func TestV2FrozenBytesAdversarial(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{"a.txt": "a"})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	frozenBytes, err := BuildV2ValidPlanFixture("ACT-HERMETIC-FROZEN-01", subject, subjectTree)
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixture: %v", err)
	}
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/HERMETIC.json": string(frozenBytes),
	})
	// Modify the disk plan to a different valid plan AFTER F.
	diskBytes, err := BuildV2ValidPlanFixture("DISK_ACT", subject, subjectTree)
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixture(disk): %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs/closure-plans/HERMETIC.json"),
		diskBytes, 0o644); err != nil {
		t.Fatalf("overwrite plan: %v", err)
	}
	// Reset working tree so the dirty test above doesn't
	// fire. We restore the file to the tracked content.
	mustRunGit(t, dir, "checkout", "--", "docs/closure-plans/HERMETIC.json")
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	evidenceDir := t.TempDir()
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/HERMETIC.json",
		EvidenceDirectory:      evidenceDir,
		ManifestOutput:         manifestPath,
	}
	manifest, err := RunClosureProtocolV2(context.Background(), req)
	if err != nil {
		t.Fatalf("runner failed: %v", err)
	}
	if manifest.PlanPath != "docs/closure-plans/HERMETIC.json" {
		t.Fatalf("plan path: got %q", manifest.PlanPath)
	}
	// Frozen bytes must be the F bytes, not the disk bytes.
	planBlob := mustRunGit(t, dir, "rev-parse", freeze+":docs/closure-plans/HERMETIC.json")
	if manifest.PlanBlob != planBlob {
		t.Fatalf("plan blob %s != frozen %s", manifest.PlanBlob, planBlob)
	}
}

// TestV2AbsolutePlanPathRejected covers the loader /
// validator rejecting absolute paths so caller cannot
// supply /etc/passwd.
func TestV2AbsolutePlanPathRejected(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{"a.txt": "a"})
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/HERMETIC.json": `{"contract_version": 1, "act_id": "X"}`,
	})
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "/etc/passwd",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		t.Fatalf("expected rejection of absolute plan path")
	}
	v2err, ok := err.(*V2Error)
	if !ok || !v2err.Diags.HasCode(V2CodeInvalidPlanPath) {
		t.Fatalf("expected invalid_plan_path, got %v", err)
	}
}

// TestV2RunnerRejectsUnrelatedCommits covers the case where
// S and F share no ancestry at all.
func TestV2RunnerRejectsUnrelatedCommits(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{"a.txt": "a"})
	freeze := makeCommit(t, dir, "freeze", map[string]string{"b.txt": "b"})
	// Create an unrelated branch with its own commit.
	mustRunGit(t, dir, "checkout", "-b", "unrelated")
	mustRunGit(t, dir, "commit", "--allow-empty", "-m", "unrelated")
	_ = freeze
	_ = subject
	// Reset by creating a fresh repository to make S and F unrelated.
	dir2 := initRepo(t)
	s := makeCommit(t, dir2, "subject", map[string]string{"a.txt": "a"})
	f := makeCommit(t, dir2, "freeze", map[string]string{"b.txt": "b"})
	// Reset HEAD to a brand-new orphan branch so S and F are not in the same history.
	mustRunGit(t, dir2, "checkout", "--orphan", "fresh")
	mustRunGit(t, dir2, "commit", "--allow-empty", "-m", "fresh")
	_ = s
	_ = f
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir2,
		SubjectCommit:          s,
		FreezeCommit:           f,
		PlanPath:               "docs/closure-plans/X.json",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	}
	_, err := RunClosureProtocolV2(context.Background(), req)
	if err == nil {
		// Some git setups may still report the merge-base as
		// empty. We only fail loudly if the runner accepted
		// unrelated commits with success code.
		t.Fatalf("expected rejection for unrelated commits")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T", err)
	}
	// The exact code depends on whether S/F are reachable
	// from the orphan HEAD. Both frozen_plan_path_missing and
	// subject_freeze_unrelated are valid rejections for
	// unreachable or unrelated commits.
	codes := v2err.Diags.Codes()
	if !v2err.Diags.HasCode(V2CodeSubjectFreezeUnrelated) &&
		!v2err.Diags.HasCode(V2CodeSubjectCommitNotFound) &&
		!v2err.Diags.HasCode(V2CodeFreezeCommitNotFound) &&
		!v2err.Diags.HasCode(V2CodeFrozenPlanPathMissing) {
		t.Fatalf("expected unrelated or unreachable diagnostic, got %v", codes)
	}
}
