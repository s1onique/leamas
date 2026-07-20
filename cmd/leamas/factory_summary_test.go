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
	// Should have exactly 2 lane checks: fast-lane (fail) and long-lane (skip)
	if len(agg.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(agg.Checks))
	}
	// First check is fast-lane fail
	if agg.Checks[0].Name != "fast-lane" || agg.Checks[0].Status != gate.CheckStatusFail {
		t.Errorf("first check should be fast-lane fail")
	}
	// Second check is long-lane skip (not fail)
	if agg.Checks[1].Name != "long-lane" || agg.Checks[1].Status != "skip" {
		t.Errorf("second check should be long-lane skip, got %s", agg.Checks[1].Status)
	}
}

// writeTestSummaries writes both fast and long summaries for testing.
func writeTestSummaries(t *testing.T, dir, fastStatus, longStatus string, longFailed int) {
	fastPath := filepath.Join(dir, ".factory", "gate-fast-summary.json")
	os.MkdirAll(filepath.Dir(fastPath), 0755)
	fastSummary := gate.GateSummary{
		SchemaVersion: 1,
		OverallStatus: fastStatus,
	}
	data, _ := json.Marshal(fastSummary)
	os.WriteFile(fastPath, data, 0644)

	longPath := filepath.Join(dir, ".factory", "gate-long-summary.json")
	longSummary := testLongSummary{
		SchemaVersion: 1,
		Total:        2,
		Passed:       2 - longFailed,
		Failed:       longFailed,
		Tests:        []testLongResult{{ID: "LT-001", Passed: longFailed == 0}, {ID: "LT-002", Passed: true}},
	}
	data, _ = json.Marshal(longSummary)
	os.WriteFile(longPath, data, 0644)
}

func TestWriteAggregateForFullMode_BothPass(t *testing.T) {
	dir := t.TempDir()
	origCwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origCwd)

	writeTestSummaries(t, dir, "pass", "pass", 0)

	if err := writeAggregateForFullMode(); err != nil {
		t.Fatalf("writeAggregateForFullMode: %v", err)
	}

	aggPath := filepath.Join(dir, ".factory", "gate-summary.json")
	data, _ := os.ReadFile(aggPath)
	var agg gate.GateSummary
	json.Unmarshal(data, &agg)

	// Both lanes pass → overall must pass
	if agg.OverallStatus != "pass" {
		t.Errorf("both pass: expected overall_status=pass, got %s", agg.OverallStatus)
	}
	// Exactly 2 lane checks
	if len(agg.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(agg.Checks))
	}
}

func TestWriteAggregateForFullMode_FastFails(t *testing.T) {
	dir := t.TempDir()
	origCwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origCwd)

	writeTestSummaries(t, dir, "fail", "pass", 0)

	if err := writeAggregateForFullMode(); err != nil {
		t.Fatalf("writeAggregateForFullMode: %v", err)
	}

	aggPath := filepath.Join(dir, ".factory", "gate-summary.json")
	data, _ := os.ReadFile(aggPath)
	var agg gate.GateSummary
	json.Unmarshal(data, &agg)

	if agg.OverallStatus != "fail" {
		t.Errorf("fast fails: expected overall_status=fail, got %s", agg.OverallStatus)
	}
	if agg.Checks[0].Status != gate.CheckStatusFail {
		t.Errorf("fast-lane should be fail")
	}
	if agg.Checks[1].Status != gate.CheckStatusPass {
		t.Errorf("long-lane should be pass")
	}
}

func TestWriteAggregateForFullMode_LongFails(t *testing.T) {
	dir := t.TempDir()
	origCwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origCwd)

	writeTestSummaries(t, dir, "pass", "fail", 1)

	if err := writeAggregateForFullMode(); err != nil {
		t.Fatalf("writeAggregateForFullMode: %v", err)
	}

	aggPath := filepath.Join(dir, ".factory", "gate-summary.json")
	data, _ := os.ReadFile(aggPath)
	var agg gate.GateSummary
	json.Unmarshal(data, &agg)

	if agg.OverallStatus != "fail" {
		t.Errorf("long fails: expected overall_status=fail, got %s", agg.OverallStatus)
	}
	if agg.Checks[0].Status != gate.CheckStatusPass {
		t.Errorf("fast-lane should be pass")
	}
	if agg.Checks[1].Status != gate.CheckStatusFail {
		t.Errorf("long-lane should be fail")
	}
}

func TestRemoveIfExists_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("test"), 0644)

	if err := removeIfExists(path); err != nil {
		t.Errorf("removeIfExists: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be removed")
	}
}

func TestRemoveIfExists_IgnoresMissing(t *testing.T) {
	if err := removeIfExists("/nonexistent/path/file.txt"); err != nil {
		t.Errorf("removeIfExists should ignore missing files: %v", err)
	}
}
