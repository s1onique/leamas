package gatesummary

import (
	"strings"
	"testing"
)

func TestDecodeV2Minimal(t *testing.T) {
	data := readFixture(t, "testdata/valid/v2-minimal.json")
	res := Decode(strings.NewReader(string(data)))
	if !res.Success() {
		t.Fatalf("expected success, got diags=%v err=%v", res.Diagnostics, res.Err)
	}
	if res.Document.Version() != Version2 {
		t.Fatalf("expected v2, got %s", res.Document.Version())
	}
	v2, ok := res.Document.V2()
	if !ok {
		t.Fatal("expected V2()")
	}
	if v2.SchemaVersion != 2 {
		t.Errorf("schema_version=%d", v2.SchemaVersion)
	}
	if len(v2.Checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(v2.Checks))
	}
}

func TestDecodeV2Full(t *testing.T) {
	data := readFixture(t, "testdata/valid/v2-full.json")
	res := Decode(strings.NewReader(string(data)))
	if !res.Success() {
		t.Fatalf("expected success, got diags=%v", res.Diagnostics)
	}
	v2, ok := res.Document.V2()
	if !ok {
		t.Fatal("expected V2()")
	}
	if v2.Checks[0].Total == nil {
		t.Error("expected v2-full to have a meaningful test total")
	}
}

func TestDecodeV2ClinemmMicroc3(t *testing.T) {
	data := readFixture(t, "testdata/valid/v2-clinemm-microc3.json")
	res := Decode(strings.NewReader(string(data)))
	if !res.Success() {
		t.Fatalf("expected success, got diags=%v", res.Diagnostics)
	}
	v2, ok := res.Document.V2()
	if !ok {
		t.Fatal("expected V2()")
	}
	if v2.OverallStatus != "fail" {
		t.Errorf("overall_status=%q, want fail", v2.OverallStatus)
	}
	if len(v2.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(v2.Checks))
	}
}

func TestDecodeV2RootScope(t *testing.T) {
	data := readFixture(t, "testdata/valid/v2-root-scope.json")
	res := Decode(strings.NewReader(string(data)))
	if !res.Success() {
		t.Fatalf("expected success, got diags=%v", res.Diagnostics)
	}
	v2, _ := res.Document.V2()
	if v2.ParentAct != "" {
		t.Errorf("root scope parent_act=%q, want empty", v2.ParentAct)
	}
}

func TestDecodeV2LeamasSelfHosted(t *testing.T) {
	data := readFixture(t, "testdata/valid/v2-leamas-self-hosted.json")
	res := Decode(strings.NewReader(string(data)))
	if !res.Success() {
		t.Fatalf("expected success, got diags=%v", res.Diagnostics)
	}
	v2, _ := res.Document.V2()
	if len(v2.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(v2.Checks))
	}
}

func TestDecodeV2ExitCodeNullDistinct(t *testing.T) {
	// A v2 check with exit_code: 0 (integer) vs null (nullable)
	// must remain distinguishable.
	data := []byte(`{
		"schema_version": 2,
		"generated_at": "2026-07-19T08:43:26Z",
		"scope_id": "ACT-X",
		"scope_status": "CLOSED",
		"scope_disposition": "d",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "d",
		"overall_status": "pass",
		"overall_disposition": "d",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
			{
				"name": "int_zero",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "e",
				"detail": "d",
				"extras": {
					"argv": [],
					"exit_code": 0,
					"duration_ms": 0,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			},
			{
				"name": "null_exit",
				"scope": "ROOT",
				"status": "skip",
				"evidence": "e",
				"detail": "d",
				"extras": {
					"argv": [],
					"exit_code": null,
					"duration_ms": 0,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			}
		]
	}`)
	res := Decode(strings.NewReader(string(data)))
	if !res.Success() {
		t.Fatalf("expected success, got diags=%v", res.Diagnostics)
	}
	v2, _ := res.Document.V2()
	if v2.Checks[0].Extras.ExitCode == nil {
		t.Fatal("expected integer zero to remain non-nil")
	}
	value, ok := v2.Checks[0].Extras.ExitCode.Int64()
	if !ok || value != 0 {
		t.Fatalf("integer zero corrupted: value=%d representable=%v", value, ok)
	}
	if v2.Checks[1].Extras.ExitCode != nil {
		t.Fatalf("expected null exit_code to remain nil")
	}
}

func TestDecodeV2TotalsAbsent(t *testing.T) {
	// v2-minimal has no test totals; the optional fields must
	// remain absent (nil pointer), not synthetic zero.
	data := readFixture(t, "testdata/valid/v2-minimal.json")
	res := Decode(strings.NewReader(string(data)))
	if !res.Success() {
		t.Fatalf("expected success, got diags=%v", res.Diagnostics)
	}
	v2, _ := res.Document.V2()
	c := v2.Checks[0]
	if c.Total != nil || c.PassCount != nil || c.FailCount != nil ||
		c.SkipCount != nil || c.UnavailableCount != nil {
		t.Errorf("v2-minimal should have no totals, got %+v", c)
	}
}
