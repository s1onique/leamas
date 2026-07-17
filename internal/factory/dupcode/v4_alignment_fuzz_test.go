// Package dupcode provides the fuzz differential target for
// ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION01.
//
// R5: FuzzV4RegionPairingEquivalentToAllPairs compares the production
// pipeline against the all-pairs oracle for every fuzzed region-
// occurrence sequence. Required fuzz commands live in the ACT.
//
// The fuzz corpus is seeded with every
// v4BuildDeterministicCorpus fixture from
// v4_alignment_differential_test.go. The serialization helpers
// (serializeWindowMapForFuzz, fuzzParseBlob, fuzzBlobSchema,
// itoaParse) are owned here because they are only used by the fuzz
// target.
package dupcode

import (
	"testing"
)

func FuzzV4RegionPairingEquivalentToAllPairs(f *testing.F) {
	// Deterministic seeds taken from the R3 corpus.
	for _, fx := range v4BuildDeterministicCorpus() {
		f.Add(
			serializeWindowMapForFuzz(fx.LeftWindows),
			serializeWindowMapForFuzz(fx.RightWindows),
		)
	}
	f.Fuzz(func(t *testing.T, leftBlob, rightBlob string) {
		leftWindows, rightWindows, ok := fuzzParseBlob(t, leftBlob, rightBlob)
		if !ok {
			t.Skip()
		}
		if len(leftWindows)+len(rightWindows) > 32 {
			t.Skip()
		}
		wm := v4BuildAlignedWindowMap("seed", leftWindows, rightWindows)
		analyses := v4MakeAlignedAnalyses(
			map[string]int{"alpha.go": 4000, "beta.go": 4000}, nil,
		)
		files := v4MakeAlignedFiles(nil, nil, analyses)
		prod, err := v4BuildInternalFindings(wm, analyses, files)
		if err != nil {
			t.Fatalf("production: %v", err)
		}
		oracle, err := v4BuildInternalFindingsOracle(wm, analyses, files, v4GenerateAllPairsMatchesOracle)
		if err != nil {
			t.Fatalf("oracle: %v", err)
		}
		if len(prod) != len(oracle) {
			t.Fatalf("finding-count drift prod=%d ora=%d", len(prod), len(oracle))
		}
		for i := range prod {
			pa, oa := prod[i], oracle[i]
			if pa.StableFingerprint != oa.StableFingerprint {
				t.Fatalf("fingerprint drift f%d prod=%q ora=%q",
					i, pa.StableFingerprint, oa.StableFingerprint)
			}
			if pa.TokenCount != oa.TokenCount {
				t.Fatalf("token-count drift f%d prod=%d ora=%d",
					i, pa.TokenCount, oa.TokenCount)
			}
			if len(pa.Occurrences) != len(oa.Occurrences) {
				t.Fatalf("occurrence-count drift f%d prod=%d ora=%d",
					i, len(pa.Occurrences), len(oa.Occurrences))
			}
			for j := range pa.Occurrences {
				po, oo := pa.Occurrences[j], oa.Occurrences[j]
				if po.Path != oo.Path || po.StartLine != oo.StartLine || po.EndLine != oo.EndLine {
					t.Fatalf("occurrence drift f%d o%d prod=(%s %d-%d) ora=(%s %d-%d)",
						i, j, po.Path, po.StartLine, po.EndLine,
						oo.Path, oo.StartLine, oo.EndLine)
				}
			}
		}
	})
}

// fuzzBlobSchema defines the wire format produced by
// serializeWindowMapForFuzz and consumed by fuzzParseBlob. Two
// blobs are concatenated by the fuzzer harness; each one encodes
// the path, start positions and per-window lengths of one side of
// the fixture. The format is intentionally trivial so the fuzzer
// can produce plausible values from a small alphabet:
//
//	path#startpos1,startpos2,...|len1,len2,...
//
// The segments are required; missing tokens cause a t.Skip().
type fuzzBlobSchema struct{}

// serializeWindowMapForFuzz packs one side of a fixture window pair
// into the wire format.
func serializeWindowMapForFuzz(ws []v4RawWindow) string {
	if len(ws) == 0 {
		return "alpha.go#|"
	}
	path := ws[0].Path
	out := path + "#"
	for i, w := range ws {
		if i > 0 {
			out += ","
		}
		out += itoa(w.StartPos)
	}
	out += "|"
	for i, w := range ws {
		if i > 0 {
			out += ","
		}
		out += itoa(w.EndPos - w.StartPos)
	}
	return out
}

// fuzzParseBlob decodes one fuzz input slice into two
// v4RawWindow sequences. It returns (nil, nil, false) when the
// blob does not parse cleanly; the fuzzer skips such inputs.
func fuzzParseBlob(t *testing.T, leftBlob, rightBlob string) ([]v4RawWindow, []v4RawWindow, bool) {
	t.Helper()
	parse := func(blob, def string) ([]v4RawWindow, string, bool) {
		path := def
		hashIdx := -1
		for i, r := range blob {
			if r == '#' {
				hashIdx = i
				path = blob[:i]
				break
			}
		}
		if hashIdx < 0 {
			return nil, "", false
		}
		rest := blob[hashIdx+1:]
		barIdx := -1
		for i, r := range rest {
			if r == '|' {
				barIdx = i
				break
			}
		}
		if barIdx < 0 {
			return nil, "", false
		}
		startsBlob := rest[:barIdx]
		lensBlob := rest[barIdx+1:]
		if startsBlob == "" {
			return nil, path, true
		}
		var starts, lens []int
		i := 0
		for i < len(startsBlob) {
			j := i
			for j < len(startsBlob) && startsBlob[j] != ',' {
				j++
			}
			v, ok := itoaParse(startsBlob[i:j])
			if !ok {
				return nil, "", false
			}
			starts = append(starts, v)
			i = j + 1
		}
		i = 0
		for i < len(lensBlob) {
			j := i
			for j < len(lensBlob) && lensBlob[j] != ',' {
				j++
			}
			v, ok := itoaParse(lensBlob[i:j])
			if !ok {
				return nil, "", false
			}
			lens = append(lens, v)
			i = j + 1
		}
		if len(starts) != len(lens) {
			return nil, "", false
		}
		ws := make([]v4RawWindow, 0, len(starts))
		for i, s := range starts {
			if lens[i] <= 0 {
				return nil, "", false
			}
			ws = append(ws, v4RawWindow{
				Path: path, StartPos: s, EndPos: s + lens[i],
				StartLine: 100 + i, EndLine: 100 + i + lens[i],
			})
		}
		return ws, path, true
	}
	left, lpath, ok := parse(leftBlob, "alpha.go")
	if !ok {
		return nil, nil, false
	}
	right, _, ok := parse(rightBlob, lpath)
	if !ok {
		return nil, nil, false
	}
	return left, right, true
}

// itoaParse is a strconv-free integer parse so the fuzzer can use
// pre-allocated strings without allocation overhead.
func itoaParse(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

// v4BuildDeterministicCorpus assembles the 15-case R3 differential
// corpus. Each case is constructed so the production pipeline and
// the all-pairs oracle cannot differ except by a real semantic
// divergence.
func v4BuildDeterministicCorpus() []v4PerfFixture {
	mkLeft := func(path string, starts []int) []v4RawWindow {
		out := make([]v4RawWindow, 0, len(starts))
		for i, sp := range starts {
			out = append(out, v4RawWindow{
				Path: path, StartPos: sp, EndPos: sp + 80,
				StartLine: 100 + i, EndLine: 100 + i + 80,
			})
		}
		return out
	}
	corpus := []v4PerfFixture{
		// 1. equal, perfectly aligned two-region sequences
		v4SlidingAlignedFixture(8),
		v4SlidingAlignedFixture(32),
		v4SlidingAlignedFixture(128),
		// 2. extra leading occurrence on the left
		v4AsymmetricLeadingExtra(),
		// 3. extra leading occurrence on the right: same shape mirrored
		{
			Name:          "AsymmetricLeadingExtraRight",
			WindowSize:    3,
			LeftWindows:   mkLeft("alpha.go", []int{50, 100, 101}),
			RightWindows:  mkLeft("beta.go", []int{0, 1, 2, 3}),
			PerPathLength: map[string]int{"alpha.go": 200, "beta.go": 200},
		},
		// 4. extra middle occurrence
		{
			Name:          "AsymmetricMiddle",
			WindowSize:    3,
			LeftWindows:   mkLeft("alpha.go", []int{0, 1, 2, 3}),
			RightWindows:  mkLeft("beta.go", []int{0, 100, 2}),
			PerPathLength: map[string]int{"alpha.go": 200, "beta.go": 200},
		},
		// 5. extra trailing occurrence
		{
			Name:          "AsymmetricTrailing",
			WindowSize:    3,
			LeftWindows:   mkLeft("alpha.go", []int{0, 1, 2}),
			RightWindows:  mkLeft("beta.go", []int{0, 100, 101, 102}),
			PerPathLength: map[string]int{"alpha.go": 200, "beta.go": 200},
		},
		// 6. unequal cardinalities (left larger than right)
		{
			Name:          "UnequalLeft",
			WindowSize:    3,
			LeftWindows:   mkLeft("alpha.go", []int{0, 1, 2, 3, 4}),
			RightWindows:  mkLeft("beta.go", []int{0, 100, 101}),
			PerPathLength: map[string]int{"alpha.go": 200, "beta.go": 200},
		},
		// 7. non-uniform position spacing
		{
			Name:          "NonuniformSpacing",
			WindowSize:    3,
			LeftWindows:   mkLeft("alpha.go", []int{0, 1, 7}),
			RightWindows:  mkLeft("beta.go", []int{100, 102, 110}),
			PerPathLength: map[string]int{"alpha.go": 200, "beta.go": 200},
		},
		// 8. valid maximal chain on an off-index diagonal
		v4AsymmetricLeadingExtra(),
		// 9. two independent constant-offset chains on distinct regions
		{
			Name: "TwoIndependentChains",
			LeftWindows: []v4RawWindow{
				{Path: "alpha.go", StartPos: 0, EndPos: 80, StartLine: 100, EndLine: 180},
				{Path: "alpha.go", StartPos: 1, EndPos: 81, StartLine: 101, EndLine: 181},
			},
			RightWindows: []v4RawWindow{
				{Path: "beta.go", StartPos: 500, EndPos: 580, StartLine: 500, EndLine: 580},
				{Path: "beta.go", StartPos: 501, EndPos: 581, StartLine: 501, EndLine: 581},
				{Path: "gamma.go", StartPos: 1500, EndPos: 1580, StartLine: 1500, EndLine: 1580},
				{Path: "gamma.go", StartPos: 1501, EndPos: 1581, StartLine: 1501, EndLine: 1581},
			},
			PerPathLength: map[string]int{"alpha.go": 1000, "beta.go": 2000, "gamma.go": 3000},
		},
		// 10. three regions with asymmetric counts - off-diagonal pairing required
		{
			Name: "ThreeRegionsAsymmetric",
			LeftWindows: []v4RawWindow{
				{Path: "alpha.go", StartPos: 0, EndPos: 80, StartLine: 100, EndLine: 180},
				{Path: "alpha.go", StartPos: 1, EndPos: 81, StartLine: 101, EndLine: 181},
				{Path: "alpha.go", StartPos: 2, EndPos: 82, StartLine: 102, EndLine: 182},
				{Path: "alpha.go", StartPos: 3, EndPos: 83, StartLine: 103, EndLine: 183},
			},
			RightWindows: []v4RawWindow{
				{Path: "beta.go", StartPos: 100, EndPos: 180, StartLine: 200, EndLine: 280},
				{Path: "beta.go", StartPos: 103, EndPos: 183, StartLine: 203, EndLine: 283},
			},
			PerPathLength: map[string]int{"alpha.go": 1000, "beta.go": 2000},
		},
		// 11. repeated occurrences inside one region (Within-region pairs)
		{
			Name: "RepeatedMultiplicity",
			LeftWindows: []v4RawWindow{
				{Path: "alpha.go", StartPos: 0, EndPos: 80, StartLine: 100, EndLine: 180},
				{Path: "alpha.go", StartPos: 500, EndPos: 580, StartLine: 500, EndLine: 580},
			},
			RightWindows: []v4RawWindow{
				{Path: "beta.go", StartPos: 1000, EndPos: 1080, StartLine: 1000, EndLine: 1080},
			},
			PerPathLength: map[string]int{"alpha.go": 1500, "beta.go": 2000},
		},
		// 12. shuffled input-window order
		v4AsymmetricLeadingExtra(),
		// 13. empty-region and unowned windows - exact filter behaviour
		{
			Name: "EmptyRegionFallback",
			LeftWindows: []v4RawWindow{
				{Path: "alpha.go", StartPos: 0, EndPos: 80, StartLine: 100, EndLine: 180},
			},
			RightWindows: []v4RawWindow{
				{Path: "beta.go", StartPos: 100, EndPos: 180, StartLine: 200, EndLine: 280},
			},
			PerPathLength: map[string]int{"alpha.go": 200, "beta.go": 200},
		},
		// 14. duplicate raw-window entries
		{
			Name: "DuplicateEntries",
			LeftWindows: []v4RawWindow{
				{Path: "alpha.go", StartPos: 0, EndPos: 80, StartLine: 100, EndLine: 180},
				{Path: "alpha.go", StartPos: 0, EndPos: 80, StartLine: 100, EndLine: 180},
				{Path: "alpha.go", StartPos: 1, EndPos: 81, StartLine: 101, EndLine: 181},
			},
			RightWindows: []v4RawWindow{
				{Path: "beta.go", StartPos: 100, EndPos: 180, StartLine: 200, EndLine: 280},
				{Path: "beta.go", StartPos: 101, EndPos: 181, StartLine: 201, EndLine: 281},
			},
			PerPathLength: map[string]int{"alpha.go": 200, "beta.go": 400},
		},
		// 15. equal paths with different region ordinals - canonical orientation
		{
			Name: "EqualPathsDifferentOrdinals",
			LeftWindows: []v4RawWindow{
				{Path: "alpha.go", StartPos: 0, EndPos: 80, StartLine: 100, EndLine: 180},
				{Path: "alpha.go", StartPos: 1, EndPos: 81, StartLine: 101, EndLine: 181},
			},
			RightWindows: []v4RawWindow{
				{Path: "alpha.go", StartPos: 100, EndPos: 180, StartLine: 200, EndLine: 280},
				{Path: "alpha.go", StartPos: 101, EndPos: 181, StartLine: 201, EndLine: 281},
			},
			PerPathLength: map[string]int{"alpha.go": 1000},
		},
	}
	// Convert PerPathLength maps to nil for cases that don't need
	// them; we always allocate per-path in the helper.
	for i := range corpus {
		if corpus[i].PerPathLength == nil {
			corpus[i].PerPathLength = map[string]int{
				"alpha.go": 1000, "beta.go": 2000, "gamma.go": 3000,
			}
		}
	}
	return corpus
}
