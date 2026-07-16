// Package dupcode provides exact geometry ordering contracts for V4.
package dupcode

import (
	"path/filepath"
	"reflect"
	"testing"
)

// TestV4ExactGeometry_CanonicalFindingOrdering compares the raw published
// finding sequence against independently frozen fingerprint-first keys. The
// expected slice is literal and is never sorted from production output.
func TestV4ExactGeometry_CanonicalFindingOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "ind_a.go")
	fileB := filepath.Join(tmpDir, "ind_b.go")

	cloneCounter = 0
	contentA := generateForLoopClone("a", 1) + "\n" + generateWhileLoopClone("a", 2)
	contentB := generateForLoopClone("b", 1) + "\n" + generateWhileLoopClone("b", 2)
	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	validateFrozenFindingOrder(t, wantIndependentBodyOrder)

	findings, err := CheckRepo(tmpDir, Config{MinLines: 40, MinTokens: 400})
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	comparePublishedFindingOrder(t, findings, tmpDir, wantIndependentBodyOrder)
}

// TestV4ExactGeometry_CanonicalOccurrenceOrdering verifies the exact
// Path, StartLine, EndLine order within the repeated-multiplicity finding.
// Equal paths are required to use both line fields as tie-breakers.
func TestV4ExactGeometry_CanonicalOccurrenceOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "repeat_a.go")
	fileB := filepath.Join(tmpDir, "repeat_b.go")

	cloneCounter = 0
	contentA := makeCloneFunc("OrderA1", 150) + makeCloneFunc("OrderA2", 150)
	contentB := makeCloneFunc("OrderB1", 150)
	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	findings, err := CheckRepo(tmpDir, Config{MinLines: 40, MinTokens: 400})
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	actual := make([]exactFindingGeometry, len(findings))
	for i, finding := range findings {
		actual[i] = projectFindingGeometry(t, finding, tmpDir)
	}
	want := exactFindingGeometry{
		TokenCount: wantMediumCloneTokenCount,
		Occurrences: []exactOccurrenceGeometry{
			{Path: "repeat_a.go", StartLine: 3, EndLine: 155},
			{Path: "repeat_a.go", StartLine: 157, EndLine: 309},
			{Path: "repeat_b.go", StartLine: 3, EndLine: 155},
		},
	}

	if len(actual) != 1 {
		t.Errorf("canonical occurrence ordering: finding cardinality %d, want 1", len(actual))
	}
	if len(actual) > 0 && !reflect.DeepEqual(actual[0], want) {
		t.Errorf("canonical occurrence ordering: finding[0] = %+v, want %+v", actual[0], want)
	}

	for findingIndex, finding := range actual {
		for occurrenceIndex := 1; occurrenceIndex < len(finding.Occurrences); occurrenceIndex++ {
			previous := finding.Occurrences[occurrenceIndex-1]
			current := finding.Occurrences[occurrenceIndex]
			if exactOccurrenceGeometryLess(current, previous) {
				t.Errorf("canonical occurrence ordering: finding[%d] occurrence[%d] = %+v precedes %+v",
					findingIndex, occurrenceIndex, current, previous)
			}
		}
	}
}

func exactOccurrenceGeometryLess(left, right exactOccurrenceGeometry) bool {
	if left.Path != right.Path {
		return left.Path < right.Path
	}
	if left.StartLine != right.StartLine {
		return left.StartLine < right.StartLine
	}
	return left.EndLine < right.EndLine
}
