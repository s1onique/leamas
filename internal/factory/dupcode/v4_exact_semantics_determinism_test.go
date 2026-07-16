// Package dupcode provides exact semantic contract tests for the V4 algorithm.
//
// This file groups the determinism contract: repeated execution produces
// identical output (fingerprints, occurrences, ordering).
//
// The test currently PASSES because run-to-run stability is the only contract
// being asserted; production V4 is internally consistent even though it
// over-emits findings.
package dupcode

import (
	"path/filepath"
	"testing"
)

// TestV4ExactSemantics_Determinism verifies the EXACT contract:
// - Repeated execution produces identical output
func TestV4ExactSemantics_Determinism(t *testing.T) {
	tmpDir := t.TempDir()
	fileA := filepath.Join(tmpDir, "det_a.go")
	fileB := filepath.Join(tmpDir, "det_b.go")

	cloneCounter = 0
	cloneA := generateLargeCloneBody("det_a")
	cloneB := generateLargeCloneBody("det_b")

	writeTestFile(t, fileA, cloneA)
	writeTestFile(t, fileB, cloneB)
	verifyFixturesTypeCheck(t, fileA, fileB)

	cfg := Config{MinLines: 40, MinTokens: 400}

	// Run multiple times
	var firstResult []Finding
	for i := 0; i < 5; i++ {
		findings, err := CheckRepo(tmpDir, cfg)
		if err != nil {
			t.Fatalf("CheckRepo run %d failed: %v", i, err)
		}

		if i == 0 {
			firstResult = findings
			continue
		}

		// Compare finding counts
		if len(findings) != len(firstResult) {
			t.Errorf("run %d: expected %d findings, got %d", i, len(firstResult), len(findings))
		}

		// Compare fingerprints (stable ordering)
		for j := range findings {
			if findings[j].StableFingerprint != firstResult[j].StableFingerprint {
				t.Errorf("run %d: fingerprint mismatch at index %d", i, j)
			}
			if len(findings[j].Occurrences) != len(firstResult[j].Occurrences) {
				t.Errorf("run %d: occurrence count mismatch at index %d", i, j)
			}
			for k := range findings[j].Occurrences {
				if findings[j].Occurrences[k] != firstResult[j].Occurrences[k] {
					t.Errorf("run %d: occurrence mismatch at [%d][%d]", i, j, k)
				}
			}
		}
	}
}