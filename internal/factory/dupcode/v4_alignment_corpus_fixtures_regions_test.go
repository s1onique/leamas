// Package dupcode provides the multi-region, ownership, multiplicity,
// shuffled, and same-path rows of the canonical alignment corpus.
package dupcode

func v4ThreeRegionsAsymmetricFixture() v4CorpusFixture {
	fx := v4TwoPathCorpusFixture(
		v4ThreeRegionsAsymmetric,
		[]int{0, 1, 2},
		[]int{100, 200, 201, 202},
	)
	fx.Regions = append(fx.Regions, v4CorpusRegion("gamma.go", 0, 0, 999))
	fx.FileLength["gamma.go"] = 1000
	fx.RawWindows = append(fx.RawWindows,
		v4CorpusWindow("gamma.go", 400),
		v4CorpusWindow("gamma.go", 401),
	)
	return fx
}

func v4RepeatedWithinRegionFixture() v4CorpusFixture {
	return v4TwoPathCorpusFixture(
		v4RepeatedWithinRegion,
		[]int{0, 100},
		[]int{300},
	)
}

func v4PermuteRawWindows(windows []v4RawWindow) []v4RawWindow {
	canonical := v4CanonicalRawWindows(windows)
	out := make([]v4RawWindow, 0, len(canonical))
	left, right := 0, len(canonical)-1
	for left <= right {
		out = append(out, canonical[right])
		right--
		if left <= right {
			out = append(out, canonical[left])
			left++
		}
	}
	return out
}

func v4ShuffledRawInputFixture() v4CorpusFixture {
	fx := v4TwoPathCorpusFixture(
		v4ShuffledRawInput,
		[]int{20, 21, 22},
		[]int{70, 320, 321, 322},
	)
	fx.RawWindows = v4PermuteRawWindows(fx.RawWindows)
	return fx
}

func v4UnownedWindowFixture() v4CorpusFixture {
	fx := v4CorpusFixture{
		Name: string(v4UnownedWindow), Dimension: v4UnownedWindow,
		Regions: []v4FixtureRegion{
			v4CorpusRegion("alpha.go", 0, 0, 100),
			v4CorpusRegion("beta.go", 0, 50, 300),
		},
		FileLength:  map[string]int{"alpha.go": 600, "beta.go": 350},
		LeftRegion:  v4SyntaxRegionID{Path: "alpha.go", Ordinal: 0},
		RightRegion: v4SyntaxRegionID{Path: "beta.go", Ordinal: 0},
	}
	for _, start := range []int{0, 1, 2} {
		fx.RawWindows = append(fx.RawWindows, v4CorpusWindow("alpha.go", start))
	}
	for _, start := range []int{100, 101, 102} {
		fx.RawWindows = append(fx.RawWindows, v4CorpusWindow("beta.go", start))
	}
	// Distinct ownership failures: no analysis for missing.go, and an
	// alpha.go interval outside every explicitly declared region.
	fx.RawWindows = append(fx.RawWindows,
		v4CorpusWindow("missing.go", 10),
		v4CorpusWindow("alpha.go", 500),
	)
	return fx
}

func v4DuplicateRawWindowFixture() v4CorpusFixture {
	fx := v4TwoPathCorpusFixture(
		v4DuplicateRawWindow,
		[]int{0, 1},
		[]int{100, 101},
	)
	duplicate := fx.RawWindows[0]
	fx.RawWindows = append([]v4RawWindow{duplicate}, fx.RawWindows...)
	return fx
}

func v4SamePathDifferentOrdinalsFixture() v4CorpusFixture {
	windows := []v4RawWindow{
		v4CorpusWindow("shared.go", 0),
		v4CorpusWindow("shared.go", 1),
		v4CorpusWindow("shared.go", 2),
		v4CorpusWindow("shared.go", 200),
		v4CorpusWindow("shared.go", 201),
		v4CorpusWindow("shared.go", 202),
	}
	return v4CorpusFixture{
		Name:      string(v4SamePathDifferentOrdinals),
		Dimension: v4SamePathDifferentOrdinals,
		Regions: []v4FixtureRegion{
			v4CorpusRegion("shared.go", 0, 0, 100),
			v4CorpusRegion("shared.go", 1, 180, 320),
		},
		FileLength:  map[string]int{"shared.go": 350},
		RawWindows:  windows,
		LeftRegion:  v4SyntaxRegionID{Path: "shared.go", Ordinal: 0},
		RightRegion: v4SyntaxRegionID{Path: "shared.go", Ordinal: 1},
	}
}
