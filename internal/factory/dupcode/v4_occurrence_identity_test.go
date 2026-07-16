// Package dupcode provides focused tests for V4 occurrence identity.
//
// These tests isolate the dedup identity and the line-geometry
// invariant from the full production pipeline. The dedup identity is
// (Path, StartPos, EndPos); line geometry is asserted to be consistent
// for any pair of occurrences sharing the token-position identity.
package dupcode

import (
	"testing"
)

func TestV4OccurrenceKey_IgnoresLineGeometry(t *testing.T) {
	// Two occurrences with same Path+StartPos+EndPos but different
	// line geometry must produce the same identity key.
	a := maximalOccurrence{Path: "a.go", StartPos: 10, EndPos: 100, StartLine: 3, EndLine: 7}
	b := maximalOccurrence{Path: "a.go", StartPos: 10, EndPos: 100, StartLine: 4, EndLine: 9}
	if occurrenceKey(a) != occurrenceKey(b) {
		t.Errorf("expected occurrenceKey to ignore line geometry, got %q vs %q",
			occurrenceKey(a), occurrenceKey(b))
	}
}

func TestV4OccurrenceKey_DifferentPathsAreDistinct(t *testing.T) {
	a := maximalOccurrence{Path: "a.go", StartPos: 10, EndPos: 100}
	b := maximalOccurrence{Path: "b.go", StartPos: 10, EndPos: 100}
	if occurrenceKey(a) == occurrenceKey(b) {
		t.Errorf("expected different paths to produce different keys, both %q", occurrenceKey(a))
	}
}

func TestV4OccurrenceKey_DifferentPositionsAreDistinct(t *testing.T) {
	a := maximalOccurrence{Path: "a.go", StartPos: 10, EndPos: 100}
	b := maximalOccurrence{Path: "a.go", StartPos: 10, EndPos: 200}
	if occurrenceKey(a) == occurrenceKey(b) {
		t.Errorf("expected different positions to produce different keys, both %q", occurrenceKey(a))
	}
}

func TestV4OccurrenceIdentityInvariants_InconsistentLinesPanic(t *testing.T) {
	// assertOccurrenceIdentityInvariants must panic when two
	// occurrences share a token-position key but disagree on
	// StartLine or EndLine.
	occs := []maximalOccurrence{
		{Path: "a.go", StartPos: 10, EndPos: 100, StartLine: 3, EndLine: 7},
		{Path: "a.go", StartPos: 10, EndPos: 100, StartLine: 4, EndLine: 9},
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic on inconsistent line geometry")
			return
		}
	}()
	assertOccurrenceIdentityInvariants(occs)
}

func TestV4OccurrenceIdentityInvariants_ConsistentLinesPass(t *testing.T) {
	occs := []maximalOccurrence{
		{Path: "a.go", StartPos: 10, EndPos: 100, StartLine: 3, EndLine: 7},
		{Path: "a.go", StartPos: 10, EndPos: 100, StartLine: 3, EndLine: 7},
	}
	assertOccurrenceIdentityInvariants(occs)
}

func TestV4OccurrenceIdentityInvariants_DifferentPathsPass(t *testing.T) {
	occs := []maximalOccurrence{
		{Path: "a.go", StartPos: 10, EndPos: 100, StartLine: 3, EndLine: 7},
		{Path: "b.go", StartPos: 10, EndPos: 100, StartLine: 3, EndLine: 7},
	}
	assertOccurrenceIdentityInvariants(occs)
}

func TestV4OccurrenceIdentityInvariants_DifferentPositionsPass(t *testing.T) {
	occs := []maximalOccurrence{
		{Path: "a.go", StartPos: 10, EndPos: 100, StartLine: 3, EndLine: 7},
		{Path: "a.go", StartPos: 200, EndPos: 300, StartLine: 20, EndLine: 30},
	}
	assertOccurrenceIdentityInvariants(occs)
}

func TestV4OccurrenceFromChain_DedupsAcrossSides(t *testing.T) {
	// A chain whose left and right contribute the SAME occurrence
	// (same Path, StartPos, EndPos) must record it once.
	c := cloneChain{
		Matches: []seedMatch{
			{
				Left:  rawWindow{Path: "a.go", StartPos: 3, EndPos: 402, StartLine: 3, EndLine: 70},
				Right: rawWindow{Path: "b.go", StartPos: 3, EndPos: 402, StartLine: 3, EndLine: 70},
			},
		},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 3, EndPos: 402},
		RightRange: tokenRange{StartPos: 3, EndPos: 402},
		Offset:     0,
	}
	got := v4OccurrenceFromChain(c)
	if len(got) != 2 {
		t.Fatalf("expected 2 unique occurrences (a.go, b.go), got %d", len(got))
	}
	// Both must survive dedup because they have different paths.
}

func TestV4OccurrenceFromChain_PreservesMultipleSameFileSpans(t *testing.T) {
	// Two non-overlapping occurrences in the same file are both
	// retained, because they carry different (Path, StartPos, EndPos)
	// tuples.
	c := cloneChain{
		Matches: []seedMatch{
			{
				Left: rawWindow{Path: "a.go", StartPos: 3, EndPos: 402, StartLine: 3, EndLine: 70},
				Right: rawWindow{
					Path:      "a.go",
					StartPos:  500,
					EndPos:    899,
					StartLine: 87,
					EndLine:   155,
				},
			},
		},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 3, EndPos: 402},
		RightRange: tokenRange{StartPos: 500, EndPos: 899},
		Offset:     497,
	}
	got := v4OccurrenceFromChain(c)
	if len(got) != 2 {
		t.Fatalf("expected 2 non-overlapping same-file occurrences, got %d", len(got))
	}
}
