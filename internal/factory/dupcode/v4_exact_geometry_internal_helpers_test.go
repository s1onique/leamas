// Package dupcode provides exact geometry contract tests for the V4 algorithm.
//
// v4PipelineInternal invokes the same production-owned merge seam used by the
// public V4 path. Test code only prepares fixtures, invokes production stages,
// normalizes fixture paths, and projects retained token positions.
package dupcode

import (
	"testing"
)

// v4PipelineInternal invokes production tokenization, region-aware
// analysis, and the shared V4 internal pipeline
// (v4BuildInternalFindings). The exact same internal findings are
// produced for both this internal projection and the public
// v4FindingsFromChains path; the public projection is a direct view
// of the same internal values.
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

	type entry struct {
		path     string
		fileToks fileTokens
		analysis v4FileAnalysis
		analyzed v4AnalyzedFile
	}
	entries := make([]entry, 0, len(paths))
	allAnalyses := make(map[string]*v4FileAnalysis)
	allAnalyzedFiles := make(map[string]*v4AnalyzedFile)
	for _, path := range paths {
		analyzed, err := analyzeV4AnalyzedFile(path)
		if err != nil {
			t.Fatalf("analyse %s: %v", path, err)
		}
		normalized := NormalizePathForBaseline(analyzed.FileTokens.path, normRoot)
		rebaseV4AnalyzedFilePath(&analyzed, normalized)

		entries = append(entries, entry{
			path:     normalized,
			fileToks: analyzed.FileTokens,
			analysis: analyzed.Analysis,
			analyzed: analyzed,
		})
	}
	for i := range entries {
		allAnalyses[entries[i].path] = &entries[i].analysis
		allAnalyzedFiles[entries[i].path] = &entries[i].analyzed
	}

	windowMap := make(map[string][]rawWindow)
	fingerprintTokens := make(map[string]int)
	for i := 0; i < len(entries); i++ {
		if len(entries[i].fileToks.tokens) < cfg.MinTokens {
			continue
		}
		for j := i + 1; j < len(entries); j++ {
			if len(entries[j].fileToks.tokens) < cfg.MinTokens {
				continue
			}
			findCommonWindows(entries[i].fileToks, entries[j].fileToks, cfg, windowMap, fingerprintTokens)
		}
	}
	if len(windowMap) == 0 {
		return nil
	}
	// Same checked production seam as the public CheckRepo path.
	internal, err := v4BuildInternalFindingsChecked(windowMap, allAnalyses, allAnalyzedFiles)
	if err != nil {
		t.Fatalf("build internal findings: %v", err)
	}

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
