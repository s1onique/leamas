package closure

import (
	"strings"
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

// TestApplicabilityRuleIdentityDuplicateConflictingRules proves
// the descriptor identity validator rejects a field that
// carries two rules sharing (Sibling="mode", Value="run") but
// with conflicting presence values. The validator emits a
// duplicate_applicability_rule diagnostic before the walker
// ever inspects the document, and the walker skips per-check
// processing for the offending field.
func TestApplicabilityRuleIdentityDuplicateConflictingRules(t *testing.T) {
	contract := minimalChecksField(
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceForbidden},
	)
	diags := validateDescriptorApplicabilityIdentity(contract)
	found := false
	for _, d := range diags {
		if d.Code == PlanCodeDuplicateApplicabilityRule && d.PropertyName == "argv" {
			found = true
			if d.InstancePath != "/checks/argv" {
				t.Fatalf("duplicate diagnostic path = %q, want %q", d.InstancePath, "/checks/argv")
			}
			if !strings.Contains(d.Message, "Sibling=\"mode\"") || !strings.Contains(d.Message, "Value=\"run\"") {
				t.Fatalf("duplicate diagnostic message must identify (Sibling, Value): %q", d.Message)
			}
		}
	}
	if !found {
		t.Fatalf("expected duplicate_applicability_rule for argv; got %v", diags)
	}
	// The walker must NOT emit any per-check diagnostic for
	// argv; the field is closed-path until the descriptor is
	// repaired.
	absentRoot := map[string]any{
		"checks": []any{
			map[string]any{"id": "noop", "mode": CheckModeRun},
		},
	}
	walk := ValidateModeDependentApplicability(absentRoot, contract)
	for _, d := range walk {
		if d.Code == PlanCodeRequiredPropertyMissing || d.Code == PlanCodeSemanticConstraintFailed {
			t.Fatalf("walker must skip argv after duplicate-rule diagnostic; got %v", d)
		}
	}
}

// TestApplicabilityRuleIdentityDuplicateIdenticalRules proves
// the descriptor identity validator also rejects two rules that
// share (Sibling, Value, Presence) verbatim. Identical
// duplicates would emit two copies of the same diagnostic and
// must be rejected at descriptor time.
func TestApplicabilityRuleIdentityDuplicateIdenticalRules(t *testing.T) {
	contract := minimalChecksField(
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
	)
	diags := validateDescriptorApplicabilityIdentity(contract)
	found := false
	for _, d := range diags {
		if d.Code == PlanCodeDuplicateApplicabilityRule && d.PropertyName == "argv" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected duplicate_applicability_rule for identical duplicates; got %v", diags)
	}
}

// TestApplicabilityRuleIdentityDistinctRulesAccepted proves
// the production-shaped "one rule per (Sibling, Value)" pattern
// passes validation. Two distinct (Sibling="mode") rules with
// different Value strings (run vs exclude) are unique.
func TestApplicabilityRuleIdentityDistinctRulesAccepted(t *testing.T) {
	contract := minimalChecksField(
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
		fieldApplicabilityRule{Sibling: "mode", Value: CheckModeExclude, Presence: PresenceForbidden},
	)
	diags := validateDescriptorApplicabilityIdentity(contract)
	if len(diags) != 0 {
		t.Fatalf("distinct (Sibling, Value) rules must pass identity validation; got %v", diags)
	}
	// Run-mode document, argv missing: walker still emits the
	// required_property_missing diagnostic.
	runRoot := map[string]any{
		"checks": []any{
			map[string]any{"id": "noop", "mode": CheckModeRun},
		},
	}
	runDiags := ValidateModeDependentApplicability(runRoot, contract)
	if !hasDiagnosticAt(runDiags, "/checks/0/argv", PlanCodeRequiredPropertyMissing) {
		t.Fatalf("mode=run, argv missing must emit required_property_missing; got %v", runDiags)
	}
	// Exclude-mode document, argv present: walker emits the
	// semantic_constraint_failed diagnostic.
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

// TestApplicabilityRuleIdentityProductionDescriptorAccepted
// proves the production v1 descriptor inventory passes identity
// validation unchanged. The fixture pins the production contract
// so a future descriptor rewrite cannot silently introduce
// duplicate (Sibling, Value) rules.
func TestApplicabilityRuleIdentityProductionDescriptorAccepted(t *testing.T) {
	diags := validateDescriptorApplicabilityIdentity(planContractV1())
	for _, d := range diags {
		if d.Code == PlanCodeDuplicateApplicabilityRule {
			t.Fatalf("production descriptor must not carry duplicate applicability rules: %v", d)
		}
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
