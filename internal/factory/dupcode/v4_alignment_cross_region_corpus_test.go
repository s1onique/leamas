// Package dupcode provides the R6 three-case minimal differential
// table for the R1 cross-region regression proof owned by
// ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02.
//
// The corpus pins the three shapes the R1 ACT is responsible for
// proving:
//
//  1. AlignedDistinctRegions      — guard true, diagonal valid
//  2. AsymmetricLeadingExtraRight — guard false, offset-100 chain
//  3. AsymmetricLeadingExtraLeft  — mirror of case 2
//
// Every case asserts its intended guard verdict before comparing
// production with the oracle so the failure diagnostic localises a
// regression to one row.
//
// The full adversarial corpus, committed fuzz regression, 30-second
// fuzz run, benchmark confirmation, and whitespace cleanup belong
// to the successor ACT (CORRECTION02-CORPUS-AND-EVIDENCE01); this
// file owns ONLY the three-case minimal differential table.
package dupcode

import (
	"testing"
)

// TestV4Alignment_MinimalCrossRegionCorpus is the three-case R6
// differential table. It is intentionally NOT the full CORRECTION02
// corpus; it pins the three shapes the R1 ACT is responsible for
// proving:
//
//  1. AlignedDistinctRegions      — guard true, diagonal valid
//  2. AsymmetricLeadingExtraRight — guard false, offset-100 chain
//  3. AsymmetricLeadingExtraLeft  — mirror of case 2
//
// Every case asserts its intended guard verdict before comparing
// production with the oracle so the failure diagnostic localises a
// regression to one row.
func TestV4Alignment_MinimalCrossRegionCorpus(t *testing.T) {
	cases := []crossRegionCorpusCase{
		{
			Name:          "AlignedDistinctRegions",
			Fixture:       alignedDistinctFixture(),
			WantGuardOK:   true,
			WantMinChain:  3,
			WantOffset:    100,
			WantLeftPath:  "alpha.go",
			WantRightPath: "beta.go",
		},
		{
			Name:          "AsymmetricLeadingExtraRight",
			Fixture:       asymmetricRightFixture(),
			WantGuardOK:   false,
			WantMinChain:  3,
			WantOffset:    100,
			WantLeftPath:  "alpha.go",
			WantRightPath: "beta.go",
		},
		{
			Name:          "AsymmetricLeadingExtraLeft",
			Fixture:       asymmetricLeftFixture(),
			WantGuardOK:   false,
			WantMinChain:  3,
			WantOffset:    -100,
			WantLeftPath:  "alpha.go",
			WantRightPath: "beta.go",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			fx := tc.Fixture

			annotated, idxA, idxB, _, _, _, partitions :=
				asymmetricAnnotatedInputs(t, fx)
			aligned := regionsArePositionallyAligned(idxA, idxB, annotated)
			if aligned != tc.WantGuardOK {
				t.Fatalf("%s: guard verdict drift got=%v want=%v",
					tc.Name, aligned, tc.WantGuardOK)
			}

			// The off-index maximal chain must survive in the
			// appropriate constant-offset partition. The chain's
			// canonical offset is what we assert.
			matched := false
			for key, members := range partitions {
				if key.Offset != tc.WantOffset {
					continue
				}
				if len(members) < tc.WantMinChain {
					continue
				}
				for _, m := range members {
					if m.Match.Left.Path == tc.WantLeftPath &&
						m.Match.Right.Path == tc.WantRightPath {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				t.Fatalf("%s: constant-offset partition (offset=%d) did not survive\n"+
					"  partitions = %v",
					tc.Name, tc.WantOffset, summarizePartitions(partitions))
			}

			// Final canonical equivalence uses the one authoritative
			// complete structural comparator shared by corpus and fuzz.
			v4RunDifferentialCase(t, fx)
		})
	}
}
