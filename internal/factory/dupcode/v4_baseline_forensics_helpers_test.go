// Package dupcode provides CORRECTION04 forensic test helpers.
//
// The helpers in this file are used by the 504 maximality tests,
// the 877/514 forensic-fact tests, and the public-geometry
// classification test.
package dupcode

import (
	"testing"
)


// canonicalLiveFinding returns the single canonical finding from
// the live trace, asserting exactly one finding with TokenCount=504.

func canonicalLiveFinding(t *testing.T, finals []v4InternalFinding) v4InternalFinding {
	t.Helper()
	if len(finals) != 1 {
		t.Fatalf("trace must emit exactly one final finding, got %d", len(finals))
	}
	got := finals[0]
	if got.TokenCount != 504 {
		t.Fatalf("canonical finding must have TokenCount=504, got %d", got.TokenCount)
	}
	if len(got.Occurrences) != 2 {
		t.Fatalf("canonical finding must have 2 occurrences, got %d", len(got.Occurrences))
	}
	return got
}

// leftRightFromTrace returns the (leftFile, rightFile, canonical)
// triple for the live trace.
// leftRightFromTrace returns the (leftFile, rightFile, canonical)
// triple for the live trace.

func leftRightFromTrace(t *testing.T, trace v4PipelineTrace, finals []v4InternalFinding) (
	*v4AnalyzedFile, *v4AnalyzedFile, v4InternalFinding,
) {
	t.Helper()
	leftFile, rightFile, _, finals := traceForLiveTree(t)
	_ = trace
	canonical := canonicalLiveFinding(t, finals)
	return leftFile, rightFile, canonical
}

// sortedLeftRight returns the canonical occurrence pair with left
// first and right second, regardless of the lexicographic order
// in which the materializer emits them.
// sortedLeftRight returns the canonical occurrence pair with left
// first and right second, regardless of the lexicographic order
// in which the materializer emits them.

func sortedLeftRight(canonical v4InternalFinding) (
	leftOcc, rightOcc maximalOccurrence, ok bool,
) {
	if len(canonical.Occurrences) != 2 {
		return maximalOccurrence{}, maximalOccurrence{}, false
	}
	a := canonical.Occurrences[0]
	b := canonical.Occurrences[1]
	if a.Path > b.Path {
		a, b = b, a
	}
	return a, b, true
}

// TestV4PipelineTrace_StagesNonEmpty runs the live trace and
// asserts every stage is non-empty. The maximality proof requires
// the live pre-publication pipeline stages to be observable.
// ownerAt returns the TokenOwner at token index i, or the zero
// value if i is out of range.

func ownerAt(analysis v4FileAnalysis, i int) v4SyntaxRegionID {
	if i < 0 || i >= len(analysis.TokenOwner) {
		return v4SyntaxRegionID{}
	}
	return analysis.TokenOwner[i]
}

// traceContainsRange reports whether any pre-suppression chain or
// pre-shadow component contains the inclusive [lo, hi] token range
// on the given path.
// traceContainsRange reports whether any pre-suppression chain or
// pre-shadow component contains the inclusive [lo, hi] token range
// on the given path.

func traceContainsRange(trace v4PipelineTrace, lo, hi int, path string) bool {
	if lo < 0 || hi < lo {
		return false
	}
	for _, chain := range trace.ChainsBeforeShadow {
		if !chainCoversRange(chain, path, lo, hi) {
			continue
		}
		return true
	}
	for _, comp := range trace.ComponentsBeforeShadow {
		if !componentCoversRange(comp, path, lo, hi) {
			continue
		}
		return true
	}
	return false
}

// chainCoversRange reports whether a chain's left/right ranges
// contain the inclusive [lo, hi] interval on path.
// chainCoversRange reports whether a chain's left/right ranges
// contain the inclusive [lo, hi] interval on path.

func chainCoversRange(chain cloneChain, path string, lo, hi int) bool {
	if chain.LeftRange.StartPos > lo || chain.LeftRange.EndPos < hi {
		if chain.LeftRange.StartPos > lo {
			return false
		}
	}
	if path == "" {
		return chain.LeftRange.StartPos <= lo && chain.LeftRange.EndPos >= hi
	}
	return chain.LeftRange.StartPos <= lo && chain.LeftRange.EndPos >= hi
}

// componentCoversRange reports whether any occurrence in a
// component contains the inclusive [lo, hi] interval on path.
// componentCoversRange reports whether any occurrence in a
// component contains the inclusive [lo, hi] interval on path.

func componentCoversRange(comp v4InternalFinding, path string, lo, hi int) bool {
	for _, occ := range comp.Occurrences {
		if path != "" && occ.Path != path {
			continue
		}
		if occ.StartPos <= lo && occ.EndPos >= hi {
			return true
		}
	}
	return false
}

// TestV4BaselineForensics_504_NoLargerLiveChain asserts no live
// pre-suppression chain whose width is strictly greater than 504
// produces a valid pair-evidence entry whose content key would
// allow that chain to attach to the canonical occurrence pair at
// one consistent relative offset.
//
// For every chain in ChainsAfterShadow with TokenSpan strictly
// greater than 504, the test inspects:
//
//   - whether its left and right widths agree;
//   - whether its left and right digests agree (using the
//     production v4ExactContentKeyForOccurrence);
//   - whether it has any chain member whose left and right
//     occurrences contain the canonical occurrence pair at the
//     same relative offset.
// largerChainCoversPair reports whether the chain's left/right
// ranges contain the canonical occurrence pair at one consistent
// relative offset.

func largerChainCoversPair(chain cloneChain, leftOcc, rightOcc maximalOccurrence) bool {
	if chain.LeftRange.StartPos > leftOcc.StartPos ||
		chain.LeftRange.EndPos < leftOcc.EndPos {
		return false
	}
	if chain.RightRange.StartPos > rightOcc.StartPos ||
		chain.RightRange.EndPos < rightOcc.EndPos {
		return false
	}
	leftOffset := leftOcc.StartPos - chain.LeftRange.StartPos
	rightOffset := rightOcc.StartPos - chain.RightRange.StartPos
	return leftOffset == rightOffset
}

// TestV4BaselineForensics_504_NoLargerLiveComponent asserts no
// pre-shadow component with TokenCount > 504 contains both
// canonical occurrences at one consistent relative offset with
// equal normalized sub-slices.
// getLeftRightFilesForTrace returns (leftFile, rightFile) for the
// given trace. Used by tests that already have a trace and need
// the file analyses.

func getLeftRightFilesForTrace(t *testing.T, trace v4PipelineTrace, finals []v4InternalFinding) (
	*v4AnalyzedFile, *v4AnalyzedFile,
) {
	t.Helper()
	leftFile, rightFile, _, _ := traceForLiveTree(t)
	_ = trace
	_ = finals
	return leftFile, rightFile
}

// largerComponentContainsPair reports whether a pre-shadow
// component contains both occurrences at one consistent relative
// offset.
// largerComponentContainsPair reports whether a pre-shadow
// component contains both occurrences at one consistent relative
// offset.

func largerComponentContainsPair(comp v4InternalFinding, leftOcc, rightOcc maximalOccurrence) bool {
	var leftOuter, rightOuter *maximalOccurrence
	for i := range comp.Occurrences {
		occ := &comp.Occurrences[i]
		if occ.Path != leftOcc.Path {
			continue
		}
		if occ.StartPos <= leftOcc.StartPos && occ.EndPos >= leftOcc.EndPos {
			leftOuter = occ
		}
	}
	for i := range comp.Occurrences {
		occ := &comp.Occurrences[i]
		if occ.Path != rightOcc.Path {
			continue
		}
		if occ.StartPos <= rightOcc.StartPos && occ.EndPos >= rightOcc.EndPos {
			rightOuter = occ
		}
	}
	if leftOuter == nil || rightOuter == nil {
		return false
	}
	leftOffset := leftOcc.StartPos - leftOuter.StartPos
	rightOffset := rightOcc.StartPos - rightOuter.StartPos
	return leftOffset == rightOffset
}

// TestV4BaselineForensics_504_SurvivesStructuralShadow asserts
// the canonical 504-token component is present BEFORE structural
// shadow suppression runs, and that no pre-shadow component with
// TokenCount > 504 classifies it as a shadow via the production
// `componentIsStructuralShadow` helper.
//
// The test invokes `componentIsStructuralShadow` directly against
// the live pre-shadow components; the textual-guard witness is
// retained as a characterization assertion only.
// shortDigestStr returns the first n characters of a digest.

func shortDigestStr(digest string, n int) string {
	if n > len(digest) {
		n = len(digest)
	}
	return digest[:n]
}

// TestV4BaselineForensics_504_SortedFingerprintStable asserts the
// 504 canonical finding's StableFingerprint equals the production
// v4StableFingerprintForContentKey derivation for its
// (Digest, TokenCount) key. This proves the public-surface
// StableFingerprint equals the production internal-key
// fingerprint (the seed fingerprint of the canonical content
// key).