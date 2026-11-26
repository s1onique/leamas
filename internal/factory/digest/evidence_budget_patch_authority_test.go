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
	// AMENDMENT-01 (A5): assert the COMPLETE expected patch against
	// the rendered file-evidence section, not merely that the changed
	// marker appears somewhere in the digest.
	section := patchBudgetFileSection(t, d, "agent-runtime.ts")
	if !strings.Contains(section,
		strings.TrimRight(patchRaw, "\n")) {
		t.Fatalf("the complete unified patch must be rendered in "+
			"the file section. Section:\n%s", section)
	}
	if !strings.Contains(section, "EvidenceCompleteness: COMPLETE") {
		t.Fatalf("a patch that fits the budget must be published "+
			"complete. Section:\n%s", section)
	}
	// The changed line MUST be visible in the embedded patch.
	if !strings.Contains(section, "CHANGED") {
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
	// AMENDMENT-01 (A5): assert the ACTUAL deletion hunk, not
	// merely that the path appears in the changeset manifest.
	section := patchBudgetFileSection(t, d, "bounded-objective.ts")
	if !strings.Contains(section, "EvidenceCompleteness: COMPLETE") {
		t.Fatalf("a small deletion that fits the budget must be "+
			"published complete. Section:\n%s", section)
	}
	deletedLine := strings.SplitAfter(body, "\n")[100]
	if !strings.Contains(section, "-"+strings.TrimSuffix(deletedLine, "\n")) {
		t.Fatalf("the actual deletion hunk is missing from the "+
			"evidence section. Expected the deleted line %q. "+
			"Section:\n%s", deletedLine, section)
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
	// AMENDMENT-01 (A5): assert the ACTUAL rename evidence. The
	// predecessor only checked that the destination name appeared
	// somewhere in the digest, which the changeset manifest already
	// satisfies, so the assertion could not fail.
	section := patchBudgetFileSection(t, d, "new-name.ts")
	if !strings.Contains(section, "EvidenceCompleteness: COMPLETE") {
		t.Fatalf("a small rename that fits the budget must be "+
			"published complete. Section:\n%s", section)
	}
	// The rename header must name BOTH sides.
	//
	// AMENDMENT-01 FINALIZATION (F3): the predecessor combined the two
	// negative predicates with "&&", so the assertion only failed
	// when BOTH headers were absent. The contract requires BOTH
	// headers to be present, so each is asserted separately.
	if !strings.Contains(section, "rename from") {
		t.Fatalf("the rename evidence must carry the 'rename from' "+
			"header naming the source path. Section:\n%s", section)
	}
	if !strings.Contains(section, "rename to") {
		t.Fatalf("the rename evidence must carry the 'rename to' "+
			"header naming the destination path. Section:\n%s",
			section)
	}
	// The exact expected headers, so a rename of the WRONG file
	// cannot satisfy the assertion.
	if !strings.Contains(section, "rename from old-name.ts") {
		t.Fatalf("the rename evidence must name the SOURCE path "+
			"exactly. Section:\n%s", section)
	}
	if !strings.Contains(section, "rename to new-name.ts") {
		t.Fatalf("the rename evidence must name the DESTINATION "+
			"path exactly. Section:\n%s", section)
	}
}
