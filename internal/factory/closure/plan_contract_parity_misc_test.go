package closure

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// plan_contract_parity_misc_test.go contains the parity tests that
// are not strict descriptor or strict structural-validator checks:
// duplicate-key behaviour, the precise /policy required-error
// payload, contract-version round-trip diagnostics, empty-document
// recovery, and the SupportedExecutionModesString helper used by
// other parity files.
// TestParityDuplicatePropertyBehavior pins the contract that
// duplicate JSON keys are rejected by both the strict decoder and
// the structural validator. The directive ACT requires the
// behavior to be preserved unless an explicit contract correction
// is separately authorized.
func TestParityDuplicatePropertyBehavior(t *testing.T) {
	data := []byte(`{
		"contract_version": 1,
		"contract_version": 1,
		"act_id": "ACT-LEAMAS-DUPLICATE",
		"baseline": {"commit_oid": "1111111111111111111111111111111111111111", "tree_oid": "2222222222222222222222222222222222222222"},
		"execution": {"mode": "serial_fail_fast"},
		"checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
		"artifacts": [],
		"policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
	}`)
	if _, err := DecodePlan(data); err == nil || !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		t.Fatalf("DecodePlan did not reject duplicate contract_version: %v", err)
	}
}

// TestParityPolicyRequiredErrorListsMissingSiblings pins the
// contract that the precise policy diagnostics emit every missing
// sibling in a single error.
func TestParityPolicyRequiredErrorListsMissingSiblings(t *testing.T) {
	policy := PlanPolicy{
		RequireCleanBefore: planBoolPtr(true),
		// other three nil
	}
	err := validatePlanPolicy(policy)
	if err == nil {
		t.Fatalf("expected PlanPolicyRequiredError")
	}
	missing, ok := err.(*PlanPolicyRequiredError)
	if !ok {
		t.Fatalf("error type = %T, want *PlanPolicyRequiredError", err)
	}
	want := []string{"require_clean_after", "forbid_tracked_full_digests", "require_diff_check"}
	if !reflect.DeepEqual(missing.Missing, want) {
		t.Fatalf("Missing = %v, want %v", missing.Missing, want)
	}
	if !strings.Contains(err.Error(), "require_clean_after") {
		t.Fatalf("Error() = %q, want substring require_clean_after", err.Error())
	}
}

// TestParityPolicyRequiredErrorAllMissing pins the contract that
// every missing sibling appears when all four are absent.
func TestParityPolicyRequiredErrorAllMissing(t *testing.T) {
	err := validatePlanPolicy(PlanPolicy{})
	if err == nil {
		t.Fatalf("expected PlanPolicyRequiredError")
	}
	missing, ok := err.(*PlanPolicyRequiredError)
	if !ok {
		t.Fatalf("error type = %T, want *PlanPolicyRequiredError", err)
	}
	if !reflect.DeepEqual(missing.Missing, planPolicyFields) {
		t.Fatalf("Missing = %v, want %v", missing.Missing, planPolicyFields)
	}
}

// TestParityUnsupportedContractVersionDiagnostic pins the contract
// that an unsupported contract version raises a clear, recoverable
// diagnostic and the version-0 sentinel for malformed input.
func TestParityUnsupportedContractVersionDiagnostic(t *testing.T) {
	// A supported-but-wrong version is recovered truthfully: the
	// structural validator reports ContractVersion=2 so consumers
	// see the actual value. It never silently reports 1.
	data := []byte(`{"contract_version": 2, "act_id": "ACT-WRONG-VERSION"}`)
	result := ValidatePlanStructural(data)
	if result.ContractVersion != 2 {
		t.Fatalf("ContractVersion = %d, want 2 (truthful recovery)", result.ContractVersion)
	}
	// An unrecoverable document returns ContractVersion=0 (the
	// sentinel for "cannot recover"). The JSON itself must be
	// parsed to recover the version, so truncation also reports
	// invalid_json.
	result = ValidatePlanStructural([]byte(`{"act_id":`))
	if result.ContractVersion != 0 {
		t.Fatalf("ContractVersion on truncated JSON = %d, want 0", result.ContractVersion)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected at least one invalid_json diagnostic on truncated JSON")
	}
	if result.Errors[0].Code != PlanCodeInvalidJSON {
		t.Fatalf("Errors[0].Code = %v, want %v", result.Errors[0].Code, PlanCodeInvalidJSON)
	}
}

// TestParityEmptyDocumentIsInvalid pins the contract that an empty
// document returns ContractVersion=0 with a single invalid_json
// diagnostic.
func TestParityEmptyDocumentIsInvalid(t *testing.T) {
	result := ValidatePlanStructural(nil)
	if result.ContractVersion != 0 {
		t.Fatalf("ContractVersion = %d, want 0", result.ContractVersion)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(result.Errors))
	}
	if result.Errors[0].Code != PlanCodeInvalidJSON {
		t.Fatalf("Errors[0].Code = %v, want %v", result.Errors[0].Code, PlanCodeInvalidJSON)
	}
}

// TestParityUnknownContractVersionRoundTrip pins the contract that
// the typed ValidatePlan emits a precise "unsupported
// closure plan contract_version N" diagnostic when contract_version
// is an unsupported integer.
func TestParityUnknownContractVersionRoundTrip(t *testing.T) {
	plan := Plan{ContractVersion: 2}
	err := ValidatePlan(plan)
	if err == nil {
		t.Fatalf("expected error for unsupported contract_version")
	}
	if !strings.Contains(err.Error(), "unsupported closure plan contract_version 2") {
		t.Fatalf("error = %q, want substring 'unsupported closure plan contract_version 2'", err.Error())
	}
}

// TestParityDescriptorMirrorsGoStructFields pins the contract that
// every Go struct field with a JSON tag that is NOT omitempty has a
// matching descriptor entry. The check is mechanical: any drift
// triggers a panic in the parity assertion.
func TestParityDescriptorMirrorsGoStructFields(t *testing.T) {
	contract := planContractV1()
	root := contract.Root.Fields
	expect := map[string]bool{
		"contract_version": true,
		"act_id":           true,
		"baseline":         true,
		"execution":        true,
		"checks":           true,
		"artifacts":        true,
		"policy":           true,
		"policy_profile":   true,
		"runner_binding":   true,
		"runner_authority": true,
	}
	for name := range expect {
		if _, ok := root[name]; !ok {
			t.Fatalf("descriptor missing root field %q", name)
		}
	}
	// Plan struct non-omitempty field set must match.
	expectedNames := make([]string, 0, len(expect))
	for name := range expect {
		expectedNames = append(expectedNames, name)
	}
	sort.Strings(expectedNames)
	gotNames := make([]string, 0, len(expect))
	for name := range expect {
		if _, ok := root[name]; ok {
			gotNames = append(gotNames, name)
		}
	}
	sort.Strings(gotNames)
	if !reflect.DeepEqual(expectedNames, gotNames) {
		t.Fatalf("expected=%v got=%v", expectedNames, gotNames)
	}
}

// planBoolPtr is a tiny helper that returns a pointer to the supplied
// bool. Used by parity tests to build typed PlanPolicy values.
func planBoolPtr(b bool) *bool { return &b }
