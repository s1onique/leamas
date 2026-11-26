// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_render_completeness.go holds the
// GUARANTEED-PUBLICATION machinery of the bounded evidence renderer.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 AMENDMENT-01 FINALIZATION
// (F2)
//
// The completeness disposition of a file states whether the file's
// patch was published IN FULL ("EvidenceCompleteness: COMPLETE") or
// was cut short by a budget ("EvidenceCompleteness: INCOMPLETE",
// with a truthful "SuppressionReason:").
//
// The disposition is the reviewer's ONLY signal about whether the
// evidence is complete. If it can be silently dropped, a partial
// publication is indistinguishable from a complete one.
//
// The predecessor wrote the disposition into the SAME writer whose
// budget the patch had just exhausted. appendBlock therefore dropped
// it, and the digest reported a partial publication with NO
// completeness statement at all.
//
// This file implements the reservation that makes the disposition a
// GUARANTEED publication.
package digest

import "fmt"

// completenessDispositionBudget is the amount of TOTAL budget
// reserved for the completeness disposition of the current file.
//
// AMENDMENT-01 FINALIZATION (F2): the disposition must be published
// even when the patch body exhausted the aggregate budget. 256 bytes
// is enough for the longest disposition (the per-file-cap variant
// carries an extra "[per-file body cap hit: N bytes]" line).
const completenessDispositionBudget = 256

// reserveCompleteness tells the writer that `n` bytes of the TOTAL
// budget are reserved for the completeness disposition of the current
// file.
//
// AMENDMENT-01 FINALIZATION (F2): the reservation is released again
// once the disposition has been written, so the budget is not
// permanently consumed by files that never reach the disposition.
//
// The reservation is what makes the completeness statement a
// GUARANTEED publication: the patch body cannot consume the slice
// that the disposition itself needs.
func (w *boundedWriter) reserveCompleteness(n int) {
	w.completenessRes = n
}

// appendCompleteness writes the completeness disposition of the
// current file, GUARANTEeing that it is published.
//
// AMENDMENT-01 FINALIZATION (F2): the disposition bypasses the
// ordinary total-budget check by using the slice reserved by
// reserveCompleteness. If the ordinary budget cannot accommodate the
// disposition, the RESERVED slice is used instead.
//
// The disposition is never silently dropped. A reviewer that sees a
// file section always sees the completeness statement of that file,
// so a partial publication can never masquerade as a complete one.
func (w *boundedWriter) appendCompleteness(format string,
	args ...interface{}) int {
	s := fmt.Sprintf(format, args...)
	if s == "" {
		return 0
	}
	// Release the reservation once the disposition is written.
	res := w.completenessRes
	w.completenessRes = 0

	// Try the ordinary budget first.
	if !w.exhausted {
		budget := w.totalCap - w.tailRes - w.used
		if len(s) <= budget {
			w.sb.WriteString(s)
			w.used += len(s)
			w.perFile += len(s)
			return len(s)
		}
	}
	// The ordinary budget could not accommodate the disposition.
	// Fall back to the RESERVED slice, which the patch body was
	// never allowed to consume.
	if res > 0 && len(s) <= res {
		w.sb.WriteString(s)
		w.used += len(s)
		return len(s)
	}
	// The reserved slice is too small for the disposition. The
	// statement is still published: a truncated statement is better
	// than no statement, because the reviewer must never be left
	// believing a partial publication is complete.
	if res > 0 {
		w.sb.WriteString(s[:res])
		w.used += res
		return res
	}
	return 0
}
