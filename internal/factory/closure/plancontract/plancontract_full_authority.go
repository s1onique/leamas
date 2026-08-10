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
//
// B2-R7 migration: the optional tool-block fields tree_oid,
// tag_name, and tag_object_oid (which the closure package's
// runner_authority.go validated separately) are now
// validated here. The leaf is the single authority; the
// closure package's ValidateRunnerAuthority is a thin
// adapter that calls this leaf and maps any DecodeError
// back to its legacy typed-error contract.
package plancontract

import "fmt"

// validateRunnerAuthorityOptional enforces:
//   - runner_authority MUST be a JSON object when present.
//   - mode MUST be "subject_exact" or "tool_release_exact".
//   - subject_exact MUST NOT carry a tool block.
//   - tool_release_exact MUST carry a tool block with
//     revision (valid OID) and binary_sha256 (64-char
//     lowercase hex); when present the optional fields
//     tree_oid and tag_object_oid (40- or 64-char lowercase
//     hex) and tag_name are validated too.
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
		if err := validateToolBlock(tool); err != nil {
			return err
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

// validateToolBlock enforces the per-field rules for the
// /runner_authority/tool object. revision and binary_sha256
// are required; tree_oid and tag_object_oid are optional but
// MUST be a 40- or 64-character lowercase hex string when
// present; tag_name and version are unconstrained strings.
//
// B2-R7 single-authority rule: every wire-contract field
// that exists in the Plan Contract v1 ToolAuthority shape
// is validated here. The closure package's old
// validateToolBlock helper is the duplicate that the B2-R7
// task deletes; this leaf function replaces it.
func validateToolBlock(tool map[string]any) error {
	revision, ok := tool["revision"].(string)
	if !ok || revision == "" {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "runner_authority.tool.revision",
			InstancePath: "/runner_authority/tool/revision",
			Message:      "runner_authority.tool.revision is required",
		}
	}
	if !OIDPattern.MatchString(revision) || containsClosurePlaceholder(revision) {
		return &DecodeError{
			Code:         "invalid_tool_revision",
			Field:        "runner_authority.tool.revision",
			InstancePath: "/runner_authority/tool/revision",
			Message:      "runner_authority.tool.revision is not a valid 40- or 64-character lowercase hex OID",
		}
	}
	// B2-R6: binary_sha256 MUST be exactly 64 lowercase hex
	// characters. Uppercase, wrong length, and non-hex
	// characters are all rejected. The previous check only
	// validated length, which accepted uppercase hex.
	sha, ok := tool["binary_sha256"].(string)
	if !ok || sha == "" {
		return &DecodeError{
			Code:         "missing_field",
			Field:        "runner_authority.tool.binary_sha256",
			InstancePath: "/runner_authority/tool/binary_sha256",
			Message:      "runner_authority.tool.binary_sha256 is required",
		}
	}
	if !lowercaseHex64Pattern.MatchString(sha) {
		return &DecodeError{
			Code:         "invalid_tool_sha256",
			Field:        "runner_authority.tool.binary_sha256",
			InstancePath: "/runner_authority/tool/binary_sha256",
			Message:      "runner_authority.tool.binary_sha256 is not a 64-character lowercase hexadecimal string",
		}
	}
	// Optional tree_oid: when present MUST be 40- or 64-char
	// lowercase hex. The OID pattern matches both lengths.
	if v, ok := tool["tree_oid"]; ok {
		s, isStr := v.(string)
		if !isStr || (s != "" && !OIDPattern.MatchString(s)) {
			return &DecodeError{
				Code:         "invalid_tool_tree_oid",
				Field:        "runner_authority.tool.tree_oid",
				InstancePath: "/runner_authority/tool/tree_oid",
				Message:      "runner_authority.tool.tree_oid must be a 40- or 64-character lowercase hexadecimal string",
			}
		}
	}
	// Optional tag_object_oid: same rule as tree_oid.
	if v, ok := tool["tag_object_oid"]; ok {
		s, isStr := v.(string)
		if !isStr || (s != "" && !OIDPattern.MatchString(s)) {
			return &DecodeError{
				Code:         "invalid_tool_tag_object_oid",
				Field:        "runner_authority.tool.tag_object_oid",
				InstancePath: "/runner_authority/tool/tag_object_oid",
				Message:      "runner_authority.tool.tag_object_oid must be a 40- or 64-character lowercase hexadecimal string",
			}
		}
	}
	// tag_name and version are unconstrained strings when
	// present; an empty string is the producer's "absent"
	// signal and the canonical ValidatedPlan records it as
	// empty so callers see exactly what the wire declared.
	//
	// B2-R7-R1 type parity: when present in the wire bytes,
	// version and tag_name MUST be JSON strings. Wrong-type
	// values (numbers, booleans, arrays, objects) MUST be
	// rejected so the leaf agrees with the typed-Plan
	// decoder about what shape is acceptable. Without this
	// check the leaf silently coerces wrong-type wire
	// values to "" while the typed decoder rejects them,
	// creating a parity hole between execution and
	// evidence.
	if v, ok := tool["version"]; ok {
		if _, isStr := v.(string); !isStr {
			return &DecodeError{
				Code:         "invalid_tool_version_type",
				Field:        "runner_authority.tool.version",
				InstancePath: "/runner_authority/tool/version",
				Message:      "runner_authority.tool.version must be a JSON string",
			}
		}
	}
	if v, ok := tool["tag_name"]; ok {
		if _, isStr := v.(string); !isStr {
			return &DecodeError{
				Code:         "invalid_tool_tag_name_type",
				Field:        "runner_authority.tool.tag_name",
				InstancePath: "/runner_authority/tool/tag_name",
				Message:      "runner_authority.tool.tag_name must be a JSON string",
			}
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

// ValidateRunnerAuthorityBytes is the public surface for
// the runner_authority subtree. It accepts the canonical
// Plan Contract v1 wire bytes of the { "runner_authority": ... }
// object and returns a typed *DecodeError on failure or
// nil on success.
//
// B2-R7 adapter surface: the closure package's
// ValidateRunnerAuthority wraps this function so the
// closure runner and the evidence package share the same
// authoritative runner_authority semantic pass. The
// closure-side adapter never re-implements a wire rule.
func ValidateRunnerAuthorityBytes(data []byte) error {
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
	return validateRunnerAuthorityOptional(obj)
}
