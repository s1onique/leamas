// Package dupcode verifies the behavior-preserving V4 internal-finding extraction.
package dupcode

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestV4InternalFindingExtraction_CharacterizedPublicOutput freezes the
// pre-extraction public projection. The representative chain set exercises
// fingerprint-first finding order, N-way merge, token-position deduplication,
// occurrence sorting, and maximum LineCount selection.
func TestV4InternalFindingExtraction_CharacterizedPublicOutput(t *testing.T) {
	chains := characterizationChains()

	gotBytes, err := json.Marshal(v4FindingsFromChains(chains))
	if err != nil {
		t.Fatalf("marshal characterized public output: %v", err)
	}
	const want = `[` +
		`{"Fingerprint":"11977938dde0d43619ed1c04c2a9da2c78b0848c...",` +
		`"StableFingerprint":"11977938dde0d43619ed1c04c2a9da2c78b0848c045d3f5bdfc06f29f9ac3f4c",` +
		`"SeedFingerprint":"","Occurrences":[` +
		`{"Path":"d.go","StartLine":20,"EndLine":21},` +
		`{"Path":"e.go","StartLine":22,"EndLine":23}],"TokenCount":8,"LineCount":2},` +
		`{"Fingerprint":"15884a61ef01cb8f595249b7e0e3a03a585ec264...",` +
		`"StableFingerprint":"15884a61ef01cb8f595249b7e0e3a03a585ec26455826ea63d4e83fef80f8996",` +
		`"SeedFingerprint":"","Occurrences":[` +
		`{"Path":"a.go","StartLine":2,"EndLine":4},` +
		`{"Path":"b.go","StartLine":7,"EndLine":8},` +
		`{"Path":"c.go","StartLine":12,"EndLine":14}],"TokenCount":10,"LineCount":3}]`
	if got := string(gotBytes); got != want {
		t.Fatalf("characterized public output changed:\n got: %s\nwant: %s", got, want)
	}
}

func TestV4InternalFindingExtraction_CharacterizedEdgeOutputs(t *testing.T) {
	cases := []struct {
		name   string
		chains []cloneChain
		want   string
	}{
		{name: "nil input", chains: nil, want: "null"},
		{name: "empty input", chains: []cloneChain{}, want: "null"},
		{name: "all chains filtered", chains: []cloneChain{{}}, want: "[]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotBytes, err := json.Marshal(v4FindingsFromChains(tc.chains))
			if err != nil {
				t.Fatalf("marshal characterized edge output: %v", err)
			}
			if got := string(gotBytes); got != tc.want {
				t.Fatalf("characterized edge output = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestV4InternalFindingExtraction_PublicProjectionUsesSharedSeam(t *testing.T) {
	chains := characterizationChains()
	internal := v4InternalFindingsFromChains(chains)
	actual := v4FindingsFromChains(chains)
	if len(actual) != len(internal) {
		t.Fatalf("public finding count = %d, internal finding count = %d", len(actual), len(internal))
	}
	for i, finding := range internal {
		want := coalescedFinding{
			Fingerprint:       truncateFingerprint(finding.StableFingerprint),
			StableFingerprint: finding.StableFingerprint,
			SeedFingerprint:   "",
			Occurrences:       convertOccurrences(finding.Occurrences),
			TokenCount:        finding.TokenCount,
			LineCount:         finding.LineCount,
		}
		if !reflect.DeepEqual(actual[i], want) {
			t.Fatalf("public finding[%d] = %+v, shared-seam projection want %+v", i, actual[i], want)
		}
	}
}

func characterizationChains() []cloneChain {
	return []cloneChain{
		characterizationChain(
			"body-alpha", 10, 3,
			rawWindow{Path: "c.go", StartPos: 50, EndPos: 59, StartLine: 12, EndLine: 14},
			rawWindow{Path: "a.go", StartPos: 10, EndPos: 19, StartLine: 2, EndLine: 4},
		),
		characterizationChain(
			"body-beta", 8, 2,
			rawWindow{Path: "e.go", StartPos: 70, EndPos: 77, StartLine: 22, EndLine: 23},
			rawWindow{Path: "d.go", StartPos: 60, EndPos: 67, StartLine: 20, EndLine: 21},
		),
		characterizationChain(
			"body-alpha", 10, 2,
			rawWindow{Path: "b.go", StartPos: 30, EndPos: 39, StartLine: 7, EndLine: 8},
			rawWindow{Path: "a.go", StartPos: 10, EndPos: 19, StartLine: 2, EndLine: 3},
		),
	}
}

func characterizationChain(
	contentHash string,
	tokenSpan int,
	lineSpan int,
	left rawWindow,
	right rawWindow,
) cloneChain {
	return cloneChain{
		Matches:     []seedMatch{{Left: left, Right: right}},
		TokenSpan:   tokenSpan,
		LineSpan:    lineSpan,
		ContentHash: contentHash,
	}
}
