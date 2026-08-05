package closure

import (
	"encoding/json"
	"fmt"
	"math"

	"testing"
)

// plan_run_execution_fields_differential_test.go owns the
// generated-schema differential harness for the Closure Protocol
// v1 run-mode working_directory and timeout_seconds fields.
//
// The harness drives every decision from JSONSchema() output.
// No field-specific handwritten expectation function
// determines schema acceptance; the harness builds the
// generic evaluator by consuming the schema document and the
// supplied extension values.
//
// Three independent outcomes are recorded per matrix case:
//
//   - STANDARD_SCHEMA_RESULT: portable JSON Schema subset only
//     (type/required/properties/minLength/pattern/minimum/
//     maximum/items/additionalProperties/const/enum); Leamas
//     extensions are ignored.
//   - EXTENSION_AWARE_SCHEMA_RESULT: standard subset PLUS the
//     x-leamas-repository-relative-path and x-applicability
//     extensions interpreted from the emitted schema.
//   - RUNTIME_RESULT: the Leamas composed validator's accept
//     decision.
//
// Acceptance requires EXTENSION_AWARE_SCHEMA_RESULT ==
// RUNTIME_RESULT for every matrix case. STANDARD_SCHEMA_RESULT
// may differ (and is reported separately) when the runtime
// enforces constraints represented only by Leamas extensions.

// canonicalRunPlan is the canonical run-mode plan fixture used
// by the differential matrix.
const canonicalRunPlan = `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-DIFF-RUN-PARITY",
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

// buildCheckInstance is a small helper that mutates the check
// item at index 0 via the supplied function and returns the
// marshalled bytes.
func buildCheckInstance(t *testing.T, base []byte, mutate func(c map[string]any)) []byte {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(base, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	checks, _ := body["checks"].([]any)
	if len(checks) == 0 {
		t.Fatalf("fixture has no /checks")
	}
	check, _ := checks[0].(map[string]any)
	mutate(check)
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return out
}

// runEvalDiff composes the three outcomes for the supplied
// instance bytes.
type runEvalDiff struct {
	Standard       schemaEvaluation
	ExtensionAware schemaEvaluation
	Runtime        bool
	RuntimeIssues  []string
}

// evaluateRunDiff builds the schema-driven + runtime outcomes
// for the supplied instance.
func evaluateRunDiff(t *testing.T, instance []byte) runEvalDiff {
	t.Helper()
	schema, err := JSONSchema()
	if err != nil {
		t.Fatalf("JSONSchema() error = %v", err)
	}
	var rootMap map[string]any
	if err := json.Unmarshal(instance, &rootMap); err != nil {
		t.Fatalf("unmarshal instance: %v", err)
	}
	t.Logf("instance: %s", string(instance))
	standard := evaluateWithSchemaStandard(schema, rootMap)
	extension := evaluateWithSchemaExtensionAware(schema, rootMap)
	composed := ValidatePlanComposed(instance)
	var issues []string
	for _, d := range composed.Structural.Errors {
		issues = append(issues, fmt.Sprintf("%s/%s", d.Code, d.Keyword))
	}
	for _, d := range composed.SemanticErrors {
		issues = append(issues, fmt.Sprintf("%s/%s", d.Code, d.Keyword))
	}
	return runEvalDiff{
		Standard:       standard,
		ExtensionAware: extension,
		Runtime:        composed.Valid,
		RuntimeIssues:  issues,
	}
}

// TestDifferentialWorkingDirectoryMatrix exercises the complete
// working-directory language. For every case the test asserts
// the extension-aware outcome equals the runtime outcome.
func TestDifferentialWorkingDirectoryMatrix(t *testing.T) {
	type matrix struct {
		name   string
		value  string
		absent bool
	}
	cases := []matrix{
		{"dot", ".", false},
		{"internal_foo", "internal/foo", false},
		{"empty_string", "", false},
		{"absolute", "/absolute/path", false},
		{"leading_double_slash", "//server/share", false},
		{"parent_escape", "../escape", false},
		{"nested_parent_traversal", "foo/../bar", false},
		{"dot_slash_foo", "./foo", false},
		{"foo_dot_bar", "foo/./bar", false},
		{"double_slash", "foo//bar", false},
		{"trailing_slash", "foo/", false},
		{"backslash_foo", "foo\\bar", false},
		{"backslash_dotdot", "..\\escape", false},
		{"windows_volume_backslash", "C:\\escape", false},
		{"windows_volume_forward", "C:/escape", false},
		{"unc_backslash", "\\\\server\\share", false},
		{"nul_byte", "foo\u0000bar", false},
		{"newline", "foo\nbar", false},
		{"unicode_char", "é", false},
		{"unicode_segment", "目录", false},
		{"unicode_path", "foo/目录", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data := buildCheckInstance(t, []byte(canonicalRunPlan), func(c map[string]any) {
				if tc.absent {
					delete(c, "working_directory")
				} else {
					c["working_directory"] = tc.value
				}
			})
			ev := evaluateRunDiff(t, data)
			if ev.ExtensionAware.Accept != ev.Runtime {
				t.Fatalf("%s: extension_aware=%v runtime=%v issues=%v",
					tc.name, ev.ExtensionAware.Accept, ev.Runtime, ev.RuntimeIssues)
			}
			t.Logf("%s: standard=%v extension_aware=%v runtime=%v",
				tc.name, ev.Standard.Accept, ev.ExtensionAware.Accept, ev.Runtime)
		})
	}
}

// TestDifferentialTimeoutMatrix exercises the timeout_seconds
// value matrix with the same acceptance contract.
func TestDifferentialTimeoutMatrix(t *testing.T) {
	type matrix struct {
		name   string
		value  any
		absent bool
		null   bool
	}
	beyondNative := math.MaxInt64
	cases := []matrix{
		{"null", nil, false, true},
		{"neg_one", -1, false, false},
		{"zero", 0, false, false},
		{"one", 1, false, false},
		{"six_hundred", 600, false, false},
		{"six_hundred_one", 601, false, false},
		{"string_sixty", "60", false, false},
		{"float_sixty", 60.0, false, false},
		{"float_sixty_point_five", 60.5, false, false},
		{"beyond_native_int", beyondNative, false, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data := buildCheckInstance(t, []byte(canonicalRunPlan), func(c map[string]any) {
				if tc.absent {
					delete(c, "timeout_seconds")
				} else if tc.null {
					c["timeout_seconds"] = nil
				} else {
					c["timeout_seconds"] = tc.value
				}
			})
			ev := evaluateRunDiff(t, data)
			if ev.ExtensionAware.Accept != ev.Runtime {
				t.Fatalf("%s: extension_aware=%v runtime=%v issues=%v",
					tc.name, ev.ExtensionAware.Accept, ev.Runtime, ev.RuntimeIssues)
			}
			t.Logf("%s: standard=%v extension_aware=%v runtime=%v",
				tc.name, ev.Standard.Accept, ev.ExtensionAware.Accept, ev.Runtime)
		})
	}
}

// TestDifferentialApplicabilityMatrix exercises mode-dependent
// presence for both run-mode fields. Forbidden fields must
// report forbidden_presence exclusively.
func TestDifferentialApplicabilityMatrix(t *testing.T) {
	cases := []struct {
		name       string
		workingDir any
		timeoutSec any
		mode       string
	}{
		{"run_absent", nil, nil, "run"},
		{"run_valid", ".", 60, "run"},
		{"exclude_valid_present", ".", 60, "exclude"},
		{"exclude_invalid_string", "", 60, "exclude"},
		{"exclude_parent_escape", "../escape", 60, "exclude"},
		{"exclude_wrong_type_working_directory", 42, 60, "exclude"},
		{"exclude_invalid_timeout", ".", 0, "exclude"},
		{"exclude_oob_timeout", ".", 601, "exclude"},
		{"exclude_null_working_directory", nil, 60, "exclude"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data := buildCheckInstance(t, []byte(canonicalRunPlan), func(c map[string]any) {
				c["mode"] = tc.mode
				if tc.mode == "exclude" {
					c["reason"] = "noop"
				}
				if tc.workingDir != nil {
					c["working_directory"] = tc.workingDir
				} else if tc.name == "exclude_null_working_directory" {
					c["working_directory"] = nil
				}
				if tc.timeoutSec != nil {
					c["timeout_seconds"] = tc.timeoutSec
				}
			})
			composed := ValidatePlanComposed(data)
			if tc.mode == "exclude" {
				if composed.Valid {
					t.Fatalf("%s: exclude mode should be invalid", tc.name)
				}
				wdForbidden := false
				tsForbidden := false
				for _, d := range composed.Structural.Errors {
					if d.Code != PlanCodeForbiddenPresence {
						t.Fatalf("%s: forbidden field %s classified as %s/%s (must be forbidden_presence); issues=%+v",
							tc.name, d.PropertyName, d.Code, d.Keyword, composed.Structural.Errors)
					}
					switch d.PropertyName {
					case "working_directory":
						wdForbidden = true
					case "timeout_seconds":
						tsForbidden = true
					}
				}
				if tc.workingDir != nil && !wdForbidden {
					t.Fatalf("%s: working_directory forbidden presence expected", tc.name)
				}
				if tc.timeoutSec != nil && !tsForbidden {
					t.Fatalf("%s: timeout_seconds forbidden presence expected", tc.name)
				}
			}
		})
	}
}
