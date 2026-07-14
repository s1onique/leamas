// Package dupcode provides tests for v3 baseline handling.
package dupcode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestV3_AlgorithmVersionBumped verifies that AlgorithmVersion is 3.
func TestV3_AlgorithmVersionBumped(t *testing.T) {
	if AlgorithmVersion != 3 {
		t.Errorf("expected AlgorithmVersion=3, got %d", AlgorithmVersion)
	}
}

// TestV3_RejectsV2Baseline tests that algorithm-v2 baselines are rejected.
func TestV3_RejectsV2Baseline(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, "baseline.json")

	baseline := Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: 2,
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

	_, err = LoadBaseline(baselinePath)
	if err == nil {
		t.Error("expected error for v2 baseline")
	}
}

// TestV3_AcceptsV3Baseline tests that algorithm-v3 baselines are accepted.
func TestV3_AcceptsV3Baseline(t *testing.T) {
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
		t.Errorf("expected no error for v3 baseline, got: %v", err)
	}
	if loaded.AlgorithmVersion != 3 {
		t.Errorf("expected algorithm version 3, got %d", loaded.AlgorithmVersion)
	}
}

// TestV3_RejectsMissingAlgorithmVersion tests that baselines without algorithm version are rejected.
func TestV3_RejectsMissingAlgorithmVersion(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, "baseline.json")

	baseline := map[string]interface{}{
		"schema_version": 1,
		"generated_at":   "2026-07-09T00:00:00Z",
		"tool":           "leamas dupcode",
		"thresholds": map[string]int{
			"min_lines":  40,
			"min_tokens": 400,
		},
		"findings": []interface{}{},
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
}

// TestV3_AlgorithmVersionMismatchClearError tests that v2 baseline produces clear error message.
func TestV3_AlgorithmVersionMismatchClearError(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, "baseline.json")

	baseline := Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: 2,
		GeneratedAt:      "2026-07-09T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds: BaselineThresholds{
			MinLines:  40,
			MinTokens: 400,
		},
		Findings: []BaselineFinding{},
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
		t.Error("expected error for v2 baseline")
	}
}

// TestV3_WriteBaselineUsesV3 verifies that WriteBaseline uses the current algorithm version.
func TestV3_WriteBaselineUsesV3(t *testing.T) {
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

	loaded, err := LoadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("LoadBaseline failed: %v", err)
	}

	if loaded.AlgorithmVersion != 3 {
		t.Errorf("expected AlgorithmVersion=3 in baseline, got %d", loaded.AlgorithmVersion)
	}
}
