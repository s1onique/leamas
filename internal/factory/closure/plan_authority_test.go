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
