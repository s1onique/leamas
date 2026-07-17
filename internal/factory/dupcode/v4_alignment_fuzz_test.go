// Package dupcode provides the persistent production/all-pairs fuzz
// differential for CORRECTION02-CORPUS-AND-EVIDENCE01.
package dupcode

import "testing"

func FuzzV4RegionPairingEquivalentToAllPairs(f *testing.F) {
	for _, seed := range v4AlignmentFuzzSeeds() {
		f.Add(seed.Value)
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		fixture := v4DecodeFuzzFixture(wire)
		production := v4RunProductionCorpusFixture(fixture)
		oracle := v4RunOracleCorpusFixture(fixture)
		v4AssertDifferentialResultsEqual(t, "fuzz-wire", production, oracle)
	})
}
