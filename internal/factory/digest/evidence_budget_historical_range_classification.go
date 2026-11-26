// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_historical_range_classification.go
// holds the RANGE-MODE historical classification decision.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 FINAL AUTHORITY CORRECTION
// (G1)
//
// INDEPENDENTLY REPRODUCED DEFECT: the range renderer classified from
// the HEAD endpoint alone whenever it resolved. A range that REPLACED
// a historical digest artifact with ordinary text therefore lost the
// preimage's artifact classification, even though the range still
// CONSUMED that artifact.
//
// The artifact-exclusion policy describes the artifact's IDENTITY, not
// the bytes that happen to sit at the path today.
//
// The companion files are:
//   - evidence_budget_historical_classifier.go   (the loaders)
//   - evidence_budget_historical_classification.go (the conservation
//     selector)
package digest

// classifyHistoricalRangeFile classifies ONE range file from the
// historical blobs of BOTH relevant endpoints.
//
// It returns:
//
//   - the CONSERVED classification;
//   - the evidence source, naming the endpoint that carried the
//     CLASSIFICATION AUTHORITY ("HEAD", "BASE" or "NONE");
//   - whether historical evidence was available at all.
//
// The evidence source distinguishes a GENUINELY ABSENT endpoint object
// (a deletion, or a path the endpoint never held) from UNAVAILABLE
// evidence (the object exists but its content could not be recovered):
// an absent endpoint is not "unavailable", the other endpoint is simply
// the authority.
//
// When an artifact classification was conserved from the BASE endpoint,
// the BASE endpoint carried the authority even though the HEAD endpoint
// resolved.
func classifyHistoricalRangeFile(repoRoot string, f RangeFile,
	fullPath string, blobSize int64, outputAbs string,
	head, base HistoricalClassifierData) (
	EvidenceClass, string, bool) {
	headClass := EvidenceClass("")
	if head.OK {
		headClass = classifyFileEvidence(classifierInput{
			repoRoot:  repoRoot,
			relPath:   f.Path,
			fullPath:  fullPath,
			size:      blobSize,
			rawPrefix: head.Prefix,
			outputAbs: outputAbs,
			bodyBytes: head.Body,
		})
	}
	baseClass := EvidenceClass("")
	if base.OK {
		baseClass = classifyFileEvidence(classifierInput{
			repoRoot:  repoRoot,
			relPath:   f.Path,
			fullPath:  fullPath,
			size:      blobSize,
			rawPrefix: base.Prefix,
			outputAbs: outputAbs,
			bodyBytes: base.Body,
		})
	}

	class := selectHistoricalClassifierClass(headClass, baseClass)

	source := "NONE"
	switch {
	case !head.OK && !base.OK:
		// NO historical evidence is available.
		source = "NONE"
	case isArtifactExclusionClass(class) && baseClass == class &&
		base.OK && headClass != class:
		// The BASE endpoint carried the artifact-exclusion
		// authority.
		source = "BASE"
	case head.OK:
		source = "HEAD"
	case base.OK:
		source = "BASE"
	}
	return class, source, head.OK || base.OK
}
