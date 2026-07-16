// Package dupcode provides focused tests for V4 shadow suppression.
//
// The tests below isolate the suppression logic from the full
// production pipeline so the within-file detection rule, the
// strict-containment rule, and the offset-invariant chain-pair key
// can be exercised independently of chain construction.
package dupcode

import (
	"reflect"
	"testing"
)

func TestV4SuppressShadowChains_OneMaximalClone(t *testing.T) {
	// One maximal chain and one shifted shadow chain sharing the
	// same (LeftPath, RightPath) pair. The shadow must be dropped.
	maximal := cloneChain{
		Matches:    []seedMatch{{Left: window("a.go", 3, 802), Right: window("b.go", 3, 802)}},
		TokenSpan:  800,
		LeftRange:  tokenRange{StartPos: 3, EndPos: 802},
		RightRange: tokenRange{StartPos: 3, EndPos: 802},
		Offset:     0,
	}
	shadow := cloneChain{
		Matches:    []seedMatch{{Left: window("a.go", 50, 449), Right: window("b.go", 50, 449)}},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 50, EndPos: 449},
		RightRange: tokenRange{StartPos: 50, EndPos: 449},
		Offset:     0,
	}
	got := v4SuppressShadowChains([]cloneChain{maximal, shadow})
	if len(got) != 1 {
		t.Fatalf("expected 1 surviving chain, got %d", len(got))
	}
	if got[0].TokenSpan != 800 {
		t.Errorf("expected maximal chain (span=800) to survive, got span=%d", got[0].TokenSpan)
	}
}

func TestV4SuppressShadowChains_DifferentOffsetsAreNotShadows(t *testing.T) {
	// Two chains with different offsets share the same file pair but
	// must NOT suppress each other. They represent independent
	// sliding-window variants of different clone bodies.
	a := cloneChain{
		Matches:    []seedMatch{{Left: window("a.go", 3, 402), Right: window("b.go", 3, 402)}},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 3, EndPos: 402},
		RightRange: tokenRange{StartPos: 3, EndPos: 402},
		Offset:     0,
	}
	b := cloneChain{
		Matches:    []seedMatch{{Left: window("a.go", 200, 599), Right: window("b.go", 300, 699)}},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 200, EndPos: 599},
		RightRange: tokenRange{StartPos: 300, EndPos: 699},
		Offset:     100,
	}
	got := v4SuppressShadowChains([]cloneChain{a, b})
	if len(got) != 2 {
		t.Fatalf("expected 2 surviving chains (different offsets), got %d", len(got))
	}
}

func TestV4SuppressShadowChains_WithinFileOverlapFiltered(t *testing.T) {
	// Within-file chain whose LeftRange overlaps its RightRange must
	// be filtered out as a sub-window self-match. The filter runs
	// only with at least two chains in the same group, so we pair
	// the self-match with a non-overlapping chain to trigger it.
	selfMatch := cloneChain{
		Matches:    []seedMatch{{Left: window("a.go", 3, 402), Right: window("a.go", 50, 449)}},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 3, EndPos: 402},
		RightRange: tokenRange{StartPos: 50, EndPos: 449},
		Offset:     47,
	}
	other := cloneChain{
		Matches:    []seedMatch{{Left: window("a.go", 500, 899), Right: window("a.go", 1000, 1399)}},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 500, EndPos: 899},
		RightRange: tokenRange{StartPos: 1000, EndPos: 1399},
		Offset:     500,
	}
	got := v4SuppressShadowChains([]cloneChain{selfMatch, other})
	if len(got) != 1 {
		t.Fatalf("expected self-match to be filtered, got %d chains", len(got))
	}
	if got[0].TokenSpan != 400 || got[0].Offset != 500 {
		t.Errorf("expected non-overlapping chain to survive, got span=%d offset=%d",
			got[0].TokenSpan, got[0].Offset)
	}
}

func TestV4SuppressShadowChains_ReversedOrientationProducesSameKey(t *testing.T) {
	// A chain whose left and right are swapped must collapse to the
	// same group key as its non-swapped counterpart, so the
	// strict-containment rule can suppress it.
	a := cloneChain{
		Matches:    []seedMatch{{Left: window("a.go", 3, 402), Right: window("b.go", 3, 402)}},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 3, EndPos: 402},
		RightRange: tokenRange{StartPos: 3, EndPos: 402},
		Offset:     0,
	}
	b := cloneChain{
		Matches:    []seedMatch{{Left: window("b.go", 3, 402), Right: window("a.go", 3, 402)}},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 3, EndPos: 402},
		RightRange: tokenRange{StartPos: 3, EndPos: 402},
		Offset:     0,
	}
	got := v4SuppressShadowChains([]cloneChain{a, b})
	// Both chains are identical-position duplicates (same offset,
	// same range) so the equal-range tie-break keeps exactly one.
	if len(got) != 1 {
		t.Fatalf("expected exactly one survivor from duplicate-range chains, got %d", len(got))
	}
}

func TestV4SuppressShadowChains_TwoIndependentChainsSurvive(t *testing.T) {
	// Two non-overlapping chains from independent bodies in the
	// same file pair must both survive.
	a := cloneChain{
		Matches:    []seedMatch{{Left: window("a.go", 3, 402), Right: window("b.go", 3, 402)}},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 3, EndPos: 402},
		RightRange: tokenRange{StartPos: 3, EndPos: 402},
		Offset:     0,
	}
	b := cloneChain{
		Matches:    []seedMatch{{Left: window("a.go", 500, 899), Right: window("b.go", 500, 899)}},
		TokenSpan:  400,
		LeftRange:  tokenRange{StartPos: 500, EndPos: 899},
		RightRange: tokenRange{StartPos: 500, EndPos: 899},
		Offset:     0,
	}
	got := v4SuppressShadowChains([]cloneChain{a, b})
	if len(got) != 2 {
		t.Fatalf("expected 2 non-overlapping chains to survive, got %d", len(got))
	}
}

func TestV4ChainPairKeyForChain_PathSetDelimiterIndependent(t *testing.T) {
	// A chain whose left token range numerically exceeds its right
	// token range must still collapse to the canonical
	// (leftPath, rightPath) key; the chain-pair key NEVER inspects
	// the encoded PathSet for within-file heuristics.
	c := cloneChain{
		Matches: []seedMatch{
			{Left: window("a.go", 3, 100), Right: window("a.go", 200, 297)},
		},
		TokenSpan:  98,
		LeftRange:  tokenRange{StartPos: 3, EndPos: 100},
		RightRange: tokenRange{StartPos: 200, EndPos: 297},
		Offset:     197,
	}
	key := chainPairKeyForChain(c, nil)
	// When analysesByPath is nil, region ordinals default to -1. The
	// key is structured, so path punctuation cannot alter identity.
	want := v4ShadowGroupKey{LeftPath: "a.go", LeftRegion: -1, RightPath: "a.go", RightRegion: -1}
	if key != want {
		t.Errorf("expected canonical within-file key %+v, got %+v", want, key)
	}
}

func TestV4ChainRangeRelationBetween_StrictContainment(t *testing.T) {
	outer := cloneChain{
		LeftRange:  tokenRange{StartPos: 0, EndPos: 100},
		RightRange: tokenRange{StartPos: 0, EndPos: 100},
	}
	inner := cloneChain{
		LeftRange:  tokenRange{StartPos: 10, EndPos: 90},
		RightRange: tokenRange{StartPos: 10, EndPos: 90},
	}
	rel := chainRangeRelationBetween(outer, inner)
	if rel != chainRangeStrictlyContains {
		t.Errorf("expected strict containment, got %v", rel)
	}
}

func TestV4ChainRangeRelationBetween_EqualDuplicates(t *testing.T) {
	a := cloneChain{
		LeftRange:  tokenRange{StartPos: 0, EndPos: 100},
		RightRange: tokenRange{StartPos: 0, EndPos: 100},
	}
	b := cloneChain{
		LeftRange:  tokenRange{StartPos: 0, EndPos: 100},
		RightRange: tokenRange{StartPos: 0, EndPos: 100},
	}
	rel := chainRangeRelationBetween(a, b)
	if rel != chainRangeEqual {
		t.Errorf("expected chainRangeEqual, got %v", rel)
	}
}

func TestV4ChainRangeRelationBetween_Unrelated(t *testing.T) {
	a := cloneChain{
		LeftRange:  tokenRange{StartPos: 0, EndPos: 100},
		RightRange: tokenRange{StartPos: 0, EndPos: 100},
	}
	b := cloneChain{
		LeftRange:  tokenRange{StartPos: 200, EndPos: 300},
		RightRange: tokenRange{StartPos: 200, EndPos: 300},
	}
	rel := chainRangeRelationBetween(a, b)
	if rel != chainRangeUnrelated {
		t.Errorf("expected chainRangeUnrelated, got %v", rel)
	}
}

func TestV4TokenRangesOverlap_AdjacencyIsNotOverlap(t *testing.T) {
	// Adjacent ranges share no token position.
	a := rawWindow{StartPos: 0, EndPos: 99}
	b := rawWindow{StartPos: 100, EndPos: 199}
	if tokenRangesOverlap(a, b) {
		t.Errorf("expected adjacency (start=end+1) NOT to overlap")
	}
}

func TestV4TokenRangesOverlap_PartialOverlapDetected(t *testing.T) {
	a := rawWindow{StartPos: 0, EndPos: 150}
	b := rawWindow{StartPos: 100, EndPos: 199}
	if !tokenRangesOverlap(a, b) {
		t.Errorf("expected partial overlap to be detected")
	}
}

func TestV4TokenRangesOverlap_ContainmentDetected(t *testing.T) {
	a := rawWindow{StartPos: 0, EndPos: 199}
	b := rawWindow{StartPos: 50, EndPos: 99}
	if !tokenRangesOverlap(a, b) {
		t.Errorf("expected containment to be detected")
	}
}

// window is a focused-test helper that builds a rawWindow at the
// supplied token positions.
func window(path string, startPos, endPos int) rawWindow {
	return rawWindow{Path: path, StartPos: startPos, EndPos: endPos, StartLine: 1, EndLine: 1}
}

// guard against unused-import lint
var _ = reflect.DeepEqual
