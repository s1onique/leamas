// Package dupcode provides exact geometry contract tests for the V4 algorithm.
//
// v4PipelineInternal invokes the same production-owned merge seam used by the
// public V4 path. Test code only prepares fixtures, invokes production stages,
// normalizes fixture paths, and projects retained token positions.
package dupcode

import (
	"sort"
	"testing"
)

// v4PipelineInternal invokes production tokenization, seed discovery,
// chaining, and v4InternalFindingsFromChains. The last function owns grouping,
// N-way merge, deduplication, occurrence sorting, and finding ordering for both
// this internal projection and the public v4FindingsFromChains path.
func v4PipelineInternal(t *testing.T, root string, paths []string, cfg Config) []exactInternalFindingGeometry {
	t.Helper()

	if cfg.MinLines == 0 {
		cfg.MinLines = DefaultConfig().MinLines
	}
	if cfg.MinTokens == 0 {
		cfg.MinTokens = DefaultConfig().MinTokens
	}

	normRoot := "."
	if root != "" && root != "." {
		normRoot = root
	}

	allFiles := make([]fileTokens, 0, len(paths))
	for _, path := range paths {
		ft, err := tokenizeFile(path)
		if err != nil {
			t.Fatalf("tokenize %s: %v", path, err)
		}
		ft.path = NormalizePathForBaseline(ft.path, normRoot)
		allFiles = append(allFiles, ft)
	}

	windowMap := make(map[string][]rawWindow)
	fingerprintTokens := make(map[string]int)
	for i := 0; i < len(allFiles); i++ {
		if len(allFiles[i].tokens) < cfg.MinTokens {
			continue
		}
		for j := i + 1; j < len(allFiles); j++ {
			if len(allFiles[j].tokens) < cfg.MinTokens {
				continue
			}
			findCommonWindows(allFiles[i], allFiles[j], cfg, windowMap, fingerprintTokens)
		}
	}
	if len(windowMap) == 0 {
		return nil
	}

	fps := make([]string, 0, len(windowMap))
	for fp := range windowMap {
		fps = append(fps, fp)
	}
	sort.Strings(fps)

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

	internal := v4InternalFindingsFromChains(chains)
	projected := make([]exactInternalFindingGeometry, len(internal))
	for i, finding := range internal {
		occurrences := make([]exactInternalOccurrenceGeometry, len(finding.Occurrences))
		for j, occurrence := range finding.Occurrences {
			occurrences[j] = exactInternalOccurrenceGeometry{
				Path:      occurrence.Path,
				StartPos:  occurrence.StartPos,
				EndPos:    occurrence.EndPos,
				StartLine: occurrence.StartLine,
				EndLine:   occurrence.EndLine,
			}
		}
		projected[i] = exactInternalFindingGeometry{
			TokenCount:  finding.TokenCount,
			Occurrences: occurrences,
		}
	}
	return projected
}
