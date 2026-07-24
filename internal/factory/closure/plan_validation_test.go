package closure

import (
	"os"
	"testing"
)

func TestValidatePlanStructure_ValidPlan(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"contract_version": 1,
		"description": "Test plan",
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

	err = ValidatePlanStructure(tmp.Name())
	if err != nil {
		t.Errorf("expected valid plan: %v", err)
	}
}

func TestValidatePlanStructure_RejectsFreezeCommit(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"freeze_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711"
	}`

	tmp, err := os.CreateTemp("", "plan-invalid-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plan); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	err = ValidatePlanStructure(tmp.Name())
	if err == nil {
		t.Error("expected rejection of freeze_commit")
	}
}

func TestValidatePlanStructure_RejectsSubjectCommit(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"subject_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711"
	}`

	tmp, err := os.CreateTemp("", "plan-invalid-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plan); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	err = ValidatePlanStructure(tmp.Name())
	if err == nil {
		t.Error("expected rejection of subject_commit")
	}
}

func TestValidatePlanStructure_RejectsClosureCommit(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"closure_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711"
	}`

	tmp, err := os.CreateTemp("", "plan-invalid-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plan); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	err = ValidatePlanStructure(tmp.Name())
	if err == nil {
		t.Error("expected rejection of closure_commit")
	}
}

func TestValidatePlanStructure_RejectsTagOID(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"tag_oid": "8362d35c65f66ccd140f5b5044b776f435fdc711"
	}`

	tmp, err := os.CreateTemp("", "plan-invalid-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plan); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	err = ValidatePlanStructure(tmp.Name())
	if err == nil {
		t.Error("expected rejection of tag_oid")
	}
}

func TestValidatePlanStructure_RejectsTagObjectOID(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"tag_object_oid": "8362d35c65f66ccd140f5b5044b776f435fdc711"
	}`

	tmp, err := os.CreateTemp("", "plan-invalid-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plan); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	err = ValidatePlanStructure(tmp.Name())
	if err == nil {
		t.Error("expected rejection of tag_object_oid")
	}
}

func TestValidatePlanStructure_RejectsTagTarget(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"tag_target": "8362d35c65f66ccd140f5b5044b776f435fdc711"
	}`

	tmp, err := os.CreateTemp("", "plan-invalid-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plan); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	err = ValidatePlanStructure(tmp.Name())
	if err == nil {
		t.Error("expected rejection of tag_target")
	}
}

func TestValidatePlanStructure_RejectsPeeledTarget(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"peeled_target": "8362d35c65f66ccd140f5b5044b776f435fdc711"
	}`

	tmp, err := os.CreateTemp("", "plan-invalid-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plan); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	err = ValidatePlanStructure(tmp.Name())
	if err == nil {
		t.Error("expected rejection of peeled_target")
	}
}

func TestValidatePlanStructure_RejectsNestedForbiddenKey(t *testing.T) {
	// Nested in an object
	plan := `{
		"act_id": "ACT-TEST-01",
		"nested": {
			"freeze_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711"
		}
	}`

	tmp, err := os.CreateTemp("", "plan-invalid-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plan); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	err = ValidatePlanStructure(tmp.Name())
	if err == nil {
		t.Error("expected rejection of nested freeze_commit")
	}
}

func TestValidatePlanStructure_RejectsForbiddenKeyInArray(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"items": [
			{"subject_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711"}
		]
	}`

	tmp, err := os.CreateTemp("", "plan-invalid-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(plan); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()

	err = ValidatePlanStructure(tmp.Name())
	if err == nil {
		t.Error("expected rejection of forbidden key in array")
	}
}

func TestValidatePlanBytes_RejectsSHA256OID(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"freeze_commit": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	}`

	err := ValidatePlanBytes([]byte(plan))
	if err == nil {
		t.Error("expected rejection of SHA256 OID in forbidden field")
	}
}

func TestValidatePlanBytes_RejectsNullValue(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"closure_commit": null
	}`

	err := ValidatePlanBytes([]byte(plan))
	if err == nil {
		t.Error("expected rejection of null in forbidden field")
	}
}

func TestValidatePlanBytes_RejectsNumberValue(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"freeze_tree": 12345
	}`

	err := ValidatePlanBytes([]byte(plan))
	if err == nil {
		t.Error("expected rejection of number in forbidden field")
	}
}

func TestValidatePlanBytes_RejectsArrayValue(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"subject_tree": [1, 2, 3]
	}`

	err := ValidatePlanBytes([]byte(plan))
	if err == nil {
		t.Error("expected rejection of array in forbidden field")
	}
}

func TestValidatePlanBytes_RejectsPlaceholderValue(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"tag_oid": "TODO"
	}`

	err := ValidatePlanBytes([]byte(plan))
	if err == nil {
		t.Error("expected rejection of placeholder in forbidden field")
	}
}
