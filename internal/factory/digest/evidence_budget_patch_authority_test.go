// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_test.go is the
// focused regression matrix for ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01.
//
// The defect under correction: range-mode evidence budgeting decided
// per-file suppression from the SIZE OF THE UNDERLYING SOURCE BLOB
// (EffectiveSize) rather than from the size of the ACTUAL UNIFIED
// PATCH. A 1.2 MB source file with a 623-byte one-line patch was
// therefore suppressed with LARGE_FILE_EVIDENCE_BOUNDED.
//
// The matrix covers the C3 conservation table:
//
//   - Large source, tiny patch          -> Included
//   - Large source, small deletion      -> Included
//   - Large source, small rename/edit   -> Included
//   - Genuinely oversized patch          -> Bounded (INCOMPLETE)
//   - Aggregate budget exceeded         -> existing policy conserved
//   - Binary change                     -> existing disposition conserved
//   - Sensitive source                  -> existing redaction conserved
//   - New large file                    -> addition patch budget applies
package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// patchBudgetLargeSourceBody returns a source body whose byte size
// comfortably exceeds MaxPerFileBytes (the per-file evidence budget)
// while remaining ordinary text.
func patchBudgetLargeSourceBody(t *testing.T) string {
	t.Helper()
	var sb strings.Builder
	for i := 0; i < 20000; i++ {
		sb.WriteString("ordinary source line for the large file body\n")
	}
	body := sb.String()
	if int64(len(body)) <= MaxPerFileBytes {
		t.Fatalf("fixture setup: body=%d bytes must exceed "+
			"MaxPerFileBytes=%d", len(body), MaxPerFileBytes)
	}
	return body
}

// patchBudgetRangeDigest generates a range digest over `revRange` and
// returns the rendered digest text.
//
// It exercises the FULL production write path (Generate followed by
// WriteWithWarnings), because the redaction policy is applied by the
// writer, not by the renderer. A digest produced through the CLI
// therefore carries the REDACTION_POLICY metadata.
func patchBudgetRangeDigest(t *testing.T, dir, revRange string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "next-targeted-digest.txt")
	content, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: revRange, Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := WriteWithWarnings(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: revRange, Output: out,
	}, content); err != nil {
		t.Fatalf("write with warnings: %v", err)
	}
	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read written digest: %v", err)
	}
	return string(written)
}

// patchBudgetCommitAll stages and commits everything in `dir`.
func patchBudgetCommitAll(t *testing.T, dir, msg string) {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", msg)
}

// patchBudgetExcerpt returns the tail of a digest for failure
// diagnostics. It keeps failure output bounded.
func patchBudgetExcerpt(d string) string {
	const max = 3000
	if len(d) <= max {
		return d
	}
	return d[len(d)-max:]
}

// ---------------------------------------------------------------------------
// Case 1: large source, tiny patch -> INCLUDED.
// ---------------------------------------------------------------------------

// TestPatchBudget_LargeSourceTinyPatch_IsIncluded is the primary
// discriminator for this ACT. The source blob is ~900 KB (well over
// the 64 KiB per-file budget) but the actual unified patch is a few
// hundred bytes. The patch MUST be embedded in full.
func TestPatchBudget_LargeSourceTinyPatch_IsIncluded(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	body := patchBudgetLargeSourceBody(t)
	big := filepath.Join(dir, "agent-runtime.ts")
	writeFilePath(t, big, body)
	patchBudgetCommitAll(t, dir, "add large source")

	// One-line modification: the patch is tiny.
	modified := strings.Replace(body,
		"ordinary source line for the large file body\n",
		"ordinary source line for the large file body CHANGED\n", 1)
	writeFilePath(t, big, modified)
	patchBudgetCommitAll(t, dir, "modify one line")

	// The ACTUAL patch size, measured independently.
	patchRaw, err := RunGit(dir, []string{
		"diff", "--unified=3", "HEAD~1..HEAD", "--", "agent-runtime.ts"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	patchBytes := len(patchRaw)
	if patchBytes > MaxPerFileBytes {
		t.Fatalf("fixture setup: patch=%d bytes must fit "+
			"MaxPerFileBytes=%d", patchBytes, MaxPerFileBytes)
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")

	if !strings.Contains(d, "=== agent-runtime.ts ===") {
		t.Fatalf("setup: agent-runtime.ts missing from digest")
	}
	// The changed line MUST be visible in the embedded patch.
	if !strings.Contains(d, "CHANGED") {
		t.Fatalf("patch budget authority violated: the actual "+
			"unified patch (%d bytes) was NOT embedded even though "+
			"it fits the per-file budget of %d bytes. Digest "+
			"excerpt:\n%s", patchBytes, MaxPerFileBytes,
			patchBudgetExcerpt(d))
	}
	// The blob-size-driven suppression must be gone.
	if strings.Contains(d, "LARGE_FILE_EVIDENCE_BOUNDED") {
		t.Fatalf("patch budget authority violated: digest still "+
			"reports LARGE_FILE_EVIDENCE_BOUNDED for a patch that "+
			"fits the budget. Digest excerpt:\n%s",
			patchBudgetExcerpt(d))
	}
	// The blob size is still reported, but only as metadata.
	if !strings.Contains(d, "BlobSize:") {
		t.Fatalf("blob-size metadata was dropped; it must remain " +
			"separable from the budget decision")
	}
	// The actual patch size must be reported truthfully.
	if !strings.Contains(d, fmt.Sprintf("PatchBytes: %d", patchBytes)) {
		t.Fatalf("digest does not report the actual patch size %d. "+
			"Digest excerpt:\n%s", patchBytes, patchBudgetExcerpt(d))
	}
}

// ---------------------------------------------------------------------------
// Case 2: large source, small deletion -> INCLUDED.
// ---------------------------------------------------------------------------

// TestPatchBudget_LargeSourceSmallDeletion_IsIncluded verifies that a
// small deletion inside a very large source file is embedded.
func TestPatchBudget_LargeSourceSmallDeletion_IsIncluded(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	body := patchBudgetLargeSourceBody(t)
	big := filepath.Join(dir, "bounded-objective.ts")
	writeFilePath(t, big, body)
	patchBudgetCommitAll(t, dir, "add large source")

	// Delete a single small line.
	lines := strings.SplitAfter(body, "\n")
	kept := append([]string{}, lines[:100]...)
	kept = append(kept, lines[101:]...)
	shortened := strings.Join(kept, "")
	writeFilePath(t, big, shortened)
	patchBudgetCommitAll(t, dir, "delete one line")

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	if strings.Contains(d, "LARGE_FILE_EVIDENCE_BOUNDED") {
		t.Fatalf("patch budget authority violated: small deletion in "+
			"a large source was suppressed. Digest excerpt:\n%s",
			patchBudgetExcerpt(d))
	}
	if !strings.Contains(d, "=== bounded-objective.ts ===") {
		t.Fatalf("setup: bounded-objective.ts missing from digest")
	}
}

// ---------------------------------------------------------------------------
// Case 3: large source, small rename -> INCLUDED.
// ---------------------------------------------------------------------------

// TestPatchBudget_LargeSourceSmallRename_IsIncluded verifies that a
// rename of a large source file (Git detects it as a pure rename, so
// the patch is small) is embedded rather than suppressed.
func TestPatchBudget_LargeSourceSmallRename_IsIncluded(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	body := patchBudgetLargeSourceBody(t)
	oldPath := filepath.Join(dir, "old-name.ts")
	writeFilePath(t, oldPath, body)
	patchBudgetCommitAll(t, dir, "add large source")

	newPath := filepath.Join(dir, "new-name.ts")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatalf("rename: %v", err)
	}
	patchBudgetCommitAll(t, dir, "rename large source")

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	if strings.Contains(d, "LARGE_FILE_EVIDENCE_BOUNDED") {
		t.Fatalf("patch budget authority violated: small rename of a "+
			"large source was suppressed. Digest excerpt:\n%s",
			patchBudgetExcerpt(d))
	}
	if !strings.Contains(d, "new-name.ts") {
		t.Fatalf("setup: renamed path missing from digest")
	}
}

// ---------------------------------------------------------------------------
// Case 4: genuinely oversized patch -> BOUNDED with an explicit
// incomplete-evidence disposition and a deterministic retrieval path.
// ---------------------------------------------------------------------------

// TestPatchBudget_GenuinelyOversizedPatch_IsBounded verifies that a
// patch that really exceeds the per-file budget is bounded, that the
// disposition is explicitly INCOMPLETE, and that a deterministic
// retrieval path is retained.
func TestPatchBudget_GenuinelyOversizedPatch_IsBounded(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// A file whose CONTENT change alone exceeds the budget.
	var sb strings.Builder
	for i := 0; i < 20000; i++ {
		sb.WriteString("first revision content line for oversize patch\n")
	}
	writeFilePath(t, filepath.Join(dir, "oversize.txt"), sb.String())
	patchBudgetCommitAll(t, dir, "base")

	var sb2 strings.Builder
	for i := 0; i < 20000; i++ {
		sb2.WriteString("second revision content line for oversize patch\n")
	}
	writeFilePath(t, filepath.Join(dir, "oversize.txt"), sb2.String())
	patchBudgetCommitAll(t, dir, "rewrite")

	patchRaw, err := RunGit(dir, []string{
		"diff", "--unified=3", "HEAD~1..HEAD", "--", "oversize.txt"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	if len(patchRaw) <= MaxPerFileBytes {
		t.Fatalf("fixture setup: patch=%d bytes must EXCEED "+
			"MaxPerFileBytes=%d", len(patchRaw), MaxPerFileBytes)
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	if !strings.Contains(d, "EvidenceCompleteness: INCOMPLETE") {
		t.Fatalf("oversized patch must yield an explicit "+
			"incomplete-evidence disposition. Digest excerpt:\n%s",
			patchBudgetExcerpt(d))
	}
	if !strings.Contains(d,
		"SuppressionReason: PATCH_BYTES_EXCEED_PER_FILE_BUDGET") {
		t.Fatalf("oversized patch must report a truthful suppression "+
			"reason. Digest excerpt:\n%s", patchBudgetExcerpt(d))
	}
	if !strings.Contains(d, "RetrievalPath: git diff --unified=3") {
		t.Fatalf("oversized patch must retain a deterministic "+
			"retrieval path. Digest excerpt:\n%s",
			patchBudgetExcerpt(d))
	}
	// The patch body must NOT be silently truncated and presented
	// as complete.
	if strings.Contains(d, "second revision content line for oversize patch") {
		t.Fatalf("oversized patch body was embedded; it must be " +
			"bounded, never truncated-and-presented-as-complete")
	}
}

// ---------------------------------------------------------------------------
// Case 8: new large file -> the ACTUAL ADDITION patch budget applies.
// ---------------------------------------------------------------------------

// TestPatchBudget_NewLargeFile_ActualAdditionPatchBudgetApplies verifies
// that a newly added large file is bounded because its ADDITION patch
// is large, and that the reported PatchBytes reflects the real
// addition patch rather than an unrelated quantity.
func TestPatchBudget_NewLargeFile_ActualAdditionPatchBudgetApplies(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	body := patchBudgetLargeSourceBody(t)
	writeFilePath(t, filepath.Join(dir, "added-large.ts"), body)
	patchBudgetCommitAll(t, dir, "add large file")

	patchRaw, err := RunGit(dir, []string{
		"diff", "--unified=3", "HEAD~1..HEAD", "--", "added-large.ts"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	if len(patchRaw) <= MaxPerFileBytes {
		t.Fatalf("fixture setup: addition patch=%d bytes must "+
			"EXCEED MaxPerFileBytes=%d", len(patchRaw),
			MaxPerFileBytes)
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	if !strings.Contains(d, "EvidenceCompleteness: INCOMPLETE") {
		t.Fatalf("new large file must be bounded by the actual "+
			"addition patch budget. Digest excerpt:\n%s",
			patchBudgetExcerpt(d))
	}
	if !strings.Contains(d,
		fmt.Sprintf("PatchBytes: %d", len(patchRaw))) {
		t.Fatalf("digest must report the actual addition patch size "+
			"%d. Digest excerpt:\n%s", len(patchRaw),
			patchBudgetExcerpt(d))
	}
}

// ---------------------------------------------------------------------------
// Case 6: sensitive source -> existing redaction policy conserved.
// ---------------------------------------------------------------------------

// TestPatchBudget_SensitiveSource_RedactionConserved verifies that
// including a small patch inside a large source file does NOT bypass
// the existing redaction policy: non-source files are still redacted
// and source files are still scanned for secret-like patterns.
