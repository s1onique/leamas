// SPDX-License-Identifier: Apache-2.0

// Package digest: gate_summary_render_v1.go renders the missing
// and invalid (read/decode/normalize) gate summary states.
// Each renderer emits the binding block ahead of the historical
// verdict so the authoritative qualification is adjacent to
// (and above) the verdict it qualifies.
package digest

import (
	"fmt"
	"strings"

	"github.com/s1onique/leamas/internal/gatesummary"
)

// renderGateSummaryMissing renders the missing source state.
// The binding classifier is invoked with SourceAbsent so the
// section reports NOT_APPLICABLE for the verdict.
func renderGateSummaryMissing(sourcePath string, authority DigestAuthority) string {
	var sb strings.Builder
	binding := EvaluateGateEvidenceBinding(GateSummaryIdentity{}, authority, SourceAbsent)
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=missing\n")
	sb.WriteString("failure_stage=\n")
	sb.WriteString(bindingFieldKV(binding, GateSummaryIdentity{}))
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
// Source is present but unreadable (e.g. directory or permission
// denied). The classifier returns EVIDENCE_INVALID, distinct
// from NOT_APPLICABLE.
func renderGateSummaryInvalidRead(sourcePath string, authority DigestAuthority) string {
	var sb strings.Builder
	binding := EvaluateGateEvidenceBinding(GateSummaryIdentity{}, authority, SourceInvalid)
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=invalid\n")
	sb.WriteString("failure_stage=read\n")
	sb.WriteString(bindingFieldKV(binding, GateSummaryIdentity{}))
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
// The file is present but its JSON is malformed. The classifier
// returns EVIDENCE_INVALID.
func renderGateSummaryInvalidDecode(sourcePath string, diagnostics []gatesummary.Diagnostic, authority DigestAuthority) string {
	var sb strings.Builder
	binding := EvaluateGateEvidenceBinding(GateSummaryIdentity{}, authority, SourceInvalid)
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=invalid\n")
	sb.WriteString("failure_stage=decode\n")
	sb.WriteString(bindingFieldKV(binding, GateSummaryIdentity{}))
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
// The file is present and parses, but fails structural
// normalization. The classifier returns EVIDENCE_INVALID.
func renderGateSummaryInvalidNormalize(sourcePath string, version gatesummary.Version, diagnostics []gatesummary.Diagnostic, authority DigestAuthority) string {
	var sb strings.Builder
	binding := EvaluateGateEvidenceBinding(GateSummaryIdentity{}, authority, SourceInvalid)
	sb.WriteString("## GATE_SUMMARY\n")
	sb.WriteString(fmt.Sprintf("source=%s\n", gateSummaryPath))
	sb.WriteString("source_status=invalid\n")
	sb.WriteString("failure_stage=normalize\n")
	sb.WriteString(bindingFieldKV(binding, GateSummaryIdentity{}))
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
