// Package dupcode provides exact semantic contract tests for the V4 algorithm.
// These tests verify exact behavioral contracts using the existing fixture
// generation pattern from v4_semantics_test.go.
//
// EXACT CONTRACT TESTS: These tests assert the exact behavioral contracts of V4.
// They will FAIL if production V4 does not implement the exact semantics.
// The tests serve as regression detection for production correctness.
//
// Note: Exact boundary equality assertions are tracked in ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01.
//
// Sibling files in this contract group:
//   - v4_exact_semantics_bodies_test.go       (TwoIndependentBodies, NoShadowSubFindings)
//   - v4_exact_semantics_determinism_test.go  (Determinism)
//   - v4_exact_semantics_ordering_test.go     (CanonicalOrdering)
package dupcode

import (
	"fmt"
	"path/filepath"
	"testing"
)

// countOccurrencesByFile counts occurrences per file in a finding.
func countOccurrencesByFile(occs []Occurrence) map[string]int {
	counts := make(map[string]int)
	for _, occ := range occs {
		counts[filepath.Base(occ.Path)]++
	}
	return counts
}

// TestV4ExactSemantics_OneMaximalClone verifies the EXACT contract:
// - Exactly 1 finding in the complete production result set
// - Exactly 2 occurrences in that finding
// - Valid occurrence geometry (path, start line, end line)
func TestV4ExactSemantics_OneMaximalClone(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "a.go")
	fileB := filepath.Join(tmpDir, "b.go")

	cloneCounter = 0
	cloneA := generateLargeCloneBody("a")
	cloneB := generateLargeCloneBody("b")

	writeTestFile(t, fileA, cloneA)
	writeTestFile(t, fileB, cloneB)
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

	f := findings[0]

	// EXACT contract: exactly 2 occurrences (one per file)
	if len(f.Occurrences) != 2 {
		t.Fatalf("EXACT CONTRACT FAIL: expected exactly 2 occurrences, got %d", len(f.Occurrences))
	}

	// EXACT contract: verify each occurrence has valid geometry
	for i, occ := range f.Occurrences {
		if occ.StartLine <= 0 {
			t.Errorf("EXACT CONTRACT FAIL: occurrence[%d] StartLine must be positive, got %d", i, occ.StartLine)
		}
		if occ.EndLine < occ.StartLine {
			t.Errorf("EXACT CONTRACT FAIL: occurrence[%d] EndLine %d < StartLine %d", i, occ.EndLine, occ.StartLine)
		}
	}

	// EXACT contract: verify distinct files
	counts := countOccurrencesByFile(f.Occurrences)
	if len(counts) != 2 {
		t.Fatalf("EXACT CONTRACT FAIL: expected occurrences in 2 distinct files, got %d", len(counts))
	}
}

// TestV4ExactSemantics_RepeatedMultiplicity verifies the EXACT contract:
// - Exactly 1 finding in the complete production result set
// - repeat_a.go has exactly 2 occurrences
// - repeat_b.go has exactly 1 occurrence
// - Total: 3 occurrences
// - Valid occurrence geometry for each occurrence
func TestV4ExactSemantics_RepeatedMultiplicity(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "repeat_a.go")
	fileB := filepath.Join(tmpDir, "repeat_b.go")

	cloneCounter = 0
	cloneA1 := makeCloneFunc("RepeatA1", 150)
	cloneA2 := makeCloneFunc("RepeatA2", 150)
	cloneB1 := makeCloneFunc("RepeatB1", 150)

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

	f := findings[0]

	// EXACT contract: exactly 3 total occurrences
	if len(f.Occurrences) != 3 {
		t.Fatalf("EXACT CONTRACT FAIL: expected exactly 3 total occurrences, got %d", len(f.Occurrences))
	}

	// EXACT contract: repeat_a.go has exactly 2 occurrences
	// EXACT contract: repeat_b.go has exactly 1 occurrence
	counts := countOccurrencesByFile(f.Occurrences)
	expectedRepeatA := 2
	expectedRepeatB := 1

	if count := counts["repeat_a.go"]; count != expectedRepeatA {
		t.Errorf("EXACT CONTRACT FAIL: expected repeat_a.go to have exactly %d occurrences, got %d", expectedRepeatA, count)
	}
	if count := counts["repeat_b.go"]; count != expectedRepeatB {
		t.Errorf("EXACT CONTRACT FAIL: expected repeat_b.go to have exactly %d occurrence, got %d", expectedRepeatB, count)
	}

	// EXACT contract: no duplicate occurrence records
	seen := make(map[string]bool)
	for _, occ := range f.Occurrences {
		key := fmt.Sprintf("%s:%d:%d", filepath.Base(occ.Path), occ.StartLine, occ.EndLine)
		if seen[key] {
			t.Errorf("EXACT CONTRACT FAIL: duplicate occurrence record: %s", key)
		}
		seen[key] = true
	}

	// EXACT contract: verify each occurrence has valid geometry
	for i, occ := range f.Occurrences {
		if occ.StartLine <= 0 {
			t.Errorf("EXACT CONTRACT FAIL: occurrence[%d] StartLine must be positive, got %d", i, occ.StartLine)
		}
		if occ.EndLine < occ.StartLine {
			t.Errorf("EXACT CONTRACT FAIL: occurrence[%d] EndLine %d < StartLine %d", i, occ.EndLine, occ.StartLine)
		}
	}
}

// TestV4ExactSemantics_NWayClone verifies the EXACT contract:
// - Exactly 1 finding in the complete production result set
// - Exactly 3 occurrences (one per file)
// - Valid occurrence geometry
func TestV4ExactSemantics_NWayClone(t *testing.T) {
	tmpDir := t.TempDir()
	files := []string{
		filepath.Join(tmpDir, "nw_a.go"),
		filepath.Join(tmpDir, "nw_b.go"),
		filepath.Join(tmpDir, "nw_c.go"),
	}

	cloneCounter = 0
	for i, f := range files {
		writeTestFile(t, f, generateLargeCloneBody([]string{"a", "b", "c"}[i]))
	}
	verifyFixturesTypeCheck(t, files...)

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	// EXACT contract: exactly 1 finding in the complete production result set
	if len(findings) != 1 {
		t.Fatalf("EXACT CONTRACT FAIL: expected exactly 1 finding, got %d", len(findings))
	}

	f := findings[0]

	// EXACT contract: exactly 3 occurrences (one per file)
	if len(f.Occurrences) != 3 {
		t.Fatalf("EXACT CONTRACT FAIL: expected exactly 3 occurrences, got %d", len(f.Occurrences))
	}

	// EXACT contract: distinct files = 3
	counts := countOccurrencesByFile(f.Occurrences)
	if len(counts) != 3 {
		t.Errorf("EXACT CONTRACT FAIL: expected occurrences in 3 distinct files, got %d: %v", len(counts), counts)
	}

	// Verify each file has exactly 1 occurrence
	for _, path := range []string{"nw_a.go", "nw_b.go", "nw_c.go"} {
		if count := counts[path]; count != 1 {
			t.Errorf("EXACT CONTRACT FAIL: expected exactly 1 occurrence in %s, got %d", path, count)
		}
	}

	// EXACT contract: verify each occurrence has valid geometry
	for i, occ := range f.Occurrences {
		if occ.StartLine <= 0 {
			t.Errorf("EXACT CONTRACT FAIL: occurrence[%d] StartLine must be positive, got %d", i, occ.StartLine)
		}
		if occ.EndLine < occ.StartLine {
			t.Errorf("EXACT CONTRACT FAIL: occurrence[%d] EndLine %d < StartLine %d", i, occ.EndLine, occ.StartLine)
		}
	}
}