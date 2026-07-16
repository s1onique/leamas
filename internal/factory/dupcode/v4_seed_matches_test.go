// Package dupcode provides tests for seed match generation.
package dupcode

import (
	"testing"
)

// TestBuildSeedMatches_AllPairsWithSignedOffset proves that buildSeedMatches
// generates all-pairs matches with signed offsets, not just offset=0.
func TestBuildSeedMatches_AllPairsWithSignedOffset(t *testing.T) {
	windows := []rawWindow{
		{Path: "a.go", StartPos: 100, EndPos: 499, StartLine: 10, EndLine: 50},
		{Path: "b.go", StartPos: 800, EndPos: 1199, StartLine: 100, EndLine: 140},
	}

	matches := buildSeedMatches("test-fp", windows)

	// Should have exactly 1 match (1 window in a.go × 1 window in b.go)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	m := matches[0]
	if m.Offset != 700 {
		t.Errorf("expected Offset=700 (800-100), got %d", m.Offset)
	}
	if m.Left.Path != "a.go" {
		t.Errorf("expected Left.Path=a.go, got %s", m.Left.Path)
	}
	if m.Right.Path != "b.go" {
		t.Errorf("expected Right.Path=b.go, got %s", m.Right.Path)
	}
}

// TestBuildSeedMatches_MultipleWindows produces all pairs including within-file.
func TestBuildSeedMatches_MultipleWindows(t *testing.T) {
	// Windows are non-overlapping: a[0-99], a[200-299], b[100-199], b[400-499]
	windows := []rawWindow{
		{Path: "a.go", StartPos: 0, EndPos: 99, StartLine: 10, EndLine: 50},
		{Path: "a.go", StartPos: 200, EndPos: 299, StartLine: 60, EndLine: 100},
		{Path: "b.go", StartPos: 100, EndPos: 199, StartLine: 100, EndLine: 140},
		{Path: "b.go", StartPos: 400, EndPos: 499, StartLine: 150, EndLine: 190},
	}

	matches := buildSeedMatches("test-fp", windows)

	// Should have 6 matches total:
	// - 2 within-file (a.go: 1 pair, b.go: 1 pair) - both non-overlapping
	// - 4 cross-file (2 a.go × 2 b.go)
	if len(matches) != 6 {
		t.Fatalf("expected 6 matches (2 within + 4 cross-file), got %d", len(matches))
	}

	// Verify we have non-zero offsets (proves all-pairs matching)
	hasNonZeroOffset := false
	for _, m := range matches {
		if m.Offset != 0 {
			hasNonZeroOffset = true
			break
		}
	}
	if !hasNonZeroOffset {
		t.Error("expected at least one non-zero offset")
	}
}

// TestBuildSeedMatches_WithinFileRequiresNonOverlap verifies within-file matches
// must be non-overlapping.
func TestBuildSeedMatches_WithinFileRequiresNonOverlap(t *testing.T) {
	// Overlapping windows in same file
	windows := []rawWindow{
		{Path: "a.go", StartPos: 0, EndPos: 99, StartLine: 10, EndLine: 50},
		{Path: "a.go", StartPos: 50, EndPos: 149, StartLine: 40, EndLine: 80}, // overlaps
	}

	matches := buildSeedMatches("test-fp", windows)

	// No within-file match because windows overlap
	// Should have 0 matches (only 1 file)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for overlapping same-file windows, got %d", len(matches))
	}
}

// TestBuildSeedMatches_SingleFile returns within-file matches.
func TestBuildSeedMatches_SingleFile(t *testing.T) {
	windows := []rawWindow{
		{Path: "a.go", StartPos: 0, EndPos: 99, StartLine: 10, EndLine: 50},
		{Path: "a.go", StartPos: 200, EndPos: 299, StartLine: 60, EndLine: 100},
	}

	matches := buildSeedMatches("test-fp", windows)

	// Single file - 1 within-file match (for repeated occurrences, non-overlapping)
	if len(matches) != 1 {
		t.Errorf("expected 1 match for single file (within-file), got %d", len(matches))
	}
}
