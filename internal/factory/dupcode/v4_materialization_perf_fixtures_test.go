// Package dupcode provides the deterministic V4 materialization
// performance fixture creators for
// ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01.
//
// Every fixture is built from a closure or array literal. The fixtures
// do not depend on the developer's working tree, the wall clock, a
// temporary directory name, or map iteration order. The fixtures
// carry a synthetic v4FileAnalysis map so the benchmarks exercise the
// production V4 region-bounded chain construction rather than the V3
// fallback used by v4CoalesceFindings.
package dupcode

import (
	"go/token"
	"runtime"
	"strconv"
	"testing"
)

// indexCount describes one fixture variant in the N-way grid.
type indexCount struct {
	size          int
	descriptiveID string
}

// fixtureSizes is the canonical size list required by R1 of the ACT.
var fixtureSizes = []indexCount{
	{size: 8, descriptiveID: "N8"},
	{size: 32, descriptiveID: "N32"},
	{size: 128, descriptiveID: "N128"},
}

// makeSyntheticFileAnalysis builds a minimal v4FileAnalysis for
// synthetic fixtures. Every token in [0, length) is assigned to a
// single region whose OwnerID encodes the file path so the synthetic
// regions are distinguishable across distinct file paths.
func makeSyntheticFileAnalysis(path string, length int) *v4FileAnalysis {
	if length <= 0 {
		length = 1
	}
	tokens := make([]token.Token, length)
	lines := make([]int, length)
	entries := make([]v4TokenEntry, length)
	normalized := make([]string, length)
	owner := v4SyntaxRegionID{Path: path, Ordinal: 0}
	tokenOwners := make([]v4SyntaxRegionID, length)
	for i := 0; i < length; i++ {
		tokens[i] = token.IDENT
		lines[i] = i + 1
		entries[i] = v4TokenEntry{Pos: token.Pos(i + 1), Tok: token.IDENT}
		normalized[i] = "IDENT"
		tokenOwners[i] = owner
	}
	region := v4SyntaxRegion{
		Path:      path,
		Kind:      v4FunctionDeclarationRegion,
		Ordinal:   0,
		StartPos:  0,
		EndPos:    length - 1,
		StartLine: 1,
		EndLine:   length,
	}
	return &v4FileAnalysis{
		Path:             path,
		Tokens:           tokens,
		Lines:            lines,
		Entries:          entries,
		Regions:          []v4SyntaxRegion{region},
		TokenOwner:       tokenOwners,
		NormalizedTokens: normalized,
	}
}

// collectAnalysesForWindowMap builds a synthetic analysis map for
// every unique file path mentioned in windowMap. Each file gets one
// big region covering [0, maxEnd+1] where maxEnd is the largest
// EndPos seen for that path.
func collectAnalysesForWindowMap(wm map[string][]rawWindow) map[string]*v4FileAnalysis {
	maxEnd := make(map[string]int)
	for _, wins := range wm {
		for _, w := range wins {
			if w.EndPos > maxEnd[w.Path] {
				maxEnd[w.Path] = w.EndPos
			}
		}
	}
	out := make(map[string]*v4FileAnalysis, len(maxEnd))
	for path, end := range maxEnd {
		out[path] = makeSyntheticFileAnalysis(path, end+2)
	}
	return out
}

// makeSlidingWindowMap builds a fingerprint-bucketed window map
// representing a two-file exact duplicate.
func makeSlidingWindowMap(positions int) map[string][]rawWindow {
	const fp = "perf-marker"
	wm := make(map[string][]rawWindow)
	wm[fp] = make([]rawWindow, 0, positions*2)
	for i := 0; i < positions; i++ {
		wm[fp] = append(wm[fp],
			rawWindow{
				Path:      "left.go",
				StartPos:  i,
				EndPos:    i + 40,
				StartLine: 10 + i,
				EndLine:   10 + i + 40,
			},
			rawWindow{
				Path:      "right.go",
				StartPos:  i + 1000,
				EndPos:    i + 1040,
				StartLine: 100 + i,
				EndLine:   100 + i + 40,
			},
		)
	}
	return wm
}

// makeTwoIndependentBodies builds a fingerprint-bucketed window map
// with a shared sliding bucket plus two disjoint two-file duplicate
// bodies.
func makeTwoIndependentBodies(positions int) map[string][]rawWindow {
	const sharedFP = "shared-exact-dup"
	const bodyAFP = "body-A"
	const bodyBFP = "body-B"
	wm := make(map[string][]rawWindow)
	wm[sharedFP] = make([]rawWindow, 0, positions*2)
	for i := 0; i < positions; i++ {
		wm[sharedFP] = append(wm[sharedFP],
			rawWindow{
				Path:      "shared-left.go",
				StartPos:  i,
				EndPos:    i + 40,
				StartLine: 10 + i,
				EndLine:   10 + i + 40,
			},
			rawWindow{
				Path:      "shared-right.go",
				StartPos:  i + 1000,
				EndPos:    i + 1040,
				StartLine: 100 + i,
				EndLine:   100 + i + 40,
			},
		)
	}
	wm[bodyAFP] = []rawWindow{
		{Path: "a-left.go", StartPos: 0, EndPos: 40, StartLine: 1, EndLine: 41},
		{Path: "a-right.go", StartPos: 100, EndPos: 140, StartLine: 1, EndLine: 41},
	}
	wm[bodyBFP] = []rawWindow{
		{Path: "b-left.go", StartPos: 0, EndPos: 40, StartLine: 1, EndLine: 41},
		{Path: "b-right.go", StartPos: 100, EndPos: 140, StartLine: 1, EndLine: 41},
	}
	return wm
}

// makeShadowFixture builds a fingerprint-bucketed window map with
// three windows in each file at positions that create overlapping
// shadow windows.
func makeShadowFixture() map[string][]rawWindow {
	const fp = "shadow-marker"
	wm := make(map[string][]rawWindow)
	wm[fp] = make([]rawWindow, 0, 6)
	for _, base := range []int{0, 8, 16} {
		wm[fp] = append(wm[fp],
			rawWindow{
				Path:      "left.go",
				StartPos:  base,
				EndPos:    base + 40,
				StartLine: 10 + base,
				EndLine:   10 + base + 40,
			},
			rawWindow{
				Path:      "right.go",
				StartPos:  base + 1000,
				EndPos:    base + 1040,
				StartLine: 100 + base,
				EndLine:   100 + base + 40,
			},
		)
	}
	return wm
}

// emptyWindowMap returns the empty map fixture required by the ACT.
func emptyWindowMap() map[string][]rawWindow {
	return map[string][]rawWindow{}
}

// repeatedMultiplicityFixture is one fingerprint bucket with two
// within-file occurrences and one cross-file occurrence.
func repeatedMultiplicityFixture() map[string][]rawWindow {
	const fp = "repeated-multiplicity"
	return map[string][]rawWindow{
		fp: {
			{Path: "left.go", StartPos: 0, EndPos: 40, StartLine: 1, EndLine: 41},
			{Path: "left.go", StartPos: 100, EndPos: 140, StartLine: 50, EndLine: 90},
			{Path: "right.go", StartPos: 200, EndPos: 240, StartLine: 10, EndLine: 50},
		},
	}
}

// canonicalChainInputsString renders the combined slice and partition
// counts for a windowMap. The rendering is deterministic because
// v4BuildRegionBoundedChainInputs sorts fingerprints before
// processing them.
func canonicalChainInputsString(wm map[string][]rawWindow) string {
	analyses := collectAnalysesForWindowMap(wm)
	combined, partitions := v4BuildRegionBoundedChainInputs(wm, analyses)
	out := "combined=" + strconv.Itoa(len(combined)) + "|partitions="
	for _, key := range sortedChainKeys(partitions) {
		out += key.String() + ":" + strconv.Itoa(len(partitions[key])) + ";"
	}
	return out
}

// sortedChainKeys returns the chain-pair keys in the same order used
// by the production pipeline so the determinism comparison stays
// stable across runs.
func sortedChainKeys(partitions map[v4ChainPairKey][]v4RegionSeedMatch) []v4ChainPairKey {
	keys := make([]v4ChainPairKey, 0, len(partitions))
	for k := range partitions {
		keys = append(keys, k)
	}
	sortChainPairKeys(keys)
	return keys
}

// TestV4Perf_FixturesAreDeterministic guards against fixture-level
// non-determinism. It runs every representative benchmark fixture
// through the V4 chain-input build twice and asserts the same
// partition counts and combined-slice lengths.
func TestV4Perf_FixturesAreDeterministic(t *testing.T) {
	type spec struct {
		name string
		make func() map[string][]rawWindow
	}
	specs := []spec{
		{"N8", func() map[string][]rawWindow { return makeSlidingWindowMap(8) }},
		{"N32", func() map[string][]rawWindow { return makeSlidingWindowMap(32) }},
		{"N128", func() map[string][]rawWindow { return makeSlidingWindowMap(128) }},
		{"TwoIndependentBodies", func() map[string][]rawWindow { return makeTwoIndependentBodies(64) }},
		{"ShadowFixture", makeShadowFixture},
		{"RepeatedMultiplicity", repeatedMultiplicityFixture},
		{"EmptyCorpus", emptyWindowMap},
	}
	for _, s := range specs {
		s := s
		t.Run(s.name, func(t *testing.T) {
			first := canonicalChainInputsString(s.make())
			second := canonicalChainInputsString(s.make())
			if first != second {
				t.Fatalf("%s fixture is non-deterministic:\n  first=%s\n  second=%s",
					s.name, first, second)
			}
		})
	}
}

// TestV4Perf_BenchmarkSizes pins the fixture sizes used by the
// benchmarks to the R1 requirements.
func TestV4Perf_BenchmarkSizes(t *testing.T) {
	requirements := []int{8, 32, 128}
	for _, expected := range requirements {
		found := false
		for _, fixture := range fixtureSizes {
			if fixture.size == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing required fixture size %d", expected)
		}
	}
	got := strconv.Itoa(runtime.NumCPU()) + "/" +
		strconv.Itoa(runtime.GOMAXPROCS(0)) + "/" +
		strconv.Itoa(len(fixtureSizes))
	if len(got) == 0 {
		t.Fatal("diagnostic label empty")
	}
}
