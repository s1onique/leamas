package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

// TestClosurePolicyProfileTypedFieldsPreserveAdversarialValues
// is the CORRECTION02 closure-side adversarial matrix.
// It proves the closure adapter reads
// plancontract.DecodeError.Profile/CheckID as typed
// fields, not as parsed substrings of the human-readable
// Message. Before CORRECTION02 the adapter reconstructed
// ProfileName/CheckID by scanning the leaf's display
// prose for quote characters; an embedded `"` would
// corrupt the reconstructed value. After CORRECTION02 the
// adapter passes the typed fields through verbatim.
//
// Rows:
//   - A1 unknown profile name with embedded "
//   - A2 unknown profile name with backslash
//   - A3 unknown profile name with non-ASCII (Unicode)
//   - A4 unknown profile name with embedded tab/newline
//   - A5 missing-check carries CheckID structurally
func TestClosurePolicyProfileTypedFieldsPreserveAdversarialValues(t *testing.T) {
	cases := []struct {
		name      string
		rawValue  string
		wantValue string
	}{
		{
			name:      "A1_unknown_profile_with_embedded_quote",
			rawValue:  `foo"bar`,
			wantValue: `foo"bar`,
		},
		{
			name:      "A2_unknown_profile_with_backslash",
			rawValue:  `foo\bar`,
			wantValue: `foo\bar`,
		},
		{
			name:      "A3_unknown_profile_with_unicode",
			rawValue:  `профиль-неизвестен`,
			wantValue: `профиль-неизвестен`,
		},
		{
			name:      "A4_unknown_profile_with_tab_and_newline",
			rawValue:  "foo\tbar\nbaz",
			wantValue: "foo\tbar\nbaz",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			plan := canonicalPlan()
			plan.PolicyProfile = tc.rawValue
			err := ValidatePlan(plan)
			if err == nil {
				t.Fatalf("expected error for profile %q", tc.rawValue)
			}
			var ppe *PolicyProfileError
			if !errors.As(err, &ppe) {
				t.Fatalf("error is not *PolicyProfileError: %v", err)
			}
			if ppe.Kind != PolicyProfileErrorUnknown {
				t.Fatalf("Kind = %d, want PolicyProfileErrorUnknown", ppe.Kind)
			}
			if ppe.ProfileName != tc.wantValue {
				t.Fatalf("ProfileName = %q (len=%d), want %q (len=%d)",
					ppe.ProfileName, len(ppe.ProfileName),
					tc.wantValue, len(tc.wantValue))
			}
		})
	}

	// A5: missing-check carries CheckID structurally.
	// The Expert's required addition: the missing-check
	// path is the second half of the original display-
	// prose roundtrip. The test verifies the adapter
	// reads decodeErr.CheckID directly.
	t.Run("A5_missing_check_carries_check_id_structurally", func(t *testing.T) {
		plan := canonicalPlan()
		// Drop the diff-check (the canonical last check);
		// the leaf must reject because leamas-act-v1
		// requires it. The first missing required check
		// is "focused-count-1" because the loop iterates
		// in the declared order.
		plan.PolicyProfile = PolicyProfileLeamasActV1
		var kept []PlanCheck
		for _, c := range plan.Checks {
			if c.ID != "focused-count-1" {
				kept = append(kept, c)
			}
		}
		plan.Checks = kept
		err := ValidatePlan(plan)
		if err == nil {
			t.Fatal("expected error for missing required check")
		}
		var ppe *PolicyProfileError
		if !errors.As(err, &ppe) {
			t.Fatalf("error is not *PolicyProfileError: %v", err)
		}
		if ppe.Kind != PolicyProfileErrorMissingCheck {
			t.Fatalf("Kind = %d, want PolicyProfileErrorMissingCheck", ppe.Kind)
		}
		if ppe.CheckID != "focused-count-1" {
			t.Fatalf("CheckID = %q, want focused-count-1", ppe.CheckID)
		}
		if ppe.ProfileName != PolicyProfileLeamasActV1 {
			t.Fatalf("ProfileName = %q, want %q", ppe.ProfileName, PolicyProfileLeamasActV1)
		}
	})
}
