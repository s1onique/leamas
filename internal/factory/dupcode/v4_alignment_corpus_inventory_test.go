// Package dupcode defines the canonical 17-dimension alignment corpus
// inventory for CORRECTION02-CORPUS-AND-EVIDENCE01.
package dupcode

const (
	v4AlignedN8                  v4CorpusDimension = "AlignedN8"
	v4AlignedN32                 v4CorpusDimension = "AlignedN32"
	v4AlignedN128                v4CorpusDimension = "AlignedN128"
	v4LeadingExtraLeft           v4CorpusDimension = "LeadingExtraLeft"
	v4LeadingExtraRight          v4CorpusDimension = "LeadingExtraRight"
	v4MiddleExtra                v4CorpusDimension = "MiddleExtra"
	v4TrailingExtra              v4CorpusDimension = "TrailingExtra"
	v4UnequalCardinality         v4CorpusDimension = "UnequalCardinality"
	v4NonUniformSpacing          v4CorpusDimension = "NonUniformSpacing"
	v4OffIndexMaximalChain       v4CorpusDimension = "OffIndexMaximalChain"
	v4TwoIndependentOffsetChains v4CorpusDimension = "TwoIndependentOffsetChains"
	v4ThreeRegionsAsymmetric     v4CorpusDimension = "ThreeRegionsAsymmetric"
	v4RepeatedWithinRegion       v4CorpusDimension = "RepeatedWithinRegion"
	v4ShuffledRawInput           v4CorpusDimension = "ShuffledRawInput"
	v4UnownedWindow              v4CorpusDimension = "UnownedWindow"
	v4DuplicateRawWindow         v4CorpusDimension = "DuplicateRawWindow"
	v4SamePathDifferentOrdinals  v4CorpusDimension = "SamePathDifferentOrdinals"
)

var requiredV4CorpusDimensions = []v4CorpusDimension{
	v4AlignedN8,
	v4AlignedN32,
	v4AlignedN128,
	v4LeadingExtraLeft,
	v4LeadingExtraRight,
	v4MiddleExtra,
	v4TrailingExtra,
	v4UnequalCardinality,
	v4NonUniformSpacing,
	v4OffIndexMaximalChain,
	v4TwoIndependentOffsetChains,
	v4ThreeRegionsAsymmetric,
	v4RepeatedWithinRegion,
	v4ShuffledRawInput,
	v4UnownedWindow,
	v4DuplicateRawWindow,
	v4SamePathDifferentOrdinals,
}

func v4CorpusRegion(path string, ordinal, start, end int) v4FixtureRegion {
	return v4FixtureRegion{
		Path: path, Ordinal: ordinal, StartPos: start, EndPos: end,
		StartLine: start + 1, EndLine: end + 1,
	}
}

func v4CorpusWindow(path string, start int) v4RawWindow {
	const width = 16
	return v4RawWindow{
		Path: path, StartPos: start, EndPos: start + width - 1,
		StartLine: start + 1, EndLine: start + width,
	}
}

func v4AlignedCorpusFixture(dimension v4CorpusDimension, n int) v4CorpusFixture {
	windows := make([]v4RawWindow, 0, 2*n)
	for i := 0; i < n; i++ {
		windows = append(windows, v4CorpusWindow("alpha.go", i))
	}
	for i := 0; i < n; i++ {
		windows = append(windows, v4CorpusWindow("beta.go", 400+i))
	}
	return v4CorpusFixture{
		Name: string(dimension), Dimension: dimension,
		Regions: []v4FixtureRegion{
			v4CorpusRegion("alpha.go", 0, 0, 255),
			v4CorpusRegion("beta.go", 0, 350, 700),
		},
		FileLength:  map[string]int{"alpha.go": 300, "beta.go": 750},
		RawWindows:  windows,
		LeftRegion:  v4SyntaxRegionID{Path: "alpha.go", Ordinal: 0},
		RightRegion: v4SyntaxRegionID{Path: "beta.go", Ordinal: 0},
	}
}

// v4BuildAlignmentCorpus returns exactly one primary fixture for every
// required semantic dimension. The order is the canonical inventory order.
func v4BuildAlignmentCorpus() []v4CorpusFixture {
	return []v4CorpusFixture{
		v4AlignedCorpusFixture(v4AlignedN8, 8),
		v4AlignedCorpusFixture(v4AlignedN32, 32),
		v4AlignedCorpusFixture(v4AlignedN128, 128),
		v4LeadingExtraLeftFixture(),
		v4LeadingExtraRightFixture(),
		v4MiddleExtraFixture(),
		v4TrailingExtraFixture(),
		v4UnequalCardinalityFixture(),
		v4NonUniformSpacingFixture(),
		v4OffIndexMaximalChainFixture(),
		v4TwoIndependentOffsetChainsFixture(),
		v4ThreeRegionsAsymmetricFixture(),
		v4RepeatedWithinRegionFixture(),
		v4ShuffledRawInputFixture(),
		v4UnownedWindowFixture(),
		v4DuplicateRawWindowFixture(),
		v4SamePathDifferentOrdinalsFixture(),
	}
}
