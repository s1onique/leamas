// Package main provides factory summary tests.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/gate"
)

func TestValidateFastLaneSummary_Valid(t *testing.T) {
	s := &fastLaneSummary{SchemaVersion: 1, OverallStatus: "pass"}
	if err := validateFastLaneSummary(s); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestValidateFastLaneSummary_Missing(t *testing.T) {
	if err := validateFastLaneSummary(nil); err != ErrMissingFastSummary {
		t.Errorf("expected ErrMissingFastSummary, got %v", err)
	}
}

func TestValidateFastLaneSummary_WrongSchema(t *testing.T) {
	s := &fastLaneSummary{SchemaVersion: 0, OverallStatus: "pass"}
	if err := validateFastLaneSummary(s); err == nil {
		t.Error("expected error for wrong schema")
	}
}

func TestValidateFastLaneSummary_InvalidStatus(t *testing.T) {
	s := &fastLaneSummary{SchemaVersion: 1, OverallStatus: "invalid"}
	if err := validateFastLaneSummary(s); err != ErrInvalidFastStatus {
		t.Errorf("expected ErrInvalidFastStatus, got %v", err)
	}
}

func TestValidateLongLaneSummary_Valid(t *testing.T) {
	s := &testLongSummary{
		SchemaVersion: 1,
		Total:         2,
		Passed:        1,
		Failed:        1,
		Tests: []testLongResult{
			{ID: "LT-001", Passed: true},
			{ID: "LT-002", Passed: false},
		},
	}
	if err := validateLongLaneSummary(s); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestValidateLongLaneSummary_Missing(t *testing.T) {
	if err := validateLongLaneSummary(nil); err != ErrMissingLongSummary {
		t.Errorf("expected ErrMissingLongSummary, got %v", err)
	}
}

func TestValidateLongLaneSummary_ZeroTotal(t *testing.T) {
	s := &testLongSummary{SchemaVersion: 1, Total: 0, Tests: []testLongResult{}}
	if err := validateLongLaneSummary(s); err != ErrInvalidLongTotal {
		t.Errorf("expected ErrInvalidLongTotal, got %v", err)
	}
}

func TestValidateLongLaneSummary_CountMismatch(t *testing.T) {
	s := &testLongSummary{
		SchemaVersion: 1,
		Total:         2,
		Passed:        1,
		Failed:        0, // mismatch: 1+0 != 2
		Tests: []testLongResult{
			{ID: "LT-001", Passed: true},
			{ID: "LT-002", Passed: true}, // wrong count, but length matches
		},
	}
	if err := validateLongLaneSummary(s); err != ErrLongCountMismatch {
		t.Errorf("expected ErrLongCountMismatch, got %v", err)
	}
}

func TestValidateLongLaneSummary_ResultMismatch(t *testing.T) {
	s := &testLongSummary{
		SchemaVersion: 1,
		Total:         2,
		Passed:        2, // claims 2 passed
		Failed:        0,
		Tests: []testLongResult{
			{ID: "LT-001", Passed: true},
			{ID: "LT-002", Passed: false}, // but one actually failed
		},
	}
	if err := validateLongLaneSummary(s); err != ErrTestResultMismatch {
		t.Errorf("expected ErrTestResultMismatch, got %v", err)
	}
}

func TestValidateLongLaneSummary_LengthMismatch(t *testing.T) {
	s := &testLongSummary{
		SchemaVersion: 1,
		Total:         3, // claims 3 tests
		Passed:        3,
		Failed:        0,
		Tests: []testLongResult{ // but only 2 results
			{ID: "LT-001", Passed: true},
			{ID: "LT-002", Passed: true},
		},
	}
	if err := validateLongLaneSummary(s); err == nil {
		t.Error("expected error for length mismatch")
	}
}

func TestWriteAggregateAfterFastFailure(t *testing.T) {
	dir := t.TempDir()
	origCwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origCwd)

	// Write a dummy fast summary to verify it's overwritten
	fastPath := filepath.Join(dir, ".factory", "gate-fast-summary.json")
	os.MkdirAll(filepath.Dir(fastPath), 0755)
	fastSummary := gate.GateSummary{
		SchemaVersion: 1,
		OverallStatus: "fail",
		Checks:        []gate.Check{{Name: "fast-lane", Status: gate.CheckStatusFail}},
	}
	data, _ := json.Marshal(fastSummary)
	os.WriteFile(fastPath, data, 0644)

	// Write the aggregate after fast failure
	if err := writeAggregateAfterFastFailure(); err != nil {
		t.Fatalf("writeAggregateAfterFastFailure: %v", err)
	}

	// Verify the aggregate was written
	aggPath := filepath.Join(dir, ".factory", "gate-summary.json")
	aggData, err := os.ReadFile(aggPath)
	if err != nil {
		t.Fatalf("failed to read aggregate: %v", err)
	}

	var agg gate.GateSummary
	if err := json.Unmarshal(aggData, &agg); err != nil {
		t.Fatalf("failed to parse aggregate: %v", err)
	}

	if agg.OverallStatus != "fail" {
		t.Errorf("expected overall_status=fail, got %s", agg.OverallStatus)
	}
	if len(agg.Checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(agg.Checks))
	}
}
