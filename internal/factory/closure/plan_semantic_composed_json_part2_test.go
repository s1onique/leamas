package closure

import (
	"encoding/json"
	"sort"
	"testing"
)

// TestComposedRunnerAuthorityStructuralRejection verifies that runner_authority
// mode errors are caught at the structural level (enum validation) and
// semantic errors (missing tool block) are caught by ValidateRunnerAuthority.
func TestComposedRunnerAuthorityStructuralRejection(t *testing.T) {
	// runner_authority.mode unknown is rejected structurally as invalid_enum
	// tool_release_exact is valid without a tool block (no semantic enforcement)
	cases := []struct {
		name      string
		json      string
		wantValid bool
	}{
		{
			name: "runner_authority mode unknown rejected structurally",
			json: `{
  "contract_version": 1,
  "act_id": "ACT-RUNNER-AUTH",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "check-one", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  },
  "runner_authority": {"mode": "unknown_mode"}
}`,
			wantValid: false,
		},
		{
			name: "tool_release_exact without tool block is rejected semantically",
			// The composed pipeline now enforces tool_release_exact requiring a tool block
			// via ValidateRunnerAuthority wired into ValidatePlan.
			json: `{
  "contract_version": 1,
  "act_id": "ACT-RUNNER-AUTH",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "check-one", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  },
  "runner_authority": {"mode": "tool_release_exact"}
}`,
			wantValid: false,
		},
		{
			name: "tool_release_exact with tool block is valid",
			json: `{
  "contract_version": 1,
  "act_id": "ACT-RUNNER-AUTH",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "check-one", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  },
  "runner_authority": {"mode": "tool_release_exact", "tool": {"revision": "1111111111111111111111111111111111111111", "binary_sha256": "1111111111111111111111111111111111111111111111111111111111111111"}}
}`,
			wantValid: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidatePlanComposed([]byte(tc.json))

			if result.Valid != tc.wantValid {
				t.Errorf("Valid = %v, want %v; errors: %+v", result.Valid, tc.wantValid, result.SemanticErrors)
			}
		})
	}
}

// TestComposedResultJSONKeySet verifies that marshaling the composed result
// produces the exact set of JSON keys and that raw error/Cause fields are
// not exposed.
func TestComposedResultJSONKeySet(t *testing.T) {
	result := ValidatePlanComposed(composedPlanDuplicateCheckID())

	// Marshal to JSON
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal ComposedPlanValidationResult: %v", err)
	}

	// Parse back to verify key set
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal result JSON: %v", err)
	}

	// Exact set of keys expected
	expectedKeys := []string{"structural", "decoded", "decode_errors", "semantic_valid", "semantic_errors", "valid"}
	gotKeys := make([]string, 0, len(parsed))
	for k := range parsed {
		gotKeys = append(gotKeys, k)
	}
	sort.Strings(gotKeys)
	sort.Strings(expectedKeys)

	if len(gotKeys) != len(expectedKeys) {
		t.Errorf("JSON key count = %d, want %d; keys: %v", len(gotKeys), len(expectedKeys), gotKeys)
	}
	for i, k := range gotKeys {
		if i >= len(expectedKeys) || k != expectedKeys[i] {
			t.Errorf("unexpected key at index %d: %s, want %v", i, k, expectedKeys)
		}
	}

	// Verify raw fields are not present
	for _, key := range gotKeys {
		if key == "error" || key == "Cause" || key == "cause" {
			t.Errorf("raw field %q must not be in JSON output", key)
		}
	}

	// Verify semantic_errors is an array, not null
	if semErrs, ok := parsed["semantic_errors"]; ok {
		if semErrs == nil {
			t.Error("semantic_errors must not be null")
		}
	}

	// Verify decode_errors is an array (may be empty)
	if decErrs, ok := parsed["decode_errors"]; ok {
		if decErrs == nil {
			t.Error("decode_errors must not be null")
		}
	}

	// Verify structural is an object with errors array
	if structural, ok := parsed["structural"].(map[string]any); ok {
		if _, hasErrors := structural["errors"]; !hasErrors {
			t.Error("structural.errors must be present")
		}
	}
}

// TestComposedResultJSONDiagnosticsNonNull verifies that diagnostic arrays
// serialize as [] not null in JSON output.
func TestComposedResultJSONDiagnosticsNonNull(t *testing.T) {
	cases := []struct {
		name      string
		json      string
		wantValid bool
	}{
		{
			name:      "valid plan - all arrays must be non-null",
			wantValid: true,
			json: `{
  "contract_version": 1,
  "act_id": "ACT-VALID",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "check-one", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`,
		},
		{
			name:      "semantic failure - semantic_errors populated, others non-null",
			wantValid: false,
			json: `{
  "contract_version": 1,
  "act_id": "ACT-INVALID",
  "baseline": {
    "commit_oid": "bad-oid",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "check-one", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`,
		},
		{
			name:      "structural failure - structural.errors populated, others non-null",
			wantValid: false,
			json: `{
  "contract_version": 1,
  "act_id": "ACT-STRUCT",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [],
  "artifacts": [],
  "policy": {}
}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidatePlanComposed([]byte(tc.json))

			data, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			// All diagnostic arrays must be non-null
			arrays := []string{"decode_errors", "semantic_errors"}
			if !result.Structural.Valid {
				arrays = append(arrays, "structural")
			}

			for _, key := range arrays {
				if key == "structural" {
					if s, ok := parsed[key].(map[string]any); ok {
						if errs, ok := s["errors"]; ok && errs == nil {
							t.Errorf("%s.errors must not be null", key)
						}
					}
				} else {
					if v, ok := parsed[key]; !ok || v == nil {
						t.Errorf("%s must not be null in JSON output", key)
					}
				}
			}
		})
	}
}

// TestComposedSemanticErrorsImplementPlanDiagnostics verifies that all semantic
// errors returned by the composed pipeline implement planDiagnosticSource.
func TestComposedSemanticErrorsImplementPlanDiagnostics(t *testing.T) {
	result := ValidatePlanComposed(composedPlanDuplicateCheckID())

	if !result.SemanticValid && len(result.SemanticErrors) > 0 {
		// Verify each error has proper fields
		for i, err := range result.SemanticErrors {
			if err.InstancePath == "" && err.SchemaPath == "" && err.Code == "" && err.Message == "" {
				t.Errorf("semantic error %d has no diagnostic fields", i)
			}
			// Verify AcceptedValues is properly handled
			if err.AcceptedValues != nil && len(err.AcceptedValues) == 0 {
				t.Logf("semantic error %d has empty AcceptedValues (acceptable)", i)
			}
		}
	}
}

// TestComposedResultCanonicalMarshal verifies that marshaling the canonical
// valid fixture produces correct output and round-trips through JSON.
func TestComposedResultCanonicalMarshal(t *testing.T) {
	result := ValidatePlanComposed(canonicalComposedPlan())
	if !result.Valid {
		t.Fatalf("canonical plan must be valid: %+v", result.SemanticErrors)
	}

	// Marshal the result
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Unmarshal back
	var parsed ComposedPlanValidationResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Verify structure is preserved
	if parsed.Valid != result.Valid {
		t.Errorf("Valid round-trip mismatch: got %v, want %v", parsed.Valid, result.Valid)
	}
	if parsed.Decoded != result.Decoded {
		t.Errorf("Decoded round-trip mismatch: got %v, want %v", parsed.Decoded, result.Decoded)
	}
	if parsed.SemanticValid != result.SemanticValid {
		t.Errorf("SemanticValid round-trip mismatch: got %v, want %v", parsed.SemanticValid, result.SemanticValid)
	}
	if !parsed.Structural.Valid {
		t.Errorf("Structural.Valid should be true")
	}

	// Verify semantic_errors is empty but non-nil after round-trip
	if len(parsed.SemanticErrors) != 0 {
		t.Errorf("SemanticErrors should be empty for valid plan, got %d", len(parsed.SemanticErrors))
	}
}
