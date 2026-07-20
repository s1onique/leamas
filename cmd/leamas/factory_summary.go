// Package main provides factory subcommand handlers.
package main

import (
	"encoding/json"
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

// writeAggregateSummary generates the aggregate summary from child summaries.
// It is fail-closed: requires both fast and long summaries to exist and pass.
func writeAggregateSummary() error {
	summary := gate.GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Tool:          "leamas factory gate",
		OverallStatus: "fail",
		Checks:        []gate.Check{},
	}

	// Read fast summary - fail if missing or malformed
	fastSummary, err := readFastSummary()
	if err != nil {
		return fmt.Errorf("missing or malformed fast summary: %w", err)
	}
	summary.Checks = append(summary.Checks, gate.Check{Name: "fast-lane", Status: gate.CheckStatusPass})
	if fastSummary.OverallStatus == "fail" {
		summary.OverallStatus = "fail"
		summary.Checks = append(summary.Checks, gate.Check{Name: "fast-lane-status", Status: gate.CheckStatusFail})
	}

	// Read long summary - fail if missing or malformed
	longSummary, err := readLongSummary()
	if err != nil {
		return fmt.Errorf("missing or malformed long summary: %w", err)
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

// writeAggregateSummaryWithStatus writes the aggregate summary using provided status.
// Used by full mode when long lane hasn't run yet.
func writeAggregateSummaryWithStatus(initialStatus string, fastFailed bool, longExitCode int) error {
	summary := gate.GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Tool:          "leamas factory gate",
		OverallStatus: initialStatus,
		Checks:        []gate.Check{},
	}

	// Read fast summary - fail if missing or malformed
	fastSummary, err := readFastSummary()
	if err != nil {
		return fmt.Errorf("missing or malformed fast summary: %w", err)
	}
	summary.Checks = append(summary.Checks, gate.Check{Name: "fast-lane", Status: gate.CheckStatusPass})
	if fastSummary.OverallStatus == "fail" || fastFailed {
		summary.OverallStatus = "fail"
		summary.Checks = append(summary.Checks, gate.Check{Name: "fast-lane-status", Status: gate.CheckStatusFail})
	}

	// Only include long lane if it has been run
	longSummary, err := readLongSummary()
	if err == nil && longSummary != nil {
		summary.Checks = append(summary.Checks, gate.Check{Name: "long-lane", Status: gate.CheckStatusPass})
		if longSummary.Failed > 0 || longExitCode != 0 {
			summary.OverallStatus = "fail"
			summary.Checks = append(summary.Checks, gate.Check{Name: "long-lane-status", Status: gate.CheckStatusFail})
		}
	} else if longExitCode >= 0 {
		// Long lane ran but failed to produce summary
		return fmt.Errorf("missing long summary after long lane execution")
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
