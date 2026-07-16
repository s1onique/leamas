package dupcode

import (
	"go/token"
	"testing"
)

func TestV4RegionBoundedChain_IndependentRegionsUseDistinctPartitions(t *testing.T) {
	left := manualRegionAnalysis("a.go", 0, 9, 0, 10, 1)
	right := manualRegionAnalysis("b.go", 0, 9, 0, 10, 1)
	windows := map[string][]rawWindow{
		"seed": {
			{Path: "a.go", StartPos: 0, EndPos: 4},
			{Path: "a.go", StartPos: 10, EndPos: 14},
			{Path: "b.go", StartPos: 0, EndPos: 4},
			{Path: "b.go", StartPos: 10, EndPos: 14},
		},
	}
	filtered := filterWindowsToRegions(windows, map[string]*v4FileAnalysis{"a.go": &left, "b.go": &right})
	_, partitions := v4BuildRegionBoundedChainInputs(filtered, map[string]*v4FileAnalysis{"a.go": &left, "b.go": &right})
	if len(partitions) < 2 {
		t.Fatalf("independent regions were conflated into one partition: %d", len(partitions))
	}
}

func TestV4RegionBoundedChain_FilterRejectsUnownedToken(t *testing.T) {
	analysis := manualRegionAnalysis("a.go", 2, 9, 0, 10, 1)
	windows := map[string][]rawWindow{
		"seed": {{Path: "a.go", StartPos: 0, EndPos: 4}},
	}
	if got := filterWindowsToRegions(windows, map[string]*v4FileAnalysis{"a.go": &analysis}); len(got) != 0 {
		t.Fatalf("unowned window survived region filtering: %+v", got)
	}
}

func TestV4RegionBoundedChain_FilterDoesNotMutateInput(t *testing.T) {
	analysis := manualRegionAnalysis("a.go", 0, 9, 0, 10, 1)
	original := rawWindow{Path: "a.go", StartPos: 0, EndPos: 4}
	windows := map[string][]rawWindow{"seed": {original}}
	_ = filterWindowsToRegions(windows, map[string]*v4FileAnalysis{"a.go": &analysis})
	if windows["seed"][0] != original {
		t.Fatalf("region filtering mutated input window: %+v", windows["seed"][0])
	}
}

func manualRegionAnalysis(path string, firstStart, firstEnd, firstOrdinal, secondStart, secondOrdinal int) v4FileAnalysis {
	n := 20
	tokens := make([]token.Token, n)
	lines := make([]int, n)
	entries := make([]v4TokenEntry, n)
	owners := make([]v4SyntaxRegionID, n)
	normalized := make([]string, n)
	for i := 0; i < n; i++ {
		tokens[i] = token.IDENT
		lines[i] = i + 1
		entries[i] = v4TokenEntry{Pos: token.Pos(i + 1), Tok: token.IDENT}
		normalized[i] = "IDENT"
	}
	first := v4SyntaxRegion{Path: path, Kind: v4FunctionDeclarationRegion, Ordinal: firstOrdinal, StartPos: firstStart, EndPos: firstEnd, StartLine: firstStart + 1, EndLine: firstEnd + 1}
	second := v4SyntaxRegion{Path: path, Kind: v4FunctionDeclarationRegion, Ordinal: secondOrdinal, StartPos: secondStart, EndPos: 19, StartLine: secondStart + 1, EndLine: 20}
	for i := firstStart; i <= firstEnd; i++ {
		owners[i] = v4SyntaxRegionID{Path: path, Ordinal: firstOrdinal}
	}
	for i := secondStart; i < n; i++ {
		owners[i] = v4SyntaxRegionID{Path: path, Ordinal: secondOrdinal}
	}
	return v4FileAnalysis{Path: path, Tokens: tokens, Lines: lines, Entries: entries, Regions: []v4SyntaxRegion{first, second}, TokenOwner: owners, NormalizedTokens: normalized}
}
