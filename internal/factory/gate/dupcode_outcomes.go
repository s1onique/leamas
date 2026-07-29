// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// DupcodeFindingDTO is a gate-owned DTO for a single dupcode finding.
type DupcodeFindingDTO struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
	Fingerprint string `json:"fingerprint,omitempty"`
	TokenCount  int    `json:"token_count,omitempty"`
}

// DupcodeReportDTO is a gate-owned DTO that carries the typed scan report.
type DupcodeReportDTO struct {
	Findings     []DupcodeFindingDTO `json:"findings"`
	MinLines     int                 `json:"min_lines"`
	MinTokens    int                 `json:"min_tokens"`
	Root         string              `json:"root,omitempty"`
	FindingCount int                 `json:"finding_count"`
}

// DupcodeComparisonDTO is a gate-owned DTO for the baseline comparison.
type DupcodeComparisonDTO struct {
	HasChanges       bool                `json:"has_changes"`
	NewCount         int                 `json:"new_count"`
	WorsenedCount    int                 `json:"worsened_count"`
	NewFindings      []DupcodeFindingDTO `json:"new_findings"`
	WorsenedFindings []DupcodeFindingDTO `json:"worsened_findings"`
}

// DupcodeVerifyOutcome carries the typed dispatch result AND the typed
// scan/comparison payload across the data-only dispatch boundary.
type DupcodeVerifyOutcome struct {
	Dispatch   verifierdispatch.Result
	Report     DupcodeReportDTO
	Comparison DupcodeComparisonDTO
}

// DupcodeUpdateBaselineOutcome carries the typed dispatch result AND the
// typed scan report across the data-only dispatch boundary.
type DupcodeUpdateBaselineOutcome struct {
	Dispatch verifierdispatch.Result
	Report   DupcodeReportDTO
}

// DupcodeBaselineOutcome carries the typed dispatch result AND the typed
// findings.
type DupcodeBaselineOutcome struct {
	Dispatch verifierdispatch.Result
	Findings []checks.Finding
}

// baselineFindingPath returns the first occurrence path for a baseline
// finding, or "" if absent.
func baselineFindingPath(f dupcode.BaselineFinding) string {
	if len(f.Occurrences) > 0 {
		return f.Occurrences[0].Path
	}
	return ""
}

// newFindingPath returns the first occurrence path for a new finding.
func newFindingPath(f dupcode.NewFinding) string {
	if len(f.Occurrences) > 0 {
		return f.Occurrences[0].Path
	}
	return ""
}

// worsenedFindingPath returns the first new-occurrence path for a
// worsened finding, falling back to baseline occurrence.
func worsenedFindingPath(f dupcode.WorsenedFinding) string {
	if len(f.NewOccurrences) > 0 {
		return f.NewOccurrences[0].Path
	}
	if len(f.BaselineOccurrences) > 0 {
		return f.BaselineOccurrences[0].Path
	}
	return ""
}

// findingPath returns the first occurrence path for a scan finding.
func findingPath(f dupcode.Finding) string {
	if len(f.Occurrences) > 0 {
		return f.Occurrences[0].Path
	}
	return ""
}

// baselineFindingsToDTO converts a slice of dupcode.BaselineFinding into
// gate DTOs.
func baselineFindingsToDTO(src []dupcode.BaselineFinding) []DupcodeFindingDTO {
	out := make([]DupcodeFindingDTO, 0, len(src))
	for _, f := range src {
		out = append(out, DupcodeFindingDTO{
			Path:        baselineFindingPath(f),
			Kind:        "duplicate",
			Message:    "duplicate block",
			Severity:    "error",
			Fingerprint: f.Fingerprint,
			TokenCount:  f.TokenCount,
		})
	}
	return out
}

// newFindingsToDTO converts a slice of dupcode.NewFinding into gate DTOs.
func newFindingsToDTO(src []dupcode.NewFinding) []DupcodeFindingDTO {
	out := make([]DupcodeFindingDTO, 0, len(src))
	for _, f := range src {
		out = append(out, DupcodeFindingDTO{
			Path:        newFindingPath(f),
			Kind:        "new_duplicate",
			Message:    "new duplicate",
			Severity:    "error",
			Fingerprint: f.Fingerprint,
			TokenCount:  f.TokenCount,
		})
	}
	return out
}

// worsenedFindingsToDTO converts a slice of dupcode.WorsenedFinding into
// gate DTOs.
func worsenedFindingsToDTO(src []dupcode.WorsenedFinding) []DupcodeFindingDTO {
	out := make([]DupcodeFindingDTO, 0, len(src))
	for _, f := range src {
		out = append(out, DupcodeFindingDTO{
			Path:        worsenedFindingPath(f),
			Kind:        "worsened_duplicate",
			Message:    "worsened duplicate",
			Severity:    "error",
			Fingerprint: f.Fingerprint,
			TokenCount:  f.TotalNow,
		})
	}
	return out
}

// findingsToDTO converts a slice of checks.Finding into gate-owned DTOs.
func findingsToDTO(src []checks.Finding) []DupcodeFindingDTO {
	out := make([]DupcodeFindingDTO, 0, len(src))
	for _, f := range src {
		out = append(out, DupcodeFindingDTO{
			Path:     f.Path,
			Kind:     f.Kind,
			Message:  f.Message,
			Severity: severityString(f.Severity),
		})
	}
	return out
}

func severityString(s checks.Severity) string {
	switch s {
	case checks.SeverityError:
		return "error"
	case checks.SeverityWarn:
		return "warning"
	default:
		return ""
	}
}

// reportToDTO converts an exact dupcode.Report into a typed DupcodeReportDTO
// preserving the real findings, thresholds, and root.
func reportToDTO(report dupcode.Report, spec DupcodeVerifySpec) DupcodeReportDTO {
	findings := make([]DupcodeFindingDTO, 0, len(report.Findings))
	for _, f := range report.Findings {
		findings = append(findings, DupcodeFindingDTO{
			Path:        findingPath(f),
			Kind:        "duplicate",
			Message:    "duplicate block",
			Severity:    "error",
			Fingerprint: f.Fingerprint,
			TokenCount:  f.TokenCount,
		})
	}
	return DupcodeReportDTO{
		Findings:     findings,
		MinLines:     spec.MinLines,
		MinTokens:    spec.MinTokens,
		Root:         report.Root,
		FindingCount: len(findings),
	}
}

// compareResultToDTO converts an exact dupcode.CompareResult into a typed
// DupcodeComparisonDTO preserving new_count, worsened_count, has_changes,
// and the real new/worsened finding lists.
func compareResultToDTO(result dupcode.CompareResult) DupcodeComparisonDTO {
	return DupcodeComparisonDTO{
		HasChanges:       result.HasChanges,
		NewCount:         len(result.NewFindings),
		WorsenedCount:    len(result.WorsenedFindings),
		NewFindings:      newFindingsToDTO(result.NewFindings),
		WorsenedFindings: worsenedFindingsToDTO(result.WorsenedFindings),
	}
}

// convertCompareResult converts a dupcode.CompareResult into a checks.Finding
// slice. Used by the legacy named bound runners.
func convertCompareResult(result dupcode.CompareResult) []checks.Finding {
	if !result.HasChanges {
		return nil
	}
	var findings []checks.Finding
	for range result.NewFindings {
		findings = append(findings, checks.Finding{
			Path:     "dupcode",
			Kind:     "new_duplicate",
			Message:  "new duplicate",
			Severity: checks.SeverityError,
		})
	}
	for range result.WorsenedFindings {
		findings = append(findings, checks.Finding{
			Path:     "dupcode",
			Kind:     "worsened_duplicate",
			Message:  "worsened duplicate",
			Severity: checks.SeverityError,
		})
	}
	return findings
}

// findingsToReport converts a checks.Finding slice into a typed
// DupcodeReportDTO carrying the established CLI payload fields.
func findingsToReport(findings []checks.Finding, spec DupcodeVerifySpec) DupcodeReportDTO {
	dtos := findingsToDTO(findings)
	return DupcodeReportDTO{
		Findings:     dtos,
		MinLines:     spec.MinLines,
		MinTokens:    spec.MinTokens,
		FindingCount: len(dtos),
	}
}

// findingsToComparison converts a checks.Finding slice into a typed
// DupcodeComparisonDTO.
func findingsToComparison(findings []checks.Finding) DupcodeComparisonDTO {
	dtos := findingsToDTO(findings)
	var newCount, worsenedCount int
	var newFindings, worsenedFindings []DupcodeFindingDTO
	for _, d := range dtos {
		switch d.Kind {
		case "new_duplicate":
			newCount++
			newFindings = append(newFindings, d)
		case "worsened_duplicate":
			worsenedCount++
			worsenedFindings = append(worsenedFindings, d)
		}
	}
	return DupcodeComparisonDTO{
		HasChanges:       len(newFindings) > 0 || len(worsenedFindings) > 0,
		NewCount:         newCount,
		WorsenedCount:    worsenedCount,
		NewFindings:      newFindings,
		WorsenedFindings: worsenedFindings,
	}
}