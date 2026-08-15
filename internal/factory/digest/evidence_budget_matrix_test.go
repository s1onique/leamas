// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_matrix_test.go is the broader
// adversarial matrix for the bounded evidence renderer introduced
// by ACT-LEAMAS-TARGETED-DIGEST-RECURSIVE-EVIDENCE-GUARD01.
//
// The companion file evidence_budget_test.go covers the critical
// G1/G9 invariants. This file covers:
//
//   - G2 / G5 / G6 / G7: digest recognition + recursion diagnostic
//   - G3 / G4:           large file / huge line boundedness
//   - T7 / T8:           unrelated large + huge line
//   - T10:               many large files
//   - T12 / T13:         digest-like filename / terminology
//   - T16:               committed-range historical artifact
//   - Unit-level classifier tests.
package digest

import (
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// G2 + G5 + G6: Recognised targeted-digest content signature must
// be bounded AND emit an explicit recursion diagnostic AND keep
// identity (path + sha256) visible.
// ---------------------------------------------------------------------------

func TestEvidenceBudget_RecognizedTargetedDigest_IsBounded(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Commit 1: small real change.
	small := filepath.Join(dir, "real-change.txt")
	writeFilePath(t, small, "real change\n")
	runGit(t, dir, "add", "real-change.txt")
	runGit(t, dir, "commit", "-m", "real change")

	// Commit 2 (HEAD): also adds the previous-digest artifact.
	prev := filepath.Join(dir, "docs", "evidence", "previous-targeted-digest.txt")
	header := "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n" +
		"LEAMAS_VERSION: 0.1.0\n" +
		"LEAMAS_COMMIT: deadbeefdeadbeefdeadbeefdeadbeefdeadbeef\n"
	var pad strings.Builder
	for i := 0; i < 20_000; i++ {
		pad.WriteString("+ some additive line of plausible digest content\n")
	}
	writeFilePath(t, prev, header+"\n# Targeted digest\n\n"+pad.String())
	runGit(t, dir, "add", "docs/evidence/previous-targeted-digest.txt")
	runGit(t, dir, "commit", "-m", "previous targeted digest committed")

	out := filepath.Join(dir, "next-targeted-digest.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if strings.Contains(d, pad.String()) {
		t.Fatalf("G2 violated: full body of recognised targeted digest was embedded")
	}
	if !strings.Contains(d, "DIGEST_RECURSION") &&
		!strings.Contains(d, "DERIVED_DIGEST_BODY_BOUNDED") {
		t.Fatalf("G5 violated: no derived-digest diagnostic in bounded digest")
	}
	if !strings.Contains(d, "previous-targeted-digest.txt") {
		t.Fatalf("G6 violated: bounded digest dropped the artifact path")
	}
	h := identityHash(t,
		filepath.Join(dir, "docs/evidence/previous-targeted-digest.txt"))
	if !strings.Contains(d, h) {
		t.Fatalf("G6 violated: identity hash %s not present in bounded digest", h)
	}
}

// ---------------------------------------------------------------------------
// G3: An ordinary multi-megabyte text file (no digest markers) must
//     still be bounded.
// ---------------------------------------------------------------------------

func TestEvidenceBudget_LargeOrdinaryTextFile_IsBounded(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	big := filepath.Join(dir, "big.txt")
	var sb strings.Builder
	// Need > 1 MiB so the bounded gate fires.
	for i := 0; i < 30_000; i++ {
		sb.WriteString("this is a totally ordinary line of text used only for size, ")
		sb.WriteString("it must not blow up the digest output\n")
	}
	writeFilePath(t, big, sb.String())
	runGit(t, dir, "add", "big.txt")
	runGit(t, dir, "commit", "-m", "big ordinary text")

	out := filepath.Join(dir, "next-targeted-digest.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(d) > maxBudgetBytes {
		t.Fatalf("G3 violated: digest=%d bytes exceeds %d byte budget",
			len(d), maxBudgetBytes)
	}
	if !strings.Contains(d, "LARGE_FILE_EVIDENCE_BOUNDED") &&
		!strings.Contains(d, "Classification: BOUNDED_BODY") {
		t.Fatalf("G3 violated: no bounded-evidence marker present")
	}
}

// ---------------------------------------------------------------------------
// G4: One multi-megabyte physical line must be bounded.
// ---------------------------------------------------------------------------

func TestEvidenceBudget_HugeSingleLine_IsBounded(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	big := filepath.Join(dir, "huge-line.txt")
	content := hugeLine(2*1024*1024) + "\n"
	writeFilePath(t, big, content)
	runGit(t, dir, "add", "huge-line.txt")
	runGit(t, dir, "commit", "-m", "huge single line")

	out := filepath.Join(dir, "next-targeted-digest.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if strings.Contains(d, content[:1024]) {
		t.Fatalf("G4 violated: huge-line content was embedded into digest")
	}
	if len(d) > maxBudgetBytes {
		t.Fatalf("G4 violated: digest=%d bytes exceeds %d byte budget",
			len(d), maxBudgetBytes)
	}
}

// ---------------------------------------------------------------------------
// G7: A small ordinary source change must remain fully rendered.
// ---------------------------------------------------------------------------

func TestEvidenceBudget_NormalSmallSource_RemainsFull(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	f := filepath.Join(dir, "a.go")
	writeFilePath(t, f, "package a\n")
	runGit(t, dir, "add", "a.go")
	runGit(t, dir, "commit", "-m", "init")

	writeFilePath(t, f, "package a\n\n// hello\n")

	out := filepath.Join(dir, "next-targeted-digest.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeDirty, Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if !strings.Contains(d, "package a") {
		t.Fatalf("G7 violated: ordinary source diff was lost")
	}
	if !strings.Contains(d, "// hello") {
		t.Fatalf("G7 violated: ordinary source diff body was lost")
	}
}
