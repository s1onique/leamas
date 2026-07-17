// Package dupcode provides the R2-R5 tests of the R1 cross-region
// regression proof for
// ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02.
//
// The tests in this file pin the asymmetric right-side-extra fixture
// from four independent angles:
//
//   - R2: preconditions (window counts, paths, region IDs, start
//     sequences) — TestV4Alignment_AsymmetricLeadingExtra_FixtureContract
//   - R3: alignment guard verdict —
//     TestV4Alignment_AsymmetricLeadingExtra_AlignmentGuardRejects
//   - R4: conservative candidate geometry —
//     TestV4Alignment_AsymmetricLeadingExtra_ConservativeCandidateGeometry
//   - R5: final canonical equality with the legacy all-pairs oracle
//     — TestV4Alignment_AsymmetricLeadingExtra_ProductionEqualsOracle
//
// Every test fails closed with a diagnostic that names the offending
// path, region, start position, or finding field. The mirrored
// left-side-extra case and the three-case differential table live in
// v4_alignment_cross_region_corpus_test.go.
package dupcode

import (
	"fmt"
	"strings"
	"testing"
)

// TestV4Alignment_AsymmetricLeadingExtra_FixtureContract pins the
// preconditions of the corrected asymmetric right-side-extra
// fixture BEFORE the production candidate generator runs. The
// assertions fail closed when:
//
//   - the left and right sides accidentally share a path;
//   - either side has the wrong number of windows;
//   - either side's start positions drift from the canonical shape;
//   - the production analyses collapse both paths into one region.
//
// This is the R2 contract: the asymmetric proof means nothing if
// the fixture itself is malformed.
func TestV4Alignment_AsymmetricLeadingExtra_FixtureContract(t *testing.T) {
	fx := asymmetricRightFixture()

	if len(fx.LeftWindows) != 3 {
		t.Fatalf("left window count = %d, want 3", len(fx.LeftWindows))
	}
	if len(fx.RightWindows) != 4 {
		t.Fatalf("right window count = %d, want 4", len(fx.RightWindows))
	}

	for i, w := range fx.LeftWindows {
		if w.Path != "alpha.go" {
			t.Fatalf("left window[%d] path = %q, want %q\n  left starts = %s",
				i, w.Path, "alpha.go", pathStarts(fx.LeftWindows))
		}
	}
	for j, w := range fx.RightWindows {
		if w.Path != "beta.go" {
			t.Fatalf("right window[%d] path = %q, want %q\n  right starts = %s",
				j, w.Path, "beta.go", pathStarts(fx.RightWindows))
		}
	}

	gotLeftStarts := make([]int, len(fx.LeftWindows))
	for i, w := range fx.LeftWindows {
		gotLeftStarts[i] = w.StartPos
	}
	if !intSliceEqual(gotLeftStarts, expectedAsymmetricRightStarts) {
		t.Fatalf("left starts = %v, want %v", gotLeftStarts, expectedAsymmetricRightStarts)
	}
	gotRightStarts := make([]int, len(fx.RightWindows))
	for j, w := range fx.RightWindows {
		gotRightStarts[j] = w.StartPos
	}
	if !intSliceEqual(gotRightStarts, expectedAsymmetricLeftStarts) {
		t.Fatalf("right starts = %v, want %v", gotRightStarts, expectedAsymmetricLeftStarts)
	}

	if _, ok := fx.PerPathLength["alpha.go"]; !ok {
		t.Fatalf("PerPathLength missing alpha.go")
	}
	if _, ok := fx.PerPathLength["beta.go"]; !ok {
		t.Fatalf("PerPathLength missing beta.go")
	}

	// Production regions must be distinct. If both paths collapse to
	// the same region the fixture has been silently corrupted.
	analyses := v4MakeAlignedAnalyses(fx.PerPathLength, nil)
	leftRid := pickRegion(t, analyses, fx.LeftWindows[0])
	rightRid := pickRegion(t, analyses, fx.RightWindows[0])
	if leftRid == rightRid {
		t.Fatalf("left region %s == right region %s (fixture collapsed to a single region)\n  left starts = %s\n  right starts = %s",
			leftRid, rightRid, pathStarts(fx.LeftWindows), pathStarts(fx.RightWindows))
	}
}

// TestV4Alignment_AsymmetricLeadingExtra_AlignmentGuardRejects
// proves that the production alignment guard returns false for the
// corrected asymmetric fixture. The diagnostic prints every value a
// reviewer needs to localise a future regression: the two region
// IDs, both start-position sequences, and the observed alignment
// result.
//
// This assertion runs INDEPENDENTLY of the final canonical
// comparison; the guard verdict is observable on its own.
func TestV4Alignment_AsymmetricLeadingExtra_AlignmentGuardRejects(t *testing.T) {
	fx := asymmetricRightFixture()
	annotated, idxA, idxB, leftRegion, rightRegion, _, _ :=
		asymmetricAnnotatedInputs(t, fx)

	if leftRegion.Path == rightRegion.Path {
		t.Fatalf("fixture collapsed to one region %s; guard verdict would be meaningless", leftRegion)
	}

	aligned := regionsArePositionallyAligned(idxA, idxB, annotated)
	leftStarts := make([]int, len(idxA))
	rightStarts := make([]int, len(idxB))
	for i, ix := range idxA {
		leftStarts[i] = annotated[ix].w.StartPos
	}
	for j, ix := range idxB {
		rightStarts[j] = annotated[ix].w.StartPos
	}

	if aligned {
		t.Fatalf("alignment guard returned true for the asymmetric fixture\n"+
			"  leftRegion   = %s\n"+
			"  rightRegion  = %s\n"+
			"  left starts  = %v\n"+
			"  right starts = %v\n"+
			"  observed     = true\n"+
			"  expected     = false",
			leftRegion, rightRegion, leftStarts, rightStarts)
	}
}

// TestV4Alignment_AsymmetricLeadingExtra_ConservativeCandidateGeometry
// pins the candidate set produced by the conservative cross-region
// all-pairs fallback for the corrected asymmetric fixture.
//
// The asserted matches are the three offset-100 links that the
// original unconditional diagonal missed:
//
//	left alpha.go@0  ↔ right beta.go@100
//	left alpha.go@1  ↔ right beta.go@101
//	left alpha.go@2  ↔ right beta.go@102
//
// Each match must carry offset = 100, the left region must be
// alpha.go, the right region must be beta.go, and all three matches
// must belong to a SINGLE constant-offset partition.
//
// The candidate set comes from the production
// v4BuildRegionBoundedChainInputs seam: the same seam the live
// detector calls. The test-only all-pairs oracle
// (v4GenerateAllPairsMatchesOracle) is exercised separately in R5.
func TestV4Alignment_AsymmetricLeadingExtra_ConservativeCandidateGeometry(t *testing.T) {
	fx := asymmetricRightFixture()
	_, _, _, leftRegion, rightRegion, combined, partitions :=
		asymmetricAnnotatedInputs(t, fx)

	want := []struct {
		leftStart  int
		rightStart int
	}{
		{0, 100},
		{1, 101},
		{2, 102},
	}

	type key struct {
		leftPath   string
		leftStart  int
		rightPath  string
		rightStart int
	}
	observed := make(map[key]v4RegionSeedMatch)
	for _, m := range combined {
		observed[key{
			leftPath:   m.Match.Left.Path,
			leftStart:  m.Match.Left.StartPos,
			rightPath:  m.Match.Right.Path,
			rightStart: m.Match.Right.StartPos,
		}] = m
	}
	for _, w := range want {
		k := key{leftPath: "alpha.go", leftStart: w.leftStart,
			rightPath: "beta.go", rightStart: w.rightStart}
		m, ok := observed[k]
		if !ok {
			t.Fatalf("candidate set missing required match alpha.go@%d ↔ beta.go@%d\n  combined = %d matches",
				w.leftStart, w.rightStart, len(combined))
		}
		if m.Match.Offset != 100 {
			t.Fatalf("alpha.go@%d ↔ beta.go@%d offset = %d, want 100",
				w.leftStart, w.rightStart, m.Match.Offset)
		}
		if m.LeftRegion.Path != leftRegion.Path || m.LeftRegion.Path != "alpha.go" {
			t.Fatalf("alpha.go@%d ↔ beta.go@%d LeftRegion = %s, want alpha.go",
				w.leftStart, w.rightStart, m.LeftRegion)
		}
		if m.RightRegion.Path != rightRegion.Path || m.RightRegion.Path != "beta.go" {
			t.Fatalf("alpha.go@%d ↔ beta.go@%d RightRegion = %s, want beta.go",
				w.leftStart, w.rightStart, m.RightRegion)
		}
	}

	// All three offset-100 matches must share a single
	// constant-offset partition. The chain-pair key is canonical so
	// the offset-100 partition is uniquely identified.
	var partitionKeys []v4ChainPairKey
	for k := range partitions {
		partitionKeys = append(partitionKeys, k)
	}
	var offsetPartitionKey v4ChainPairKey
	var offsetPartitionCount int
	var offsetPartitionMembers []v4RegionSeedMatch
	for _, k := range partitionKeys {
		if k.Offset == 100 {
			offsetPartitionKey = k
			offsetPartitionMembers = partitions[k]
			offsetPartitionCount = len(offsetPartitionMembers)
			break
		}
	}
	if offsetPartitionCount == 0 {
		t.Fatalf("no constant-offset partition with offset=100\n  partitions = %v", partitionKeys)
	}
	if len(offsetPartitionMembers) != len(want) {
		names := make([]string, 0, len(offsetPartitionMembers))
		for _, m := range offsetPartitionMembers {
			names = append(names, fmt.Sprintf("%s@%d↔%s@%d",
				m.Match.Left.Path, m.Match.Left.StartPos,
				m.Match.Right.Path, m.Match.Right.StartPos))
		}
		t.Fatalf("offset-100 partition has %d members, want %d\n  members = [%s]\n  partition key = %s",
			len(offsetPartitionMembers), len(want),
			strings.Join(names, ", "), offsetPartitionKey)
	}
	// Canonicalised membership check: every want tuple must appear
	// in the partition.
	for _, w := range want {
		found := false
		for _, m := range offsetPartitionMembers {
			if m.Match.Left.Path == "alpha.go" &&
				m.Match.Left.StartPos == w.leftStart &&
				m.Match.Right.Path == "beta.go" &&
				m.Match.Right.StartPos == w.rightStart {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("offset-100 partition missing alpha.go@%d ↔ beta.go@%d",
				w.leftStart, w.rightStart)
		}
	}
}

// TestV4Alignment_AsymmetricLeadingExtra_ProductionEqualsOracle
// pins the final canonical equality between the production pipeline
// and the legacy all-pairs oracle for the corrected asymmetric
// fixture.
//
// The comparison covers every canonical internal value the seam
// surfaces at this point: finding count, StableFingerprint,
// TokenCount, LineCount, occurrence count, occurrence paths,
// occurrence StartPos / EndPos / StartLine / EndLine, and the
// implicit ordering of findings and occurrences.
//
// The assertion is described as "structurally equal"; the test does
// not render the findings as text or JSON, it compares the live
// canonical structs field-by-field.
func TestV4Alignment_AsymmetricLeadingExtra_ProductionEqualsOracle(t *testing.T) {
	fx := asymmetricRightFixture()
	v4RunDifferentialCase(t, fx)
	result := v4RunProductionCorpusFixture(v4CorpusFixtureFromPerf(fx))
	if len(result.Findings) == 0 {
		t.Fatal("production returned zero findings; the asymmetric offset-100 chain must survive")
	}
}
