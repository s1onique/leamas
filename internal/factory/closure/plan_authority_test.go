package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

const fullCommitOIDForAuthority = "1111111111111111111111111111111111111111"
const fullTreeOIDForAuthority = "2222222222222222222222222222222222222222"

func canonicalPlanWithProfile(profile string, freezeCommit, freezeBlob string) Plan {
	plan := canonicalPlan()
	plan.PolicyProfile = profile
	plan.Freeze = PlanFreeze{CommitOID: freezeCommit, BlobOID: freezeBlob}
	return plan
}

func frozenPlanBytes(t *testing.T) []byte {
	t.Helper()
	plan := canonicalPlan()
	plan.Freeze = PlanFreeze{CommitOID: fullCommitOIDForAuthority, BlobOID: ""}
	plan.Freeze.BlobOID = ""
	if err := computeAndApplyFreezeBlob(&plan); err != nil {
		t.Fatal(err)
	}
	data, err := jsonMarshalPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestClosurePlanFreezeBlobsBindAcross(t *testing.T) {
	bytes := frozenPlanBytes(t)
	blobHash := sha256.Sum256(bytes)
	blob := hex.EncodeToString(blobHash[:])
	plan := canonicalPlanWithProfile("leamas-act-v1", fullCommitOIDForAuthority, blob)
	plan.Freeze.BlobOID = blob
	if err := ValidatePlan(plan); err != nil {
		t.Fatalf("ValidatePlan() error = %v", err)
	}
}

func TestClosurePlanFreezeRejectsMissingBlobBinding(t *testing.T) {
	plan := canonicalPlanWithProfile("leamas-act-v1", fullCommitOIDForAuthority, "")
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "blob_oid") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePlanFreezeRejectsMissingCommitBinding(t *testing.T) {
	bytes := frozenPlanBytes(t)
	blobHash := sha256.Sum256(bytes)
	blob := hex.EncodeToString(blobHash[:])
	plan := canonicalPlanWithProfile("leamas-act-v1", "", blob)
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "freeze.commit_oid") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePolicyProfileEnforcesRequiredChecks(t *testing.T) {
	bytes := frozenPlanBytes(t)
	blobHash := sha256.Sum256(bytes)
	blob := hex.EncodeToString(blobHash[:])
	plan := canonicalPlan()
	plan.Checks = plan.Checks[:1]
	plan.Freeze = PlanFreeze{CommitOID: fullCommitOIDForAuthority, BlobOID: blob}
	plan.PolicyProfile = "leamas-act-v1"
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "policy profile") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosureUnknownPolicyProfileIsRejected(t *testing.T) {
	plan := canonicalPlanWithProfile("mystery-profile", fullCommitOIDForAuthority, strings.Repeat("a", 64))
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "policy_profile") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosureSubjectExactRunnerBindingRejectsMismatchedRevision(t *testing.T) {
	manifest := passingManifest()
	manifest.Runner.VCSRevision = strings.Repeat("a", 40)
	if err := VerifyRunnerBinding(RunnerBindingSubjectExact, manifest); err == nil {
		t.Fatal("expected rejection when runner.vcs_revision != subject.commit_oid")
	}
}

func TestClosureSubjectExactRunnerBindingAcceptsMatchingRevision(t *testing.T) {
	manifest := passingManifest()
	manifest.Runner.VCSRevision = manifest.Subject.CommitOID
	if err := VerifyRunnerBinding(RunnerBindingSubjectExact, manifest); err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestClosureTrustedCleanRunnerBindingAlwaysAccepts(t *testing.T) {
	manifest := passingManifest()
	manifest.Runner.VCSRevision = strings.Repeat("a", 40)
	if err := VerifyRunnerBinding(RunnerBindingTrustedClean, manifest); err != nil {
		t.Fatalf("error = %v", err)
	}
}
