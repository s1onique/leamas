// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_bounded_test.go
// holds the BOUNDED-disposition witnesses for
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01.
//
// The companion file evidence_budget_patch_authority_test.go holds the
// INCLUDED-disposition witnesses.
package digest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

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
	// AMENDMENT-01 (A5): the retrieval path must not merely LOOK
	// like a command. It must be the canonical serialization of the
	// SAME specification that produced the measured patch, so
	// executing it reproduces the patch byte for byte.
	// The per-file retrieval path is the PATHSCOPING suffix. It is
	// appended to the canonical range-level command, which carries
	// the immutable comparison endpoints and the diff options.
	scoping := patchBudgetExtractRetrievalPath(t, d)
	rangeCmd := patchBudgetExtractRangeRetrievalPath(t, d)
	if !strings.HasPrefix(rangeCmd, "git diff") {
		t.Fatalf("the range must publish a canonical retrieval "+
			"command. Digest excerpt:\n%s", patchBudgetExcerpt(d))
	}
	retrieval := rangeCmd + " " + scoping
	reproduced := patchBudgetRunShell(t, dir, retrieval)
	if len(reproduced) != len(patchRaw) {
		t.Fatalf("retrieval equivalence violated: the published "+
			"retrieval command produced %d bytes but the measured "+
			"patch was %d bytes. Retrieval:\n%s",
			len(reproduced), len(patchRaw), retrieval)
	}
	if reproduced != patchRaw {
		t.Fatalf("retrieval equivalence violated: the published "+
			"retrieval command produced a DIFFERENT patch than the "+
			"one measured. Retrieval:\n%s", retrieval)
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
