// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"fmt"
	"sort"
)

// chainToOccurrence converts a chain to an Occurrence.
func chainToOccurrence(chain cloneChain, path string, lineStart int) Occurrence {
	return Occurrence{
		Path:      path,
		StartLine: lineStart,
		EndLine:   lineStart + chain.LineSpan - 1,
	}
}

// occurrenceFromChain converts a cloneChain to occurrences using final chain ranges.
// Each file in the chain produces one occurrence based on the maximal range.
func occurrenceFromChain(chain cloneChain) []Occurrence {
	// Collect unique files from the path set and build one occurrence per file
	pathRanges := make(map[string]struct{ StartLine, EndLine int })

	for _, m := range chain.Matches {
		// Left file
		if _, ok := pathRanges[m.Left.Path]; !ok {
			pathRanges[m.Left.Path] = struct{ StartLine, EndLine int }{
				StartLine: m.Left.StartLine,
				EndLine:   m.Left.EndLine,
			}
		}
		// Right file
		if _, ok := pathRanges[m.Right.Path]; !ok {
			pathRanges[m.Right.Path] = struct{ StartLine, EndLine int }{
				StartLine: m.Right.StartLine,
				EndLine:   m.Right.EndLine,
			}
		}
	}

	// Build sorted list of occurrences
	var occurrences []Occurrence
	var paths []string
	for p := range pathRanges {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		r := pathRanges[path]
		occurrences = append(occurrences, Occurrence{
			Path:      path,
			StartLine: r.StartLine,
			EndLine:   r.EndLine,
		})
	}

	return occurrences
}

// fingerprintFromChain computes the stable fingerprint from the final chain content.
func fingerprintFromChain(chain cloneChain) string {
	// Build path set from chain PathSet
	pathSet := chain.PathSet

	// Include offset in fingerprint to distinguish clones at different positions
	// This prevents duplicate fingerprints for clones at different offsets
	contentKey := fmt.Sprintf("%d-%d-%s", chain.TokenSpan, chain.Offset, pathSet)
	return v3SeedFingerprint(contentKey, pathSet)
}

// findingsKey identifies unique clone bodies for consolidation.
type findingsKey struct {
	TokenSpan int
	PathSet   string
}

// pathSetWrapper wraps a finding with its path set for merge operations.
type pathSetWrapper struct {
	finding coalescedFinding
	pathSet string
}

// mergeFindings consolidates findings representing the same N-way clone.
// IMPORTANT: For N-way clones, all pair findings with same SeedFingerprint should merge.
func mergeFindings(findings []coalescedFinding) []coalescedFinding {
	if len(findings) <= 1 {
		return findings
	}

	// Group by seed fingerprint
	bySeedFP := make(map[string][]coalescedFinding)
	for _, f := range findings {
		seedFP := f.SeedFingerprint
		bySeedFP[seedFP] = append(bySeedFP[seedFP], f)
	}

	// Sort keys for deterministic iteration
	var seedFPs []string
	for sf := range bySeedFP {
		seedFPs = append(seedFPs, sf)
	}
	sort.Strings(seedFPs)

	var merged []coalescedFinding
	for _, sf := range seedFPs {
		group := bySeedFP[sf]
		if len(group) == 0 {
			continue
		}
		if len(group) == 1 {
			merged = append(merged, group[0])
			continue
		}

		// For same SeedFingerprint, collect all unique files across all pair findings
		allFiles := make(map[string]bool)
		for _, f := range group {
			for _, occ := range f.Occurrences {
				allFiles[occ.Path] = true
			}
		}

		// If multiple pair findings cover multiple files, merge into N-way clone
		if len(allFiles) >= 3 && len(group) >= 2 {
			merged = append(merged, mergeToNWayClone(group)...)
		} else {
			// Keep separate for 2-file pairs
			for _, f := range group {
				merged = append(merged, f)
			}
		}
	}

	// Sort deterministically
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].StableFingerprint < merged[j].StableFingerprint
	})

	// Deduplicate by StableFingerprint to handle cases where different seeds
	// produce clones with identical (TokenSpan, Offset, PathSet)
	var deduped []coalescedFinding
	seenFP := make(map[string]bool)
	for _, f := range merged {
		if !seenFP[f.StableFingerprint] {
			seenFP[f.StableFingerprint] = true
			deduped = append(deduped, f)
		}
	}

	return deduped
}

// mergeToNWayClone merges pair findings into an N-way clone.
func mergeToNWayClone(group []coalescedFinding) []coalescedFinding {
	if len(group) == 0 {
		return nil
	}

	if len(group) == 1 {
		return group
	}

	// Collect all occurrences from all pair findings
	var allOccs []Occurrence
	seen := make(map[string]bool)
	for _, f := range group {
		for _, occ := range f.Occurrences {
			if !seen[occ.Path] {
				seen[occ.Path] = true
				allOccs = append(allOccs, occ)
			}
		}
	}
	sort.Slice(allOccs, func(i, j int) bool {
		return allOccs[i].Path < allOccs[j].Path
	})

	// Create merged finding using the first one's data
	merged := group[0]
	merged.Occurrences = allOccs

	return []coalescedFinding{merged}
}

// mergeAllToNWay merges all findings into a single N-way clone.
func mergeAllToNWay(findings []coalescedFinding) coalescedFinding {
	if len(findings) == 0 {
		return coalescedFinding{}
	}
	if len(findings) == 1 {
		return findings[0]
	}

	// Collect all occurrences from all findings
	var allOccs []Occurrence
	seen := make(map[string]bool)
	for _, f := range findings {
		for _, occ := range f.Occurrences {
			if !seen[occ.Path] {
				seen[occ.Path] = true
				allOccs = append(allOccs, occ)
			}
		}
	}
	sort.Slice(allOccs, func(i, j int) bool {
		return allOccs[i].Path < allOccs[j].Path
	})

	merged := findings[0]
	merged.Occurrences = allOccs
	return merged
}

// buildPathSet creates a sorted path set string from occurrences.
func buildPathSet(occurrences []Occurrence) string {
	paths := make(map[string]bool)
	for _, occ := range occurrences {
		paths[occ.Path] = true
	}
	var sortedPaths []string
	for p := range paths {
		sortedPaths = append(sortedPaths, p)
	}
	sort.Strings(sortedPaths)
	return fmt.Sprintf("%v", sortedPaths)
}

// findingsFromChains converts clone chains to coalesced findings.
func findingsFromChains(chains []cloneChain) []coalescedFinding {
	if len(chains) == 0 {
		return nil
	}

	var findings []coalescedFinding
	for _, chain := range chains {
		occurrences := occurrenceFromChain(chain)
		if len(occurrences) < 2 {
			continue
		}

		stableFP := fingerprintFromChain(chain)

		// Capture the first seed fingerprint to distinguish different clones
		var seedFP string
		if len(chain.Matches) > 0 {
			seedFP = chain.Matches[0].SeedFingerprint
		}

		findings = append(findings, coalescedFinding{
			Fingerprint:       truncateFingerprint(stableFP),
			StableFingerprint: stableFP,
			SeedFingerprint:   seedFP,
			TokenCount:        chain.TokenSpan,
			LineCount:         chain.LineSpan,
			Occurrences:       occurrences,
		})
	}

	// Consolidate N-way clones
	findings = mergeFindings(findings)

	// Sort deterministically
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].StableFingerprint < findings[j].StableFingerprint
	})

	return findings
}

// v3CoalesceFindings is the v3 algorithm for finding maximal clones.
func v3CoalesceFindings(windowMap map[string][]rawWindow, fingerprintTokens map[string]int) []coalescedFinding {
	// Step 1: Collect all seed matches across all fingerprints
	// Sort fingerprints to ensure deterministic iteration order
	var fps []string
	for fp := range windowMap {
		fps = append(fps, fp)
	}
	sort.Strings(fps)

	var allMatches []seedMatch
	for _, fp := range fps {
		matches := buildSeedMatches(fp, windowMap[fp])
		allMatches = append(allMatches, matches...)
	}

	if len(allMatches) == 0 {
		return nil
	}

	// Step 2: Chain across all matches (partitioning by file pair + offset)
	chains := buildChainsWithPartitioning(allMatches)

	if len(chains) == 0 {
		return nil
	}

	// Step 3: Convert chains to findings
	return findingsFromChains(chains)
}
