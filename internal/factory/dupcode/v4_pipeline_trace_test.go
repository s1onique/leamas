// Package dupcode provides test-only pipeline trace helpers for
// the V4 exact-content merge pipeline.
//
// CORRECTION04 requires the maximality proof to inspect the
// actual pre-publication pipeline stages rather than starting
// from the already-suppressed final finding. The trace helpers
// in this file expose every intermediate value produced by the
// production V4 pipeline (filtered windows, partitions, chains
// before and after chain-shadow suppression, validated pair
// evidence, components before and after structural-shadow
// suppression, and final findings) without duplicating any
// production algorithm.
//
// The trace lives in a `_test.go` file so production callers do
// not gain access to intermediate state. Ordinary production
// execution continues to use the existing
// `v4BuildInternalFindingsChecked` seam.
package dupcode

import (
	"testing"
)

// v4PipelineTrace captures every intermediate value produced by
// the production V4 pipeline for one invocation. The trace is
// the evidence substrate for the CORRECTION04 maximality proof:
// the proof reads from the trace's components, chains, and pair
// evidence rather than from the already-suppressed final finding.
type v4PipelineTrace struct {
	// FilteredWindows is the region-aware window map produced by
	// `filterWindowsToRegions`. Windows whose token interval
	// crosses an executable-region boundary are removed.
	FilteredWindows map[string][]rawWindow

	// Partitions is the map from chain-pair key to seed-match
	// group produced by `v4BuildRegionBoundedChainInputs`.
	Partitions map[v4ChainPairKey][]v4RegionSeedMatch

	// ChainsBeforeShadow is the list of clone chains produced by
	// `extendRegionBoundedChain` before chain-shadow suppression.
	ChainsBeforeShadow []cloneChain

	// ChainsAfterShadow is the list of clone chains that survive
	// `v4SuppressShadowChainsRegionBounded`.
	ChainsAfterShadow []cloneChain

	// PairEvidence is the validated pair-evidence slice produced by
	// `v4PairEvidenceFromChain` for every surviving chain.
	PairEvidence []v4PairCloneEvidence

	// ComponentsBeforeShadow is the list of connected components
	// produced by `v4MaterializeComponents` BEFORE
	// `v4SuppressComponentShadows` and
	// `v4SuppressContainedSameFileShadows` are applied.
	ComponentsBeforeShadow []v4InternalFinding

	// ComponentsAfterShadow is the list of connected components
	// AFTER both component-shadow suppressions are applied.
	ComponentsAfterShadow []v4InternalFinding

	// FinalFindings is the sorted projection returned to the caller.
	FinalFindings []v4InternalFinding
}

// v4BuildInternalFindingsTrace runs the same production V4
// pipeline as `v4BuildInternalFindingsChecked` but also captures
// every intermediate value in a `v4PipelineTrace` for forensic
// analysis. The function lives in a `_test.go` file so production
// callers cannot request the trace.
//
// The pipeline stages are invoked in the same order as the
// production seam:
//
//  1. `filterWindowsToRegions`       -> FilteredWindows
//  2. `v4BuildRegionBoundedChainInputs` -> Partitions
//  3. `extendRegionBoundedChain`     -> ChainsBeforeShadow
//  4. `v4SuppressShadowChainsRegionBounded` -> ChainsAfterShadow
//  5. `v4PairEvidenceFromChain`     -> PairEvidence (per chain)
//  6. `v4MaterializeComponents`      -> ComponentsBeforeShadow
//  7. `v4SuppressComponentShadows`   -> component-shadow filter
//  8. `v4SuppressContainedSameFileShadows` -> same-file shadow filter
//  9. `sortV4InternalFindings`      -> final ordering
//
// Stages 7–9 produce ComponentsAfterShadow, which equals
// FinalFindings. The trace lets the maximality proof reason
// about the live pre-suppression component set in addition to the
// already-suppressed final findings.
func v4BuildInternalFindingsTrace(
	windowMap map[string][]rawWindow,
	analyses map[string]*v4FileAnalysis,
	files map[string]*v4AnalyzedFile,
) ([]v4InternalFinding, v4PipelineTrace, error) {
	trace := v4PipelineTrace{
		FilteredWindows:        map[string][]rawWindow{},
		Partitions:             map[v4ChainPairKey][]v4RegionSeedMatch{},
		ChainsBeforeShadow:     []cloneChain{},
		ChainsAfterShadow:      []cloneChain{},
		PairEvidence:           []v4PairCloneEvidence{},
		ComponentsBeforeShadow: []v4InternalFinding{},
		ComponentsAfterShadow:  []v4InternalFinding{},
		FinalFindings:          []v4InternalFinding{},
	}
	if len(windowMap) == 0 || len(analyses) == 0 || len(files) == 0 {
		return nil, trace, nil
	}
	regionFiltered := filterWindowsToRegions(windowMap, analyses)
	trace.FilteredWindows = regionFiltered
	if len(regionFiltered) == 0 {
		return nil, trace, nil
	}
	_, partitions := v4BuildRegionBoundedChainInputs(regionFiltered, analyses)
	trace.Partitions = partitions
	if len(partitions) == 0 {
		return nil, trace, nil
	}

	keys := make([]v4ChainPairKey, 0, len(partitions))
	for key := range partitions {
		keys = append(keys, key)
	}
	sortChainPairKeys(keys)
	var chains []cloneChain
	for _, key := range keys {
		chains = append(chains, extendRegionBoundedChain(partitions[key])...)
	}
	trace.ChainsBeforeShadow = chains

	// Suppress positional sliding-window shadows before exact edge
	// materialization, while preserving disjoint same-file multiplicity.
	chains = v4SuppressShadowChainsRegionBounded(chains, analyses)
	trace.ChainsAfterShadow = chains

	// Capture pair evidence for every surviving chain BEFORE
	// component materialization. A chain whose left/right widths
	// or digests disagree fails here, so the surviving evidence
	// is exactly the set that drives the connected-component
	// materializer.
	pairEvidence := make([]v4PairCloneEvidence, 0, len(chains))
	for _, chain := range chains {
		evidence, err := v4PairEvidenceFromChain(chain, files)
		if err != nil {
			return nil, trace, err
		}
		pairEvidence = append(pairEvidence, evidence)
	}
	trace.PairEvidence = pairEvidence

	findings, err := v4MaterializeComponents(chains, files)
	if err != nil {
		return nil, trace, err
	}
	trace.ComponentsBeforeShadow = findings

	findings = v4SuppressComponentShadows(findings, files)
	findings = v4SuppressContainedSameFileShadows(findings, files)
	sortV4InternalFindings(findings)
	trace.ComponentsAfterShadow = findings
	trace.FinalFindings = findings
	return findings, trace, nil
}

// traceForLiveTree builds the v4PipelineTrace for the actual
// production tree (cmd/leamas/claim_commands.go and
// cmd/leamas/evidence_commands.go). The helper runs the
// production analyzer pipeline to compute the window map, then
// invokes v4BuildInternalFindingsTrace.
//
// traceForLiveTree is the canonical forensic entry point used by
// the CORRECTION04 maximality tests.
func traceForLiveTree(t *testing.T) (
	leftFile, rightFile *v4AnalyzedFile,
	trace v4PipelineTrace,
	finals []v4InternalFinding,
) {
	t.Helper()
	root := deltaRepoRoot(t)
	leftAbs := repoRel(root, "cmd/leamas/claim_commands.go")
	rightAbs := repoRel(root, "cmd/leamas/evidence_commands.go")

	leftVal, err := analyzeV4AnalyzedFile(leftAbs)
	if err != nil {
		t.Fatalf("analyze %s: %v", leftAbs, err)
	}
	rightVal, err := analyzeV4AnalyzedFile(rightAbs)
	if err != nil {
		t.Fatalf("analyze %s: %v", rightAbs, err)
	}
	rebaseV4AnalyzedFilePath(&leftVal, "cmd/leamas/claim_commands.go")
	rebaseV4AnalyzedFilePath(&rightVal, "cmd/leamas/evidence_commands.go")
	leftFile = &leftVal
	rightFile = &rightVal

	filesMap := map[string]*v4AnalyzedFile{
		"cmd/leamas/claim_commands.go":    leftFile,
		"cmd/leamas/evidence_commands.go": rightFile,
	}
	analysesMap := map[string]*v4FileAnalysis{
		"cmd/leamas/claim_commands.go":    &leftVal.Analysis,
		"cmd/leamas/evidence_commands.go": &rightVal.Analysis,
	}

	windowMap := make(map[string][]rawWindow)
	fingerprintTokens := make(map[string]int)
	for i, ft1 := range []fileTokens{leftVal.FileTokens, rightVal.FileTokens} {
		if len(ft1.tokens) < DefaultConfig().MinTokens {
			continue
		}
		for j := i + 1; j < 2; j++ {
			ft2 := []fileTokens{leftVal.FileTokens, rightVal.FileTokens}[j]
			if len(ft2.tokens) < DefaultConfig().MinTokens {
				continue
			}
			findCommonWindows(ft1, ft2, DefaultConfig(), windowMap, fingerprintTokens)
		}
	}

	finals, trace, err = v4BuildInternalFindingsTrace(windowMap, analysesMap, filesMap)
	if err != nil {
		t.Fatalf("v4BuildInternalFindingsTrace: %v", err)
	}
	return leftFile, rightFile, trace, finals
}
