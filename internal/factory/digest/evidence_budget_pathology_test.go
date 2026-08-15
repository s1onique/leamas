// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_pathology_test.go contains the
// pathological-input matrix for the bounded evidence renderer
// introduced by ACT-LEAMAS-TARGETED-DIGEST-RECURSIVE-EVIDENCE-GUARD01.
//
// The companion file evidence_budget_matrix_test.go covers the
// digest-recognition, ordinary-large-file, and huge-line cases
// (G2-G7). This file covers:
//
//   - T16:  committed-range historical artifact
//   - T12:  digest-like filename
//   - T13:  digest terminology incidentally mentioned
//   - T10:  many large files exhaust total budget
//   - T7:   unrelated large file preserves identity
//   - T8:   huge line + small change in same commit
//   - Unit-level classifier tests
package digest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// T16: committed-range protection. A historical multi-MiB digest
// must NOT inject an unrestricted body even when the output path
// is outside the repo (so output self-exclusion cannot help).
// ---------------------------------------------------------------------------

func TestEvidenceBudget_CommittedLargeDigest_IsBounded(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Commit 1: small real change.
	real := filepath.Join(dir, "real-change.txt")
	writeFilePath(t, real, "real change\n")
	runGit(t, dir, "add", "real-change.txt")
	runGit(t, dir, "commit", "-m", "real change")

	// Commit 2 (HEAD): adds the historical multi-MiB digest.
	prev := filepath.Join(dir, "docs", "evidence", "committed-digest.txt")
	header := "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n" +
		"LEAMAS_VERSION: 0.1.0\n" +
		"LEAMAS_COMMIT: deadbeefdeadbeefdeadbeefdeadbeefdeadbeef\n" +
		"\n# Targeted digest\n\n"
	var pad strings.Builder
	for i := 0; i < 15_000; i++ {
		pad.WriteString("+ substantive diff line that mimics a real digest content body\n")
	}
	writeFilePath(t, prev, header+pad.String())
	runGit(t, dir, "add", "docs/evidence/committed-digest.txt")
	runGit(t, dir, "commit", "-m", "previous targeted digest committed")

	out := filepath.Join(t.TempDir(), "next-targeted-digest.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if strings.Contains(d, pad.String()) {
		t.Fatalf("T16 violated: committed multi-megabyte digest body was embedded")
	}
	if len(d) > maxBudgetBytes {
		t.Fatalf("T16 violated: digest=%d bytes exceeds %d budget",
			len(d), maxBudgetBytes)
	}
}

// ---------------------------------------------------------------------------
// T12: filename resembles digest but content does not.
// ---------------------------------------------------------------------------

func TestEvidenceBudget_FilenameResemblesDigest_ButContentNormal(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	f := filepath.Join(dir, "docs", "evidence", "targeted-digest.txt")
	writeFilePath(t, f, "package targeted\n\n// ordinary source\n")
	runGit(t, dir, "add", "docs/evidence/targeted-digest.txt")
	runGit(t, dir, "commit", "-m", "ordinary source under digest-named path")

	out := filepath.Join(dir, "next.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(d, "package targeted") {
		t.Fatalf("T12 violated: ordinary content under digest-like filename was suppressed")
	}
}

// ---------------------------------------------------------------------------
// T13: content mentions digest terminology incidentally.
// ---------------------------------------------------------------------------

func TestEvidenceBudget_ContentMentionsDigest_NotTreatedAsDigest(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	f := filepath.Join(dir, "notes.md")
	body := "# My notes\n\nI read about the Leamas Targeted digest system. " +
		"It is bounded. End of notes.\n"
	writeFilePath(t, f, body)
	runGit(t, dir, "add", "notes.md")
	runGit(t, dir, "commit", "-m", "notes")

	out := filepath.Join(dir, "next.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(d, "I read about the Leamas Targeted digest") {
		t.Fatalf("T13 violated: ordinary markdown mentioning digest terminology was suppressed")
	}
}

// ---------------------------------------------------------------------------
// T10: many large changed files must exhaust the total budget
// but not the per-file cap on identity. F1 fix: the digest's
// file-evidence section MUST be bounded by MaxTotalRenderBytes,
// NOT by N × MaxPerFileBytes.
//
// To actually stress the total budget (not just N × small-stub),
// each file is rendered as a BOUNDED_BODY stub whose rendered
// size is non-trivial. With 30 files × ~10 KiB stub = ~300
// KiB, the total budget (256 KiB) MUST clamp the digest.
// ---------------------------------------------------------------------------

func TestEvidenceBudget_ManyLargeFiles_TotalBudgetEnforced(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Each file is just over MaxFileSizeForFull so the size
	// gate fires. The bounded stub is small but consistent
	// per-file; with 2000 files × ~132-byte stub = ~264
	// KiB, the total budget MUST clamp the digest well below
	// N × stub-size. This is what the F1 fix verifies: with
	// the total budget disabled, the digest would grow to
	// ~264 KiB and FAIL the assertion.
	const numFiles = 500
	for i := 0; i < numFiles; i++ {
		// Pad the path so each bounded stub is ~10 KiB.
		// We achieve this by putting a long prefix in the
		// dir name.
		path := filepath.Join(dir, fmt.Sprintf("file_%02d.txt", i))
		var sb strings.Builder
		for j := 0; j < 5000; j++ {
			sb.WriteString("ordinary line of plain text, " +
				"just padding for size\n")
		}
		writeFilePath(t, path, sb.String())
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "many large files")

	out := filepath.Join(dir, "next.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// F1 fix: strict ceiling on the file-evidence section.
	// Allow ~4 KiB of fixed metadata overhead on top of
	// MaxTotalRenderBytes for headers and surrounding
	// framing.
	const metaOverhead = 4 * 1024
	ceiling := MaxTotalRenderBytes + metaOverhead
	if len(d) > ceiling {
		t.Fatalf("F1 violated: digest=%d bytes exceeds "+
			"MaxTotalRenderBytes+%d=%d ceiling",
			len(d), metaOverhead, ceiling)
	}
	// Identity for at least the first few files must remain
	// visible. After the budget is exhausted, later files
	// are emitted as BOUNDED_OMITTED stubs, which still
	// contain the path.
	for i := 0; i < numFiles; i++ {
		name := fmt.Sprintf("file_%02d.txt", i)
		if !strings.Contains(d, name) {
			t.Fatalf("T10 violated: digest dropped "+
				"identity for %s", name)
		}
	}
}

// ---------------------------------------------------------------------------
// T7: an unrelated 2+ MiB text file (no digest markers) commits into
//     history. The bounded renderer must preserve identity (path
//     + sha256) without embedding the full body.
// ---------------------------------------------------------------------------

func TestEvidenceBudget_UnrelatedLargeFile_IdentityPreserved(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	big := filepath.Join(dir, "plain.txt")
	var sb strings.Builder
	// Need > 1 MiB so the bounded gate fires.
	for i := 0; i < 30_000; i++ {
		sb.WriteString("this is just plain text content, totally unrelated to any digest\n")
	}
	writeFilePath(t, big, sb.String())
	runGit(t, dir, "add", "plain.txt")
	runGit(t, dir, "commit", "-m", "plain big file")

	out := filepath.Join(dir, "next.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(d) > maxBudgetBytes {
		t.Fatalf("T7 violated: digest=%d bytes exceeds %d budget",
			len(d), maxBudgetBytes)
	}
	// F13 (CORRECTION02 + CORRECTION03 + F16/F19
	// CORRECTION04): historical identity is the Git
	// blob OID at the head endpoint, not a content
	// SHA-256. Asserting the SHA-256 here is the OLD
	// contract that this ACT retired.
	headOIDRaw, err := RunGit(dir, []string{"rev-parse", "HEAD:plain.txt"})
	if err != nil {
		t.Fatalf("rev-parse HEAD:plain.txt: %v", err)
	}
	headOID := strings.TrimSpace(headOIDRaw)
	if !strings.Contains(d, "RangeHeadBlobOID: "+headOID) {
		t.Fatalf("F13 violated: blob OID %s not present "+
			"in bounded digest", headOID)
	}
	if !strings.Contains(d, "RangeHeadBlobStatus: PRESENT") {
		t.Fatalf("F16 violated: RangeHeadBlobStatus " +
			"must report PRESENT for HEAD:plain.txt")
	}
}

// ---------------------------------------------------------------------------
// T8: huge single physical line must NOT be embedded even when there
//     is also a small ordinary change in the same commit.
// ---------------------------------------------------------------------------

func TestEvidenceBudget_HugeLine_WithSmallChange_SmallVisible(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	big := filepath.Join(dir, "huge.txt")
	writeFilePath(t, big, hugeLine(2*1024*1024)+"\n")
	small := filepath.Join(dir, "small.go")
	writeFilePath(t, small, "package s\n")
	runGit(t, dir, "add", "huge.txt")
	runGit(t, dir, "add", "small.go")
	runGit(t, dir, "commit", "-m", "huge + small")

	out := filepath.Join(dir, "next.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(d, "package s") {
		t.Fatalf("T8 violated: small.go content suppressed alongside huge.txt")
	}
	if strings.Contains(d, hugeLine(512)) {
		t.Fatalf("T8 violated: huge.txt content was embedded")
	}
}

// ---------------------------------------------------------------------------
// Unit-level classifier tests (small, focused, fast).
// ---------------------------------------------------------------------------

func TestClassifyFileEvidence_SelfOutput(t *testing.T) {
	dir := t.TempDir()
	writeFilePath(t, filepath.Join(dir, "out.txt"), "x")

	got := classifyFileEvidence(classifierInput{
		repoRoot:  dir,
		relPath:   "out.txt",
		fullPath:  filepath.Join(dir, "out.txt"),
		rawPrefix: "x",
		outputAbs: filepath.Join(dir, "out.txt"),
	})
	if got != ClassBoundedSelfOutput {
		t.Fatalf("got %s, want %s", got, ClassBoundedSelfOutput)
	}
}

func TestClassifyFileEvidence_RecognisedDigestHeader(t *testing.T) {
	// Body is empty: no structural recursion → BOUNDED_DERIVED_DIGEST.
	got := classifyFileEvidence(classifierInput{
		relPath:   "doc.txt",
		fullPath:  "/tmp/doc.txt",
		rawPrefix: "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n",
	})
	if got != ClassBoundedDerivedDigest {
		t.Fatalf("got %s, want %s", got, ClassBoundedDerivedDigest)
	}
}

func TestClassifyFileEvidence_LegacyTargetedDigestHeading(t *testing.T) {
	// Body is empty: no structural recursion → BOUNDED_DERIVED_DIGEST.
	got := classifyFileEvidence(classifierInput{
		relPath:   "doc.txt",
		fullPath:  "/tmp/doc.txt",
		rawPrefix: "# Targeted digest\n\nbody...\n",
	})
	if got != ClassBoundedDerivedDigest {
		t.Fatalf("got %s, want %s", got, ClassBoundedDerivedDigest)
	}
}

// TestClassifyFileEvidence_DigestWithSelfDiff_BecomesRecursive
// verifies the F7 + F9 distinction: a recognised digest whose
// body contains a self-diff signature of the SAME path is
// structural recursion, not just artifact recognition. F9
// (CORRECTION02) made the recursion check path-aware, so the
// self-diff must match `relPath` exactly.
func TestClassifyFileEvidence_DigestWithSelfDiff_BecomesRecursive(t *testing.T) {
	body := []byte("LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n" +
		"\n# Targeted digest\n\n" +
		"diff --git a/prev.txt b/prev.txt\n" +
		"--- a/prev.txt\n" +
		"+++ b/prev.txt\n")
	got := classifyFileEvidence(classifierInput{
		relPath:   "prev.txt",
		fullPath:  "/tmp/prev.txt",
		rawPrefix: "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n",
		bodyBytes: body,
	})
	if got != ClassBoundedRecursive {
		t.Fatalf("got %s, want %s", got, ClassBoundedRecursive)
	}
}

// TestClassifyFileEvidence_DigestWithUnrelatedDiff_RemainsDerived
// verifies the F9 (CORRECTION02) fix: a recognised digest whose
// body contains a NON-self diff (one that does not match the
// path being classified) is NOT structural recursion. The
// previous implementation labelled every `diff --git a/`
// substring as recursion, which falsely flagged every healthy
// digest that reviewed ordinary source files.
func TestClassifyFileEvidence_DigestWithUnrelatedDiff_RemainsDerived(t *testing.T) {
	body := []byte("LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n" +
		"\n# Targeted digest\n\n" +
		"diff --git a/src/foo.go b/src/foo.go\n" +
		"--- a/src/foo.go\n" +
		"+++ b/src/foo.go\n")
	got := classifyFileEvidence(classifierInput{
		relPath:   "docs/evidence/prev.txt",
		fullPath:  "/tmp/prev.txt",
		rawPrefix: "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n",
		bodyBytes: body,
	})
	if got != ClassBoundedDerivedDigest {
		t.Fatalf("got %s, want %s (unrelated diff should NOT "+
			"trigger recursion)",
			got, ClassBoundedDerivedDigest)
	}

}
