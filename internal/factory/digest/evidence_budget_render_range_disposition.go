// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_render_range_disposition.go holds
// the two range-mode disposition renderers used by
// evidence_budget_render_range.go.
//
// They were split into their own file by
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 to keep every file
// within the LLM-friendly 400-line limit.
//
// Both dispositions preserve the base/head Git blob identities and the
// F16 blob-status contract. Neither ever presents a truncated patch
// as if it were complete.
package digest

import "path/filepath"

// renderRangeIdentityDisposition renders the range-mode block for a
// CONTENT-DRIVEN disposition: the digest's own output path, a
// recognised digest artifact, or a structurally recursive artifact.
//
// F17 (CORRECTION04): the decision is rendered directly rather than
// through boundedFileBlock, because boundedFileBlock would
// reclassify against the worktree path (deleted = size=0, small
// replacement = NORMAL) and would drop the historical OID.
//
// F21 (CORRECTION05): the SELF_OUTPUT_EXCLUDED diagnostic is
// preserved in range mode; it was contract-drifted to the generic
// LARGE_FILE_EVIDENCE_BOUNDED.
func renderRangeIdentityDisposition(
	bw *boundedWriter, repoRoot string, f RangeFile,
	class EvidenceClass, blobSize int64,
	rangeBlobOIDMap map[string]RangeBlobResult,
	left, right string, evidenceSource string,
) {
	// AMENDMENT-01 (A3): the base blob identity for a rename or
	// copy is resolved through the SOURCE path, because the
	// preimage lives there.
	baseRes := rangeBlobOIDMap[left+":"+rangeBaseIdentityPath(f)]
	headRes := rangeBlobOIDMap[right+":"+f.Path]
	appendBlock(bw, "\n=== %s ===\n", PathEscape(f.Path))
	appendBlock(bw, "Status: %s\n\n", f.Status)
	warningCode := WarningCodeLargeFileBounded
	bodyNote := "Body: suppressed (historical diff " +
		"size exceeds per-file budget)"
	switch class {
	case ClassBoundedSelfOutput:
		warningCode = WarningCodeSelfOutput
		bodyNote = "Body: suppressed (SELF_OUTPUT_EXCLUDED)"
	case ClassBoundedRecursive:
		warningCode = WarningCodeDigestRecursion
		bodyNote = "Body: suppressed (DIGEST_RECURSION)"
	case ClassBoundedDerivedDigest:
		warningCode = WarningCodeDerivedDigestBounded
		bodyNote = "Body: suppressed (DERIVED_DIGEST_BODY_BOUNDED)"
	}
	// SHA256 for digest-artifact classes: the G6 matrix needs it;
	// worktree identity and Git blob OID are complementary.
	sha256Line := ""
	if class == ClassBoundedRecursive ||
		class == ClassBoundedDerivedDigest {
		sha256Line = "SHA256: " +
			sha256HexFile(filepath.Join(repoRoot, f.Path)) + "\n"
	}
	appendBlock(bw,
		"Classification: %s\n"+
			"EffectiveSize: %d\n"+
			sha256Line+
			"HistoricalEvidenceSource: %s\n"+
			"RangeBaseBlobOID: %s\n"+
			"RangeBaseBlobStatus: %s\n"+
			"RangeHeadBlobOID: %s\n"+
			"RangeHeadBlobStatus: %s\n"+
			"WarningCode: %s\n"+
			"%s\n",
		class, blobSize, sha256Line2(evidenceSource), baseRes.OID,
		baseRes.Status.String(), headRes.OID,
		headRes.Status.String(), warningCode, bodyNote)
}

// sha256Line2 is a tiny helper that keeps the disposition's format
// arguments aligned.
//
// AMENDMENT-01 FINALIZATION (F1): the historical evidence source is
// rendered alongside the classification so the reviewer can tell WHICH
// endpoint carried the classification authority.
func sha256Line2(evidenceSource string) string {
	if evidenceSource == "" {
		return "NONE"
	}
	return evidenceSource
}

// renderRangeEndpointUnavailableDisposition renders the range-mode
// block for a file whose HISTORICAL COMPARISON ENDPOINTS could not be
// resolved.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 FINAL AUTHORITY
// CORRECTION (G3): a `git diff` WITHOUT comparison endpoints does not
// fail. It compares the mutable WORKTREE against the index and exits
// 0, so it produces a SUCCESSFUL comparison against the WRONG
// authority.
//
// The disposition therefore states explicitly that the evidence is
// UNAVAILABLE, and it never presents a comparison against mutable
// state as if it were the historical range.
func renderRangeEndpointUnavailableDisposition(
	bw *boundedWriter, f RangeFile) {
	appendBlock(bw, "\n=== %s ===\n", PathEscape(f.Path))
	appendBlock(bw, "Status: %s\n\n", f.Status)
	appendBlock(bw,
		"EvidenceCompleteness: UNAVAILABLE\n"+
			"SuppressionReason: COMPARISON_ENDPOINTS_UNAVAILABLE\n"+
			"WarningCode: %s\n"+
			"Note: the historical comparison endpoints of "+
			"this range could not be resolved, so no "+
			"patch was executed. A diff without "+
			"comparison endpoints would compare the "+
			"mutable working tree against the index "+
			"and succeed, which is NOT the historical "+
			"range this evidence describes.\n",
		WarningCodeComparisonEndpointsUnavailable)
}

// renderRangeIncompleteDisposition renders the range-mode block for a
// file whose ACTUAL unified patch exceeds the per-file evidence
// budget.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01: the disposition is
// explicitly INCOMPLETE and carries a deterministic retrieval path.
// The patch body is never emitted, so it can never be mistaken for a
// complete body.
//
// AMENDMENT-01 (A2/A3): the retrieval path is the serialization of
// the SAME canonical PatchSpec that produced the measured patch, so
// executing it reproduces the exact patch. The base blob identity is
// resolved through the RENAME SOURCE path.
func renderRangeIncompleteDisposition(
	bw *boundedWriter, f RangeFile, blobSize int64, patchBytes int,
	rangeBlobOIDMap map[string]RangeBlobResult,
	left, right string, spec PatchSpec,
) {
	baseRes := rangeBlobOIDMap[left+":"+rangeBaseIdentityPath(f)]
	headRes := rangeBlobOIDMap[right+":"+f.Path]
	appendBlock(bw, "\n=== %s ===\n", PathEscape(f.Path))
	appendBlock(bw, "Status: %s\n\n", f.Status)
	appendBlock(bw,
		"Classification: BOUNDED_BODY\n"+
			"EvidenceCompleteness: INCOMPLETE\n"+
			"SuppressionReason: PATCH_BYTES_EXCEED_PER_FILE_BUDGET\n"+
			"PatchBytes: %d\n"+
			"BlobSize: %d\n"+
			"RangeBaseBlobOID: %s\n"+
			"RangeBaseBlobStatus: %s\n"+
			"RangeHeadBlobOID: %s\n"+
			"RangeHeadBlobStatus: %s\n"+
			"WarningCode: %s\n"+
			"RetrievalPath: %s\n",
		patchBytes, blobSize,
		baseRes.OID, baseRes.Status.String(),
		headRes.OID, headRes.Status.String(),
		WarningCodeLargeFileBounded,
		spec.RetrievalPathspecEscaped())
}
