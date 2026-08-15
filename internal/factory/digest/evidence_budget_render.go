// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_render.go is the rendering half
// of the bounded evidence renderer
// (ACT-LEAMAS-TARGETED-DIGEST-RECURSIVE-EVIDENCE-GUARD01
// CORRECTION01).
//
// The companion file evidence_budget.go provides the policy
// (classifyFileEvidence, budgets, helpers). This file contains:
//
//   - boundedWriter              (per-file + total byte budgets)
//   - boundedFileBlock           (per-file block for all classes)
//   - renderChangedFilesAndDiffsBounded      (full Changed files + Diffs)
//   - renderRangeFileEvidenceBoundedWithRunner (range mode variant)
//   - truncateLongLines          (per-line cap helper)
//
// Splitting across two files keeps each file within the LLM-friendly
// 400-line limit.
package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// boundedWriter enforces the per-file and total byte budgets
// declared in evidence_budget.go (MaxPerFileBytes,
// MaxTotalRenderBytes). Writes past the per-file cap are
// silently dropped; writes past the total cap transition the
// writer into "exhausted" mode where all subsequent writes are
// dropped, which is how the renderer enforces the hard total
// ceiling on file-evidence size.
//
// F10 (CORRECTION02): the writer now reserves a fixed tail
// budget via reserveTail so the termination record can always
// be emitted, regardless of how many files were processed.
// F11 (CORRECTION02): the writer also reserves a fixed
// per-file truncation marker budget so the marker survives
// even when the body fills the per-file cap exactly.
// F12 (CORRECTION02): when the total cap is exceeded inside a
// structured block, the partial block is dropped instead of
// being truncated mid-token.
type boundedWriter struct {
	sb                    strings.Builder
	used                  int  // bytes written so far (excluding tail reservation)
	perFile               int  // bytes written in the current file
	totalCap              int  // MaxTotalRenderBytes
	perFileCap            int  // MaxPerFileBytes
	tailRes               int  // bytes reserved for the termination record
	exhausted             bool // true once totalCap exceeded
	maxMarker             int  // bytes reserved for per-file truncation marker
	perFileMarkerReserved bool // true after the marker has been written
}

func newBoundedWriter(totalCap, perFileCap int) *boundedWriter {
	return &boundedWriter{
		totalCap:   totalCap,
		perFileCap: perFileCap,
	}
}

// String returns the accumulated body. Callers should check
// Exhausted() before relying on completeness.
func (w *boundedWriter) String() string { return w.sb.String() }

// Used returns the cumulative bytes written.
func (w *boundedWriter) Used() int { return w.used }

// PerFileUsed returns the bytes written in the current file.
func (w *boundedWriter) PerFileUsed() int { return w.perFile }

// Exhausted reports whether the total budget has been exceeded.
func (w *boundedWriter) Exhausted() bool { return w.exhausted }

// beginFile resets the per-file counter. Call before rendering a
// new file's body.
func (w *boundedWriter) beginFile() {
	w.perFile = 0
	w.perFileMarkerReserved = false
}

// reserveTail tells the writer that the next `n` bytes of the
// total budget are reserved for the closing termination record.
// Once the writer is exhausted, the tail is still available for
// the single emitOmissionRecord call. F10 (CORRECTION02).
func (w *boundedWriter) reserveTail(n int) { w.tailRes = n }

// reservePerFileMarker tells the writer that a fixed slice of
// the per-file cap is reserved for the truncation marker line.
// F11 (CORRECTION02).
func (w *boundedWriter) reservePerFileMarker(n int) {
	w.maxMarker = n
}

// appendString appends `s` to the writer, subject to the total
// budget. Returns the bytes actually appended. After exceeding
// the total budget the writer becomes exhausted.
func (w *boundedWriter) appendString(s string) int {
	if w.exhausted || s == "" {
		return 0
	}
	budget := w.totalCap - w.tailRes - w.used
	if budget <= 0 {
		w.exhausted = true
		return 0
	}
	if len(s) > budget {
		// F12 (CORRECTION02): drop the partial block
		// rather than emit a truncated mid-token
		// fragment.
		w.exhausted = true
		return 0
	}
	w.sb.WriteString(s)
	w.used += len(s)
	w.perFile += len(s)
	return len(s)
}

// appendTerminationRecord writes s to the writer, bypassing
// the per-file cap and the normal total budget. This is used
// ONLY for the closing termination record (F10 CORRECTION02)
// and is the mechanism that surfaces omitted files once the
// writer is exhausted.
func (w *boundedWriter) appendTerminationRecord(s string) int {
	if s == "" {
		return 0
	}
	if len(s) > w.tailRes {
		s = s[:w.tailRes]
	}
	w.sb.WriteString(s)
	w.used += len(s)
	w.tailRes -= len(s)
	return len(s)
}

// appendFileString appends `s` subject to BOTH the per-file
// budget and the total budget. Returns (appended, perFileCapped)
// where perFileCapped indicates the per-file cap was hit.
func (w *boundedWriter) appendFileString(s string) (int, bool) {
	if w.exhausted || s == "" {
		return 0, false
	}
	// F11: the per-file cap reserves maxMarker bytes for
	// the truncation marker line. The reservation is
	// RELEASED once the marker has been written for this
	// file (so subsequent body bytes can use the full
	// cap) and is REINSTATED at beginFile().
	effectiveCap := w.perFileCap
	if !w.perFileMarkerReserved {
		effectiveCap -= w.maxMarker
		if effectiveCap < 0 {
			effectiveCap = 0
		}
	}
	remaining := effectiveCap - w.perFile
	if remaining <= 0 {
		return 0, true
	}
	perFileCapped := false
	if len(s) > remaining {
		// F11: when truncating the body, leave room for
		// the marker line. The marker will be written
		// separately after this call.
		s = s[:remaining]
		perFileCapped = true
	}
	n := w.appendString(s)
	return n, perFileCapped
}

// markPerFileMarkerReserved records that the per-file
// truncation marker has been written, freeing the reserved
// slice for the BODY slot. F11 (CORRECTION02).
func (w *boundedWriter) markPerFileMarkerReserved() {
	w.perFileMarkerReserved = true
}

// appendBlock is a small helper that formats a block via
// fmt.Sprintf, then appends it to the boundedWriter. Returns
// the number of bytes actually appended (subject to budgets).
func appendBlock(bw *boundedWriter, format string, args ...interface{}) int {
	s := fmt.Sprintf(format, args...)
	n, _ := bw.appendFileString(s)
	return n
}

// boundedFileBlock renders a single changed file under the
// bounded policy. It uses the boundedWriter to enforce per-file
// and total byte budgets. The returned `class` is the
// classification actually applied.
//
// The function is total: pathological inputs are reported as
// bounded stubs.
func boundedFileBlock(bw *boundedWriter, repoRoot, relPath,
	outputAbs string) EvidenceClass {
	fullPath := filepath.Join(repoRoot, relPath)
	var size int64
	if info, err := os.Stat(fullPath); err == nil {
		size = info.Size()
	}
	prefix, body, ok := loadClassifierData(fullPath)
	class := classifyFileEvidence(classifierInput{
		repoRoot:  repoRoot,
		relPath:   relPath,
		fullPath:  fullPath,
		size:      size,
		rawPrefix: prefix,
		outputAbs: outputAbs,
		bodyBytes: body,
	})
	if !ok && size > 0 && size > MaxFileSizeForFull {
		class = ClassBoundedBody
	}

	bw.beginFile()

	switch class {
	case ClassBoundedSelfOutput:
		appendBlock(bw,
			"\n=== %s ===\n"+
				"Classification: %s\n"+
				"Status: excluded (digest output path)\n"+
				"Bytes: %d\n"+
				"Reason: this path is the digest's own "+
				"output; its body is never embedded to "+
				"prevent recursive amplification.\n"+
				"WarningCode: %s\n",
			PathEscape(relPath), class, size,
			WarningCodeSelfOutput)
		return class
	case ClassBoundedRecursive:
		appendBlock(bw,
			"\n=== %s ===\n"+
				"Classification: %s\n"+
				"Status: recognised Leamas digest artifact "+
				"with structural recursion\n"+
				"Bytes: %d\n"+
				"SHA256: %s\n"+
				"Body: suppressed (DIGEST_RECURSION)\n"+
				"WarningCode: %s\n"+
				"Note: the digest artifact's body "+
				"contains structural recursion (nested "+
				"contract markers or a self-diff). Its "+
				"body is intentionally suppressed to "+
				"bound output size.\n",
			PathEscape(relPath), class, size,
			sha256HexFile(fullPath),
			WarningCodeDigestRecursion)
		return class
	case ClassBoundedDerivedDigest:
		appendBlock(bw,
			"\n=== %s ===\n"+
				"Classification: %s\n"+
				"Status: recognised Leamas digest "+
				"artifact\n"+
				"Bytes: %d\n"+
				"SHA256: %s\n"+
				"Body: suppressed "+
				"(DERIVED_DIGEST_BODY_BOUNDED)\n"+
				"WarningCode: %s\n"+
				"Note: a previous generation of the "+
				"targeted digest was detected. Its body "+
				"is suppressed to bound output size; no "+
				"structural recursion was observed.\n",
			PathEscape(relPath), class, size,
			sha256HexFile(fullPath),
			WarningCodeDerivedDigestBounded)
		return class
	case ClassBoundedBody:
		hash := sha256HexFile(fullPath)
		wc := WarningCodeLargeFileBounded
		if hasPathologicalLine(prefix) {
			wc = WarningCodeLineTooLong
		}
		appendBlock(bw,
			"\n=== %s ===\n"+
				"Classification: %s\n"+
				"Status: body suppressed (bounded "+
				"evidence)\n"+
				"Bytes: %d\n"+
				"SHA256: %s\n"+
				"WarningCode: %s\n"+
				"Body: suppressed\n"+
				"Note: full content was suppressed by "+
				"the bounded renderer because its size "+
				"or line length exceeded the safe "+
				"evidence budget.\n",
			PathEscape(relPath), class, size, hash, wc)
		return class
	default:
		appendBlock(bw,
			"\n=== %s ===\n"+
				"Classification: %s\n"+
				"Bytes: %d\n",
			PathEscape(relPath), class, size)
		return class
	}
}

// terminationRecordBudget is the amount of total budget reserved
// (via reserveTail) for the closing termination record. F10.
// 8 KiB is enough to list ~150 omitted paths at ~50 bytes each.
const terminationRecordBudget = 8 * 1024

// perFileMarkerBudget is the amount of per-file budget reserved
// for the truncation marker line. F11.
// 256 bytes is enough for "[per-file body cap hit: 65536 bytes]".
const perFileMarkerBudget = 256

// emitOmissionRecord writes a single deterministic terminal
// record enumerating any files omitted due to total budget
// exhaustion. F10 (CORRECTION02). The record is emitted via
// the writer's reserved tail so it always appears, even after
// the writer is exhausted.
func emitOmissionRecord(bw *boundedWriter, omitted []string,
	reason string) {
	if len(omitted) == 0 {
		return
	}
	// Build the record header.
	count := fmt.Sprintf("EVIDENCE_TRUNCATED=true\n"+
		"omitted_files=%d\n"+
		"reason=%s\n",
		len(omitted), reason)
	bw.appendTerminationRecord("---\n## File Evidence Omission\n")
	bw.appendTerminationRecord(count)
	// Cap the listed paths to fit the tail reservation.
	// Each path costs ~50 bytes, so tail roughly fits
	// tailRes / 50 paths.
	maxPaths := bw.tailRes / 50
	if maxPaths > len(omitted) {
		maxPaths = len(omitted)
	}
	if maxPaths > 0 {
		bw.appendTerminationRecord("first_omitted_paths:\n")
		for i := 0; i < maxPaths; i++ {
			bw.appendTerminationRecord("  - " +
				PathEscape(omitted[i]) + "\n")
		}
		if len(omitted) > maxPaths {
			bw.appendTerminationRecord(fmt.Sprintf(
				"  ... %d more omitted\n",
				len(omitted)-maxPaths))
		}
	}
}
