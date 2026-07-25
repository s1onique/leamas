// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultV2FinalizeNewPreparesBButFailsClosed(t *testing.T) {
	repo, subject, subjectTree := prepareObjectTransactionRepository(t, ObjectFormatSHA1)
	staging := makeEvidenceStagingUnder(t,
		filepath.Join(repo, ".factory", "closure-evidence", objectTransactionActID),
		map[string]string{"authority-check.stdout": "ok\n", "authority-check.stderr": ""})
	input := finalizeTestInput(repo, subject, subjectTree, staging)
	before := captureRepositoryWindow(t, repo)
	result, err := defaultV2FinalizeNew(context.Background(), input)
	if result != nil {
		t.Fatalf("object-only transaction returned successful result: %+v", result)
	}
	var incomplete *PublicationIncompleteError
	if !errors.As(err, &incomplete) || incomplete.ClosureCommit == "" ||
		incomplete.TagObject == "" || incomplete.EvidenceHash == "" {
		t.Fatalf("error = %v, want populated PublicationIncompleteError", err)
	}
	assertRepositoryWindowEqual(t, before, captureRepositoryWindow(t, repo))
	// The finalizer leaves the qualified evidence index in STAGING.
	// The orchestrator owns the unique publish step.
	if _, statErr := os.Stat(filepath.Join(staging, v2EvidenceIndexName)); statErr != nil {
		t.Fatalf("qualified evidence index absent in staging: %v", statErr)
	}
	finalPath := evidenceDirectoryPath(repo, objectTransactionActID, subject)
	if _, statErr := os.Lstat(finalPath); !os.IsNotExist(statErr) {
		t.Fatalf("finalizer created final evidence; the orchestrator must own the unique publish: %v", statErr)
	}
	if refs := rawGitStdout(t, repo, "for-each-ref", "refs/tags"); len(refs) != 0 {
		t.Fatalf("Object Transaction B created tag ref: %q", refs)
	}
}

func TestDefaultV2FinalizeNewNegativeVerdictPublishesNoEvidenceOrTag(t *testing.T) {
	repo, subject, subjectTree := prepareObjectTransactionRepository(t, ObjectFormatSHA1)
	parent := filepath.Join(repo, ".factory", "closure-evidence", objectTransactionActID)
	staging := makeEvidenceStagingUnder(t, parent, map[string]string{"check.stdout": "bad\n"})
	input := finalizeTestInput(repo, subject, subjectTree, staging)
	input.Checks[0].Status = CheckStatusFail
	finalPath := evidenceDirectoryPath(repo, objectTransactionActID, subject)
	_, err := defaultV2FinalizeNew(context.Background(), input)
	if err == nil || !strings.Contains(err.Error(), "verdict is fail") {
		t.Fatalf("negative verdict error = %v", err)
	}
	if _, statErr := os.Lstat(finalPath); !os.IsNotExist(statErr) {
		t.Fatalf("negative verdict created final evidence: %v", statErr)
	}
	if refs := rawGitStdout(t, repo, "for-each-ref", "refs/tags"); len(refs) != 0 {
		t.Fatalf("negative verdict created tag ref: %q", refs)
	}
}

func TestDefaultV2FinalizeNewTagFailureLeavesFinalEvidenceAbsent(t *testing.T) {
	repo, subject, subjectTree := prepareObjectTransactionRepository(t, ObjectFormatSHA1)
	parent := filepath.Join(repo, ".factory", "closure-evidence", objectTransactionActID)
	staging := makeEvidenceStagingUnder(t, parent, map[string]string{"check.stdout": "ok\n"})
	input := finalizeTestInput(repo, subject, subjectTree, staging)
	input.Runner.BinarySHA256 = "invalid"
	before := captureRepositoryWindow(t, repo)
	_, err := defaultV2FinalizeNew(context.Background(), input)
	if err == nil || !strings.Contains(err.Error(), "runner binary SHA-256") {
		t.Fatalf("tag failure error = %v", err)
	}
	assertRepositoryWindowEqual(t, before, captureRepositoryWindow(t, repo))
	finalPath := evidenceDirectoryPath(repo, objectTransactionActID, subject)
	if _, statErr := os.Lstat(finalPath); !os.IsNotExist(statErr) {
		t.Fatalf("tag failure created final evidence: %v", statErr)
	}
}

func finalizeTestInput(repo, subject, subjectTree, staging string) v2FinalizeInput {
	check := canonicalArtifactTestInput().Checks[0]
	check.SubjectTreeOID = subjectTree
	plan := minimalValidPlan()
	plan.ActID = objectTransactionActID
	return v2FinalizeInput{
		Dependencies: v2Dependencies{Git: RealGit{}}, RepositoryRoot: repo,
		ObjectFormat: ObjectFormatSHA1, Plan: plan,
		CanonicalPlanPath:  "docs/closure-plans/" + objectTransactionActID + ".json",
		AuthoritativeBytes: []byte("plan\n"), PlanBlobOID: strings.Repeat("a", 40),
		FreezeCommit: subject, FreezeTree: subjectTree,
		SubjectCommit: subject, SubjectTree: subjectTree, Branch: "main",
		EvidenceDirectory: staging,
		Runner:            RunnerIdentity{LeamasVersion: "test", BinarySHA256: strings.Repeat("1", 64), VCSRevision: subject},
		Checks:            []CheckResult{check},
		Patch: policyEvaluation[PatchHygiene]{
			Value: PatchHygiene{Status: CheckStatusPass}, Passed: true,
		},
		Closure: policyEvaluation[ClosurePolicyResult]{
			Value: ClosurePolicyResult{TrackedFullDigestStatus: CheckStatusPass}, Passed: true,
		},
	}
}
