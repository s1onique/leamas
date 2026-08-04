package closure

import (
	"encoding/json"
	"sort"
	"testing"
)

// composedCanonicalPlanIndented returns the canonical v1 closure
// plan JSON with stable, LLM-friendly indentation. Each line is
// well under the 240-char limit so the file stays LLM-friendly.
func composedCanonicalPlanIndented() []byte {
	return []byte(`{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-CANONICAL",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid":   "2222222222222222222222222222222222222222"
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
    "require_clean_before":        true,
    "require_clean_after":         true,
    "forbid_tracked_full_digests": true,
    "require_diff_check":          true
  }
}`)
}

// composedPlanWithInvalidEnum returns a v1 plan whose check mode
// uses a value outside the closed enum set. The bounded parser
// accepts the document; structural validation rejects it with
// invalid_enum.
func composedPlanWithInvalidEnum() []byte {
	return []byte(`{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-INVALID-ENUM",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid":   "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {
      "id": "noop",
      "mode": "unknown-mode",
      "argv": ["true"],
      "working_directory": ".",
      "timeout_seconds": 60,
      "environment": {}
    }
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before":        true,
    "require_clean_after":         true,
    "forbid_tracked_full_digests": true,
    "require_diff_check":          true
  }
}`)
}

// composedPlanWithDuplicateCheckID returns a v1 plan whose checks
// array carries two items with the same id. Structural validation
// accepts the document; semantic validation rejects it.
func composedPlanWithDuplicateCheckID() []byte {
	return []byte(`{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-DUP-ID",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid":   "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "x", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}},
    {"id": "x", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before":        true,
    "require_clean_after":         true,
    "forbid_tracked_full_digests": true,
    "require_diff_check":          true
  }
}`)
}

// composedPlanWithDuplicateRootKey returns a JSON object whose
// root level carries a duplicate key. The bounded parser rejects
// the document with duplicate_property before any structural
// validation runs.
func composedPlanWithDuplicateRootKey() []byte {
	return []byte(`{
  "contract_version": 1,
  "contract_version": 1
}`)
}

// composedResultJSONKeys is the exact top-level key inventory the
// composed result must expose, in lexicographic order. The slice
// is the contract; the test must fail if a future field is added
// or removed.
var composedResultJSONKeys = []string{
	"decode_errors", "decoded", "semantic_errors", "semantic_valid",
	"structural", "valid",
}

// nestedStructuralResultJSONKeys is the exact key inventory the
// nested structural object must expose, in lexicographic order.
var nestedStructuralResultJSONKeys = []string{
	"contract_version", "errors", "valid",
}

// sortedRawKeys returns the keys of m in stable lexicographic
// order. JSON-RawMessage maps are unordered; the test needs a
// deterministic view to compare against the documented inventory.
func sortedRawKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestResultJSONTopLevelKeys asserts the composed result exposes
// the documented top-level key set exactly, with no extras and no
// missing entries, for every reachable outcome. The same test
// also pins the nested structural object's key inventory.
func TestResultJSONTopLevelKeys(t *testing.T) {
	cases := []struct {
		name string
		plan []byte
	}{
		{"success", composedCanonicalPlanIndented()},
		{"parse-failure", []byte(`not valid json`)},
		{"structural-failure", composedPlanWithInvalidEnum()},
		{"semantic-failure", composedPlanWithDuplicateCheckID()},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			composed := validatePlanComposedWithObserver(
				tc.plan, noopCompositionObserver{})
			raw, err := json.Marshal(composed)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var top map[string]json.RawMessage
			if err := json.Unmarshal(raw, &top); err != nil {
				t.Fatalf("unmarshal top: %v", err)
			}
			keys := sortedRawKeys(top)
			if len(keys) != len(composedResultJSONKeys) {
				t.Fatalf("top-level key count = %d, want %d; keys = %v",
					len(keys), len(composedResultJSONKeys), keys)
			}
			for i, k := range keys {
				if k != composedResultJSONKeys[i] {
					t.Fatalf("top-level key[%d] = %q, want %q",
						i, k, composedResultJSONKeys[i])
				}
			}
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(top["structural"], &nested); err != nil {
				t.Fatalf("unmarshal nested structural: %v", err)
			}
			nestedKeys := sortedRawKeys(nested)
			if len(nestedKeys) != len(nestedStructuralResultJSONKeys) {
				t.Fatalf("nested structural key count = %d, want %d; keys = %v",
					len(nestedKeys), len(nestedStructuralResultJSONKeys), nestedKeys)
			}
			for i, k := range nestedKeys {
				if k != nestedStructuralResultJSONKeys[i] {
					t.Fatalf("nested structural key[%d] = %q, want %q",
						i, k, nestedStructuralResultJSONKeys[i])
				}
			}
		})
	}
}

// TestResultJSONArraysNeverNull asserts every diagnostic array
// encodes as a JSON array (never null) on every reachable
// outcome. The structural case is the regression target: prior to
// the correction, Structural.Errors encoded as null on success
// because the underlying slice was nil.
func TestResultJSONArraysNeverNull(t *testing.T) {
	cases := []struct {
		name string
		plan []byte
	}{
		{"success", composedCanonicalPlanIndented()},
		{"parse-failure", []byte(`not valid json`)},
		{"structural-failure", composedPlanWithInvalidEnum()},
		{"semantic-failure", composedPlanWithDuplicateCheckID()},
	}
	nullCount := 0
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			composed := validatePlanComposedWithObserver(
				tc.plan, noopCompositionObserver{})
			raw, err := json.Marshal(composed)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var top map[string]json.RawMessage
			if err := json.Unmarshal(raw, &top); err != nil {
				t.Fatalf("unmarshal top: %v", err)
			}
			if string(top["decode_errors"]) == "null" {
				nullCount++
				t.Fatalf("decode_errors encoded as null")
			}
			if string(top["semantic_errors"]) == "null" {
				nullCount++
				t.Fatalf("semantic_errors encoded as null")
			}
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(top["structural"], &nested); err != nil {
				t.Fatalf("unmarshal nested structural: %v", err)
			}
			if string(nested["errors"]) == "null" {
				nullCount++
				t.Fatalf("structural.errors encoded as null")
			}
		})
	}
	if nullCount != 0 {
		t.Fatalf("NULL_DIAGNOSTIC_ARRAY_COUNT = %d, want 0", nullCount)
	}
}

// TestResultJSONSuccessExactEmptyArrays pins the exact "[]"
// encoding for every diagnostic array on the canonical success
// case. This is the strongest contract test: any future change
// that flips a non-nil-but-empty slice to nil, or returns a JSON
// object instead of an array, fails this test.
func TestResultJSONSuccessExactEmptyArrays(t *testing.T) {
	composed := validatePlanComposedWithObserver(
		composedCanonicalPlanIndented(), noopCompositionObserver{})
	if !composed.Valid {
		t.Fatalf("canonical plan must compose-validate: %v",
			composed.Structural.Errors)
	}
	raw, err := json.Marshal(composed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("unmarshal top: %v", err)
	}
	if string(top["decode_errors"]) != "[]" {
		t.Fatalf("decode_errors = %s, want []", top["decode_errors"])
	}
	if string(top["semantic_errors"]) != "[]" {
		t.Fatalf("semantic_errors = %s, want []",
			top["semantic_errors"])
	}
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(top["structural"], &nested); err != nil {
		t.Fatalf("unmarshal nested structural: %v", err)
	}
	if string(nested["errors"]) != "[]" {
		t.Fatalf("structural.errors = %s, want []", nested["errors"])
	}
}
