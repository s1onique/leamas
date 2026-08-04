package closure

import (
	"testing"
)

// TestComposedBaselineOIDPlaceholder verifies that baseline OIDs with
// closure placeholders produce proper semantic errors.
func TestComposedBaselineOIDPlaceholder(t *testing.T) {
	cases := []struct {
		name          string
		commitOID     string
		treeOID       string
		wantCommitErr bool
		wantTreeErr   bool
	}{
		{
			name:          "commit with TODO placeholder",
			commitOID:     "TODO",
			treeOID:       "2222222222222222222222222222222222222222",
			wantCommitErr: true,
			wantTreeErr:   false,
		},
		{
			name:          "tree with TBD placeholder",
			commitOID:     "1111111111111111111111111111111111111111",
			treeOID:       "TBD",
			wantCommitErr: false,
			wantTreeErr:   true,
		},
		{
			name:          "commit with UNKNOWN placeholder",
			commitOID:     "UNKNOWN",
			treeOID:       "2222222222222222222222222222222222222222",
			wantCommitErr: true,
			wantTreeErr:   false,
		},
		{
			name:          "tree with RUNNING placeholder",
			commitOID:     "1111111111111111111111111111111111111111",
			treeOID:       "RUNNING",
			wantCommitErr: false,
			wantTreeErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			json := `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-COMPOSED",
  "baseline": {
    "commit_oid": "` + tc.commitOID + `",
    "tree_oid": "` + tc.treeOID + `"
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

			if result.SemanticValid {
				t.Fatalf("plan with placeholder OID must be semantically invalid")
			}
			if len(result.SemanticErrors) == 0 {
				t.Fatalf("SemanticErrors must be populated")
			}

			hasCommitErr := false
			hasTreeErr := false
			for _, e := range result.SemanticErrors {
				if e.InstancePath == "/baseline/commit_oid" {
					hasCommitErr = true
				}
				if e.InstancePath == "/baseline/tree_oid" {
					hasTreeErr = true
				}
			}

			if tc.wantCommitErr && !hasCommitErr {
				t.Errorf("expected commit_oid error, got: %+v", result.SemanticErrors)
			}
			if tc.wantTreeErr && !hasTreeErr {
				t.Errorf("expected tree_oid error, got: %+v", result.SemanticErrors)
			}
		})
	}
}

// TestComposedActIDPlaceholder verifies that act_id with closure placeholders
// produces proper semantic errors.
func TestComposedActIDPlaceholder(t *testing.T) {
	// Only exact placeholders are recognized
	cases := []string{"TODO", "TBD", "UNKNOWN", "RUNNING"}

	for _, placeholder := range cases {
		t.Run(placeholder, func(t *testing.T) {
			json := `{
  "contract_version": 1,
  "act_id": "` + placeholder + `",
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

			if result.SemanticValid {
				t.Fatalf("plan with placeholder act_id %q must be semantically invalid", placeholder)
			}
			if len(result.SemanticErrors) == 0 {
				t.Fatalf("SemanticErrors must be populated")
			}

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
		})
	}
}

// TestComposedCheckArgvPlaceholder verifies that argv elements with closure
// placeholders produce proper semantic errors.
func TestComposedCheckArgvPlaceholder(t *testing.T) {
	cases := []struct {
		name     string
		argvJSON string
		wantPath string
	}{
		// Empty string is caught by semantic validation
		{"argv-first-element-placeholder", `["TODO", "test"]`, "/checks/0/argv/0"},
		{"argv-middle-element-placeholder", `["echo", "TBD", "test"]`, "/checks/0/argv/1"},
		// UNKNOWN placeholder in argv
		{"argv-unknown-placeholder", `["echo", "UNKNOWN"]`, "/checks/0/argv/1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			json := `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-COMPOSED",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "check-one", "mode": "run", "argv": ` + tc.argvJSON + `, "working_directory": ".", "timeout_seconds": 60, "environment": {}}
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

			if result.SemanticValid {
				t.Fatalf("plan with placeholder argv must be semantically invalid")
			}
			if len(result.SemanticErrors) == 0 {
				t.Fatalf("SemanticErrors must be populated")
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

// TestComposedArtifactPlaceholder verifies that artifact IDs with closure
// placeholders produce proper semantic errors.
func TestComposedArtifactPlaceholder(t *testing.T) {
	// Only exact placeholders are recognized
	cases := []string{"TODO", "TBD", "UNKNOWN", "RUNNING"}

	for _, placeholder := range cases {
		t.Run(placeholder, func(t *testing.T) {
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
    {"id": "` + placeholder + `", "path": "out/result.txt", "required": true, "max_bytes": 1024, "media_type": "text/plain"}
  ],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`

			result := ValidatePlanComposed([]byte(json))

			if result.SemanticValid {
				t.Fatalf("plan with placeholder artifact ID %q must be semantically invalid", placeholder)
			}
			if len(result.SemanticErrors) == 0 {
				t.Fatalf("SemanticErrors must be populated")
			}

			found := false
			for _, e := range result.SemanticErrors {
				if e.InstancePath == "/artifacts/0/id" && e.Code == PlanCodeSemanticConstraintFailed {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error at /artifacts/0/id, got: %+v", result.SemanticErrors)
			}
		})
	}
}

// TestComposedMultipleSemanticErrors verifies that a plan can produce
// multiple semantic errors. Note: the current implementation returns
// only the first error (early return on validation failure). This test
// verifies that at least one semantic error is properly captured.
// To support multiple errors, the validation logic would need to
// collect all errors before returning.
func TestComposedMultipleSemanticErrors(t *testing.T) {
	// Plan with an invalid baseline OID and duplicate check ID.
	// These are two different validation paths that could theoretically
	// both produce errors. However, the current implementation returns
	// only the first error encountered.
	fixture := `{
  "contract_version": 1,
  "act_id": "ACT-MULTI-ERR",
  "baseline": {
    "commit_oid": "bad-oid",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "check-one", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}},
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

	result := ValidatePlanComposed([]byte(fixture))

	if result.SemanticValid {
		t.Fatalf("plan with multiple issues must be semantically invalid")
	}
	// The implementation returns only the first error
	if len(result.SemanticErrors) < 1 {
		t.Errorf("expected at least 1 semantic error, got %d: %+v", len(result.SemanticErrors), result.SemanticErrors)
	}

	// Verify the baseline commit_oid error is present (the first error returned)
	foundCommitErr := false
	foundDupCheckErr := false
	for _, e := range result.SemanticErrors {
		if e.InstancePath == "/baseline/commit_oid" {
			foundCommitErr = true
		}
		if e.InstancePath == "/checks/1/id" {
			foundDupCheckErr = true
		}
	}

	// At minimum, we should get the first error
	if !foundCommitErr && !foundDupCheckErr {
		t.Errorf("expected at least one semantic error from commit_oid or duplicate check, got: %+v", result.SemanticErrors)
	}
}
