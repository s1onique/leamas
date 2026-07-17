// Package dupcode provides the single authoritative structural
// comparator used by deterministic, shuffled, ownership, and fuzz proofs.
package dupcode

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type v4WindowProjection struct {
	Path      string
	StartPos  int
	EndPos    int
	StartLine int
	EndLine   int
}

type v4OccurrenceProjection struct {
	Path      string
	StartPos  int
	EndPos    int
	StartLine int
	EndLine   int
}

type v4FindingProjection struct {
	StableFingerprint string
	TokenCount        int
	LineCount         int
	Occurrences       []v4OccurrenceProjection
}

type v4OwnershipDiagnostic struct {
	Classification string
	Path           string
	StartPos       int
	EndPos         int
}

type v4ErrorProjection struct {
	Present        bool
	Classification string
}

type v4DifferentialResult struct {
	KeptWindows []v4WindowProjection
	Findings    []v4FindingProjection
	Diagnostics []v4OwnershipDiagnostic
	Error       v4ErrorProjection
}

func v4RunProductionCorpusFixture(fx v4CorpusFixture) v4DifferentialResult {
	analyses := v4BuildFixtureAnalyses(fx)
	files := v4BuildFixtureFiles(analyses)
	windowMap := v4FixtureWindowMap(fx)
	filtered := filterWindowsToRegions(windowMap, analyses)
	findings, err := v4BuildInternalFindings(windowMap, analyses, files)
	return v4DifferentialResult{
		KeptWindows: v4ProjectWindowMap(filtered),
		Findings:    v4ProjectFindings(findings),
		Diagnostics: v4ProductionOwnershipDiagnostics(fx, analyses),
		Error:       v4ProjectError(err),
	}
}

func v4RunOracleCorpusFixture(fx v4CorpusFixture) v4DifferentialResult {
	analyses := v4BuildFixtureAnalyses(fx)
	files := v4BuildFixtureFiles(analyses)
	filtered, diagnostics := v4OracleFilterFixture(fx)
	findings, err := v4RunFullPipelineForOracleFiltered(
		filtered,
		analyses,
		files,
		v4GenerateAllPairsMatchesOracle,
	)
	return v4DifferentialResult{
		KeptWindows: v4ProjectWindowMap(filtered),
		Findings:    v4ProjectFindings(findings),
		Diagnostics: diagnostics,
		Error:       v4ProjectError(err),
	}
}

func v4ProjectFindings(findings []v4InternalFinding) []v4FindingProjection {
	if findings == nil {
		return nil
	}
	out := make([]v4FindingProjection, len(findings))
	for i, finding := range findings {
		out[i] = v4FindingProjection{
			StableFingerprint: finding.StableFingerprint,
			TokenCount:        finding.TokenCount,
			LineCount:         finding.LineCount,
			Occurrences:       make([]v4OccurrenceProjection, len(finding.Occurrences)),
		}
		for j, occurrence := range finding.Occurrences {
			out[i].Occurrences[j] = v4OccurrenceProjection{
				Path: occurrence.Path, StartPos: occurrence.StartPos,
				EndPos: occurrence.EndPos, StartLine: occurrence.StartLine,
				EndLine: occurrence.EndLine,
			}
		}
	}
	return out
}

func v4ProjectWindowMap(windowMap map[string][]rawWindow) []v4WindowProjection {
	var out []v4WindowProjection
	for _, windows := range windowMap {
		for _, w := range windows {
			out = append(out, v4WindowProjection{
				Path: w.Path, StartPos: w.StartPos, EndPos: w.EndPos,
				StartLine: w.StartLine, EndLine: w.EndLine,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return v4WindowProjectionLess(out[i], out[j]) })
	return out
}

func v4WindowProjectionLess(left, right v4WindowProjection) bool {
	if left.Path != right.Path {
		return left.Path < right.Path
	}
	if left.StartPos != right.StartPos {
		return left.StartPos < right.StartPos
	}
	if left.EndPos != right.EndPos {
		return left.EndPos < right.EndPos
	}
	if left.StartLine != right.StartLine {
		return left.StartLine < right.StartLine
	}
	return left.EndLine < right.EndLine
}

func v4ProductionOwnershipDiagnostics(
	fx v4CorpusFixture,
	analyses map[string]*v4FileAnalysis,
) []v4OwnershipDiagnostic {
	var diagnostics []v4OwnershipDiagnostic
	for _, w := range fx.RawWindows {
		a, ok := analyses[w.Path]
		if !ok {
			diagnostics = append(diagnostics, v4DiscardDiagnostic("missing-analysis", w))
			continue
		}
		if _, ok := a.windowFitsRegion(w.StartPos, w.EndPos); !ok {
			diagnostics = append(diagnostics, v4DiscardDiagnostic("outside-declared-region", w))
		}
	}
	v4SortOwnershipDiagnostics(diagnostics)
	return diagnostics
}

func v4OracleFilterFixture(fx v4CorpusFixture) (
	map[string][]rawWindow,
	[]v4OwnershipDiagnostic,
) {
	filtered := make(map[string][]rawWindow)
	var diagnostics []v4OwnershipDiagnostic
	for _, w := range fx.RawWindows {
		if !v4FixtureDeclaresAnalysisPath(fx, w.Path) {
			diagnostics = append(diagnostics, v4DiscardDiagnostic("missing-analysis", w))
			continue
		}
		if _, ok := v4DeclaredWindowOwner(fx, w); !ok {
			diagnostics = append(diagnostics, v4DiscardDiagnostic("outside-declared-region", w))
			continue
		}
		filtered["corpus-seed"] = append(filtered["corpus-seed"], rawWindow{
			Path: w.Path, StartPos: w.StartPos, EndPos: w.EndPos,
			StartLine: w.StartLine, EndLine: w.EndLine,
		})
	}
	v4SortOwnershipDiagnostics(diagnostics)
	return filtered, diagnostics
}

func v4FixtureDeclaresAnalysisPath(fx v4CorpusFixture, path string) bool {
	if _, ok := fx.FileLength[path]; ok {
		return true
	}
	for _, region := range fx.Regions {
		if region.Path == path {
			return true
		}
	}
	return false
}

func v4DiscardDiagnostic(classification string, w v4RawWindow) v4OwnershipDiagnostic {
	return v4OwnershipDiagnostic{
		Classification: classification,
		Path:           w.Path, StartPos: w.StartPos, EndPos: w.EndPos,
	}
}

func v4SortOwnershipDiagnostics(diagnostics []v4OwnershipDiagnostic) {
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Classification != diagnostics[j].Classification {
			return diagnostics[i].Classification < diagnostics[j].Classification
		}
		if diagnostics[i].Path != diagnostics[j].Path {
			return diagnostics[i].Path < diagnostics[j].Path
		}
		if diagnostics[i].StartPos != diagnostics[j].StartPos {
			return diagnostics[i].StartPos < diagnostics[j].StartPos
		}
		return diagnostics[i].EndPos < diagnostics[j].EndPos
	})
}

func v4ProjectError(err error) v4ErrorProjection {
	if err == nil {
		return v4ErrorProjection{}
	}
	var chain []string
	for current := err; current != nil; current = errors.Unwrap(current) {
		chain = append(chain, fmt.Sprintf("%T", current))
	}
	return v4ErrorProjection{Present: true, Classification: strings.Join(chain, " -> ")}
}

func v4AssertDifferentialResultsEqual(
	t *testing.T,
	label string,
	left, right v4DifferentialResult,
) {
	t.Helper()
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("%s: canonical internal structural mismatch\nleft:  %#v\nright: %#v",
			label, left, right)
	}
}

func TestV4Alignment_CorpusProductionEqualsOracle(t *testing.T) {
	corpus := v4BuildAlignmentCorpus()
	v4RequireCorpusContracts(t, corpus)
	for _, fixture := range corpus {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			production := v4RunProductionCorpusFixture(fixture)
			oracle := v4RunOracleCorpusFixture(fixture)
			v4AssertDifferentialResultsEqual(t, fixture.Name, production, oracle)
		})
	}
}
