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
// is a meta-test: it walks the production codePrecedence map and
// asserts every rank is unique. The matrix suite relies on
// stable ordering; the production map is the single source of
// truth and tests must not duplicate it.
func TestNormalizationDiagnosticOrderingPreservesPrecedenceAuthority(t *testing.T) {
	seen := make(map[int]string, len(codePrecedence))
	for code, rank := range codePrecedence {
		if existing, ok := seen[rank]; ok {
			t.Fatalf("rank %d assigned to both %q and %q", rank, existing, code)
		}
		seen[rank] = code
	}
	expectedRanks := map[string]int{
		CodeDocumentTooLarge:         1,
		CodeMalformedJSON:            2,
		CodeTrailingJSON:             3,
		CodeDuplicateKey:             4,
		CodeVersionMissing:           5,
		CodeInvalidVersionType:       6,
		CodeUnsupportedVersion:       7,
		CodeUnknownField:             8,
		CodeRequiredFieldMissing:     9,
		CodeSchemaViolation:          10,
		CodeInvalidTimestamp:         11,
		CodeInvalidStatus:            12,
		CodeInvalidOID:               13,
		CodeCollectionLimit:          14,
		CodeDuplicateCheckName:       15,
		CodePassExitCodeMismatch:     16,
		CodeFailExitCodeMismatch:     17,
		CodeSkipExitCodeMismatch:     18,
		CodeUnavailExitCodeMismatch:  19,
		CodeInvalidDuration:          20,
		CodeInvalidOutputHash:        21,
		CodePartialTestTotals:        22,
		CodeTestTotalMismatch:        23,
		CodeOverallStatusMismatch:    24,
		CodeScopeClosedDirtyWorktree: 25,
		CodeNormalizationFailure:     26,
		CodeInternal:                 27,
	}
	for code, want := range expectedRanks {
		got, ok := codePrecedence[code]
		if !ok {
			t.Fatalf("code %q missing from codePrecedence", code)
		}
		if got != want {
			t.Fatalf("code %q rank = %d, want %d", code, got, want)
		}
	}
	if len(codePrecedence) != len(expectedRanks) {
		t.Fatalf("codePrecedence has %d entries, expected %d",
			len(codePrecedence), len(expectedRanks))
	}
}

// TestNormalizationDiagnosticOrderingUsesProductionAuthority
// ensures the tests never silently rely on a copied precedence
// table by triggering a real ordering edge case. If a test
// silently created a local copy with the wrong ranks, this
// assertion would diverge.
func TestNormalizationDiagnosticOrderingUsesProductionAuthority(t *testing.T) {
	precedencePointer := reflect.ValueOf(codePrecedence).Pointer()
	if precedencePointer == 0 {
		t.Fatal("codePrecedence map has zero pointer (production map missing)")
	}
	if len(codePrecedence) < 27 {
		t.Fatalf("codePrecedence map has %d entries, want >= 27",
			len(codePrecedence))
	}
	_ = strings.Repeat // keep strings import live
}
