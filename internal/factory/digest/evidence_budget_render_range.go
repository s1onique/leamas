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

	// F13+F15: batch-lookup range blob OIDs in one Git
	// process so the F13 historical-identity emission
	// does not regress multi-file performance.
	refs := make([]string, 0, 2*len(files))
	for _, f := range files {
		refs = append(refs, left+":"+f.Path)
		refs = append(refs, right+":"+f.Path)
	}
	rangeBlobOIDMap := rangeBlobOIDsBatch(runner, repoRoot, refs)

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

		// F22 (CORRECTION05): classifier reads the
		// CURRENT worktree file, not the historical
		// blob. Digest-recursion classification of
		// deleted/replaced historical artifacts is
		// therefore not proven — body boundedness IS
		// proven. Follow-up: git show <ref>:<path>
		// prefix loader.
		prefix, body, ok := loadClassifierData(fullPath)
		class := classifyFileEvidence(classifierInput{
			repoRoot:  repoRoot,
			relPath:   f.Path,
			fullPath:  fullPath,
			size:      blobSize,
			rawPrefix: prefix,
			outputAbs: outputAbs,
			bodyBytes: body,
		})
		if !ok && blobSize > MaxFileSizeForFull {
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
				class, blobSize, rangeBlobOIDMap, left, right)
			continue
		}

		// Generate the ACTUAL unified patch FIRST. The
		// per-file budget is applied to these bytes.
		patch, patchErr := runner.Run(repoRoot, rangeDiffArgs(rangeSpec, f))
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
				patchBytes, rangeBlobOIDMap, left, right, rangeSpec)
			continue
		}

		appendBlock(bw, "\n=== %s ===\n", PathEscape(f.Path))
		appendBlock(bw, "Status: %s\n\n", f.Status)
		appendBlock(bw, "PatchBytes: %d\n", patchBytes)
		appendBlock(bw, "BlobSize: %d\n", blobSize)

		if patch == "" {
			appendBlock(bw, "(no diff available)\n")
			continue
		}

		bw.beginFile()
		patchBody := truncateLongLines(patch)
		n, perFileCapped := bw.appendFileString(patchBody)
		_ = n
		if perFileCapped {
			bw.markPerFileMarkerReserved()
			appendBlock(bw, "\n[per-file body cap hit: %d bytes]\n",
				MaxPerFileBytes)
		}
	}

	emitOmissionRecord(bw, omitted, "TOTAL_RENDER_BUDGET")
	return bw.String()
}
