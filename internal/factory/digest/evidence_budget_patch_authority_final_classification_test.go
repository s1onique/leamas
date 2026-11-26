// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_final_classification_test.go
// holds the G1 historical-classification-conservation witnesses for
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 FINAL AUTHORITY CORRECTION.
//
// INDEPENDENTLY REPRODUCED DEFECT: the historical classifier selected
// the HEAD endpoint whenever it resolved. A range that REPLACED a
// historical digest artifact with ordinary text therefore lost the
// preimage's artifact classification: the artifact-exclusion policy
// describes the artifact's IDENTITY, and the range still contains that
// artifact as its preimage.
//
// The companion files hold the A1/A2, A3/A4, F1 and G2/G3 witnesses.
package digest

import (
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// G1: historical classification conservation.
// ---------------------------------------------------------------------------

// TestAmendment_G1_ReplacedArtifactKeepsItsArtifactClassification
// verifies that a range which REPLACES a historical digest artifact with
// ordinary text still carries the artifact-exclusion classification.
//
// The artifact-exclusion policy (BOUNDED_DERIVED_DIGEST and
// BOUNDED_RECURSIVE) describes the artifact's IDENTITY, not the size of
// the bytes that happen to be at the path today. A range whose base
// endpoint holds a digest artifact and whose head endpoint holds
// ordinary text is a range that CONSUMED a digest artifact, so the
// exclusion must survive.
//
// The witness REUSES the existing historical-classification witness
// shape: the same recognised-artifact fixture, the same worktree
// contamination, the same "HistoricalEvidenceSource" assertion.
func TestAmendment_G1_ReplacedArtifactKeepsItsArtifactClassification(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// A recognised Leamas digest artifact at the BASE endpoint.
	artifact := "# Targeted digest\n\n" +
		"LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n" +
		"ordinary artifact body\n"
	writeFilePath(t, filepath.Join(dir, "replaced-artifact.txt"), artifact)
	patchBudgetCommitAll(t, dir, "add artifact")

	// The artifact is REPLACED by ordinary text at the HEAD endpoint.
	writeFilePath(t, filepath.Join(dir, "replaced-artifact.txt"),
		"ordinary replacement text\n")
	patchBudgetCommitAll(t, dir, "replace artifact")

	// PRECONDITION: BOTH endpoints hold a blob at that path. The head
	// endpoint holds ordinary text and the base endpoint holds the
	// artifact. This is the case the predecessor lost.
	headOut, headErr := RunGit(dir, []string{
		"rev-parse", "HEAD:replaced-artifact.txt"})
	if headErr != nil {
		t.Fatalf("fixture setup: the head endpoint must hold a blob: %v",
			headErr)
	}
	baseOut, baseErr := RunGit(dir, []string{
		"rev-parse", "HEAD~1:replaced-artifact.txt"})
	if baseErr != nil {
		t.Fatalf("fixture setup: the base endpoint must hold a blob: %v",
			baseErr)
	}
	if strings.TrimSpace(headOut) == strings.TrimSpace(baseOut) {
		t.Fatalf("fixture setup: the two endpoints must hold " +
			"DIFFERENT blobs")
	}
	// PRECONDITION: the head endpoint really holds ordinary text.
	headBody, err := RunGit(dir, []string{
		"cat-file", "blob", "HEAD:replaced-artifact.txt"})
	if err != nil {
		t.Fatalf("fixture setup: read the head blob: %v", err)
	}
	if strings.Contains(headBody,
		"LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION:") {
		t.Fatalf("fixture setup: the head endpoint must hold ORDINARY " +
			"text for this witness to be a discriminator")
	}

	d := patchBudgetRangeDigest(t, dir, "HEAD~1..HEAD")
	section := patchBudgetFileSection(t, d, "replaced-artifact.txt")

	// POSTCONDITION: the artifact-exclusion classification survived.
	if !strings.Contains(section, "DERIVED_DIGEST") {
		t.Fatalf("historical classification conservation violated: a "+
			"range that replaced a digest artifact with ordinary "+
			"text lost the preimage's artifact classification. The "+
			"artifact-exclusion policy describes the artifact's "+
			"identity, and the range still contains it as its "+
			"preimage. Section:\n%s", section)
	}
}

// TestAmendment_G1_EndpointAbsenceIsNotUnavailableEvidence verifies that
// a genuinely ABSENT endpoint object is distinguished from UNAVAILABLE
// evidence.
//
// The two conditions have different meanings:
//
//   - ABSENT:      the endpoint object does not exist (a deletion, or a
//     path the endpoint never held). The other endpoint is the
//     authority.
//   - UNAVAILABLE: the endpoint object exists but its content could not
//     be recovered (for example it exceeded the capture limit). The
//     evidence is degraded and must be reported as such.
//
// The witness exercises the SELECTOR directly, because that is where
// the distinction lives.
func TestAmendment_G1_EndpointAbsenceIsNotUnavailableEvidence(t *testing.T) {
	artifact := HistoricalClassifierData{
		Prefix: "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n",
		Body:   []byte("LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n"),
		OK:     true,
	}
	ordinary := HistoricalClassifierData{
		Prefix: "ordinary text\n",
		Body:   []byte("ordinary text\n"),
		OK:     true,
	}

	// The HEAD endpoint resolves and holds the artifact: the HEAD
	// endpoint carries the authority.
	_, _, ok, source := selectHistoricalClassifierEvidence(artifact, ordinary)
	if !ok || source != "HEAD" {
		t.Fatalf("the HEAD endpoint must carry the authority when it "+
			"resolves, got ok=%v source=%q", ok, source)
	}

	// The HEAD endpoint is ABSENT (not OK) and the BASE endpoint
	// resolves: the BASE endpoint carries the authority, and the
	// evidence is AVAILABLE.
	_, _, ok, source = selectHistoricalClassifierEvidence(
		HistoricalClassifierData{}, artifact)
	if !ok || source != "BASE" {
		t.Fatalf("an ABSENT head endpoint must yield the BASE "+
			"authority with available evidence, got ok=%v source=%q",
			ok, source)
	}

	// NEITHER endpoint resolves: no historical evidence is available.
	// The worktree is never substituted.
	_, _, ok, source = selectHistoricalClassifierEvidence(
		HistoricalClassifierData{}, HistoricalClassifierData{})
	if ok || source != "NONE" {
		t.Fatalf("no available endpoint must yield unavailable "+
			"evidence with source NONE, got ok=%v source=%q",
			ok, source)
	}
}

// TestAmendment_G1_ArtifactPolicyAppliesToBothEndpoints verifies that
// the artifact-exclusion policy is applied to BOTH relevant endpoints.
//
// The policy decides whether the range CONSUMED a digest artifact. The
// HEAD endpoint alone cannot answer that: a range that replaced an
// artifact with ordinary text has an ordinary HEAD blob and an
// artifact BASE blob.
//
// The witness exercises the CLASSIFIER SELECTOR directly.
func TestAmendment_G1_ArtifactPolicyAppliesToBothEndpoints(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// A recognised artifact at the BASE endpoint, ordinary text at the
	// HEAD endpoint.
	artifact := "# Targeted digest\n\n" +
		"LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3\n" +
		"ordinary artifact body\n"
	writeFilePath(t, filepath.Join(dir, "p.txt"), artifact)
	patchBudgetCommitAll(t, dir, "add artifact")
	writeFilePath(t, filepath.Join(dir, "p.txt"), "ordinary text\n")
	patchBudgetCommitAll(t, dir, "replace artifact")

	base, head, _, _, _ := resolvePatchEndpoints(realGitRunner{}, dir,
		"HEAD~1..HEAD", "", "")
	if base == "" || head == "" {
		t.Fatalf("fixture setup: the endpoints must resolve")
	}

	headData := loadHistoricalClassifierDataBatch(realGitRunner{}, dir,
		head, []string{"p.txt"})
	baseData := loadHistoricalClassifierDataBatch(realGitRunner{}, dir,
		base, []string{"p.txt"})

	headRec := headData["p.txt"]
	baseRec := baseData["p.txt"]
	if !headRec.OK || !baseRec.OK {
		t.Fatalf("fixture setup: both endpoints must be available")
	}

	// The BASE endpoint classifies as an artifact.
	baseClass := classifyFileEvidence(classifierInput{
		relPath:   "p.txt",
		size:      int64(len(baseRec.Body)),
		rawPrefix: baseRec.Prefix,
		bodyBytes: baseRec.Body,
	})
	if baseClass != ClassBoundedDerivedDigest {
		t.Fatalf("fixture setup: the BASE endpoint must classify as "+
			"%s, got %s", ClassBoundedDerivedDigest, baseClass)
	}

	// The HEAD endpoint classifies as ordinary text.
	headClass := classifyFileEvidence(classifierInput{
		relPath:   "p.txt",
		size:      int64(len(headRec.Body)),
		rawPrefix: headRec.Prefix,
		bodyBytes: headRec.Body,
	})
	if headClass != ClassNormal {
		t.Fatalf("fixture setup: the HEAD endpoint must classify as "+
			"%s, got %s", ClassNormal, headClass)
	}

	// The CONSERVED classification must be the artifact's, because the
	// range consumed the artifact.
	conserved := selectHistoricalClassifierClass(headClass, baseClass)
	if conserved != ClassBoundedDerivedDigest {
		t.Fatalf("historical classification conservation violated: the "+
			"artifact-exclusion classification of the BASE endpoint "+
			"was lost. head=%s base=%s conserved=%s",
			headClass, baseClass, conserved)
	}
}
