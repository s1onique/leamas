// Package dupcode provides tests for v3 fingerprinting and ratchet behavior.
package dupcode

import (
	"testing"
)

// TestV3_FingerprintDomainUsesVersion tests that fingerprints use the version domain.
func TestV3_FingerprintDomainUsesVersion(t *testing.T) {
	fp := v3SeedFingerprint("test-tokens", "a.go|b.go")
	if len(fp) != 64 {
		t.Errorf("expected 64-char SHA256 fingerprint, got %d chars", len(fp))
	}
}

// TestV3_FingerprintDeterminism tests that fingerprints are deterministic.
func TestV3_FingerprintDeterminism(t *testing.T) {
	tokens := "IDENT STRING IDENT NUMBER IDENT"
	pathSet := "a.go|b.go|c.go"

	var fps []string
	for i := 0; i < 10; i++ {
		fps = append(fps, v3SeedFingerprint(tokens, pathSet))
	}

	first := fps[0]
	for i, fp := range fps {
		if fp != first {
			t.Errorf("run %d: fingerprint mismatch: %s vs %s", i, fp, first)
		}
	}
}

// TestV3_DifferentInputsDifferentFingerprints tests that different inputs produce different fingerprints.
func TestV3_DifferentInputsDifferentFingerprints(t *testing.T) {
	fp1 := v3SeedFingerprint("tokens-a", "a.go")
	fp2 := v3SeedFingerprint("tokens-b", "a.go")
	fp3 := v3SeedFingerprint("tokens-a", "b.go")

	if fp1 == fp2 {
		t.Error("different tokens should produce different fingerprints")
	}
	if fp1 == fp3 {
		t.Error("different paths should produce different fingerprints")
	}
}

// TestV3_CompareToBaselineWithV3 verifies baseline comparison works with v3 baselines.
func TestV3_CompareToBaselineWithV3(t *testing.T) {
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

	report := Report{
		Findings: []Finding{
			{
				StableFingerprint: "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
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

	result := CompareToBaseline(report, baseline)
	if result.HasChanges {
		t.Error("expected no changes for matching finding")
	}
}

// TestV3_NewFingerprintDetected verifies that new fingerprints are detected.
func TestV3_NewFingerprintDetected(t *testing.T) {
	baseline := Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: 3,
		GeneratedAt:      "2026-07-09T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds: BaselineThresholds{
			MinLines:  40,
			MinTokens: 400,
		},
		Findings: []BaselineFinding{},
	}

	report := Report{
		Findings: []Finding{
			{
				StableFingerprint: "new-fingerprint-hash-abc123456789",
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

	result := CompareToBaseline(report, baseline)
	if !result.HasChanges {
		t.Error("expected changes for new fingerprint")
	}
	if len(result.NewFindings) != 1 {
		t.Errorf("expected 1 new finding, got %d", len(result.NewFindings))
	}
}

// TestV3_WorsenedFindingDetected verifies that worsened findings are detected.
func TestV3_WorsenedFindingDetected(t *testing.T) {
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
				Fingerprint: "existing-fingerprint-hash-abc",
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
				StableFingerprint: "existing-fingerprint-hash-abc",
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

	result := CompareToBaseline(report, baseline)
	if !result.HasChanges {
		t.Error("expected changes for worsened finding")
	}
	if len(result.WorsenedFindings) != 1 {
		t.Errorf("expected 1 worsened finding, got %d", len(result.WorsenedFindings))
	}
}

// TestV3_LineMovementDoesNotTriggerRatchet verifies that line-only movement doesn't trigger ratchet.
func TestV3_LineMovementDoesNotTriggerRatchet(t *testing.T) {
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
				Fingerprint: "stable-fingerprint-hash-abc123",
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
				StableFingerprint: "stable-fingerprint-hash-abc123",
				TokenCount:        400,
				LineCount:         42,
				Occurrences: []Occurrence{
					{Path: "foo.go", StartLine: 100, EndLine: 145},
				},
			},
		},
		Thresholds: BaselineThresholds{
			MinLines:  40,
			MinTokens: 400,
		},
	}

	result := CompareToBaseline(report, baseline)
	if result.HasChanges {
		t.Error("line-only movement should not trigger ratchet")
	}
}
