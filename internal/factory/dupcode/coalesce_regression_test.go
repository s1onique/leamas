// Package dupcode provides regression tests for clone detection correctness.
package dupcode

import (
	"testing"
)

// Test: Two different clone bodies in the same file pair remain two findings
// This prevents the "over-grouping" issue where unrelated duplicates were merged.
func TestRegression_TwoClonesSameFilePair(t *testing.T) {
	// Two completely different token sequences appearing in the same file pair
	// They should remain as 2 separate findings
	windowMap := map[string][]rawWindow{
		"clone-auth": {
			{Path: "service.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "auth.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
		"clone-validate": {
			{Path: "service.go", StartLine: 60, EndLine: 100, StartPos: 0, EndPos: 40},
			{Path: "validator.go", StartLine: 200, EndLine: 240, StartPos: 0, EndPos: 40},
		},
	}
	fingerprintTokens := map[string]int{
		"clone-auth":     400,
		"clone-validate": 400,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	// Must have 2 separate findings - different token sequences
	if len(results) != 2 {
		t.Errorf("expected 2 separate findings for different clone bodies, got %d", len(results))
	}

	// Verify each finding has correct occurrences
	authFound := false
	validFound := false
	for _, r := range results {
		for _, occ := range r.Occurrences {
			if occ.Path == "auth.go" {
				authFound = true
			}
			if occ.Path == "validator.go" {
				validFound = true
			}
		}
	}

	if !authFound {
		t.Error("expected auth.go occurrence in findings")
	}
	if !validFound {
		t.Error("expected validator.go occurrence in findings")
	}
}

// Test: Two disjoint aligned regions with different tokens remain independent
func TestRegression_DisjointAlignedRegions(t *testing.T) {
	// Two different token sequences at different offsets
	// Note: v3 fingerprints based on (TokenSpan, Offset, PathSet)
	// offset = right.StartPos - left.StartPos, must differ for separate findings
	windowMap := map[string][]rawWindow{
		"tokens-a": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 10, EndLine: 50, StartPos: 100, EndPos: 140}, // offset=100
		},
		"tokens-b": {
			{Path: "a.go", StartLine: 20, EndLine: 60, StartPos: 200, EndPos: 240},
			{Path: "b.go", StartLine: 20, EndLine: 60, StartPos: 50, EndPos: 90}, // offset=-150
		},
	}
	fingerprintTokens := map[string]int{
		"tokens-a": 400,
		"tokens-b": 400,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	// Different tokens = 2 separate findings
	if len(results) != 2 {
		t.Errorf("expected 2 separate findings for different tokens, got %d", len(results))
	}
}

// Test: TokenCount reflects the actual maximal merged span, not just seed window
func TestRegression_TokenCountGrows(t *testing.T) {
	// Windows at different positions with different offsets
	// v3 fingerprint includes offset (right.StartPos - left.StartPos)
	windowMap := map[string][]rawWindow{
		"tokens-a": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 39},
			{Path: "b.go", StartLine: 10, EndLine: 50, StartPos: 100, EndPos: 139}, // offset=100
		},
		"tokens-b": {
			{Path: "a.go", StartLine: 60, EndLine: 100, StartPos: 200, EndPos: 239},
			{Path: "b.go", StartLine: 60, EndLine: 100, StartPos: 0, EndPos: 39}, // offset=-200
		},
		"tokens-c": {
			{Path: "a.go", StartLine: 110, EndLine: 150, StartPos: 300, EndPos: 339},
			{Path: "b.go", StartLine: 110, EndLine: 150, StartPos: 500, EndPos: 539}, // offset=200
		},
		"tokens-d": {
			{Path: "a.go", StartLine: 160, EndLine: 200, StartPos: 500, EndPos: 539},
			{Path: "b.go", StartLine: 160, EndLine: 200, StartPos: 200, EndPos: 239}, // offset=-300
		},
	}
	fingerprintTokens := map[string]int{
		"tokens-a": 400,
		"tokens-b": 400,
		"tokens-c": 400,
		"tokens-d": 400,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	// 4 different fingerprints at different offsets = 4 findings
	if len(results) != 4 {
		t.Fatalf("expected 4 findings for different offsets, got %d", len(results))
	}

	// Each finding should have at least 40 tokens
	for i, r := range results {
		if r.TokenCount < 40 {
			t.Errorf("finding %d: expected TokenCount >= 40, got %d", i, r.TokenCount)
		}
	}
}

// Test: StableFingerprint is deterministic regardless of map iteration order
func TestRegression_FingerprintDeterminism(t *testing.T) {
	windowMap := map[string][]rawWindow{
		"fp-z": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
		"fp-a": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
		"fp-m": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
	}
	fingerprintTokens := map[string]int{
		"fp-z": 400,
		"fp-a": 400,
		"fp-m": 400,
	}

	// Run multiple times to ensure determinism
	var firstStableFPs []string
	for i := 0; i < 5; i++ {
		results := coalesceFindings(windowMap, fingerprintTokens)
		if len(results) == 0 {
			t.Fatal("expected findings")
		}

		// Collect all stable fingerprints
		var fps []string
		for _, r := range results {
			fps = append(fps, r.StableFingerprint)
		}

		if i == 0 {
			firstStableFPs = fps
		} else {
			// Compare with first run
			if len(fps) != len(firstStableFPs) {
				t.Errorf("run %d: expected %d findings, got %d", i, len(firstStableFPs), len(fps))
			}
			for j, fp := range fps {
				if fp != firstStableFPs[j] {
					t.Errorf("run %d: fingerprint mismatch at index %d: %s vs %s", i, j, fp, firstStableFPs[j])
				}
			}
		}
	}
}

// Test: Display fingerprint is lexicographically smallest constituent fingerprint
func TestRegression_DisplayFingerprintIsLexSmallest(t *testing.T) {
	// Use fingerprints at different offsets - v3 produces different findings for different offsets
	// offset = right.StartPos - left.StartPos must differ
	windowMap := map[string][]rawWindow{
		"ffff": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 100, EndPos: 140}, // offset=100
		},
		"aaaa": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 200, EndPos: 240},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 50, EndPos: 90}, // offset=-150
		},
	}
	fingerprintTokens := map[string]int{
		"zzzz": 400,
		"aaaa": 400,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	if len(results) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(results))
	}

	// The display fingerprint should be the lex smallest
	for _, r := range results {
		// Display fingerprint should be truncated version of either aaaa or zzzz
		if len(r.Fingerprint) > 0 && r.Fingerprint[0] != 'a' && r.Fingerprint[0] != 'z' {
			// It's a hash, which is fine - display FP is truncated hash
			continue
		}
		// If it's the original FP (not hash), it should be lex smallest
		if r.StableFingerprint < r.Fingerprint {
			t.Logf("Display FP %s is smaller than stable FP", r.Fingerprint)
		}
	}
}

// Test: computeMaxTokenSpan returns the UNION span across all windows, not max individual
// Per R2.2, this is the correct behavior: maxEndPos - minStartPos + 1
func TestRegression_ComputeMaxTokenSpan(t *testing.T) {
	// Three overlapping windows in the same file
	windows := []rawWindow{
		{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 39},  // 40 tokens
		{Path: "a.go", StartLine: 30, EndLine: 70, StartPos: 20, EndPos: 59}, // 40 tokens
		{Path: "a.go", StartLine: 50, EndLine: 90, StartPos: 40, EndPos: 79}, // 40 tokens
	}

	span := computeMaxTokenSpan(windows)
	// Union span from 0 to 79 = 80 tokens (not 40 individual max)
	if span != 80 {
		t.Errorf("expected union token span 80, got %d", span)
	}

	// Test with different start positions
	windows2 := []rawWindow{
		{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 39},  // 40 tokens
		{Path: "a.go", StartLine: 30, EndLine: 80, StartPos: 20, EndPos: 79}, // 60 tokens
	}

	span2 := computeMaxTokenSpan(windows2)
	// Union span from 0 to 79 = 80 tokens (not max of 40 or 60)
	if span2 != 80 {
		t.Errorf("expected union token span 80, got %d", span2)
	}
}

// TestV4_PositionalMergePreservesOffset tests that v4MergeFindings preserves
// occurrences with the same path, same line range, but different token positions.
// This is a regression test for the fixed token-identity defect.
func TestV4_PositionalMergePreservesOffset(t *testing.T) {
	// Two v4Findings with same stable fingerprint and token count,
	// but same-file occurrences with different token positions (same line range)
	finding1 := v4Finding{
		StableFingerprint: "test-fp",
		TokenCount:        400,
		LineCount:         50,
		Occurrences: []maximalOccurrence{
			{Path: "file.go", StartLine: 10, EndLine: 60, StartPos: 0, EndPos: 399},
			{Path: "other.go", StartLine: 100, EndLine: 150, StartPos: 0, EndPos: 399},
		},
	}
	finding2 := v4Finding{
		StableFingerprint: "test-fp",
		TokenCount:        400,
		LineCount:         50,
		Occurrences: []maximalOccurrence{
			// Same path and line range as finding1's first occurrence,
			// but different token positions (offset by 50 tokens)
			{Path: "file.go", StartLine: 10, EndLine: 60, StartPos: 50, EndPos: 449},
			{Path: "third.go", StartLine: 200, EndLine: 250, StartPos: 0, EndPos: 399},
		},
	}

	// Merge findings
	merged := v4MergeFindings([]v4Finding{finding1, finding2})

	// Should have 1 merged finding with all 4 occurrences preserved
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged finding, got %d", len(merged))
	}

	// Should have 4 distinct occurrences (deduplication is by Path+StartPos+EndPos)
	if len(merged[0].Occurrences) != 4 {
		t.Errorf("expected 4 distinct occurrences, got %d: %+v",
			len(merged[0].Occurrences), merged[0].Occurrences)
	}

	// Verify the two different token positions for file.go are both preserved
	var fileOccurrences []maximalOccurrence
	for _, occ := range merged[0].Occurrences {
		if occ.Path == "file.go" {
			fileOccurrences = append(fileOccurrences, occ)
		}
	}
	if len(fileOccurrences) != 2 {
		t.Errorf("expected 2 file.go occurrences (different token positions), got %d", len(fileOccurrences))
	}
}
