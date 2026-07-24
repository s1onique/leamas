package closure

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestDecodeAttestation_Valid(t *testing.T) {
	data := []byte(`{
		"attestation_version": 1,
		"act_id": "ACT-TEST-01",
		"protocol_version": "1",
		"attested_at": "2026-07-23T19:47:00Z",
		"description": "test",
		"closure_reference": {
			"closure_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711",
			"closure_tree": "906b2afc0c7a8da3a003231dcc0c32a91d5e829b"
		},
		"tag_identity": {
			"tag_name": "act/ACT-TEST-01",
			"tag_object_oid": "0235c610b7679647c94ce8bfb2dc752b63384031",
			"tag_type": "annotated",
			"peeled_target": "8362d35c65f66ccd140f5b5044b776f435fdc711"
		},
		"freeze_reference": {
			"freeze_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711",
			"freeze_tree": "906b2afc0c7a8da3a003231dcc0c32a91d5e829b",
			"plan_blob_oid": "334829218ab7af3283e6c5ee29e31fc062860d5f"
		},
		"subject_reference": {
			"subject_commit": "64b6c20c0e0230f1eeb8aa1f5e21f96220f9bf28",
			"subject_tree": "7a8734d221eb54924c8810d625fb0ac3d2b4a997"
		},
		"chain_validity": {
			"F_not_equal_S": true,
			"tag_peeled_target_matches_C": true
		},
		"no_self_reference_in_plan": {}
	}`)
	a, err := DecodeAttestation(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ActID != "ACT-TEST-01" {
		t.Errorf("expected ACT-TEST-01, got %s", a.ActID)
	}
}

func TestDecodeAttestation_InvalidTagType(t *testing.T) {
	data := []byte(`{
		"attestation_version": 1,
		"act_id": "ACT-TEST-01",
		"protocol_version": "1",
		"attested_at": "2026-07-23T19:47:00Z",
		"description": "test",
		"closure_reference": {
			"closure_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711",
			"closure_tree": "906b2afc0c7a8da3a003231dcc0c32a91d5e829b"
		},
		"tag_identity": {
			"tag_name": "act/ACT-TEST-01",
			"tag_object_oid": "0235c610b7679647c94ce8bfb2dc752b63384031",
			"tag_type": "invalid",
			"peeled_target": "8362d35c65f66ccd140f5b5044b776f435fdc711"
		},
		"freeze_reference": {
			"freeze_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711",
			"freeze_tree": "906b2afc0c7a8da3a003231dcc0c32a91d5e829b",
			"plan_blob_oid": "334829218ab7af3283e6c5ee29e31fc062860d5f"
		},
		"subject_reference": {
			"subject_commit": "64b6c20c0e0230f1eeb8aa1f5e21f96220f9bf28",
			"subject_tree": "7a8734d221eb54924c8810d625fb0ac3d2b4a997"
		},
		"chain_validity": {
			"F_not_equal_S": true,
			"tag_peeled_target_matches_C": true
		},
		"no_self_reference_in_plan": {}
	}`)
	_, err := DecodeAttestation(data)
	if err == nil {
		t.Error("expected error for invalid tag type")
	}
}

func TestDecodeAttestation_PlaceholderOID(t *testing.T) {
	data := []byte(`{
		"attestation_version": 1,
		"act_id": "ACT-TEST-01",
		"protocol_version": "1",
		"attested_at": "2026-07-23T19:47:00Z",
		"description": "test",
		"closure_reference": {
			"closure_commit": "TODO",
			"closure_tree": "906b2afc0c7a8da3a003231dcc0c32a91d5e829b"
		},
		"tag_identity": {
			"tag_name": "act/ACT-TEST-01",
			"tag_object_oid": "0235c610b7679647c94ce8bfb2dc752b63384031",
			"tag_type": "annotated",
			"peeled_target": "8362d35c65f66ccd140f5b5044b776f435fdc711"
		},
		"freeze_reference": {
			"freeze_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711",
			"freeze_tree": "906b2afc0c7a8da3a003231dcc0c32a91d5e829b",
			"plan_blob_oid": "334829218ab7af3283e6c5ee29e31fc062860d5f"
		},
		"subject_reference": {
			"subject_commit": "64b6c20c0e0230f1eeb8aa1f5e21f96220f9bf28",
			"subject_tree": "7a8734d221eb54924c8810d625fb0ac3d2b4a997"
		"chain_validity": {
			"F_not_equal_S": true,
			"tag_peeled_target_matches_C": true
		},
		"chain_validity": {},
		"no_self_reference_in_plan": {}
	}`)
	_, err := DecodeAttestation(data)
	if err == nil {
		t.Error("expected error for placeholder OID")
	}
}

func TestAttestationJSONRoundTrip(t *testing.T) {
	a := Attestation{
		AttestationVersion: 1,
		ActID:              "ACT-TEST-01",
		ProtocolVersion:    "1",
		AttestedAt:         "2026-07-23T19:47:00Z",
		Description:        "Test",
		ClosureReference: ClosureReference{
			ClosureCommit: "8362d35c65f66ccd140f5b5044b776f435fdc711",
			ClosureTree:   "906b2afc0c7a8da3a003231dcc0c32a91d5e829b",
		},
		TagIdentity: TagIdentity{
			TagName:      "act/ACT-TEST-01",
			TagObjectOID: "0235c610b7679647c94ce8bfb2dc752b63384031",
			TagType:      "annotated",
			PeeledTarget: "8362d35c65f66ccd140f5b5044b776f435fdc711",
		},
		FreezeReference: FreezeReference{
			FreezeCommit: "8362d35c65f66ccd140f5b5044b776f435fdc711",
			FreezeTree:   "906b2afc0c7a8da3a003231dcc0c32a91d5e829b",
			PlanBlobOID:  "334829218ab7af3283e6c5ee29e31fc062860d5f",
		},
		SubjectReference: SubjectReference{
			SubjectCommit: "64b6c20c0e0230f1eeb8aa1f5e21f96220f9bf28",
			SubjectTree:   "7a8734d221eb54924c8810d625fb0ac3d2b4a997",
		},
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundTrip Attestation
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if roundTrip.ActID != a.ActID {
		t.Error("act_id mismatch after round trip")
	}
}

func TestCheckPlanNoSelfReference(t *testing.T) {
	content := `{
		"act_id": "ACT-TEST-01",
		"protocol_version": "1",
		"description": "Test plan without self-references"
	}`
	tmp, err := os.CreateTemp("", "plan-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()
	noSelf, err := CheckPlanNoSelfReference(tmp.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if noSelf.PlanFreezeCommitInPlan {
		t.Error("should not detect freeze_commit in plan")
	}
}

func TestCheckPlanNoSelfReference_WithSelfReference(t *testing.T) {
	content := `{
		"act_id": "ACT-TEST-01",
		"freeze_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711",
		"protocol_version": "1"
	}`
	tmp, err := os.CreateTemp("", "plan-*.json")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()
	noSelf, err := CheckPlanNoSelfReference(tmp.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !noSelf.PlanFreezeCommitInPlan {
		t.Error("should detect freeze_commit in plan")
	}
}

func TestChainValidationResultOutput(t *testing.T) {
	result := ChainValidationResult{
		Verdict:   "PASS",
		Errors:    nil,
		AllChecks: []string{"freeze=8362d35c65f66ccd140f5b5044b776f435fdc711 (commit)"},
	}
	var sb strings.Builder
	result.Output(&sb, false)
	if sb.String() == "" {
		t.Error("expected non-empty output")
	}
}

func TestChainValidationResultOutputJSON(t *testing.T) {
	result := ChainValidationResult{
		Verdict:   "PASS",
		Errors:    nil,
		AllChecks: []string{"freeze=8362d35c65f66ccd140f5b5044b776f435fdc711 (commit)"},
	}
	var sb strings.Builder
	result.Output(&sb, true)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(sb.String()), &parsed); err != nil {
		t.Errorf("expected valid JSON: %v", err)
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "commit",
		Message: "invalid value",
	}
	expected := "commit: invalid value"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
