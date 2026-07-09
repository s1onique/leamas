// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/version"
)

// RangeFile represents a file changed in a commit range.
type RangeFile struct {
	Path   string
	From   string
	To     string
	Status string // "added", "modified", "deleted"
}

// GetRangeFiles returns files changed in the given revision range.
func GetRangeFiles(repoRoot, revRange string) ([]RangeFile, error) {
	// Get list of changed files with status using NUL delimiter
	output, err := RunGit(repoRoot, []string{"diff", "--name-status", "-z", revRange})
	if err != nil {
		return nil, err
	}

	parts := splitNULList(output)
	var files []RangeFile

	for i := 0; i < len(parts)-1; i += 2 {
		if parts[i] == "" {
			continue
		}
		status := parts[i]
		path := parts[i+1]

		var from, to string
		switch {
		case status == "A" || strings.HasPrefix(status, "A"):
			// Added: old side is /dev/null, new side is the file
			from = ""
			to = path
		case status == "D" || strings.HasPrefix(status, "D"):
			// Deleted: old side is the file, new side is /dev/null
			from = path
			to = ""
		case strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C"):
			// Renamed or copied: old name followed by new name
			if i+3 < len(parts) {
				from = path
				to = parts[i+3]
				i += 2
			}
		default:
			from = ""
			to = ""
		}

		files = append(files, RangeFile{
			Path:   path,
			From:   from,
			To:     to,
			Status: statusToHuman(status),
		})
	}

	// Deduplicate files and sort
	files = UniqueRangeFiles(files)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// RenderRangeDigest creates digest for commit range changes.
func RenderRangeDigest(repoRoot string, files []RangeFile, revRange string) (string, error) {
	resolved := &ResolvedMode{
		Mode:   ModeRange,
		Range:  revRange,
		Reason: "explicit range mode",
	}
	return RenderRangeDigestWithResolved(repoRoot, files, resolved)
}

// RenderRangeDigestWithResolved creates digest for commit range with resolved mode info.
func RenderRangeDigestWithResolved(repoRoot string, files []RangeFile, resolved *ResolvedMode) (string, error) {
	var sb strings.Builder

	// Get version metadata and timestamp once
	v := version.Get()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	// Contract header - prepend versioned metadata
	headerInfo := HeaderInfo{
		Version:   v.Version,
		Commit:    v.Commit,
		BuildTime: v.BuildTime,
		Mode:      resolved.Mode,
		CreatedAt: createdAt,
	}
	sb.WriteString(RenderContractHeader(headerInfo))

	// Build review evidence sections from manifest
	manifest := BuildRangeManifest(files)
	stats := ComputeStats(manifest, repoRoot)
	reviewMap := BuildReviewMap(manifest, repoRoot)
	riskSignals := ComputeRiskSignals(stats, manifest, repoRoot)

	// Legacy header (preserved for backwards compatibility)
	sb.WriteString("# Targeted digest\n\n")
	sb.WriteString(fmt.Sprintf("Generated at: %s\n", createdAt))
	sb.WriteString(fmt.Sprintf("Repo: %s\n", repoRoot))
	sb.WriteString(fmt.Sprintf("Mode: %s\n", resolved.Mode))
	sb.WriteString(fmt.Sprintf("Range: %s\n", resolved.Range))
	// Only show resolved info for auto mode, not explicit range mode
	if resolved.Reason != "explicit range mode" {
		sb.WriteString(fmt.Sprintf("Resolved from: auto\n"))
		sb.WriteString(fmt.Sprintf("Reason: %s\n", resolved.Reason))
	}
	sb.WriteString("\n")

	// Render review evidence sections (v2 contract) - store for hashing
	manifestSection := RenderManifest(manifest)
	statsSection := RenderStats(stats)
	reviewMapSection := RenderReviewMap(reviewMap)
	riskSignalsSection := RenderRiskSignals(riskSignals)

	sb.WriteString(manifestSection)
	sb.WriteString("\n")
	sb.WriteString(statsSection)
	sb.WriteString("\n")
	sb.WriteString(reviewMapSection)
	sb.WriteString("\n")
	sb.WriteString(riskSignalsSection)
	sb.WriteString("\n")

	// PATCH_HYGIENE section
	patchHygiene := RunPatchHygiene(repoRoot, resolved.Range)
	patchHygieneSection := RenderPatchHygiene(patchHygiene)
	sb.WriteString(patchHygieneSection)

	// Build file evidence section for hashing (using shared function)
	fileEvidenceSection := RenderRangeFileEvidence(repoRoot, files, resolved.Range)

	// Compute evidence hashes
	evidenceHashes := ComputeEvidenceHashes(
		manifestSection,
		statsSection,
		reviewMapSection,
		riskSignalsSection,
		patchHygieneSection,
		fileEvidenceSection,
	)
	sb.WriteString(RenderEvidenceHashes(evidenceHashes))

	// Write Changed files and Diffs sections
	sb.WriteString(fileEvidenceSection)

	sb.WriteString("\n## Workflow anchors\n")

	// Load and render anchors
	anchorsConfig, err := LoadAnchors(repoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to load anchors: %w", err)
	}
	sb.WriteString(RenderAnchors(anchorsConfig))

	return sb.String(), nil
}

// statusToHuman converts git status letter to human-readable string.
func statusToHuman(status string) string {
	switch status {
	case "A":
		return "added"
	case "D":
		return "deleted"
	case "M":
		return "modified"
	case "R":
		return "renamed"
	case "C":
		return "copied"
	default:
		return status
	}
}

// splitNULList splits NUL-delimited string into slice.
func splitNULList(output string) []string {
	if output == "" {
		return nil
	}
	parts := strings.Split(output, "\x00")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// UniqueRangeFiles removes duplicate files from a range file list.
// It preserves the first-seen occurrence and maintains stable ordering.
func UniqueRangeFiles(files []RangeFile) []RangeFile {
	if len(files) <= 1 {
		return files
	}

	seen := make(map[string]bool)
	result := make([]RangeFile, 0, len(files))

	for _, f := range files {
		if !seen[f.Path] {
			seen[f.Path] = true
			result = append(result, f)
		}
	}

	return result
}

// RenderDigestWithResolved creates digest with resolved mode information.
func RenderDigestWithResolved(mode Mode, repoRoot string, files []ChangedFile, resolved *ResolvedMode, explicit bool) (string, error) {
	var sb strings.Builder

	// Get version metadata and timestamp once
	v := version.Get()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	// Contract header - prepend versioned metadata
	headerInfo := HeaderInfo{
		Version:   v.Version,
		Commit:    v.Commit,
		BuildTime: v.BuildTime,
		Mode:      mode,
		CreatedAt: createdAt,
	}
	sb.WriteString(RenderContractHeader(headerInfo))

	// Build review evidence sections from manifest
	manifest := BuildManifest(files)
	stats := ComputeStats(manifest, repoRoot)
	reviewMap := BuildReviewMap(manifest, repoRoot)
	riskSignals := ComputeRiskSignals(stats, manifest, repoRoot)

	// Legacy header (preserved for backwards compatibility)
	sb.WriteString("# Targeted digest\n\n")
	sb.WriteString(fmt.Sprintf("Generated at: %s\n", createdAt))
	sb.WriteString(fmt.Sprintf("Repo: %s\n", repoRoot))
	sb.WriteString(fmt.Sprintf("Mode: %s\n", mode))

	// Show resolved info only for auto mode
	if resolved != nil && !explicit {
		sb.WriteString("Resolved from: auto\n")
		sb.WriteString(fmt.Sprintf("Reason: %s\n", resolved.Reason))
	}

	sb.WriteString("\n")

	// Render review evidence sections (v2 contract) - store for hashing
	manifestSection := RenderManifest(manifest)
	statsSection := RenderStats(stats)
	reviewMapSection := RenderReviewMap(reviewMap)
	riskSignalsSection := RenderRiskSignals(riskSignals)

	sb.WriteString(manifestSection)
	sb.WriteString("\n")
	sb.WriteString(statsSection)
	sb.WriteString("\n")
	sb.WriteString(reviewMapSection)
	sb.WriteString("\n")
	sb.WriteString(riskSignalsSection)
	sb.WriteString("\n")

	// PATCH_HYGIENE section (only for dirty mode)
	var patchHygiene PatchHygiene
	if mode == ModeDirty {
		patchHygiene = RunPatchHygieneDirty(repoRoot)
	}
	patchHygieneSection := RenderPatchHygiene(patchHygiene)
	sb.WriteString(patchHygieneSection)

	// Build file evidence section for hashing (using shared function)
	fileEvidenceSection := RenderChangedFilesAndDiffs(repoRoot, files)

	// Compute evidence hashes
	evidenceHashes := ComputeEvidenceHashes(
		manifestSection,
		statsSection,
		reviewMapSection,
		riskSignalsSection,
		patchHygieneSection,
		fileEvidenceSection,
	)
	sb.WriteString(RenderEvidenceHashes(evidenceHashes))

	// Write Changed files and Diffs sections
	sb.WriteString(fileEvidenceSection)

	sb.WriteString("\n## Workflow anchors\n")

	// Load and render anchors
	anchorsConfig, err := LoadAnchors(repoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to load anchors: %w", err)
	}
	sb.WriteString(RenderAnchors(anchorsConfig))

	return sb.String(), nil
}
