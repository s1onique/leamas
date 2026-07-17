// Package dupcode provides the aligned and asymmetric two-region rows
// of the canonical alignment corpus.
package dupcode

func v4TwoPathCorpusFixture(
	dimension v4CorpusDimension,
	leftStarts, rightStarts []int,
) v4CorpusFixture {
	windows := make([]v4RawWindow, 0, len(leftStarts)+len(rightStarts))
	for _, start := range leftStarts {
		windows = append(windows, v4CorpusWindow("alpha.go", start))
	}
	for _, start := range rightStarts {
		windows = append(windows, v4CorpusWindow("beta.go", start))
	}
	return v4CorpusFixture{
		Name: string(dimension), Dimension: dimension,
		Regions: []v4FixtureRegion{
			v4CorpusRegion("alpha.go", 0, 0, 999),
			v4CorpusRegion("beta.go", 0, 0, 999),
		},
		FileLength:  map[string]int{"alpha.go": 1000, "beta.go": 1000},
		RawWindows:  windows,
		LeftRegion:  v4SyntaxRegionID{Path: "alpha.go", Ordinal: 0},
		RightRegion: v4SyntaxRegionID{Path: "beta.go", Ordinal: 0},
	}
}

func v4LeadingExtraLeftFixture() v4CorpusFixture {
	return v4TwoPathCorpusFixture(
		v4LeadingExtraLeft,
		[]int{0, 50, 100, 101},
		[]int{200, 250, 251},
	)
}

func v4LeadingExtraRightFixture() v4CorpusFixture {
	return v4TwoPathCorpusFixture(
		v4LeadingExtraRight,
		[]int{0, 1, 2},
		[]int{50, 100, 101, 102},
	)
}

func v4MiddleExtraFixture() v4CorpusFixture {
	return v4TwoPathCorpusFixture(
		v4MiddleExtra,
		[]int{0, 50, 51},
		[]int{100, 101, 150, 151},
	)
}

func v4TrailingExtraFixture() v4CorpusFixture {
	return v4TwoPathCorpusFixture(
		v4TrailingExtra,
		[]int{0, 1, 2},
		[]int{100, 101, 102, 150},
	)
}

func v4UnequalCardinalityFixture() v4CorpusFixture {
	return v4TwoPathCorpusFixture(
		v4UnequalCardinality,
		[]int{0, 1, 2, 3, 4},
		[]int{100, 102, 103},
	)
}

func v4NonUniformSpacingFixture() v4CorpusFixture {
	return v4TwoPathCorpusFixture(
		v4NonUniformSpacing,
		[]int{0, 2, 9},
		[]int{100, 103, 111},
	)
}

func v4OffIndexMaximalChainFixture() v4CorpusFixture {
	return v4TwoPathCorpusFixture(
		v4OffIndexMaximalChain,
		[]int{10, 11, 12},
		[]int{60, 210, 211, 212},
	)
}

func v4TwoIndependentOffsetChainsFixture() v4CorpusFixture {
	return v4TwoPathCorpusFixture(
		v4TwoIndependentOffsetChains,
		[]int{0, 1, 100, 101},
		[]int{300, 301, 700, 701},
	)
}
