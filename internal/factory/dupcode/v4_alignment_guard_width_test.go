// Package dupcode pins the variable-width alignment precondition found
// by the persistent CORRECTION02 fuzz differential.
package dupcode

import "testing"

// v4VariableWidthMismatchVariant is a documented non-primary corpus
// variant. It does not satisfy inventory cardinality. Both sides have the
// same relative StartPos and EndPos sequences, but corresponding windows
// have different widths because the cross-side base start/end offsets
// differ. A diagonal is not semantically safe for this geometry.
func v4VariableWidthMismatchVariant() v4CorpusFixture {
	window := func(path string, start, end int) v4RawWindow {
		return v4RawWindow{
			Path: path, StartPos: start, EndPos: end,
			StartLine: start + 1, EndLine: end + 1,
		}
	}
	return v4CorpusFixture{
		Name:      "VariableWidthCrossBaseMismatchVariant",
		Dimension: "DocumentedVariant",
		Regions: []v4FixtureRegion{
			v4CorpusRegion("alpha.go", 0, 0, 999),
			v4CorpusRegion("shared.go", 0, 0, 999),
		},
		FileLength: map[string]int{"alpha.go": 1000, "shared.go": 1000},
		RawWindows: []v4RawWindow{
			window("alpha.go", 48, 304),
			window("alpha.go", 49, 304),
			window("shared.go", 49, 304),
			window("shared.go", 50, 304),
		},
		LeftRegion:  v4SyntaxRegionID{Path: "alpha.go", Ordinal: 0},
		RightRegion: v4SyntaxRegionID{Path: "shared.go", Ordinal: 0},
	}
}

func TestV4Alignment_VariableWidthCrossBaseMismatchRejectsDiagonal(t *testing.T) {
	fixture := v4VariableWidthMismatchVariant()
	analyses := v4BuildFixtureAnalyses(fixture)
	annotated, left, right := v4AnnotatedPairForFixture(t, fixture, analyses)
	if regionsArePositionallyAligned(left, right, annotated) {
		t.Fatal("alignment guard accepted cross-side base windows with different widths")
	}
}

func TestV4Alignment_VariableWidthCrossBaseMismatchProductionEqualsOracle(t *testing.T) {
	fixture := v4VariableWidthMismatchVariant()
	v4AssertDifferentialResultsEqual(
		t,
		fixture.Name,
		v4RunProductionCorpusFixture(fixture),
		v4RunOracleCorpusFixture(fixture),
	)
}
