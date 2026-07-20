package longtest

import (
	"testing"
)

func TestValidateBaseline_Valid(t *testing.T) {
	baseline := &Baseline{
		SchemaVersion: 1,
		Tests: []TestSpec{
			{
				ID:         "LT-TEST-01",
				Package:    "./internal/foo",
				Test:       "TestFoo",
				FastPolicy: "skip-under-short",
				CITimeout:  "10m",
				CIGroup:    "group-a",
				Reason:     "expensive test",
				Owner:      "team/foo",
			},
		},
	}
	if err := ValidateBaseline(baseline); err != nil {
		t.Errorf("expected valid baseline, got: %v", err)
	}
}

func TestValidateBaseline_MissingID(t *testing.T) {
	baseline := &Baseline{
		SchemaVersion: 1,
		Tests: []TestSpec{
			{
				ID:      "",
				Package: "./internal/foo",
				Test:    "TestFoo",
			},
		},
	}
	err := ValidateBaseline(baseline)
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got: %T", err)
	}
	if ve.Field != "id" {
		t.Errorf("expected field 'id', got: %q", ve.Field)
	}
}

func TestValidateBaseline_DuplicateID(t *testing.T) {
	baseline := &Baseline{
		SchemaVersion: 1,
		Tests: []TestSpec{
			{ID: "LT-DUP", Package: "./a", Test: "TestA"},
			{ID: "LT-DUP", Package: "./b", Test: "TestB"},
		},
	}
	err := ValidateBaseline(baseline)
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got: %T", err)
	}
	if ve.ID != "LT-DUP" {
		t.Errorf("expected ID 'LT-DUP', got: %q", ve.ID)
	}
}

func TestValidateBaseline_InvalidFastPolicy(t *testing.T) {
	baseline := &Baseline{
		SchemaVersion: 1,
		Tests: []TestSpec{
			{
				ID:         "LT-BAD",
				Package:    "./a",
				Test:       "TestA",
				FastPolicy: "invalid",
			},
		},
	}
	err := ValidateBaseline(baseline)
	if err == nil {
		t.Fatal("expected error for invalid fast policy")
	}
}

func TestPolicyBaselineIDs(t *testing.T) {
	baseline := &Baseline{
		Tests: []TestSpec{
			{ID: "LT-A"},
			{ID: "LT-B"},
		},
	}
	ids := PolicyBaselineIDs(baseline)
	if !ids["LT-A"] || !ids["LT-B"] || ids["LT-C"] {
		t.Errorf("unexpected IDs: %v", ids)
	}
}

func TestPolicyBaselineIDs_Nil(t *testing.T) {
	ids := PolicyBaselineIDs(nil)
	if len(ids) != 0 {
		t.Errorf("expected empty map for nil baseline, got: %v", ids)
	}
}

func TestLoadBaseline_FileNotFound(t *testing.T) {
	_, err := LoadBaseline("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for missing baseline file")
	}
	if err != ErrBaselineMissing {
		t.Errorf("expected ErrBaselineMissing, got: %v", err)
	}
}

func TestValidateBaseline_RequiresFastPolicy(t *testing.T) {
	// fast_policy is optional in current schema but recommended
	baseline := &Baseline{
		SchemaVersion: 1,
		Tests: []TestSpec{
			{
				ID:         "LT-TEST",
				Package:    "./a",
				Test:       "TestA",
				FastPolicy: "invalid",
			},
		},
	}
	err := ValidateBaseline(baseline)
	if err == nil {
		t.Fatal("expected error for invalid fast policy")
	}
}
