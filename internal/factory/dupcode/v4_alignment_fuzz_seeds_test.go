// Package dupcode defines and validates the complete named fuzz seed
// inventory derived from the deterministic semantic corpus.
package dupcode

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const v4PersistentFuzzSeedHash = "3fc61698be2e2294"

type v4NamedFuzzSeed struct {
	Name  string
	Value []byte
}

func v4AlignmentFuzzSeeds() []v4NamedFuzzSeed {
	corpus := v4BuildAlignmentCorpus()
	seeds := make([]v4NamedFuzzSeed, 0, len(corpus))
	for _, fixture := range corpus {
		seeds = append(seeds, v4NamedFuzzSeed{
			Name:  fixture.Name,
			Value: v4EncodeFuzzFixture(fixture),
		})
	}
	return seeds
}

func v4AsymmetricRegressionFuzzSeed() v4NamedFuzzSeed {
	for _, seed := range v4AlignmentFuzzSeeds() {
		if seed.Name == string(v4LeadingExtraRight) {
			return seed
		}
	}
	panic("LeadingExtraRight fuzz seed is not registered")
}

func TestV4Alignment_FuzzSeedInventory(t *testing.T) {
	seeds := v4AlignmentFuzzSeeds()
	if len(seeds) != len(requiredV4CorpusDimensions) {
		t.Fatalf("registered fuzz seed count=%d, want %d",
			len(seeds), len(requiredV4CorpusDimensions))
	}
	seenNames := make(map[string]bool)
	seenValues := make(map[string]string)
	for i, seed := range seeds {
		wantName := string(requiredV4CorpusDimensions[i])
		if seed.Name != wantName {
			t.Errorf("seed[%d] name=%q, want %q", i, seed.Name, wantName)
		}
		if seenNames[seed.Name] {
			t.Errorf("duplicate registered seed name %q", seed.Name)
		}
		seenNames[seed.Name] = true
		if previous, ok := seenValues[string(seed.Value)]; ok {
			t.Errorf("seeds %q and %q have identical serialized values without a documented alias",
				previous, seed.Name)
		}
		seenValues[string(seed.Value)] = seed.Name
	}
}

func TestV4Alignment_FuzzWireRoundTripCorpus(t *testing.T) {
	for _, fixture := range v4BuildAlignmentCorpus() {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			decoded := v4DecodeFuzzFixture(v4EncodeFuzzFixture(fixture))
			if !reflect.DeepEqual(decoded.Regions, fixture.Regions) {
				t.Fatalf("region round-trip drift\ngot:  %#v\nwant: %#v", decoded.Regions, fixture.Regions)
			}
			if !reflect.DeepEqual(decoded.RawWindows, fixture.RawWindows) {
				t.Fatalf("raw-window round-trip drift\ngot:  %#v\nwant: %#v", decoded.RawWindows, fixture.RawWindows)
			}
		})
	}
}

func TestV4Alignment_AsymmetricPersistentSeedContract(t *testing.T) {
	seed := v4AsymmetricRegressionFuzzSeed()
	fixture := v4DecodeFuzzFixture(seed.Value)
	if fixture.Regions[0].Path != "alpha.go" || fixture.Regions[1].Path != "beta.go" {
		t.Fatalf("persistent seed sides=(%q,%q), want alpha.go/beta.go",
			fixture.Regions[0].Path, fixture.Regions[1].Path)
	}
	left := v4WindowsForDeclaredRegion(fixture, fixture.LeftRegion)
	right := v4WindowsForDeclaredRegion(fixture, fixture.RightRegion)
	if v4SequencesPositionallyAligned(left, right) {
		t.Fatal("persistent asymmetric seed is positionally aligned; fallback is not activated")
	}

	analyses := v4BuildFixtureAnalyses(fixture)
	annotated, leftIndexes, rightIndexes := v4AnnotatedPairForFixture(
		t,
		fixture,
		analyses,
	)
	if regionsArePositionallyAligned(leftIndexes, rightIndexes, annotated) {
		t.Fatal("production alignment guard accepted persistent asymmetric seed")
	}
	v4AssertDifferentialResultsEqual(
		t,
		"persistent asymmetric seed",
		v4RunProductionCorpusFixture(fixture),
		v4RunOracleCorpusFixture(fixture),
	)
}

func TestV4Alignment_PersistentFuzzSeedFile(t *testing.T) {
	seed := v4AsymmetricRegressionFuzzSeed()
	want := []byte(fmt.Sprintf("go test fuzz v1\n[]byte(%q)\n", seed.Value))
	digest := sha256.Sum256(want)
	if got := fmt.Sprintf("%x", digest)[:16]; got != v4PersistentFuzzSeedHash {
		t.Fatalf("canonical seed hash=%s, want %s", got, v4PersistentFuzzSeedHash)
	}
	path := filepath.Join(
		"testdata", "fuzz", "FuzzV4RegionPairingEquivalentToAllPairs",
		v4PersistentFuzzSeedHash,
	)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persistent fuzz seed %s: %v", path, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("persistent fuzz seed %s does not encode LeadingExtraRight", path)
	}
}

func v4AnnotatedPairForFixture(
	t *testing.T,
	fixture v4CorpusFixture,
	analyses map[string]*v4FileAnalysis,
) ([]v4AnnotatedWindow, []int, []int) {
	t.Helper()
	windows := v4CanonicalRawWindows(fixture.RawWindows)
	annotated := make([]v4AnnotatedWindow, 0, len(windows))
	var leftIndexes, rightIndexes []int
	for _, window := range windows {
		a, ok := analyses[window.Path]
		if !ok {
			continue
		}
		owner, ok := a.windowFitsRegion(window.StartPos, window.EndPos)
		if !ok {
			continue
		}
		index := len(annotated)
		annotated = append(annotated, v4AnnotatedWindow{
			w: rawWindow{
				Path: window.Path, StartPos: window.StartPos, EndPos: window.EndPos,
				StartLine: window.StartLine, EndLine: window.EndLine,
			},
			region: owner,
		})
		if owner == fixture.LeftRegion {
			leftIndexes = append(leftIndexes, index)
		}
		if owner == fixture.RightRegion {
			rightIndexes = append(rightIndexes, index)
		}
	}
	if len(leftIndexes) == 0 || len(rightIndexes) == 0 {
		t.Fatalf("fuzz fixture pair has left=%d right=%d owned windows",
			len(leftIndexes), len(rightIndexes))
	}
	return annotated, leftIndexes, rightIndexes
}
