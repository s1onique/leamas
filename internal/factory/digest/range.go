// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/version"
)

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

	v := version.Get()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	headerInfo := HeaderInfo{
		Version:   v.Version,
		Commit:    v.Commit,
		BuildTime: v.BuildTime,
		Mode:      resolved.Mode,
		CreatedAt: createdAt,
	}
	sb.WriteString(RenderContractHeader(headerInfo))

	manifest := BuildRangeManifest(files)
	stats := ComputeStats(manifest, repoRoot)
	reviewMap := BuildReviewMap(manifest, repoRoot)
	riskSignals := ComputeRiskSignals(stats, manifest, repoRoot)

	sb.WriteString("# Targeted digest\n\n")
	sb.WriteString(fmt.Sprintf("Generated at: %s\n", createdAt))
	sb.WriteString(fmt.Sprintf("Repo: %s\n", repoRoot))
	sb.WriteString(fmt.Sprintf("Mode: %s\n", resolved.Mode))
	sb.WriteString(fmt.Sprintf("Range: %s\n", resolved.Range))
	if resolved.Reason != "explicit range mode" {
		sb.WriteString(fmt.Sprintf("Resolved from: auto\n"))
		sb.WriteString(fmt.Sprintf("Reason: %s\n", resolved.Reason))
	}
	sb.WriteString("\n")

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

	patchHygiene := RunPatchHygiene(repoRoot, resolved.Range)
	patchHygieneSection := RenderPatchHygiene(patchHygiene)
	sb.WriteString(patchHygieneSection)

	publicSurfaceDeltaSection := RenderEmptyPublicSurfaceDelta()
	if delta, err := CollectRangePublicSurfaceDelta(repoRoot, files, resolved.Range); err == nil {
		publicSurfaceDeltaSection = RenderPublicSurfaceDelta(delta)
	}

	dependencyDeltaSection := RenderEmptyDependencyDelta()
	if delta, err := CollectRangeDependencyDelta(repoRoot, files, resolved.Range); err == nil {
		dependencyDeltaSection = RenderDependencyDelta(delta)
	}

	fileEvidenceSection := RenderRangeFileEvidence(repoRoot, files, resolved.Range)

	gateSummaryPath := filepath.Join(repoRoot, ".factory", "gate-summary.json")
	var gateSummarySection string
	var gateSummaryErr error
	if gate.GateSummaryExists(gateSummaryPath) {
		if gs, err := gate.ReadGateSummary(gateSummaryPath); err == nil {
			gateSummarySection = gate.RenderGateSummary(gs, nil)
		} else {
			gateSummaryErr = err
			gateSummarySection = gate.RenderGateSummary(nil, err)
		}
	} else {
		gateSummarySection = gate.RenderGateSummary(nil, nil)
	}

	evidenceHashes := ComputeEvidenceHashes(
		manifestSection, statsSection, reviewMapSection, riskSignalsSection,
		patchHygieneSection, publicSurfaceDeltaSection, dependencyDeltaSection,
		gateSummarySection, fileEvidenceSection,
	)

	sb.WriteString(RenderEvidenceHashes(evidenceHashes))
	sb.WriteString(gateSummarySection)
	sb.WriteString(publicSurfaceDeltaSection)
	sb.WriteString("\n")
	sb.WriteString(dependencyDeltaSection)
	sb.WriteString(fileEvidenceSection)

	_ = gateSummaryErr

	sb.WriteString("\n## Workflow anchors\n")

	anchorsConfig, err := LoadAnchors(repoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to load anchors: %w", err)
	}
	sb.WriteString(RenderAnchors(anchorsConfig))

	return sb.String(), nil
}

// RenderDigestWithResolved creates digest with resolved mode information.
func RenderDigestWithResolved(mode Mode, repoRoot string, files []ChangedFile, resolved *ResolvedMode, explicit bool) (string, error) {
	var sb strings.Builder

	v := version.Get()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	headerInfo := HeaderInfo{
		Version:   v.Version,
		Commit:    v.Commit,
		BuildTime: v.BuildTime,
		Mode:      mode,
		CreatedAt: createdAt,
	}
	sb.WriteString(RenderContractHeader(headerInfo))

	manifest := BuildManifest(files)
	stats := ComputeStats(manifest, repoRoot)
	reviewMap := BuildReviewMap(manifest, repoRoot)
	riskSignals := ComputeRiskSignals(stats, manifest, repoRoot)

	sb.WriteString("# Targeted digest\n\n")
	sb.WriteString(fmt.Sprintf("Generated at: %s\n", createdAt))
	sb.WriteString(fmt.Sprintf("Repo: %s\n", repoRoot))
	sb.WriteString(fmt.Sprintf("Mode: %s\n", mode))

	if resolved != nil && !explicit {
		sb.WriteString("Resolved from: auto\n")
		sb.WriteString(fmt.Sprintf("Reason: %s\n", resolved.Reason))
	}
	sb.WriteString("\n")

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

	var patchHygiene PatchHygiene
	if mode == ModeDirty {
		patchHygiene = RunPatchHygieneDirty(repoRoot)
	}
	patchHygieneSection := RenderPatchHygiene(patchHygiene)
	sb.WriteString(patchHygieneSection)

	publicSurfaceDeltaSection := RenderEmptyPublicSurfaceDelta()
	if delta, err := CollectPublicSurfaceDelta(mode, repoRoot, files); err == nil {
		publicSurfaceDeltaSection = RenderPublicSurfaceDelta(delta)
	}

	dependencyDeltaSection := RenderEmptyDependencyDelta()
	if delta, err := CollectDependencyDelta(mode, repoRoot, files); err == nil {
		dependencyDeltaSection = RenderDependencyDelta(delta)
	}

	fileEvidenceSection := RenderChangedFilesAndDiffs(repoRoot, files)

	gateSummaryPath := filepath.Join(repoRoot, ".factory", "gate-summary.json")
	var gateSummarySection string
	var gateSummaryErr error
	if gate.GateSummaryExists(gateSummaryPath) {
		if gs, err := gate.ReadGateSummary(gateSummaryPath); err == nil {
			gateSummarySection = gate.RenderGateSummary(gs, nil)
		} else {
			gateSummaryErr = err
			gateSummarySection = gate.RenderGateSummary(nil, err)
		}
	} else {
		gateSummarySection = gate.RenderGateSummary(nil, nil)
	}

	evidenceHashes := ComputeEvidenceHashes(
		manifestSection, statsSection, reviewMapSection, riskSignalsSection,
		patchHygieneSection, publicSurfaceDeltaSection, dependencyDeltaSection,
		gateSummarySection, fileEvidenceSection,
	)

	sb.WriteString(RenderEvidenceHashes(evidenceHashes))
	sb.WriteString(gateSummarySection)
	sb.WriteString(publicSurfaceDeltaSection)
	sb.WriteString("\n")
	sb.WriteString(dependencyDeltaSection)
	sb.WriteString(fileEvidenceSection)

	_ = gateSummaryErr

	sb.WriteString("\n## Workflow anchors\n")

	anchorsConfig, err := LoadAnchors(repoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to load anchors: %w", err)
	}
	sb.WriteString(RenderAnchors(anchorsConfig))

	return sb.String(), nil
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
