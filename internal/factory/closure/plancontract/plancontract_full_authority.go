// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full_authority.go owns
// the optional Plan Contract v1 /runner_authority validation
// and the small path-validation helpers shared by every
// per-section helper file.
//
// The runner_authority block selects how the runner pins
// itself. B2-R5 mirrors the closure package's two-mode enum
// (subject_exact | tool_release_exact) and the tool-block
// requirements exactly. The closure package's
// ValidateRunnerAuthority and validateToolBlock were
// deleted in B2-R5 because the wire-contract rules now live
// exclusively here.
package plancontract

import "fmt"

// validateRunnerAuthorityOptional enforces:
//   - runner_authority MUST be a JSON object when present.
//   - mode MUST be "subject_exact" or "tool_release_exact".
//   - subject_exact MUST NOT carry a tool block.
//   - tool_release_exact MUST carry a tool block with
//     revision (valid OID) and binary_sha256 (64-char hex).
//   - JSON null is treated as "no runner_authority" so the
//     typed Plan's *RunnerAuthority pointer can be nil.
func validateRunnerAuthorityOptional(obj map[string]any) error {
	rawRA, ok := obj["runner_authority"]
	if !ok || rawRA == nil {
		return nil
	}
	ra, ok := rawRA.(map[string]any)
	if !ok {
		return &DecodeError{
			Code:         "invalid_type",
			Field:        "runner_authority",
			InstancePath: "/runner_authority",
			Message:      "runner_authority is not an object",
		}
	}
	return validateRunnerAuthorityMap(ra)
}

// validateRunnerAuthorityMap enforces the runner_authority
// per-field rules.
func validateRunnerAuthorityMap(ra map[string]any) error {
	mode, ok := ra["mode"].(string)
	if !ok {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "runner_authority.mode",
			InstancePath: "/runner_authority/mode",
			Message:      "runner_authority.mode is required",
		}
	}
	switch mode {
	case "subject_exact":
		if hasToolBlock(ra) {
			return &DecodeError{
				Code:         "subject_exact_with_tool",
				Field:        "runner_authority.tool",
				InstancePath: "/runner_authority/tool",
				Message:      "runner_authority.subject_exact must not carry a tool block",
			}
		}
	case "tool_release_exact":
		tool, ok := toolBlock(ra)
		if !ok {
			return &DecodeError{
				Code:         "missing_field",
				Field:        "runner_authority.tool",
				InstancePath: "/runner_authority/tool",
				Message:      "runner_authority.tool_release_exact requires a tool block",
			}
		}
		revision, ok := tool["revision"].(string)
		if !ok || !oidPattern.MatchString(revision) || containsClosurePlaceholder(revision) {
			return &DecodeError{
				Code:         "invalid_tool_revision",
				Field:        "runner_authority.tool.revision",
				InstancePath: "/runner_authority/tool/revision",
				Message:      "runner_authority.tool.revision is not a valid OID",
			}
		}
// B2-R6: binary_sha256 MUST be exactly 64 lowercase hex
// characters. The previous check only validated length.
		sha, ok := tool["binary_sha256"].(string)
		if !ok || !lowercaseHex64Pattern.MatchString(sha) {
			return &DecodeError{
				Code:         "invalid_tool_sha256",
				Field:        "runner_authority.tool.binary_sha256",
				InstancePath: "/runner_authority/tool/binary_sha256",
				Message:      "runner_authority.tool.binary_sha256 is not a 64-character lowercase hexadecimal string",
			}
		}
	default:
		return &DecodeError{
			Code:         "invalid_runner_authority_mode",
			Field:        "runner_authority.mode",
			InstancePath: "/runner_authority/mode",
			Message:      fmt.Sprintf("runner_authority.mode %q is not subject_exact|tool_release_exact", mode),
		}
	}
	return nil
}

// hasToolBlock returns true iff the runner_authority block
// carries a tool field whose value is a JSON object (i.e.
// NOT null). JSON null is treated as "no tool" because the
// typed Plan's *ToolAuthority is a pointer and Go's JSON
// decoder reads null as a nil pointer.
func hasToolBlock(ra map[string]any) bool {
	v, ok := ra["tool"]
	if !ok || v == nil {
		return false
	}
	_, isObj := v.(map[string]any)
	return isObj
}

// toolBlock returns the tool block as a map. When the
// field is absent or null the second return is false so
// the caller can emit a missing_field diagnostic.
func toolBlock(ra map[string]any) (map[string]any, bool) {
	v, ok := ra["tool"]
	if !ok || v == nil {
		return nil, false
	}
	tool, isObj := v.(map[string]any)
	return tool, isObj
}
