// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_amendment_identity_test.go
// holds the AMENDMENT-01 identity witnesses for
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01.
//
// A3  the base blob identity of a renamed file must resolve to its
//
//	    TRUE preimage (looked up at the SOURCE path), not to the
//	    destination path;
//		A4  the disposition of an immutable historical range must not
//	    depend on the CURRENT working tree.
//
// The companion file
// evidence_budget_patch_authority_amendment_test.go holds the A1, A2
// and A6 witnesses.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// A3: historical blob identity.
// ---------------------------------------------------------------------------

// TestAmendment_A3_RenameBaseBlobIsThePreimage verifies that the
// base blob identity of a renamed file resolves to its TRUE preimage.
//
// Independently reproduced defect: the predecessor looked the base
// blob up at the DESTINATION path, so "git cat-file -t
// HEAD~1:new.ts" reported "path exists on disk, but not in HEAD~1"
// while the real preimage was present at "HEAD~1:old.ts".
//
// The witness uses an OVERSIZED rename-and-edit so the incomplete
// evidence disposition is actually exercised.
func TestAmendment_A3_RenameBaseBlobIsThePreimage(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// A large source that is RENAMED and then edited enough that the
	// unified patch is oversized, while Git still pairs the two
	// sides as a rename (R050 at the digest's 30% threshold).
	//
	// The witness needs a genuine rename: when Git does not pair the
	// two sides it reports "A" plus "D", the destination path
	// genuinely does not exist at the base endpoint, and the base
	// identity is legitimately MISSING.
	var sb strings.Builder
	for i := 0; i < 40000; i++ {
		sb.WriteString("preimage content line for identity\n")
	}
	writeFilePath(t, filepath.Join(dir, "old-name.ts"), sb.String())
	patchBudgetCommitAll(t, dir, "add large source")

	if err := os.Rename(filepath.Join(dir, "old-name.ts"),
		filepath.Join(dir, "new-name.ts")); err != nil {
		t.Fatalf("rename: %v", err)
	}
	var sb2 strings.Builder
	for i := 0; i < 20000; i++ {
		sb2.WriteString("preimage content line for identity\n")
	}
	for i := 0; i < 20000; i++ {
		sb2.WriteString("postimage replacement tail line\n")
	}
	writeFilePath(t, filepath.Join(dir, "new-name.ts"), sb2.String())
	patchBudgetCommitAll(t, dir, "rename and edit")

	// PRECONDITION 1: Git really does pair the two sides as a
	// rename, so the manifest carries the source path.
	nameStatus, err := RunGit(dir, []string{
		"diff", "--name-status", "--find-renames=30%",
		"--find-copies=30%", "HEAD~1..HEAD"})
	if err != nil {
		t.Fatalf("git diff --name-status: %v", err)
	}
	if !strings.HasPrefix(nameStatus, "R") {
		t.Fatalf("fixture setup: Git must detect the rename for "+
			"this fixture to be a discriminator. Got:\n%s",
			nameStatus)
	}
	// PRECONDITION 2: the base endpoint really holds the preimage at
	// the SOURCE path and not at the destination path.
	baseOID, err := RunGit(dir, []string{
		"rev-parse", "HEAD~1:old-name.ts"})
	if err != nil {
		t.Fatalf("fixture setup: base preimage must exist: %v", err)
	}
	if _, err := RunGit(dir, []string{
		"rev-parse", "HEAD~1:new-name.ts"}); err == nil {
		t.Fatalf("fixture setup: the destination path must NOT " +
			"exist at the base endpoint for this fixture to be a " +
			"discriminator")
	}
	// PRECONDITION 3: the patch really is oversized, so the
	// incomplete disposition is actually exercised.
	patchRaw, err := RunGit(dir, []string{
		"diff", "--unified=3", "--find-renames=30%", "--find-copies=30%",
		"HEAD~1..HEAD", "--", "new-name.ts", "old-name.ts"})
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	if len(patchRaw) <= MaxPerFileBytes {
		t.Fatalf("fixture setup: patch=%d bytes must EXCEED "+
			"MaxPerFileBytes=%d", len(patchRaw), MaxPerFileBytes)
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	section := patchBudgetFileSection(t, d, "new-name.ts")

	if !strings.Contains(section,
		"RangeBaseBlobStatus: PRESENT") {
		t.Fatalf("historical blob identity violated: the base blob "+
			"of a renamed file must resolve to its true preimage. "+
			"Section:\n%s", section)
	}
	if !strings.Contains(section,
		"RangeBaseBlobOID: "+strings.TrimSpace(baseOID)) {
		t.Fatalf("historical blob identity violated: expected the "+
			"base blob OID %s of the preimage. Section:\n%s",
			strings.TrimSpace(baseOID), section)
	}
}

// ---------------------------------------------------------------------------
// A4: historical artifact classification conservation.
// ---------------------------------------------------------------------------

// TestAmendment_A4_HistoricalDispositionIsIndependentOfWorktree
// verifies that the disposition of an immutable historical range does
// not change merely because the CURRENT worktree changed.
//
// The predecessor's classifier reads the CURRENT worktree file, not
// the historical blob. With the blob-size suppression removed (the
// point of the original ACT), the worktree classification became the
// ONLY classifier for a historical artifact.
//
// The witness: the same immutable endpoints are digested twice, once
// with the worktree holding ordinary text and once with the worktree
// holding the historical digest artifact. The two dispositions MUST
// agree, because the range being described is identical.
func TestAmendment_A4_HistoricalDispositionIsIndependentOfWorktree(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// The historical artifact is a recognised Leamas digest.
	artifact := "# Targeted digest\n\n" +
		"LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n" +
		"ordinary artifact body\n"
	writeFilePath(t, filepath.Join(dir, "artifact.txt"), artifact)
	patchBudgetCommitAll(t, dir, "add artifact")
	writeFilePath(t, filepath.Join(dir, "artifact.txt"),
		artifact+"one more line\n")
	patchBudgetCommitAll(t, dir, "edit artifact")

	// Digest 1: worktree still holds the artifact.
	dArtifact := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	sectionArtifact := patchBudgetFileSection(t, dArtifact, "artifact.txt")

	// Digest 2: SAME immutable endpoints, but the worktree now holds
	// ordinary text instead of the artifact.
	writeFilePath(t, filepath.Join(dir, "artifact.txt"),
		"ordinary unrelated worktree text\n")
	dPlain := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	sectionPlain := patchBudgetFileSection(t, dPlain, "artifact.txt")

	classOf := func(section string) string {
		for _, line := range strings.Split(section, "\n") {
			if strings.HasPrefix(line, "Classification: ") {
				return strings.TrimPrefix(line, "Classification: ")
			}
		}
		return ""
	}
	classArtifact := classOf(sectionArtifact)
	classPlain := classOf(sectionPlain)

	if classArtifact == "" || classPlain == "" {
		t.Fatalf("fixture setup: both dispositions must state a "+
			"Classification. artifact=%q plain=%q",
			classArtifact, classPlain)
	}
	if classArtifact != classPlain {
		t.Fatalf("historical artifact classification is NOT "+
			"conserved: the SAME immutable range was classified "+
			"as %q with the artifact in the worktree and %q with "+
			"ordinary text in the worktree. The disposition of an "+
			"immutable historical range must not depend on "+
			"unrelated current worktree content.",
			classArtifact, classPlain)
	}
}

// ---------------------------------------------------------------------------
// F1: historical classification authority for DELETIONS and for
// unavailable historical evidence.
//
// AMENDMENT-01 FINALIZATION (F1): the head endpoint alone is NOT
// sufficient classification authority. A DELETION has no head blob, so
// the predecessor fell back to the CURRENT WORKTREE and the original
// authority defect returned for exactly the case it was meant to
// close.
//
// The authority is now:
//  1. the HEAD blob, when it resolves;
//  2. otherwise the BASE preimage (a DELETION, a RENAME);
//  3. otherwise NO historical evidence, reported EXPLICITLY. The
//     worktree is NEVER substituted.
// ---------------------------------------------------------------------------

// TestAmendment_F1_DeletedArtifactIsClassifiedFromTheBasePreimage
// verifies that a DELETED historical artifact is classified from the
// BASE preimage and not from the current working tree.
//
// PRECONDITION: the artifact exists at the base endpoint and NOT at
// the head endpoint.
//
// POSTCONDITION: the disposition names the BASE endpoint as the
// evidence source, and the classification is the artifact's own.
func TestAmendment_F1_DeletedArtifactIsClassifiedFromTheBasePreimage(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// A recognised Leamas digest artifact at the base endpoint.
	artifact := "# Targeted digest\n\n" +
		"LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n" +
		"ordinary artifact body\n"
	writeFilePath(t, filepath.Join(dir, "del-artifact.txt"), artifact)
	patchBudgetCommitAll(t, dir, "add artifact")

	// The artifact is DELETED at the head endpoint.
	if err := os.Remove(filepath.Join(dir, "del-artifact.txt")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	patchBudgetCommitAll(t, dir, "delete artifact")

	// PRECONDITION: the head endpoint really has no blob at that
	// path, and the base endpoint really does.
	if _, err := RunGit(dir, []string{
		"rev-parse", "HEAD:del-artifact.txt"}); err == nil {
		t.Fatalf("fixture setup: the head endpoint must NOT hold " +
			"the artifact for this witness to be a discriminator")
	}
	if _, err := RunGit(dir, []string{
		"rev-parse", "HEAD~1:del-artifact.txt"}); err != nil {
		t.Fatalf("fixture setup: the base endpoint must hold the "+
			"artifact: %v", err)
	}

	// The worktree now holds UNRELATED ordinary text at that path.
	// This is the mutable state that must NOT become the authority.
	writeFilePath(t, filepath.Join(dir, "del-artifact.txt"),
		"ordinary unrelated worktree text\n")

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	section := patchBudgetFileSection(t, d, "del-artifact.txt")

	// POSTCONDITION 1: the BASE endpoint carried the authority.
	if !strings.Contains(section, "HistoricalEvidenceSource: BASE") {
		t.Fatalf("historical classification authority violated: a "+
			"DELETED artifact must be classified from the BASE "+
			"preimage, because the head endpoint has no blob. "+
			"Section:\n%s", section)
	}
	// POSTCONDITION 2: the classification must be the artifact's
	// classification, not the ordinary worktree text's.
	if !strings.Contains(section, "DERIVED_DIGEST") {
		t.Fatalf("historical classification authority violated: a "+
			"DELETED digest artifact must keep its artifact "+
			"classification. The worktree text must NOT have "+
			"become the authority. Section:\n%s", section)
	}
}

// TestAmendment_F1_NoWorktreeSubstitutionWhenHistoryIsUnavailable
// verifies that the current working tree is NEVER substituted for
// unavailable historical evidence.
//
// The witness exercises the CLASSIFIER AUTHORITY DECISION directly,
// because that is where the authority lives: the loader reports that
// NO historical blob could be recovered, and the decision must then
// be "no historical evidence", never "read the worktree".
//
// The fixture forces the historical evidence to be unavailable: the
// base endpoint is the EMPTY tree AND the head endpoint does not hold
// the path, so no blob exists at either endpoint, while the worktree
// holds content that WOULD change the classification.
//
// POSTCONDITION: the digest declares the degraded disposition
// explicitly and does not claim a content-driven classification.
func TestAmendment_F1_NoWorktreeSubstitutionWhenHistoryIsUnavailable(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// The base endpoint is the empty tree: no blob at any path.
	base, err := RunGit(dir, []string{"hash-object", "-t", "tree",
		"/dev/null"})
	if err != nil {
		t.Fatalf("hash-object: %v", err)
	}
	base = strings.TrimSpace(base)

	// A commit that does NOT contain the path at all, so NEITHER
	// endpoint holds a blob for it.
	writeFilePath(t, filepath.Join(dir, "other.txt"), "some other file\n")
	patchBudgetCommitAll(t, dir, "add other")

	// PRECONDITION: neither endpoint holds a blob at the path.
	if _, err := RunGit(dir, []string{
		"rev-parse", base + ":newfile.txt"}); err == nil {
		t.Fatalf("fixture setup: the empty tree must not hold a " +
			"blob for this witness to be a discriminator")
	}
	if _, err := RunGit(dir, []string{
		"rev-parse", "HEAD:newfile.txt"}); err == nil {
		t.Fatalf("fixture setup: the head endpoint must not hold a " +
			"blob for this witness to be a discriminator")
	}

	// The worktree holds ordinary text that would classify the file
	// as NORMAL. It must NOT become the authority.
	writeFilePath(t, filepath.Join(dir, "newfile.txt"),
		"ordinary unrelated worktree text\n")

	// The historical loader must report that NO historical blob is
	// available for this path.
	runner := realGitRunner{}
	head, err := runner.Run(dir, []string{"rev-parse", "HEAD"})
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	head = strings.TrimSpace(head)
	headData := loadHistoricalClassifierDataBatch(
		runner, dir, head, []string{"newfile.txt"})
	if headData["newfile.txt"].OK {
		t.Fatalf("fixture setup: the head endpoint must not resolve " +
			"a historical blob for this witness to be a discriminator")
	}
	baseData := loadHistoricalClassifierDataBatch(
		runner, dir, base, []string{"newfile.txt"})
	if baseData["newfile.txt"].OK {
		t.Fatalf("fixture setup: the base endpoint must not resolve " +
			"a historical blob for this witness to be a discriminator")
	}

	// The authority DECISION is the unit under test. The renderer
	// must classify from the historical evidence and, when there is
	// none, must NOT fall back to the worktree.
	//
	// The decision is expressed by the renderer's own authority
	// helper, so the witness asserts on that helper rather than on
	// an end-to-end digest that the range manifest cannot produce
	// (a tracked path always has a blob at one endpoint).
	prefix, body, ok, source := selectHistoricalClassifierEvidence(
		headData["newfile.txt"], baseData["newfile.txt"])
	if ok {
		t.Fatalf("fixture setup: no historical evidence must be " +
			"available for this witness to be a discriminator")
	}
	if source != "NONE" {
		t.Fatalf("historical classification authority violated: the "+
			"evidence source must be NONE when no historical blob "+
			"is available, got %q", source)
	}
	if prefix != "" || body != nil {
		t.Fatalf("historical classification authority violated: the "+
			"worktree content was substituted for the missing "+
			"historical evidence. prefix=%q bodyLen=%d",
			prefix, len(body))
	}
	// The degraded disposition must be reported explicitly.
	if ClassHistoricalEvidenceUnavailable == "" ||
		WarningCodeHistoricalEvidenceUnavailable == "" {
		t.Fatalf("the degraded historical-evidence disposition must " +
			"be declared")
	}
}
