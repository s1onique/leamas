package dupcode

import (
	"go/token"
	"testing"
)

func TestV4ComponentMerge_OneMaximalClone(t *testing.T) {
	files := manualAnalyzedFiles("a.go", "b.go")
	findings, err := v4MaterializeComponents([]cloneChain{manualChain("a.go", 0, 4, "b.go", 0, 4)}, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || len(findings[0].Occurrences) != 2 || findings[0].TokenCount != 5 {
		t.Fatalf("findings=%+v", findings)
	}
}

func TestV4ComponentMerge_RepeatedMultiplicity(t *testing.T) {
	files := manualAnalyzedFiles("repeat_a.go", "repeat_b.go")
	chains := []cloneChain{
		manualChain("repeat_a.go", 0, 4, "repeat_a.go", 10, 14),
		manualChain("repeat_a.go", 0, 4, "repeat_b.go", 0, 4),
	}
	findings, err := v4MaterializeComponents(chains, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || len(findings[0].Occurrences) != 3 {
		t.Fatalf("findings=%+v", findings)
	}
}

func TestV4ComponentMerge_NWayClone(t *testing.T) {
	files := manualAnalyzedFiles("a.go", "b.go", "c.go")
	chains := []cloneChain{
		manualChain("a.go", 0, 4, "b.go", 0, 4),
		manualChain("b.go", 0, 4, "c.go", 0, 4),
	}
	findings, err := v4MaterializeComponents(chains, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || len(findings[0].Occurrences) != 3 {
		t.Fatalf("connected component was not materialized: %+v", findings)
	}
}

func TestV4ComponentMerge_IndependentBodiesRemainSeparate(t *testing.T) {
	files := manualAnalyzedFiles("a.go", "b.go")
	for _, file := range files {
		file.NormalizedTokens[10] = "-"
		file.Analysis.NormalizedTokens[10] = "-"
	}
	chains := []cloneChain{
		manualChain("a.go", 0, 4, "b.go", 0, 4),
		manualChain("a.go", 10, 14, "b.go", 10, 14),
	}
	findings, err := v4MaterializeComponents(chains, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("independent content keys merged: %+v", findings)
	}
}

func TestV4ComponentMerge_CrossFindingLineConflictFailsClosed(t *testing.T) {
	group := []v4InternalFinding{
		{StableFingerprint: "same", TokenCount: 5, Occurrences: []maximalOccurrence{{Path: "a.go", StartPos: 0, EndPos: 4, StartLine: 1, EndLine: 5}}},
		{StableFingerprint: "same", TokenCount: 5, Occurrences: []maximalOccurrence{{Path: "a.go", StartPos: 0, EndPos: 4, StartLine: 2, EndLine: 6}}},
	}
	if _, err := v4MergeToNWayCloneChecked(group); err == nil {
		t.Fatal("expected cross-finding line conflict to fail closed")
	}
}

func TestV4ComponentMerge_CrossFindingConsistentDuplicateDedups(t *testing.T) {
	occ := maximalOccurrence{Path: "a.go", StartPos: 0, EndPos: 4, StartLine: 1, EndLine: 5}
	group := []v4InternalFinding{
		{StableFingerprint: "same", TokenCount: 5, Occurrences: []maximalOccurrence{occ}},
		{StableFingerprint: "same", TokenCount: 5, Occurrences: []maximalOccurrence{occ}},
	}
	merged, err := v4MergeToNWayCloneChecked(group)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Occurrences) != 1 {
		t.Fatalf("consistent duplicate was not deduplicated: %+v", merged)
	}
}

func TestV4ComponentMerge_DistinctTokenSpansWithSamePublicLinesSurvive(t *testing.T) {
	files := manualAnalyzedFiles("a.go", "b.go")
	chains := []cloneChain{
		manualChain("a.go", 0, 4, "b.go", 0, 4),
		manualChain("a.go", 10, 14, "b.go", 10, 14),
	}
	findings, err := v4MaterializeComponents(chains, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("same public line geometry collapsed distinct spans: %+v", findings)
	}
}

func TestV4ShadowGroupKey_PathPunctuationIsStructural(t *testing.T) {
	paths := []string{"pipe|path.go", "hash#path.go", "colon:path.go", "space path.go", "ユニコード.go"}
	for _, left := range paths {
		for _, right := range paths {
			key := v4ShadowGroupKey{LeftPath: left, LeftRegion: 1, RightPath: right, RightRegion: 2}
			if key.LeftPath != left || key.RightPath != right {
				t.Fatalf("structured key changed path: %+v", key)
			}
		}
	}
}

func manualAnalyzedFiles(paths ...string) map[string]*v4AnalyzedFile {
	files := make(map[string]*v4AnalyzedFile, len(paths))
	for _, path := range paths {
		tokens := make([]token.Token, 20)
		normalized := make([]string, 20)
		lines := make([]int, 20)
		entries := make([]v4TokenEntry, 20)
		owners := make([]v4SyntaxRegionID, 20)
		for i := range tokens {
			tokens[i] = token.IDENT
			normalized[i] = "IDENT"
			lines[i] = 1
			entries[i] = v4TokenEntry{Pos: token.Pos(i + 1), Tok: token.IDENT}
		}
		analysis := v4FileAnalysis{
			Path: path, Tokens: tokens, Lines: lines, Entries: entries,
			TokenOwner: owners, NormalizedTokens: normalized,
		}
		files[path] = &v4AnalyzedFile{
			FileTokens: fileTokens{path: path, tokens: tokens, lines: lines},
			Analysis:   analysis, NormalizedTokens: normalized,
		}
	}
	return files
}

func manualChain(leftPath string, leftStart, leftEnd int, rightPath string, rightStart, rightEnd int) cloneChain {
	return cloneChain{
		Matches: []seedMatch{{
			Left:  rawWindow{Path: leftPath, StartPos: leftStart, EndPos: leftEnd},
			Right: rawWindow{Path: rightPath, StartPos: rightStart, EndPos: rightEnd},
		}},
		LeftRange:  tokenRange{StartPos: leftStart, EndPos: leftEnd},
		RightRange: tokenRange{StartPos: rightStart, EndPos: rightEnd},
	}
}
