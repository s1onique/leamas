// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestRunClosureV2PreparedRecoveryRejectsBranchSwitchBeforeMutation proves
// that a PREPARED recovery invocation whose currently attached branch does
// not match the branch originally authorized by the run that produced E is
// rejected before any object, ref, worktree, or tag mutation. The branch
// that originally approved E remains at S; no other branch advances; the
// tag is absent.
func TestRunClosureV2PreparedRecoveryRejectsBranchSwitchBeforeMutation(t *testing.T) {
	fixture := prepareV2Repository(t)
	// Create a second branch that also points at S. After PREPARED state is
	// created on main, switching to release must not let recovery publish to
	// release while keeping the original E.
	v2Git(t, fixture.root, "branch", "release", "HEAD")

	failureGit := &v2FailingGit{failCommand: "update-ref"}
	if _, err := runProductionV2Test(fixture,
		productionV2TestDependencies(fixture, failureGit, nil)); err == nil {
		t.Fatal("injected ref-publication failure was accepted")
	}

	// Capture the protected state on main BEFORE the switch so we can prove
	// that no later step writes a manifest or report anywhere.
	mainBefore := v2Git(t, fixture.root, "rev-parse", "refs/heads/main")
	releaseBefore := v2Git(t, fixture.root, "rev-parse", "refs/heads/release")

	// Switch HEAD to release and rerun. Recovery must reject because the
	// recorded publication branch (main) does not match the attached branch
	// (release).
	v2Git(t, fixture.root, "checkout", "release")
	before := captureV2StableState(t, fixture)
	recorder := &v2RecordingGit{delegate: RealGit{}}
	checks := 0
	deps := productionV2TestDependencies(fixture, recorder, &checks)
	_, err := runProductionV2Test(fixture, deps)
	if err == nil || !strings.Contains(err.Error(), "publication branch") {
		t.Fatalf("err = %v, want publication branch rejection", err)
	}
	if checks != 0 {
		t.Fatalf("recovery checks = %d, want 0", checks)
	}
	if len(recorder.objectWrites) != 0 {
		t.Fatalf("object writes preceded branch-rejection: %v", recorder.objectWrites)
	}
	after := captureV2StableState(t, fixture)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("branch-rejection mutated repository:\nbefore=%#v\nafter=%#v", before, after)
	}
	if got := v2Git(t, fixture.root, "rev-parse", "refs/heads/main"); got != mainBefore {
		t.Fatalf("main moved from %s to %s", mainBefore, got)
	}
	if got := v2Git(t, fixture.root, "rev-parse", "refs/heads/release"); got != releaseBefore {
		t.Fatalf("release moved from %s to %s", releaseBefore, got)
	}
	if got := v2Git(t, fixture.root, "rev-parse", "HEAD"); got != releaseBefore {
		t.Fatalf("HEAD moved to %s", got)
	}
	assertV2TagAbsent(t, fixture)
}

// TestRunClosureV2PreparedRecoveryRejectsAfterSwitchedHead proves that even
// when the runner only switches the symbolic-ref (not the ref database)
// between PREPARED and recovery, the recorded publication branch still
// pins the publication target.
func TestRunClosureV2PreparedRecoveryRejectsAfterSwitchedHead(t *testing.T) {
	fixture := prepareV2Repository(t)
	v2Git(t, fixture.root, "branch", "release", "HEAD")

	if _, err := runProductionV2Test(fixture,
		productionV2TestDependencies(fixture, &v2FailingGit{failCommand: "update-ref"}, nil)); err == nil {
		t.Fatal("injected ref-publication failure was accepted")
	}

	mainBefore := v2Git(t, fixture.root, "rev-parse", "refs/heads/main")
	releaseBefore := v2Git(t, fixture.root, "rev-parse", "refs/heads/release")
	before := captureV2StableState(t, fixture)
	v2Git(t, fixture.root, "checkout", "release")

	recorder := &v2RecordingGit{delegate: RealGit{}}
	checks := 0
	deps := productionV2TestDependencies(fixture, recorder, &checks)
	_, err := runProductionV2Test(fixture, deps)
	if err == nil || !strings.Contains(err.Error(), "publication branch") {
		t.Fatalf("err = %v, want publication branch rejection", err)
	}
	if len(recorder.objectWrites) != 0 {
		t.Fatalf("object writes preceded branch-rejection: %v", recorder.objectWrites)
	}
	after := captureV2StableState(t, fixture)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("rejection mutated repository:\nbefore=%#v\nafter=%#v", before, after)
	}
	if got := v2Git(t, fixture.root, "rev-parse", "refs/heads/main"); got != mainBefore {
		t.Fatalf("main moved from %s to %s", mainBefore, got)
	}
	if got := v2Git(t, fixture.root, "rev-parse", "refs/heads/release"); got != releaseBefore {
		t.Fatalf("release moved from %s to %s", releaseBefore, got)
	}
	assertV2TagAbsent(t, fixture)
}

// TestRunClosureV2NewPostRenameEvidenceMutationAbortsBeforeRefPublication
// proves that on the NEW path a concurrent or escaped descendant that
// mutates an indexed file after the staging→final rename but before ref
// publication causes the transaction to abort. Refs and tag remain at S
// or absent; the canonical closure artifacts are NOT written; the
// tampered bytes are visible in the published E directory (proof the
// mutation occurred and was caught by re-read), but the orchestrator
// never published the closure commit or tag.
func TestRunClosureV2NewPostRenameEvidenceMutationAbortsBeforeRefPublication(t *testing.T) {
	fixture := prepareV2Repository(t)
	branchBefore := v2Git(t, fixture.root, "rev-parse", "refs/heads/main")
	tagBefore := rawGitStdout(t, fixture.root, "for-each-ref", "refs/tags/act/")

	checks := 0
	deps := productionV2TestDependencies(fixture, RealGit{}, &checks)
	originalPublish := deps.PublishEvidence
	deps.PublishEvidence = func(stagingPath, finalPath string, ev v2QualifiedEvidence) (v2QualifiedEvidence, error) {
		out, err := originalPublish(stagingPath, finalPath, ev)
		if err != nil {
			return out, err
		}
		// Mutate one indexed evidence file inside the final directory. The
		// orchestrator's re-read must detect the divergence from the
		// captured index hash and abort before publishV2Refs runs.
		victim := filepath.Join(finalPath, "authority-check.stdout")
		if err := os.WriteFile(victim, []byte("tampered-after-rename\n"), 0o600); err != nil {
			return out, err
		}
		return out, nil
	}

	_, err := runProductionV2Test(fixture, deps)
	if err == nil || !strings.Contains(err.Error(), "reverify final evidence") {
		t.Fatalf("err = %v, want post-rename reverification rejection", err)
	}
	if checks != 1 {
		t.Fatalf("NEW checks = %d, want 1", checks)
	}

	// Refs and tag set must be unchanged.
	if got := v2Git(t, fixture.root, "rev-parse", "refs/heads/main"); got != branchBefore {
		t.Fatalf("branch moved from %s to %s", branchBefore, got)
	}
	if tagAfter := rawGitStdout(t, fixture.root, "for-each-ref", "refs/tags/act/"); !bytes.Equal(tagAfter, tagBefore) {
		t.Fatalf("tag set changed: before=%q after=%q", tagBefore, tagAfter)
	}
	assertV2TagAbsent(t, fixture)

	// Canonical closure artifacts must NOT exist in the worktree.
	for _, path := range []string{
		canonicalV2ManifestPath(v2OrchestratorActID),
		canonicalV2ReportPath(v2OrchestratorActID),
	} {
		full := filepath.Join(fixture.root, filepath.FromSlash(path))
		if _, err := os.Lstat(full); !os.IsNotExist(err) {
			t.Fatalf("canonical artifact %s exists after rejection: %v", path, err)
		}
	}

	// Tampered bytes are visible in the published evidence directory
	// (proof the mutation was caught at re-read, not before).
	tampered, err := os.ReadFile(filepath.Join(v2EvidencePath(fixture), "authority-check.stdout"))
	if err != nil {
		t.Fatalf("tampered file missing: %v", err)
	}
	if string(tampered) != "tampered-after-rename\n" {
		t.Fatalf("tampered bytes = %q, want %q", tampered, "tampered-after-rename\n")
	}
}

// TestRunClosureV2PreparedRecoverySurvivesUnreachableObjectPruning proves
// that the deterministic reconstruction is the only authority: once
// prepared C/T objects are unreachable (no tag, no branch), reflog
// expiration + git prune do not weaken recovery. The recovery run
// reconstructs identical C/T, runs zero checks, and exits 0.
func TestRunClosureV2PreparedRecoverySurvivesUnreachableObjectPruning(t *testing.T) {
	fixture := prepareV2Repository(t)
	if _, err := runProductionV2Test(fixture,
		productionV2TestDependencies(fixture, &v2FailingGit{failCommand: "update-ref"}, nil)); err == nil {
		t.Fatal("injected ref-publication failure was accepted")
	}

	// Wipe reflog and aggressively prune unreachable objects. The prepared
	// C/T objects have no tag or branch pointing to them after the
	// interrupted transaction, so this exercises the "evidence is
	// authority; objects are only a cache" contract.
	v2Git(t, fixture.root, "reflog", "expire", "--expire=now", "--all")
	v2Git(t, fixture.root, "reflog", "expire", "--expire-unreachable=now", "--all")
	v2Git(t, fixture.root, "gc", "--prune=now", "--aggressive")

	recorder := &v2RecordingGit{delegate: RealGit{}}
	checks := 0
	deps := productionV2TestDependencies(fixture, recorder, &checks)
	result, err := runProductionV2Test(fixture, deps)
	if err != nil {
		t.Fatalf("recovery after prune: %v", err)
	}
	if checks != 0 {
		t.Fatalf("recovery checks = %d, want 0", checks)
	}
	assertCompleteV2Result(t, fixture, result)
}

// TestWriteV2RuntimeEvidenceRequiresBranch proves the publication-branch
// field is mandatory on the write path. An empty branch must be rejected
// before any file is created.
func TestWriteV2RuntimeEvidenceRequiresBranch(t *testing.T) {
	input := v2FinalizeInput{
		EvidenceDirectory: makeEvidenceStaging(t, nil),
		Plan:              Plan{ActID: objectTransactionActID},
		Runner:            RunnerIdentity{VCSRevision: strings.Repeat("a", 40), BinarySHA256: strings.Repeat("b", 64)},
	}
	if err := writeV2RuntimeEvidence(input); err == nil ||
		!strings.Contains(err.Error(), "publication branch") {
		t.Fatalf("err = %v, want missing-publication-branch rejection", err)
	}
}

// TestValidateV2EvidenceAuthorityRejectsBranchMismatch is a focused unit
// test that proves validateV2EvidenceAuthority pins the publication target
// without involving the orchestrator. Each variant must reject with a
// message identifying the branch as the cause.
func TestValidateV2EvidenceAuthorityRejectsBranchMismatch(t *testing.T) {
	plan := minimalValidPlan()
	plan.ActID = objectTransactionActID
	runtime := v2RuntimeEvidence{
		ContractVersion:   1,
		ActID:             plan.ActID,
		PublicationBranch: "main",
		Runner:            RunnerIdentity{VCSRevision: strings.Repeat("a", 40), BinarySHA256: strings.Repeat("b", 64)},
	}
	evidence := v2EvidenceSnapshot{
		Present:    true,
		Runtime:    runtime,
		IndexBytes: []byte(`{"contract_version":1,"entries":[]}`),
		IndexHash:  "",
	}
	evidence.IndexHash = SHA256Hex(evidence.IndexBytes)

	t.Run("current branch empty", func(t *testing.T) {
		err := validateV2EvidenceAuthority(plan, strings.Repeat("c", 40),
			runtime.Runner, "", evidence)
		if err == nil || !strings.Contains(err.Error(), "current attached branch") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("recorded branch empty", func(t *testing.T) {
		stale := runtime
		stale.PublicationBranch = ""
		ev := evidence
		ev.Runtime = stale
		err := validateV2EvidenceAuthority(plan, strings.Repeat("c", 40),
			runtime.Runner, "main", ev)
		if err == nil || !strings.Contains(err.Error(), "runtime publication branch is empty") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("mismatch", func(t *testing.T) {
		err := validateV2EvidenceAuthority(plan, strings.Repeat("c", 40),
			runtime.Runner, "release", evidence)
		if err == nil || !strings.Contains(err.Error(), "does not match current attached branch") {
			t.Fatalf("err = %v", err)
		}
	})
}
