// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_authority_amendment_completeness_test.go
// holds the AMENDMENT-01 FINALIZATION (F2) aggregate-completeness
// witnesses for ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01.
//
// F2  a patch that the AGGREGATE budget cut short must still publish
//
//	its completeness disposition.
//
// The companion file
// evidence_budget_patch_authority_amendment_test.go holds the A1, A2
// and A6 witnesses, and
// evidence_budget_patch_authority_amendment_identity_test.go holds the
// A3, A4 and F1 witnesses.
package digest

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// F2: aggregate completeness.
//
// AMENDMENT-01 FINALIZATION (F2): a patch that the AGGREGATE budget
// cut short must still publish its completeness disposition.
//
// The predecessor wrote the disposition into the SAME writer whose
// budget the patch had just exhausted. appendBlock therefore dropped
// it, and the digest reported a partial publication with NO
// completeness statement at all. A reviewer could not tell a partial
// publication from a complete one.
//
// The witness exercises the WRITER directly, because that is the
// boundary the defect lived at. It establishes the precondition (the
// total budget IS exhausted) and then asserts that the disposition is
// nevertheless published.
// ---------------------------------------------------------------------------

// TestAmendment_F2_CompletenessDispositionSurvivesAggregateExhaustion
// verifies that the completeness disposition is published even when
// the aggregate (total) render budget has been exhausted.
//
// PRECONDITION: the total budget is exhausted before the disposition
// is written.
//
// POSTCONDITION: the disposition is nevertheless present in the
// rendered output.
func TestAmendment_F2_CompletenessDispositionSurvivesAggregateExhaustion(t *testing.T) {
	bw := newBoundedWriter(MaxTotalRenderBytes, MaxPerFileBytes)
	bw.reserveTail(terminationRecordBudget)
	bw.reservePerFileMarker(perFileMarkerBudget)
	bw.beginFile()
	appendBlock(bw, "## Diffs\n")

	// Fill the total budget with blocks that FIT, until the writer
	// refuses to accept any more.
	chunk := strings.Repeat("x", 8*1024)
	for i := 0; i < 1000; i++ {
		bw.beginFile()
		if appendBlock(bw, "%s", chunk) == 0 {
			break
		}
	}
	// PRECONDITION: the total budget really is exhausted.
	if !bw.Exhausted() {
		t.Fatalf("fixture setup: the total budget must be exhausted "+
			"for this witness to be a discriminator. used=%d",
			bw.Used())
	}

	// Now write the completeness disposition for the interrupted
	// file. The reservation must guarantee its publication.
	bw.reserveCompleteness(completenessDispositionBudget)
	bw.appendCompleteness(
		"\nEvidenceCompleteness: INCOMPLETE\n" +
			"SuppressionReason: PATCH_TRUNCATED_BY_TOTAL_CAP\n")

	out := bw.String()
	if !strings.Contains(out, "EvidenceCompleteness: INCOMPLETE") {
		t.Fatalf("aggregate completeness violated: the "+
			"completeness disposition was DROPPED after the total "+
			"budget was exhausted. A partial publication must never "+
			"be presented without its completeness statement. "+
			"Writer used=%d tailRes=%d",
			bw.Used(), bw.tailRes)
	}
	if !strings.Contains(out, "PATCH_TRUNCATED_BY_TOTAL_CAP") {
		t.Fatalf("aggregate completeness violated: the "+
			"suppression reason was dropped together with the "+
			"disposition. Writer used=%d", bw.Used())
	}
}

// TestAmendment_F2_CompleteDispositionSurvivesAggregateExhaustion
// verifies the SAME guarantee for the COMPLETE disposition: a patch
// that fit the per-file budget must still be declared complete even
// when the aggregate budget was exhausted right after it.
//
// This is the second edge case the review raised: a complete patch
// might fit exactly, but the subsequent "EvidenceCompleteness:
// COMPLETE" marker might not.
func TestAmendment_F2_CompleteDispositionSurvivesAggregateExhaustion(t *testing.T) {
	bw := newBoundedWriter(MaxTotalRenderBytes, MaxPerFileBytes)
	bw.reserveTail(terminationRecordBudget)
	bw.reservePerFileMarker(perFileMarkerBudget)
	bw.beginFile()
	appendBlock(bw, "## Diffs\n")

	chunk := strings.Repeat("x", 8*1024)
	for i := 0; i < 1000; i++ {
		bw.beginFile()
		if appendBlock(bw, "%s", chunk) == 0 {
			break
		}
	}
	if !bw.Exhausted() {
		t.Fatalf("fixture setup: the total budget must be exhausted "+
			"for this witness to be a discriminator. used=%d",
			bw.Used())
	}

	bw.reserveCompleteness(completenessDispositionBudget)
	bw.appendCompleteness("EvidenceCompleteness: COMPLETE\n")

	out := bw.String()
	if !strings.Contains(out, "EvidenceCompleteness: COMPLETE") {
		t.Fatalf("aggregate completeness violated: the COMPLETE "+
			"disposition was dropped after the total budget was "+
			"exhausted. Writer used=%d", bw.Used())
	}
}

// TestAmendment_F2_CompletenessReservationIsReleased verifies that the
// reservation does not permanently consume the total budget.
//
// The reservation exists only for the duration of one file's
// disposition. If it were never released, every file would shrink
// the usable budget by the reservation size and the aggregate budget
// would be materially weakened.
func TestAmendment_F2_CompletenessReservationIsReleased(t *testing.T) {
	bw := newBoundedWriter(MaxTotalRenderBytes, MaxPerFileBytes)
	bw.reserveTail(terminationRecordBudget)
	bw.reservePerFileMarker(perFileMarkerBudget)
	bw.beginFile()

	before := bw.Used()
	bw.reserveCompleteness(completenessDispositionBudget)
	bw.appendCompleteness("EvidenceCompleteness: COMPLETE\n")
	if bw.completenessRes != 0 {
		t.Fatalf("the completeness reservation must be released "+
			"once the disposition is written, got %d",
			bw.completenessRes)
	}
	after := bw.Used()
	if after-before != len("EvidenceCompleteness: COMPLETE\n") {
		t.Fatalf("the disposition must cost exactly its own "+
			"length, got %d bytes", after-before)
	}
}
