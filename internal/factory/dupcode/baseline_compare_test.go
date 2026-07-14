// Package dupcode provides tests for baseline comparison.
package dupcode

import (
	"testing"
)

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
					{Path: "bar.go", StartLine: 20, EndLine: 65},
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

	if result.WorsenedFindings[0].TotalNow != 2 {
		t.Errorf("expected TotalNow=2, got %d", result.WorsenedFindings[0].TotalNow)
	}
}

func TestCompareToBaseline_MultipleFindings(t *testing.T) {
	baseline := Baseline{
		Findings: []BaselineFinding{
			{
				Fingerprint: "hash-1",
				TokenCount:  400,
				LineCount:   42,
				Occurrences: []BaselineOccurrence{
					{Path: "a.go", StartLine: 10, EndLine: 55},
				},
			},
			{
				Fingerprint: "hash-2",
				TokenCount:  420,
				LineCount:   44,
				Occurrences: []BaselineOccurrence{
					{Path: "b.go", StartLine: 20, EndLine: 70},
				},
			},
		},
	}

	report := Report{
		Findings: []Finding{
			{
				Fingerprint:       "hash-1",
				StableFingerprint: "hash-1",
				TokenCount:        400,
				LineCount:         42,
				Occurrences: []Occurrence{
					{Path: "a.go", StartLine: 10, EndLine: 55},
					{Path: "a2.go", StartLine: 30, EndLine: 75},
				},
			},
			{
				Fingerprint:       "hash-2",
				StableFingerprint: "hash-2",
				TokenCount:        420,
				LineCount:         44,
				Occurrences: []Occurrence{
					{Path: "b.go", StartLine: 20, EndLine: 70},
				},
			},
			{
				Fingerprint:       "hash-3",
				StableFingerprint: "hash-3",
				TokenCount:        380,
				LineCount:         40,
				Occurrences: []Occurrence{
					{Path: "c.go", StartLine: 50, EndLine: 90},
					{Path: "d.go", StartLine: 60, EndLine: 100},
				},
			},
		},
	}

	result := CompareToBaseline(report, baseline)

	if !result.HasChanges {
		t.Error("expected changes")
	}

	if len(result.NewFindings) != 1 {
		t.Errorf("expected 1 new finding, got %d", len(result.NewFindings))
	}

	if len(result.WorsenedFindings) != 1 {
		t.Errorf("expected 1 worsened finding, got %d", len(result.WorsenedFindings))
	}
}

func TestCompareToBaseline_EmptyBaseline(t *testing.T) {
	baseline := Baseline{}

	report := Report{
		Findings: []Finding{
			{
				Fingerprint:       "hash-1",
				StableFingerprint: "hash-1",
				TokenCount:        400,
				LineCount:         42,
				Occurrences: []Occurrence{
					{Path: "a.go", StartLine: 10, EndLine: 55},
				},
			},
		},
	}

	result := CompareToBaseline(report, baseline)

	if !result.HasChanges {
		t.Error("expected changes")
	}

	if len(result.NewFindings) != 1 {
		t.Errorf("expected 1 new finding, got %d", len(result.NewFindings))
	}
}

func TestCompareToBaseline_EmptyReport(t *testing.T) {
	baseline := Baseline{
		Findings: []BaselineFinding{
			{
				Fingerprint: "hash-1",
				TokenCount:  400,
				LineCount:   42,
				Occurrences: []BaselineOccurrence{
					{Path: "a.go", StartLine: 10, EndLine: 55},
				},
			},
		},
	}

	report := Report{
		Findings: []Finding{},
	}

	result := CompareToBaseline(report, baseline)

	if result.HasChanges {
		t.Error("expected no changes (removal is not worsened)")
	}
}

func TestExitCodeFromCompareResult(t *testing.T) {
	if ExitCodeFromCompareResult(CompareResult{HasChanges: false}) != 0 {
		t.Error("expected exit code 0 when no changes")
	}

	if ExitCodeFromCompareResult(CompareResult{HasChanges: true}) != 1 {
		t.Error("expected exit code 1 when has changes")
	}
}
