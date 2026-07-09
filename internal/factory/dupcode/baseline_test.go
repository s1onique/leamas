// Package dupcode provides tests for duplicate code baseline functionality.
package dupcode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBaseline_Success(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, "baseline.json")

	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds: BaselineThresholds{
			MinLines:  40,
			MinTokens: 400,
		},
		Findings: []BaselineFinding{
			{
				Fingerprint: "abc123",
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
		SchemaVersion: 99,
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
		t.Error("expected error for unsupported schema version")
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

func TestCompareToBaseline_NoChanges(t *testing.T) {
	baseline := Baseline{
		Findings: []BaselineFinding{
			{
				Fingerprint: "stable-hash-abc123",
				TokenCount:  400,
				LineCount:   42,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 55},
					{Path: "bar.go", StartLine: 20, EndLine: 65},
				},
			},
		},
	}

	report := Report{
		Findings: []Finding{
			{
				Fingerprint:       "stable-hash-abc123",
				StableFingerprint: "stable-hash-abc123",
				TokenCount:        400,
				LineCount:         42,
				Occurrences: []Occurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 55},
					{Path: "bar.go", StartLine: 20, EndLine: 65},
				},
			},
		},
	}

	result := CompareToBaseline(report, baseline)

	if result.HasChanges {
		t.Error("expected no changes")
	}
}

func TestCompareToBaseline_NewFingerprint(t *testing.T) {
	baseline := Baseline{
		Findings: []BaselineFinding{
			{
				Fingerprint: "existing-hash",
				TokenCount:  400,
				LineCount:   42,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 55},
				},
			},
		},
	}

	report := Report{
		Findings: []Finding{
			{
				Fingerprint:       "existing-hash",
				StableFingerprint: "existing-hash",
				TokenCount:        400,
				LineCount:         42,
				Occurrences: []Occurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 55},
				},
			},
			{
				Fingerprint:       "new-hash",
				StableFingerprint: "new-hash",
				TokenCount:        450,
				LineCount:         50,
				Occurrences: []Occurrence{
					{Path: "baz.go", StartLine: 30, EndLine: 80},
				},
			},
		},
	}

	result := CompareToBaseline(report, baseline)

	if !result.HasChanges {
		t.Error("expected changes (new fingerprint)")
	}

	if len(result.NewFindings) != 1 {
		t.Errorf("expected 1 new finding, got %d", len(result.NewFindings))
	}

	if len(result.WorsenedFindings) != 0 {
		t.Errorf("expected 0 worsened findings, got %d", len(result.WorsenedFindings))
	}
}

func TestCompareToBaseline_Worsened(t *testing.T) {
	baseline := Baseline{
		Findings: []BaselineFinding{
			{
				Fingerprint: "existing-hash",
				TokenCount:  400,
				LineCount:   42,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 55},
				},
			},
		},
	}

	report := Report{
		Findings: []Finding{
			{
				Fingerprint:       "existing-hash",
				StableFingerprint: "existing-hash",
				TokenCount:        400,
				LineCount:         42,
				Occurrences: []Occurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 55},
					{Path: "bar.go", StartLine: 20, EndLine: 65}, // NEW occurrence
				},
			},
		},
	}

	result := CompareToBaseline(report, baseline)

	if !result.HasChanges {
		t.Error("expected changes (worsened)")
	}

	if len(result.NewFindings) != 0 {
		t.Errorf("expected 0 new findings, got %d", len(result.NewFindings))
	}

	if len(result.WorsenedFindings) != 1 {
		t.Errorf("expected 1 worsened finding, got %d", len(result.WorsenedFindings))
	}

	if len(result.WorsenedFindings[0].NewOccurrences) != 1 {
		t.Errorf("expected 1 new occurrence, got %d", len(result.WorsenedFindings[0].NewOccurrences))
	}
}

func TestStableFingerprintHash_Deterministic(t *testing.T) {
	input := "IDENT STRING IDENT NUMBER IDENT"

	hash1 := StableFingerprintHash(input)
	hash2 := StableFingerprintHash(input)

	if hash1 != hash2 {
		t.Error("expected deterministic hash")
	}

	// SHA256 produces 64 hex characters
	if len(hash1) != 64 {
		t.Errorf("expected 64-char hash, got %d", len(hash1))
	}
}

func TestStableFingerprintHash_DifferentInputs(t *testing.T) {
	hash1 := StableFingerprintHash("input one")
	hash2 := StableFingerprintHash("input two")

	if hash1 == hash2 {
		t.Error("expected different hashes for different inputs")
	}
}

func TestWriteBaseline_Deterministic(t *testing.T) {
	tmpDir := t.TempDir()

	report := Report{
		Findings: []Finding{
			{
				Fingerprint:       "fp1",
				StableFingerprint: "stable1",
				TokenCount:        100,
				LineCount:         10,
				Occurrences: []Occurrence{
					{Path: "b.go", StartLine: 5, EndLine: 15},
					{Path: "a.go", StartLine: 1, EndLine: 10},
				},
			},
			{
				Fingerprint:       "fp2",
				StableFingerprint: "stable2",
				TokenCount:        200,
				LineCount:         20,
				Occurrences: []Occurrence{
					{Path: "c.go", StartLine: 10, EndLine: 30},
				},
			},
		},
		Thresholds: BaselineThresholds{MinLines: 40, MinTokens: 400},
	}

	// Write twice
	path1 := filepath.Join(tmpDir, "baseline1.json")
	path2 := filepath.Join(tmpDir, "baseline2.json")

	if err := WriteBaseline(path1, report); err != nil {
		t.Fatalf("WriteBaseline 1 failed: %v", err)
	}
	if err := WriteBaseline(path2, report); err != nil {
		t.Fatalf("WriteBaseline 2 failed: %v", err)
	}

	// Compare content
	data1, _ := os.ReadFile(path1)
	data2, _ := os.ReadFile(path2)

	if string(data1) != string(data2) {
		t.Error("expected deterministic output")
	}
}

func TestExitCodeFromCompareResult(t *testing.T) {
	tests := []struct {
		result   CompareResult
		wantCode int
	}{
		{CompareResult{HasChanges: false}, 0},
		{CompareResult{HasChanges: true, NewFindings: []NewFinding{{Fingerprint: "new"}}}, 1},
		{CompareResult{HasChanges: true, WorsenedFindings: []WorsenedFinding{{Fingerprint: "worse"}}}, 1},
	}

	for _, tc := range tests {
		got := ExitCodeFromCompareResult(tc.result)
		if got != tc.wantCode {
			t.Errorf("ExitCodeFromCompareResult(%+v) = %d, want %d", tc.result, got, tc.wantCode)
		}
	}
}
