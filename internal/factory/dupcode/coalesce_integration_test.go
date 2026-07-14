// Package dupcode provides integration regression tests for clone detection.
package dupcode

import (
	"testing"
)

// TestIntegration_CoalescingBehavior verifies coalescing logic via coalesceFindings directly.
// With all-pairs matching, windows at the same position chain together.
func TestIntegration_CoalescingBehavior(t *testing.T) {
	// Simulate production scenario: windows at same position across files
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

	// With all-pairs matching:
	// - offset=0: file_a[0]-file_b[0] and file_a[400]-file_b[400] chain together → 1 finding
	// - offset=400: file_a[0]-file_b[400] → 1 finding
	// - offset=-400: file_a[400]-file_b[0] → 1 finding
	// Total: 3 findings
	if len(results) != 3 {
		t.Errorf("expected 3 findings, got %d", len(results))
	}

	// First finding should be the chained one at offset=0
	if len(results) > 0 {
		r := results[0]
		// Should have 2 occurrences (one per file) for the chained finding
		if len(r.Occurrences) != 2 {
			t.Errorf("expected 2 occurrences for chained finding, got %d", len(r.Occurrences))
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
