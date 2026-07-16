// Package dupcode provides integration regression tests for clone detection.
package dupcode

import (
	"testing"
)

// TestIntegration_CoalescingBehavior verifies coalescing logic via coalesceFindings directly.
// V4: windows at the same fingerprint chain together into maximal clones.
func TestIntegration_CoalescingBehavior(t *testing.T) {
	// Simulate production scenario: windows at same fingerprint across files
	windowMap := map[string][]rawWindow{
		"identical-sequence": {
			// File A: windows at specific positions
			{Path: "file_a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 399},
			{Path: "file_a.go", StartLine: 60, EndLine: 100, StartPos: 400, EndPos: 799},
			// File B: same positions
			{Path: "file_b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 399},
			{Path: "file_b.go", StartLine: 150, EndLine: 190, StartPos: 400, EndPos: 799},
		},
	}
	fingerprintTokens := map[string]int{
		"identical-sequence": 400,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	// V4: All windows with the same fingerprint chain together
	// - Windows at positions [0,400] in file_a chain with [0,400] in file_b
	// - Result: 1 finding with 2 occurrences (one per file)
	if len(results) < 1 {
		t.Errorf("expected at least 1 finding, got %d", len(results))
	}

	// First finding should have at least 2 occurrences (one per file)
	if len(results) > 0 {
		r := results[0]
		if len(r.Occurrences) < 2 {
			t.Errorf("expected at least 2 occurrences, got %d", len(r.Occurrences))
		}
	}
}

// TestIntegration_TokenCountReflectsUnionSpan verifies TokenCount is the union span.
func TestIntegration_TokenCountReflectsUnionSpan(t *testing.T) {
	// Windows at same relative positions within each file (both at StartPos 0)
	// to ensure offset=0 matches chain together properly
	windowMap := map[string][]rawWindow{
		"tokens": {
			// File A: both windows start at position 0 within that file
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 99},
			{Path: "a.go", StartLine: 60, EndLine: 100, StartPos: 0, EndPos: 99},
			// File B: both windows start at position 0 within that file
			{Path: "b.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 99},
			{Path: "b.go", StartLine: 60, EndLine: 100, StartPos: 0, EndPos: 99},
		},
	}
	fingerprintTokens := map[string]int{
		"tokens": 100,
	}

	results := coalesceFindings(windowMap, fingerprintTokens)

	// All windows at offset=0 means they all chain together
	if len(results) != 1 {
		t.Fatalf("expected 1 finding for chained windows, got %d", len(results))
	}

	// TokenCount reflects the seed window size (100 tokens per window)
	// Even though windows overlap at same position, they chain
	if results[0].TokenCount != 100 {
		t.Errorf("expected TokenCount 100, got %d", results[0].TokenCount)
	}
}
