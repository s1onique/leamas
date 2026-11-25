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
