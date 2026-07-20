package gatesummary

import (
	"strings"
	"testing"
)

// TestSealedDocumentValidation tests that Normalize rejects invalid sealed Document states.
func TestSealedDocumentValidation(t *testing.T) {
	// Neither version populated
	t.Run("neither version populated", func(t *testing.T) {
		doc := Document{}
		result := Normalize(doc)
		if result.Success() {
			t.Error("expected failure for zero-value document")
		}
		if result.Err == nil {
			t.Error("expected non-nil error")
		}
	})

	// Both versions populated (impossible sealed state)
	t.Run("both versions populated", func(t *testing.T) {
		doc := Document{
			v1: &V1Summary{},
			v2: &V2Summary{},
		}
		result := Normalize(doc)
		if result.Success() {
			t.Error("expected failure for dual-populated document")
		}
		if result.Err == nil {
			t.Error("expected non-nil error")
		}
	})
}

// TestInvalidIntegerNormalization tests that invalid WireInteger values fail normalization.
func TestInvalidIntegerNormalization(t *testing.T) {
	t.Run("newIntegerFromWire empty", func(t *testing.T) {
		var w WireInteger
		_, err := newIntegerFromWire(w)
		if err == nil {
			t.Error("newIntegerFromWire: expected error for empty wire integer")
		}
	})

	t.Run("newIntegerFromWire invalid string", func(t *testing.T) {
		var w WireInteger
		_ = w.UnmarshalJSON([]byte("not-a-number"))
		_, err := newIntegerFromWire(w)
		if err == nil {
			t.Error("newIntegerFromWire: expected error for invalid integer string")
		}
	})

	t.Run("newIntegerFromWire non_JSON lexical", func(t *testing.T) {
		// These forms pass big.Int.SetString but are invalid JSON integer spellings.
		for _, raw := range []string{"+1", "01", "-01", "00", "-00"} {
			var w WireInteger
			_ = w.UnmarshalJSON([]byte(raw))
			_, err := newIntegerFromWire(w)
			if err == nil {
				t.Errorf("newIntegerFromWire(%q): expected error for non-JSON lexical form", raw)
			}
		}
	})

	t.Run("newIntegerFromWire valid JSON", func(t *testing.T) {
		// Valid JSON integer spellings that must be accepted.
		for _, raw := range []string{"0", "-0", "1", "-1", "123456789012345678901234567890"} {
			var w WireInteger
			_ = w.UnmarshalJSON([]byte(raw))
			_, err := newIntegerFromWire(w)
			if err != nil {
				t.Errorf("newIntegerFromWire(%q): unexpected error: %v", raw, err)
			}
		}
	})
}

// TestDuplicateNameMultipleOccurrences tests that three identical names produce
// two distinct diagnostics: /checks/1/name and /checks/2/name.
func TestDuplicateNameMultipleOccurrences(t *testing.T) {
	// Create a v2 document with three checks named "duplicate"
	doc := `{
		"schema_version": 2,
		"generated_at": "2026-07-19T08:43:26Z",
		"scope_id": "TEST",
		"scope_status": "OPEN",
		"scope_disposition": "test",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "root",
		"overall_status": "pass",
		"overall_disposition": "ok",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
			{
				"name": "duplicate",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "x",
				"detail": "y",
				"extras": {
					"argv": ["x"],
					"exit_code": 0,
					"duration_ms": 0,
					"stdout_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			},
			{
				"name": "duplicate",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "x",
				"detail": "y",
				"extras": {
					"argv": ["x"],
					"exit_code": 0,
					"duration_ms": 0,
					"stdout_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			},
			{
				"name": "duplicate",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "x",
				"detail": "y",
				"extras": {
					"argv": ["x"],
					"exit_code": 0,
					"duration_ms": 0,
					"stdout_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			}
		]
	}`

	result := Decode(strings.NewReader(doc))
	if !result.Success() {
		t.Fatalf("decode failed: %v", result.Diagnostics)
	}

	normResult := Normalize(result.Document)
	if normResult.Success() {
		t.Fatal("expected normalization failure for duplicate names")
	}

	// Find all GS_DUPLICATE_CHECK_NAME diagnostics
	var dupDiags []Diagnostic
	for _, d := range normResult.Diagnostics {
		if d.Code == CodeDuplicateCheckName {
			dupDiags = append(dupDiags, d)
		}
	}

	if len(dupDiags) != 2 {
		t.Errorf("expected 2 duplicate diagnostics, got %d: %v", len(dupDiags), dupDiags)
	}

	// Verify the paths are distinct and use index format
	expectedPaths := map[string]bool{
		"/checks/1/name": true,
		"/checks/2/name": true,
	}
	for _, d := range dupDiags {
		if !expectedPaths[d.Path] {
			t.Errorf("unexpected path %q, expected /checks/1/name or /checks/2/name", d.Path)
		}
	}
}

// TestIntegerValidationEdgeCases tests edge cases in integer validation.
func TestIntegerValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"zero", "0", false},
		{"negative_zero", "-0", false},
		{"positive", "42", false},
		{"negative", "-42", false},
		{"large", "123456789012345678901234567890", false},
		{"float", "3.14", true},
		{"hex", "0x10", true},
		{"scientific", "1e10", true},
		{"leading_plus", "+1", true},
		{"leading_zero", "01", true},
		{"negative_leading_zero", "-01", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w WireInteger
			if tt.input != "" {
				_ = w.UnmarshalJSON([]byte(tt.input))
			}
			_, err := newIntegerFromWire(w)
			if (err != nil) != tt.wantErr {
				t.Errorf("newIntegerFromWire(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
