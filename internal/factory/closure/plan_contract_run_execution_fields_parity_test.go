package closure

import (
	"encoding/json"

	"testing"
)

// plan_contract_run_execution_fields_parity_test.go centralises
// the parity tests that recover the public contract for the
// Closure Protocol v1 run-mode `working_directory` and
// `timeout_seconds` fields. The tests prove the public schema,
// the structural validator, and the semantic validator agree on:
//
//   - presence: required for run mode, forbidden for exclude mode
//   - working-directory value rules: non-empty, non-absolute,
//     lexically clean
//   - timeout-seconds value rules: integer in [1, 600]
//   - diagnostic shape: stable eight-key public surface
//   - example and Mac-shaped regression: each yields the
//     documented outcome.

// runExecutionFieldsParityFixture is the canonical run-mode plan
// the parity suite mutates to exercise individual cases. The
// fixture includes every required field so the structural walker
// passes before the per-field mutations land.
const runExecutionFieldsParityFixture = `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-RUN-EXECUTION-FIELDS-PARITY",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {
      "id": "bounded_smoke",
      "mode": "run",
      "argv": ["sh", "-c", "exit 0"],
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

// runExecutionFieldsExcludeFixture is the canonical exclude-mode
// plan used to prove these fields are forbidden when sibling mode
// equals "exclude".
const runExecutionFieldsExcludeFixture = `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-RUN-EXECUTION-FIELDS-PARITY-EXCLUDE",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "bounded_smoke", "mode": "exclude", "reason": "noop"}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

// applyRunExecutionParityMutations returns a run-mode plan body
// with the supplied field mutation applied to the only check
// item.
func applyRunExecutionParityMutations(t *testing.T, mut func(map[string]any)) []byte {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal([]byte(runExecutionFieldsParityFixture), &body); err != nil {
		t.Fatalf("unmarshal parity fixture: %v", err)
	}
	checks := body["checks"].([]any)
	check := checks[0].(map[string]any)
	mut(check)
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal parity fixture: %v", err)
	}
	return out
}

// findDiagnosticAt reports the first diagnostic whose
// InstancePath matches the supplied path; returns nil if absent.
func findDiagnosticAt(diags []PlanValidationError, path string) *PlanValidationError {
	for i := range diags {
		if diags[i].InstancePath == path {
			return &diags[i]
		}
	}
	return nil
}

// TestRunExecutionFieldsDescriptorRules pins the contract that
// the v1 descriptor declares the run-mode PresenceRequired and
// exclude-mode PresenceForbidden rules for both
// `working_directory` and `timeout_seconds`. The descriptor is the
// source of truth; the runtime must agree with the descriptor.
func TestRunExecutionFieldsDescriptorRules(t *testing.T) {
	contract := planContractV1()
	checks := contract.Root.Fields["checks"].ItemDescriptor.Children
	for _, fieldName := range []string{"working_directory", "timeout_seconds"} {
		field, ok := checks.Fields[fieldName]
		if !ok {
			t.Fatalf("descriptor missing /checks[].%s", fieldName)
		}
		// Required: false at the structural level: the mode-
		// dependent applicability rules encode the presence
		// semantics so exclude-mode checks can omit the field
		// without triggering structural false positives.
		if field.Required {
			t.Fatalf("%s: Required must stay false at structural level", fieldName)
		}
		var sawRun, sawExclude bool
		for _, rule := range field.ApplicabilityRules {
			if rule.Sibling != "mode" {
				continue
			}
			switch rule.Value {
			case CheckModeRun:
				if rule.Presence != PresenceRequired {
					t.Fatalf("%s run-mode rule must be PresenceRequired, got %v", fieldName, rule.Presence)
				}
				sawRun = true
			case CheckModeExclude:
				if rule.Presence != PresenceForbidden {
					t.Fatalf("%s exclude-mode rule must be PresenceForbidden, got %v", fieldName, rule.Presence)
				}
				sawExclude = true
			}
		}
		if !sawRun || !sawExclude {
			t.Fatalf("%s applicability rules incomplete: run=%v exclude=%v", fieldName, sawRun, sawExclude)
		}
	}
}

// TestRunExecutionFieldsOmissionRequired proves the absence of
// either field under mode=run is rejected structurally with a
// required_property_missing diagnostic whose keyword is "required"
// and whose property_name is the missing field.
func TestRunExecutionFieldsOmissionRequired(t *testing.T) {
	for _, fieldName := range []string{"working_directory", "timeout_seconds"} {
		data := applyRunExecutionParityMutations(t, func(check map[string]any) {
			delete(check, fieldName)
		})
		result := ValidatePlanStructural(data)
		if result.Valid {
			t.Fatalf("omitting %s must fail structural validation; got valid", fieldName)
		}
		diag := findDiagnosticAt(result.Errors, "/checks/0/"+fieldName)
		if diag == nil {
			t.Fatalf("expected diagnostic at /checks/0/%s; got %+v", fieldName, result.Errors)
		}
		if diag.Code != PlanCodeRequiredPropertyMissing {
			t.Fatalf("%s omission code = %q, want required_property_missing", fieldName, diag.Code)
		}
		if diag.Keyword != KeywordRequired {
			t.Fatalf("%s omission keyword = %q, want required", fieldName, diag.Keyword)
		}
		if diag.PropertyName != fieldName {
			t.Fatalf("%s omission property_name = %q, want %q", fieldName, diag.PropertyName, fieldName)
		}
		if diag.RejectedValue != nil {
			t.Fatalf("%s omission rejected_value = %v, want null", fieldName, diag.RejectedValue)
		}
	}
}

// TestRunExecutionFieldsCanonicalValuesAccepted proves the
// canonical run-mode fixture passes the composed validator and
// each field's canonical value is accepted.
func TestRunExecutionFieldsCanonicalValuesAccepted(t *testing.T) {
	data := []byte(runExecutionFieldsParityFixture)
	result := ValidatePlanComposed(data)
	if !result.Valid {
		t.Fatalf("canonical run-mode fixture must be valid: %+v", result)
	}
}

// TestRunExecutionFieldsExcludeModePresenceRejected proves an
// exclude-mode check that still carries `working_directory` or
// `timeout_seconds` is rejected by the structural walker. The
// diagnostic classification depends on the value: a key that is
// present (regardless of value) is rejected either by the value
// constraints (minLength/pattern/minimum) when the supplied value
// fails them, or by the applicability walker (semantic_constraint_failed)
// when the value passes the constraints but the field is
// forbidden. Both outcomes are accepted by this test.
func TestRunExecutionFieldsExcludeModePresenceRejected(t *testing.T) {
	for _, fieldName := range []string{"working_directory", "timeout_seconds"} {
		var body map[string]any
		if err := json.Unmarshal([]byte(runExecutionFieldsExcludeFixture), &body); err != nil {
			t.Fatalf("unmarshal exclude fixture: %v", err)
		}
		check := body["checks"].([]any)[0].(map[string]any)
		switch fieldName {
		case "working_directory":
			check["working_directory"] = "."
		case "timeout_seconds":
			check["timeout_seconds"] = 60
		}
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal exclude fixture: %v", err)
		}
		result := ValidatePlanStructural(data)
		if result.Valid {
			t.Fatalf("exclude-mode presence of %s must fail structural validation", fieldName)
		}
		diag := findDiagnosticAt(result.Errors, "/checks/0/"+fieldName)
		if diag == nil {
			t.Fatalf("expected diagnostic at /checks/0/%s; got %+v", fieldName, result.Errors)
		}
		// Both InvalidType (structural value/pattern/minimum) and
		// SemanticConstraintFailed (applicability walker) are
		// acceptable outcomes; the diagnostic path and
		// property_name are the stable contract.
		if diag.PropertyName != fieldName {
			t.Fatalf("%s exclude presence property_name = %q, want %q", fieldName, diag.PropertyName, fieldName)
		}
	}
}

// TestRunExecutionFieldsWorkingDirectoryMatrix exercises the full
// working-directory value matrix. Each case asserts the exact
// diagnostic classification, instance path, and stable property
// name. Cases that pass the structural walker but fail the
// semantic validator are checked via ValidatePlan.
func TestRunExecutionFieldsWorkingDirectoryMatrix(t *testing.T) {
	type want struct {
		structuralValid bool
		semanticValid   bool
		keyword         PlanValidationKeyword
		propertyName    string
	}
	cases := []struct {
		name      string
		value     any
		wantValid bool // composed-level validity
		want      want
	}{
		{
			name:      "dot",
			value:     ".",
			wantValid: true,
			want:      want{structuralValid: true, semanticValid: true, propertyName: "working_directory"},
		},
		{
			name:      "valid_relative",
			value:     "internal/foo",
			wantValid: true,
			want:      want{structuralValid: true, semanticValid: true, propertyName: "working_directory"},
		},
		{
			name:      "empty_string",
			value:     "",
			wantValid: false,
			want:      want{structuralValid: false, semanticValid: false, keyword: KeywordMinLength, propertyName: "working_directory"},
		},
		{
			name:      "absolute_path",
			value:     "/absolute/path",
			wantValid: false,
			want:      want{structuralValid: false, semanticValid: false, keyword: KeywordPattern, propertyName: "working_directory"},
		},
		{
			name:      "parent_escape",
			value:     "../escape",
			wantValid: false,
			want:      want{structuralValid: true, semanticValid: false, keyword: KeywordType, propertyName: "working_directory"},
		},
		{
			name:      "nested_parent_traversal",
			value:     "foo/../../escape",
			wantValid: false,
			want:      want{structuralValid: true, semanticValid: false, keyword: KeywordType, propertyName: "working_directory"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data := applyRunExecutionParityMutations(t, func(check map[string]any) {
				if tc.value == nil {
					delete(check, "working_directory")
				} else {
					check["working_directory"] = tc.value
				}
			})
			structResult := ValidatePlanStructural(data)
			if structResult.Valid != tc.want.structuralValid {
				t.Fatalf("%s: structural.Valid = %v, want %v (errors=%+v)",
					tc.name, structResult.Valid, tc.want.structuralValid, structResult.Errors)
			}
			composed := ValidatePlanComposed(data)
			if composed.Valid != tc.wantValid {
				t.Fatalf("%s: composed.Valid = %v, want %v (semantic=%+v)",
					tc.name, composed.Valid, tc.wantValid, composed.SemanticErrors)
			}
			if tc.want.structuralValid {
				return
			}
			diag := findDiagnosticAt(structResult.Errors, "/checks/0/working_directory")
			if diag == nil {
				t.Fatalf("%s: expected structural diagnostic at /checks/0/working_directory; got %+v",
					tc.name, structResult.Errors)
			}
			if diag.Keyword != tc.want.keyword {
				t.Fatalf("%s: keyword = %q, want %q (errors=%+v)",
					tc.name, diag.Keyword, tc.want.keyword, structResult.Errors)
			}
			if diag.PropertyName != tc.want.propertyName {
				t.Fatalf("%s: property_name = %q, want %q",
					tc.name, diag.PropertyName, tc.want.propertyName)
			}
		})
	}
}

// TestRunExecutionFieldsTimeoutMatrix exercises the full
// timeout-seconds value matrix. Each case asserts the structural
// keyword and the property name.
