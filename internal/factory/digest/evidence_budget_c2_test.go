// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_c2_test.go contains the
// CORRECTION02 regression tests (F10/F11/F12: terminal
// truncation record, per-file marker reservation, mid-token
// termination).
package digest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// TestEvidenceBudget_NoMidTokenTruncation verifies the F12
// (CORRECTION02) fix: when the total budget is exhausted
// mid-block, the partial block is dropped rather than
// truncated mid-token.
func TestEvidenceBudget_NoMidTokenTruncation(t *testing.T) {
	// Epsilon: mutate MaxTotalRenderBytes temporarily via
	// constructing a writer with a tiny cap to force
	// truncation mid-block.
	// F12 (CORRECTION02): a total-cap exhaustion
	// mid-block must DROP the block, not truncate it.
	// Set up a writer with NO per-file or tail
	// reservation so the total-cap check in
	// appendString is the one being exercised.
	bw := newBoundedWriter(50, 1000)
	bw.beginFile()
	appendBlock(bw, "prefix-1")
	// 1000-byte block: with 50 total cap and ~8
	// already used, F12 must drop rather than
	// truncate. After this call the writer MUST be
	// exhausted AND the output must contain no "x"
	// characters (the block is dropped entirely).
	appendBlock(bw, "%s", strings.Repeat("x", 1000))
	if !bw.Exhausted() {
		t.Fatalf("F12 setup: writer should be exhausted "+
			"after F12 drop; got used=%d", bw.used)
	}
	out := bw.String()
	// A truncated block would contain 42 chars of
	// "x"; a dropped block contains zero. Check
	// for the smallest signature of truncation.
	if strings.Contains(out, strings.Repeat("x", 20)) {
		t.Fatalf("F12 violated: partial block in output: %q", out)
	}
}

// TestEvidenceBudget_TotalExhaustion_EmitsOmissionRecord
// verifies the F10 (CORRECTION02) fix: when the total budget
// is exhausted by many files, the renderer emits ONE terminal
// record listing omitted paths, instead of silently dropping
// individual BOUNDED_OMITTED stubs.
func TestEvidenceBudget_TotalExhaustion_EmitsOmissionRecord(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Enough files to definitely exhaust the 256 KiB budget.
	// Each base stub is ~150 bytes, so 1500 files saturate
	// the total cap.
	for i := 0; i < 1500; i++ {
		path := filepath.Join(dir, fmt.Sprintf("file_%05d.txt", i))
		var sb strings.Builder
		for j := 0; j < 20000; j++ {
			sb.WriteString("ordinary line of plain text, " +
				"just padding for size\n")
		}
		writeFilePath(t, path, sb.String())
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "many large files")

	out := filepath.Join(t.TempDir(), "next.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if !strings.Contains(d, "EVIDENCE_TRUNCATED=true") {
		t.Fatalf("F10 violated: digest did not emit "+
			"terminal truncation record. Excerpt:\n%s",
			"## Diffs")
	}
	if !strings.Contains(d, "omitted_files=") {
		t.Fatalf("F10 violated: missing omitted_files count")
	}
	if !strings.Contains(d, "reason=TOTAL_RENDER_BUDGET") {
		t.Fatalf("F10 violated: missing reason field")
	}
}

// TestEvidenceBudget_PerFileMarker_EmittedEvenAtCap
// verifies the F11 (CORRECTION02) fix: when the per-file cap
// is exactly hit, the per-file truncation marker must still
// be emitted (the cap reserves space for the marker).
func TestEvidenceBudget_PerFileMarker_EmittedEvenAtCap(t *testing.T) {
	bw := newBoundedWriter(MaxTotalRenderBytes, MaxPerFileBytes)
	bw.reservePerFileMarker(perFileMarkerBudget)
	bw.beginFile()
	body := strings.Repeat("y", 100*1024)
	n, perFileCapped := bw.appendFileString(body)
	if !perFileCapped {
		t.Fatalf("F11 setup: perFileCapped should be true, "+
			"n=%d, perFile=%d", n, bw.perFile)
	}
	bw.markPerFileMarkerReserved()
	n = appendBlock(bw, "\n[per-file body cap hit: %d bytes]\n",
		MaxPerFileBytes)
	if n == 0 {
		t.Fatalf("F11 violated: per-file truncation marker " +
			"was dropped after perFileCapped")
	}
}

// TestEvidenceBudget_F13_DemotedPath_EmitsHistoricalBlobOID
// verifies the F13 (CORRECTION02 + CORRECTION03) fix: when a
// range-mode file is demoted to BOUNDED_BODY because its
// historical blob exceeds the per-file cap, the rendered
// stub records the actual Git blob OID for the historical
// endpoint, NOT the working-tree file's identity. The
// previous implementation reported the SHA-256 of the
// current working-tree file (which was empty for a deleted
// file OR the wrong artifact for a replaced file).
//
// F13 also requires CORRECTION03's colon-key fix in
// rangeBlobOIDsBatch retrieval.
func TestEvidenceBudget_F13_DemotedPath_EmitsHistoricalBlobOID(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Build a large file, commit, then modify it slightly
	// so the historical blob is huge (> MaxPerFileBytes)
	// and the worktree file is also large.
	big := filepath.Join(dir, "big.txt")
	var sb strings.Builder
	for i := 0; i < 20_000; i++ {
		sb.WriteString("big content padding line for size\n")
	}
	writeFilePath(t, big, sb.String())
	runGit(t, dir, "add", "big.txt")
	runGit(t, dir, "commit", "-m", "add big")

	// Move to next commit so the blob is "historical".
	runGit(t, dir, "commit", "--allow-empty", "-m", "move along")

	out := filepath.Join(t.TempDir(), "next.txt")
	// Range covers the commit that added big.txt so the
	// changeset manifest actually lists big.txt as
	// changed. RangeBaseBlobOID should report the blob
	// OID at the base (BEFORE that change), which is the
	// empty manifest at HEAD~2. The blob at HEAD~1 is
	// the large 400 KB blob.
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~2..HEAD~1", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// The big.txt blob at HEAD~1 is the historic one
	// (the large 400 KB blob we just added). The HEAD
	// endpoint of the range IS HEAD~1 (where the blob
	// exists), so RangeHeadBlobOID must report HEAD~1
	// blob OID. The HEAD~2 endpoint predates the file
	// (RangeBaseBlobOID is empty / non-existent blob).
	expectedHeadOID, err := RunGit(dir, []string{
		"rev-parse", "HEAD~1:big.txt"})
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	expectedHeadOID = strings.TrimSpace(expectedHeadOID)

	wantSub := "RangeHeadBlobOID: " + expectedHeadOID
	if !strings.Contains(d, wantSub) {
		t.Fatalf("F13 violated: digest does not contain "+
			"the historical blob OID for HEAD~1:big.txt. "+
			"Expected substring %q in digest.", wantSub)
	}
}

// TestEvidenceBudget_F14_ThreeDotResolvesToMergeBase
// verifies the F14 (CORRECTION03) semantic fix: a `A...B`
// range classifies against the merge-base of A and B, not
// against A and B themselves. The previous implementation
// failed to demote files whose merge-base blob was huge but
// whose A and B endpoint blobs were small.
//
// The fixture builds a divergent history with a large
// common ancestor and small divergent branches.
func TestEvidenceBudget_F14_ThreeDotResolvesToMergeBase(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Base commit with a large blob.
	big := filepath.Join(dir, "shared.txt")
	var sb strings.Builder
	for i := 0; i < 20_000; i++ {
		sb.WriteString("shared divergent history content\n")
	}
	writeFilePath(t, big, sb.String())
	runGit(t, dir, "add", "shared.txt")
	runGit(t, dir, "commit", "-m", "base")

	// Branch A: divergent but small blob.
	runGit(t, dir, "checkout", "-b", "branchA")
	writeFilePath(t, big, "a\n")
	runGit(t, dir, "add", "shared.txt")
	runGit(t, dir, "commit", "-m", "A: tiny")

	// Branch B: divergent but small blob.
	runGit(t, dir, "checkout", "-b", "branchB", "master")
	writeFilePath(t, big, "b\n")
	runGit(t, dir, "add", "shared.txt")
	runGit(t, dir, "commit", "-m", "B: tiny")

	// Three-dot range: A...B. The merge-base is `base`
	// with the huge blob. The endpoints A and B are tiny.
	// The bounded policy MUST fire because the merge-base
	// blob is larger than the per-file cap.
	out := filepath.Join(t.TempDir(), "next.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "branchA...branchB", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// shared.txt must be demoted to BOUNDED_BODY because
	// the merge-base blob is huge.
	if !strings.Contains(d, "=== shared.txt ===") {
		t.Fatalf("F14 setup: shared.txt missing from diff")
	}
	if !strings.Contains(d,
		"RangeBaseBlobOID:") {
		t.Fatalf("F14 violated: three-dot range did not "+
			"demote shared.txt via merge-base classification. "+
			"Excerpt:\n%s",
			d[max(0, len(d)-500):])
	}
}
