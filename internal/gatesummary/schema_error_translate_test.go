package gatesummary

import (
	"strings"
	"testing"
)

func TestTranslateNilRootIsInternal(t *testing.T) {
	tr := schemaErrorTranslator{}
	ds := tr.translate()
	if len(ds) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(ds))
	}
	if ds[0].Code != CodeInternal {
		t.Fatalf("expected %s, got %s", CodeInternal, ds[0].Code)
	}
}

func TestTranslateRequiredMissing(t *testing.T) {
	set, err := schemas()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	data := []byte(`{"schema_version": 2}`)
	if verr := validateAgainstSchema(set.v2, data); verr == nil {
		t.Fatal("expected validation error")
	} else {
		tr := schemaErrorTranslator{root: mustValidationError(t, verr)}
		ds := tr.translate()
		if len(ds) == 0 {
			t.Fatal("expected diagnostics")
		}
		// Required fanout produces one diagnostic per missing
		// property; every code must be GS_REQUIRED_FIELD_MISSING.
		for _, d := range ds {
			if d.Code != CodeRequiredFieldMissing {
				t.Errorf("expected %s, got %s", CodeRequiredFieldMissing, d.Code)
			}
		}
	}
}

func TestTranslateAdditionalProperties(t *testing.T) {
	set, err := schemas()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	// v1 with v2-only field "tool" present at top level is OK
	// (tool is in v1). We need a v2 with an unknown field.
	data := []byte(`{
		"schema_version": 2,
		"generated_at": "2026-07-19T08:43:26Z",
		"scope_id": "X",
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
		"checks": [],
		"unknown_field": 1
	}`)
	if verr := validateAgainstSchema(set.v2, data); verr == nil {
		t.Fatal("expected validation error")
	} else {
		tr := schemaErrorTranslator{root: mustValidationError(t, verr)}
		ds := tr.translate()
		var found bool
		for _, d := range ds {
			if d.Code == CodeUnknownField {
				found = true
				if !strings.Contains(d.Path, "unknown_field") {
					t.Errorf("expected path to mention unknown_field, got %q", d.Path)
				}
			}
		}
		if !found {
			t.Fatalf("expected %s, got %+v", CodeUnknownField, ds)
		}
	}
}

func TestTranslateInvalidStatus(t *testing.T) {
	set, err := schemas()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	// bad-status enum triggers GS_INVALID_STATUS
	data := readFixture(t, "testdata/invalid/v2-bad-status-enum.json")
	if verr := validateAgainstSchema(set.v2, data); verr == nil {
		t.Fatal("expected validation error")
	} else {
		tr := schemaErrorTranslator{root: mustValidationError(t, verr)}
		ds := tr.translate()
		var found bool
		for _, d := range ds {
			if d.Code == CodeInvalidStatus {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %s, got %+v", CodeInvalidStatus, ds)
		}
	}
}

func TestTranslatePartialTestTotals(t *testing.T) {
	set, err := schemas()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	data := readFixture(t, "testdata/invalid/v2-partial-test-totals.json")
	if verr := validateAgainstSchema(set.v2, data); verr == nil {
		t.Fatal("expected validation error")
	} else {
		tr := schemaErrorTranslator{root: mustValidationError(t, verr)}
		ds := tr.translate()
		count := 0
		for _, d := range ds {
			if d.Code == CodePartialTestTotals {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("expected exactly one %s, got %d (all=%+v)",
				CodePartialTestTotals, count, ds)
		}
	}
}

func TestTranslateLowercaseLifecycle(t *testing.T) {
	set, err := schemas()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	data := readFixture(t, "testdata/invalid/v2-lower-lifecycle.json")
	if verr := validateAgainstSchema(set.v2, data); verr == nil {
		t.Fatal("expected validation error")
	} else {
		tr := schemaErrorTranslator{root: mustValidationError(t, verr)}
		ds := tr.translate()
		var found bool
		for _, d := range ds {
			if d.Code == CodeInvalidStatus {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %s, got %+v", CodeInvalidStatus, ds)
		}
	}
}

func TestTranslateInvalidTimestamp(t *testing.T) {
	set, err := schemas()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	data := readFixture(t, "testdata/invalid/v2-invalid-timestamp.json")
	if verr := validateAgainstSchema(set.v2, data); verr == nil {
		t.Fatal("expected validation error")
	} else {
		tr := schemaErrorTranslator{root: mustValidationError(t, verr)}
		ds := tr.translate()
		var found bool
		for _, d := range ds {
			if d.Code == CodeInvalidTimestamp {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %s, got %+v", CodeInvalidTimestamp, ds)
		}
	}
}

func TestTranslateUppercaseOID(t *testing.T) {
	set, err := schemas()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	data := readFixture(t, "testdata/invalid/v2-uppercase-oid.json")
	if verr := validateAgainstSchema(set.v2, data); verr == nil {
		t.Fatal("expected validation error")
	} else {
		tr := schemaErrorTranslator{root: mustValidationError(t, verr)}
		ds := tr.translate()
		var found bool
		for _, d := range ds {
			if d.Code == CodeInvalidOID {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %s, got %+v", CodeInvalidOID, ds)
		}
	}
}

func TestTranslateInvalidOutputHash(t *testing.T) {
	set, err := schemas()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	data := readFixture(t, "testdata/invalid/v2-invalid-hash.json")
	if verr := validateAgainstSchema(set.v2, data); verr == nil {
		t.Fatal("expected validation error")
	} else {
		tr := schemaErrorTranslator{root: mustValidationError(t, verr)}
		ds := tr.translate()
		var found bool
		for _, d := range ds {
			if d.Code == CodeInvalidOutputHash {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %s, got %+v", CodeInvalidOutputHash, ds)
		}
	}
}

func TestTranslateNegativeDuration(t *testing.T) {
	set, err := schemas()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	data := readFixture(t, "testdata/invalid/v2-negative-duration.json")
	if verr := validateAgainstSchema(set.v2, data); verr == nil {
		t.Fatal("expected validation error")
	} else {
		tr := schemaErrorTranslator{root: mustValidationError(t, verr)}
		ds := tr.translate()
		var found bool
		for _, d := range ds {
			if d.Code == CodeInvalidDuration {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %s, got %+v", CodeInvalidDuration, ds)
		}
	}
}
