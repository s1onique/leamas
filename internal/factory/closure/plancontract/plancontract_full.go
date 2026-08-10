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
// This file owns only the entry points (ValidateFull,
// ValidateFullAndProject, ValidateFullMap,
// DecodeAndValidateFull) and dispatches to the per-section
// helpers in plancontract_full_*.go. Each helper file stays
// under the LLM-friendly 400-line threshold.
package plancontract

import "encoding/json"

// DecodeAndValidateFull is the B2-R5 single complete
// authority for Plan Contract v1 decoding and semantic
// validation. It composes the bounded syntactic decoder
// (DecodeBytes) with the full semantic pass (ValidateFull).
//
// The function is the only entry point both the closure
// runner and the evidence package use for the F:P contract.
// No closure-package or evidence-package code path may
// bypass it; doing so would re-introduce the second
// authority B2-R4 left behind.
func DecodeAndValidateFull(data []byte) error {
	root, err := DecodeBytes(data)
	if err != nil {
		return err
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return &DecodeError{
			Code:    "invalid_json",
			Message: "root is not a JSON object",
		}
	}
	return ValidateFullMap(obj)
}

// ValidateFull enforces the full Plan Contract v1 semantic
// invariants on the supplied bytes. It is a convenience
// wrapper around DecodeAndValidateFull.
func ValidateFull(data []byte) error {
	return DecodeAndValidateFull(data)
}

// ValidateFullAndProject is the canonical evidence-package
// entry point. It applies the full semantic pass and then
// projects the parsed document into the minimal
// DecodeResult (contract_version + ordered checks) that the
// evidence package's PlanCheckSpec list derives from. The
// combined call avoids a redundant second decode pass.
//
// The closure runner calls ValidateFull on the frozen
// bytes; the evidence package calls ValidateFullAndProject
// so it can both reject invalid plans AND extract the
// canonical check set in a single decoder pass.
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
// Callers that already have a parsed root (for example, the
// closure runner after its bounded syntactic decode) may
// invoke this directly without re-decoding the bytes.
//
// The function returns a typed *DecodeError so callers can
// distinguish failure categories by Code.
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
