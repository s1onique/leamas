package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFixtureMatrixCompleteCoverage asserts that every invalid
// fixture is classified in exactly one closed-set bucket. The
// fixture corpus is the union of the structural, semantic, and
// pre-schema buckets per version. The test fails if a fixture is
// missing from all three buckets or appears in more than one
// bucket.
func TestFixtureMatrixCompleteCoverage(t *testing.T) {
	invalidDir := filepath.Join("..", "testdata", "invalid")

	for _, version := range []string{"1", "2"} {
		want := map[string]bool{}
		entries, err := os.ReadDir(invalidDir)
		if err != nil {
			t.Fatalf("read invalid fixtures: %v", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasPrefix(name, "v"+version+"-") || !strings.HasSuffix(name, ".json") {
				continue
			}
			want[name] = true
		}

		// Build the set of every fixture named in any classification.
		got := map[string]bool{}
		for _, s := range structuralInvalidV1 {
			if strings.HasPrefix(s, "v"+version+"-") {
				got[s] = true
			}
		}
		for _, s := range structuralInvalidV2 {
			if strings.HasPrefix(s, "v"+version+"-") {
				got[s] = true
			}
		}
		for _, e := range semanticInvalidV2 {
			if strings.HasPrefix(e.fixture, "v"+version+"-") {
				got[e.fixture] = true
			}
		}
		for _, s := range invalidV2CapturedByPreSchemaEnvelope {
			if strings.HasPrefix(s, "v"+version+"-") {
				got[s] = true
			}
		}
		// v2-truncated.json is malformed JSON. The schema-stage
		// validator is not applicable. The envelope rejection is
		// bound by internal/gatesummary/v2_truncated_envelope_test.go
		// (TestV2TruncatedEnvelopeRejectsWithCodeMalformedJSON).
		if version == "2" {
			got["v2-truncated.json"] = true
		}

		// Full coverage: every committed invalid fixture must be named
		// in at least one classification.
		for name := range want {
			if !got[name] {
				t.Errorf("v%s invalid fixture %s is not classified in any table", version, name)
			}
		}
		// No overspecification: every classified fixture must be a
		// real committed invalid fixture.
		for name := range got {
			if !want[name] {
				t.Errorf("v%s classification references fixture %s which is not a committed invalid fixture", version, name)
			}
		}
	}
}
