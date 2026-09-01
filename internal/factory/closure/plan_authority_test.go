package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

const fullCommitOIDForAuthority = "1111111111111111111111111111111111111111"

func TestClosurePlanAuthorityRejectsUnknownProfile(t *testing.T) {
	plan := canonicalPlan()
	plan.PolicyProfile = "mystery-profile"
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePlanAuthorityRejectsDisabledProfile(t *testing.T) {
	plan := canonicalPlan()
	plan.PolicyProfile = "indeep-act-v1"
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "not yet implemented") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePlanAuthorityEnforcesAllLeamasChecks(t *testing.T) {
	plan := canonicalPlan()
	plan.Checks = plan.Checks[:1]
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "policy profile") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePlanAuthorityRejectsTrailingArgvMutation(t *testing.T) {
	plan := canonicalPlan()
	for index := range plan.Checks {
		if plan.Checks[index].ID == "focused-count-1" {
			plan.Checks[index].Argv = append([]string(nil), plan.Checks[index].Argv...)
			plan.Checks[index].Argv = append(plan.Checks[index].Argv, "-run", "^$")
		}
	}
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "policy profile") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePlanAuthorityRejectsShortMutation(t *testing.T) {
	plan := canonicalPlan()
	for index := range plan.Checks {
		if plan.Checks[index].ID == "focused-count-1" {
			plan.Checks[index].Argv = []string{"go", "test", "-count=1", "-short", "./internal/factory/closure/...", "./cmd/leamas/..."}
		}
	}
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "policy profile") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePlanAuthorityRejectsExtraPackageMutation(t *testing.T) {
	plan := canonicalPlan()
	for index := range plan.Checks {
		if plan.Checks[index].ID == "focused-count-1" {
			plan.Checks[index].Argv = []string{"go", "test", "-count=1", "./internal/factory/closure/..."}
		}
	}
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "policy profile") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePlanAuthorityRejectsGateFastTargetMutation(t *testing.T) {
	plan := canonicalPlan()
	for index := range plan.Checks {
		if plan.Checks[index].ID == "gate-fast" {
			plan.Checks[index].Argv = []string{"make", "test"}
		}
	}
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "policy profile") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePlanAuthorityRejectsCountZero(t *testing.T) {
	plan := canonicalPlan()
	for index := range plan.Checks {
		if plan.Checks[index].ID == "focused-count-1" {
			plan.Checks[index].Argv = []string{"go", "test", "-count=0", "./internal/factory/closure/...", "./cmd/leamas/..."}
		}
	}
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "policy profile") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePlanAuthorityRejectsDryRun(t *testing.T) {
	plan := canonicalPlan()
	for index := range plan.Checks {
		if plan.Checks[index].ID == "build" {
			plan.Checks[index].Argv = []string{"go", "build", "-buildvcs=true", "-dry-run", "-trimpath", "-o", "/tmp/leamas-closure-protocol-v1-self", "./cmd/leamas"}
		}
	}
	if err := ValidatePlan(plan); err == nil || !strings.Contains(err.Error(), "policy profile") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosurePlanAuthorityAcceptsExactPlan(t *testing.T) {
	plan := canonicalPlan()
	if err := ValidatePlan(plan); err != nil {
		t.Fatalf("error = %v", err)
	}
}

// TestClosurePlanAuthorityAcceptsOmittedPolicyProfile
// is the F1.5 positive control: a structurally valid plan
// without any policy_profile claim must still pass the
// leaf's authority pass. The closure package historically
// treated omitted as "no profile enforced", and
// B2-R7-CORRECTION01 preserves that legacy contract.
func TestClosurePlanAuthorityAcceptsOmittedPolicyProfile(t *testing.T) {
	plan := canonicalPlan()
	plan.PolicyProfile = ""
	if err := ValidatePlan(plan); err != nil {
		t.Fatalf("error = %v", err)
	}
}

// TestClosurePlanAuthorityRejectsOmittedRequiredCheck is
// the F1.3 canonical authority row: the candidate plan
// claims an enabled profile but is missing one of its
// required checks. The leaf's missing_policy_profile_check
// code must surface as a non-nil error.
func TestClosurePlanAuthorityRejectsOmittedRequiredCheck(t *testing.T) {
	plan := canonicalPlan()
	// Drop the diff-check (the canonical last check); the
	// leaf must reject because leamas-act-v1 requires it.
	var kept []PlanCheck
	for _, c := range plan.Checks {
		if c.ID != "diff-check" {
			kept = append(kept, c)
		}
	}
	plan.Checks = kept
	err := ValidatePlan(plan)
	if err == nil {
		t.Fatal("expected error for omitted required check")
	}
	if !strings.Contains(err.Error(), "policy profile") {
		t.Fatalf("error = %v, want substring 'policy profile'", err)
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

var _ = sha256.New
var _ = hex.EncodeToString
var _ = fullCommitOIDForAuthority
