// Package dupcode provides the shared production-owned V4 pipeline.
package dupcode

// v4BuildInternalFindings is the one production seam. It accepts both
// the lexical analysis map and the analyzed-file inventory so
// component materialization has every input required to fail closed on
// a canonical-content or occurrence-geometry conflict.
//
// Production CheckRepo and the focused test projection
// (v4PipelineInternal) BOTH use this checked variant so an
// exact-content conflict is returned to the caller rather than
// silently merged.
func v4BuildInternalFindings(
	windowMap map[string][]rawWindow,
	analyses map[string]*v4FileAnalysis,
	files map[string]*v4AnalyzedFile,
) ([]v4InternalFinding, error) {
	if files == nil {
		files = v4AnalyzedFilesFromAnalyses(analyses)
	}
	return v4BuildInternalFindingsChecked(windowMap, analyses, files)
}

func v4BuildInternalFindingsChecked(
	windowMap map[string][]rawWindow,
	analyses map[string]*v4FileAnalysis,
	files map[string]*v4AnalyzedFile,
) ([]v4InternalFinding, error) {
	if len(windowMap) == 0 || len(analyses) == 0 || len(files) == 0 {
		return nil, nil
	}
	regionFiltered := filterWindowsToRegions(windowMap, analyses)
	if len(regionFiltered) == 0 {
		return nil, nil
	}
	_, partitions := v4BuildRegionBoundedChainInputs(regionFiltered, analyses)
	if len(partitions) == 0 {
		return nil, nil
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

	// Suppress positional sliding-window shadows before exact edge
	// materialization, while preserving disjoint same-file multiplicity.
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

func v4AnalyzedFilesFromAnalyses(analyses map[string]*v4FileAnalysis) map[string]*v4AnalyzedFile {
	files := make(map[string]*v4AnalyzedFile, len(analyses))
	for path, analysis := range analyses {
		if analysis == nil {
			continue
		}
		file := &v4AnalyzedFile{
			FileTokens: fileTokens{
				path:   path,
				tokens: analysis.Tokens,
				lines:  analysis.Lines,
			},
			Analysis:         *analysis,
			NormalizedTokens: analysis.NormalizedTokens,
		}
		files[path] = file
	}
	return files
}

// v4ContentIdentityFromChain is a compatibility helper for narrow chain
// tests. Production identity is materialized by v4ExactContentKey from both
// independently read occurrence slices.
func v4ContentIdentityFromChain(chain cloneChain) string {
	if len(chain.Matches) == 0 {
		return ""
	}
	return v4SeedFingerprint(computeContentHash(chain.Matches))
}
