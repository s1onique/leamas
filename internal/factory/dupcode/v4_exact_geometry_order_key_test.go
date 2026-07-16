// Package dupcode freezes the exact V4 published finding-order key.
package dupcode

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type exactFindingOrderKey struct {
	StableFingerprint string
	TokenCount        int
	LineCount         int
	Occurrences       []exactOccurrenceGeometry
}

const (
	wantForLoopStableFingerprint   = "78b75750feff94c4f09d1b48e00fb737cb72e81d417b8fac6f3f1cd4ecabab43"
	wantWhileLoopStableFingerprint = "9c779aa5a1dff976e5c91dfcfd38c9e3b6aab17961d5de0f5dd7d9e61673098e"
)

// These literals were derived without reading CheckRepo findings. A standalone
// Go scanner audit normalized each 491-token fixture body, enumerated its 92
// 400-token seeds, and hashed the ordered seed stream with the V4 domain and
// adjacent-window tuple :0:0:399:399. The independently calculated content
// hashes were:
//
//   - addition body:    1ec5b6bf6c957ee225f3b576c836b627861b3e90e4215f85496cf23c3f0e4773
//   - subtraction body: bad6b66e77cf9288d534970eb0c2a891ee6452ea43398ef62c746f2d77492306
//
// Applying SHA-256 to "leamas-dupcode-v4:" plus each content hash produced
// the two stable fingerprints below. Fingerprint order, not source-line order,
// places the addition body first because 0x78 is less than 0x9c.
var wantIndependentBodyOrder = []exactFindingOrderKey{
	{
		StableFingerprint: wantForLoopStableFingerprint,
		TokenCount:        wantLoopCloneTokenCount,
		LineCount:         83,
		Occurrences: []exactOccurrenceGeometry{
			{Path: "ind_a.go", StartLine: 3, EndLine: 85},
			{Path: "ind_b.go", StartLine: 3, EndLine: 85},
		},
	},
	{
		StableFingerprint: wantWhileLoopStableFingerprint,
		TokenCount:        wantLoopCloneTokenCount,
		LineCount:         83,
		Occurrences: []exactOccurrenceGeometry{
			{Path: "ind_a.go", StartLine: 87, EndLine: 169},
			{Path: "ind_b.go", StartLine: 87, EndLine: 169},
		},
	},
}

func projectFindingOrderKey(t *testing.T, finding Finding, fixtureRoot string) exactFindingOrderKey {
	t.Helper()
	geometry := projectFindingGeometry(t, finding, fixtureRoot)
	return exactFindingOrderKey{
		StableFingerprint: finding.StableFingerprint,
		TokenCount:        finding.TokenCount,
		LineCount:         finding.LineCount,
		Occurrences:       geometry.Occurrences,
	}
}

// exactFindingOrderLess mirrors production precedence: StableFingerprint,
// TokenCount, LineCount, then the canonical occurrence sequence. The public
// ordering oracle intentionally projects only normative public occurrence
// geometry; internal token positions remain covered by the internal suite.
func exactFindingOrderLess(left, right exactFindingOrderKey) bool {
	if left.StableFingerprint != right.StableFingerprint {
		return left.StableFingerprint < right.StableFingerprint
	}
	if left.TokenCount != right.TokenCount {
		return left.TokenCount < right.TokenCount
	}
	if left.LineCount != right.LineCount {
		return left.LineCount < right.LineCount
	}
	return exactOccurrenceOrderKey(left.Occurrences) < exactOccurrenceOrderKey(right.Occurrences)
}

func exactOccurrenceOrderKey(occurrences []exactOccurrenceGeometry) string {
	canonical := canonicalizeOccurrences(occurrences)
	parts := make([]string, len(canonical))
	for i, occurrence := range canonical {
		parts[i] = fmt.Sprintf("%s:%d:%d", occurrence.Path, occurrence.StartLine, occurrence.EndLine)
	}
	return strings.Join(parts, "|")
}

func validateFrozenFindingOrder(t *testing.T, expected []exactFindingOrderKey) {
	t.Helper()
	if len(expected) != 2 {
		t.Fatalf("frozen independent-body key count = %d, want 2", len(expected))
	}
	seen := make(map[string]struct{}, len(expected))
	for i, key := range expected {
		if len(key.StableFingerprint) != 64 {
			t.Fatalf("expected key[%d] fingerprint length = %d, want 64", i, len(key.StableFingerprint))
		}
		decoded, err := hex.DecodeString(key.StableFingerprint)
		if err != nil || len(decoded) != 32 {
			t.Fatalf("expected key[%d] fingerprint %q is not 32-byte lowercase hex: %v",
				i, key.StableFingerprint, err)
		}
		if hex.EncodeToString(decoded) != key.StableFingerprint {
			t.Fatalf("expected key[%d] fingerprint %q is not canonical lowercase hex", i, key.StableFingerprint)
		}
		if _, duplicate := seen[key.StableFingerprint]; duplicate {
			t.Fatalf("expected key[%d] unexpectedly duplicates fingerprint %q", i, key.StableFingerprint)
		}
		seen[key.StableFingerprint] = struct{}{}
	}
	for i := 1; i < len(expected); i++ {
		if !exactFindingOrderLess(expected[i-1], expected[i]) {
			t.Fatalf("frozen keys are not strictly ordered at [%d,%d]: prev=%+v curr=%+v",
				i-1, i, expected[i-1], expected[i])
		}
	}
}

func comparePublishedFindingOrder(
	t *testing.T,
	findings []Finding,
	fixtureRoot string,
	expected []exactFindingOrderKey,
) {
	t.Helper()
	actual := make([]exactFindingOrderKey, len(findings))
	for i, finding := range findings {
		actual[i] = projectFindingOrderKey(t, finding, fixtureRoot)
	}

	if len(actual) != len(expected) {
		t.Errorf("canonical finding ordering: finding cardinality %d, want %d", len(actual), len(expected))
	}
	compareCount := len(actual)
	if len(expected) < compareCount {
		compareCount = len(expected)
	}
	for i := 0; i < compareCount; i++ {
		if !reflect.DeepEqual(actual[i], expected[i]) {
			t.Errorf("canonical finding ordering: key[%d] = %+v, want %+v", i, actual[i], expected[i])
		}
	}
}
