// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_render_render.go contains the
// full-section renderers that walk a list of changed files.
//
// The companion files are:
//   - evidence_budget.go               (policy, classifier)
//   - evidence_budget_render.go        (boundedWriter, boundedFileBlock)
//   - evidence_budget_render_helpers.go (range-stat, line truncation)
//
// This file contains:
//   - renderChangedFilesAndDiffsBounded
//   - renderRangeFileEvidenceBoundedWithRunner
//
// F10/F11/F12 (CORRECTION02): consumers reserve a tail
// budget (reserveTail) and a per-file marker budget
// (reservePerFileMarker) so the bounded-writer can always
// emit a closure record and a truncation marker, even when
// the total budget is exhausted.
package digest

import (
	"os"
	"path/filepath"
)

// renderChangedFilesAndDiffsBounded is the bounded-policy
// replacement for RenderChangedFilesAndDiffs.
func renderChangedFilesAndDiffsBounded(
	repoRoot string, files []ChangedFile, outputAbs string,
) string {
	bw := newBoundedWriter(MaxTotalRenderBytes, MaxPerFileBytes)
	bw.reserveTail(terminationRecordBudget)
	bw.reservePerFileMarker(perFileMarkerBudget)
	bw.beginFile()

	appendBlock(bw, "## Changed files\n")
	if len(files) == 0 {
		appendBlock(bw, "No changed files found.\n")
	} else {
		for _, f := range files {
			kindStr := string(f.Kind)
			if f.Untracked {
				kindStr = StatusUntracked
			}
			escapedPath := PathEscape(f.Path)
			if f.Tracked {
				stagedStr := "no"
				if f.StagedPresent {
					stagedStr = "yes"
				}
				unstagedStr := "no"
				if f.UnstagedPresent {
					unstagedStr = "yes"
				}
				if f.OldPath != "" && f.OldPath != f.Path {
					appendBlock(bw, "%s  [tracked, kind: %s, "+
						"staged present: %s, "+
						"unstaged present: %s, "+
						"old path: %s]\n",
						escapedPath, kindStr,
						stagedStr, unstagedStr,
						PathEscape(f.OldPath))
				} else {
					appendBlock(bw, "%s  [tracked, kind: %s, "+
						"staged present: %s, "+
						"unstaged present: %s]\n",
						escapedPath, kindStr,
						stagedStr, unstagedStr)
				}
			} else {
				appendBlock(bw, "%s  [untracked, staged present: "+
					"no, unstaged present: yes]\n",
					escapedPath)
			}
		}
	}
	appendBlock(bw, "\n")

	appendBlock(bw, "## Diffs\n")
	if len(files) == 0 {
		appendBlock(bw, "No diffs to show.\n")
		return bw.String()
	}

	var omitted []string
	for _, f := range files {
		if bw.Exhausted() {
			omitted = append(omitted, f.Path)
			continue
		}

		fullPath := filepath.Join(repoRoot, f.Path)

		var size int64
		if info, err := os.Stat(fullPath); err == nil {
			size = info.Size()
		}
		prefix, body, ok := loadClassifierData(fullPath)
		class := classifyFileEvidence(classifierInput{
			repoRoot:  repoRoot,
			relPath:   f.Path,
			fullPath:  fullPath,
			size:      size,
			rawPrefix: prefix,
			outputAbs: outputAbs,
			bodyBytes: body,
		})
		if !ok && size > 0 && size > MaxFileSizeForFull {
			class = ClassBoundedBody
		}

		if class != ClassNormal {
			boundedFileBlock(bw, repoRoot, f.Path, outputAbs)
			continue
		}

		appendBlock(bw, "\n=== %s ===\n", PathEscape(f.Path))

		if bw.Exhausted() {
			omitted = append(omitted, f.Path)
			continue
		}
		if f.Tracked {
			stagedStr := "yes"
			if !f.StagedPresent {
				stagedStr = "no"
			}
			unstagedStr := "yes"
			if !f.UnstagedPresent {
				unstagedStr = "no"
			}
			appendBlock(bw, "Metadata: tracked, staged present: "+
				"%s, unstaged present: %s\n",
				stagedStr, unstagedStr)
		}
		appendBlock(bw, "\n")

		if f.Untracked {
			appendBlock(bw, "--- untracked file content ---\n")
			content, isBinary := ReadFileFull(fullPath)
			if isBinary {
				appendBlock(bw, "(binary file)\n")
				continue
			}
			bw.beginFile()
			body := truncateLongLines(content)
			n, perFileCapped := bw.appendFileString(body)
			_ = n
			if perFileCapped {
				bw.markPerFileMarkerReserved()
				appendBlock(bw, "\n[per-file body cap hit: %d bytes]\n",
					MaxPerFileBytes)
			}
		} else {
			if f.StagedPresent {
				appendBlock(bw, "--- staged diff ---\n")
				diff, err := RunGit(repoRoot, []string{
					"diff", "--cached", "--", f.Path})
				if err == nil && diff != "" {
					bw.beginFile()
					body := truncateLongLines(diff)
					n, perFileCapped := bw.appendFileString(body)
					_ = n
					if perFileCapped {
						bw.markPerFileMarkerReserved()
						appendBlock(bw, "\n[per-file body cap hit: %d bytes]\n",
							MaxPerFileBytes)
					}
				} else if err != nil {
					appendBlock(bw, "(staged diff unavailable: %v)\n", err)
				}
			}
			if f.UnstagedPresent {
				appendBlock(bw, "--- unstaged diff ---\n")
				diff, err := RunGit(repoRoot, []string{
					"diff", "--", f.Path})
				if err == nil && diff != "" {
					bw.beginFile()
					body := truncateLongLines(diff)
					n, perFileCapped := bw.appendFileString(body)
					_ = n
					if perFileCapped {
						bw.markPerFileMarkerReserved()
						appendBlock(bw, "\n[per-file body cap hit: %d bytes]\n",
							MaxPerFileBytes)
					}
				} else if err != nil {
					appendBlock(bw, "(unstaged diff unavailable: %v)\n", err)
				}
			}
		}
	}

	emitOmissionRecord(bw, omitted, "TOTAL_RENDER_BUDGET")
	return bw.String()
}

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

		effectiveSize := worktreeSize
		if rs, ok := rangeMaxBytes[f.Path]; ok && rs > effectiveSize {
			effectiveSize = rs
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
			size:      effectiveSize,
			rawPrefix: prefix,
			outputAbs: outputAbs,
			bodyBytes: body,
		})
		if !ok && effectiveSize > MaxFileSizeForFull {
			class = ClassBoundedBody
		}

		if class != ClassNormal {
			// F17 (CORRECTION04): render the range
			// decision directly; calling
			// boundedFileBlock() here would
			// reclassify against the worktree path
			// (deleted = size=0, small replacement
			// = NORMAL) and drop the historical OID.
			baseRes := rangeBlobOIDMap[left+":"+f.Path]
			headRes := rangeBlobOIDMap[right+":"+f.Path]
			appendBlock(bw, "\n=== %s ===\n", PathEscape(f.Path))
			appendBlock(bw, "Status: %s\n\n", f.Status)
			warningCode := WarningCodeLargeFileBounded
			bodyNote := "Body: suppressed (historical diff " +
				"size exceeds per-file budget)"
			switch class {
			case ClassBoundedSelfOutput:
				// F21 (CORRECTION05): preserve
				// SELF_OUTPUT_EXCLUDED diagnostic in
				// range mode (was contract-drifted
				// to LARGE_FILE_EVIDENCE_BOUNDED).
				warningCode = WarningCodeSelfOutput
				bodyNote = "Body: suppressed (SELF_OUTPUT_EXCLUDED)"
			case ClassBoundedRecursive:
				warningCode = WarningCodeDigestRecursion
				bodyNote = "Body: suppressed (DIGEST_RECURSION)"
			case ClassBoundedDerivedDigest:
				warningCode = WarningCodeDerivedDigestBounded
				bodyNote = "Body: suppressed " +
					"(DERIVED_DIGEST_BODY_BOUNDED)"
			}
			// SHA256 for digest-artifact classes:
			// G6 matrix needs it; worktree identity
			// and Git blob OID are complementary.
			sha256Line := ""
			if class == ClassBoundedRecursive ||
				class == ClassBoundedDerivedDigest {
				sha256Line = "SHA256: " +
					sha256HexFile(fullPath) + "\n"
			}
			appendBlock(bw,
				"Classification: %s\n"+
					"EffectiveSize: %d\n"+
					sha256Line+
					"RangeBaseBlobOID: %s\n"+
					"RangeBaseBlobStatus: %s\n"+
					"RangeHeadBlobOID: %s\n"+
					"RangeHeadBlobStatus: %s\n"+
					"WarningCode: %s\n"+
					"%s\n",
				class, effectiveSize, baseRes.OID,
				baseRes.Status.String(), headRes.OID,
				headRes.Status.String(), warningCode, bodyNote)
			continue
		}

		appendBlock(bw, "\n=== %s ===\n", PathEscape(f.Path))
		appendBlock(bw, "Status: %s\n\n", f.Status)
		if effectiveSize > MaxFileSizeForFull ||
			rangeMaxBytes[f.Path] > MaxPerFileBytes {
			// F13+F16 (CORRECTION02-04): historical Git
			// blob OID via batched cat-file; OID field
			// populated only when Status == PRESENT.
			baseRes := rangeBlobOIDMap[left+":"+f.Path]
			headRes := rangeBlobOIDMap[right+":"+f.Path]
			appendBlock(bw,
				"Classification: BOUNDED_BODY\n"+
					"Status: body suppressed (bounded evidence)\n"+
					"EffectiveSize: %d\n"+
					"RangeBaseBlobOID: %s\n"+
					"RangeBaseBlobStatus: %s\n"+
					"RangeHeadBlobOID: %s\n"+
					"RangeHeadBlobStatus: %s\n"+
					"WarningCode: %s\n"+
					"Body: suppressed (historical diff "+
					"size exceeds per-file budget)\n",
				effectiveSize,
				baseRes.OID, baseRes.Status.String(),
				headRes.OID, headRes.Status.String(),
				WarningCodeLargeFileBounded)
			continue
		}

		diff, err := runner.Run(repoRoot, []string{
			"diff", "--unified=3", rangeSpec, "--", f.Path})
		if err == nil && diff != "" {
			bw.beginFile()
			body := truncateLongLines(diff)
			n, perFileCapped := bw.appendFileString(body)
			_ = n
			if perFileCapped {
				bw.markPerFileMarkerReserved()
				appendBlock(bw, "\n[per-file body cap hit: %d bytes]\n",
					MaxPerFileBytes)
			}
		} else if err != nil {
			appendBlock(bw, "(range diff unavailable: %v)\n", err)
		} else {
			appendBlock(bw, "(no diff available)\n")
		}
	}

	emitOmissionRecord(bw, omitted, "TOTAL_RENDER_BUDGET")
	return bw.String()
}
