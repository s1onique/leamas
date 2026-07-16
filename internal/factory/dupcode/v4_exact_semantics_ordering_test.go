// Package dupcode provides exact semantic contract tests for the V4 algorithm.
//
// This file groups the canonical-ordering contract: occurrences are sorted
// by Path (ascending), with within-path ordering by StartLine and EndLine.
//
// The test currently FAILs against production V4 because production over-emits
// findings; the ordering assertions are about the survivors of any
// cardinality gate.
package dupcode

import (
	"path/filepath"
	"testing"
)

// groupOccurrencesByPath groups occurrences by file path.
func groupOccurrencesByPath(occs []Occurrence) map[string][]Occurrence {
	result := make(map[string][]Occurrence)
	for _, occ := range occs {
		result[filepath.Base(occ.Path)] = append(result[filepath.Base(occ.Path)], occ)
	}
	return result
}

// TestV4ExactSemantics_CanonicalOrdering verifies the EXACT contract:
// - Exactly 1 finding in the complete production result set
// - Occurrences ordered by: Path ≥ StartLine ≥ EndLine
// - Uses repeated-multiplicity fixture to test within-file ordering
func TestV4ExactSemantics_CanonicalOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "repeat_a.go")
	fileB := filepath.Join(tmpDir, "repeat_b.go")

	cloneCounter = 0
	cloneA1 := makeCloneFunc("OrderA1", 150)
	cloneA2 := makeCloneFunc("OrderA2", 150)
	cloneB1 := makeCloneFunc("OrderB1", 150)

	contentA := cloneA1 + cloneA2
	contentB := cloneB1

	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	// EXACT contract: exactly 1 finding in the complete production result set
	if len(findings) != 1 {
		t.Fatalf("EXACT CONTRACT FAIL: expected exactly 1 finding, got %d", len(findings))
	}

	occs := findings[0].Occurrences

	// EXACT contract: Path must be nonstrictly ascending (equal paths allowed)
	for i := 1; i < len(occs); i++ {
		prev := occs[i-1]
		curr := occs[i]
		if curr.Path < prev.Path {
			t.Errorf("EXACT CONTRACT FAIL: occurrence[%d] path %q should be >= previous %q", i, filepath.Base(curr.Path), filepath.Base(prev.Path))
		}
	}

	// EXACT contract: within same path, StartLine must be nondecreasing
	fileOccs := groupOccurrencesByPath(occs)
	for path, occList := range fileOccs {
		for i := 1; i < len(occList); i++ {
			prev := occList[i-1]
			curr := occList[i]
			if curr.StartLine < prev.StartLine {
				t.Errorf("EXACT CONTRACT FAIL: within %s, occurrence[%d] start line %d should be >= previous %d", path, i, curr.StartLine, prev.StartLine)
			}
			if curr.StartLine == prev.StartLine && curr.EndLine < prev.EndLine {
				t.Errorf("EXACT CONTRACT FAIL: within %s, occurrence[%d] end line %d should be >= previous %d", path, i, curr.EndLine, prev.EndLine)
			}
		}
	}

	// EXACT contract: repeat_a.go should have 2 occurrences, repeat_b.go should have 1
	counts := countOccurrencesByFile(occs)
	if count := counts["repeat_a.go"]; count != 2 {
		t.Errorf("EXACT CONTRACT FAIL: expected 2 occurrences in repeat_a.go, got %d", count)
	}
	if count := counts["repeat_b.go"]; count != 1 {
		t.Errorf("EXACT CONTRACT FAIL: expected 1 occurrence in repeat_b.go, got %d", count)
	}
}
