// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_render_range.go is the range-mode
// half of the bounded evidence renderer.
//
// It was split out of evidence_budget_render_render.go by
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 so that both files stay
// within the LLM-friendly 400-line limit.
//
// PATCH BUDGET AUTHORITY (ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01)
//
// The per-file evidence budget applies to the ACTUAL UNIFIED PATCH,
// never to the size of the underlying source blob.
//
// The predecessor of this renderer computed an "EffectiveSize" from
// the working-tree file size and the historical endpoint blob sizes,
// and suppressed the whole file body whenever that number exceeded
// MaxPerFileBytes. The decision was made BEFORE the patch was
// generated, so a 1.2 MB source file carrying a 623-byte one-line
// patch was suppressed with LARGE_FILE_EVIDENCE_BOUNDED and the
// reviewer never saw the change.
//
// The correction:
//
//   - The ACTUAL unified patch is generated FIRST.
//   - The per-file budget is applied to the patch BYTES.
//   - Blob size is retained as metadata (BlobSize) but NEVER
//     decides inclusion.
//   - A patch that does fit the budget is included IN FULL.
//   - A patch that does NOT fit the budget yields an explicit
//     incomplete-evidence disposition with a deterministic
//     retrieval path. A patch is never silently truncated and
//     represented as complete.
//   - Content-driven dispositions (digest self-output, recognised
//     digest artifact, structural recursion) keep their existing
//     policy and remain decided before patch generation, because
//     they describe the artifact's identity rather than its size.
//
// The aggregate (total) render budget, the binary disposition, the
// redaction policy and the base/head blob identities are unchanged.
package digest

import (
	"os"
	"path/filepath"
)

// renderRangeFileEvidenceBoundedWithRunner is the bounded-policy
// range-mode replacement.
func renderRangeFileEvidenceBoundedWithRunner(
	runner gitRunner, repoRoot string, files []RangeFile,
	rangeSpec string, outputAbs string,
) string {
	bw := newBoundedWriter(MaxTotalRenderBytes, MaxPerFileBytes)
	bw.reserveTail(terminationRecordBudget)
	bw.reservePerFileMarker(perFileMarkerBudget)
	bw.beginFile()

	appendBlock(bw, "## Changed files\n")
	if len(files) == 0 {
		appendBlock(bw, "No changed files found in range.\n")
	} else {
		for _, f := range files {
			appendBlock(bw, "%s  [%s]\n",
				PathEscape(f.Path), f.Status)
		}
	}
	appendBlock(bw, "\n")

	appendBlock(bw, "## Diffs\n")
	if len(files) == 0 {
		appendBlock(bw, "No diffs to show.\n")
		return bw.String()
	}

	rangeMaxBytes := computeRangeMaxBytes(runner, repoRoot,
		files, rangeSpec)
	// F14 (CORRECTION03): for three-dot ranges, anchor
	// the classification left endpoint to merge-base so
	// divergent histories with a large merge-base blob
	// still trigger the bounded policy. The original
	// rangeSpec is still passed to git diff downstream.
	left, right, _ := resolveRangeEndpoints(runner, repoRoot, rangeSpec)

	// AMENDMENT-01 (A2): resolve the comparison endpoints to
	// IMMUTABLE object IDs ONCE for the whole range. The
	// resolution is a pair of Git invocations; doing it per file
	// would add two subprocesses for every changed file.
	//
	// The resolved endpoints are the single authority for both the
	// generated patches and the published retrieval commands.
	immutableBase, immutableHead, _, _, dotDot :=
		resolvePatchEndpoints(runner, repoRoot, rangeSpec, left, right)

	// AMENDMENT-01 (A2): publish ONE canonical retrieval command
	// for the WHOLE range, right after the Diffs heading.
	//
	// The range-level retrieval command reproduces every patch the
	// range contains. It is the SAME command the renderer itself
	// uses for the comparison, so the reviewer can always
	// reconstruct the complete evidence independently of the
	// per-file budget.
	//
	// The per-file retrieval paths are published only for the
	// BOUNDED dispositions, where the embedded patch is absent and
	// the reviewer needs a way to obtain that one file's patch.
	// Publishing a retrieval path for every included file would add
	// roughly 200 bytes per file to the digest and would materially
	// weaken the aggregate render budget for large ranges.
	appendBlock(bw, "RangeRetrievalPath: %s\n",
		formatRangeRetrieval(immutableBase, immutableHead, dotDot))

	// F13+F15: batch-lookup range blob OIDs in one Git
	// process so the F13 historical-identity emission
	// does not regress multi-file performance.
	//
	// AMENDMENT-01 (A3): the BASE endpoint lookup for a rename
	// or copy MUST use the SOURCE path. The preimage of
	// "old.ts -> new.ts" lives at "<base>:old.ts"; looking it up
	// at "<base>:new.ts" reports a MISSING blob even though the
	// true preimage is present. The HEAD endpoint keeps using
	// the destination path.
	refs := make([]string, 0, 2*len(files))
	for _, f := range files {
		refs = append(refs, left+":"+rangeBaseIdentityPath(f))
		refs = append(refs, right+":"+f.Path)
	}
	rangeBlobOIDMap := rangeBlobOIDsBatch(runner, repoRoot, refs)

	// AMENDMENT-01 (A4): classify the HISTORICAL blobs of the range
	// in ONE batched Git process. The per-file reading would spawn
	// one subprocess per changed file.
	//
	// AMENDMENT-01 FINALIZATION (F1): the classifier needs the
	// historical blob of BOTH endpoints.
	//
	// The HEAD endpoint describes the artifact the range PRODUCES.
	// The BASE endpoint describes the artifact the range CONSUMED.
	// A DELETION has no head blob at all, so the head endpoint
	// alone cannot classify it; its classification authority is the
	// BASE preimage.
	//
	// Both batches are loaded. The head batch is authoritative when
	// it resolves; the base batch is the authority for a file whose
	// head blob is absent (a deletion) and for a rename or copy
	// (whose preimage lives at the source path).
	histPaths := make([]string, 0, len(files))
	baseHistPaths := make([]string, 0, len(files))
	for _, f := range files {
		histPaths = append(histPaths, f.Path)
		baseHistPaths = append(baseHistPaths, rangeBaseIdentityPath(f))
	}
	historicalClassifier := loadHistoricalClassifierDataBatch(
		runner, repoRoot, immutableHead, histPaths)
	baseHistoricalClassifier := loadHistoricalClassifierDataBatch(
		runner, repoRoot, immutableBase, baseHistPaths)

	var omitted []string
	for _, f := range files {
		if bw.Exhausted() {
			omitted = append(omitted, f.Path)
			continue
		}

		// F11: reset per-file accumulator at the start
		// of every file. Without this, the per-file
		// budget accumulates across the whole loop and
		// clamps the render at 64 KiB (per-file cap)
		// instead of clamping per file.
		bw.beginFile()

		fullPath := filepath.Join(repoRoot, f.Path)

		var worktreeSize int64
		if info, err := os.Stat(fullPath); err == nil {
			worktreeSize = info.Size()
		}

		// BlobSize is the size of the underlying source blob
		// (working tree or historical endpoint). It is
		// METADATA ONLY: it never decides whether the
		// unified patch is included. See the patch-budget
		// authority block below.
		blobSize := worktreeSize
		if rs, ok := rangeMaxBytes[f.Path]; ok && rs > blobSize {
			blobSize = rs
		}

		// F22 (CORRECTION05): the predecessor classifier read the
		// CURRENT worktree file, not the historical blob. The
		// classification of a deleted/replaced historical artifact
		// was therefore not proven.
		//
		// AMENDMENT-01 (A4): the range-mode classifier now reads the
		// HISTORICAL blob at the head endpoint. The disposition of
		// an immutable historical range must not change merely
		// because the working tree changed.
		//
		// AMENDMENT-01 FINALIZATION (F1): the HEAD endpoint alone
		// is NOT sufficient authority. A DELETION has no head blob,
		// so the predecessor fell back to the CURRENT WORKTREE and
		// the original authority defect returned for exactly the
		// case it was meant to close.
		//
		// The authority is now:
		//
		//   1. the HEAD blob, when it resolves (the artifact the
		//      range produces);
		//   2. otherwise the BASE preimage (the artifact the range
		//      consumed) -- this covers a DELETION and a RENAME;
		//   3. otherwise NO historical evidence is available. The
		//      working tree is NOT substituted for it: a mutable
		//      state must never become the classification
		//      authority of an immutable historical range. The
		//      disposition is degraded explicitly instead.
		hist := historicalClassifier[f.Path]
		baseHist := baseHistoricalClassifier[rangeBaseIdentityPath(f)]
		prefix, body, historicalEvidenceAvailable, evidenceSource :=
			selectHistoricalClassifierEvidence(hist, baseHist)
		class := classifyFileEvidence(classifierInput{
			repoRoot:  repoRoot,
			relPath:   f.Path,
			fullPath:  fullPath,
			size:      blobSize,
			rawPrefix: prefix,
			outputAbs: outputAbs,
			bodyBytes: body,
		})
		if !historicalEvidenceAvailable {
			// The historical evidence could not be recovered.
			// The disposition MUST say so instead of silently
			// presenting a guess.
			class = ClassHistoricalEvidenceUnavailable
		}
		if !historicalEvidenceAvailable && blobSize > MaxFileSizeForFull {
			class = ClassBoundedBody
		}

		// Content-driven dispositions (digest self-output,
		// recognised digest artifact, structural recursion)
		// are decided BEFORE patch generation: they describe
		// the artifact's identity, not its size. Their policy
		// is unchanged by this ACT.
		if class == ClassBoundedSelfOutput ||
			class == ClassBoundedRecursive ||
			class == ClassBoundedDerivedDigest {
			renderRangeIdentityDisposition(bw, repoRoot, f,
				class, blobSize, rangeBlobOIDMap, left, right,
				evidenceSource)
			continue
		}

		// Generate the ACTUAL unified patch FIRST. The
		// per-file budget is applied to these bytes.
		//
		// AMENDMENT-01 (A2): the patch is produced by the CANONICAL
		// PatchSpec, the very same specification whose
		// serialization is published as the retrieval path. The
		// generation and the retrieval commands therefore cannot
		// drift apart.
		spec := newPatchSpec(immutableBase, immutableHead,
			"", "", dotDot, f)
		patch, patchErr := runner.Run(repoRoot, spec.Args())
		patchBytes := len(patch)

		if patchErr != nil {
			// A diff ERROR is not a budget disposition. Keep
			// the pre-existing degraded-evidence contract:
			// report the failure honestly and never fall
			// back to an empty-tree comparison.
			appendBlock(bw, "\n=== %s ===\n", PathEscape(f.Path))
			appendBlock(bw, "Status: %s\n\n", f.Status)
			appendBlock(bw, "(range diff unavailable: %v)\n", patchErr)
			continue
		}

		if patchBytes > MaxPerFileBytes {
			// The ACTUAL patch does not fit the per-file
			// evidence budget. Emit an explicit
			// INCOMPLETE-EVIDENCE disposition with a
			// deterministic retrieval path. We never
			// truncate a patch and present it as complete.
			renderRangeIncompleteDisposition(bw, f, blobSize,
				patchBytes, rangeBlobOIDMap, left, right, spec)
			continue
		}

		appendBlock(bw, "\n=== %s ===\n", PathEscape(f.Path))
		appendBlock(bw, "Status: %s\n\n", f.Status)
		appendBlock(bw, "PatchBytes: %d\n", patchBytes)
		appendBlock(bw, "BlobSize: %d\n", blobSize)

		// AMENDMENT-01 FINALIZATION (F1): when no historical blob
		// could be recovered at either endpoint, the digest must
		// SAY so. The patch itself is still published, because the
		// patch is the evidence; but no content-driven
		// classification is claimed for it, and the reviewer is
		// told that the classification was degraded rather than
		// silently derived from mutable worktree state.
		if !historicalEvidenceAvailable {
			appendBlock(bw,
				"Classification: %s\n"+
					"WarningCode: %s\n"+
					"HistoricalEvidenceSource: NONE\n"+
					"Note: the historical blob of this file could "+
					"not be recovered at either endpoint of the "+
					"range. The current working tree was NOT "+
					"used as a substitute, because mutable state "+
					"cannot be the classification authority of "+
					"an immutable historical range.\n",
				ClassHistoricalEvidenceUnavailable,
				WarningCodeHistoricalEvidenceUnavailable)
		} else if evidenceSource == "BASE" {
			// The classification was derived from the BASE
			// preimage (a deletion or a rename). The reviewer is
			// told which endpoint carried the authority.
			appendBlock(bw, "HistoricalEvidenceSource: %s\n",
				evidenceSource)
		}

		if patch == "" {
			appendBlock(bw, "(no diff available)\n")
			continue
		}

		// AMENDMENT-01 (A1): the patch FITS the per-file budget,
		// so it must be published COMPLETE. The predecessor
		// applied a per-LINE truncation (truncateLongLines) and
		// then discarded the writer's report, so a patch whose
		// bytes fit could still be silently transformed while
		// the digest reported the original PatchBytes.
		//
		// The per-line cap is a PRESENTATION constraint for
		// unbounded file bodies; for a patch that already fits
		// the per-file budget it would destroy evidence without
		// being disclosed. The patch is therefore emitted
		// verbatim, and any budget-induced partial publication
		// is reported as explicitly INCOMPLETE.
		bw.beginFile()

		// AMENDMENT-01 FINALIZATION (F2): reserve the slice of the
		// TOTAL budget that the completeness disposition needs
		// BEFORE the patch body is written. The patch can then
		// never consume the budget the disposition itself
		// requires, so the disposition is a GUARANTEED
		// publication.
		bw.reserveCompleteness(completenessDispositionBudget)

		n, perFileCapped := bw.appendFileString(patch)
		if perFileCapped {
			// The patch was admitted by the byte budget but
			// the writer's per-file cap cut it. That is a
			// partial publication and MUST be disclosed.
			bw.markPerFileMarkerReserved()
			bw.appendCompleteness(
				"\nEvidenceCompleteness: INCOMPLETE\n"+
					"SuppressionReason: PATCH_TRUNCATED_BY_PER_FILE_CAP\n"+
					"[per-file body cap hit: %d bytes]\n",
				MaxPerFileBytes)
			continue
		}
		if n != len(patch) {
			// The writer dropped bytes without reaching its
			// per-file cap (the TOTAL budget was exhausted).
			// Disclose the partial publication.
			bw.appendCompleteness(
				"\nEvidenceCompleteness: INCOMPLETE\n" +
					"SuppressionReason: PATCH_TRUNCATED_BY_TOTAL_CAP\n")
			continue
		}
		bw.appendCompleteness("EvidenceCompleteness: COMPLETE\n")
	}

	emitOmissionRecord(bw, omitted, "TOTAL_RENDER_BUDGET")
	return bw.String()
}
