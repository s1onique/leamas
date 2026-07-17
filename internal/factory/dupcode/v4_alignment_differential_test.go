// Package dupcode provides the differential test corpus for
// ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION01.
//
// The differential tests consume the oracle from
// v4_alignment_oracle_test.go and assert byte-identical canonical
// findings for the production pipeline.
//
// This file owns:
//
//   - v4RawWindow: the per-corpus-case test-only window shape;
//   - v4PerfFixture: the corpus case record;
//   - v4BuildAlignedWindowMap / v4MakeAlignedAnalyses /
//     v4MakeAlignedFiles: synthetic analysis/file constructors;
//   - v4RunDifferentialCase: the byte-identical production-vs-oracle
//     comparison helper;
//   - v4SlidingAlignedFixture / v4AsymmetricLeadingExtra: the two
//     fixture constructors exposed to the corpus builder;
//   - the TestV4Alignment_* tests and the v4BuildDeterministicCorpus
//     builder.
package dupcode

import (
	"go/token"
	"testing"
)

type v4RawWindow struct {
	Path      string
	StartPos  int
	EndPos    int
	StartLine int
	EndLine   int
}

// v4BuildAlignedWindowMap constructs a one-fingerprint-bucket
// windowMap from the supplied per-path window sequences. The
// resulting windows belong to one shared seed fingerprint.
func v4BuildAlignedWindowMap(seed string, leftWindows, rightWindows []v4RawWindow) map[string][]rawWindow {
	wm := make(map[string][]rawWindow)
	for _, w := range leftWindows {
		wm[seed] = append(wm[seed], rawWindow{
			Path:      w.Path,
			StartPos:  w.StartPos,
			EndPos:    w.EndPos,
			StartLine: w.StartLine,
			EndLine:   w.EndLine,
		})
	}
	for _, w := range rightWindows {
		wm[seed] = append(wm[seed], rawWindow{
			Path:      w.Path,
			StartPos:  w.StartPos,
			EndPos:    w.EndPos,
			StartLine: w.StartLine,
			EndLine:   w.EndLine,
		})
	}
	return wm
}

// v4MakeAlignedAnalyses returns synthetic analyses covering the
// supplied StartPos/EndPos ranges for each path. Every token in
// [0, length) is owned by the file's region-0; the synthetic
// regions are distinguishable across distinct file paths because
// the owner ID encodes the path.
func v4MakeAlignedAnalyses(perPathLength map[string]int, perPathWindows map[string][]v4RawWindow) map[string]*v4FileAnalysis {
	out := make(map[string]*v4FileAnalysis, len(perPathLength))
	for path, length := range perPathLength {
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
		out[path] = &v4FileAnalysis{
			Path:             path,
			Tokens:           tokens,
			Lines:            lines,
			Entries:          entries,
			Regions:          []v4SyntaxRegion{region},
			TokenOwner:       tokenOwners,
			NormalizedTokens: normalized,
		}
	}
	return out
}

// v4MakeAlignedFiles returns file inventory for the analyses.
func v4MakeAlignedFiles(perPathLength map[string]int, perPathWindows map[string][]v4RawWindow, analyses map[string]*v4FileAnalysis) map[string]*v4AnalyzedFile {
	out := make(map[string]*v4AnalyzedFile, len(analyses))
	for path, analysis := range analyses {
		out[path] = &v4AnalyzedFile{
			FileTokens: fileTokens{
				path:   path,
				tokens: analysis.Tokens,
				lines:  analysis.Lines,
			},
			Analysis:         *analysis,
			NormalizedTokens: analysis.NormalizedTokens,
		}
	}
	return out
}

// v4PerfFixture is the per-corpus-case fixture consumed by the
// differential comparison helpers below.
type v4PerfFixture struct {
	Name          string
	WindowSize    int
	LeftWindows   []v4RawWindow
	RightWindows  []v4RawWindow
	PerPathLength map[string]int
}

// v4RunDifferentialCase runs a single differential corpus case and
// returns true iff the production and oracle canonical findings are
// byte-identical. Each difference is reported via t.Errorf so a
// single failing case is easy to localise.
func v4RunDifferentialCase(t *testing.T, fx v4PerfFixture) {
	t.Helper()
	wm := v4BuildAlignedWindowMap("seed", fx.LeftWindows, fx.RightWindows)
	analyses := v4MakeAlignedAnalyses(fx.PerPathLength, nil)
	files := v4MakeAlignedFiles(fx.PerPathLength, nil, analyses)
	prod, err := v4BuildInternalFindings(wm, analyses, files)
	if err != nil {
		t.Fatalf("%s: production pipeline error: %v", fx.Name, err)
	}
	oracle, err := v4BuildInternalFindingsOracle(wm, analyses, files, v4GenerateAllPairsMatchesOracle)
	if err != nil {
		t.Fatalf("%s: oracle pipeline error: %v", fx.Name, err)
	}
	if len(prod) != len(oracle) {
		t.Fatalf("%s: finding-count drift production=%d oracle=%d", fx.Name, len(prod), len(oracle))
	}
	for i := range prod {
		pa, oa := prod[i], oracle[i]
		if pa.StableFingerprint != oa.StableFingerprint {
			t.Errorf("%s: fingerprint drift at finding %d\n  prod=%q\n  ora =%q", fx.Name, i, pa.StableFingerprint, oa.StableFingerprint)
		}
		if pa.TokenCount != oa.TokenCount {
			t.Errorf("%s: token-count drift at finding %d prod=%d ora=%d", fx.Name, i, pa.TokenCount, oa.TokenCount)
		}
		if len(pa.Occurrences) != len(oa.Occurrences) {
			t.Errorf("%s: occurrence-count drift at finding %d prod=%d ora=%d", fx.Name, i, len(pa.Occurrences), len(oa.Occurrences))
		}
		for j := range pa.Occurrences {
			po, oo := pa.Occurrences[j], oa.Occurrences[j]
			if po.Path != oo.Path || po.StartLine != oo.StartLine || po.EndLine != oo.EndLine {
				t.Errorf("%s: occurrence drift f%d o%d\n  prod=(%s %d-%d)\n  ora =(%s %d-%d)",
					fx.Name, i, j, po.Path, po.StartLine, po.EndLine,
					oo.Path, oo.StartLine, oo.EndLine)
			}
		}
	}
}

// v4SlidingAlignedFixture constructs the canonical aligned sliding
// fixture of size `n` per file (so the cluster size across two
// files is 2n).
func v4SlidingAlignedFixture(n int) v4PerfFixture {
	left := make([]v4RawWindow, 0, n)
	right := make([]v4RawWindow, 0, n)
	for i := 0; i < n; i++ {
		left = append(left, v4RawWindow{
			Path: "alpha.go", StartPos: i, EndPos: i + 80,
			StartLine: 10 + i, EndLine: 10 + i + 80,
		})
		right = append(right, v4RawWindow{
			Path: "beta.go", StartPos: 1000 + i, EndPos: 1000 + i + 80,
			StartLine: 100 + i, EndLine: 100 + i + 80,
		})
	}
	return v4PerfFixture{
		Name:         "SlidingAligned",
		WindowSize:   n,
		LeftWindows:  left,
		RightWindows: right,
		PerPathLength: map[string]int{
			"alpha.go": 10000, "beta.go": 11000,
		},
	}
}

// makeRawWindows constructs a per-path sequence of v4RawWindow for the
// supplied path and start positions. Every window is 80 tokens wide;
// the line numbers follow the index so each window has a unique
// non-overlapping line range that survives coalesce.
//
// The path is REQUIRED; the fixture MUST NOT infer a window's side
// from its position in some enclosing fixture. The
// CORRECTION02-R1 cross-region proof depends on the explicit path.
func makeRawWindows(path string, starts []int) []v4RawWindow {
	out := make([]v4RawWindow, 0, len(starts))
	for i, sp := range starts {
		out = append(out, v4RawWindow{
			Path:      path,
			StartPos:  sp,
			EndPos:    sp + 80,
			StartLine: 100 + i,
			EndLine:   100 + i + 80,
		})
	}
	return out
}

// v4AsymmetricLeadingExtra constructs left=alpha.go[0,1,2],
// right=beta.go[50,100,101,102]. The two paths resolve to distinct
// production syntax regions, the right side has an extra leading
// occurrence, the diagonal guard returns false, and the maximal
// offset-100 chain survives the conservative all-pairs fallback.
//
// The fixture MUST use the path-aware makeRawWindows constructor;
// using a single-path helper for both sides silently re-collapses
// the fixture to within-region matching.
func v4AsymmetricLeadingExtra() v4PerfFixture {
	return v4PerfFixture{
		Name:          "AsymmetricLeadingExtra",
		WindowSize:    3,
		LeftWindows:   makeRawWindows("alpha.go", []int{0, 1, 2}),
		RightWindows:  makeRawWindows("beta.go", []int{50, 100, 101, 102}),
		PerPathLength: map[string]int{"alpha.go": 200, "beta.go": 200},
	}
}

// TestV4Alignment_AsymmetricLeadingExtra_Regression documents the
// original R1 regression that motivated the alignment guard. With
// the guard removed (or replaced with an unconditional diagonal)
// the production code would emit:
//
//	left[0] ↔ right[0]    offset  50
//	left[1] ↔ right[1]    offset  99
//	left[2] ↔ right[2]    offset 100
//
// and miss the offset-100 chain entirely. The all-pairs oracle
// discovers the maximal constant-offset chain:
//
//	left[0] ↔ right[1]    offset 100
//	left[1] ↔ right[2]    offset 100
//	left[2] ↔ right[3]    offset 100
//
// This test pins the production output against the oracle so any
// future regression that reverts to an unconditional diagonal is
// caught at `go test` time.
func TestV4Alignment_AsymmetricLeadingExtra_Regression(t *testing.T) {
	v4RunDifferentialCase(t, v4AsymmetricLeadingExtra())
}

// TestV4Alignment_DeterministicCorpus exercises every required R3
// differential corpus case and asserts production matches the
// all-pairs oracle on the final canonical output.
func TestV4Alignment_DeterministicCorpus(t *testing.T) {
	corpus := v4BuildDeterministicCorpus()
	for _, fx := range corpus {
		fx := fx
		t.Run(fx.Name, func(t *testing.T) {
			v4RunDifferentialCase(t, fx)
		})
	}
}

// TestV4Alignment_RegionsArePositionallyAligned is a unit-level
// test of the alignment guard predicate. It is small enough to
// run inside the fast subset and pin the guard's behaviour.
func TestV4Alignment_RegionsArePositionallyAligned(t *testing.T) {
	cases := []struct {
		name   string
		leftA  []int
		leftB  []int
		wantOK bool
	}{
		{"EqualEmpty", []int{}, []int{}, true},
		{"EqualAligned", []int{0, 1, 2}, []int{100, 101, 102}, true},
		{"EqualSizesUnaligned", []int{0, 1, 2}, []int{100, 200, 102}, false},
		{"UnequalSizes", []int{0, 1, 2}, []int{100, 101}, false},
		{"UnequalSizesReverse", []int{0, 1}, []int{100, 101, 102}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			aw := make([]v4AnnotatedWindow, 0, len(tc.leftA)+len(tc.leftB))
			idxA := make([]int, len(tc.leftA))
			idxB := make([]int, len(tc.leftB))
			for i, sp := range tc.leftA {
				aw = append(aw, v4AnnotatedWindow{
					w: rawWindow{
						Path: "alpha.go", StartPos: sp,
						EndPos: sp + 80, StartLine: 100 + i, EndLine: 200 + i,
					},
					region: v4SyntaxRegionID{Path: "alpha.go", Ordinal: 0},
				})
				idxA[i] = i
			}
			base := len(aw)
			for j, sp := range tc.leftB {
				aw = append(aw, v4AnnotatedWindow{
					w: rawWindow{
						Path: "beta.go", StartPos: sp,
						EndPos: sp + 80, StartLine: 300 + j, EndLine: 400 + j,
					},
					region: v4SyntaxRegionID{Path: "beta.go", Ordinal: 0},
				})
				idxB[j] = base + j
			}
			got := regionsArePositionallyAligned(idxA, idxB, aw)
			if got != tc.wantOK {
				t.Fatalf("%s: regionsArePositionallyAligned=%v want=%v", tc.name, got, tc.wantOK)
			}
		})
	}
}
