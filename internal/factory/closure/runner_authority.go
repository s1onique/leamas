// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"fmt"
)

// RunnerAuthorityError represents errors in runner authority validation.
type RunnerAuthorityError struct {
	Field   string
	Message string
	Cause   error
}

// runnerAuthorityFieldPaths maps plan-declaration fields to their JSON pointer paths.
// Runtime-only identities (vcs.revision, vcs.modified, binary_sha256, target.subject, target.tree)
// are NOT in this map - they use empty InstancePath with PropertyName set.
var runnerAuthorityFieldPaths = map[string]string{
	"mode":                "/runner_authority/mode",
	"tool":                "/runner_authority/tool",
	"tool.revision":       "/runner_authority/tool/revision",
	"tool.tree_oid":       "/runner_authority/tool/tree_oid",
	"tool.binary_sha256":  "/runner_authority/tool/binary_sha256",
	"tool.version":        "/runner_authority/tool/version",
	"tool.tag_name":       "/runner_authority/tool/tag_name",
	"tool.tag_object_oid": "/runner_authority/tool/tag_object_oid",
}

// runnerAuthorityRuntimeIdentities are runtime-only and use PropertyName instead of InstancePath.
var runnerAuthorityRuntimeIdentities = map[string]bool{
	"vcs.revision":   true,
	"vcs.modified":   true,
	"binary_sha256":  true,
	"target.subject": true,
	"target.tree":    true,
}

// runnerAuthorityDiagnosticIdentity maps a field to its diagnostic representation.
// Plan fields: returns (JSONPointer, false)
// Runtime identities: returns ("", true)
// Unknown: returns ("", false) - caller must create deterministic invariant diagnostic.
func runnerAuthorityDiagnosticIdentity(field string) (instancePath string, isRuntimeIdentity bool) {
	if path, ok := runnerAuthorityFieldPaths[field]; ok {
		return path, false
	}
	if runnerAuthorityRuntimeIdentities[field] {
		return "", true
	}
	return "", false
}

// PlanDiagnostics implements planDiagnosticSource.
// For plan-declaration fields: returns diagnostic with JSON Pointer InstancePath.
// For runtime identities: returns diagnostic with empty InstancePath and PropertyName set.
// For unknown fields: returns one deterministic internal-invariant diagnostic.
func (e *RunnerAuthorityError) PlanDiagnostics() []PlanValidationError {
	instancePath, isRuntime := runnerAuthorityDiagnosticIdentity(e.Field)

	diag := PlanValidationError{
		InstancePath: instancePath,
		SchemaPath:   instancePath,
		Code:         PlanCodeSemanticConstraintFailed,
		Keyword:      KeywordType,
		Message:      e.Error(),
	}

	if isRuntime {
		diag.InstancePath = ""
		diag.PropertyName = e.Field
	} else if instancePath == "" {
		// Unknown field - use deterministic invariant diagnostic
		diag.InstancePath = ""
		diag.PropertyName = e.Field
	}

	return []PlanValidationError{clonePlanValidationError(diag)}
}

func (e *RunnerAuthorityError) Error() string {
	return fmt.Sprintf("runner_authority.%s: %s", e.Field, e.Message)
}

func (e *RunnerAuthorityError) Unwrap() error {
	return e.Cause
}

type ResolvedRunnerAuthority struct {
	Mode                RunnerAuthorityMode
	ExecutablePath      string
	ExecutableSHA256    string
	VCSRevision         string
	VCSModified         bool
	PinnedToolRevision  string
	PinnedToolTree      string
	PinnedBinarySHA256  string
	TargetSubjectCommit string
	TargetSubjectTree   string
}

// ValidateRunnerAuthority validates the runner_authority block in a plan.
func ValidateRunnerAuthority(authority *RunnerAuthority) error {
	if authority == nil {
		return nil
	}

	switch authority.Mode {
	case RunnerAuthoritySubjectExact:
		if authority.Tool != nil {
			if authority.Tool.Revision != "" || authority.Tool.BinarySHA256 != "" {
				return &RunnerAuthorityError{
					Field:   "tool",
					Message: "tool block not allowed for subject_exact mode",
				}
			}
		}
	case RunnerAuthorityToolReleaseExact:
		if authority.Tool == nil {
			return &RunnerAuthorityError{
				Field:   "tool",
				Message: "tool block is required for tool_release_exact mode",
			}
		}
		if err := validateToolBlock(authority.Tool); err != nil {
			return err
		}
	default:
		return &RunnerAuthorityError{
			Field:   "mode",
			Message: fmt.Sprintf("unknown runner authority mode %q", authority.Mode),
		}
	}

	return nil
}

func validateToolBlock(tool *ToolAuthority) error {
	if tool == nil {
		return &RunnerAuthorityError{
			Field:   "tool",
			Message: "tool block is required for tool_release_exact mode",
		}
	}

	if tool.Revision == "" {
		return &RunnerAuthorityError{
			Field:   "tool.revision",
			Message: "revision is required",
		}
	}
	if len(tool.Revision) != 40 {
		return &RunnerAuthorityError{
			Field:   "tool.revision",
			Message: fmt.Sprintf("revision must be 40 characters, got %d", len(tool.Revision)),
		}
	}
	if !isValidHex40(tool.Revision) {
		return &RunnerAuthorityError{
			Field:   "tool.revision",
			Message: "revision must be lowercase hexadecimal",
		}
	}

	if tool.BinarySHA256 == "" {
		return &RunnerAuthorityError{
			Field:   "tool.binary_sha256",
			Message: "binary_sha256 is required",
		}
	}
	if len(tool.BinarySHA256) != 64 {
		return &RunnerAuthorityError{
			Field:   "tool.binary_sha256",
			Message: fmt.Sprintf("binary_sha256 must be 64 characters, got %d", len(tool.BinarySHA256)),
		}
	}
	if !isValidHex64(tool.BinarySHA256) {
		return &RunnerAuthorityError{
			Field:   "tool.binary_sha256",
			Message: "binary_sha256 must be lowercase hexadecimal",
		}
	}

	if tool.TreeOID != "" {
		if len(tool.TreeOID) != 40 && len(tool.TreeOID) != 64 {
			return &RunnerAuthorityError{
				Field:   "tool.tree_oid",
				Message: "tree_oid must be 40 or 64 characters",
			}
		}
		if !isValidOID(tool.TreeOID) {
			return &RunnerAuthorityError{
				Field:   "tool.tree_oid",
				Message: "tree_oid must be lowercase hexadecimal",
			}
		}
	}

	if tool.TagObjectOID != "" {
		if len(tool.TagObjectOID) != 40 && len(tool.TagObjectOID) != 64 {
			return &RunnerAuthorityError{
				Field:   "tool.tag_object_oid",
				Message: "tag_object_oid must be 40 or 64 characters",
			}
		}
		if !isValidOID(tool.TagObjectOID) {
			return &RunnerAuthorityError{
				Field:   "tool.tag_object_oid",
				Message: "tag_object_oid must be lowercase hexadecimal",
			}
		}
	}

	return nil
}

func isValidHex40(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func isValidHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func isValidOID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
