// SPDX-License-Identifier: Apache-2.0

package closure

// v2_mac_canary_test.go exercises the full v2 runner against a
// meaningful S < F < D repository to prove the Mac canary
// readiness required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01.
//
// The repository is constructed so that:
//
//	S:
//	  - subject-only file exists
//	  - plan file absent
//
//	F (= child of S):
//	  - valid frozen plan added at docs/closure-plans/PATH.json
//	  - freeze-only file present in F tree
//	  - plan baseline binds S and S^{tree}
//
//	D (= child of F):
//	  - plan mutated at docs/closure-plans/PATH.json
//	  - descendant-only file added
//
//	HEAD = D
//	caller worktree clean
//
// The run check, built from the contract-valid fixture builder
// (BuildV2ValidPlanFixtureWithCheck), proves:
//
//	test -f subject-only.txt   (subject tree contains S's file)
//	test ! -e freeze-only.txt  (subject tree does NOT contain F-only)
//	test ! -e descendant-only.txt (subject tree does NOT contain D-only)
//
// The tests assert every manifest binding (subject, freeze,
// execution tree, plan blob, plan sha256, plan path,
// protocol/contract versions) and the no-drift invariants
// (caller HEAD/tree/status, linked-worktree registrations) that
// the runner must guarantee on a successful Mac canary run.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildMacCanaryThreeCommitHistory constructs the meaningful
// S < F < D repository used by every Mac canary test. The
// helper returns the temp directory so callers operate on the
// SAME repository, not a separate one. Returned values:
//
//	dir          - the temp directory holding the S < F < D repo
//	subject      - SHA of S
//	subjectTree  - SHA of S^{tree}
//	freeze       - SHA of F (child of S)
//	d            - SHA of D (child of F)
//	frozenBytes  - exact bytes of the F:PATH plan
//	mutatedBytes - exact bytes of the D:PATH plan
//	planPath     - relative plan path "docs/closure-plans/PATH.json"
func buildMacCanaryThreeCommitHistory(t *testing.T) (dir, subject, subjectTree, freeze, d string, frozenBytes, mutatedBytes []byte, planPath string) {
	t.Helper()
	dir = initRepo(t)
	planPath = "docs/closure-plans/PATH.json"
	subject = makeCommit(t, dir, "subject: implement", map[string]string{
		"subject-only.txt": "subject implementation\n",
	})
	subjectTree = mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	frozenBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-MAC-CANARY-V2-01",
		subject, subjectTree, v2FixtureCheck{
			ID:               "mac_canary_proof",
			Mode:             "run",
			Argv:             []string{"sh", "-c", "test -f subject-only.txt && test ! -e freeze-only.txt && test ! -e descendant-only.txt"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}
	freeze = makeCommit(t, dir, "freeze: add closure plan", map[string]string{
		planPath:          string(frozenBytes),
		"freeze-only.txt": "freeze-only marker\n",
	})
	mutatedBytes = []byte(`{"contract_version": 1, "act_id": "ACT-MAC-CANARY-MUTATED"}`)
	d = makeCommit(t, dir, "descendant: mutate plan", map[string]string{
		planPath:              string(mutatedBytes),
		"descendant-only.txt": "descendant-only marker\n",
	})
	return dir, subject, subjectTree, freeze, d, frozenBytes, mutatedBytes, planPath
}

// TestV2MacCanaryFullRunnerDescendantProof is the Mac canary
// Phase 2 proof. It invokes the production runner against a
// meaningful S < F < D repository with HEAD = D and asserts
// every manifest binding plus every no-drift invariant.
func TestV2MacCanaryFullRunnerDescendantProof(t *testing.T) {
	dir, subject, subjectTree, freeze, d, frozenBytes, _, planPath :=
		buildMacCanaryThreeCommitHistory(t)
	// HEAD must equal D.
	if got := mustRunGit(t, dir, "rev-parse", "HEAD"); got != d {
		t.Fatalf("HEAD must be D: got=%s want=%s", got, d)
	}
	evidenceDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	// Snapshot caller + worktree registrations BEFORE the run.
	headBefore := mustRunGit(t, dir, "rev-parse", "HEAD")
	headTreeBefore := mustRunGit(t, dir, "rev-parse", "HEAD^{tree}")
	statusBefore := mustRunGit(t, dir, "status", "--porcelain=v2", "--untracked-files=all")
	worktreesBefore := mustRunGit(t, dir, "worktree", "list", "--porcelain")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               planPath,
		EvidenceDirectory:      evidenceDir,
		ManifestOutput:         manifestPath,
	}
	manifest, err := runClosureProtocolV2ForTest(t, context.Background(), req)
	if err != nil {
		t.Fatalf("RunClosureProtocolV2WithBinary: %v", err)
	}
	// Snapshot caller + worktree registrations AFTER the run.
	headAfter := mustRunGit(t, dir, "rev-parse", "HEAD")
	headTreeAfter := mustRunGit(t, dir, "rev-parse", "HEAD^{tree}")
	statusAfter := mustRunGit(t, dir, "status", "--porcelain=v2", "--untracked-files=all")
	worktreesAfter := mustRunGit(t, dir, "worktree", "list", "--porcelain")

	// Manifest bindings — every field must agree.
	if manifest.SubjectCommit != subject {
		t.Fatalf("subject_commit: got=%s want=%s", manifest.SubjectCommit, subject)
	}
	if manifest.SubjectTree != subjectTree {
		t.Fatalf("subject_tree: got=%s want=%s", manifest.SubjectTree, subjectTree)
	}
	if manifest.FreezeCommit != freeze {
		t.Fatalf("freeze_commit: got=%s want=%s", manifest.FreezeCommit, freeze)
	}
	if manifest.ExecutionTree != subjectTree {
		t.Fatalf("execution_tree must equal subject_tree: got=%s want=%s",
			manifest.ExecutionTree, subjectTree)
	}
	if manifest.PlanPath != planPath {
		t.Fatalf("plan_path: got=%s want=%s", manifest.PlanPath, planPath)
	}
	wantPlanBlob := mustRunGit(t, dir, "rev-parse", freeze+":"+planPath)
	if manifest.PlanBlob != wantPlanBlob {
		t.Fatalf("plan_blob: got=%s want=%s", manifest.PlanBlob, wantPlanBlob)
	}
	wantPlanSHA := sha256Hex(frozenBytes)
	if manifest.PlanSHA256 != wantPlanSHA {
		t.Fatalf("plan_sha256: got=%s want=%s", manifest.PlanSHA256, wantPlanSHA)
	}
	if manifest.ClosureProtocolVersion != ClosureProtocolV2 {
		t.Fatalf("closure_protocol_version: got=%s want=%s",
			manifest.ClosureProtocolVersion, ClosureProtocolV2)
	}
	if manifest.PlanContractVersion != 1 {
		t.Fatalf("plan_contract_version: got=%d want=1", manifest.PlanContractVersion)
	}
	if manifest.CallerHead != d {
		t.Fatalf("caller_head must equal pre-run D: got=%s want=%s",
			manifest.CallerHead, d)
	}

	// No-drift invariants: caller repository must remain
	// unchanged in HEAD, HEAD tree, working-tree status, and
	// linked-worktree registrations.
	if headBefore != headAfter {
		t.Fatalf("caller HEAD drifted: before=%s after=%s", headBefore, headAfter)
	}
	if headTreeBefore != headTreeAfter {
		t.Fatalf("caller HEAD tree drifted: before=%s after=%s",
			headTreeBefore, headTreeAfter)
	}
	if statusBefore != statusAfter {
		t.Fatalf("caller worktree status drifted:\nbefore=%q\nafter=%q",
			statusBefore, statusAfter)
	}
	if worktreesBefore != worktreesAfter {
		t.Fatalf("linked-worktree registrations leaked:\nbefore=%q\nafter=%q",
			worktreesBefore, worktreesAfter)
	}

	// Manifest must be on disk and valid JSON.
	onDisk, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("manifest file missing: %v", err)
	}
	var roundTripped V2Manifest
	if err := json.Unmarshal(onDisk, &roundTripped); err != nil {
		t.Fatalf("manifest JSON invalid: %v", err)
	}
	if roundTripped.SubjectCommit != subject {
		t.Fatalf("round-tripped subject_commit mismatch")
	}

	// The check must have passed against S^{tree}.
	if len(manifest.CheckResults) != 1 {
		t.Fatalf("expected 1 check result, got %d", len(manifest.CheckResults))
	}
	if manifest.CheckResults[0].Outcome != CheckStatusPass {
		t.Fatalf("mac_canary_proof must pass on S^{tree}: %+v",
			manifest.CheckResults[0])
	}
}

// TestV2MacCanaryWorkingAssertionRejectsDescendantPlan is the
// Mac canary Phase 2 working-plan assertion. With caller HEAD = D
// and OptionalWorkingPlanAssertion = a working copy of D:P, the
// runner must reject with V2CodeWorkingPlanMismatch BEFORE any
// executor call and MUST NOT write a manifest.
func TestV2MacCanaryWorkingAssertionRejectsDescendantPlan(t *testing.T) {
	dir, subject, _, freeze, d, _, mutatedBytes, planPath :=
		buildMacCanaryThreeCommitHistory(t)
	if got := mustRunGit(t, dir, "rev-parse", "HEAD"); got != d {
		t.Fatalf("HEAD must be D: got=%s want=%s", got, d)
	}
	// Place the descendant bytes at a detached working-plan
	// location. They MUST differ from F:P for the assertion
	// to fail.
	working := filepath.Join(t.TempDir(), "DESCENDANT-PLAN.json")
	if err := os.WriteFile(working, mutatedBytes, 0o644); err != nil {
		t.Fatalf("write working: %v", err)
	}
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	evidenceDir := t.TempDir()
	req := V2Request{
		ClosureProtocolVersion:       ClosureProtocolV2,
		PlanContractVersion:          1,
		RepositoryRoot:               dir,
		SubjectCommit:                subject,
		FreezeCommit:                 freeze,
		PlanPath:                     planPath,
		EvidenceDirectory:            evidenceDir,
		ManifestOutput:               manifestPath,
		OptionalWorkingPlanAssertion: working,
	}
	_, err := runClosureProtocolV2ForTest(t, context.Background(), req)
	if err == nil {
		t.Fatalf("working_plan_assertion = D:P must reject (HEAD=D)")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeWorkingPlanMismatch) {
		t.Fatalf("expected working_plan_mismatch, got %v", v2err.Diags.Codes())
	}
	if _, statErr := os.Stat(manifestPath); statErr == nil {
		t.Fatalf("manifest must NOT be written on working plan mismatch")
	}
	// Caller HEAD must still be D after the rejection.
	if got := mustRunGit(t, dir, "rev-parse", "HEAD"); got != d {
		t.Fatalf("caller HEAD drifted on rejection: got=%s want=%s", got, d)
	}
}

// TestV2MacCanaryNoCallerStateDrift asserts the
// lifecycle-invariant invariants explicitly on a successful
// S < F < D run, complementing the assertions in
// TestV2MacCanaryFullRunnerDescendantProof.
func TestV2MacCanaryNoCallerStateDrift(t *testing.T) {
	dir, subject, _, freeze, d, _, _, planPath :=
		buildMacCanaryThreeCommitHistory(t)
	if got := mustRunGit(t, dir, "rev-parse", "HEAD"); got != d {
		t.Fatalf("HEAD must be D: got=%s want=%s", got, d)
	}
	// Capture state BEFORE.
	before := captureMacCanaryCallerState(t, dir)
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               planPath,
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	}
	if _, err := runClosureProtocolV2ForTest(t, context.Background(), req); err != nil {
		t.Fatalf("runner failed: %v", err)
	}
	after := captureMacCanaryCallerState(t, dir)
	if !bytes.Equal(before.head, after.head) {
		t.Fatalf("caller HEAD drifted: before=%s after=%s",
			string(before.head), string(after.head))
	}
	if !bytes.Equal(before.headTree, after.headTree) {
		t.Fatalf("caller HEAD tree drifted: before=%s after=%s",
			string(before.headTree), string(after.headTree))
	}
	if !bytes.Equal(before.status, after.status) {
		t.Fatalf("caller status drifted:\nbefore=%q\nafter=%q",
			string(before.status), string(after.status))
	}
	if !bytes.Equal(before.worktrees, after.worktrees) {
		t.Fatalf("linked-worktree registrations leaked:\nbefore=%q\nafter=%q",
			string(before.worktrees), string(after.worktrees))
	}
}

// TestV2MacCanaryNoWorktreeLeak asserts the linked-worktree
// registration count is identical before and after a successful
// S < F < D run, even when the runner's own internal worktree
// was used during execution.
func TestV2MacCanaryNoWorktreeLeak(t *testing.T) {
	dir, subject, _, freeze, _, _, _, planPath :=
		buildMacCanaryThreeCommitHistory(t)
	beforeCount := countLinkedWorktreeRegistrations(t, dir)
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               planPath,
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	}
	if _, err := runClosureProtocolV2ForTest(t, context.Background(), req); err != nil {
		t.Fatalf("runner failed: %v", err)
	}
	afterCount := countLinkedWorktreeRegistrations(t, dir)
	if beforeCount != afterCount {
		t.Fatalf("linked-worktree registration count drifted: before=%d after=%d",
			beforeCount, afterCount)
	}
}

// macCanaryCallerState captures the caller repository state for
// the Mac canary no-drift assertions.
type macCanaryCallerState struct {
	head      []byte
	headTree  []byte
	status    []byte
	worktrees []byte
}

func captureMacCanaryCallerState(t *testing.T, dir string) macCanaryCallerState {
	t.Helper()
	return macCanaryCallerState{
		head:      []byte(mustRunGit(t, dir, "rev-parse", "HEAD")),
		headTree:  []byte(mustRunGit(t, dir, "rev-parse", "HEAD^{tree}")),
		status:    []byte(mustRunGit(t, dir, "status", "--porcelain=v2", "--untracked-files=all")),
		worktrees: []byte(mustRunGit(t, dir, "worktree", "list", "--porcelain")),
	}
}

// countLinkedWorktreeRegistrations returns the number of
// `worktree <path>` entries in `git worktree list --porcelain`.
// The first entry is always the main worktree; subsequent entries
// are linked worktrees. We count only the linked entries so the
// assertion is sensitive to leaks while ignoring the original.
func countLinkedWorktreeRegistrations(t *testing.T, dir string) int {
	t.Helper()
	listing := mustRunGit(t, dir, "worktree", "list", "--porcelain")
	lines := strings.Split(listing, "\n")
	entries := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			entries++
		}
	}
	// Subtract the main worktree entry.
	if entries > 0 {
		return entries - 1
	}
	return 0
}

// sha256Hex returns the lowercase hex SHA-256 of b. The runner
// emits hex.EncodeToString(sha256.Sum256(b)[:]) so we use the
// same encoding here.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
