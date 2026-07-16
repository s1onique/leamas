// Package dupcode provides the CORRECTION04 larger-chain,
// larger-component, and structural-shadow survival tests for the
// surviving 504-token finding.
//
// The tests inspect every live pre-publication chain and every
// pre-shadow component that contains the canonical occurrence
// pair. They also invoke componentIsStructuralShadow and
// v4SuppressComponentShadows against the live pre-shadow
// components to prove the canonical 504-token component
// survives suppression.
package dupcode

import (
	"strings"
	"testing"
)

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

func TestV4BaselineForensics_504_NoLargerLiveChain(t *testing.T) {
	_, _, trace, finals := traceForLiveTree(t)
	canonical := canonicalLiveFinding(t, finals)
	leftOcc, rightOcc, ok := sortedLeftRight(canonical)
	if !ok {
		t.Fatal("canonical finding does not have 2 occurrences")
	}

	if len(trace.ChainsAfterShadow) == 0 {
		t.Fatal("ChainsAfterShadow is empty")
	}

	for i, chain := range trace.ChainsAfterShadow {
		// Every surviving chain has TokenSpan fields that capture
		// the chain's effective width on each side. A chain whose
		// token span exceeds 504 cannot directly absorb the 504-token
		// occurrence pair unless the chain's left and right ranges
		// contain the canonical occurrence pair at the same
		// relative offset AND the pair evidence hashes agree.
		if !largerChainCoversPair(chain, leftOcc, rightOcc) {
			continue
		}
		// Inspect pair evidence: a chain that covers both
		// occurrences must have produced a pair-evidence entry
		// whose content key matches the canonical key (so the
		// chain and the canonical are edges of the same connected
		// component).
		if i >= len(trace.PairEvidence) {
			t.Errorf("chain[%d] covers pair but has no pair evidence", i)
			continue
		}
		ev := trace.PairEvidence[i]
		if ev.ContentKey.TokenCount > 504 {
			t.Fatalf("chain[%d]: larger chain produced wider pair evidence "+
				"TokenCount=%d; current canonical is NOT maximal", i,
				ev.ContentKey.TokenCount)
		}
	}
}

// largerChainCoversPair reports whether the chain's left/right
// ranges contain the canonical occurrence pair at one consistent
// relative offset.
// TestV4BaselineForensics_504_NoLargerLiveComponent asserts no
// pre-shadow component with TokenCount > 504 contains both
// canonical occurrences at one consistent relative offset with
// equal normalized sub-slices.

func TestV4BaselineForensics_504_NoLargerLiveComponent(t *testing.T) {
	_, rightFile, trace, finals := traceForLiveTree(t)
	canonical := canonicalLiveFinding(t, finals)
	leftOcc, rightOcc, ok := sortedLeftRight(canonical)
	if !ok {
		t.Fatal("canonical finding does not have 2 occurrences")
	}

	for i, comp := range trace.ComponentsBeforeShadow {
		if comp.TokenCount <= 504 {
			continue
		}
		if !largerComponentContainsPair(comp, leftOcc, rightOcc) {
			continue
		}
		// A larger component contains both occurrences at one
		// consistent offset. If equal normalized sub-slices also
		// match, the larger component is a candidate owner of the
		// canonical finding; that would invalidate maximality.
		leftFile, _ := getLeftRightFilesForTrace(t, trace, finals)
		leftCand := leftFile.NormalizedTokens[leftOcc.StartPos : leftOcc.EndPos+1]
		rightCand := rightFile.NormalizedTokens[rightOcc.StartPos : rightOcc.EndPos+1]
		leftDigest := sha256Hex(strings.Join(leftCand, " "))
		rightDigest := sha256Hex(strings.Join(rightCand, " "))
		if leftDigest != rightDigest {
			continue
		}
		t.Fatalf("larger_component[%d] (TokenCount=%d) contains the "+
			"canonical pair with matching normalized content; "+
			"current canonical is NOT maximal", i, comp.TokenCount)
	}
}

// getLeftRightFilesForTrace returns (leftFile, rightFile) for the
// given trace. Used by tests that already have a trace and need
// the file analyses.
// TestV4BaselineForensics_504_SurvivesStructuralShadow asserts
// the canonical 504-token component is present BEFORE structural
// shadow suppression runs, and that no pre-shadow component with
// TokenCount > 504 classifies it as a shadow via the production
// `componentIsStructuralShadow` helper.
//
// The test invokes `componentIsStructuralShadow` directly against
// the live pre-shadow components; the textual-guard witness is
// retained as a characterization assertion only.

func TestV4BaselineForensics_504_SurvivesStructuralShadow(t *testing.T) {
	leftFile, rightFile, trace, finals := traceForLiveTree(t)
	canonical := canonicalLiveFinding(t, finals)
	leftOcc, rightOcc, ok := sortedLeftRight(canonical)
	if !ok {
		t.Fatal("canonical finding does not have 2 occurrences")
	}

	// (1) Canonical component present BEFORE shadow suppression.
	foundBefore := false
	for _, c := range trace.ComponentsBeforeShadow {
		if c.TokenCount == 504 && len(c.Occurrences) == 2 {
			foundBefore = true
			break
		}
	}
	if !foundBefore {
		t.Fatal("canonical 504-token component not in ComponentsBeforeShadow")
	}

	// (2) Canonical component still present AFTER shadow suppression.
	foundAfter := false
	for _, c := range trace.ComponentsAfterShadow {
		if c.TokenCount == 504 && len(c.Occurrences) == 2 {
			foundAfter = true
			break
		}
	}
	if !foundAfter {
		t.Fatal("canonical 504-token component not in ComponentsAfterShadow")
	}

	// (3) No pre-shadow component with TokenCount > 504 classifies
	// the canonical finding as a shadow.
	canonicalComp := v4InternalFinding{
		StableFingerprint: canonical.StableFingerprint,
		TokenCount:        canonical.TokenCount,
		Occurrences:       canonical.Occurrences,
	}
	files := map[string]*v4AnalyzedFile{
		leftFile.FileTokens.path:  leftFile,
		rightFile.FileTokens.path: rightFile,
	}
	for i, larger := range trace.ComponentsBeforeShadow {
		if larger.TokenCount <= 504 {
			continue
		}
		if !largerComponentContainsPair(larger, leftOcc, rightOcc) {
			continue
		}
		if componentIsStructuralShadow(canonicalComp, larger, files) {
			t.Fatalf("larger_component[%d] (TokenCount=%d) classifies "+
				"the canonical 504-token finding as a shadow; "+
				"current canonical is NOT maximal", i, larger.TokenCount)
		}
	}

	// (4) v4SuppressComponentShadows does not remove the canonical
	// component when run against the live pre-shadow set.
	suppressed := v4SuppressComponentShadows(trace.ComponentsBeforeShadow, files)
	for _, c := range suppressed {
		if c.TokenCount == 504 && len(c.Occurrences) == 2 {
			return // canonical survived
		}
	}
	t.Fatal("v4SuppressComponentShadows removed the canonical 504-token component")
}

// TestV4BaselineForensics_877_LockFacts asserts the concrete
// forensic facts quoted by the CORRECTION03/CORRECTION04 reports
// for the 877 historical public line range: owner count,
// unowned-token presence, mapped token count, mapped internal
// positions, and full independent SHA-256 digests.
//
// This test does NOT compare a computed label with a hard-coded
// label; it asserts the actual recorded counts and digests.
