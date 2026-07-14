// Package dupcode provides tests for duplicate code baseline functionality.
package dupcode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBaseline_Success(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, "baseline.json")

	baseline := Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: 3,
		GeneratedAt:      "2026-07-09T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds: BaselineThresholds{
			MinLines:  40,
			MinTokens: 400,
		},
		Findings: []BaselineFinding{
			{
				Fingerprint: "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
				TokenCount:  400,
				LineCount:   42,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 55},
				},
			},
		},
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal baseline: %v", err)
	}
	if err := os.WriteFile(baselinePath, data, 0644); err != nil {
		t.Fatalf("failed to write baseline: %v", err)
	}

	loaded, err := LoadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("LoadBaseline failed: %v", err)
	}

	if loaded.SchemaVersion != 1 {
		t.Errorf("expected schema version 1, got %d", loaded.SchemaVersion)
	}
	if len(loaded.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(loaded.Findings))
	}
}

func TestLoadBaseline_UnsupportedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, "baseline.json")

	baseline := Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: 99, // Invalid algorithm version
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal baseline: %v", err)
	}
	if err := os.WriteFile(baselinePath, data, 0644); err != nil {
		t.Fatalf("failed to write baseline: %v", err)
	}

	_, err = LoadBaseline(baselinePath)
	if err == nil {
		t.Error("expected error for unsupported algorithm version")
	}
}

func TestLoadBaseline_MissingAlgorithmVersion(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, "baseline.json")

	baseline := Baseline{
		SchemaVersion: 1,
		// No AlgorithmVersion field - simulating old format
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal baseline: %v", err)
	}
	if err := os.WriteFile(baselinePath, data, 0644); err != nil {
		t.Fatalf("failed to write baseline: %v", err)
	}

	_, err = LoadBaseline(baselinePath)
	if err == nil {
		t.Error("expected error for missing algorithm_version")
	}
	if !strings.Contains(err.Error(), "algorithm_version") {
		t.Errorf("expected error to mention algorithm_version, got: %v", err)
	}
}

func TestLoadBaseline_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, "baseline.json")

	if err := os.WriteFile(baselinePath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write baseline: %v", err)
	}

	_, err := LoadBaseline(baselinePath)
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestLoadBaseline_FileNotFound(t *testing.T) {
	_, err := LoadBaseline("/nonexistent/path/baseline.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestWriteBaseline_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, "baseline.json")

	report := Report{
		Findings: []Finding{
			{
				Fingerprint:       "display-fp",
				StableFingerprint: "stable-hash-abc123",
				TokenCount:        400,
				LineCount:         42,
				Occurrences: []Occurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 55},
					{Path: "bar.go", StartLine: 20, EndLine: 65},
				},
			},
		},
		Thresholds: BaselineThresholds{
			MinLines:  40,
			MinTokens: 400,
		},
	}

	if err := WriteBaseline(baselinePath, report); err != nil {
		t.Fatalf("WriteBaseline failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(baselinePath); err != nil {
		t.Fatalf("baseline file not created: %v", err)
	}

	// Load and verify
	loaded, err := LoadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("LoadBaseline failed: %v", err)
	}

	if len(loaded.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(loaded.Findings))
	}

	// Verify stable fingerprint was used
	if loaded.Findings[0].Fingerprint != "stable-hash-abc123" {
		t.Errorf("expected stable fingerprint, got %s", loaded.Findings[0].Fingerprint)
	}

	if len(loaded.Findings[0].Occurrences) != 2 {
		t.Errorf("expected 2 occurrences, got %d", len(loaded.Findings[0].Occurrences))
	}

	// Verify thresholds
	if loaded.Thresholds.MinLines != 40 || loaded.Thresholds.MinTokens != 400 {
		t.Errorf("thresholds mismatch: got %+v", loaded.Thresholds)
	}
}
