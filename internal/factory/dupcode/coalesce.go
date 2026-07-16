// Package dupcode provides duplicate code detection for Go source files.
package dupcode

// rawWindow represents a sliding window over a file's tokens.
// StartPos and EndPos are token offsets (0-based).
type rawWindow struct {
	Path      string
	StartLine int // 1-based line number
	EndLine   int
	StartPos  int // 0-based token position
	EndPos    int
}

// tokenRange represents a span in token coordinates.
type tokenRange struct {
	StartPos int
	EndPos   int
}

// coalescedFinding represents a coalesced clone finding.
type coalescedFinding struct {
	Fingerprint       string
	StableFingerprint string
	SeedFingerprint   string // Original seed fingerprint that generated this finding
	Occurrences       []Occurrence
	TokenCount        int
	LineCount         int
}

// windowMatch represents a matched pair of windows from two files.
// DEPRECATED: Use seedMatch from v3.go for algorithm v3.
type windowMatch struct {
	Fingerprint string
	Left        rawWindow
	Right       rawWindow
}

// alignedComponent represents a maximal aligned clone component.
// DEPRECATED: Use cloneChain from v3.go for algorithm v3.
type alignedComponent struct {
	Fingerprint   string
	Occurrences   map[string][]tokenRange
	MaxTokenCount int
	MaxLineCount  int
}

// coalesceFindings is the V4 algorithm entry point used by callers
// that only have a fingerprint-bucketed window map and no
// region-aware analysis map. The legacy default-config path remains
// useful for narrow tests and for the legacy fallback used by some
// callers; production CheckRepo builds the analysis map itself and
// calls v4BuildInternalFindings directly. When the window map is
// empty the function returns nil; otherwise it projects the V4
// internal findings through the public coalesced type.
func coalesceFindings(
	windowMap map[string][]rawWindow,
	fingerprintTokens map[string]int,
) []coalescedFinding {
	if len(windowMap) == 0 {
		return nil
	}
	// Legacy callers that do not pass an analysis map cannot use the
	// region-aware pipeline. They receive the legacy non-region
	// chain construction that was the v4 algorithm in
	// ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01.
	return legacyV4CoalesceFindings(windowMap, fingerprintTokens)
}

// legacyV4CoalesceFindings is the pre-region chain construction that
// the V4 algorithm used before ACT-LEAMAS-FACTORY-DUPCODE-V4-REGION-BOUNDED-CHAIN-CONSTRUCTION02.
// It is retained only for callers that cannot supply an analysis map.
// Production callers (CheckRepo, v4PipelineInternal) route through
// v4BuildInternalFindings.
func legacyV4CoalesceFindings(windowMap map[string][]rawWindow, fingerprintTokens map[string]int) []coalescedFinding {
	var fps []string
	for fp := range windowMap {
		fps = append(fps, fp)
	}
	sortStrings(fps)

	var allMatches []seedMatch
	for _, fp := range fps {
		allMatches = append(allMatches, buildSeedMatches(fp, windowMap[fp])...)
	}

	if len(allMatches) == 0 {
		return nil
	}

	chains := v4BuildChainsWithPartitioning(allMatches)
	if len(chains) == 0 {
		return nil
	}

	return v4FindingsFromChains(chains)
}

// sortStrings is a tiny indirection so this file does not import
// "sort" when callers only use the wrapper above.
func sortStrings(xs []string) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j-1] > xs[j]; j-- {
			xs[j-1], xs[j] = xs[j], xs[j-1]
		}
	}
}
