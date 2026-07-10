// Package main provides unit tests for the gate-summary logic.
// These tests call the internal gate/summary package directly, avoiding
// subprocess launch that could cause recursion with execgate.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/gate"
)

// TestGateSummary_ReadValidArtifact tests reading a valid gate summary artifact.
func TestGateSummary_ReadValidArtifact(t *testing.T) {
	// Create a temporary valid gate summary file
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "gate-summary.json")

	summary := gate.GateSummary{
		SchemaVersion: gate.GateSummarySchemaVersion,
		GeneratedAt:   "2026-01-15T10:00:00Z",
		Tool:          "leamas factory gate",
		OverallStatus: string(gate.CheckStatusPass),
		Checks: []gate.Check{
			{
				Name:       "coverage",
				Status:     gate.CheckStatusPass,
				DurationMs: 1234,
				Evidence:   "coverage 85%",
			},
			{
				Name:       "lint",
				Status:     gate.CheckStatusPass,
				DurationMs: 500,
				Evidence:   "no issues",
			},
		},
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal summary: %v", err)
	}

	if err := os.WriteFile(artifactPath, data, 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	// Read it back
	read, err := gate.ReadGateSummary(artifactPath)
	if err != nil {
		t.Fatalf("failed to read summary: %v", err)
	}

	if read.SchemaVersion != summary.SchemaVersion {
		t.Errorf("expected schema version %d, got %d", summary.SchemaVersion, read.SchemaVersion)
	}

	if read.OverallStatus != summary.OverallStatus {
		t.Errorf("expected overall status %s, got %s", summary.OverallStatus, read.OverallStatus)
	}

	if len(read.Checks) != len(summary.Checks) {
		t.Errorf("expected %d checks, got %d", len(summary.Checks), len(read.Checks))
	}
}

// TestGateSummary_ReadMissingFile tests reading a non-existent artifact.
func TestGateSummary_ReadMissingFile(t *testing.T) {
	_, err := gate.ReadGateSummary("/nonexistent/path/gate-summary.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
	if err != gate.ErrGateSummaryMissing {
		t.Errorf("expected ErrGateSummaryMissing, got %v", err)
	}
}

// TestGateSummary_ReadInvalidJSON tests reading an invalid JSON artifact.
func TestGateSummary_ReadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "gate-summary.json")

	invalidJSON := []byte(`{"schema_version": 1, "invalid json`)
	if err := os.WriteFile(artifactPath, invalidJSON, 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	_, err := gate.ReadGateSummary(artifactPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestGateSummary_RenderTextOutput tests rendering a summary as text.
func TestGateSummary_RenderTextOutput(t *testing.T) {
	summary := &gate.GateSummary{
		SchemaVersion: gate.GateSummarySchemaVersion,
		GeneratedAt:   "2026-01-15T10:00:00Z",
		Tool:          "leamas factory gate",
		OverallStatus: string(gate.CheckStatusPass),
		Checks: []gate.Check{
			{
				Name:       "coverage",
				Status:     gate.CheckStatusPass,
				DurationMs: 1234,
				Evidence:   "coverage 85%",
			},
		},
	}

	rendered := gate.RenderGateSummary(summary, nil)

	// Should contain key fields
	if !strings.Contains(rendered, "source_status=present") {
		t.Error("expected source_status=present")
	}
	if !strings.Contains(rendered, "overall_status=pass") {
		t.Error("expected overall_status=pass")
	}
	if !strings.Contains(rendered, "checks_total=1") {
		t.Error("expected checks_total=1")
	}
	if !strings.Contains(rendered, "checks_passed=1") {
		t.Error("expected checks_passed=1")
	}

	// Should NOT contain prose
	prosePatterns := []string{"checking", "verified", "completed", "Running", "Gate summary written to"}
	for _, pattern := range prosePatterns {
		if strings.Contains(rendered, pattern) {
			t.Errorf("rendered output should not contain prose '%s': %s", pattern, rendered)
		}
	}
}

// TestGateSummary_RenderMissingArtifact tests rendering when artifact is missing.
// When summary is nil with no error, it's considered "missing".
func TestGateSummary_RenderMissingArtifact(t *testing.T) {
	rendered := gate.RenderGateSummary(nil, nil)

	if !strings.Contains(rendered, "source_status=missing") {
		t.Error("expected source_status=missing")
	}
	if !strings.Contains(rendered, "overall_status=unavailable") {
		t.Error("expected overall_status=unavailable")
	}
}

// TestGateSummary_RenderInvalidArtifact tests rendering when artifact is invalid.
func TestGateSummary_RenderInvalidArtifact(t *testing.T) {
	rendered := gate.RenderGateSummary(nil, gate.ErrGateSummaryInvalid)

	if !strings.Contains(rendered, "source_status=invalid") {
		t.Error("expected source_status=invalid")
	}
}

// TestGateSummary_ParseStatus tests parsing overall_status from rendered text.
func TestGateSummary_ParseStatus(t *testing.T) {
	tests := []struct {
		rendered string
		expected string
	}{
		{"overall_status=pass\n", "pass"},
		{"overall_status=fail\n", "fail"},
		{"overall_status=unavailable\n", "unavailable"},
		{"no status here\n", "unavailable"},
	}

	for _, tc := range tests {
		parsed := gate.ParseGateSummaryStatus(tc.rendered)
		if parsed != tc.expected {
			t.Errorf("ParseGateSummaryStatus(%q): expected %q, got %q", tc.rendered, tc.expected, parsed)
		}
	}
}

// TestGateSummary_ParseSourceStatus tests parsing source_status from rendered text.
func TestGateSummary_ParseSourceStatus(t *testing.T) {
	tests := []struct {
		rendered string
		expected string
	}{
		{"source_status=present\n", "present"},
		{"source_status=missing\n", "missing"},
		{"source_status=invalid\n", "invalid"},
		{"no status here\n", "missing"},
	}

	for _, tc := range tests {
		parsed := gate.ParseGateSummarySourceStatus(tc.rendered)
		if parsed != tc.expected {
			t.Errorf("ParseGateSummarySourceStatus(%q): expected %q, got %q", tc.rendered, tc.expected, parsed)
		}
	}
}

// TestGateSummary_Exists tests the existence check.
func TestGateSummary_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "gate-summary.json")

	// Should not exist yet
	if gate.GateSummaryExists(artifactPath) {
		t.Error("expected file to not exist")
	}

	// Create the file
	if err := os.WriteFile(artifactPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Should exist now
	if !gate.GateSummaryExists(artifactPath) {
		t.Error("expected file to exist")
	}
}

// TestGateSummary_WriteGateSummary tests writing a gate summary.
func TestGateSummary_WriteGateSummary(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".factory", "gate-summary.json")

	// WriteGateSummary should create the directory and file
	err := gate.WriteGateSummary(tmpDir, outputPath)
	if err != nil {
		t.Fatalf("WriteGateSummary failed: %v", err)
	}

	// Verify the file exists
	if !gate.GateSummaryExists(outputPath) {
		t.Error("expected gate summary file to exist after WriteGateSummary")
	}

	// Verify it can be read back
	summary, err := gate.ReadGateSummary(outputPath)
	if err != nil {
		t.Fatalf("failed to read written summary: %v", err)
	}

	// Should have tool set by WriteGateSummary
	if summary.Tool != "leamas factory gate-summary" {
		t.Errorf("expected tool 'leamas factory gate-summary', got %q", summary.Tool)
	}
}

// TestGateSummary_ReadSanitizesEvidence tests that ReadGateSummary sanitizes evidence.
func TestGateSummary_ReadSanitizesEvidence(t *testing.T) {
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "gate-summary.json")

	// Create a summary with long evidence
	summary := gate.GateSummary{
		SchemaVersion: gate.GateSummarySchemaVersion,
		GeneratedAt:   "2026-01-15T10:00:00Z",
		OverallStatus: string(gate.CheckStatusPass),
		Checks: []gate.Check{
			{
				Name:     "long_evidence",
				Status:   gate.CheckStatusPass,
				Evidence: strings.Repeat("x", 500), // Way over MaxEvidenceLength
			},
		},
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if err := os.WriteFile(artifactPath, data, 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	// ReadGateSummary sanitizes the evidence
	read, err := gate.ReadGateSummary(artifactPath)
	if err != nil {
		t.Fatalf("failed to read summary: %v", err)
	}

	// Evidence should be sanitized to max length
	if len(read.Checks[0].Evidence) > gate.MaxEvidenceLength {
		t.Errorf("evidence should be truncated to %d chars, got %d", gate.MaxEvidenceLength, len(read.Checks[0].Evidence))
	}
}
