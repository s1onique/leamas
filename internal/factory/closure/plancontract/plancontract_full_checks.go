// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full_checks.go owns
// the per-check validation for the Plan Contract v1
// "checks" array. Each entry is a JSON object with an id,
// mode, and mode-specific shape rules.
//
// Run-mode checks require argv (non-empty), working_directory
// (relative path), timeout_seconds ([1, 600]), and a
// well-formed environment map. Exclude-mode checks require
// only a reason and MUST NOT carry run-only fields.
package plancontract

import (
	"encoding/json"
	"fmt"
)

// validateChecksRequired enforces:
//   - checks MUST be present as a JSON array.
//   - checks MUST be non-empty.
//   - 0 < len(checks) <= MaxChecks.
//   - each entry satisfies validateCheckMap.
func validateChecksRequired(obj map[string]any) error {
	checksAny, ok := obj["checks"]
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "checks",
			InstancePath: "/checks",
			Message:      "checks is required",
		}
	}
	checks, ok := checksAny.([]any)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        "checks",
			InstancePath: "/checks",
			Message:      "checks is not an array",
		}
	}
	if len(checks) == 0 {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "checks",
			InstancePath: "/checks",
			Message:      "checks must be non-empty",
		}
	}
	if len(checks) > MaxChecks {
		return &DecodeError{
			Code:         "too_many_checks",
			Field:        "checks",
			InstancePath: "/checks",
			Message:      fmt.Sprintf("checks count %d exceeds %d", len(checks), MaxChecks),
		}
	}
	seenIDs := map[string]struct{}{}
	for i, rawCheck := range checks {
		if err := validateCheckMap(i, rawCheck, seenIDs); err != nil {
			return err
		}
	}
	return nil
}

// validateCheckMap enforces every per-check semantic rule.
// seenIDs is the duplicate-check-ID accumulator scoped to
// this call.
func validateCheckMap(index int, raw any, seenIDs map[string]struct{}) error {
	check, ok := raw.(map[string]any)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        fmt.Sprintf("checks[%d]", index),
			InstancePath: fmt.Sprintf("/checks/%d", index),
			Message:      fmt.Sprintf("checks[%d] is not an object", index),
		}
	}

	id, ok := check["id"].(string)
	if !ok || id == "" {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("checks[%d].id", index),
			InstancePath: fmt.Sprintf("/checks/%d/id", index),
			Message:      fmt.Sprintf("checks[%d].id is required", index),
		}
	}
	if !itemIDPattern.MatchString(id) || containsClosurePlaceholder(id) {
		return &DecodeError{
			Code:         "invalid_check_id",
			Field:        fmt.Sprintf("checks[%d].id", index),
			InstancePath: fmt.Sprintf("/checks/%d/id", index),
			Message:      fmt.Sprintf("checks[%d].id %q is invalid", index, id),
		}
	}
	if _, dup := seenIDs[id]; dup {
		return &DecodeError{
			Code:         "duplicate_check_id",
			Field:        fmt.Sprintf("checks[%d].id", index),
			InstancePath: fmt.Sprintf("/checks/%d/id", index),
			Message:      fmt.Sprintf("duplicate check id %q at checks[%d]", id, index),
		}
	}
	seenIDs[id] = struct{}{}

	mode, _ := check["mode"].(string)
	switch mode {
	case "run":
		return validateRunnableCheckMap(index, check)
	case "exclude":
		return validateExcludeCheckMap(index, check)
	default:
		return &DecodeError{
			Code:         "invalid_mode",
			Field:        fmt.Sprintf("checks[%d].mode", index),
			InstancePath: fmt.Sprintf("/checks/%d/mode", index),
			Message:      fmt.Sprintf("checks[%d].mode %q is not run|exclude", index, mode),
		}
	}
}

// validateRunnableCheckMap enforces the run-mode rules:
// argv non-empty and well-formed, working_directory is a
// relative path, timeout_seconds in [1, 600], environment
// is a well-formed map of string keys to non-empty string
// values, and reason is absent.
func validateRunnableCheckMap(index int, check map[string]any) error {
	argvAny, ok := check["argv"]
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("checks[%d].argv", index),
			InstancePath: fmt.Sprintf("/checks/%d/argv", index),
			Message:      fmt.Sprintf("checks[%d].argv is required", index),
		}
	}
	argv, ok := argvAny.([]any)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        fmt.Sprintf("checks[%d].argv", index),
			InstancePath: fmt.Sprintf("/checks/%d/argv", index),
			Message:      fmt.Sprintf("checks[%d].argv is not an array", index),
		}
	}
	if len(argv) == 0 {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("checks[%d].argv", index),
			InstancePath: fmt.Sprintf("/checks/%d/argv", index),
			Message:      fmt.Sprintf("checks[%d].argv is empty", index),
		}
	}
	if len(argv) > MaxArgvElements {
		return &DecodeError{
			Code:         "invalid_argv_count",
			Field:        fmt.Sprintf("checks[%d].argv", index),
			InstancePath: fmt.Sprintf("/checks/%d/argv", index),
			Message:      fmt.Sprintf("checks[%d].argv count %d exceeds %d", index, len(argv), MaxArgvElements),
		}
	}
	for argIndex, raw := range argv {
		arg, ok := raw.(string)
		if !ok {
			return &DecodeError{
				Code:         "invalid_type",
				Field:        fmt.Sprintf("checks[%d].argv[%d]", index, argIndex),
				InstancePath: fmt.Sprintf("/checks/%d/argv/%d", index, argIndex),
				Message:      fmt.Sprintf("checks[%d].argv[%d] is not a string", index, argIndex),
			}
		}
		if arg == "" || containsNulByte(arg) || containsClosurePlaceholder(arg) {
			return &DecodeError{
				Code:         "invalid_argv_element",
				Field:        fmt.Sprintf("checks[%d].argv[%d]", index, argIndex),
				InstancePath: fmt.Sprintf("/checks/%d/argv/%d", index, argIndex),
				Message:      fmt.Sprintf("checks[%d].argv[%d] %q is invalid", index, argIndex, arg),
			}
		}
	}

	wd, ok := check["working_directory"].(string)
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("checks[%d].working_directory", index),
			InstancePath: fmt.Sprintf("/checks/%d/working_directory", index),
			Message:      fmt.Sprintf("checks[%d].working_directory is required", index),
		}
	}
	if err := validateRepositoryRelativePath(wd, true, false); err != nil {
		return &DecodeError{
			Code:         "invalid_working_directory",
			Field:        fmt.Sprintf("checks[%d].working_directory", index),
			InstancePath: fmt.Sprintf("/checks/%d/working_directory", index),
			Message:      fmt.Sprintf("checks[%d].working_directory %q is invalid: %s", index, wd, err),
		}
	}

	rawTimeout, ok := check["timeout_seconds"]
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("checks[%d].timeout_seconds", index),
			InstancePath: fmt.Sprintf("/checks/%d/timeout_seconds", index),
			Message:      fmt.Sprintf("checks[%d].timeout_seconds is required", index),
		}
	}
	timeoutN, ok := rawTimeout.(json.Number)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        fmt.Sprintf("checks[%d].timeout_seconds", index),
			InstancePath: fmt.Sprintf("/checks/%d/timeout_seconds", index),
			Message:      fmt.Sprintf("checks[%d].timeout_seconds is not a number", index),
		}
	}
	timeout, err := timeoutN.Int64()
	if err != nil {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        fmt.Sprintf("checks[%d].timeout_seconds", index),
			InstancePath: fmt.Sprintf("/checks/%d/timeout_seconds", index),
			Message:      fmt.Sprintf("checks[%d].timeout_seconds is not an integer", index),
		}
	}
	if timeout < 1 || timeout > MaxCheckTimeoutSeconds {
		return &DecodeError{
			Code:         "invalid_timeout",
			Field:        fmt.Sprintf("checks[%d].timeout_seconds", index),
			InstancePath: fmt.Sprintf("/checks/%d/timeout_seconds", index),
			Message:      fmt.Sprintf("checks[%d].timeout_seconds %d is not in [1, %d]", index, timeout, MaxCheckTimeoutSeconds),
		}
	}

	envAny, ok := check["environment"]
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("checks[%d].environment", index),
			InstancePath: fmt.Sprintf("/checks/%d/environment", index),
			Message:      fmt.Sprintf("checks[%d].environment is required", index),
		}
	}
	env, ok := envAny.(map[string]any)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        fmt.Sprintf("checks[%d].environment", index),
			InstancePath: fmt.Sprintf("/checks/%d/environment", index),
			Message:      fmt.Sprintf("checks[%d].environment is not an object", index),
		}
	}
	if len(env) > MaxEnvironmentEntries {
		return &DecodeError{
			Code:         "too_many_env_entries",
			Field:        fmt.Sprintf("checks[%d].environment", index),
			InstancePath: fmt.Sprintf("/checks/%d/environment", index),
			Message:      fmt.Sprintf("checks[%d].environment count %d exceeds %d", index, len(env), MaxEnvironmentEntries),
		}
	}
	for name, rawValue := range env {
		if !environmentNamePattern.MatchString(name) {
			return &DecodeError{
				Code:         "invalid_env_name",
				Field:        fmt.Sprintf("checks[%d].environment.%s", index, name),
				InstancePath: fmt.Sprintf("/checks/%d/environment/%s", index, name),
				Message:      fmt.Sprintf("checks[%d].environment key %q is invalid", index, name),
			}
		}
		value, ok := rawValue.(string)
		if !ok || containsNulByte(value) {
			return &DecodeError{
				Code:         "invalid_env_value",
				Field:        fmt.Sprintf("checks[%d].environment.%s", index, name),
				InstancePath: fmt.Sprintf("/checks/%d/environment/%s", index, name),
				Message:      fmt.Sprintf("checks[%d].environment[%q] is invalid", index, name),
			}
		}
	}

	if reason, ok := check["reason"].(string); ok && reason != "" {
		return &DecodeError{
			Code:         "runnable_check_with_reason",
			Field:        fmt.Sprintf("checks[%d].reason", index),
			InstancePath: fmt.Sprintf("/checks/%d/reason", index),
			Message:      fmt.Sprintf("checks[%d] is run-mode but carries a reason", index),
		}
	}

	return nil
}

// validateExcludeCheckMap enforces the exclude-mode rules:
// reason is present and well-formed; argv, working_directory,
// timeout_seconds, and environment are absent.
func validateExcludeCheckMap(index int, check map[string]any) error {
	reason, ok := check["reason"].(string)
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        fmt.Sprintf("checks[%d].reason", index),
			InstancePath: fmt.Sprintf("/checks/%d/reason", index),
			Message:      fmt.Sprintf("checks[%d].reason is required for exclude mode", index),
		}
	}
	if !isValidExcludeReason(reason) {
		return &DecodeError{
			Code:         "invalid_reason",
			Field:        fmt.Sprintf("checks[%d].reason", index),
			InstancePath: fmt.Sprintf("/checks/%d/reason", index),
			Message:      fmt.Sprintf("checks[%d].reason %q is invalid", index, reason),
		}
	}
	if v, ok := check["argv"]; ok {
		if arr, ok := v.([]any); ok && len(arr) > 0 {
			return &DecodeError{
				Code:         "exclude_with_run_only_field",
				Field:        fmt.Sprintf("checks[%d].argv", index),
				InstancePath: fmt.Sprintf("/checks/%d/argv", index),
				Message:      fmt.Sprintf("checks[%d] is exclude-mode but carries argv", index),
			}
		}
	}
	if v, ok := check["working_directory"].(string); ok && v != "" {
		return &DecodeError{
			Code:         "exclude_with_run_only_field",
			Field:        fmt.Sprintf("checks[%d].working_directory", index),
			InstancePath: fmt.Sprintf("/checks/%d/working_directory", index),
			Message:      fmt.Sprintf("checks[%d] is exclude-mode but carries working_directory", index),
		}
	}
	if v, ok := check["timeout_seconds"]; ok {
		if n, ok := v.(json.Number); ok {
			if iv, err := n.Int64(); err == nil && iv != 0 {
				return &DecodeError{
					Code:         "exclude_with_run_only_field",
					Field:        fmt.Sprintf("checks[%d].timeout_seconds", index),
					InstancePath: fmt.Sprintf("/checks/%d/timeout_seconds", index),
					Message:      fmt.Sprintf("checks[%d] is exclude-mode but carries timeout_seconds", index),
				}
			}
		}
	}
	if v, ok := check["environment"]; ok {
		if env, ok := v.(map[string]any); ok && len(env) > 0 {
			return &DecodeError{
				Code:         "exclude_with_run_only_field",
				Field:        fmt.Sprintf("checks[%d].environment", index),
				InstancePath: fmt.Sprintf("/checks/%d/environment", index),
				Message:      fmt.Sprintf("checks[%d] is exclude-mode but carries environment", index),
			}
		}
	}
	return nil
}
