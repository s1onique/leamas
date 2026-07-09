// Package dupcode provides tests for baseline validation.
package dupcode

import "testing"

func TestValidateBaselinePaths_AbsolutePathFails(t *testing.T) {
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "002ec5ff009cad28f7e278c01749ac4268d1ed3a1325a86df39db87d7c909edb",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "/tmp/repo/foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	}

	findings := ValidateBaselinePaths(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "absolute_path_in_baseline" {
		t.Errorf("expected kind 'absolute_path_in_baseline', got %q", findings[0].Kind)
	}
}

func TestValidateBaselinePaths_BackslashPathFails(t *testing.T) {
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "002ec5ff009cad28f7e278c01749ac4268d1ed3a1325a86df39db87d7c909edb",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "internal\\foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	}

	findings := ValidateBaselinePaths(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "os_specific_path_in_baseline" {
		t.Errorf("expected kind 'os_specific_path_in_baseline', got %q", findings[0].Kind)
	}
}

func TestValidateBaselinePaths_ParentTraversalFails(t *testing.T) {
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "002ec5ff009cad28f7e278c01749ac4268d1ed3a1325a86df39db87d7c909edb",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "../foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	}

	findings := ValidateBaselinePaths(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "path_escapes_repo_root" {
		t.Errorf("expected kind 'path_escapes_repo_root', got %q", findings[0].Kind)
	}
}

func TestValidateBaselinePaths_InvalidStartLineFails(t *testing.T) {
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "002ec5ff009cad28f7e278c01749ac4268d1ed3a1325a86df39db87d7c909edb",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 0, EndLine: 50},
				},
			},
		},
	}

	findings := ValidateBaselinePaths(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "invalid_start_line" {
		t.Errorf("expected kind 'invalid_start_line', got %q", findings[0].Kind)
	}
}

func TestValidateBaselinePaths_EndLineBeforeStartLineFails(t *testing.T) {
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "002ec5ff009cad28f7e278c01749ac4268d1ed3a1325a86df39db87d7c909edb",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 50, EndLine: 10},
				},
			},
		},
	}

	findings := ValidateBaselinePaths(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "end_line_before_start_line" {
		t.Errorf("expected kind 'end_line_before_start_line', got %q", findings[0].Kind)
	}
}

func TestValidateBaselineFingerprints_EmptyFingerprintFails(t *testing.T) {
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	}

	findings := ValidateBaselineFingerprints(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "empty_fingerprint" {
		t.Errorf("expected kind 'empty_fingerprint', got %q", findings[0].Kind)
	}
}

func TestValidateBaselineFingerprints_InvalidFingerprintFormatFails(t *testing.T) {
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "not-a-valid-sha256",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	}

	findings := ValidateBaselineFingerprints(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "invalid_fingerprint_format" {
		t.Errorf("expected kind 'invalid_fingerprint_format', got %q", findings[0].Kind)
	}
}

func TestValidateBaselineFingerprints_DuplicateFingerprintFails(t *testing.T) {
	fp := "002ec5ff009cad28f7e278c01749ac4268d1ed3a1325a86df39db87d7c909edb"
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: fp,
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
			{
				Fingerprint: fp, // Duplicate!
				TokenCount:  450,
				LineCount:   80,
				Occurrences: []BaselineOccurrence{
					{Path: "bar.go", StartLine: 20, EndLine: 60},
				},
			},
		},
	}

	findings := ValidateBaselineFingerprints(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "duplicate_fingerprint" {
		t.Errorf("expected kind 'duplicate_fingerprint', got %q", findings[0].Kind)
	}
}

func TestValidateBaselineOrdering_FindingsNotSortedFails(t *testing.T) {
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "bbb",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
			{
				Fingerprint: "aaa", // Out of order!
				TokenCount:  450,
				LineCount:   80,
				Occurrences: []BaselineOccurrence{
					{Path: "bar.go", StartLine: 20, EndLine: 60},
				},
			},
		},
	}

	findings := ValidateBaselineOrdering(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "findings_not_sorted" {
		t.Errorf("expected kind 'findings_not_sorted', got %q", findings[0].Kind)
	}
}

func TestValidateBaselineOrdering_OccurrencesNotSortedFails(t *testing.T) {
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "aaa",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 50, EndLine: 60}, // Out of order!
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	}

	findings := ValidateBaselineOrdering(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "occurrences_not_sorted" {
		t.Errorf("expected kind 'occurrences_not_sorted', got %q", findings[0].Kind)
	}
}

func TestValidateBaselineOrdering_OccurrencesNotSortedByEndLineFails(t *testing.T) {
	baseline := Baseline{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-09T00:00:00Z",
		Tool:          "leamas dupcode",
		Thresholds:    BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "aaa",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 80}, // Same path, same start, higher end - out of order!
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	}

	findings := ValidateBaselineOrdering(baseline)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "occurrences_not_sorted" {
		t.Errorf("expected kind 'occurrences_not_sorted', got %q", findings[0].Kind)
	}
}
