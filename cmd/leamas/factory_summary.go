// Package main provides factory subcommand handlers.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/s1onique/leamas/internal/factory/gate"
)

// testLongSummary represents the summary written by test-long command.
type testLongSummary struct {
	SchemaVersion int              `json:"schema_version"`
	GeneratedAt   string           `json:"generated_at"`
	Tests         []testLongResult `json:"tests"`
	Passed        int              `json:"passed"`
	Failed        int              `json:"failed"`
	Total         int              `json:"total"`
}

type testLongResult struct {
	ID       string `json:"id"`
	Package  string `json:"package"`
	Test     string `json:"test"`
	Passed   bool   `json:"passed"`
	Duration string `json:"duration,omitempty"`
	Error    string `json:"error,omitempty"`
}

// ErrMissingFastSummary indicates the fast lane summary is missing.
var ErrMissingFastSummary = errors.New("missing fast lane summary")

// ErrMissingLongSummary indicates the long lane summary is missing.
var ErrMissingLongSummary = errors.New("missing long lane summary")

// ErrInvalidFastStatus indicates the fast lane status is not valid.
var ErrInvalidFastStatus = errors.New("invalid fast lane status: must be 'pass' or 'fail'")

// ErrInvalidLongTotal indicates the long lane total is invalid.
var ErrInvalidLongTotal = errors.New("invalid long lane total: must be > 0")

// ErrLongCountMismatch indicates the long lane counts don't match.
var ErrLongCountMismatch = errors.New("long lane counts mismatch: passed + failed != total")

// ErrTestResultMismatch indicates a test result disagrees with totals.
var ErrTestResultMismatch = errors.New("test result disagrees with passed/failed totals")

// validateFastLaneSummary validates the fast lane summary.
func validateFastLaneSummary(s *fastLaneSummary) error {
	if s == nil {
		return ErrMissingFastSummary
	}
	if s.SchemaVersion != 1 {
		return fmt.Errorf("invalid schema_version: got %d, want 1", s.SchemaVersion)
	}
	if s.OverallStatus != "pass" && s.OverallStatus != "fail" {
		return ErrInvalidFastStatus
	}
	return nil
}

// validateLongLaneSummary validates the long lane summary.
func validateLongLaneSummary(s *testLongSummary) error {
	if s == nil {
		return ErrMissingLongSummary
	}
	if s.SchemaVersion != 1 {
		return fmt.Errorf("invalid schema_version: got %d, want 1", s.SchemaVersion)
	}
	if s.Total <= 0 {
		return ErrInvalidLongTotal
	}
	if len(s.Tests) != s.Total {
		return fmt.Errorf("invalid tests length: got %d, want %d", len(s.Tests), s.Total)
	}
	if s.Passed+s.Failed != s.Total {
		return ErrLongCountMismatch
	}
	// Count actual results and verify against totals
	var actualPassed, actualFailed int
	for _, r := range s.Tests {
		if r.Passed {
			actualPassed++
		} else {
			actualFailed++
		}
	}
	if actualPassed != s.Passed || actualFailed != s.Failed {
		return ErrTestResultMismatch
	}
	return nil
}

// writeFastSummary writes the fast lane summary to .factory/gate-fast-summary.json.
func writeFastSummary(startedAt, finishedAt time.Time, exitCode int) error {
	status := gate.CheckStatusPass
	if exitCode != 0 {
		status = gate.CheckStatusFail
	}
	summary := gate.GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   finishedAt.Format(time.RFC3339),
		Tool:          "leamas factory gate",
		OverallStatus: string(status),
		Checks: []gate.Check{
			{Name: "fast-lane", Status: status},
		},
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal fast summary: %w", err)
	}
	return os.WriteFile(".factory/gate-fast-summary.json", data, 0644)
}

// writeAggregateSummary generates the aggregate summary from validated child summaries.
// It is fail-closed: requires both fast and long summaries to exist, be valid, and pass.
func writeAggregateSummary() error {
	summary := gate.GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Tool:          "leamas factory gate",
		OverallStatus: "fail",
		Checks:        []gate.Check{},
	}

	// Read and validate fast summary
	fastSummary, err := readFastSummary()
	if err != nil {
		return fmt.Errorf("fast summary read: %w", err)
	}
	if err := validateFastLaneSummary(fastSummary); err != nil {
		return fmt.Errorf("fast summary validation: %w", err)
	}
	summary.Checks = append(summary.Checks, gate.Check{Name: "fast-lane", Status: gate.CheckStatusPass})
	if fastSummary.OverallStatus == "fail" {
		summary.OverallStatus = "fail"
		summary.Checks = append(summary.Checks, gate.Check{Name: "fast-lane-status", Status: gate.CheckStatusFail})
	}

	// Read and validate long summary
	longSummary, err := readLongSummary()
	if err != nil {
		return fmt.Errorf("long summary read: %w", err)
	}
	if err := validateLongLaneSummary(longSummary); err != nil {
		return fmt.Errorf("long summary validation: %w", err)
	}
	summary.Checks = append(summary.Checks, gate.Check{Name: "long-lane", Status: gate.CheckStatusPass})
	if longSummary.Failed > 0 {
		summary.OverallStatus = "fail"
		summary.Checks = append(summary.Checks, gate.Check{Name: "long-lane-status", Status: gate.CheckStatusFail})
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal aggregate summary: %w", err)
	}
	return os.WriteFile(".factory/gate-summary.json", data, 0644)
}

// writeAggregateForFullMode writes the aggregate summary after a full mode run.
// Both lanes must have been executed and validated.
func writeAggregateForFullMode() error {
	// Read and validate fast summary
	fastSummary, err := readFastSummary()
	if err != nil {
		return fmt.Errorf("fast summary read: %w", err)
	}
	if err := validateFastLaneSummary(fastSummary); err != nil {
		return fmt.Errorf("fast summary validation: %w", err)
	}
	fastStatus := gate.CheckStatusPass
	if fastSummary.OverallStatus == "fail" {
		fastStatus = gate.CheckStatusFail
	}

	// Read and validate long summary
	longSummary, err := readLongSummary()
	if err != nil {
		return fmt.Errorf("long summary read: %w", err)
	}
	if err := validateLongLaneSummary(longSummary); err != nil {
		return fmt.Errorf("long summary validation: %w", err)
	}
	longStatus := gate.CheckStatusPass
	if longSummary.Failed > 0 {
		longStatus = gate.CheckStatusFail
	}

	// Derive overall status from lane statuses
	overallStatus := "pass"
	if fastStatus != gate.CheckStatusPass || longStatus != gate.CheckStatusPass {
		overallStatus = "fail"
	}

	summary := gate.GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Tool:          "leamas factory gate",
		OverallStatus: overallStatus,
		Checks: []gate.Check{
			{Name: "fast-lane", Status: fastStatus},
			{Name: "long-lane", Status: longStatus},
		},
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal aggregate summary: %w", err)
	}
	return os.WriteFile(".factory/gate-summary.json", data, 0644)
}

// writeAggregateAfterFastFailure writes the aggregate summary when fast lane fails.
// Long lane is skipped (not failed) in this case.
func writeAggregateAfterFastFailure() error {
	summary := gate.GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Tool:          "leamas factory gate",
		OverallStatus: "fail",
		Checks: []gate.Check{
			{Name: "fast-lane", Status: gate.CheckStatusFail, Evidence: "fast lane failed"},
			{Name: "long-lane", Status: "skip", Evidence: "not executed due to fast lane failure"},
		},
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal aggregate summary: %w", err)
	}
	return os.WriteFile(".factory/gate-summary.json", data, 0644)
}

type fastLaneSummary struct {
	SchemaVersion int    `json:"schema_version"`
	OverallStatus string `json:"overall_status"`
	GeneratedAt   string `json:"generated_at"`
}

func readFastSummary() (*fastLaneSummary, error) {
	data, err := os.ReadFile(".factory/gate-fast-summary.json")
	if err != nil {
		return nil, err
	}
	var s fastLaneSummary
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// removeIfExists removes a file if it exists, ignoring ErrNotExist.
// Returns an error if removal fails for any other reason.
func removeIfExists(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("remove stale artifact %s: %w", path, err)
}

func readLongSummary() (*testLongSummary, error) {
	data, err := os.ReadFile(".factory/gate-long-summary.json")
	if err != nil {
		return nil, err
	}
	var s testLongSummary
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
