// Package dupcode provides explicit multi-region, same-path ordinal,
// unowned-window, and shuffled-input proofs for the semantic corpus.
package dupcode

import (
	"reflect"
	"testing"
)

func TestV4Alignment_CorpusAnalysisUsesDeclaredRegions(t *testing.T) {
	samePath := v4SamePathDifferentOrdinalsFixture()
	analyses := v4BuildFixtureAnalyses(samePath)
	analysis := analyses["shared.go"]
	if analysis == nil {
		t.Fatal("shared.go analysis is absent")
	}
	if len(analysis.Regions) != 2 {
		t.Fatalf("shared.go region count=%d, want 2", len(analysis.Regions))
	}
	wantLeft := v4SyntaxRegionID{Path: "shared.go", Ordinal: 0}
	wantRight := v4SyntaxRegionID{Path: "shared.go", Ordinal: 1}
	if got := analysis.TokenOwner[0]; got != wantLeft {
		t.Fatalf("shared.go token 0 owner=%s, want %s", got, wantLeft)
	}
	if got := analysis.TokenOwner[200]; got != wantRight {
		t.Fatalf("shared.go token 200 owner=%s, want %s", got, wantRight)
	}
	if got := analysis.TokenOwner[120]; got.Path != "" {
		t.Fatalf("shared.go inter-region token 120 owner=%s, want unowned", got)
	}

	unowned := v4UnownedWindowFixture()
	unownedAnalyses := v4BuildFixtureAnalyses(unowned)
	if _, ok := unownedAnalyses["missing.go"]; ok {
		t.Fatal("missing.go unexpectedly received an analysis")
	}
	if got := unownedAnalyses["alpha.go"].TokenOwner[500]; got.Path != "" {
		t.Fatalf("alpha.go token 500 owner=%s, want unowned", got)
	}
}

func TestV4Alignment_SamePathDifferentOrdinalsProof(t *testing.T) {
	fixture := v4SamePathDifferentOrdinalsFixture()
	if fixture.LeftRegion.Path != fixture.RightRegion.Path {
		t.Fatalf("paths differ: left=%s right=%s", fixture.LeftRegion, fixture.RightRegion)
	}
	if fixture.LeftRegion.Ordinal == fixture.RightRegion.Ordinal {
		t.Fatalf("ordinals collapse: left=%s right=%s", fixture.LeftRegion, fixture.RightRegion)
	}

	analyses := v4BuildFixtureAnalyses(fixture)
	filtered := filterWindowsToRegions(v4FixtureWindowMap(fixture), analyses)
	combined, _ := v4BuildRegionBoundedChainInputs(filtered, analyses)
	if len(combined) == 0 {
		t.Fatal("same-path/different-ordinal fixture emitted no candidates")
	}
	for i, match := range combined {
		if match.LeftRegion != fixture.LeftRegion || match.RightRegion != fixture.RightRegion {
			t.Fatalf("candidate %d orientation=(%s,%s), want=(%s,%s)",
				i, match.LeftRegion, match.RightRegion, fixture.LeftRegion, fixture.RightRegion)
		}
		if match.Match.Left.Path != match.Match.Right.Path {
			t.Fatalf("candidate %d paths differ: left=%q right=%q",
				i, match.Match.Left.Path, match.Match.Right.Path)
		}
	}

	production := v4RunProductionCorpusFixture(fixture)
	oracle := v4RunOracleCorpusFixture(fixture)
	v4AssertDifferentialResultsEqual(t, fixture.Name, production, oracle)

	shuffled := fixture
	shuffled.Name += "/shuffled-variant"
	shuffled.RawWindows = v4PermuteRawWindows(fixture.RawWindows)
	if reflect.DeepEqual(shuffled.RawWindows, fixture.RawWindows) {
		t.Fatal("same-path shuffled variant did not change raw input order")
	}
	v4AssertDifferentialResultsEqual(
		t,
		shuffled.Name,
		production,
		v4RunProductionCorpusFixture(shuffled),
	)
	v4AssertDifferentialResultsEqual(
		t,
		shuffled.Name+"/oracle",
		oracle,
		v4RunOracleCorpusFixture(shuffled),
	)
}

func TestV4Alignment_UnownedWindowProof(t *testing.T) {
	fixture := v4UnownedWindowFixture()
	production := v4RunProductionCorpusFixture(fixture)
	oracle := v4RunOracleCorpusFixture(fixture)
	v4AssertDifferentialResultsEqual(t, fixture.Name, production, oracle)

	wantDiagnostics := []v4OwnershipDiagnostic{
		{Classification: "missing-analysis", Path: "missing.go", StartPos: 10, EndPos: 25},
		{Classification: "outside-declared-region", Path: "alpha.go", StartPos: 500, EndPos: 515},
	}
	if !reflect.DeepEqual(production.Diagnostics, wantDiagnostics) {
		t.Fatalf("ownership diagnostics=%#v, want %#v", production.Diagnostics, wantDiagnostics)
	}
	if len(production.KeptWindows) != 6 {
		t.Fatalf("kept-window count=%d, want 6 after two discards", len(production.KeptWindows))
	}
	if len(production.Findings) == 0 {
		t.Fatal("all valid canonical findings disappeared with unowned windows")
	}
	if production.Error.Present || oracle.Error.Present {
		t.Fatalf("unexpected error classification: production=%#v oracle=%#v",
			production.Error, oracle.Error)
	}
}

func TestV4Alignment_ShuffledInputProof(t *testing.T) {
	cases := []struct {
		name    string
		fixture v4CorpusFixture
	}{
		{name: "aligned", fixture: v4AlignedCorpusFixture(v4AlignedN8, 8)},
		{name: "unaligned", fixture: v4OffIndexMaximalChainFixture()},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			canonical := tc.fixture
			canonical.RawWindows = v4CanonicalRawWindows(canonical.RawWindows)
			shuffled := canonical
			shuffled.Name += "/shuffled-variant"
			shuffled.RawWindows = v4PermuteRawWindows(canonical.RawWindows)
			if reflect.DeepEqual(canonical.RawWindows, shuffled.RawWindows) {
				t.Fatal("shuffled raw input equals canonical order")
			}
			if v4PathTransitions(shuffled.RawWindows) < 3 {
				t.Fatalf("shuffled raw input does not mix paths: %#v", shuffled.RawWindows)
			}

			productionCanonical := v4RunProductionCorpusFixture(canonical)
			productionShuffled := v4RunProductionCorpusFixture(shuffled)
			oracleCanonical := v4RunOracleCorpusFixture(canonical)
			oracleShuffled := v4RunOracleCorpusFixture(shuffled)
			v4AssertDifferentialResultsEqual(t, "production canonical/shuffled", productionCanonical, productionShuffled)
			v4AssertDifferentialResultsEqual(t, "oracle canonical/shuffled", oracleCanonical, oracleShuffled)
			v4AssertDifferentialResultsEqual(t, "production/oracle shuffled", productionShuffled, oracleShuffled)
		})
	}
}

func v4PathTransitions(windows []v4RawWindow) int {
	transitions := 0
	for i := 1; i < len(windows); i++ {
		if windows[i-1].Path != windows[i].Path {
			transitions++
		}
	}
	return transitions
}
