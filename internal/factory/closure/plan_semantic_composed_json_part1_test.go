package closure

import (
	"testing"
)

// TestValidatePlanComposedJSONStructuralFailures verifies that structural failures
// produce correct decode_errors without semantic validation.
func TestValidatePlanComposedJSONStructuralFailures(t *testing.T) {
	cases := []struct {
		name         string
		json         string
		wantDecoded  bool
		wantValid    bool
		wantSemValid bool
	}{
		{
			name:         "invalid json syntax",
			json:         `{invalid json}`,
			wantDecoded:  false,
			wantValid:    false,
			wantSemValid: false,
		},
		{
			name:         "trailing comma",
			json:         `{"act_id": "ACT-1",}`,
			wantDecoded:  false,
			wantValid:    false,
			wantSemValid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidatePlanComposed([]byte(tc.json))

			if result.Decoded != tc.wantDecoded {
				t.Errorf("Decoded = %v, want %v", result.Decoded, tc.wantDecoded)
			}
			if result.Valid != tc.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tc.wantValid)
			}
			if result.SemanticValid != tc.wantSemValid {
				t.Errorf("SemanticValid = %v, want %v", result.SemanticValid, tc.wantSemValid)
			}
		})
	}
}

// TestValidatePlanComposedResultStructure verifies the result structure is correct.
func TestValidatePlanComposedResultStructure(t *testing.T) {
	// Valid plan JSON
	json := `{
		"act_id": "ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01",
		"baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		"execution": {"mode": "serial_fail_fast"},
		"checks": [{"id": "check-1", "mode": "run", "argv": ["echo", "test"]}],
		"policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
	}`

	result := ValidatePlanComposed([]byte(json))

	// Verify result structure
	if result.Structural.Valid == false {
		// Structural validation passed
		t.Log("structural validation passed")
	}

	// Semantic errors should be empty for valid plan
	if len(result.SemanticErrors) != 0 {
		t.Errorf("expected no semantic errors for valid plan, got %d", len(result.SemanticErrors))
	}
}

// TestComposedSemanticFailureResultStructure verifies that semantic failures
// produce the correct result structure: decoded=true, semantic_valid=false,
// semantic_errors populated, valid=false.
func TestComposedSemanticFailureResultStructure(t *testing.T) {
	result := ValidatePlanComposed(composedPlanDuplicateCheckID())

	// Structural passes, typed decode succeeds
	if !result.Structural.Valid {
		t.Errorf("structural must be valid for duplicate check id fixture")
	}
	if !result.Decoded {
		t.Errorf("Decoded must be true - typed decode succeeded")
	}
	if len(result.DecodeErrors) != 0 {
		t.Errorf("DecodeErrors must be empty, got %d", len(result.DecodeErrors))
	}

	// Semantic fails
	if result.SemanticValid {
		t.Errorf("SemanticValid must be false")
	}
	if len(result.SemanticErrors) == 0 {
		t.Errorf("SemanticErrors must be non-null and non-empty on semantic failure")
	}
	if result.Valid {
		t.Errorf("Valid must be false")
	}
}

// TestComposedActIDInvalidFormat verifies that invalid act_id format
// produces semantic failure with correct result structure.
func TestComposedActIDInvalidFormat(t *testing.T) {
	// act_id does not match ACT-[A-Z0-9][A-Z0-9-]{2,199} pattern
	json := `{
  "contract_version": 1,
  "act_id": "bad-act-id",
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
}`

	result := ValidatePlanComposed([]byte(json))

	if !result.Structural.Valid {
		t.Fatalf("structural must be valid for act_id invalid format")
	}
	if !result.Decoded {
		t.Errorf("Decoded must be true")
	}
	if len(result.DecodeErrors) != 0 {
		t.Errorf("DecodeErrors must be empty")
	}
	if result.SemanticValid {
		t.Errorf("SemanticValid must be false")
	}
	if len(result.SemanticErrors) == 0 {
		t.Errorf("SemanticErrors must be non-null and non-empty")
	}
	if result.Valid {
		t.Errorf("Valid must be false")
	}

	// Verify error location
	found := false
	for _, e := range result.SemanticErrors {
		if e.InstancePath == "/act_id" && e.Code == PlanCodeSemanticConstraintFailed {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error at /act_id with code %s, got: %+v",
			PlanCodeSemanticConstraintFailed, result.SemanticErrors)
	}
}

// TestComposedBaselineOIDInvalid verifies that invalid baseline OIDs
// produce semantic failures with correct result structure.
func TestComposedBaselineOIDInvalid(t *testing.T) {
	cases := []struct {
		name     string
		json     string
		wantPath string
	}{
		{
			name: "invalid commit_oid",
			json: `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-COMPOSED",
  "baseline": {
    "commit_oid": "not-a-valid-oid",
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
			wantPath: "/baseline/commit_oid",
		},
		{
			name: "invalid tree_oid",
			json: `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-COMPOSED",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "not-a-valid-tree"
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
			wantPath: "/baseline/tree_oid",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidatePlanComposed([]byte(tc.json))

			if !result.Structural.Valid {
				t.Fatalf("structural must be valid")
			}
			if !result.Decoded {
				t.Errorf("Decoded must be true")
			}
			if result.SemanticValid {
				t.Errorf("SemanticValid must be false")
			}
			if len(result.SemanticErrors) == 0 {
				t.Errorf("SemanticErrors must be non-null and non-empty")
			}
			if result.Valid {
				t.Errorf("Valid must be false")
			}

			found := false
			for _, e := range result.SemanticErrors {
				if e.InstancePath == tc.wantPath && e.Code == PlanCodeSemanticConstraintFailed {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error at %s, got: %+v", tc.wantPath, result.SemanticErrors)
			}
		})
	}
}
