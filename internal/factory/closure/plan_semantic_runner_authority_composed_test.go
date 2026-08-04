package closure

// plan_semantic_runner_authority_composed_test.go proves that
// ValidateRunnerAuthority is wired into the composed semantic
// validation pipeline (ValidatePlan via ValidatePlanComposed).

import (
	"testing"
)

// canonicalRunnerAuthorityToolReleaseExactNoToolJSON is a structurally valid plan
// with runner_authority.mode="tool_release_exact" and no runner_authority.tool.
const canonicalRunnerAuthorityToolReleaseExactNoToolJSON = `{
  "contract_version": 1,
  "act_id": "ACT-RUNNER-AUTH-TOOL-MISSING",
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
  },
  "runner_authority": {"mode": "tool_release_exact"}
}`

func TestRunnerAuthorityToolReleaseExactWithoutToolRejected(t *testing.T) {
	result := ValidatePlanComposed([]byte(canonicalRunnerAuthorityToolReleaseExactNoToolJSON))
	if !result.Structural.Valid {
		t.Fatalf("Structural.Valid must be true; got errors: %+v", result.Structural.Errors)
	}
	if !result.Decoded {
		t.Fatalf("Decoded must be true")
	}
	if len(result.DecodeErrors) != 0 {
		t.Fatalf("DecodeErrors must be empty; got %d", len(result.DecodeErrors))
	}
	if result.SemanticValid {
		t.Fatalf("SemanticValid must be false for tool_release_exact without tool")
	}
	if result.Valid {
		t.Fatalf("Valid must be false")
	}
	if len(result.SemanticErrors) != 1 {
		t.Fatalf("expected exactly 1 runner diagnostic, got %d: %+v",
			len(result.SemanticErrors), result.SemanticErrors)
	}
	diag := result.SemanticErrors[0]
	if diag.InstancePath != "/runner_authority/tool" {
		t.Errorf("InstancePath = %q, want %q", diag.InstancePath, "/runner_authority/tool")
	}
	if diag.Code != PlanCodeSemanticConstraintFailed {
		t.Errorf("Code = %q, want %q", diag.Code, PlanCodeSemanticConstraintFailed)
	}
}

func TestRunnerAuthoritySubjectExactRegression(t *testing.T) {
	json := `{
  "contract_version": 1,
  "act_id": "ACT-RUNNER-AUTH-SUBJECT-EXACT",
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
  },
  "runner_authority": {"mode": "subject_exact"}
}`
	result := ValidatePlanComposed([]byte(json))
	if !result.Structural.Valid {
		t.Fatalf("Structural.Valid must be true: %+v", result.Structural.Errors)
	}
	if !result.Decoded {
		t.Fatalf("Decoded must be true")
	}
	if len(result.DecodeErrors) != 0 {
		t.Fatalf("DecodeErrors must be empty")
	}
	if !result.SemanticValid {
		t.Fatalf("SemanticValid must be true for subject_exact; errors: %+v", result.SemanticErrors)
	}
	if !result.Valid {
		t.Fatalf("Valid must be true for subject_exact")
	}
}

func TestRunnerAuthorityToolReleaseExactCompleteRegression(t *testing.T) {
	json := `{
  "contract_version": 1,
  "act_id": "ACT-RUNNER-AUTH-TOOL-RELEASE-COMPLETE",
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
  },
  "runner_authority": {
    "mode": "tool_release_exact",
    "tool": {
      "revision": "1111111111111111111111111111111111111111",
      "binary_sha256": "1111111111111111111111111111111111111111111111111111111111111111"
    }
  }
}`
	result := ValidatePlanComposed([]byte(json))
	if !result.Structural.Valid {
		t.Fatalf("Structural.Valid must be true: %+v", result.Structural.Errors)
	}
	if !result.Decoded {
		t.Fatalf("Decoded must be true")
	}
	if len(result.DecodeErrors) != 0 {
		t.Fatalf("DecodeErrors must be empty")
	}
	if !result.SemanticValid {
		t.Fatalf("SemanticValid must be true for complete tool_release_exact; errors: %+v",
			result.SemanticErrors)
	}
	if !result.Valid {
		t.Fatalf("Valid must be true for complete tool_release_exact")
	}
}

func TestRunnerAuthorityToolReleaseExactMissingRevisionStructuralCatch(t *testing.T) {
	json := `{
  "contract_version": 1,
  "act_id": "ACT-RUNNER-AUTH-TOOL-NO-REV",
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
  },
  "runner_authority": {
    "mode": "tool_release_exact",
    "tool": {
      "binary_sha256": "1111111111111111111111111111111111111111111111111111111111111111"
    }
  }
}`
	result := ValidatePlanComposed([]byte(json))
	if result.Structural.Valid {
		t.Fatalf("Structural.Valid must be false for missing revision")
	}
	if result.Valid {
		t.Fatalf("Valid must be false")
	}
	found := false
	for _, diag := range result.Structural.Errors {
		if diag.InstancePath == "/runner_authority/tool/revision" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected structural error at /runner_authority/tool/revision; got: %+v",
			result.Structural.Errors)
	}
}

func TestRunnerAuthorityToolReleaseExactInvalidRevisionLength(t *testing.T) {
	json := `{
  "contract_version": 1,
  "act_id": "ACT-RUNNER-AUTH-TOOL-SHORT-REV",
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
  },
  "runner_authority": {
    "mode": "tool_release_exact",
    "tool": {
      "revision": "abc123",
      "binary_sha256": "1111111111111111111111111111111111111111111111111111111111111111"
    }
  }
}`
	result := ValidatePlanComposed([]byte(json))
	if !result.Structural.Valid {
		t.Fatalf("Structural.Valid must be true: %+v", result.Structural.Errors)
	}
	if result.SemanticValid {
		t.Fatalf("SemanticValid must be false for invalid revision length")
	}
	if result.Valid {
		t.Fatalf("Valid must be false")
	}
	found := false
	for _, diag := range result.SemanticErrors {
		if diag.InstancePath == "/runner_authority/tool/revision" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected semantic error at /runner_authority/tool/revision; got: %+v",
			result.SemanticErrors)
	}
}
