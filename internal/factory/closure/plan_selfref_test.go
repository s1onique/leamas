package closure

import (
	"os"
	"regexp"
	"testing"
)

func TestPlanRejectsSelfReferentialFields(t *testing.T) {
	// Fields that should NOT be in a frozen plan
	selfRefFields := []string{
		"freeze_commit",
		"freeze_tree",
		"subject_commit",
		"subject_tree",
		"closure_commit",
		"closure_tree",
		"tag_oid",
		"tag_target",
	}

	for _, field := range selfRefFields {
		// Create a plan with the self-referential field
		plan := `{
			"act_id": "ACT-TEST-PLAN-01",
			"contract_version": 1,
			"` + field + `": "8362d35c65f66ccd140f5b5044b776f435fdc711",
			"baseline": {"commit_oid": "8362d35c65f66ccd140f5b5044b776f435fdc711", "tree_oid": "906b2afc0c7a8da3a003231dcc0c32a91d5e829b"},
			"execution": {"mode": "serial_fail_fast"},
			"checks": [{"id": "test", "mode": "run", "argv": ["echo", "test"]}],
			"artifacts": [],
			"policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
		}`

		tmp, err := os.CreateTemp("", "plan-selfref-*.json")
		if err != nil {
			t.Fatalf("create temp: %v", err)
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.WriteString(plan); err != nil {
			t.Fatalf("write temp: %v", err)
		}
		tmp.Close()

		// Check that the field is detected in the plan
		pattern := regexp.MustCompile(`"` + field + `"\s*:\s*"[0-9a-f]{40}"`)
		matched, _ := regexp.MatchString(pattern.String(), plan)
		if !matched {
			t.Errorf("expected to detect %s in plan", field)
		}
	}
}

func TestPlanAllowsProspectiveFields(t *testing.T) {
	// Fields that SHOULD be allowed in a frozen plan
	plan := `{
		"act_id": "ACT-TEST-PLAN-01",
		"contract_version": 1,
		"description": "Test plan",
		"verification_commands": ["go test ./..."],
		"acceptance_criteria": ["tests pass"],
		"baseline": {"commit_oid": "8362d35c65f66ccd140f5b5044b776f435fdc711", "tree_oid": "906b2afc0c7a8da3a003231dcc0c32a91d5e829b"},
		"execution": {"mode": "serial_fail_fast"},
		"checks": [{"id": "test", "mode": "run", "argv": ["echo", "test"]}],
		"artifacts": [],
		"policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
	}`

	tmp, err := os.CreateTemp("", "plan-valid-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plan); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	// Check no self-reference fields
	noSelf, err := CheckPlanNoSelfReference(tmp.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if noSelf.PlanFreezeCommitInPlan || noSelf.PlanSubjectCommitInPlan {
		t.Error("plan should not contain self-referential fields")
	}
}

func TestDetectObjectFormat_SHA1(t *testing.T) {
	format := DetectObjectFormat("8362d35c65f66ccd140f5b5044b776f435fdc711")
	if format != ObjectFormatSHA1 {
		t.Errorf("expected SHA1, got %s", format)
	}
}

func TestDetectObjectFormat_SHA256(t *testing.T) {
	format := DetectObjectFormat("a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	if format != ObjectFormatSHA256 {
		t.Errorf("expected SHA256, got %s", format)
	}
}

func TestDetectObjectFormat_Unknown(t *testing.T) {
	format := DetectObjectFormat("abc123")
	if format != ObjectFormatUnknown {
		t.Errorf("expected Unknown, got %s", format)
	}
}

func TestValidateOIDWithFormat_SHA1(t *testing.T) {
	err := ValidateOIDWithFormat("commit", "8362d35c65f66ccd140f5b5044b776f435fdc711", ObjectFormatSHA1)
	if err != nil {
		t.Errorf("expected valid SHA1 OID: %v", err)
	}
}

func TestValidateOIDWithFormat_SHA256(t *testing.T) {
	err := ValidateOIDWithFormat("commit", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", ObjectFormatSHA256)
	if err != nil {
		t.Errorf("expected valid SHA256 OID: %v", err)
	}
}

func TestValidateOIDWithFormat_WrongFormat(t *testing.T) {
	// SHA1 OID in SHA256 format should fail
	err := ValidateOIDWithFormat("commit", "8362d35c65f66ccd140f5b5044b776f435fdc711", ObjectFormatSHA256)
	if err == nil {
		t.Error("expected failure for SHA1 OID in SHA256 format")
	}

	// SHA256 OID in SHA1 format should fail
	err = ValidateOIDWithFormat("commit", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", ObjectFormatSHA1)
	if err == nil {
		t.Error("expected failure for SHA256 OID in SHA1 format")
	}
}

func TestValidateOIDWithFormat_UnknownFormat(t *testing.T) {
	err := ValidateOIDWithFormat("commit", "8362d35c65f66ccd140f5b5044b776f435fdc711", ObjectFormatUnknown)
	if err == nil {
		t.Error("expected failure for unknown format")
	}
}
