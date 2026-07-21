package gatesummary

import (
	"reflect"
	"strings"
	"testing"
)

// TestNormalizationDiagnosticOrderingLifecycleAndCheck verifies
// that lifecycle/overall diagnostics co-occur with per-check
// diagnostics in precedence-sorted order.
func TestNormalizationDiagnosticOrderingLifecycleAndCheck(t *testing.T) {
	wire := `{
		"schema_version": 2,
		"generated_at": "2026-07-20T12:00:00Z",
		"scope_id": "ACT-LC-CHK",
		"scope_status": "CLOSED",
		"scope_disposition": "lc+chk ordering",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "root",
		"overall_status": "pass",
		"overall_disposition": "lc+chk ordering",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": false,
		"checks": [
			{
				"name": "x",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "e",
				"detail": "d",
				"extras": {
					"argv": [],
					"exit_code": 7,
					"duration_ms": 0,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			},
			{
				"name": "y",
				"scope": "ROOT",
				"status": "fail",
				"evidence": "e",
				"detail": "d",
				"extras": {
					"argv": [],
					"exit_code": 1,
					"duration_ms": 0,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			}
		]
	}`
	dec := Decode(strings.NewReader(wire))
	if !dec.Success() {
		t.Fatalf("decode failed: %v", dec.Diagnostics)
	}
	norm := Normalize(dec.Document)
	if norm.Success() {
		t.Fatal("normalize unexpectedly succeeded")
	}
	got := projectDiagnostics(norm.Diagnostics)
	// Rank order: pass_mismatch (16) < overall_mismatch (24) <
	// scope_closed_dirty (25).
	want := []diagnosticProjection{
		{Code: CodePassExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
		{Code: CodeOverallStatusMismatch, Path: "/overall_status"},
		{Code: CodeScopeClosedDirtyWorktree, Path: "/worktree_clean_after"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordering = %#v, want %#v", got, want)
	}
}

// TestNormalizationDiagnosticOrderingRepeated verifies that
// repeated runs of Normalize against the same input produce
// identical diagnostic order.
func TestNormalizationDiagnosticOrderingRepeated(t *testing.T) {
	checks := "[" +
		invalidPassCheckForTest("dup", "1") + "," +
		passCheckForTest("dup") + "]"
	data := validV2DocumentForTest(checks)
	dec := Decode(strings.NewReader(data))
	if !dec.Success() {
		t.Fatalf("decode failed: %v", dec.Diagnostics)
	}
	want := []diagnosticProjection{
		{Code: CodeDuplicateCheckName, Path: "/checks/1/name"},
		{Code: CodePassExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
	}
	for i := 0; i < 10; i++ {
		norm := Normalize(dec.Document)
		if norm.Success() {
			t.Fatalf("iteration %d: normalize unexpectedly succeeded", i)
		}
		got := projectDiagnostics(norm.Diagnostics)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: ordering = %#v, want %#v",
				i, got, want)
		}
	}
}

// TestNormalizationDiagnosticOrderingPreservesPrecedenceAuthority
// is a structural meta-test: it walks the production codePrecedence
// map and asserts structural invariants without duplicating it.
// The production map is the single source of truth; tests must
// never reproduce the code→rank mapping.
func TestNormalizationDiagnosticOrderingPreservesPrecedenceAuthority(t *testing.T) {
	// Structural invariants: rank uniqueness.
	seen := make(map[int]string, len(codePrecedence))
	for code, rank := range codePrecedence {
		if code == "" {
			t.Fatalf("codePrecedence contains empty code string")
		}
		if rank <= 0 {
			t.Fatalf("code %q has non-positive rank %d", code, rank)
		}
		if existing, ok := seen[rank]; ok {
			t.Fatalf("rank %d assigned to both %q and %q", rank, existing, code)
		}
		seen[rank] = code
	}
	// At least one code must be present (sanity).
	if len(codePrecedence) < 27 {
		t.Fatalf("codePrecedence has %d entries, want >= 27",
			len(codePrecedence))
	}
}
