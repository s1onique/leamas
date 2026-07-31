package closure

import (
	"testing"
)

// plan_contract_applicability_authority_test.go directly
// exercises applicability rules so the production walker and
// the descriptor-driven helper stay in lockstep. Each test name
// matches the ApplicabilityAuthority focused-test regex.

// minimalChecksField returns a synthetic /checks field whose
// per-item descriptor carries one applicability rule driven by
// the test. The synthetic root is otherwise empty; the walker
// only consults the /checks subtree.
func minimalChecksField(itemRules ...fieldApplicabilityRule) planContractV1Descriptor {
	return planContractV1Descriptor{
		ContractVersion: ContractVersionV1,
		Root: planObjectDescriptor{
			Path: "",
			Fields: map[string]planFieldDescriptor{
				"checks": {
					JSONName: "checks",
					Kind:     kindArray,
					Required: true,
					MinItems: 1,
					ItemDescriptor: &planFieldDescriptor{
						JSONName: "checks[]",
						Kind:     kindObject,
						Children: &planObjectDescriptor{
							Path:     "/checks",
							Required: []string{"id", "mode"},
							Fields: map[string]planFieldDescriptor{
								"id": {
									JSONName: "id",
									Kind:     kindString,
									Required: true,
								},
								"mode": {
									JSONName: "mode",
									Kind:     kindEnum,
									Required: true,
								},
								"argv": {
									JSONName:           "argv",
									Kind:               kindArray,
									Required:           false,
									ApplicabilityRules: itemRules,
								},
							},
						},
					},
				},
			},
		},
	}
}

// TestApplicabilityAuthorityRequiredRuleUnderMatchingMode proves
// that a field whose ApplicabilityRule under the matching sibling
// value is PresenceRequired emits a required_property_missing
// diagnostic when the field is absent from the JSON document.
func TestApplicabilityAuthorityRequiredRuleUnderMatchingMode(t *testing.T) {
	contract := minimalChecksField(
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
	)
	root := map[string]any{
		"checks": []any{
			map[string]any{
				"id":   "noop",
				"mode": CheckModeRun,
			},
		},
	}
	diags := ValidateModeDependentApplicability(root, contract)
	if !hasDiagnosticAt(diags, "/checks/0/argv", PlanCodeRequiredPropertyMissing) {
		t.Fatalf("expected required_property_missing at /checks/0/argv; got %v", diags)
	}
}

// TestApplicabilityAuthorityOptionalRuleUnderMatchingMode proves
// that a PresenceOptional rule under the matching sibling value
// emits no diagnostic regardless of presence or absence.
func TestApplicabilityAuthorityOptionalRuleUnderMatchingMode(t *testing.T) {
	contract := minimalChecksField(
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceOptional},
	)
	for _, present := range []bool{true, false} {
		root := map[string]any{
			"checks": []any{
				map[string]any{
					"id":   "noop",
					"mode": CheckModeRun,
				},
			},
		}
		if present {
			root["checks"].([]any)[0].(map[string]any)["argv"] = []any{"true"}
		}
		diags := ValidateModeDependentApplicability(root, contract)
		if len(diags) != 0 {
			t.Fatalf("optional rule must emit no diagnostic (present=%v); got %v", present, diags)
		}
	}
}

// TestApplicabilityAuthorityForbiddenRuleUnderMatchingMode proves
// that a PresenceForbidden rule under the matching sibling value
// emits a semantic_constraint_failed diagnostic whenever the JSON
// key is present, regardless of value.
func TestApplicabilityAuthorityForbiddenRuleUnderMatchingMode(t *testing.T) {
	contract := minimalChecksField(
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceForbidden},
	)
	root := map[string]any{
		"checks": []any{
			map[string]any{
				"id":   "noop",
				"mode": CheckModeRun,
				"argv": []any{},
			},
		},
	}
	diags := ValidateModeDependentApplicability(root, contract)
	if !hasDiagnosticAt(diags, "/checks/0/argv", PlanCodeSemanticConstraintFailed) {
		t.Fatalf("expected semantic_constraint_failed at /checks/0/argv; got %v", diags)
	}
}

// TestApplicabilityAuthorityUnmatchedMode proves that a rule
// whose Value does not equal the document's sibling value is
// inactive. The walker must emit no diagnostic.
func TestApplicabilityAuthorityUnmatchedMode(t *testing.T) {
	contract := minimalChecksField(
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
	)
	root := map[string]any{
		"checks": []any{
			map[string]any{
				"id":   "noop",
				"mode": CheckModeExclude,
			},
		},
	}
	diags := ValidateModeDependentApplicability(root, contract)
	if len(diags) != 0 {
		t.Fatalf("rule for mode=run must not fire on mode=exclude; got %v", diags)
	}
}

// TestApplicabilityAuthorityMultipleModeRules proves that a field
// carrying two ApplicabilityRule entries (one for run, one for
// exclude) activates the right branch in each case.
func TestApplicabilityAuthorityMultipleModeRules(t *testing.T) {
	contract := minimalChecksField(
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeExclude, Presence: PresenceForbidden},
	)
	// mode=run, argv missing -> required_property_missing
	runRoot := map[string]any{
		"checks": []any{
			map[string]any{"id": "noop", "mode": CheckModeRun},
		},
	}
	runDiags := ValidateModeDependentApplicability(runRoot, contract)
	if !hasDiagnosticAt(runDiags, "/checks/0/argv", PlanCodeRequiredPropertyMissing) {
		t.Fatalf("mode=run, argv missing must emit required_property_missing; got %v", runDiags)
	}
	// mode=exclude, argv present -> semantic_constraint_failed
	excludeRoot := map[string]any{
		"checks": []any{
			map[string]any{"id": "noop", "mode": CheckModeExclude, "argv": []any{}},
		},
	}
	excludeDiags := ValidateModeDependentApplicability(excludeRoot, contract)
	if !hasDiagnosticAt(excludeDiags, "/checks/0/argv", PlanCodeSemanticConstraintFailed) {
		t.Fatalf("mode=exclude, argv present must emit semantic_constraint_failed; got %v", excludeDiags)
	}
}

// TestApplicabilityAuthorityDuplicateSameModeRule proves the
// applicability walker produces a deterministic result for a
// field carrying two rules that share (Sibling="mode",
// Value="run") but conflict on presence. The walker iterates
// every matching rule and emits one diagnostic per rule whose
// presence check matches the document. The two presence checks
// (PresenceRequired and PresenceForbidden) are mutually
// exclusive on the same field, so the walker never emits
// contradictory diagnostics for the same field under one mode.
// The pinned behaviour is documented inline.
func TestApplicabilityAuthorityDuplicateSameModeRule(t *testing.T) {
	contract := minimalChecksField(
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceForbidden},
	)
	// Documented contract: the walker iterates every matching
	// rule in slice order. The presence checks are mutually
	// exclusive (Required fires only when the field is absent,
	// Forbidden fires only when the field is present), so the
	// walker emits exactly one diagnostic per document for a
	// given field under a given mode. The walker therefore never
	// produces contradictory diagnostics for the same field.

	// Field absent under mode=run: exactly one PresenceRequired diagnostic.
	absentRoot := map[string]any{
		"checks": []any{
			map[string]any{"id": "noop", "mode": CheckModeRun},
		},
	}
	absentDiags := ValidateModeDependentApplicability(absentRoot, contract)
	if len(absentDiags) != 1 {
		t.Fatalf("field absent under conflicting rules: expected exactly 1 diagnostic, got %d: %v",
			len(absentDiags), absentDiags)
	}
	if absentDiags[0].Code != PlanCodeRequiredPropertyMissing {
		t.Fatalf("field absent under conflicting rules: expected PresenceRequired diagnostic, got %v",
			absentDiags[0])
	}
	if absentDiags[0].InstancePath != "/checks/0/argv" {
		t.Fatalf("field absent under conflicting rules: expected /checks/0/argv path, got %q",
			absentDiags[0].InstancePath)
	}

	// Field present under mode=run: exactly one PresenceForbidden diagnostic.
	presentRoot := map[string]any{
		"checks": []any{
			map[string]any{"id": "noop", "mode": CheckModeRun, "argv": []any{"true"}},
		},
	}
	presentDiags := ValidateModeDependentApplicability(presentRoot, contract)
	if len(presentDiags) != 1 {
		t.Fatalf("field present under conflicting rules: expected exactly 1 diagnostic, got %d: %v",
			len(presentDiags), presentDiags)
	}
	if presentDiags[0].Code != PlanCodeSemanticConstraintFailed {
		t.Fatalf("field present under conflicting rules: expected PresenceForbidden diagnostic, got %v",
			presentDiags[0])
	}
	if presentDiags[0].InstancePath != "/checks/0/argv" {
		t.Fatalf("field present under conflicting rules: expected /checks/0/argv path, got %q",
			presentDiags[0].InstancePath)
	}
}

// TestApplicabilityAuthorityPresenceForMode proves the
// descriptor-driven helper resolves the right PresenceRule for
// known modes and defaults to PresenceOptional when the mode is
// unmatched.
func TestApplicabilityAuthorityPresenceForMode(t *testing.T) {
	field := planFieldDescriptor{
		ApplicabilityRules: []fieldApplicabilityRule{
			{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
			{Sibling: "mode", Value: CheckModeExclude, Presence: PresenceForbidden},
		},
	}
	cases := []struct {
		mode string
		want PresenceRule
	}{
		{CheckModeRun, PresenceRequired},
		{CheckModeExclude, PresenceForbidden},
		{"unknown_mode", PresenceOptional},
	}
	for _, tc := range cases {
		got := applicabilityPresenceForMode(field, tc.mode)
		if got != tc.want {
			t.Fatalf("mode=%q got %v, want %v", tc.mode, got, tc.want)
		}
	}
	// Field with no rules: default is PresenceOptional.
	empty := planFieldDescriptor{}
	if got := applicabilityPresenceForMode(empty, CheckModeRun); got != PresenceOptional {
		t.Fatalf("empty field default got %v, want PresenceOptional", got)
	}
}

// hasDiagnosticAt reports whether any diagnostic in `diags`
// targets the given instance path with the given code.
func hasDiagnosticAt(diags []PlanValidationError, path string, code PlanValidationCode) bool {
	for _, d := range diags {
		if d.InstancePath == path && d.Code == code {
			return true
		}
	}
	return false
}
