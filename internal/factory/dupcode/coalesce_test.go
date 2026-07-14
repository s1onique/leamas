// Package dupcode provides tests for duplicate code detection with coalescing.
package dupcode

import (
	"testing"
)

func TestCoalesceWindows_Overlapping(t *testing.T) {
	// Test: Many overlapping windows in the same file should become one
	windows := []rawWindow{
		{Path: "file.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
		{Path: "file.go", StartLine: 15, EndLine: 55, StartPos: 5, EndPos: 45},
		{Path: "file.go", StartLine: 20, EndLine: 60, StartPos: 10, EndPos: 50},
		{Path: "file.go", StartLine: 25, EndLine: 65, StartPos: 15, EndPos: 55},
		{Path: "file.go", StartLine: 30, EndLine: 70, StartPos: 20, EndPos: 60},
	}

	result := coalesceWindows(windows)

	if len(result) != 1 {
		t.Errorf("expected 1 coalesced occurrence, got %d", len(result))
	}

	if result[0].StartLine != 10 {
		t.Errorf("expected StartLine=10, got %d", result[0].StartLine)
	}
	if result[0].EndLine != 70 {
		t.Errorf("expected EndLine=70, got %d", result[0].EndLine)
	}
}

func TestCoalesceWindows_Disjoint(t *testing.T) {
	// Test: Disjoint windows in the same file should remain separate
	windows := []rawWindow{
		{Path: "file.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
		{Path: "file.go", StartLine: 15, EndLine: 55, StartPos: 5, EndPos: 45},
		{Path: "file.go", StartLine: 100, EndLine: 140, StartPos: 100, EndPos: 140},
		{Path: "file.go", StartLine: 105, EndLine: 145, StartPos: 105, EndPos: 145},
	}

	result := coalesceWindows(windows)

	if len(result) != 2 {
		t.Errorf("expected 2 coalesced occurrences, got %d", len(result))
	}

	if result[0].StartLine != 10 || result[0].EndLine != 55 {
		t.Errorf("first occurrence: expected 10-55, got %d-%d", result[0].StartLine, result[0].EndLine)
	}
	if result[1].StartLine != 100 || result[1].EndLine != 145 {
		t.Errorf("second occurrence: expected 100-145, got %d-%d", result[1].StartLine, result[1].EndLine)
	}
}

func TestCoalesceWindows_MultipleFiles(t *testing.T) {
	// Test: Windows in different files should each be coalesced separately
	windows := []rawWindow{
		{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
		{Path: "a.go", StartLine: 15, EndLine: 55, StartPos: 5, EndPos: 45},
		{Path: "b.go", StartLine: 20, EndLine: 60, StartPos: 0, EndPos: 40},
		{Path: "b.go", StartLine: 25, EndLine: 65, StartPos: 5, EndPos: 45},
	}

	result := coalesceWindows(windows)

	if len(result) != 2 {
		t.Errorf("expected 2 coalesced occurrences, got %d", len(result))
	}

	// Should have one per file
	foundPaths := make(map[string]bool)
	for _, occ := range result {
		foundPaths[occ.Path] = true
	}
	if len(foundPaths) != 2 {
		t.Errorf("expected 2 unique paths, got %d", len(foundPaths))
	}
}

func TestCoalesceWindows_Contiguous(t *testing.T) {
	// Test: Directly adjacent windows (end+1 == next.start) should coalesce
	windows := []rawWindow{
		{Path: "file.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
		{Path: "file.go", StartLine: 51, EndLine: 90, StartPos: 41, EndPos: 80}, // adjacent
	}

	result := coalesceWindows(windows)

	if len(result) != 1 {
		t.Errorf("expected 1 coalesced occurrence, got %d", len(result))
	}

	if result[0].StartLine != 10 || result[0].EndLine != 90 {
		t.Errorf("expected 10-90, got %d-%d", result[0].StartLine, result[0].EndLine)
	}
}

func TestCoalesceOccurrences_Basic(t *testing.T) {
	occs := []Occurrence{
		{Path: "file.go", StartLine: 10, EndLine: 50},
		{Path: "file.go", StartLine: 20, EndLine: 60},
		{Path: "file.go", StartLine: 100, EndLine: 140},
	}

	result := coalesceOccurrences(occs)

	if len(result) != 2 {
		t.Errorf("expected 2 occurrences, got %d", len(result))
	}
}

func TestCanonicalPathSet(t *testing.T) {
	occs := []Occurrence{
		{Path: "b.go", StartLine: 10, EndLine: 50},
		{Path: "a.go", StartLine: 20, EndLine: 60},
		{Path: "c.go", StartLine: 30, EndLine: 70},
	}

	ps := canonicalPathSet(occs)
	expected := "a.go|b.go|c.go"

	if ps != expected {
		t.Errorf("expected %q, got %q", expected, ps)
	}
}

func TestCanonicalOccurrenceSet(t *testing.T) {
	occs := []Occurrence{
		{Path: "file.go", StartLine: 10, EndLine: 50},
		{Path: "file.go", StartLine: 20, EndLine: 60},
	}

	cos := canonicalOccurrenceSet(occs)
	if cos == "" {
		t.Error("expected non-empty canonical occurrence set")
	}
}

func TestComputeCoalescedFingerprint(t *testing.T) {
	// Token fingerprints and path sets for computing fingerprints
	tokenFP := "IDENT STRING IDENT NUMBER"
	pathSet1 := "a.go|b.go"
	pathSet2 := "a.go|c.go"

	hash1 := computeCoalescedFingerprint(tokenFP, pathSet1)
	hash2 := computeCoalescedFingerprint(tokenFP, pathSet1)

	if hash1 != hash2 {
		t.Error("expected deterministic fingerprint")
	}

	if len(hash1) != 64 {
		t.Errorf("expected 64-char SHA256, got %d", len(hash1))
	}

	// Different path sets should produce different hash (same token FP)
	hash3 := computeCoalescedFingerprint(tokenFP, pathSet2)
	if hash1 == hash3 {
		t.Error("expected different fingerprint for different path sets")
	}

	// Different token fingerprints should produce different hash (same path set)
	hash4 := computeCoalescedFingerprint("different tokens", pathSet1)
	if hash1 == hash4 {
		t.Error("expected different fingerprint for different token sequences")
	}
}

func TestCompareOccurrences(t *testing.T) {
	tests := []struct {
		a, b    Occurrence
		wantCmp int
	}{
		{Occurrence{"a.go", 10, 50}, Occurrence{"b.go", 10, 50}, -1},
		{Occurrence{"a.go", 20, 50}, Occurrence{"a.go", 10, 50}, 1},
		{Occurrence{"a.go", 10, 50}, Occurrence{"a.go", 10, 60}, -1},
		{Occurrence{"a.go", 10, 50}, Occurrence{"a.go", 10, 50}, 0},
	}

	for _, tc := range tests {
		got := compareOccurrences(tc.a, tc.b)
		if got != tc.wantCmp {
			t.Errorf("compareOccurrences(%v, %v) = %d, want %d", tc.a, tc.b, got, tc.wantCmp)
		}
	}
}

func TestMergeOccurrences(t *testing.T) {
	a := []Occurrence{
		{Path: "a.go", StartLine: 10, EndLine: 50},
		{Path: "c.go", StartLine: 30, EndLine: 70},
	}
	b := []Occurrence{
		{Path: "b.go", StartLine: 20, EndLine: 60},
	}

	result := mergeOccurrences(a, b)

	if len(result) != 3 {
		t.Errorf("expected 3 occurrences, got %d", len(result))
	}

	if result[0].Path != "a.go" {
		t.Errorf("expected first path a.go, got %s", result[0].Path)
	}
}

func TestMergeOccurrences_WithOverlap(t *testing.T) {
	a := []Occurrence{
		{Path: "file.go", StartLine: 10, EndLine: 50},
	}
	b := []Occurrence{
		{Path: "file.go", StartLine: 40, EndLine: 80}, // overlaps
	}

	result := mergeOccurrences(a, b)

	if len(result) != 1 {
		t.Errorf("expected 1 coalesced occurrence, got %d", len(result))
	}

	if result[0].StartLine != 10 || result[0].EndLine != 80 {
		t.Errorf("expected 10-80, got %d-%d", result[0].StartLine, result[0].EndLine)
	}
}

func TestComputeLineCount(t *testing.T) {
	occs := []Occurrence{
		{Path: "a.go", StartLine: 10, EndLine: 50},
		{Path: "b.go", StartLine: 20, EndLine: 80},
		{Path: "c.go", StartLine: 30, EndLine: 45},
	}

	count := computeLineCount(occs)
	if count != 61 {
		t.Errorf("expected max line count 61, got %d", count)
	}

	empty := computeLineCount(nil)
	if empty != 0 {
		t.Errorf("expected 0 for nil, got %d", empty)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n      int
		expect string
	}{
		{0, "0"}, {1, "1"}, {9, "9"}, {10, "10"}, {42, "42"}, {100, "100"}, {999, "999"},
	}

	for _, tc := range tests {
		got := itoa(tc.n)
		if got != tc.expect {
			t.Errorf("itoa(%d) = %q, want %q", tc.n, got, tc.expect)
		}
	}
}
