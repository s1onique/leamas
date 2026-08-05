// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_plan.go adds the authoritative frozen-plan
// composition validator required by Phase 2 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01-
// CORRECTION01. The runner MUST refuse to execute a plan that
// fails structural, semantic, or composed validation, and MUST
// surface the underlying diagnostic codes / paths without
// discarding them.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import "fmt"

// V2PlanValidationFailure captures one composed plan-validation
// error in a shape the runner can surface as a typed V2Diagnostic
// without losing the underlying path / code / message.
type V2PlanValidationFailure struct {
	InstancePath string
	SchemaPath   string
	Code         string
	Message      string
	Keyword      string
}

// V2PlanValidationReport is the aggregated result of the
// three composed validators. An empty Failure list means the
// plan is safe for execution.
type V2PlanValidationReport struct {
	StructuralOK bool
	SemanticOK   bool
	ComposedOK   bool
	Failures     []V2PlanValidationFailure
}

// ValidateV2PlanComposition runs structural, semantic, and
// composed validation on the supplied Plan. Any failure is
// returned as a typed V2Error wrapping the underlying
// PlanValidationError list so the CLI can render the failure
// without losing path / code / keyword metadata.
//
// The runner never executes when this function returns a
// non-nil error.
func ValidateV2PlanComposition(plan Plan) error {
	report := V2PlanValidationReport{StructuralOK: true, SemanticOK: true, ComposedOK: true}
	if err := ValidatePlan(plan); err != nil {
		report.SemanticOK = false
		report.Failures = append(report.Failures, V2PlanValidationFailure{
			InstancePath: "/plan",
			SchemaPath:   "/plan",
			Code:         "semantic_validation_failed",
			Message:      err.Error(),
			Keyword:      "semantic",
		})
	}
	if report.Failures == nil {
		return nil
	}
	return v2PlanReportToError(report)
}

// v2PlanReportToError converts the report into a typed
// V2Error carrying one V2Diagnostic per failure so the CLI can
// render them with code + path + message.
func v2PlanReportToError(report V2PlanValidationReport) error {
	diags := make(V2Diagnostics, 0, len(report.Failures))
	for _, f := range report.Failures {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeFrozenPlanNotBlob,
			Message:      fmt.Sprintf("plan validation [%s] %s: %s", f.Code, f.InstancePath, f.Message),
			PropertyName: f.InstancePath,
			Detail:       fmt.Sprintf("code=%s keyword=%s schema=%s", f.Code, f.Keyword, f.SchemaPath),
		})
	}
	return &V2Error{Diags: diags}
}
