package closure

import (
	"reflect"
	"strings"
	"testing"
)

// plan_contract_correction_backcompat_test.go contains the
// backward-compatibility retention tests for the
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01-CORRECTION01-
// STRUCTURAL-PARITY-CLOSURE01 correction ACT. Each test pins one
// historical rejection or field enumeration that must continue to
// work after the descriptor rewrite.
// --- Backward-compat retention ---

func TestContractHistoricalExitcodeAndGateRejected(t *testing.T) {
	cases := []string{"exitcode", "gate"}
	for _, value := range cases {
		value := value
		t.Run(value, func(t *testing.T) {
			plan := canonicalValidPlan()
			data := []byte(strings.Replace(plan, `"mode": "serial_fail_fast"`, `"mode": "`+value+`"`, 1))
			result := ValidatePlanStructural(data)
			if result.Valid {
				t.Fatalf("historical rejected mode %q must be rejected", value)
			}
			found := false
			for _, e := range result.Errors {
				if e.Code == PlanCodeInvalidEnum && e.InstancePath == "/execution/mode" {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected invalid_enum at /execution/mode for %q: %v", value, result.Errors)
			}
		})
	}
}

func TestContractUnknownFieldsRejectedAtClosedObjects(t *testing.T) {
	plan := canonicalValidPlan()
	// Add "surprise" at root.
	data := []byte(strings.Replace(plan, `"artifacts": []`, `"artifacts": [], "surprise": "nope"`, 1))
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("root surprise must be rejected")
	}
}

func TestContractAllMissingPolicyFieldsNamed(t *testing.T) {
	// Build a plan with only one policy field present so the other
	// three are reported as missing in a single error.
	plan := `{"contract_version": 1, "act_id": "ACT-MISSING-POLICY",
		"baseline": {"commit_oid": "1111111111111111111111111111111111111111",
		             "tree_oid": "2222222222222222222222222222222222222222"},
		"execution": {"mode": "serial_fail_fast"},
		"checks": [{"id": "noop", "mode": "run", "argv": ["true"],
		           "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
		"artifacts": [],
		"policy": {"require_clean_before": true}}`
	_, err := DecodePlan([]byte(plan))
	if err == nil {
		t.Fatalf("DecodePlan accepted plan missing three policy fields")
	}
	missing, ok := err.(*PlanPolicyRequiredError)
	if !ok {
		t.Fatalf("error = %T, want *PlanPolicyRequiredError", err)
	}
	if !reflect.DeepEqual(missing.Missing, []string{"require_clean_after", "forbid_tracked_full_digests", "require_diff_check"}) {
		t.Fatalf("missing = %v, want [require_clean_after forbid_tracked_full_digests require_diff_check]", missing.Missing)
	}
}

// --- Test helpers ---

// canonicalValidPlan returns a closure-plan v1 JSON document that
// passes every validator. It is the regression-comparator fixture
// for the descriptor-driven DescriptorExample; both must agree on
// every validation stage.
func canonicalValidPlan() string {
	return `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-CANONICAL",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {
      "id": "noop",
      "mode": "run",
      "argv": ["true"],
      "working_directory": ".",
      "timeout_seconds": 60,
      "environment": {}
    }
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`
}

func canonicalValidPlanWithArtifact(artifact string) []byte {
	plan := canonicalValidPlan()
	return []byte(strings.Replace(plan, `"artifacts": []`, `"artifacts": [`+artifact+`]`, 1))
}

func replaceField(plan, needle, replacement string) []byte {
	return []byte(strings.Replace(plan, needle, replacement, 1))
}
