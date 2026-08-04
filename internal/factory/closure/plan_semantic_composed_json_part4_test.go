package closure

import (
	"testing"
)

// TestComposedExecutionModeStructuralRejection verifies that execution mode
// validation happens at the structural level, not semantic level.
func TestComposedExecutionModeStructuralRejection(t *testing.T) {
	// The structural validator should reject unknown execution modes
	cases := []struct {
		name      string
		json      string
		wantValid bool
	}{
		{
			name: "unknown execution mode rejected structurally",
			json: `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-COMPOSED",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "verify"},
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
			wantValid: false,
		},
		{
			name: "empty execution mode rejected structurally",
			json: `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-COMPOSED",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": ""},
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
			wantValid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidatePlanComposed([]byte(tc.json))

			// Structural validation should reject unknown/empty modes
			if result.Structural.Valid != tc.wantValid {
				t.Errorf("Structural.Valid = %v, want %v", result.Structural.Valid, tc.wantValid)
			}
			if result.Valid != tc.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tc.wantValid)
			}
		})
	}
}

// TestComposedDuplicateCheckID verifies that duplicate check IDs
// produce semantic failures with correct result structure.
func TestComposedDuplicateCheckID(t *testing.T) {
	result := ValidatePlanComposed(composedPlanDuplicateCheckID())

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
		if e.InstancePath == "/checks/1/id" && e.Code == PlanCodeSemanticConstraintFailed {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error at /checks/1/id, got: %+v", result.SemanticErrors)
	}
}

// TestComposedArgvElementFailure verifies that argv element failures
// produce semantic failures with correct result structure.
func TestComposedArgvElementFailure(t *testing.T) {
	// Empty argv element
	jsonEmpty := `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-COMPOSED",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "check-one", "mode": "run", "argv": ["", "test"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

	result := ValidatePlanComposed([]byte(jsonEmpty))

	if !result.Structural.Valid {
		t.Fatalf("structural must be valid for empty argv element")
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
		if e.InstancePath == "/checks/0/argv/0" && e.Code == PlanCodeSemanticConstraintFailed {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error at /checks/0/argv/0, got: %+v", result.SemanticErrors)
	}
}

// TestComposedDuplicateArtifactID verifies that duplicate artifact IDs
// produce semantic failures with correct result structure.
func TestComposedDuplicateArtifactID(t *testing.T) {
	json := `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-COMPOSED",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "check-one", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [
    {"id": "artifact-x", "path": "out/a.txt", "required": true, "max_bytes": 1024, "media_type": "text/plain"},
    {"id": "artifact-x", "path": "out/b.txt", "required": false, "max_bytes": 1024, "media_type": "text/plain"}
  ],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

	result := ValidatePlanComposed([]byte(json))

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
		if e.InstancePath == "/artifacts/1/id" && e.Code == PlanCodeSemanticConstraintFailed {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error at /artifacts/1/id, got: %+v", result.SemanticErrors)
	}
}

// TestComposedPolicyDiagnosticsOrdered verifies that policy diagnostics
// are returned in the order defined by PolicyFieldOrder.
func TestComposedPolicyDiagnosticsOrdered(t *testing.T) {
	result := ValidatePlanComposed([]byte(canonicalComposedPlan()))
	if !result.SemanticValid {
		t.Fatalf("canonical plan must be semantically valid")
	}
	if len(result.SemanticErrors) != 0 {
		t.Errorf("expected no semantic errors for valid plan, got %d", len(result.SemanticErrors))
	}
}
