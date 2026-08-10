// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full.go is the B2-R5
// single complete Plan Contract v1 semantic authority.
//
// B2-R4 closed the duplicate-field scanner and introduced
// ValidateFull as the canonical semantic pass for the
// closure runner and the evidence package. The leaf still
// mirrored the closure package's typed validators, so the
// rules existed in two places.
//
// B2-R5 makes plancontract the actual single semantic
// authority. Every wire-contract rule lives here. The
// closure package's ValidatePlan calls this leaf and adapts
// the DecodeError to its legacy typed PlanSemanticError;
// the evidence package calls this leaf directly.
//
// B2-R7 introduces the canonical ValidatedPlan projection
// (see plancontract_validated.go). DecodeAndValidateFull
// is the single canonical entry point that returns the
// projected ValidatedPlan; ValidateFull is a thin wrapper
// for callers that only need an error result.
//
// This file owns the convenience wrappers and dispatches
// to the per-section helpers in plancontract_full_*.go.
// Each helper file stays under the LLM-friendly 400-line
// threshold.
package plancontract

import "encoding/json"

// ValidateFull enforces the full Plan Contract v1 semantic
// invariants on the supplied bytes. It is the canonical
// public entry point for callers that only need an error
// result. The function composes DecodeBytes with the full
// semantic pass and returns nil on success.
//
// B2-R7 single-authority rule: this function is the only
// entry point both the closure runner and the evidence
// package use for the F:P contract when an error is the
// only signal needed. Callers that also need the
// canonical projected representation should use
// DecodeAndValidateFull instead.
func ValidateFull(data []byte) error {
	_, err := DecodeAndValidateFull(data)
	return err
}

// ValidateFullAndProject is the canonical evidence-package
// entry point. It applies the full semantic pass and then
// projects the parsed document into the minimal
// DecodeResult (contract_version + ordered checks) that the
// evidence package's PlanCheckSpec list derives from. The
// combined call avoids a redundant second decode pass.
//
// B2-R7: this function is preserved for callers that want
// only the minimal projection. New code that needs the
// full canonical projection should call
// DecodeAndValidateFull and read the ValidatedPlan
// directly.
func ValidateFullAndProject(data []byte) (DecodeResult, error) {
	root, err := DecodeBytes(data)
	if err != nil {
		return DecodeResult{}, err
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return DecodeResult{}, &DecodeError{
			Code:    "invalid_json",
			Message: "root is not a JSON object",
		}
	}
	if err := ValidateFullMap(obj); err != nil {
		return DecodeResult{}, err
	}
	return projectToResult(obj)
}

// ValidatePolicyBytes validates the bytes of a single
// /policy sub-object. B2-R5 exposes this narrow entry
// point so the closure package's typed validators can
// delegate to the leaf without round-tripping the entire
// plan.
//
// The function parses the bytes into a single JSON object
// and calls validatePolicyMap on it. Any *DecodeError
// returned carries the canonical Field and InstancePath so
// the closure package's adapter can map it back to the
// legacy typed error contract.
func ValidatePolicyBytes(data []byte) error {
	var policy map[string]any
	if err := json.Unmarshal(data, &policy); err != nil {
		return &DecodeError{
			Code:    "invalid_json",
			Message: err.Error(),
		}
	}
	return validatePolicyMap(policy)
}

// ValidateFullMap enforces the full Plan Contract v1
// semantic invariants on the supplied parsed JSON root.
// Callers that already have a parsed root (for example,
// the closure runner after its bounded syntactic decode)
// may invoke this directly without re-decoding the bytes.
//
// The function returns a typed *DecodeError so callers
// can distinguish failure categories by Code.
func ValidateFullMap(obj map[string]any) error {
	if err := validateContractVersion(obj); err != nil {
		return err
	}
	if err := validateActIDMap(obj); err != nil {
		return err
	}
	if err := validateBaselineRequired(obj); err != nil {
		return err
	}
	if err := validateExecutionRequired(obj); err != nil {
		return err
	}
	if err := validateChecksRequired(obj); err != nil {
		return err
	}
	if err := validateArtifactsOptional(obj); err != nil {
		return err
	}
	if err := validatePolicyRequired(obj); err != nil {
		return err
	}
	if err := validateRunnerAuthorityOptional(obj); err != nil {
		return err
	}
	return nil
}
