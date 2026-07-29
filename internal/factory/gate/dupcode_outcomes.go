// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"

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
	GeneratedAt  string              `json:"generated_at,omitempty"`
	Tool         string              `json:"tool,omitempty"`
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
// typed scan report across the data-only dispatch boundary for the
// update-baseline lane.
type DupcodeUpdateBaselineOutcome struct {
	Dispatch verifierdispatch.Result
	Report   DupcodeReportDTO
}

// DupcodeBaselineOutcome carries the typed dispatch result AND the typed
// findings (from VerifyBaseline) across the data-only dispatch boundary.
type DupcodeBaselineOutcome struct {
	Dispatch verifierdispatch.Result
	Findings []checks.Finding
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

// convertCompareResult converts a dupcode.CompareResult into a checks.Finding
// slice. Defined here so the typed dispatch entry points in dupcode_dispatch.go
// can build their typed DupcodeVerifyOutcome payload.
func convertCompareResult(result dupcode.CompareResult) []checks.Finding {
	if !result.HasChanges {
		return nil
	}
	var findings []checks.Finding
	for _, f := range result.NewFindings {
		findings = append(findings, checks.Finding{
			Path:     "dupcode",
			Kind:     "new_duplicate",
			Message:  fmt.Sprintf("new duplicate block (tokens: %d)", f.TokenCount),
			Severity: checks.SeverityError,
		})
	}
	for range result.WorsenedFindings {
		findings = append(findings, checks.Finding{
			Path:     "dupcode",
			Kind:     "worsened_duplicate",
			Message:  "worsened duplicate block",
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
