// Package dupcode provides the test-only all-pairs oracle for
// ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION01.
//
// R2: the oracle reproduces the legacy all-pairs candidate generator
// that was replaced by the alignment-guarded diagonal fast path. It
// is consumed only by the differential tests in
// v4_alignment_differential_test.go and the fuzz target in
// v4_alignment_fuzz_test.go. The oracle lives in this test-only
// file so production code cannot reach it.
package dupcode

import (
	"sort"
)

// v4GenerateAllPairsMatchesOracle is the test-only oracle. It
// reproduces the legacy all-pairs candidate generator that was
// replaced by the diagonal fast path. The oracle preserves:
//
//   - same-region overlap rejection,
//   - canonical orientation (lex smaller path → Left, then smaller Ordinal),
//   - region identities,
//   - offsets.
//
// It MUST be kept in lock step with the legacy behaviour. When the
// alignment guard in generateRegionAnnotatedMatches is removed (so the
// diagonal fires on unaligned sequences), this oracle catches the
// divergence via the differential corpus below.
func v4GenerateAllPairsMatchesOracle(
	fp string,
	windows []rawWindow,
	analysisByPath map[string]*v4FileAnalysis,
) []v4RegionSeedMatch {
	if len(windows) < 2 {
		return nil
	}

	type annotated struct {
		w      rawWindow
		region v4SyntaxRegionID
	}
	annotatedWindows := make([]annotated, len(windows))
	for i, w := range windows {
		a, ok := analysisByPath[w.Path]
		if !ok {
			annotatedWindows[i] = annotated{w: w}
			continue
		}
		rid, ok := a.windowFitsRegion(w.StartPos, w.EndPos)
		if !ok {
			annotatedWindows[i] = annotated{w: w}
			continue
		}
		annotatedWindows[i] = annotated{w: w, region: rid}
	}

	var out []v4RegionSeedMatch
	for i := 0; i < len(annotatedWindows); i++ {
		for j := i + 1; j < len(annotatedWindows); j++ {
			a := annotatedWindows[i]
			b := annotatedWindows[j]
			if a.region.Path == "" || b.region.Path == "" {
				continue
			}
			if a.region == b.region && tokenRangesOverlap(a.w, b.w) {
				continue
			}
			left, right := a.w, b.w
			leftRegion, rightRegion := a.region, b.region
			// Canonical orientation: lex smaller path → Left; within
			// the same path, smaller StartPos → Left. The production
			// candidate generator uses the same rule via WINDOW.Path
			// and WINDOW.StartPos; the oracle uses WINDOW fields too
			// so the emitted matches have the same canonical form.
			if a.w.Path > b.w.Path ||
				(a.w.Path == b.w.Path && a.w.StartPos > b.w.StartPos) {
				left, right = b.w, a.w
				leftRegion, rightRegion = b.region, a.region
			}
			out = append(out, v4RegionSeedMatch{
				Match: seedMatch{
					SeedFingerprint: fp,
					Left:            left,
					Right:           right,
					Offset:          right.StartPos - left.StartPos,
				},
				LeftRegion:  leftRegion,
				RightRegion: rightRegion,
			})
		}
	}
	return out
}

// v4RunFullPipelineForOracle runs the rest of the V4 pipeline
// (chain partition, chain extension, shadow suppression, component
// materialization, N-way merge, shadow suppression of components,
// sort) on a candidate slice produced by an externally-supplied
// generator. It exists only to support v4BuildInternalFindingsOracle
// below.
func v4RunFullPipelineForOracle(
	windowMap map[string][]rawWindow,
	analyses map[string]*v4FileAnalysis,
	files map[string]*v4AnalyzedFile,
	generator func(fp string, windows []rawWindow, analysisByPath map[string]*v4FileAnalysis) []v4RegionSeedMatch,
) ([]v4InternalFinding, error) {
	if len(windowMap) == 0 || len(analyses) == 0 || len(files) == 0 {
		return nil, nil
	}
	regionFiltered := filterWindowsToRegions(windowMap, analyses)
	return v4RunFullPipelineForOracleFiltered(regionFiltered, analyses, files, generator)
}

// v4RunFullPipelineForOracleFiltered accepts an independently filtered
// window map. The semantic corpus uses this seam so ownership filtering
// is compared separately instead of sharing production's filter helper.
func v4RunFullPipelineForOracleFiltered(
	regionFiltered map[string][]rawWindow,
	analyses map[string]*v4FileAnalysis,
	files map[string]*v4AnalyzedFile,
	generator func(fp string, windows []rawWindow, analysisByPath map[string]*v4FileAnalysis) []v4RegionSeedMatch,
) ([]v4InternalFinding, error) {
	if len(regionFiltered) == 0 || len(analyses) == 0 || len(files) == 0 {
		return nil, nil
	}

	// Run the test-only candidate generator across every
	// fingerprint bucket, mimicking v4BuildRegionBoundedChainInputs
	// behaviour except for the candidate generator itself.
	fps := make([]string, 0, len(regionFiltered))
	for fp := range regionFiltered {
		fps = append(fps, fp)
	}
	sort.Strings(fps)
	combined := make([]v4RegionSeedMatch, 0)
	for _, fp := range fps {
		combined = append(combined, generator(fp, regionFiltered[fp], analyses)...)
	}

	if len(combined) == 0 {
		return nil, nil
	}
	partitions := make(map[v4ChainPairKey][]v4RegionSeedMatch)
	for _, m := range combined {
		offset := m.Match.Right.StartPos - m.Match.Left.StartPos
		key := canonicalChainPairKey(m.LeftRegion, m.RightRegion, offset)
		partitions[key] = append(partitions[key], m)
	}
	for key := range partitions {
		group := partitions[key]
		sortRegionAnnotatedMatches(group)
		partitions[key] = group
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
	chains = v4SuppressShadowChainsRegionBounded(chains, analyses)
	findings, err := v4MaterializeComponents(chains, files)
	if err != nil {
		return nil, err
	}
	findings = v4SuppressComponentShadows(findings, files)
	findings = v4SuppressContainedSameFileShadows(findings, files)
	sortV4InternalFindings(findings)
	return findings, nil
}

// v4BuildInternalFindingsOracle computes the canonical internal
// findings using a caller-supplied candidate generator and the
// rest of the V4 pipeline.
func v4BuildInternalFindingsOracle(
	windowMap map[string][]rawWindow,
	analyses map[string]*v4FileAnalysis,
	files map[string]*v4AnalyzedFile,
	generator func(fp string, windows []rawWindow, analysisByPath map[string]*v4FileAnalysis) []v4RegionSeedMatch,
) ([]v4InternalFinding, error) {
	return v4RunFullPipelineForOracle(windowMap, analyses, files, generator)
}
