package closure

import (
	"encoding/json"
	"reflect"
	"testing"
)

// planContractParityBuilder is the shared, in-memory builder every
// parity fixture derives from. It mirrors the closure-plan v1
// canonical shape pinned by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-PLAN-EXECUTION-MODE-RECONCILIATION01
// and is intentionally minimal: every required field is present
// with a valid value; every optional field is omitted.
func planContractParityBuilder() map[string]any {
	return map[string]any{
		"contract_version": float64(ContractVersionV1),
		"act_id":           "ACT-LEAMAS-CONTRACT-PARITY",
		"baseline": map[string]any{
			"commit_oid": fullCommitOID,
			"tree_oid":   fullTreeOID,
		},
		"execution": map[string]any{
			"mode": string(ExecutionModeSerialFailFast),
		},
		"checks": []any{
			map[string]any{
				"id":                "noop",
				"mode":              CheckModeRun,
				"argv":              []any{"true"},
				"working_directory": ".",
				"timeout_seconds":   float64(60),
				"environment":       map[string]any{},
			},
		},
		"artifacts": []any{},
		"policy": map[string]any{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		},
	}
}

func planContractParityBytes(t *testing.T, mut func(map[string]any)) []byte {
	t.Helper()
	body := planContractParityBuilder()
	if mut != nil {
		mut(body)
	}
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal parity fixture: %v", err)
	}
	return out
}

// TestParityContractVersionEqualsDispatcher pins the contract that
// the descriptor's explicit ContractVersion matches the value the
// runtime dispatcher accepts in ValidatePlan.
func TestParityContractVersionEqualsDispatcher(t *testing.T) {
	contract := planContractV1()
	if contract.ContractVersion != ContractVersionV1 {
		t.Fatalf("descriptor contract_version = %d, want %d", contract.ContractVersion, ContractVersionV1)
	}
	// Round-trip: a canonical plan with contract_version = 1 must
	// decode and validate cleanly.
	data := planContractParityBytes(t, nil)
	if _, err := DecodePlan(data); err != nil {
		t.Fatalf("DecodePlan() error = %v", err)
	}
}

// TestParityExecutionModeEnumEqualsRuntime pins the contract that
// the descriptor's execution-mode enum authority is the same as
// the runtime SupportedExecutionModes set.
func TestParityExecutionModeEnumEqualsRuntime(t *testing.T) {
	contract := planContractV1()
	modeField, ok := contract.Root.Fields["execution"].Children.Fields["mode"]
	if !ok {
		t.Fatalf("descriptor missing /execution/mode")
	}
	if !reflect.DeepEqual(modeField.EnumAuthority, []string{string(ExecutionModeSerialFailFast)}) {
		t.Fatalf("execution.mode enum authority = %v, want [%s]", modeField.EnumAuthority, ExecutionModeSerialFailFast)
	}
	if !reflect.DeepEqual(modeField.EnumAuthority, SupportedExecutionModesString()) {
		t.Fatalf("descriptor enum != runtime SupportedExecutionModes: descriptor=%v runtime=%v",
			modeField.EnumAuthority, SupportedExecutionModesString())
	}
}

// SupportedExecutionModesString converts the runtime execution-mode
// authority to []string for parity checks.
func SupportedExecutionModesString() []string {
	values := SupportedExecutionModes()
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, string(v))
	}
	return out
}

// TestParityCheckModeEnumEqualsRuntime pins the contract that the
// descriptor's check-mode enum authority is the same as the
// runtime CheckModeRun / CheckModeExclude constants.
func TestParityCheckModeEnumEqualsRuntime(t *testing.T) {
	contract := planContractV1()
	modeField, ok := contract.Root.Fields["checks"].ItemDescriptor.Children.Fields["mode"]
	if !ok {
		t.Fatalf("descriptor missing /checks[].mode")
	}
	want := []string{CheckModeRun, CheckModeExclude}
	if !reflect.DeepEqual(modeField.EnumAuthority, want) {
		t.Fatalf("checks[].mode enum authority = %v, want %v", modeField.EnumAuthority, want)
	}
}

// TestParityArtifactRoleEnumEqualsRuntime pins the contract that
// the descriptor's artifact-role enum authority is the same as the
// runtime ArtifactRole* constants.
func TestParityArtifactRoleEnumEqualsRuntime(t *testing.T) {
	contract := planContractV1()
	roleField, ok := contract.Root.Fields["artifacts"].ItemDescriptor.Children.Fields["role"]
	if !ok {
		t.Fatalf("descriptor missing /artifacts[].role")
	}
	want := []string{
		string(ArtifactRoleInput),
		string(ArtifactRoleGeneratedOutput),
		string(ArtifactRoleNotRequired),
		string(ArtifactRoleFailureErratum),
		string(ArtifactRolePostCommitEvidence),
	}
	if !reflect.DeepEqual(roleField.EnumAuthority, want) {
		t.Fatalf("artifacts[].role enum authority = %v, want %v", roleField.EnumAuthority, want)
	}
}

// TestParityRunnerAuthorityModeEnumEqualsRuntime pins the contract
// that the descriptor's runner_authority.mode enum authority is
// the same as the runtime RunnerAuthority* constants.
func TestParityRunnerAuthorityModeEnumEqualsRuntime(t *testing.T) {
	contract := planContractV1()
	modeField, ok := contract.Root.Fields["runner_authority"].Children.Fields["mode"]
	if !ok {
		t.Fatalf("descriptor missing /runner_authority.mode")
	}
	want := []string{string(RunnerAuthoritySubjectExact), string(RunnerAuthorityToolReleaseExact)}
	if !reflect.DeepEqual(modeField.EnumAuthority, want) {
		t.Fatalf("runner_authority.mode enum authority = %v, want %v", modeField.EnumAuthority, want)
	}
}

// TestParityRequiredFieldsMatchRuntime pins the contract that the
// descriptor's required-field sets are the same as the runtime
// required-field semantics. The runtime uses go's encoding/json
// (non-pointer = required for primitives, pointer = nullable) so
// the descriptor's Required slice must match every non-pointer
// primitive field in the typed Plan struct.
func TestParityRequiredFieldsMatchRuntime(t *testing.T) {
	contract := planContractV1()
	// Root: must mirror the go Plan struct's required fields.
	wantRoot := []string{
		"contract_version",
		"act_id",
		"baseline",
		"execution",
		"checks",
		"artifacts",
		"policy",
	}
	if !reflect.DeepEqual(contract.Root.Required, wantRoot) {
		t.Fatalf("root.Required = %v, want %v", contract.Root.Required, wantRoot)
	}
	// /baseline: commit_oid and tree_oid are required strings.
	baseline := contract.Root.Fields["baseline"].Children
	wantBaseline := []string{"commit_oid", "tree_oid"}
	if !reflect.DeepEqual(baseline.Required, wantBaseline) {
		t.Fatalf("/baseline.Required = %v, want %v", baseline.Required, wantBaseline)
	}
	// /execution: mode is required (pointer in Go but JSON required).
	execution := contract.Root.Fields["execution"].Children
	wantExecution := []string{"mode"}
	if !reflect.DeepEqual(execution.Required, wantExecution) {
		t.Fatalf("/execution.Required = %v, want %v", execution.Required, wantExecution)
	}
	// /policy: all four siblings are required.
	policy := contract.Root.Fields["policy"].Children
	wantPolicy := planPolicyFields
	if !reflect.DeepEqual(policy.Required, wantPolicy) {
		t.Fatalf("/policy.Required = %v, want %v", policy.Required, wantPolicy)
	}
	// The runtime mirror in plan.go uses planPolicyFields; the
	// parity assertion catches drift between the descriptor and
	// the runtime mirror.
	// /checks[]: id and mode are required.
	checks := contract.Root.Fields["checks"].ItemDescriptor.Children
	wantChecks := []string{"id", "mode"}
	if !reflect.DeepEqual(checks.Required, wantChecks) {
		t.Fatalf("/checks.Required = %v, want %v", checks.Required, wantChecks)
	}
	// /artifacts[]: id, path, required, max_bytes, media_type are required.
	artifacts := contract.Root.Fields["artifacts"].ItemDescriptor.Children
	wantArtifacts := []string{"id", "path", "required", "max_bytes", "media_type"}
	if !reflect.DeepEqual(artifacts.Required, wantArtifacts) {
		t.Fatalf("/artifacts.Required = %v, want %v", artifacts.Required, wantArtifacts)
	}
}
