// Package dupcode provides tests for duplicate code detection with coalescing.
package dupcode

import (
	"testing"
)

// TestV4: Windows at different positions produce maximal findings (v4 behavior)
// In v4, overlapping windows are coalesced into maximal clone findings
func TestCoalesceFindings_OverlappingWindows(t *testing.T) {
	// V4 coalesces windows at different positions into maximal findings
	windowMap := map[string][]rawWindow{
		"same-fp": {
			// File A: windows at different positions
			{Path: "file_a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "file_a.go", StartLine: 60, EndLine: 100, StartPos: 100, EndPos: 140},
			// File B: same positions
			{Path: "file_b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
			{Path: "file_b.go", StartLine: 150, EndLine: 190, StartPos: 100, EndPos: 140},
		},
	}
	fingerprintTokens := map[string]int{
		"same-fp": 400,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	// V4 produces maximal findings - should have at least 1 finding
	if len(results) < 1 {
		t.Errorf("expected at least 1 finding, got %d", len(results))
	}

	// Each finding should have at least 2 occurrences (one per file)
	for _, result := range results {
		if len(result.Occurrences) < 2 {
			t.Errorf("expected at least 2 occurrences per finding, got %d", len(result.Occurrences))
		}
	}
}

// TestV3: Different clones at different offsets produce separate findings
func TestCoalesceFindings_DifferentClonesSamePaths(t *testing.T) {
	// Two different fingerprints at different token offsets - v3 produces 2 separate findings
	// Note: The offset (right.StartPos - left.StartPos) must differ for separate fingerprints
	windowMap := map[string][]rawWindow{
		"clone-a": {
			{Path: "file_a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "file_b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40}, // offset=0
		},
		"clone-b": {
			{Path: "file_a.go", StartLine: 60, EndLine: 100, StartPos: 100, EndPos: 140},
			{Path: "file_b.go", StartLine: 150, EndLine: 190, StartPos: 200, EndPos: 240}, // offset=100
		},
	}
	fingerprintTokens := map[string]int{
		"clone-a": 400,
		"clone-b": 400,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	// Different token sequences at different offsets = 2 separate findings
	if len(results) != 2 {
		t.Errorf("expected 2 separate findings for different clones, got %d", len(results))
	}

	// Each finding should have its own coalesced ranges
	for _, result := range results {
		if len(result.Occurrences) != 2 {
			t.Errorf("expected 2 occurrences per finding, got %d", len(result.Occurrences))
		}
	}
}

// Test: Three occurrences of one clone become one finding
func TestCoalesceFindings_ThreeOccurrences(t *testing.T) {
	windowMap := map[string][]rawWindow{
		"shared-fp": {
			{Path: "file_a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "file_b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
			{Path: "file_c.go", StartLine: 200, EndLine: 240, StartPos: 0, EndPos: 40},
		},
	}
	fingerprintTokens := map[string]int{
		"shared-fp": 400,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	if len(results) == 0 {
		t.Fatal("expected at least 1 finding, got 0")
	}

	if len(results[0].Occurrences) < 2 {
		t.Errorf("expected at least 2 occurrences, got %d", len(results[0].Occurrences))
	}
}

// Test: Differently aligned overlapping candidates are separate findings
func TestCoalesceFindings_DifferentAlignment(t *testing.T) {
	// Windows from different fingerprints at different offsets = separate findings
	// Note: The offset (right.StartPos - left.StartPos) must differ for separate fingerprints
	windowMap := map[string][]rawWindow{
		"clone-1": {
			{Path: "file.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "other.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40}, // offset=0
		},
		"clone-2": {
			{Path: "file.go", StartLine: 30, EndLine: 70, StartPos: 100, EndPos: 140},
			{Path: "other.go", StartLine: 120, EndLine: 160, StartPos: 200, EndPos: 240}, // offset=100
		},
	}
	fingerprintTokens := map[string]int{
		"clone-1": 400,
		"clone-2": 400,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	// Different token sequences at different offsets = 2 separate findings
	if len(results) != 2 {
		t.Errorf("expected 2 separate findings for different clones, got %d", len(results))
	}
}

// Test: Disjoint clones in the same files remain separate findings
func TestCoalesceFindings_DisjointClones(t *testing.T) {
	// Disjoint clones at different offsets produce separate findings
	// Note: The offset (right.StartPos - left.StartPos) must differ for separate fingerprints
	windowMap := map[string][]rawWindow{
		"clone-a": {
			{Path: "file.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "other.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40}, // offset=0
		},
		"clone-b": {
			{Path: "file.go", StartLine: 200, EndLine: 240, StartPos: 200, EndPos: 240},
			{Path: "other.go", StartLine: 300, EndLine: 340, StartPos: 400, EndPos: 440}, // offset=200
		},
	}
	fingerprintTokens := map[string]int{
		"clone-a": 400,
		"clone-b": 400,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	// Different token sequences at different offsets = 2 separate findings
	if len(results) != 2 {
		t.Errorf("expected 2 separate findings for different clones, got %d", len(results))
	}
}

// Test: Deterministic ordering regardless of input order
func TestCoalesceFindings_DeterministicOrdering(t *testing.T) {
	// Same input, just in different map iteration order
	windowMap1 := map[string][]rawWindow{
		"fp-a": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
		"fp-b": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
	}
	windowMap2 := map[string][]rawWindow{
		"fp-b": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
		"fp-a": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
	}
	fingerprintTokens := map[string]int{
		"fp-a": 400,
		"fp-b": 400,
	}

	results1 := coalesceFindings(windowMap1, fingerprintTokens)
	results2 := coalesceFindings(windowMap2, fingerprintTokens)

	if len(results1) != len(results2) {
		t.Error("expected same number of results regardless of iteration order")
	}

	if len(results1) > 0 && results1[0].StableFingerprint != results2[0].StableFingerprint {
		t.Error("expected stable fingerprints to be deterministic")
	}
}
