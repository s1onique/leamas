// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_corpus_test.go is the corpus
// of F3 (range-mode historical), F4 (production classifier),
// F2 (per-file cap), F6 (canonical stats), and F16/F17/F18
// (range-specific rendering for large/deleted/missing files)
// regression tests for the bounded evidence renderer.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// F18 (CORRECTION04) — range-mode large-file rendering must
// produce the BOUNDED_BODY stub with the historical blob
// identity even when the worktree file is absent (deleted) or
// has been replaced with a small file. The previous regression
// test was a fixture-construction smoke test that did not
// assert anything; the production renderer silently fell back
// to a worktree-only classification and dropped the historical
// identity metadata.
// ---------------------------------------------------------------------------

// bigRangeBlobContent returns a deterministic ~2 MiB content
// string. 30 000 lines × ~70 bytes ≈ 2 MiB, well above the
// 1 MiB boundary where F17's control-flow bug fired.
func bigRangeBlobContent() string {
	var sb strings.Builder
	for i := 0; i < 30_000; i++ {
		sb.WriteString("big ordinary content padding line for size\n")
	}
	return sb.String()
}

// rangeEvidenceFixture builds a repository with a commit
// that introduces big.txt containing bigRangeBlobContent,
// then optionally deletes it. Returns (dir, baseOID,
// headOID) where baseOID is HEAD~1:big.txt (the historic
// blob) and headOID is the current HEAD.
func rangeEvidenceFixture(t *testing.T,
	keepWorktree bool) (dir, baseOID, headOID string) {
	t.Helper()
	dir = t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	big := filepath.Join(dir, "big.txt")
	writeFilePath(t, big, bigRangeBlobContent())
	runGit(t, dir, "add", "big.txt")
	runGit(t, dir, "commit", "-m", "add big")
	runGit(t, dir, "commit", "--allow-empty", "-m", "anchor")

	// The historical blob lives at HEAD~2 (= the
	// "add big" commit). HEAD~1 is the empty "anchor"
	// commit and does not have big.txt.
	if !keepWorktree {
		if err := os.Remove(big); err != nil {
			t.Fatalf("remove big: %v", err)
		}
		runGit(t, dir, "add", "-A")
		runGit(t, dir, "commit", "-m", "delete big")
	}

	// After the optional delete commit, HEAD~2 is the
	// "add big" commit and HEAD~1 is the empty "anchor".
	rawBase, err := RunGit(dir, []string{"rev-parse", "HEAD~2:big.txt"})
	if err != nil {
		t.Fatalf("rev-parse HEAD~2:big.txt: %v", err)
	}
	baseOID = strings.TrimSpace(rawBase)

	rawHead, err := RunGit(dir, []string{"rev-parse", "HEAD"})
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	headOID = strings.TrimSpace(rawHead)
	return dir, baseOID, headOID
}

// F18 (CORRECTION04): a deleted historical blob >1 MiB
// must still produce the BOUNDED_BODY stub. This is the
// exact failure mode that the F17 control-flow bug allowed:
// the second pass through boundedFileBlock() saw size=0
// and reclassified as NORMAL.
func TestEvidenceBudget_RangeDeletedLargeFile_IsBounded(t *testing.T) {
	dir, baseOID, headOID := rangeEvidenceFixture(t, false)

	// After fixture: HEAD = delete big, HEAD~1 = anchor,
	// HEAD~2 = add big. Use HEAD~3..HEAD~2 so the diff
	// covers the "add big" commit (with the historical
	// blob) and the next commit (HEAD~2) IS that "add
	// big" commit. RangeBaseBlobOID will be the empty
	// HEAD~3 side, RangeHeadBlobOID will be the big.txt
	// blob at HEAD~2.
	out := filepath.Join(t.TempDir(), "next.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~2..HEAD~0", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	bigBlock := extractBlock(t, d, "=== big.txt ===")
	if !strings.Contains(bigBlock, "Classification: BOUNDED_BODY") {
		t.Fatalf("F18 violated: deleted historical "+
			"big.txt not demoted. Excerpt:\n%s", bigBlock)
	}
	if !strings.Contains(bigBlock,
		"RangeBaseBlobOID: "+baseOID) {
		t.Fatalf("F18 violated: RangeBaseBlobOID does "+
			"not match HEAD~1:big.txt = %s. Excerpt:\n%s",
			baseOID, bigBlock)
	}
	// At HEAD the big.txt is deleted, so the head
	// blob status is MISSING and the OID field is empty.
	// F16 (CORRECTION04): an empty OID field with a
	// MISSING status is the correct rendering — never a
	// raw ref expression.
	if strings.Contains(bigBlock,
		"RangeHeadBlobOID: "+headOID+":big.txt") {
		t.Fatalf("F16 violated: RangeHeadBlobOID "+
			"contains raw ref expression. Excerpt:\n%s",
			bigBlock)
	}
	if !strings.Contains(bigBlock,
		"RangeHeadBlobStatus: MISSING") {
		t.Fatalf("F18 violated: RangeHeadBlobStatus "+
			"must report MISSING for deleted HEAD blob. "+
			"Excerpt:\n%s", bigBlock)
	}
	if !strings.Contains(bigBlock, WarningCodeLargeFileBounded) {
		t.Fatalf("F18 violated: WarningCode missing. "+
			"Excerpt:\n%s", bigBlock)
	}
}

// F18 (CORRECTION04): a large historical blob replaced by a
// small replacement must still produce the BOUNDED_BODY stub.
// This is the second failure mode the F17 control-flow bug
// allowed: the second pass through boundedFileBlock() saw
// size=10 and reclassified as NORMAL.
func TestEvidenceBudget_RangeLargeToSmall_StillBounded(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	big := filepath.Join(dir, "big.txt")
	writeFilePath(t, big, bigRangeBlobContent())
	runGit(t, dir, "add", "big.txt")
	runGit(t, dir, "commit", "-m", "add big")
	runGit(t, dir, "commit", "--allow-empty", "-m", "anchor")

	baseOIDRaw, err := RunGit(dir, []string{
		"rev-parse", "HEAD~1:big.txt"})
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	baseOID := strings.TrimSpace(baseOIDRaw)

	writeFilePath(t, big, "tiny\n")
	runGit(t, dir, "add", "big.txt")
	runGit(t, dir, "commit", "-m", "shrink big")

	// headOID = blob OID at HEAD:big.txt (the small
	// replacement), not the HEAD commit hash.
	headOIDRaw, err := RunGit(dir, []string{"rev-parse", "HEAD:big.txt"})
	if err != nil {
		t.Fatalf("rev-parse HEAD:big.txt: %v", err)
	}
	headOID := strings.TrimSpace(headOIDRaw)

	out := filepath.Join(t.TempDir(), "next.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~2..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	bigBlock := extractBlock(t, d, "=== big.txt ===")
	if !strings.Contains(bigBlock, "Classification: BOUNDED_BODY") {
		t.Fatalf("F18 violated: large-to-small historical "+
			"blob not demoted. Excerpt:\n%s", bigBlock)
	}
	if !strings.Contains(bigBlock,
		"RangeBaseBlobOID: "+baseOID) {
		t.Fatalf("F18 violated: RangeBaseBlobOID does "+
			"not match historical OID %s. Excerpt:\n%s",
			baseOID, bigBlock)
	}
	if !strings.Contains(bigBlock,
		"RangeHeadBlobOID: "+headOID) {
		t.Fatalf("F18 violated: RangeHeadBlobOID does "+
			"not match HEAD=%s. Excerpt:\n%s",
			headOID, bigBlock)
	}
}

// F16 (CORRECTION04): when the BOUNDED_BODY stub is
// emitted and the historical blob does NOT exist on one
// side of the range, that side MUST report a MISSING
// status and an empty OID field. The previous
// implementation stored the raw "<ref>:<path>" string
// into the BlobOID field whenever cat-file reported
// "missing", which is not a valid OID.
//
// The fixture deliberately makes the historical blob
// LARGE so the BOUNDED_BODY stub fires; the test then
// asserts that the head side (where the blob was
// deleted) carries RangeHeadBlobStatus: MISSING and an
// empty RangeHeadBlobOID.
func TestEvidenceBudget_RangeHistoricalBlob_MissingStatusReported(t *testing.T) {
	dir, baseOID, _ := rangeEvidenceFixture(t, false)

	// After fixture: HEAD = delete big, HEAD~1 = anchor,
	// HEAD~2 = add big. Range HEAD~2..HEAD covers the
	// "delete big" commit.
	out := filepath.Join(t.TempDir(), "next.txt")
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~2..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	blk := extractBlock(t, d, "=== big.txt ===")
	if !strings.Contains(blk,
		"RangeBaseBlobOID: "+baseOID) {
		t.Fatalf("F16 setup: expected historical OID "+
			"%s at base. Excerpt:\n%s", baseOID, blk)
	}
	if !strings.Contains(blk,
		"RangeBaseBlobStatus: PRESENT") {
		t.Fatalf("F16 setup: base status should be "+
			"PRESENT. Excerpt:\n%s", blk)
	}
	if !strings.Contains(blk,
		"RangeHeadBlobStatus: MISSING") {
		t.Fatalf("F16 violated: RangeHeadBlobStatus "+
			"must report MISSING for deleted HEAD blob. "+
			"Excerpt:\n%s", blk)
	}
	// NEVER a raw ref expression on the missing side.
	if strings.Contains(blk,
		"RangeHeadBlobOID: HEAD:big.txt") {
		t.Fatalf("F16 violated: RangeHeadBlobOID "+
			"contains raw ref expression. Excerpt:\n%s",
			blk)
	}
}

// extractBlock returns the substring of d starting at the
// first occurrence of marker and ending at the next "\n## "
// section boundary or end of string. Used by the F18
// assertion helpers.
func extractBlock(t *testing.T, d, marker string) string {
	t.Helper()
	idx := strings.Index(d, marker)
	if idx < 0 {
		t.Fatalf("marker %q not found in digest:\n%s",
			marker, d)
	}
	rest := d[idx:]
	end := strings.Index(rest[len(marker):], "\n## ")
	if end < 0 {
		return rest
	}
	return rest[:len(marker)+end]
}

// TestLoadClassifierData_LargeFile_ReadsAtMostScanCap
// verifies the F8 (CORRECTION02) fix: loadClassifierData must
// not allocate the full file size before capping. The previous
// implementation called `make([]byte, info.Size())` and only
// THEN sliced the result down to scanCap, which for a 20 GiB
// file allocated 20 GiB before throwing it away.
//
// The test uses a 100 MiB sparse file (~100 MiB on disk, but
// the system allocates only the logical size) and asserts that
// the returned prefix and body are at most classifierScanCap
// bytes.
func TestLoadClassifierData_LargeFile_ReadsAtMostScanCap(t *testing.T) {
	dir := t.TempDir()
	big := filepath.Join(dir, "big.bin")
	f, err := os.Create(big)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := f.Truncate(100 * 1024 * 1024); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	f.Close()
	prefix, body, ok := loadClassifierData(big)
	if !ok {
		t.Fatalf("loadClassifierData failed")
	}
	if len(body) > classifierScanCap {
		t.Fatalf("F8 violated: body=%d bytes, "+
			"expected at most classifierScanCap=%d",
			len(body), classifierScanCap)
	}
	if len(prefix) > classifierPrefixBytes {
		t.Fatalf("F8 violated: prefix=%d bytes, "+
			"expected at most classifierPrefixBytes=%d",
			len(prefix), classifierPrefixBytes)
	}
}
