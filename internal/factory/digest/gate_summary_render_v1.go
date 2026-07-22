// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"

	"strings"

	"github.com/s1onique/leamas/internal/gatesummary"
)

// renderGateSummaryMissing renders the missing source state.
func renderGateSummaryMissing(sourcePath string) string {
	var sb strings.Builder
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=missing\n")
	sb.WriteString("failure_stage=\n")
	sb.WriteString("schema_version=0\n")
	sb.WriteString("generated_at=\n")
	sb.WriteString("overall_status=unavailable\n")
	sb.WriteString("checks_total=0\n")
	sb.WriteString("checks_passed=0\n")
	sb.WriteString("checks_failed=0\n")
	sb.WriteString("checks_skipped=0\n")
	sb.WriteString("checks_unavailable=0\n")
	return sb.String()
}

// renderGateSummaryInvalidRead renders the invalid/read source state.
func renderGateSummaryInvalidRead(sourcePath string) string {
	var sb strings.Builder
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=invalid\n")
	sb.WriteString("failure_stage=read\n")
	sb.WriteString("schema_version=0\n")
	sb.WriteString("generated_at=\n")
	sb.WriteString("overall_status=unavailable\n")
	sb.WriteString("checks_total=0\n")
	sb.WriteString("checks_passed=0\n")
	sb.WriteString("checks_failed=0\n")
	sb.WriteString("checks_skipped=0\n")
	sb.WriteString("checks_unavailable=0\n")
	sb.WriteString("diagnostics_total=1\n")
	sb.WriteString("diagnostics:\n")
	sb.WriteString(fmt.Sprintf("  - code=%s path=%s\n", diagnosticCodeReadFailed, diagnosticPath))
	return sb.String()
}

// renderGateSummaryInvalidDecode renders the invalid/decode source state.
func renderGateSummaryInvalidDecode(sourcePath string, diagnostics []gatesummary.Diagnostic) string {
	var sb strings.Builder
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=invalid\n")
	sb.WriteString("failure_stage=decode\n")
	sb.WriteString("schema_version=0\n")
	sb.WriteString("generated_at=\n")
	sb.WriteString("overall_status=unavailable\n")
	sb.WriteString("checks_total=0\n")
	sb.WriteString("checks_passed=0\n")
	sb.WriteString("checks_failed=0\n")
	sb.WriteString("checks_skipped=0\n")
	sb.WriteString("checks_unavailable=0\n")
	sb.WriteString(fmt.Sprintf("diagnostics_total=%d\n", len(diagnostics)))
	sb.WriteString("diagnostics:\n")
	for _, d := range diagnostics {
		sb.WriteString(fmt.Sprintf("  - code=%s path=%s\n", sanitizeLine(d.Code), sanitizeLine(d.Path)))
	}
	return sb.String()
}

// renderGateSummaryInvalidNormalize renders the invalid/normalize source state.
func renderGateSummaryInvalidNormalize(sourcePath string, version gatesummary.Version, diagnostics []gatesummary.Diagnostic) string {
	var sb strings.Builder
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=invalid\n")
	sb.WriteString("failure_stage=normalize\n")
	sb.WriteString(fmt.Sprintf("schema_version=%d\n", version))
	sb.WriteString("generated_at=\n")
	sb.WriteString("overall_status=unavailable\n")
	sb.WriteString("checks_total=0\n")
	sb.WriteString("checks_passed=0\n")
	sb.WriteString("checks_failed=0\n")
	sb.WriteString("checks_skipped=0\n")
	sb.WriteString("checks_unavailable=0\n")
	sb.WriteString(fmt.Sprintf("diagnostics_total=%d\n", len(diagnostics)))
	sb.WriteString("diagnostics:\n")
	for _, d := range diagnostics {
		sb.WriteString(fmt.Sprintf("  - code=%s path=%s\n", sanitizeLine(d.Code), sanitizeLine(d.Path)))
	}
	return sb.String()
}
