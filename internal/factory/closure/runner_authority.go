// SPDX-License-Identifier: Apache-2.0

// Package closure - runner_authority.go is the B2-R7
// closure-side adapter for /runner_authority validation.
//
// B2-R7 single-authority rule: the wire-contract rules for
// the runner_authority block live in the plancontract leaf
// (plancontract.ValidateRunnerAuthorityBytes). The closure
// package's ValidateRunnerAuthority is preserved as a thin
// adapter so existing tests (which assert on the legacy
// RunnerAuthorityError diagnostic contract) continue to
// be meaningful. The adapter serialises the typed
// RunnerAuthority to JSON, calls the canonical leaf,
// and adapts any DecodeError back to the legacy typed-
// error contract.
//
// The function MUST NOT re-implement any wire-contract
// rule. The legacy runnerAuthorityFieldPaths map and the
// runnerAuthorityRuntimeIdentities set are kept because
// they drive the diagnostic surface (where to render the
// failure); they do NOT contain any semantic logic.
package closure

import (
	"encoding/json"
	"fmt"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
)

// RunnerAuthorityError represents errors in runner
// authority validation. B2-R7 preserves the type for
// backward compatibility; the adapter populates it from
// the canonical plancontract.DecodeError.
type RunnerAuthorityError struct {
	Field   string
	Message string
	Cause   error
}

// runnerAuthorityFieldPaths maps plan-declaration fields
// to their JSON pointer paths. Runtime-only identities
// (vcs.revision, vcs.modified, binary_sha256, target.subject,
// target.tree) are NOT in this map - they use empty
// InstancePath with PropertyName set.
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

// runnerAuthorityRuntimeIdentities are runtime-only and
// use PropertyName instead of InstancePath.
var runnerAuthorityRuntimeIdentities = map[string]bool{
	"vcs.revision":   true,
	"vcs.modified":   true,
	"binary_sha256":  true,
	"target.subject": true,
	"target.tree":    true,
}

// runnerAuthorityDiagnosticIdentity maps a field to its
// diagnostic representation. Plan fields: returns
// (JSONPointer, false). Runtime identities: returns
// ("", true). Unknown: returns ("", false).
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
// B2-R7: the closure runner still produces this shape so
// the existing diagnostics pipeline keeps working. The
// adapter populates it from the canonical plancontract
// DecodeError.
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

// ResolvedRunnerAuthority is the typed result the runtime
// builds from a validated RunnerAuthority. B2-R7 leaves
// the shape unchanged; the adapter merely guarantees that
// every field derives from a leaf-validated plan.
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

// ValidateRunnerAuthority is the B2-R7 closure-side
// adapter for the runner_authority block. It serialises
// the typed authority to JSON, asks the canonical
// plancontract leaf to validate the synthesised root, and
// adapts any DecodeError back to the legacy typed-error
// contract.
//
// The function MUST NOT re-implement any wire-contract
// rule. The plancontract leaf owns every rule. The
// adapter only translates the leaf's typed diagnostic
// into the closure package's RunnerAuthorityError shape.
func ValidateRunnerAuthority(authority *RunnerAuthority) error {
	if authority == nil {
		return nil
	}
	// Synthesise a minimal JSON root that contains only
	// the runner_authority subtree. The leaf validates
	// the subtree in isolation so the adapter does not
	// have to construct a full Plan.
	wire := map[string]any{"runner_authority": authorityToWire(authority)}
	data, err := json.Marshal(wire)
	if err != nil {
		return &RunnerAuthorityError{Field: "mode", Message: err.Error(), Cause: err}
	}
	if err := plancontract.ValidateRunnerAuthorityBytes(data); err != nil {
		return adaptRunnerAuthorityError(err)
	}
	return nil
}

// authorityToWire converts a typed RunnerAuthority into a
// JSON-friendly shape. The function does NOT re-validate;
// it merely copies fields so json.Marshal can emit them.
func authorityToWire(authority *RunnerAuthority) any {
	if authority == nil {
		return nil
	}
	mode := string(authority.Mode)
	out := map[string]any{"mode": mode}
	if authority.Tool != nil {
		tool := map[string]any{
			"revision":      authority.Tool.Revision,
			"binary_sha256": authority.Tool.BinarySHA256,
		}
		if authority.Tool.TreeOID != "" {
			tool["tree_oid"] = authority.Tool.TreeOID
		}
		if authority.Tool.Version != "" {
			tool["version"] = authority.Tool.Version
		}
		if authority.Tool.TagName != "" {
			tool["tag_name"] = authority.Tool.TagName
		}
		if authority.Tool.TagObjectOID != "" {
			tool["tag_object_oid"] = authority.Tool.TagObjectOID
		}
		out["tool"] = tool
	}
	return out
}

// adaptRunnerAuthorityError converts the canonical leaf
// DecodeError to the legacy RunnerAuthorityError. The
// translation is a pure lookup: the leaf's Field is
// already a JSON pointer, and the legacy error uses a
// short field name (e.g. "tool.revision") to drive its
// diagnostics. The map here preserves the historical
// short-name surface that existing tests depend on.
//
// The adapter MUST NOT re-evaluate the plan; it merely
// translates the leaf's typed diagnostic.
func adaptRunnerAuthorityError(err error) error {
	decodeErr, ok := err.(*plancontract.DecodeError)
	if !ok {
		return &RunnerAuthorityError{
			Field:   "mode",
			Message: err.Error(),
			Cause:   err,
		}
	}
	field := runnerAuthorityShortField(decodeErr.Field)
	return &RunnerAuthorityError{
		Field:   field,
		Message: decodeErr.Message,
		Cause:   decodeErr,
	}
}

// runnerAuthorityShortField converts the canonical leaf
// JSON-pointer Field (e.g. "/runner_authority/tool/revision")
// to the legacy short field name (e.g. "tool.revision").
// The map is a deterministic lookup; unknown fields map
// to the empty string so the legacy diagnostic surface
// renders the unknown field as a runtime identity.
func runnerAuthorityShortField(pointer string) string {
	switch pointer {
	case "/runner_authority/mode":
		return "mode"
	case "/runner_authority/tool":
		return "tool"
	case "/runner_authority/tool/revision":
		return "tool.revision"
	case "/runner_authority/tool/tree_oid":
		return "tool.tree_oid"
	case "/runner_authority/tool/binary_sha256":
		return "tool.binary_sha256"
	case "/runner_authority/tool/tag_object_oid":
		return "tool.tag_object_oid"
	case "/runner_authority":
		return "runner_authority"
	default:
		return ""
	}
}

// isValidHex40 reports whether s is exactly 40 lowercase
// hex characters. The B2-R7 closure package retains this
// helper as an alias over the canonical plancontract
// OIDPattern; production validation MUST go through the
// leaf via ValidateRunnerAuthority. The helper exists
// only so the existing test surface can keep its small
// in-package probes.
func isValidHex40(s string) bool {
	if len(s) != 40 {
		return false
	}
	return oidPattern.MatchString(s)
}

// isValidHex64 reports whether s is exactly 64 lowercase
// hex characters. See isValidHex40 for the B2-R7
// rationale.
func isValidHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	return oidPattern.MatchString(s)
}

// isValidOID reports whether s is exactly 40 or 64
// lowercase hex characters. See isValidHex40 for the
// B2-R7 rationale.
func isValidOID(s string) bool {
	return oidPattern.MatchString(s)
}
