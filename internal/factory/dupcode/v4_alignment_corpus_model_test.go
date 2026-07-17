// Package dupcode provides the declared-region fixture model for the
// CORRECTION02 semantic differential corpus.
package dupcode

import (
	"go/token"
	"sort"
)

// v4FixtureRegion is the test-owned declaration from which synthetic
// token ownership is derived. Tokens outside every declaration remain
// deliberately unowned.
type v4FixtureRegion struct {
	Path      string
	Ordinal   int
	StartPos  int
	EndPos    int
	StartLine int
	EndLine   int
}

type v4CorpusDimension string

type v4CorpusFixture struct {
	Name        string
	Dimension   v4CorpusDimension
	Regions     []v4FixtureRegion
	FileLength  map[string]int
	RawWindows  []v4RawWindow
	LeftRegion  v4SyntaxRegionID
	RightRegion v4SyntaxRegionID
}

func v4BuildFixtureAnalyses(fx v4CorpusFixture) map[string]*v4FileAnalysis {
	lengths := make(map[string]int, len(fx.FileLength)+len(fx.Regions))
	for path, length := range fx.FileLength {
		lengths[path] = length
	}
	for _, region := range fx.Regions {
		if need := region.EndPos + 1; need > lengths[region.Path] {
			lengths[region.Path] = need
		}
	}

	analyses := make(map[string]*v4FileAnalysis, len(lengths))
	for path, length := range lengths {
		if length < 1 {
			length = 1
		}
		a := &v4FileAnalysis{
			Path:             path,
			Tokens:           make([]token.Token, length),
			Lines:            make([]int, length),
			Entries:          make([]v4TokenEntry, length),
			TokenOwner:       make([]v4SyntaxRegionID, length),
			NormalizedTokens: make([]string, length),
		}
		for i := 0; i < length; i++ {
			a.Tokens[i] = token.IDENT
			a.Lines[i] = i + 1
			a.Entries[i] = v4TokenEntry{Pos: token.Pos(i + 1), Tok: token.IDENT}
			a.NormalizedTokens[i] = "IDENT"
		}
		analyses[path] = a
	}

	for _, declared := range fx.Regions {
		a := analyses[declared.Path]
		region := v4SyntaxRegion{
			Path: declared.Path, Kind: v4FunctionDeclarationRegion,
			Ordinal: declared.Ordinal, StartPos: declared.StartPos,
			EndPos: declared.EndPos, StartLine: declared.StartLine,
			EndLine: declared.EndLine,
		}
		a.Regions = append(a.Regions, region)
		owner := v4SyntaxRegionID{Path: declared.Path, Ordinal: declared.Ordinal}
		start, end := declared.StartPos, declared.EndPos
		if start < 0 {
			start = 0
		}
		if end >= len(a.TokenOwner) {
			end = len(a.TokenOwner) - 1
		}
		for pos := start; pos <= end; pos++ {
			a.TokenOwner[pos] = owner
		}
	}
	return analyses
}

func v4BuildFixtureFiles(analyses map[string]*v4FileAnalysis) map[string]*v4AnalyzedFile {
	return v4MakeAlignedFiles(nil, nil, analyses)
}

func v4CorpusFixtureFromPerf(fx v4PerfFixture) v4CorpusFixture {
	paths := make([]string, 0, len(fx.PerPathLength))
	for path := range fx.PerPathLength {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	regions := make([]v4FixtureRegion, 0, len(paths))
	for _, path := range paths {
		length := fx.PerPathLength[path]
		if length < 1 {
			length = 1
		}
		regions = append(regions, v4CorpusRegion(path, 0, 0, length-1))
	}
	leftRegion := v4SyntaxRegionID{}
	rightRegion := v4SyntaxRegionID{}
	if len(fx.LeftWindows) != 0 {
		leftRegion = v4SyntaxRegionID{Path: fx.LeftWindows[0].Path, Ordinal: 0}
	}
	if len(fx.RightWindows) != 0 {
		rightRegion = v4SyntaxRegionID{Path: fx.RightWindows[0].Path, Ordinal: 0}
	}
	return v4CorpusFixture{
		Name: fx.Name, Dimension: v4CorpusDimension(fx.Name),
		Regions: regions, FileLength: fx.PerPathLength,
		RawWindows: append(append([]v4RawWindow(nil), fx.LeftWindows...), fx.RightWindows...),
		LeftRegion: leftRegion, RightRegion: rightRegion,
	}
}

func v4FixtureWindowMap(fx v4CorpusFixture) map[string][]rawWindow {
	windows := make([]rawWindow, 0, len(fx.RawWindows))
	for _, w := range fx.RawWindows {
		windows = append(windows, rawWindow{
			Path: w.Path, StartPos: w.StartPos, EndPos: w.EndPos,
			StartLine: w.StartLine, EndLine: w.EndLine,
		})
	}
	return map[string][]rawWindow{"corpus-seed": windows}
}

func v4DeclaredWindowOwner(fx v4CorpusFixture, w v4RawWindow) (v4SyntaxRegionID, bool) {
	ownerAt := func(pos int) v4SyntaxRegionID {
		var owner v4SyntaxRegionID
		for _, region := range fx.Regions {
			if region.Path == w.Path && pos >= region.StartPos && pos <= region.EndPos {
				owner = v4SyntaxRegionID{Path: region.Path, Ordinal: region.Ordinal}
			}
		}
		return owner
	}
	if w.StartPos < 0 || w.EndPos < w.StartPos {
		return v4SyntaxRegionID{}, false
	}
	owner := ownerAt(w.StartPos)
	if owner.Path == "" {
		return v4SyntaxRegionID{}, false
	}
	for pos := w.StartPos + 1; pos <= w.EndPos; pos++ {
		if ownerAt(pos) != owner {
			return v4SyntaxRegionID{}, false
		}
	}
	return owner, true
}

func v4WindowsForDeclaredRegion(fx v4CorpusFixture, id v4SyntaxRegionID) []v4RawWindow {
	var windows []v4RawWindow
	for _, w := range fx.RawWindows {
		owner, ok := v4DeclaredWindowOwner(fx, w)
		if ok && owner == id {
			windows = append(windows, w)
		}
	}
	sort.Slice(windows, func(i, j int) bool {
		if windows[i].StartPos != windows[j].StartPos {
			return windows[i].StartPos < windows[j].StartPos
		}
		if windows[i].EndPos != windows[j].EndPos {
			return windows[i].EndPos < windows[j].EndPos
		}
		if windows[i].StartLine != windows[j].StartLine {
			return windows[i].StartLine < windows[j].StartLine
		}
		return windows[i].EndLine < windows[j].EndLine
	})
	return windows
}

func v4CanonicalRawWindows(windows []v4RawWindow) []v4RawWindow {
	out := append([]v4RawWindow(nil), windows...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].StartPos != out[j].StartPos {
			return out[i].StartPos < out[j].StartPos
		}
		if out[i].EndPos != out[j].EndPos {
			return out[i].EndPos < out[j].EndPos
		}
		if out[i].StartLine != out[j].StartLine {
			return out[i].StartLine < out[j].StartLine
		}
		return out[i].EndLine < out[j].EndLine
	})
	return out
}
