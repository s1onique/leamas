// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full_root.go owns
// the contract_version, act_id, baseline, and execution
// field validation. These are the four root-level required
// fields of the Plan Contract v1 wire format.
//
// Each helper returns a typed *DecodeError carrying the
// canonical Code, Field, and InstancePath so the closure
// package can adapt the leaf diagnostic to its legacy
// typed-error format without re-implementing the rules.
package plancontract

import (
	"encoding/json"
	"fmt"
)

// validateContractVersion enforces:
//   - contract_version MUST be present.
//   - contract_version MUST be a JSON number.
//   - contract_version MUST equal ContractVersionV1 (1).
func validateContractVersion(obj map[string]any) error {
	rawVersion, ok := obj["contract_version"]
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "contract_version",
			InstancePath: "/contract_version",
			Message:      "contract_version is required",
		}
	}
	n, ok := rawVersion.(json.Number)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        "contract_version",
			InstancePath: "/contract_version",
			Message:      "contract_version must be a JSON number",
		}
	}
	iv, err := n.Int64()
	if err != nil {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        "contract_version",
			InstancePath: "/contract_version",
			Message:      fmt.Sprintf("contract_version %q is not an integer", n.String()),
		}
	}
	if int(iv) != ContractVersionV1 {
		return &DecodeError{
			Code:         "unsupported_version",
			Field:        "contract_version",
			InstancePath: "/contract_version",
			Message:      fmt.Sprintf("contract_version %d is not supported (only %d is)", iv, ContractVersionV1),
		}
	}
	return nil
}

// validateActIDMap enforces:
//   - act_id MUST be present.
//   - act_id MUST match ActIDPattern.
//   - act_id MUST NOT contain any closure placeholder.
func validateActIDMap(obj map[string]any) error {
	actID, ok := obj["act_id"].(string)
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "act_id",
			InstancePath: "/act_id",
			Message:      "act_id is required",
		}
	}
	if !ActIDPattern.MatchString(actID) || containsClosurePlaceholder(actID) {
		return &DecodeError{
			Code:         "invalid_act_id",
			Field:        "act_id",
			InstancePath: "/act_id",
			Message:      fmt.Sprintf("act_id %q is invalid", actID),
		}
	}
	return nil
}

// validateBaselineRequired enforces:
//   - baseline MUST be present as a JSON object.
//   - baseline.commit_oid MUST be a valid 40- or 64-character hex OID.
//   - baseline.commit_oid MUST NOT be a closure placeholder.
//   - baseline.tree_oid MUST be a valid 40- or 64-character hex OID.
//   - baseline.tree_oid MUST NOT be a closure placeholder.
//
// The placeholder check runs first so the diagnostic surfaces
// a clear "placeholder identity" message even when the value
// also fails the hex-pattern check.
func validateBaselineRequired(obj map[string]any) error {
	baseline, ok := obj["baseline"].(map[string]any)
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "baseline",
			InstancePath: "/baseline",
			Message:      "baseline is required",
		}
	}
	if err := validateBaselineMap(baseline); err != nil {
		return err
	}
	return nil
}

// validateBaselineMap enforces the per-field baseline OID rules.
func validateBaselineMap(baseline map[string]any) error {
	for _, field := range []string{"commit_oid", "tree_oid"} {
		v, ok := baseline[field].(string)
		if !ok {
			return &DecodeError{
				Code:         "missing_field",
				Field:        "baseline." + field,
				InstancePath: "/baseline/" + field,
				Message:      fmt.Sprintf("baseline.%s is required", field),
			}
		}
		if containsClosurePlaceholder(v) {
			return &DecodeError{
				Code:         "baseline_oid_placeholder",
				Field:        "baseline." + field,
				InstancePath: "/baseline/" + field,
				Message:      fmt.Sprintf("baseline.%s %q contains a closure placeholder", field, v),
			}
		}
		if !OIDPattern.MatchString(v) {
			return &DecodeError{
				Code:         "invalid_baseline_oid",
				Field:        "baseline." + field,
				InstancePath: "/baseline/" + field,
				Message:      fmt.Sprintf("baseline.%s %q is not a valid 40- or 64-character hex OID", field, v),
			}
		}
	}
	return nil
}

// validateExecutionRequired enforces:
//   - execution MUST be present as a JSON object.
//   - execution.mode MUST be present and in the closed enum
//     {"serial_fail_fast"}; whitespace and "" are rejected.
func validateExecutionRequired(obj map[string]any) error {
	execution, ok := obj["execution"].(map[string]any)
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "execution",
			InstancePath: "/execution",
			Message:      "execution is required",
		}
	}
	if err := validateExecutionMap(execution); err != nil {
		return err
	}
	return nil
}

// validateExecutionMap enforces the per-field execution rules.
func validateExecutionMap(execution map[string]any) error {
	rawMode, ok := execution["mode"]
	if !ok || rawMode == nil {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "execution.mode",
			InstancePath: "/execution/mode",
			Message:      "execution.mode is required",
		}
	}
	mode, ok := rawMode.(string)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        "execution.mode",
			InstancePath: "/execution/mode",
			Message:      "execution.mode is not a string",
		}
	}
	trimmed := trimSpaceLower(mode)
	if trimmed == "" {
		return &DecodeError{
			Code:         "invalid_mode",
			Field:        "execution.mode",
			InstancePath: "/execution/mode",
			Message:      "execution.mode is empty or whitespace",
		}
	}
	if trimmed != "serial_fail_fast" {
		return &DecodeError{
			Code:         "invalid_mode",
			Field:        "execution.mode",
			InstancePath: "/execution/mode",
			Message:      fmt.Sprintf("execution.mode %q is not in the closed enum (only serial_fail_fast)", trimmed),
		}
	}
	return nil
}
