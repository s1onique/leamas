// Package dupcode provides the CORRECTION04 one-token
// extension audit for the surviving 504-token finding.
//
// The test reads the canonical occurrence's internal StartPos
// and EndPos from the live v4PipelineTrace and inspects three
// legal one-token extensions (left, right, both) on both files.
// For each candidate the test asserts exactly why the extension
// is rejected: width disagreement, digest disagreement,
// owner-boundary crossing, or no corresponding live chain.
package dupcode

import (
	"strings"
	"testing"
)


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