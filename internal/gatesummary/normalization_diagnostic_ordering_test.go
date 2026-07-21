package gatesummary

import (
	"reflect"
	"strings"
	"testing"
)

// TestNormalizationDiagnosticOrderingTwoPrecedenceDifferent verifies
// that diagnostics with different precedence values are ordered
// by precedence, not by JSON Pointer path.
func TestNormalizationDiagnosticOrderingTwoPrecedenceDifferent(t *testing.T) {
	checks := "[" +
		invalidPassCheckForTest("dup", "1") + "," +
		passCheckForTest("dup") + "]"
	data := validV2DocumentForTest(checks)
	dec := Decode(strings.NewReader(data))
	if !dec.Success() {
		t.Fatalf("decode failed: %v", dec.Diagnostics)
	}
	norm := Normalize(dec.Document)
	if norm.Success() {
		t.Fatal("normalize unexpectedly succeeded")
	}
	got := projectDiagnostics(norm.Diagnostics)
	want := []diagnosticProjection{
		{Code: CodeDuplicateCheckName, Path: "/checks/1/name"},
		{Code: CodePassExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordering = %#v, want %#v", got, want)
	}
}

// TestNormalizationDiagnosticOrderingMultipleValidators verifies
// that diagnostics from three or more separate semantic
// validators produce deterministic, precedence-sorted output.
func TestNormalizationDiagnosticOrderingMultipleValidators(t *testing.T) {
	checks := "[" +
		`{
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
			},
			"total": 100,
			"pass_count": 1,
			"fail_count": 0,
			"skip_count": 0,
			"unavailable_count": 0
		},` +
		`{
			"name": "x",
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
		}` +
		"]"
	data := validV2DocumentForTest(checks)
	dec := Decode(strings.NewReader(data))
	if !dec.Success() {
		t.Fatalf("decode failed: %v", dec.Diagnostics)
	}
	norm := Normalize(dec.Document)
	if norm.Success() {
		t.Fatal("normalize unexpectedly succeeded")
	}
	got := projectDiagnostics(norm.Diagnostics)
	want := []diagnosticProjection{
		{Code: CodeDuplicateCheckName, Path: "/checks/1/name"},
		{Code: CodePassExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
		{Code: CodeTestTotalMismatch, Path: "/checks/0"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordering = %#v, want %#v", got, want)
	}
}

// TestNormalizationDiagnosticOrderingSameCodeDifferentPath
// verifies that two diagnostics with the same code but at
// different paths are sorted by path.
func TestNormalizationDiagnosticOrderingSameCodeDifferentPath(t *testing.T) {
	checks := "[" +
		`{
			"name": "a",
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
			},
			"total": 100,
			"pass_count": 1,
			"fail_count": 0,
			"skip_count": 0,
			"unavailable_count": 0
		},` +
		`{
			"name": "b",
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
			},
			"total": 200,
			"pass_count": 1,
			"fail_count": 0,
			"skip_count": 0,
			"unavailable_count": 0
		}` +
		"]"
	data := validV2DocumentForTest(checks)
	dec := Decode(strings.NewReader(data))
	if !dec.Success() {
		t.Fatalf("decode failed: %v", dec.Diagnostics)
	}
	norm := Normalize(dec.Document)
	if norm.Success() {
		t.Fatal("normalize unexpectedly succeeded")
	}
	got := projectDiagnostics(norm.Diagnostics)
	want := []diagnosticProjection{
		{Code: CodeTestTotalMismatch, Path: "/checks/0"},
		{Code: CodeTestTotalMismatch, Path: "/checks/1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordering = %#v, want %#v", got, want)
	}
}

// TestNormalizationDiagnosticOrderingDuplicateNames verifies
// that duplicate check names at later indices produce distinct
// index-based paths.
func TestNormalizationDiagnosticOrderingDuplicateNames(t *testing.T) {
	checks := "[" +
		`{
			"name": "dup",
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
		},` +
		`{
			"name": "dup",
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
		},` +
		`{
			"name": "dup",
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
		}` +
		"]"
	data := validV2DocumentForTest(checks)
	dec := Decode(strings.NewReader(data))
	if !dec.Success() {
		t.Fatalf("decode failed: %v", dec.Diagnostics)
	}
	norm := Normalize(dec.Document)
	if norm.Success() {
		t.Fatal("normalize unexpectedly succeeded")
	}
	got := projectDiagnostics(norm.Diagnostics)
	want := []diagnosticProjection{
		{Code: CodeDuplicateCheckName, Path: "/checks/1/name"},
		{Code: CodeDuplicateCheckName, Path: "/checks/2/name"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordering = %#v, want %#v", got, want)
	}
}

// TestNormalizationDiagnosticOrderingTotalsAndCleanliness
// verifies that totals diagnostics precede cleanliness
// diagnostics when both fire on the same document.
func TestNormalizationDiagnosticOrderingTotalsAndCleanliness(t *testing.T) {
	wire := `{
		"schema_version": 2,
		"generated_at": "2026-07-20T12:00:00Z",
		"scope_id": "ACT-ORDER",
		"scope_status": "CLOSED",
		"scope_disposition": "ordering",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "root",
		"overall_status": "pass",
		"overall_disposition": "ordering",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": false,
		"checks": [
			{
				"name": "tt",
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
				},
				"total": 100,
				"pass_count": 1,
				"fail_count": 0,
				"skip_count": 0,
				"unavailable_count": 0
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
	want := []diagnosticProjection{
		{Code: CodeTestTotalMismatch, Path: "/checks/0"},
		{Code: CodeScopeClosedDirtyWorktree, Path: "/worktree_clean_after"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordering = %#v, want %#v", got, want)
	}
}
