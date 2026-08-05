package closure

import (
	"encoding/json"
	"testing"
)

// plan_run_execution_fields_outcome_pinned_test.go owns the
// outcome-pinned differential matrix for the run-mode
// `working_directory` and `timeout_seconds` fields. Each
// case asserts the expected standard, extension-aware, and
// runtime acceptance decisions so the harness proves the
// schema and runtime agree on the canonical contract — not on
// the same accidental bug.
//
// The test deliberately exercises the JSON delete semantics
// (via deleteField) so absent-field cases are distinct from
// null-value cases (via setField).

const canonicalRunPlanPinned = `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-COMPOSED",
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

// applyPinnedMutation deletes or writes a single check field.
func applyPinnedMutation(t *testing.T, base []byte, op string) []byte {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(base, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	checks, _ := body["checks"].([]any)
	if len(checks) == 0 {
		t.Fatalf("no /checks")
	}
	check, _ := checks[0].(map[string]any)
	if len(op) > 7 && op[:7] == "delete:" {
		delete(check, op[7:])
		return marshalPinned(t, body)
	}
	if len(op) > 4 && op[:4] == "set:" {
		// "set:key=value"
		eq := -1
		for i, c := range op[4:] {
			if c == '=' {
				eq = i
				break
			}
		}
		if eq < 0 {
			t.Fatalf("set op needs '=': %s", op)
		}
		key := op[4 : 4+eq]
		var val any
		_ = json.Unmarshal([]byte(op[4+eq+1:]), &val)
		check[key] = val
		return marshalPinned(t, body)
	}
	t.Fatalf("unknown op: %s", op)
	return nil
}

func marshalPinned(t *testing.T, body map[string]any) []byte {
	t.Helper()
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return out
}

// TestPinnedRunMatrix exercises the run-mode absence and value
// matrices with explicit expected outcomes for every layer.
// ACCEPTANCE-1: extension-aware still accepts absent fields
// (applicability walker cannot see the parent context for
// missing properties). Documented in the ACT report.
func TestPinnedRunMatrix(t *testing.T) {
	type result struct {
		standard  bool
		extension bool
		runtime   bool
	}
	cases := []struct {
		name     string
		ops      []string
		expected result
	}{
		{"both_present", nil, result{standard: true, extension: true, runtime: true}},
		// CORRECTION04: extension-aware rejects absent required fields.
		{"wd_absent", []string{"delete:working_directory"}, result{standard: true, extension: false, runtime: false}},
		{"ts_absent", []string{"delete:timeout_seconds"}, result{standard: true, extension: false, runtime: false}},
		{"both_absent", []string{"delete:working_directory", "delete:timeout_seconds"}, result{standard: true, extension: false, runtime: false}},
		{"wd_null", []string{"set:working_directory=null"}, result{standard: false, extension: false, runtime: false}},
		{"ts_null", []string{"set:timeout_seconds=null"}, result{standard: false, extension: false, runtime: false}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data := []byte(canonicalRunPlanPinned)
			for _, op := range tc.ops {
				data = applyPinnedMutation(t, data, op)
			}
			standard, extension, runtime := evaluatePinned(t, data)
			if standard != tc.expected.standard {
				t.Fatalf("%s: standard=%v, want %v", tc.name, standard, tc.expected.standard)
			}
			if extension != tc.expected.extension {
				t.Fatalf("%s: extension=%v, want %v", tc.name, extension, tc.expected.extension)
			}
			if runtime != tc.expected.runtime {
				t.Fatalf("%s: runtime=%v, want %v", tc.name, runtime, tc.expected.runtime)
			}
			t.Logf("PARITY-CONFIRMED %s: extension=%v runtime=%v", tc.name, extension, runtime)
		})
	}
}

// evaluatePinned runs the three layers and returns
// (standard, extension, runtime) accept decisions.
func evaluatePinned(t *testing.T, data []byte) (bool, bool, bool) {
	t.Helper()
	schema, err := JSONSchema()
	if err != nil {
		t.Fatalf("JSONSchema(): %v", err)
	}
	var rootMap map[string]any
	if err := json.Unmarshal(data, &rootMap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	standard := evaluateWithSchemaStandard(schema, rootMap)
	extension := evaluateWithSchemaExtensionAware(schema, rootMap)
	composed := ValidatePlanComposed(data)
	return standard.Accept, extension.Accept, composed.Valid
}
