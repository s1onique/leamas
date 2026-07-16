// Package dupcode provides exact semantic contract tests for the V4 algorithm.
//
// This file groups the body-separation contracts:
//   - TwoIndependentBodies: independent bodies do not collapse into one
//   - NoShadowSubFindings:   maximal findings are not shadowed by sub-findings
//
// Both tests currently FAIL because production V4 does not yet implement the
// exact semantics. They serve as regression detection.
package dupcode

import (
	"path/filepath"
	"testing"
)

// TestV4ExactSemantics_TwoIndependentBodies verifies the EXACT contract:
// - Exactly 2 findings in the complete production result set
// - Each finding has exactly 2 occurrences
// - Each finding has distinct body identity
func TestV4ExactSemantics_TwoIndependentBodies(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "ind_a.go")
	fileB := filepath.Join(tmpDir, "ind_b.go")

	cloneCounter = 0
	clone1 := generateForLoopClone("a", 1)
	clone2 := generateWhileLoopClone("a", 2)
	clone1B := generateForLoopClone("b", 1)
	clone2B := generateWhileLoopClone("b", 2)
	contentA := clone1 + "\n" + clone2
	contentB := clone1B + "\n" + clone2B

	writeTestFile(t, fileA, contentA)
	writeTestFile(t, fileB, contentB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	// EXACT contract: exactly 2 findings in the complete production result set
	if len(findings) != 2 {
		t.Fatalf("EXACT CONTRACT FAIL: expected exactly 2 findings, got %d", len(findings))
	}

	// Count distinct cross-file fingerprints
	fingerprints := make(map[string]bool)
	for _, f := range findings {
		uniqueFiles := make(map[string]bool)
		for _, occ := range f.Occurrences {
			uniqueFiles[filepath.Base(occ.Path)] = true
		}
		if len(uniqueFiles) >= 2 {
			fingerprints[f.Fingerprint] = true
		}
	}

	// EXACT contract: exactly 2 distinct cross-file fingerprints
	if len(fingerprints) != 2 {
		t.Errorf("EXACT CONTRACT FAIL: expected exactly 2 distinct cross-file findings, got %d", len(fingerprints))
	}

	// Each finding should have 2 occurrences (one per file)
	for i, f := range findings {
		if len(f.Occurrences) != 2 {
			t.Errorf("EXACT CONTRACT FAIL: finding[%d]: expected exactly 2 occurrences, got %d", i, len(f.Occurrences))
		}
	}

	// EXACT contract: verify each finding has distinct fingerprint
	fpList := make([]string, 0, len(findings))
	for _, f := range findings {
		fpList = append(fpList, f.Fingerprint)
	}
	if len(fpList) == 2 && fpList[0] == fpList[1] {
		t.Errorf("EXACT CONTRACT FAIL: expected 2 distinct fingerprints, got identical: %s", fpList[0])
	}
}

// TestV4ExactSemantics_NoShadowSubFindings verifies the EXACT contract:
// - Exactly 1 finding (no additional findings emitted)
// - The sole finding is above threshold
// Note: Exact proof that the sole result equals the maximal fixture body
// is owned by ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01.
func TestV4ExactSemantics_NoShadowSubFindings(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "shadow_a.go")
	fileB := filepath.Join(tmpDir, "shadow_b.go")

	cloneCounter = 0
	cloneA := generateLargeCloneBody("shadow_a")
	cloneB := generateLargeCloneBody("shadow_b")

	writeTestFile(t, fileA, cloneA)
	writeTestFile(t, fileB, cloneB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	// EXACT contract: exactly 1 finding (no additional findings emitted)
	if len(findings) != 1 {
		t.Fatalf("EXACT CONTRACT FAIL: expected exactly 1 above-threshold cross-file finding, got %d", len(findings))
	}

	// Verify the finding is cross-file
	counts := countOccurrencesByFile(findings[0].Occurrences)
	if len(counts) != 2 {
		t.Errorf("EXACT CONTRACT FAIL: expected 2 files, got %d", len(counts))
	}

	// EXACT contract: the finding must have token count significantly above threshold
	if findings[0].TokenCount <= 400 {
		t.Errorf("EXACT CONTRACT FAIL: finding TokenCount %d should be > threshold 400", findings[0].TokenCount)
	}
}