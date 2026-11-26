// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_historical_classification.go holds
// the HISTORICAL CLASSIFICATION CONSERVATION selector.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 FINAL AUTHORITY CORRECTION
// (G1)
//
// INDEPENDENTLY REPRODUCED DEFECT: the historical classifier selected
// the HEAD endpoint whenever it resolved. A range that REPLACED a
// historical digest artifact with ordinary text therefore lost the
// preimage's artifact classification.
//
// The artifact-exclusion policy describes the artifact's IDENTITY, not
// the bytes that happen to sit at the path today. A range whose base
// endpoint holds a digest artifact CONSUMED that artifact, so the
// exclusion must survive the replacement.
//
// The companion file evidence_budget_historical_classifier.go holds the
// loader and the endpoint selection.
package digest

// selectHistoricalClassifierClass conserves the artifact-exclusion
// classification across BOTH relevant endpoints of the range.
//
// FINAL AUTHORITY CORRECTION (G1): the predecessor selected the HEAD
// endpoint whenever it resolved, so a range that REPLACED a historical
// digest artifact with ordinary text lost the preimage's artifact
// classification.
//
// The rule is:
//
//  1. if EITHER endpoint classifies as an artifact, the CONSERVED
//     classification is that artifact classification;
//  2. otherwise the HEAD classification is used when the HEAD endpoint
//     is available;
//  3. otherwise the BASE classification is used;
//  4. otherwise the classification is unavailable.
//
// The caller publishes the endpoint that carried the authority, so the
// reviewer can tell WHICH endpoint decided the classification.
func selectHistoricalClassifierClass(head, base EvidenceClass) EvidenceClass {
	if isArtifactExclusionClass(head) {
		return head
	}
	if isArtifactExclusionClass(base) {
		return base
	}
	if head != "" {
		return head
	}
	if base != "" {
		return base
	}
	return ""
}
