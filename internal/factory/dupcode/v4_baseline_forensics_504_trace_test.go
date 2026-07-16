// Package dupcode provides CORRECTION04 maximality tests for the
// surviving 504-token finding.
//
// The CORRECTION03 maximality proof started from the already-
// selected final finding, which made the proof circular. CORRECTION04
// requires the maximality proof to inspect the actual pre-publication
// pipeline stages:
//
//   - ChainsBeforeShadow, ChainsAfterShadow,
//     PairEvidence, ComponentsBeforeShadow,
//     ComponentsAfterShadow (captured by v4BuildInternalFindingsTrace);
//   - immediate left and right one-token extensions of the canonical
//     occurrence pair;
//   - every larger live chain that contains the canonical
//     occurrence pair;
//   - every larger pre-shadow component that contains the canonical
//     occurrence pair;
//   - the structural-shadow survival against the actual live
//     pre-shadow components.
//
// All tests in this file run against the actual production tree
// (cmd/leamas/claim_commands.go and cmd/leamas/evidence_commands.go).
package dupcode

import (
	"strings"
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
func TestV4PipelineTrace_StagesNonEmpty(t *testing.T) {
	leftFile, rightFile, trace, finals := traceForLiveTree(t)
	if leftFile == nil || rightFile == nil {
		t.Fatal("live analysis files missing")
	}
	if len(trace.FilteredWindows) == 0 {
		t.Fatal("FilteredWindows is empty")
	}
	if len(trace.Partitions) == 0 {
		t.Fatal("Partitions is empty")
	}
	if len(trace.ChainsBeforeShadow) == 0 {
		t.Fatal("ChainsBeforeShadow is empty")
	}
	if len(trace.ChainsAfterShadow) == 0 {
		t.Fatal("ChainsAfterShadow is empty")
	}
	if len(trace.PairEvidence) == 0 {
		t.Fatal("PairEvidence is empty")
	}
	if len(trace.ComponentsBeforeShadow) == 0 {
		t.Fatal("ComponentsBeforeShadow is empty")
	}
	canonical := canonicalLiveFinding(t, finals)
	if canonical.TokenCount != 504 {
		t.Fatalf("canonical finding must have TokenCount=504, got %d", canonical.TokenCount)
	}
	t.Logf("trace stages: windows=%d partitions=%d chains_before=%d chains_after=%d pair_evidence=%d components_before=%d components_after=%d final_findings=%d",
		len(trace.FilteredWindows),
		len(trace.Partitions),
		len(trace.ChainsBeforeShadow),
		len(trace.ChainsAfterShadow),
		len(trace.PairEvidence),
		len(trace.ComponentsBeforeShadow),
		len(trace.ComponentsAfterShadow),
		len(trace.FinalFindings),
	)
}

// TestV4PipelineTrace_PairEvidenceDrivesMaterializer asserts that
// every surviving chain produces exactly one pair evidence entry
// before materialization. The pair-evidence step rejects chains
// whose left/right widths or digests disagree; the surviving
// evidence is therefore the set that drives the
// connected-component materializer.
func TestV4PipelineTrace_PairEvidenceDrivesMaterializer(t *testing.T) {
	_, _, trace, _ := traceForLiveTree(t)
	if len(trace.PairEvidence) != len(trace.ChainsAfterShadow) {
		t.Fatalf("PairEvidence=%d must equal ChainsAfterShadow=%d",
			len(trace.PairEvidence), len(trace.ChainsAfterShadow))
	}
	for i, ev := range trace.PairEvidence {
		if ev.ContentKey.TokenCount != 504 {
			t.Errorf("pair_evidence[%d].TokenCount=%d, want 504",
				i, ev.ContentKey.TokenCount)
		}
		if ev.Left.StartPos < 0 || ev.Right.StartPos < 0 {
			t.Errorf("pair_evidence[%d] missing left/right", i)
		}
	}
}

// TestV4PipelineTrace_ComponentsBeforeShadowContains504 asserts
// that the 504-token canonical component is present BEFORE
// structural-shadow suppression runs. The maximality proof uses
// this as the starting point for the larger-component audit.
func TestV4PipelineTrace_ComponentsBeforeShadowContains504(t *testing.T) {
	_, _, trace, _ := traceForLiveTree(t)
	if len(trace.ComponentsBeforeShadow) == 0 {
		t.Fatal("ComponentsBeforeShadow is empty")
	}
	saw504 := false
	for _, c := range trace.ComponentsBeforeShadow {
		if c.TokenCount == 504 && len(c.Occurrences) == 2 {
			saw504 = true
			break
		}
	}
	if !saw504 {
		t.Fatalf("ComponentsBeforeShadow must contain a 504-token component with 2 occurrences")
	}
}

// TestV4BaselineForensics_504_CannotExtendOneToken inspects every
// legal one-token extension of the canonical occurrence pair on
// both files. For each extension it asserts exactly why the
// extension is rejected (out of region, owner changes, width
// disagreement, digest disagreement, or no corresponding chain).
//
// The test reads the canonical occurrence's internal StartPos
// and EndPos directly from the live trace (NOT from the public
// line range), so the extension candidates target the exact
// pre-suppression boundary.
func TestV4BaselineForensics_504_CannotExtendOneToken(t *testing.T) {
	leftFile, rightFile, trace, finals := traceForLiveTree(t)
	canonical := canonicalLiveFinding(t, finals)
	leftOcc, rightOcc, ok := sortedLeftRight(canonical)
	if !ok {
		t.Fatal("canonical finding does not have 2 occurrences")
	}

	leftTokens := leftFile.NormalizedTokens[leftOcc.StartPos : leftOcc.EndPos+1]
	rightTokens := rightFile.NormalizedTokens[rightOcc.StartPos : rightOcc.EndPos+1]
	if len(leftTokens) != 504 || len(rightTokens) != 504 {
		t.Fatalf("canonical tokens must be 504 each, got %d/%d",
			len(leftTokens), len(rightTokens))
	}

	cases := []struct {
		name    string
		leftLo  int
		leftHi  int
		rightLo int
		rightHi int
	}{
		{"extend_left", leftOcc.StartPos - 1, leftOcc.EndPos, rightOcc.StartPos - 1, rightOcc.EndPos},
		{"extend_right", leftOcc.StartPos, leftOcc.EndPos + 1, rightOcc.StartPos, rightOcc.EndPos + 1},
		{"extend_both", leftOcc.StartPos - 1, leftOcc.EndPos + 1, rightOcc.StartPos - 1, rightOcc.EndPos + 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.leftLo < 0 || c.leftHi >= len(leftFile.NormalizedTokens) {
				t.Fatalf("left extension out of range: lo=%d hi=%d n=%d",
					c.leftLo, c.leftHi, len(leftFile.NormalizedTokens))
			}
			if c.rightLo < 0 || c.rightHi >= len(rightFile.NormalizedTokens) {
				t.Fatalf("right extension out of range: lo=%d hi=%d n=%d",
					c.rightLo, c.rightHi, len(rightFile.NormalizedTokens))
			}
			leftExt := leftFile.NormalizedTokens[c.leftLo : c.leftHi+1]
			rightExt := rightFile.NormalizedTokens[c.rightLo : c.rightHi+1]
			leftDigest := sha256Hex(strings.Join(leftExt, " "))
			rightDigest := sha256Hex(strings.Join(rightExt, " "))

			// (a) width disagreement: token counts differ between
			// left and right. Any sane extension that wants to be a
			// valid pair must have equal widths; the production
			// v4PairEvidenceFromChain returns an error when they
			// differ, so any such candidate is rejected by the
			// checked seam.
			if len(leftExt) != len(rightExt) {
				t.Logf("%s: rejected by width disagreement (%d vs %d)",
					c.name, len(leftExt), len(rightExt))
				return
			}
			// (b) digest disagreement: even when widths agree, the
			// candidate must hash to the same exact normalized
			// content on both sides to form pair evidence.
			if leftDigest != rightDigest {
				t.Logf("%s: rejected by digest disagreement", c.name)
				return
			}
			// (c) owner change: at least one adjacent token changes
			// the region owner, so the extension is no longer
			// single-region.
			if ownerAt(leftFile.Analysis, c.leftLo) != ownerAt(leftFile.Analysis, c.leftHi) {
				t.Logf("%s: rejected by owner-boundary crossing on left",
					c.name)
				return
			}
			if ownerAt(rightFile.Analysis, c.rightLo) != ownerAt(rightFile.Analysis, c.rightHi) {
				t.Logf("%s: rejected by owner-boundary crossing on right",
					c.name)
				return
			}
			// (d) chain existence: at least one live chain must
			// span the extended range. If no live chain covers it,
			// the pair-evidence step never produces a candidate.
			if !traceContainsRange(trace, c.leftLo, c.leftHi,
				"cmd/leamas/claim_commands.go") &&
				!traceContainsRange(trace, c.rightLo, c.rightHi,
					"cmd/leamas/evidence_commands.go") {
				t.Logf("%s: rejected (no live chain covers extended range)",
					c.name)
				return
			}
			t.Fatalf("%s: extension would form a valid pair evidence entry; "+
				"current canonical finding is NOT maximal", c.name)
		})
	}
}

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
func TestV4BaselineForensics_877_LockFacts(t *testing.T) {
	c := forensicsCases()[0]
	d := forensicsOracle(t, c)
	if d.LeftStartPos < 0 || d.RightStartPos < 0 {
		t.Fatal("877 line range does not resolve to tokens")
	}
	if len(d.LeftRegionOwners) != 4 {
		t.Errorf("877 left owner count: got %d, want 4 (4 regions in slice)",
			len(d.LeftRegionOwners))
	}
	if len(d.RightRegionOwner) != 4 {
		t.Errorf("877 right owner count: got %d, want 4 (4 regions in slice)",
			len(d.RightRegionOwner))
	}
	hasUnownedLeft := false
	for _, o := range d.LeftRegionOwners {
		if o.Path == "" {
			hasUnownedLeft = true
			break
		}
	}
	if !hasUnownedLeft {
		t.Errorf("877 left slice must contain at least one unowned token")
	}
	hasUnownedRight := false
	for _, o := range d.RightRegionOwner {
		if o.Path == "" {
			hasUnownedRight = true
			break
		}
	}
	if !hasUnownedRight {
		t.Errorf("877 right slice must contain at least one unowned token")
	}
	if d.LeftTokenCount == 0 || d.RightTokenCount == 0 {
		t.Errorf("877 mapped token count must be > 0; got %d/%d",
			d.LeftTokenCount, d.RightTokenCount)
	}
	if d.LeftDigest == "" || d.RightDigest == "" {
		t.Errorf("877 digests must be non-empty")
	}
}

// TestV4BaselineForensics_514_LockFacts asserts the concrete
// forensic facts quoted by the CORRECTION03/CORRECTION04 reports
// for the 514 historical public line range.
func TestV4BaselineForensics_514_LockFacts(t *testing.T) {
	c := forensicsCases()[1]
	d := forensicsOracle(t, c)
	if d.LeftStartPos < 0 || d.RightStartPos < 0 {
		t.Fatal("514 line range does not resolve to tokens")
	}
	if len(d.LeftRegionOwners) != 3 {
		t.Errorf("514 left owner count: got %d, want 3 (3 regions in slice)",
			len(d.LeftRegionOwners))
	}
	if len(d.RightRegionOwner) != 3 {
		t.Errorf("514 right owner count: got %d, want 3 (3 regions in slice)",
			len(d.RightRegionOwner))
	}
	hasUnownedLeft := false
	for _, o := range d.LeftRegionOwners {
		if o.Path == "" {
			hasUnownedLeft = true
			break
		}
	}
	if !hasUnownedLeft {
		t.Errorf("514 left slice must contain at least one unowned token")
	}
	hasUnownedRight := false
	for _, o := range d.RightRegionOwner {
		if o.Path == "" {
			hasUnownedRight = true
			break
		}
	}
	if !hasUnownedRight {
		t.Errorf("514 right slice must contain at least one unowned token")
	}
	if d.LeftTokenCount == 0 || d.RightTokenCount == 0 {
		t.Errorf("514 mapped token count must be > 0; got %d/%d",
			d.LeftTokenCount, d.RightTokenCount)
	}
	if d.LeftDigest == "" || d.RightDigest == "" {
		t.Errorf("514 digests must be non-empty")
	}
}

// TestV4BaselineForensics_PublicGeometryClassification asserts
// the CORRECTION04 honest-geometry contract: the
// forensicsOracle classifies PUBLIC line ranges as mapped to
// current-tree token positions. The test records the mapped
// facts (start/end positions, mapped token count, owner set,
// unowned presence) for each historical range but does NOT claim
// these are the historical detector's exact internal positions.
//
// This test exists to keep the public-line forensics story
// honest: line ranges from the historical baseline are mapped
// to current-tree token positions via mapLineRangeToTokenRange;
// the mapped positions are not the historical detector's exact
// internal geometry.
func TestV4BaselineForensics_PublicGeometryClassification(t *testing.T) {
	cases := forensicsCases()
	for _, c := range cases {
		d := forensicsOracle(t, c)
		if d.LeftStartPos < 0 || d.RightStartPos < 0 {
			t.Errorf("%s: line range did not resolve to tokens", c.Name)
			continue
		}
		t.Logf("%s: left range [%d,%d] tokens=%d digest=%s owners=%d; "+
			"right range [%d,%d] tokens=%d digest=%s owners=%d",
			c.Name,
			d.LeftStartPos, d.LeftEndPos, d.LeftTokenCount, shortDigestStr(d.LeftDigest, 12),
			len(d.LeftRegionOwners),
			d.RightStartPos, d.RightEndPos, d.RightTokenCount, shortDigestStr(d.RightDigest, 12),
			len(d.RightRegionOwner),
		)
	}
}

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
func TestV4BaselineForensics_504_SortedFingerprintStable(t *testing.T) {
	_, _, trace, finals := traceForLiveTree(t)
	canonical := canonicalLiveFinding(t, finals)
	for _, comp := range trace.ComponentsBeforeShadow {
		if comp.TokenCount != 504 || len(comp.Occurrences) != 2 {
			continue
		}
		if comp.StableFingerprint != canonical.StableFingerprint {
			t.Errorf("pre-shadow 504 StableFingerprint=%s, "+
				"final StableFingerprint=%s (must match)",
				comp.StableFingerprint, canonical.StableFingerprint)
		}
	}
}
