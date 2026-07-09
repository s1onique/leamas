// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"fmt"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// dupCodeVerifier runs the dupcode baseline comparison.
func dupCodeVerifier(root string) []checks.Finding {
	baselinePath := ".factory/dupcode-baseline.json"
	fullBaselinePath := baselinePath
	if root != "." && root != "" {
		fullBaselinePath = filepath.Join(root, baselinePath)
	}

	// Check if baseline exists
	if !checks.FileExists(fullBaselinePath) {
		return []checks.Finding{
			{
				Path:     baselinePath,
				Kind:     "missing_baseline",
				Message:  "baseline file not found. Run 'make dupcode-baseline' to create it.",
				Severity: checks.SeverityError,
			},
		}
	}

	// Load baseline
	baseline, err := dupcode.LoadBaseline(fullBaselinePath)
	if err != nil {
		return []checks.Finding{
			{
				Path:     baselinePath,
				Kind:     "baseline_load_error",
				Message:  fmt.Sprintf("failed to load baseline: %v", err),
				Severity: checks.SeverityError,
			},
		}
	}

	cfg := dupcode.DefaultConfig()
	cfg.Root = root
	cfg.MinLines = baseline.Thresholds.MinLines
	cfg.MinTokens = baseline.Thresholds.MinTokens

	// Get current report
	report, err := dupcode.CheckReport(root, cfg)
	if err != nil {
		return []checks.Finding{
			{
				Path:     "dupcode",
				Kind:     "dupcode_error",
				Message:  fmt.Sprintf("duplicate code scan failed: %v", err),
				Severity: checks.SeverityError,
			},
		}
	}

	// Compare with baseline
	result := dupcode.CompareToBaseline(report, baseline)

	// Convert to gate findings
	return convertDupcodeCompareResult(result)
}

// convertDupcodeCompareResult converts dupcode comparison results to gate findings.
func convertDupcodeCompareResult(result dupcode.CompareResult) []checks.Finding {
	var findings []checks.Finding

	// Report new findings
	for _, f := range result.NewFindings {
		paths := make([]string, len(f.Occurrences))
		for i, occ := range f.Occurrences {
			paths[i] = fmt.Sprintf("%s:%d-%d", occ.Path, occ.StartLine, occ.EndLine)
		}
		firstPath := ""
		if len(f.Occurrences) > 0 {
			firstPath = f.Occurrences[0].Path
		}
		findings = append(findings, checks.Finding{
			Path:     firstPath,
			Kind:     "new_duplicate_code",
			Message:  fmt.Sprintf("NEW: %d tokens, %d occurrences: %v", f.TokenCount, len(f.Occurrences), paths),
			Severity: checks.SeverityError,
		})
	}

	// Report worsened findings
	for _, f := range result.WorsenedFindings {
		newPaths := make([]string, len(f.NewOccurrences))
		for i, occ := range f.NewOccurrences {
			newPaths[i] = fmt.Sprintf("%s:%d-%d", occ.Path, occ.StartLine, occ.EndLine)
		}
		firstPath := ""
		if len(f.BaselineOccurrences) > 0 {
			firstPath = f.BaselineOccurrences[0].Path
		}
		findings = append(findings, checks.Finding{
			Path:     firstPath,
			Kind:     "worsened_duplicate_code",
			Message:  fmt.Sprintf("WORSENED: fingerprint has %d new occurrences: %v", len(f.NewOccurrences), newPaths),
			Severity: checks.SeverityError,
		})
	}

	return findings
}
