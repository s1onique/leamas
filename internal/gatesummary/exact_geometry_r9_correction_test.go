package gatesummary

import (
	"io"
	"strconv"
	"strings"
	"testing"
)

// checkForTest is the internal builder for all check types.
// This is a low-level primitive that accepts any status/exitCode.
// Valid helpers below enforce closed-world constraints.
func checkForTest(name, status, exitCode string) string {
	return `{
		"name": "` + name + `",
		"scope": "ROOT",
		"status": "` + status + `",
		"evidence": "e",
		"detail": "d",
		"extras": {
			"argv": [],
			"exit_code": ` + exitCode + `,
			"duration_ms": 0,
			"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		}
	}`
}

// validV2DocumentForTest constructs a valid V2 document with the given checks JSON.
func validV2DocumentForTest(checksJSON string) string {
	return `{
		"schema_version": 2,
		"generated_at": "2026-07-20T12:00:00Z",
		"scope_id": "ACT-TEST",
		"scope_status": "OPEN",
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
		"checks": ` + checksJSON + `
	}`
}

// passCheckForTest creates a valid pass check with exit_code: 0.
func passCheckForTest(name string) string {
	return checkForTest(name, "pass", "0")
}

// failWithoutExitForTest creates a valid fail check with exit_code: null
// (infrastructure failure, no exit code available).
func failWithoutExitForTest(name string) string {
	return checkForTest(name, "fail", "null")
}

// failNonzeroForTest creates a valid fail check with a non-zero exit code.
// Panics if exitCode is zero.
func failNonzeroForTest(name string, exitCode int64) string {
	if exitCode == 0 {
		panic("failNonzeroForTest requires a non-zero exit code")
	}
	return checkForTest(name, "fail", strconv.FormatInt(exitCode, 10))
}

// skipCheckForTest creates a valid skip check with exit_code: null.
func skipCheckForTest(name string) string {
	return checkForTest(name, "skip", "null")
}

// unavailableCheckForTest creates a valid unavailable check with exit_code: null.
func unavailableCheckForTest(name string) string {
	return checkForTest(name, "unavailable", "null")
}

// invalidPassCheckForTest creates an intentionally invalid pass check for testing
// normalization failures. Pass checks should have exit_code: 0; this creates one
// with a non-zero exit code.
func invalidPassCheckForTest(name string, exitCode string) string {
	return checkForTest(name, "pass", exitCode)
}

// validV2DocumentWithOverall constructs a valid V2 document with custom overall status.
func validV2DocumentWithOverall(checksJSON, overallStatus string) string {
	return `{
		"schema_version": 2,
		"generated_at": "2026-07-20T12:00:00Z",
		"scope_id": "ACT-TEST",
		"scope_status": "OPEN",
		"scope_disposition": "d",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "d",
		"overall_status": "` + overallStatus + `",
		"overall_disposition": "d",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": ` + checksJSON + `
	}`
}

// consumeForTest demonstrates the required caller sequence.
func consumeForTest(r io.Reader, normalize func(Document) NormalizationResult) bool {
	decoded := Decode(r)
	if !decoded.Success() {
		return false
	}
	normalize(decoded.Document)
	return true
}

// TestValidV2BuilderAllStatuses verifies the builder creates semantically valid checks
// for all status types: pass, fail, skip, unavailable.
func TestValidV2BuilderAllStatuses(t *testing.T) {
	tests := []struct {
		name          string
		check         string
		overallStatus string
		wantDecode    bool
		wantNormPass  bool
	}{
		{"pass", passCheckForTest("p"), "pass", true, true},
		{"fail without exit (infra failure)", failWithoutExitForTest("f"), "fail", true, true},
		{"fail with nonzero exit", failNonzeroForTest("f", 1), "fail", true, true},
		{"skip", skipCheckForTest("s"), "unavailable", true, true},
		{"unavailable", unavailableCheckForTest("u"), "unavailable", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := validV2DocumentWithOverall("["+tt.check+"]", tt.overallStatus)
			decoded := Decode(strings.NewReader(data))
			if tt.wantDecode && !decoded.Success() {
				t.Errorf("document should decode successfully: %v", decoded.Diagnostics)
			}
			if !tt.wantDecode && decoded.Success() {
				t.Errorf("document should fail decode")
			}

			if decoded.Success() {
				normalized := Normalize(decoded.Document)
				if tt.wantNormPass && !normalized.Success() {
					t.Errorf("check should normalize successfully: %v", normalized.Diagnostics)
				}
				if !tt.wantNormPass && normalized.Success() {
					t.Errorf("check should fail normalization")
				}
			}
		})
	}
}

// TestDiagnosticPrecedenceEndToEnd verifies that diagnostics are ordered by precedence
// regardless of JSON Pointer path ordering. GS_DUPLICATE_CHECK_NAME (rank 15) must
// precede GS_PASS_EXIT_CODE_MISMATCH (rank 16), even though /checks/0/... sorts
// before /checks/1/... lexically.
func TestDiagnosticPrecedenceEndToEnd(t *testing.T) {
	// Two pass checks with the same name triggers GS_DUPLICATE_CHECK_NAME.
	// First check has nonzero exit_code which triggers GS_PASS_EXIT_CODE_MISMATCH.
	// Despite /checks/0/extras/exit_code sorting before /checks/1/name lexically,
	// the duplicate-name diagnostic must appear first because rank 15 < rank 16.
	checks := "[" +
		invalidPassCheckForTest("dup", "1") + "," +
		passCheckForTest("dup") +
		"]"
	data := validV2DocumentForTest(checks)

	decoded := Decode(strings.NewReader(data))
	if !decoded.Success() {
		t.Fatalf("document with duplicate names but valid structure should decode: %v", decoded.Diagnostics)
	}

	normalized := Normalize(decoded.Document)
	if normalized.Success() {
		t.Fatal("duplicate check names should cause normalization failure")
	}

	// Expect exactly two diagnostics
	if len(normalized.Diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d: %v", len(normalized.Diagnostics), normalized.Diagnostics)
	}

	// Verify the exact identities and ordering
	want := []struct {
		Code string
		Path string
	}{
		{CodeDuplicateCheckName, "/checks/1/name"},
		{CodePassExitCodeMismatch, "/checks/0/extras/exit_code"},
	}

	for i, w := range want {
		if normalized.Diagnostics[i].Code != w.Code {
			t.Errorf("diagnostic[%d] code = %s, want %s", i, normalized.Diagnostics[i].Code, w.Code)
		}
		if normalized.Diagnostics[i].Path != w.Path {
			t.Errorf("diagnostic[%d] path = %s, want %s", i, normalized.Diagnostics[i].Path, w.Path)
		}
	}
}

// TestStructuralDecodeRejection verifies the complete decode-rejection contract.
func TestStructuralDecodeRejection(t *testing.T) {
	data := readFixture(t, "testdata/invalid/v2-truncated.json")

	decoded := Decode(strings.NewReader(string(data)))

	// Success() must be false
	if decoded.Success() {
		t.Fatal("truncated JSON should fail decode")
	}

	// Err must be nil for ordinary invalid input
	if decoded.Err != nil {
		t.Fatalf("ordinary structural rejection should have nil Err, got: %v", decoded.Err)
	}

	// Document must have version zero (no usable version)
	if decoded.Document.Version() != 0 {
		t.Fatalf("rejected input returned usable document version %d, want 0", decoded.Document.Version())
	}

	// Wire-stage diagnostics must be present
	if len(decoded.Diagnostics) == 0 {
		t.Fatal("expected wire-stage diagnostics from structural rejection")
	}

	// Verify exact diagnostic identity
	want := []struct {
		Code string
		Path string
	}{
		{CodeMalformedJSON, "/scope_id"},
	}

	for i, w := range want {
		if i >= len(decoded.Diagnostics) {
			t.Fatalf("expected at least %d diagnostics, got %d", i+1, len(decoded.Diagnostics))
		}
		if decoded.Diagnostics[i].Code != w.Code {
			t.Errorf("diagnostic[%d] code = %s, want %s", i, decoded.Diagnostics[i].Code, w.Code)
		}
		if decoded.Diagnostics[i].Path != w.Path {
			t.Errorf("diagnostic[%d] path = %s, want %s", i, decoded.Diagnostics[i].Path, w.Path)
		}
	}

	// Verify no unexpected diagnostics
	if len(decoded.Diagnostics) > len(want) {
		t.Errorf("expected exactly %d diagnostic(s), got %d: %v", len(want), len(decoded.Diagnostics), decoded.Diagnostics)
	}
}

// TestCallerGatingBothBranches verifies both caller-gating branches executably.
func TestCallerGatingBothBranches(t *testing.T) {
	// Subtest 1: valid input → normalization invoked
	t.Run("valid input invokes normalize", func(t *testing.T) {
		checks := "[" + passCheckForTest("single") + "]"
		data := validV2DocumentForTest(checks)
		normalizeCalled := false
		normalize := func(doc Document) NormalizationResult {
			normalizeCalled = true
			return Normalize(doc)
		}
		got := consumeForTest(strings.NewReader(data), normalize)
		if !got {
			t.Error("expected normalization to be invoked for valid input")
		}
		if !normalizeCalled {
			t.Error("normalize callback should have been called")
		}
	})

	// Subtest 2: invalid input → normalization NOT invoked
	t.Run("invalid input skips normalize", func(t *testing.T) {
		data := `{"schema_version": 2, "truncated`
		normalizeCalled := false
		normalize := func(doc Document) NormalizationResult {
			normalizeCalled = true
			return Normalize(doc)
		}
		got := consumeForTest(strings.NewReader(data), normalize)
		if got {
			t.Error("expected normalization NOT to be invoked for invalid input")
		}
		if normalizeCalled {
			t.Error("normalize callback should not have been called for rejected input")
		}
	})
}
