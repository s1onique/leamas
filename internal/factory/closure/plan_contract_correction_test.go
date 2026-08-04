package closure

import (
	"reflect"
	"strings"
	"sync"
	"testing"
)

// The tests in this file cover every blocking defect the
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01-CORRECTION01-
// STRUCTURAL-PARITY-CLOSURE01 correction ACT named. Each test name
// begins with "TestContract" or "TestStructural" so the focused-test
// regex the directive ACT mandates matches them.

// --- Phase 2: separate Nullable from Pointer ---

func TestContractRequiredNullExecutionModeRejected(t *testing.T) {
	data := replaceField(canonicalValidPlan(), `"mode": "serial_fail_fast"`, `"mode": null`)
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("required null /execution/mode must be rejected")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeInvalidType && e.InstancePath == "/execution/mode" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid_type at /execution/mode: %v", result.Errors)
	}
}

func TestContractRequiredNullPolicyFieldRejected(t *testing.T) {
	data := replaceField(canonicalValidPlan(), `"require_clean_before": true`, `"require_clean_before": null`)
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("required null /policy/require_clean_before must be rejected")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeInvalidType && e.InstancePath == "/policy/require_clean_before" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid_type at /policy/require_clean_before: %v", result.Errors)
	}
}

func TestContractRequiredNullArtifactRequiredRejected(t *testing.T) {
	// Insert a single artifact so the policy of pointer-backed
	// booleans on a required artifact is exercised.
	raw := canonicalValidPlanWithArtifact(`{"id":"summary","path":".factory/x.json","required":null,"max_bytes":1,"media_type":"application/json"}`)
	result := ValidatePlanStructural(raw)
	if result.Valid {
		t.Fatalf("required null /artifacts/0/required must be rejected")
	}
}

func TestContractOptionalNullRunnerAuthorityAccepted(t *testing.T) {
	// Optional pointer-backed object fields accept null per the
	// documented contract.
	plan := canonicalValidPlan()
	data := []byte(strings.Replace(plan, `"artifacts": [],`, `"artifacts": [], "runner_authority": null,`, 1))
	result := ValidatePlanStructural(data)
	if !result.Valid {
		t.Fatalf("optional null /runner_authority must be accepted; errors=%v", result.Errors)
	}
}

// --- Phase 3: free-form string map ---

func TestContractEnvironmentEmptyAccepted(t *testing.T) {
	data := []byte(canonicalValidPlan())
	result := ValidatePlanStructural(data)
	if !result.Valid {
		t.Fatalf("plan with empty environment must pass structural validation")
	}
}

func TestContractEnvironmentStringMapAccepted(t *testing.T) {
	plan := canonicalValidPlan()
	// Replace the check environment with a non-empty string map.
	data := []byte(strings.Replace(plan, `"environment": {}`, `"environment": {"FOO": "bar", "EMPTY": ""}`, 1))
	result := ValidatePlanStructural(data)
	if !result.Valid {
		t.Fatalf("environment with arbitrary string entries must pass: %v", result.Errors)
	}
}

func TestContractEnvironmentNonStringValuesRejected(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"integer-value", `{"FOO": 42}`},
		{"object-value", `{"FOO": {}}`},
		{"array-value", `{"FOO": [1]}`},
		{"boolean-value", `{"FOO": true}`},
		{"null-value", `{"FOO": null}`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			plan := canonicalValidPlan()
			data := []byte(strings.Replace(plan, `"environment": {}`, `"environment": `+tc.value, 1))
			result := ValidatePlanStructural(data)
			if result.Valid {
				t.Fatalf("environment with %s must be rejected", tc.name)
			}
			found := false
			for _, e := range result.Errors {
				if e.Code == PlanCodeInvalidType && e.InstancePath == "/checks/0/environment/FOO" {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected invalid_type at /checks/0/environment/FOO: %v", result.Errors)
			}
		})
	}
}

// --- Phase 5: contract version diagnostic truth ---

func TestContractVersionCategories(t *testing.T) {
	cases := []struct {
		name       string
		version    string
		wantValid  bool
		wantCode   PlanValidationCode
		wantVerInt int
	}{
		{"valid-one", `"contract_version": 1`, true, "", 1},
		{"missing", `"contract_version_removed": ""`, false, PlanCodeRequiredPropertyMissing, 0},
		{"zero", `"contract_version": 0`, false, PlanCodeUnsupportedContractVersion, 0},
		{"two", `"contract_version": 2`, false, PlanCodeUnsupportedContractVersion, 2},
		{"negative-one", `"contract_version": -1`, false, PlanCodeUnsupportedContractVersion, -1},
		{"float-one-point-five", `"contract_version": 1.5`, false, PlanCodeInvalidType, 0},
		{"string-one", `"contract_version": "1"`, false, PlanCodeInvalidType, 0},
		{"null-version", `"contract_version": null`, false, PlanCodeInvalidType, 0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			plan := canonicalValidPlan()
			// The "missing" case replaces with an unrelated extra field so
			// the document remains syntactically valid; the descriptor
			// still reports the contract_version field as missing.
			version := tc.version
			if tc.name == "missing" {
				version = `"contract_version_removed": 1`
			}
			data := []byte(strings.Replace(plan, `"contract_version": 1`, version, 1))
			result := ValidatePlanStructural(data)
			if result.Valid != tc.wantValid {
				t.Fatalf("valid = %v, want %v (errors=%v)", result.Valid, tc.wantValid, result.Errors)
			}
			if result.ContractVersion != tc.wantVerInt {
				t.Fatalf("ContractVersion = %d, want %d", result.ContractVersion, tc.wantVerInt)
			}
			if tc.wantCode == "" {
				return
			}
			found := false
			for _, e := range result.Errors {
				if e.Code == tc.wantCode {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected code %s; errors=%v", tc.wantCode, result.Errors)
			}
		})
	}
}

// --- Phase 6: policy field authority ---

func TestContractPolicyFieldAuthorityCount(t *testing.T) {
	// Exactly one authority: the descriptor's /policy.Required.
	order := PolicyFieldOrder()
	if len(order) != 4 {
		t.Fatalf("PolicyFieldOrder length = %d, want 4", len(order))
	}
	want := []string{
		"require_clean_before",
		"require_clean_after",
		"forbid_tracked_full_digests",
		"require_diff_check",
	}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

// --- Phase 9: mode-dependent applicability ---

func TestContractModeRunRequiresArgvAndEnvironment(t *testing.T) {
	// /checks/0 with mode=run but no argv must be required-missing.
	plan := canonicalValidPlan()
	data := []byte(strings.Replace(plan, `"argv": ["true"]`, `"argv": []`, 1))
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("argv-empty on mode=run must be rejected by structural (minItems) and/or mode-dependent applicability")
	}
}

func TestContractModeExcludeForbidsArgv(t *testing.T) {
	// Build a plan with one exclude check that has argv.
	plan := `{"contract_version": 1, "act_id": "ACT-EXCL-ARGV",
		"baseline": {"commit_oid": "1111111111111111111111111111111111111111",
		             "tree_oid": "2222222222222222222222222222222222222222"},
		"execution": {"mode": "serial_fail_fast"},
		"checks": [{"id": "x", "mode": "exclude", "reason": "noop", "argv": ["true"],
		           "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
		"artifacts": [],
		"policy": {"require_clean_before": true, "require_clean_after": true,
		          "forbid_tracked_full_digests": true, "require_diff_check": true}}`
	composed := ValidatePlanComposed([]byte(plan))
	if composed.Valid {
		t.Fatalf("mode=exclude with argv present must fail semantic validation")
	}
}

func TestContractModeExcludeRequiresReason(t *testing.T) {
	plan := `{"contract_version": 1, "act_id": "ACT-EXCL-NOREASON",
		"baseline": {"commit_oid": "1111111111111111111111111111111111111111",
		             "tree_oid": "2222222222222222222222222222222222222222"},
		"execution": {"mode": "serial_fail_fast"},
		"checks": [{"id": "x", "mode": "exclude"}],
		"artifacts": [],
		"policy": {"require_clean_before": true, "require_clean_after": true,
		          "forbid_tracked_full_digests": true, "require_diff_check": true}}`
	composed := ValidatePlanComposed([]byte(plan))
	if composed.Valid {
		t.Fatalf("mode=exclude without reason must fail semantic validation")
	}
}

// --- Phase 10: composed validation ---

func TestContractComposedPipelineShortCircuitsOnStructuralFailure(t *testing.T) {
	// Trailing second object: structural fails; semantic MUST NOT
	// run, and Valid MUST be false.
	data := []byte(canonicalValidPlan() + ` {"x": 1}`)
	composed := ValidatePlanComposed(data)
	if composed.Valid {
		t.Fatalf("composed validation must report invalid on structural failure")
	}
	if composed.Decoded {
		t.Fatalf("composed validation must not decode when structural fails")
	}
	if len(composed.Structural.Errors) == 0 {
		t.Fatalf("structural diagnostics must be populated")
	}
}

func TestContractComposedPipelineValidatesCanonicalPlan(t *testing.T) {
	data := []byte(canonicalValidPlan())
	composed := ValidatePlanComposed(data)
	if !composed.Valid {
		t.Fatalf("canonical plan must compose-validate: structural=%v semantic=%v",
			composed.Structural.Errors, composed.SemanticErrors)
	}
}

// --- Determinism ---

func TestContractDiagnosticsDeterministicCount20(t *testing.T) {
	plan := []byte(canonicalValidPlan())
	for i := 0; i < 20; i++ {
		result := ValidatePlanStructural(plan)
		if !result.Valid {
			t.Fatalf("iteration %d: canonical plan must pass", i)
		}
	}
}

func TestContractDiagnosticsConcurrentDeterministic(t *testing.T) {
	plan := []byte(canonicalValidPlan())
	const workers = 8
	const iterations = 16
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				result := ValidatePlanStructural(plan)
				if !result.Valid {
					t.Errorf("worker iteration %d: must pass", i)
					return
				}
			}
		}()
	}
	wg.Wait()
}
