// Package dupcode provides tests for baseline drift checking from reports.
package dupcode

import (
	"testing"
)

// TestCheckBaselineDriftFromReport_MatchingReport tests that matching report produces no drift.
func TestCheckBaselineDriftFromReport_MatchingReport(t *testing.T) {
	tmpDir := t.TempDir()

	committedBaseline := Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: AlgorithmVersion,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds:       BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "aaa",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	}

	currentReport := Report{
		Findings: []Finding{
			{
				Fingerprint: "aaa",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []Occurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
		Thresholds: BaselineThresholds{MinLines: 40, MinTokens: 400},
		Root:       tmpDir,
	}

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	findings := CheckBaselineDriftFromReport(tmpDir, committedBaseline, currentReport, policy)

	if len(findings) != 0 {
		t.Errorf("expected no drift findings, got: %#v", findings)
	}
}

// TestCheckBaselineDriftFromReport_StaleReport tests that stale report produces drift finding.
func TestCheckBaselineDriftFromReport_StaleReport(t *testing.T) {
	tmpDir := t.TempDir()

	committedBaseline := Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: AlgorithmVersion,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds:       BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "aaa",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	}

	currentReport := Report{
		Findings: []Finding{
			{
				Fingerprint: "aaa",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []Occurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
			{
				Fingerprint: "bbb",
				TokenCount:  500,
				LineCount:   80,
				Occurrences: []Occurrence{
					{Path: "bar.go", StartLine: 20, EndLine: 60},
				},
			},
		},
		Thresholds: BaselineThresholds{MinLines: 40, MinTokens: 400},
		Root:       tmpDir,
	}

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	findings := CheckBaselineDriftFromReport(tmpDir, committedBaseline, currentReport, policy)

	if len(findings) != 1 {
		t.Fatalf("expected 1 drift finding, got %d: %#v", len(findings), findings)
	}
	if findings[0].Kind != "dupcode_baseline_drift" {
		t.Errorf("expected kind 'dupcode_baseline_drift', got %q", findings[0].Kind)
	}
}

// TestCheckBaselineDriftFromReport_DeterministicOutput tests that repeated calls produce identical output.
func TestCheckBaselineDriftFromReport_DeterministicOutput(t *testing.T) {
	tmpDir := t.TempDir()

	committedBaseline := Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: AlgorithmVersion,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds:       BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "aaa",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	}

	currentReport := Report{
		Findings: []Finding{
			{
				Fingerprint: "aaa",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []Occurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
		Thresholds: BaselineThresholds{MinLines: 40, MinTokens: 400},
		Root:       tmpDir,
	}

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	findings1 := CheckBaselineDriftFromReport(tmpDir, committedBaseline, currentReport, policy)
	findings2 := CheckBaselineDriftFromReport(tmpDir, committedBaseline, currentReport, policy)

	if len(findings1) != len(findings2) {
		t.Errorf("inconsistent finding count: %d vs %d", len(findings1), len(findings2))
	}
}

// TestCheckBaselineDriftFromReport_RootAwarePath tests that findings use root-aware paths.
func TestCheckBaselineDriftFromReport_RootAwarePath(t *testing.T) {
	tmpDir := t.TempDir()

	committedBaseline := Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: AlgorithmVersion,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds:       BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings:         []BaselineFinding{},
	}

	currentReport := Report{
		Findings: []Finding{
			{
				Fingerprint: "bbb",
				TokenCount:  500,
				LineCount:   80,
				Occurrences: []Occurrence{
					{Path: "bar.go", StartLine: 20, EndLine: 60},
				},
			},
		},
		Thresholds: BaselineThresholds{MinLines: 40, MinTokens: 400},
		Root:       tmpDir,
	}

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	findings := CheckBaselineDriftFromReport(tmpDir, committedBaseline, currentReport, policy)

	if len(findings) != 1 {
		t.Fatalf("expected 1 drift finding, got %d", len(findings))
	}

	if findings[0].Path != ".factory/dupcode-baseline.json" {
		t.Errorf("expected path '.factory/dupcode-baseline.json', got %q", findings[0].Path)
	}
}
