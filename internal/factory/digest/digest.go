// Package digest provides targeted digest generation for Git repositories.
// It creates reviewable artifacts of repository changes suitable for
// agent-assisted review workflows.
package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/version"
)

// Options configures digest generation.
type Options struct {
	// RepoRoot is the absolute path to the Git repository root.
	RepoRoot string
	// Mode determines which changes to include.
	Mode Mode
	// Output is the path to write the digest file.
	Output string
	// Range is the commit range for ModeRange (e.g., "HEAD~1..HEAD").
	Range string
}

// Generate creates a targeted digest and returns it as a string.
func Generate(opts Options) (string, error) {
	repoRoot := opts.RepoRoot
	if repoRoot == "" {
		var err error
		repoRoot, err = DetectRepoRoot()
		if err != nil {
			return "", fmt.Errorf("failed to detect repo root: %w", err)
		}
	}

	mode := opts.Mode
	if mode == "" {
		mode = ModeAuto // default to auto mode
	}

	// Handle auto mode
	if mode == ModeAuto {
		resolved, err := ResolveAutoMode(repoRoot)
		if err != nil {
			return "", fmt.Errorf("failed to resolve auto mode: %w", err)
		}

		if resolved.Mode == ModeDirty {
			files, err := GetDirtyFiles(repoRoot)
			if err != nil {
				return "", fmt.Errorf("failed to get dirty files: %w", err)
			}
			return RenderDigestWithResolved(ModeDirty, repoRoot, files, resolved, false)
		}

		// Clean working tree: use range mode with HEAD~1..HEAD
		files, err := GetRangeFiles(repoRoot, resolved.Range)
		if err != nil {
			return "", fmt.Errorf("failed to get range files: %w", err)
		}
		return RenderRangeDigestWithResolved(repoRoot, files, resolved)
	}

	// Handle explicit modes
	switch mode {
	case ModeDirty:
		files, err := GetDirtyFiles(repoRoot)
		if err != nil {
			return "", fmt.Errorf("failed to get dirty files: %w", err)
		}
		return RenderDigest(mode, repoRoot, files)
	case ModeStaged:
		files, err := GetStagedFiles(repoRoot)
		if err != nil {
			return "", fmt.Errorf("failed to get staged files: %w", err)
		}
		return RenderDigest(mode, repoRoot, files)
	case ModeRange:
		if opts.Range == "" {
			return "", fmt.Errorf("ModeRange requires --range option")
		}
		files, err := GetRangeFiles(repoRoot, opts.Range)
		if err != nil {
			return "", fmt.Errorf("failed to get range files: %w", err)
		}
		return RenderRangeDigest(repoRoot, files, opts.Range)
	default:
		return "", fmt.Errorf("unsupported mode: %s", mode)
	}
}

// Write generates a digest and writes it to the output file.
// The digest content is redacted before writing to prevent secret exposure.
// For source files: content is preserved for review fidelity, warnings are emitted.
// For non-source files: standard redaction is applied.
func Write(opts Options) error {
	content, err := Generate(opts)
	if err != nil {
		return err
	}

	// Apply source-aware redaction policy
	// - Source files (.py, .go, etc.) are preserved with warning metadata
	// - Non-source files (logs, config, env, etc.) are redacted
	warnings, err := WriteWithWarnings(opts, content)
	if err != nil {
		return err
	}

	// Log warnings if any source secrets were detected (optional, for visibility)
	if len(warnings) > 0 {
		// Warnings are already included in the digest content
		// This is just for any additional logging if needed
		_ = warnings
	}

	return nil
}

// WriteWithWarnings generates a digest with policy-aware redaction and returns warnings.
// This allows callers to handle warnings separately if needed.
func WriteWithWarnings(opts Options, content string) ([]SourceSecretWarning, error) {
	// Apply source-aware redaction policy
	// Source files are preserved, non-source files are redacted
	redactedContent, warnings := RedactDigestWithPolicy(content)

	// Create parent directory if needed
	dir := filepath.Dir(opts.Output)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return warnings, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Write the digest with redaction applied
	if err := os.WriteFile(opts.Output, []byte(redactedContent), 0644); err != nil {
		return warnings, fmt.Errorf("failed to write digest: %w", err)
	}

	return warnings, nil
}

// RenderDigest creates the markdown digest content.
func RenderDigest(mode Mode, repoRoot string, files []ChangedFile) (string, error) {
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
	sb.WriteString("\n")

	// Render review evidence sections (v2 contract)
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
	var patchHygiene PatchHygiene
	if mode == ModeDirty {
		patchHygiene = RunPatchHygieneDirty(repoRoot)
	} else if mode == ModeStaged {
		patchHygiene = RunPatchHygiene(repoRoot, "--cached")
	}
	patchHygieneSection := RenderPatchHygiene(patchHygiene)
	sb.WriteString(patchHygieneSection)

	// PUBLIC_SURFACE_DELTA section - compute before writing to include in evidence hashes
	publicSurfaceDeltaSection := RenderEmptyPublicSurfaceDelta()
	if delta, err := CollectPublicSurfaceDelta(mode, repoRoot, files); err == nil {
		publicSurfaceDeltaSection = RenderPublicSurfaceDelta(delta)
	}

	// DEPENDENCY_DELTA section - compute before writing to include in evidence hashes
	dependencyDeltaSection := RenderEmptyDependencyDelta()
	if delta, err := CollectDependencyDelta(mode, repoRoot, files); err == nil {
		dependencyDeltaSection = RenderDependencyDelta(delta)
	}

	// File evidence section for hashing
	fileEvidenceSection := RenderChangedFilesAndDiffs(repoRoot, files)

	// GATE_SUMMARY section - compute before writing to include in evidence hashes
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

	// Compute evidence hashes (includes PUBLIC_SURFACE_DELTA, DEPENDENCY_DELTA, and GATE_SUMMARY)
	evidenceHashes := ComputeEvidenceHashes(
		manifestSection,
		statsSection,
		reviewMapSection,
		riskSignalsSection,
		patchHygieneSection,
		publicSurfaceDeltaSection,
		dependencyDeltaSection,
		gateSummarySection,
		fileEvidenceSection,
	)

	// Write EVIDENCE_HASHES before GATE_SUMMARY (contract order)
	sb.WriteString(RenderEvidenceHashes(evidenceHashes))

	// Write GATE_SUMMARY section
	sb.WriteString(gateSummarySection)

	// Write PUBLIC_SURFACE_DELTA section
	sb.WriteString(publicSurfaceDeltaSection)
	sb.WriteString("\n")

	// Write DEPENDENCY_DELTA section
	sb.WriteString(dependencyDeltaSection)

	// Write Changed files and Diffs sections
	sb.WriteString(fileEvidenceSection)

	_ = gateSummaryErr // suppress unused warning

	sb.WriteString("\n## Workflow anchors\n")

	// Load and render anchors
	anchorsConfig, err := LoadAnchors(repoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to load anchors: %w", err)
	}
	sb.WriteString(RenderAnchors(anchorsConfig))

	return sb.String(), nil
}
