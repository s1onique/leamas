// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_amendment_retrieval_test.go
// holds the AMENDMENT-01 retrieval witnesses for
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01.
//
// A2  the published retrieval command must be the CANONICAL
//
//	    serialization of the very same specification that produced the
//	    embedded patch. It must therefore:
//
//		   - reproduce the measured patch byte for byte, including for a
//		     rename (where the predecessor's destination-only command
//	    produced a whole-blob patch instead of the rename patch);
//		   - keep describing the SAME historical range after HEAD advances,
//	    because the canonical endpoints are pinned to immutable OIDs;
//		   - use LITERAL pathspec semantics, so a path containing glob
//	    metacharacters matches exactly one file.
//
// The companion files hold the A1, A3, A4 and A6 witnesses.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// A2: canonical generation and retrieval.
// ---------------------------------------------------------------------------

// TestAmendment_A2_RetrievalReproducesTheMeasuredPatch verifies that
// the published retrieval command reproduces the EXACT patch that was
// measured, for a rename.
//
// Independently reproduced defect: the predecessor published
// "git diff --unified=3 <range> -- <destination>", which for a pure
// rename yields a whole-blob "new file" patch (20131 bytes in the
// reviewer's disposable repository) instead of the true rename patch
// (99 bytes).
func TestAmendment_A2_RetrievalReproducesTheMeasuredPatch(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	body := patchBudgetLargeSourceBody(t)
	writeFilePath(t, filepath.Join(dir, "old-name.ts"), body)
	patchBudgetCommitAll(t, dir, "add large source")

	newPath := filepath.Join(dir, "new-name.ts")
	if err := os.Rename(
		filepath.Join(dir, "old-name.ts"), newPath); err != nil {
		t.Fatalf("rename: %v", err)
	}
	patchBudgetCommitAll(t, dir, "rename large source")

	// The canonical rename-aware patch.
	canonical, err := RunGit(dir, []string{
		"diff", "--unified=3",
		"--find-renames=30%", "--find-copies=30%",
		"HEAD~1..HEAD", "--", "new-name.ts", "old-name.ts"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	// The predecessor's destination-only retrieval patch.
	legacy, err := RunGit(dir, []string{
		"diff", "--unified=3", "HEAD~1..HEAD", "--", "new-name.ts"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	// PRECONDITION: the two commands really do differ, and the
	// predecessor's one is much larger.
	if len(legacy) <= len(canonical) {
		t.Fatalf("fixture setup: legacy=%d bytes must be LARGER "+
			"than canonical=%d bytes for this fixture to be a "+
			"discriminator", len(legacy), len(canonical))
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	retrieval := patchBudgetExtractRangeRetrievalPath(t, d)

	reproduced := patchBudgetRunShell(t, dir, retrieval)
	// The range-level retrieval command reproduces EVERY patch of the
	// range. For this single-file range it must equal the canonical
	// rename-aware patch exactly.
	if reproduced != canonical {
		t.Fatalf("retrieval equivalence violated: the published "+
			"retrieval command produced %d bytes, the canonical "+
			"patch is %d bytes. Retrieval:\n%s",
			len(reproduced), len(canonical), retrieval)
	}
	if reproduced == legacy {
		t.Fatalf("retrieval equivalence violated: the published " +
			"retrieval command reproduced the LEGACY " +
			"destination-only patch, not the rename-aware one")
	}
}

// TestAmendment_A2_RetrievalSurvivesHeadAdvance verifies that the
// published retrieval command still describes the SAME historical
// range after HEAD advances.
//
// The predecessor published the caller's raw revision expression
// ("HEAD~1..HEAD"). After the next commit that expression silently
// starts describing a DIFFERENT range, so the retrieval path stops
// being a faithful reproduction of the embedded evidence.
func TestAmendment_A2_RetrievalSurvivesHeadAdvance(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	writeFilePath(t, filepath.Join(dir, "tracked.txt"), "one\n")
	patchBudgetCommitAll(t, dir, "c1")
	writeFilePath(t, filepath.Join(dir, "tracked.txt"), "two\n")
	patchBudgetCommitAll(t, dir, "c2")

	measured, err := RunGit(dir, []string{
		"diff", "--unified=3", "HEAD~1..HEAD", "--", "tracked.txt"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	if !strings.Contains(measured, "-one") ||
		!strings.Contains(measured, "+two") {
		t.Fatalf("fixture setup: measured patch does not describe "+
			"the expected change:\n%s", measured)
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")

	// Advance HEAD with an unrelated commit.
	writeFilePath(t, filepath.Join(dir, "other.txt"), "unrelated\n")
	patchBudgetCommitAll(t, dir, "c3")

	// The retrieval path is published in the digest captured BEFORE
	// the advance. It must still reproduce the same patch.
	retrieval := patchBudgetExtractRangeRetrievalPath(t, d)
	reproduced := patchBudgetRunShell(t, dir, retrieval)
	if reproduced != measured {
		t.Fatalf("retrieval equivalence violated after HEAD "+
			"advanced: the published retrieval command produced:\n%s\n"+
			"but the measured patch was:\n%s\nretrieval:\n%s",
			reproduced, measured, retrieval)
	}
	// PRECONDITION CHECK: the predecessor's expression WOULD have
	// drifted. This proves the fixture is a real discriminator.
	drifted, err := RunGit(dir, []string{
		"diff", "--unified=3", "HEAD~1..HEAD", "--", "tracked.txt"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	if drifted == measured {
		t.Fatalf("fixture setup: the legacy revision expression " +
			"must drift after HEAD advances for this fixture to be " +
			"a discriminator")
	}
}

// ---------------------------------------------------------------------------
// A2: literal pathspec conservation.
// ---------------------------------------------------------------------------

// TestAmendment_A2_LiteralPathspecConservation verifies that the
// per-file evidence budget is applied to the patch of EXACTLY ONE
// file, even when the path contains glob metacharacters.
//
// Independently reproduced defect: the predecessor passed the path to
// `git diff` as a bare pathspec. Git interprets a pathspec as a
// WILDCARD expression, so a file named "literal*.ts" also matched
// "literal-other.ts", and the per-file budget was applied to a patch
// that belonged to several files.
//
// The canonical specification uses Git's ":(literal)" pathspec magic.
func TestAmendment_A2_LiteralPathspecConservation(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Two files: one whose name contains a glob metacharacter and
	// one ordinary file that the wildcard would also match.
	writeFilePath(t, filepath.Join(dir, "literal*.ts"),
		"star file\n")
	writeFilePath(t, filepath.Join(dir, "literal-other.ts"),
		"other file\n")
	patchBudgetCommitAll(t, dir, "add both")

	writeFilePath(t, filepath.Join(dir, "literal*.ts"),
		"star file CHANGED\n")
	writeFilePath(t, filepath.Join(dir, "literal-other.ts"),
		"other file CHANGED\n")
	patchBudgetCommitAll(t, dir, "edit both")

	// PRECONDITION: the predecessor's bare pathspec really did match
	// BOTH files.
	wildcard, err := RunGit(dir, []string{
		"diff", "--unified=3", "HEAD~1..HEAD", "--", "literal*.ts"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	if !strings.Contains(wildcard, "literal-other.ts") {
		t.Fatalf("fixture setup: the bare pathspec must really "+
			"match both files for this witness to be a "+
			"discriminator. Patch:\n%s", wildcard)
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")

	// POSTCONDITION: the evidence for the glob-named file contains
	// ONLY that file's patch.
	section := patchBudgetFileSection(t, d, "literal*.ts")
	if strings.Contains(section, "literal-other.ts") {
		t.Fatalf("literal pathspec conservation violated: the "+
			"evidence for %q contains the patch of another file. "+
			"Section:\n%s", "literal*.ts", section)
	}
	if !strings.Contains(section, "star file CHANGED") {
		t.Fatalf("literal pathspec conservation violated: the "+
			"evidence for %q does not contain its own change. "+
			"Section:\n%s", "literal*.ts", section)
	}
	// The ordinary file keeps its own separate evidence.
	otherSection := patchBudgetFileSection(t, d, "literal-other.ts")
	if !strings.Contains(otherSection, "other file CHANGED") {
		t.Fatalf("literal pathspec conservation violated: the "+
			"evidence for %q does not contain its own change. "+
			"Section:\n%s", "literal-other.ts", otherSection)
	}
	if strings.Contains(otherSection, "star file CHANGED") {
		t.Fatalf("literal pathspec conservation violated: the "+
			"evidence for %q contains the patch of another file. "+
			"Section:\n%s", "literal-other.ts", otherSection)
	}
}
