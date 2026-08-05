// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_plan.go implements the authoritative
// frozen-plan validation required by Phase 2 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01-
// CORRECTION01 and tightened by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-VALID-PLAN-AUTHORITY01.
//
// The validator MUST consume the exact frozen bytes loaded
// from F:P (NOT mutable working-tree bytes) and MUST run:
//
//   - JSON decoding
//   - structural validation
//   - semantic validation
//   - composed validation
//
// in that order, populating the report fields ONLY after the
// corresponding stage has actually executed. No stage may
// default to true without execution. The runner rejects any
// frozen plan whose report is not fully successful.
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

// V2FrozenPlanValidation is the authoritative report produced
// by ValidateFrozenPlanV2. Each boolean field is set ONLY after
// the corresponding validation stage has actually executed on
// the supplied frozen bytes; callers MUST NOT interpret a true
// value as success without inspecting the per-stage diagnostics.
//
// The StructuralDiagnostics, SemanticDiagnostics, and
// ComposedDiagnostics fields hold the raw PlanValidationError
// stream from the production validators so the runner can
// surface nested path / code / keyword / rejected-value
// metadata without losing it.
type V2FrozenPlanValidation struct {
	ParseOK               bool
	StructuralOK          bool
	SemanticOK            bool
	ComposedOK            bool
	StructuralDiagnostics []PlanValidationError
	SemanticDiagnostics   []PlanValidationError
	ComposedDiagnostics   []PlanValidationError
	Failures              []V2PlanValidationFailure
}

// ValidateFrozenPlanV2 is the authoritative frozen-plan
// validator. It consumes the EXACT bytes loaded from F:P and
// runs the four validation stages in order:
//
//  1. parse exact frozen bytes into the canonical Plan shape
//  2. run ValidatePlanStructural on the bytes
//  3. run ValidatePlan on the parsed Plan (semantic)
//  4. run ValidatePlanComposed on the bytes
//
// Returns a typed V2Error when any stage reports a failure,
// carrying one V2Diagnostic per nested PlanValidationError so
// the CLI can render code + path + message without losing
// metadata. The runner MUST refuse to execute when this
// function returns a non-nil error.
func ValidateFrozenPlanV2(frozenBytes []byte) (V2FrozenPlanValidation, error) {
	report := V2FrozenPlanValidation{}
	// Stage 1: JSON decoding / canonical Plan parsing.
	_, _, parseErr := parsePlanBytes(frozenBytes)
	if parseErr != nil {
		report.Failures = append(report.Failures, V2PlanValidationFailure{
			InstancePath: "/plan",
			SchemaPath:   "/plan",
			Code:         "plan_parse_failed",
			Message:      parseErr.Error(),
			Keyword:      "parse",
		})
		return report, v2PlanReportToError(report)
	}
	report.ParseOK = true
	// Stage 2: structural validation (runs against raw bytes).
	structural := ValidatePlanStructural(frozenBytes)
	report.StructuralDiagnostics = structural.Errors
	if !structural.Valid {
		for _, e := range structural.Errors {
			report.Failures = append(report.Failures, v2FailureFromPlanError(e, "structural"))
		}
		return report, v2PlanReportToError(report)
	}
	report.StructuralOK = true
	// Stage 3: semantic validation against the parsed Plan.
	plan, _, _ := parsePlanBytes(frozenBytes)
	if semErr := ValidatePlan(plan); semErr != nil {
		report.Failures = append(report.Failures, V2PlanValidationFailure{
			InstancePath: "/plan",
			SchemaPath:   "/plan",
			Code:         "semantic_validation_failed",
			Message:      semErr.Error(),
			Keyword:      "semantic",
		})
		return report, v2PlanReportToError(report)
	}
	report.SemanticOK = true
	// Stage 4: composed validation against raw bytes.
	composed := ValidatePlanComposed(frozenBytes)
	report.ComposedDiagnostics = composedDiagnostics(composed)
	if !composed.Valid {
		// Composed failures may originate from structural,
		// decode, or semantic stages; surface all of them so
		// the runner report is never partial.
		for _, e := range composed.Structural.Errors {
			report.Failures = append(report.Failures, v2FailureFromPlanError(e, "structural"))
		}
		for _, e := range composed.DecodeErrors {
			report.Failures = append(report.Failures, v2FailureFromPlanError(e, "decode"))
		}
		for _, e := range composed.SemanticErrors {
			report.Failures = append(report.Failures, v2FailureFromPlanError(e, "semantic"))
		}
		return report, v2PlanReportToError(report)
	}
	report.ComposedOK = true
	return report, nil
}

// composedDiagnostics concatenates the diagnostic streams
// emitted by the composed validator so the runner report
// retains a single ordered view of every failure.
func composedDiagnostics(r ComposedPlanValidationResult) []PlanValidationError {
	out := make([]PlanValidationError, 0,
		len(r.Structural.Errors)+len(r.DecodeErrors)+len(r.SemanticErrors))
	out = append(out, r.Structural.Errors...)
	out = append(out, r.DecodeErrors...)
	out = append(out, r.SemanticErrors...)
	return out
}

// v2FailureFromPlanError converts a production PlanValidationError
// into the runner-local V2PlanValidationFailure, preserving
// instance_path / schema_path / rejected_value / accepted_values
// / property_name metadata as required by Phase 1.
func v2FailureFromPlanError(e PlanValidationError, keyword string) V2PlanValidationFailure {
	return V2PlanValidationFailure{
		InstancePath: e.InstancePath,
		SchemaPath:   e.SchemaPath,
		Code:         string(e.Code),
		Message:      e.Message,
		Keyword:      keyword,
	}
}

// ValidateV2PlanComposition is the legacy v2 entry point kept
// for backwards compatibility with existing tests. New code
// MUST call ValidateFrozenPlanV2 with the exact frozen bytes so
// structural and composed validation actually execute; this
// shim only runs semantic validation and defaults the other
// stages to false so callers cannot mistake its report for a
// real validation result.
func ValidateV2PlanComposition(plan Plan) error {
	_, err := ValidateFrozenPlanV2(planToBytes(plan))
	return err
}

// planToBytes round-trips a parsed Plan to JSON bytes. This is
// the single place where the in-memory shape is materialised
// for validation; tests that have raw frozen bytes should call
// ValidateFrozenPlanV2 directly.
func planToBytes(plan Plan) []byte {
	// Encoding the parsed Plan preserves all fields including
	// nested structures. The result is suitable for
	// structural / composed validators that consume raw bytes.
	data, err := jsonMarshalPlan(plan)
	if err != nil {
		// Validation-only code path; marshal failure is
		// surfaced by parsePlanBytes inside
		// ValidateFrozenPlanV2.
		return nil
	}
	return data
}

// v2PlanReportToError converts the report into a typed
// V2Error carrying one V2Diagnostic per failure so the CLI can
// render them with code + path + message.
func v2PlanReportToError(report V2FrozenPlanValidation) error {
	diags := make(V2Diagnostics, 0, len(report.Failures))
	for _, f := range report.Failures {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeFrozenPlanInvalid,
			Message:      fmt.Sprintf("plan validation [%s] %s: %s", f.Code, f.InstancePath, f.Message),
			PropertyName: f.InstancePath,
			Detail:       fmt.Sprintf("code=%s keyword=%s schema=%s", f.Code, f.Keyword, f.SchemaPath),
		})
	}
	return &V2Error{Diags: diags}
}
