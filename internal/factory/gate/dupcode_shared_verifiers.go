// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"fmt"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// DupcodeAnalysisContext holds the analysis context for a factorize invocation.
// It owns the shared dupcode provider and is passed to verifiers that need it.
type DupcodeAnalysisContext struct {
	provider *DupcodeAnalysisProvider
}

// NewDupcodeAnalysisContext creates a new context for one factorize invocation.
func NewDupcodeAnalysisContext(provider *DupcodeAnalysisProvider) *DupcodeAnalysisContext {
	return &DupcodeAnalysisContext{provider: provider}
}

// Provider returns the shared analysis provider.
func (c *DupcodeAnalysisContext) Provider() *DupcodeAnalysisProvider {
	return c.provider
}

// DupcodeVerifierFactory creates verifier functions that use shared analysis.
type DupcodeVerifierFactory struct {
	context *DupcodeAnalysisContext
}

// NewDupcodeVerifierFactory creates a factory for dupcode verifiers.
func NewDupcodeVerifierFactory(ctx *DupcodeAnalysisContext) *DupcodeVerifierFactory {
	return &DupcodeVerifierFactory{context: ctx}
}

// SharedDupCodeVerifier returns a verifier that uses the shared analysis.
// This is used for the "dupcode" verifier in factorize.
func (f *DupcodeVerifierFactory) SharedDupCodeVerifier() func(root string) []checks.Finding {
	return func(root string) []checks.Finding {
		baselinePath := ".factory/dupcode-baseline.json"
		fullBaselinePath := baselinePath
		if root != "." && root != "" {
			fullBaselinePath = filepath.Join(root, baselinePath)
		}

		if !checks.FileExists(fullBaselinePath) {
			return []checks.Finding{
				{Path: baselinePath, Kind: "missing_baseline", Message: "baseline file not found. Run 'make dupcode-baseline' to create it.", Severity: checks.SeverityError},
			}
		}

		baseline, err := dupcode.LoadBaseline(fullBaselinePath)
		if err != nil {
			return []checks.Finding{
				{Path: baselinePath, Kind: "baseline_load_error", Message: fmt.Sprintf("failed to load baseline: %v", err), Severity: checks.SeverityError},
			}
		}

		// Use shared analysis from context
		cfg := dupcode.DefaultConfig()
		cfg.Root = root
		cfg.MinLines = baseline.Thresholds.MinLines
		cfg.MinTokens = baseline.Thresholds.MinTokens
		input := newDupcodeInput(cfg)
		analysis, err := f.context.Provider().ConsumedBy("dupcode", input)
		if err != nil {
			return []checks.Finding{
				{Path: "dupcode", Kind: "dupcode_error", Message: fmt.Sprintf("duplicate code scan failed: %v", err), Severity: checks.SeverityError},
			}
		}

		report := dupcode.Report{
			Findings: analysis.Findings,
			Thresholds: dupcode.BaselineThresholds{
				MinLines:  baseline.Thresholds.MinLines,
				MinTokens: baseline.Thresholds.MinTokens,
			},
			Root: root,
		}

		result := dupcode.CompareToBaseline(report, baseline)
		return convertDupcodeCompareResult(result)
	}
}

// SharedDupcodeBaselineVerifier returns a verifier that uses the shared analysis.
// This is used for the "dupcode-baseline" verifier in factorize.
//
// Architecture:
// 1. ValidateBaselineArtifact: static validation (no scan)
// 2. provider.ConsumedBy: shared analysis (one scan total)
// 3. CheckBaselineDriftFromReport: drift check using shared report (no scan)
//
// The verifier returns early ONLY for terminal conditions (UsableForDrift=false).
// Terminal conditions include: missing, untracked, symlink, non-regular file,
// JSON parse error, schema/algorithm/threshold mismatch.
func (f *DupcodeVerifierFactory) SharedDupcodeBaselineVerifier() func(root string) []checks.Finding {
	return func(root string) []checks.Finding {
		policy := dupcode.DefaultBaselinePolicy()
		// Use repo-relative path for Git operations
		policy.Path = ".factory/dupcode-baseline.json"

		// 1. Static baseline validation (no scan)
		validation, err := dupcode.ValidateBaselineArtifact(root, policy)
		if err != nil {
			return []checks.Finding{{Path: policy.Path, Kind: "baseline_validation_error", Message: fmt.Sprintf("baseline validation failed: %v", err), Severity: checks.SeverityError}}
		}

		// Start with any validation findings (including non-terminal ones)
		result := append([]checks.Finding(nil), validation.Findings...)

		// Return early only if the baseline is not usable for drift comparison
		// (terminal failures: missing, untracked, symlink, non-regular, stat error, malformed)
		if !validation.UsableForDrift {
			return result
		}

		// 2. Get shared analysis (one scan for all consumers)
		cfg := dupcode.DefaultConfig()
		cfg.Root = root
		cfg.MinLines = validation.Baseline.Thresholds.MinLines
		cfg.MinTokens = validation.Baseline.Thresholds.MinTokens
		input := newDupcodeInput(cfg)
		analysis, err := f.context.Provider().ConsumedBy("dupcode-baseline", input)
		if err != nil {
			return append(result, checks.Finding{
				Path:     "dupcode",
				Kind:     "dupcode_error",
				Message:  fmt.Sprintf("duplicate code scan failed: %v", err),
				Severity: checks.SeverityError,
			})
		}

		report := dupcode.Report{
			Findings: analysis.Findings,
			Thresholds: dupcode.BaselineThresholds{
				MinLines:  validation.Baseline.Thresholds.MinLines,
				MinTokens: validation.Baseline.Thresholds.MinTokens,
			},
			Root: root,
		}

		// 3. Drift check using the provided report (no scan)
		// Use root-aware policy path for findings
		driftPolicy := policy
		if root != "." && root != "" {
			driftPolicy.Path = filepath.Join(root, policy.Path)
		}
		driftFindings := dupcode.CheckBaselineDriftFromReport(root, validation.Baseline, report, driftPolicy)
		for _, df := range driftFindings {
			result = append(result, checks.Finding{
				Path:     driftPolicy.Path,
				Kind:     df.Kind,
				Message:  df.Message,
				Severity: checks.SeverityError,
			})
		}
		return result
	}
}
