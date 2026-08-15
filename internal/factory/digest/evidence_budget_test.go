// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_test.go is the regression matrix
// for the bounded evidence renderer introduced by
// ACT-LEAMAS-TARGETED-DIGEST-RECURSIVE-EVIDENCE-GUARD01.
//
// These tests cover the most-critical invariants:
//
//   - G1  Output self-exclusion (dirty and range modes)
//   - G9  Repeated-generation size stability (no amplification)
//
// Companion file evidence_budget_matrix_test.go covers the broader
// adversarial matrix (T1-T16).
package digest

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// maxBudgetBytes is the deterministic amplification budget used in
// the recursive stability test. It is intentionally generous: the
// test fails only when the digest explodes past the budget, NOT
// when it grows a small bounded metadata amount.
const maxBudgetBytes = 256 * 1024

func writeFilePath(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// seedRangeHistory makes a 2-commit history in `dir` so HEAD~1 is
// always resolvable. It commits an empty initial commit and then
// a second empty commit. Caller then commits the real change.
func seedRangeHistory(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), []byte(""), 0o644); err != nil {
		t.Fatalf("write .gitkeep: %v", err)
	}
	runGit(t, dir, "add", ".gitkeep")
	runGit(t, dir, "commit", "-m", "initial")
	runGit(t, dir, "commit", "--allow-empty", "-m", "anchor")
}

func hugeLine(n int) string {
	if n <= 0 {
		return ""
	}
	return string(bytes.Repeat([]byte("A"), n))
}

func identityHash(t *testing.T, fullPath string) string {
	t.Helper()
	f, err := os.Open(fullPath)
	if err != nil {
		t.Fatalf("open %s: %v", fullPath, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("sha256: %v", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func contractMarkerCount(body string) int {
	return strings.Count(body, "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3")
}

// ---------------------------------------------------------------------------
// G1: Output self-exclusion (dirty and range).
// ---------------------------------------------------------------------------

func TestEvidenceBudget_ExcludesOwnOutput_DirtyMode(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	out := filepath.Join(dir, "docs", "evidence", "targeted-digest.txt")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	d1, err := Generate(Options{RepoRoot: dir, Mode: ModeDirty, Output: out})
	if err != nil {
		t.Fatalf("generate D1: %v", err)
	}
	if err := os.WriteFile(out, []byte(d1), 0o644); err != nil {
		t.Fatalf("write D1: %v", err)
	}

	d2, err := Generate(Options{RepoRoot: dir, Mode: ModeDirty, Output: out})
	if err != nil {
		t.Fatalf("generate D2: %v", err)
	}

	if got := contractMarkerCount(d2); got > 1 {
		t.Fatalf("G1 violated: D2 contains %d contract markers; expected exactly 1", got)
	}
	if len(d2) > 2*maxBudgetBytes {
		t.Fatalf("G1 violated: D2=%d bytes exceeds 2x budget; suggests self-inclusion",
			len(d2))
	}
}

func TestEvidenceBudget_ExcludesOwnOutput_RangeMode(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	f := filepath.Join(dir, "real.txt")
	writeFilePath(t, f, "v1\n")
	runGit(t, dir, "add", "real.txt")
	runGit(t, dir, "commit", "-m", "v1")

	out := filepath.Join(dir, "docs", "evidence", "targeted-digest.txt")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	d1, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate D1: %v", err)
	}
	if err := os.WriteFile(out, []byte(d1), 0o644); err != nil {
		t.Fatalf("write D1: %v", err)
	}

	writeFilePath(t, f, "v2\n")
	runGit(t, dir, "add", "real.txt")
	runGit(t, dir, "add", "docs/evidence/targeted-digest.txt")
	runGit(t, dir, "commit", "-m", "v2 with digest")

	d2, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate D2: %v", err)
	}

	if got := contractMarkerCount(d2); got > 1 {
		t.Fatalf("G1 violated in range mode: D2 contains %d contract markers", got)
	}
	if len(d2) > 2*maxBudgetBytes {
		t.Fatalf("G1 violated in range mode: D2=%d bytes exceeds 2x budget",
			len(d2))
	}
	// F25 (CORRECTION06): the range-mode G1 diagnostic
	// assertion was previously inside an `if` that
	// searched for "=== targeted-digest.txt ===",
	// but the actual fixture path is the longer
	// "=== docs/evidence/targeted-digest.txt ===".
	// The condition never fired, making the
	// strengthening vacuous. F21 still passes via the
	// separate F21 integration test; here we assert
	// unconditionally against the actual escaped
	// path. The classifier selects BOUNDED_DERIVED_DIGEST
	// because the prior generation's body is recognized
	// as a digest artifact; the diagnostic must therefore
	// carry DERIVED_DIGEST_BODY_BOUNDED, not the generic
	// LARGE_FILE_EVIDENCE_BOUNDED.
	selfMarker := "=== docs/evidence/targeted-digest.txt ==="
	if !strings.Contains(d2, selfMarker) {
		t.Fatalf("F21 violated: range-mode G1 missing self-output marker %q in d2", selfMarker)
	}
	// In this G1 fixture the classifier selects
	// BOUNDED_SELF_OUTPUT (the path IS the canonical
	// output), so the diagnostic MUST be
	// SELF_OUTPUT_EXCLUDED, not the generic
	// LARGE_FILE_EVIDENCE_BOUNDED.
	if !strings.Contains(d2, "WarningCode: SELF_OUTPUT_EXCLUDED") {
		t.Fatalf("F21 violated: range-mode G1 dropped SELF_OUTPUT_EXCLUDED diagnostic")
	}
	if strings.Contains(d2, "WarningCode: LARGE_FILE_EVIDENCE_BOUNDED") {
		t.Fatalf("F21 violated: range-mode G1 reports generic LARGE_FILE_EVIDENCE_BOUNDED instead of SELF_OUTPUT_EXCLUDED")
	}
	if !strings.Contains(d2, "Classification: BOUNDED_SELF_OUTPUT") {
		t.Fatalf("F21 violated: self-output path misclassified; want BOUNDED_SELF_OUTPUT")
	}
}

// ---------------------------------------------------------------------------
// G9 — Core acceptance: repeated generation must NOT amplify.
// ---------------------------------------------------------------------------

func TestEvidenceBudget_RecursiveGenerations_AreStable(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	out := filepath.Join(dir, "docs", "evidence", "targeted-digest.txt")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	src := filepath.Join(dir, "src.go")
	writeFilePath(t, src, "package s\n")
	runGit(t, dir, "add", "src.go")
	runGit(t, dir, "commit", "-m", "init src")

	d1, err := Generate(Options{RepoRoot: dir, Mode: ModeDirty, Output: out})
	if err != nil {
		t.Fatalf("generate D1: %v", err)
	}
	if err := os.WriteFile(out, []byte(d1), 0o644); err != nil {
		t.Fatalf("write D1: %v", err)
	}

	sizes := []int{len(d1)}
	for i := 0; i < 3; i++ {
		d, err := Generate(Options{
			RepoRoot: dir, Mode: ModeDirty, Output: out,
		})
		if err != nil {
			t.Fatalf("generate D%d: %v", i+2, err)
		}
		if err := os.WriteFile(out, []byte(d), 0o644); err != nil {
			t.Fatalf("write D%d: %v", i+2, err)
		}
		sizes = append(sizes, len(d))
	}

	for i, sz := range sizes {
		if sz > maxBudgetBytes {
			t.Fatalf("G9 violated: D%d=%d bytes exceeds %d budget",
				i+1, sz, maxBudgetBytes)
		}
	}

	maxJitter := 32 * 1024
	for i := 1; i < len(sizes); i++ {
		diff := sizes[i] - sizes[i-1]
		if diff < 0 {
			diff = -diff
		}
		if diff > maxJitter {
			t.Fatalf("G9 violated: |D%d - D%d|=%d exceeds %d jitter; suggests amplification",
				i+1, i, diff, maxJitter)
		}
	}
}
