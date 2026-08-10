// SPDX-License-Identifier: Apache-2.0

// Package plancontract - plancontract_full.go is the B2-R4
// single complete Plan Contract v1 semantic authority.
//
// B2-R3 introduced a minimal Validate function in plancontract
// that owned a small set of contract-version and check-shape
// rules. The closure package continued to own the typed
// Plan-level semantic rules in closure.ValidatePlan, so the
// closure runner and the evidence package still had two
// authorities for the same wire contract.
//
// B2-R4 closes that split. This file owns the COMPLETE
// semantic authority for whether a Plan Contract v1 document
// is a valid executable Closure Plan. Both the closure
// runner and the evidence package call ValidateFull (and
// DecodeAndValidateFull) so they cannot disagree.
//
// The function operates on the parsed JSON tree (a
// map[string]any produced by DecodeBytes) so the leaf does
// not need to import the closure package's typed Plan
// struct. Every rule that closure.ValidatePlan enforced on
// the typed Plan is reproduced here on the wire shape.
package plancontract

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ----------------------------------------------------------------------------
// Bound constants (canonical limits)
// ----------------------------------------------------------------------------
//
// The Plan Contract v1 wire format has fixed bounds that the
// closure runner and the evidence package MUST agree on.
// Duplicating them here keeps the leaf package independent
// of the closure package. Any drift between these values and
// the closure package's mirror constants is a contract bug
// and will be caught by the execution/evidence parity
// matrix in plancontract_parity_b2r4_test.go.

// MaxChecks / MaxArtifacts mirror the closure package's
// closure.MaxChecks / closure.MaxArtifacts limits. The
// values are duplicated here to keep the leaf independent;
// any drift is a contract bug.
const (
	// MaxChecks is the maximum number of checks a Plan
	// Contract v1 document may declare.
	MaxChecks = 10_000

	// MaxArtifacts is the maximum number of artifacts a
	// Plan Contract v1 document may declare.
	MaxArtifacts = 10_000

	// MaxArgvElements is the maximum number of argv
	// elements a single check may declare.
	MaxArgvElements = 16

	// MaxEnvironmentEntries is the maximum number of
	// environment entries a single check may declare.
	MaxEnvironmentEntries = 32

	// MaxCheckTimeoutSeconds is the inclusive upper bound
	// for the per-check timeout_seconds field.
	MaxCheckTimeoutSeconds = 600
)

// ----------------------------------------------------------------------------
// Patterns (canonical regex set)
// ----------------------------------------------------------------------------
//
// actIDPattern matches the canonical ACT identifier format.
// itemIDPattern matches the canonical lowercase identifier
// format used by check and artifact IDs. oidPattern matches
// a 40- or 64-character lowercase hex Git OID.
// environmentNamePattern matches a POSIX environment
// variable name. These are intentionally duplicated from
// the closure package; any drift is a contract bug.

var (
	actIDPattern           = regexp.MustCompile(`^ACT-[A-Z0-9][A-Z0-9-]{2,199}$`)
	itemIDPattern          = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	oidPattern             = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
	environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// ----------------------------------------------------------------------------
// Placeholder detection
// ----------------------------------------------------------------------------
//
// exactClosurePlaceholders mirrors the closure package's
// exact placeholder set so the leaf and the closure runner
// observe the same rejection set for whitespace-trimmed,
// uppercase-normalised values. The set is duplicated here
// to keep the leaf independent; any drift is caught by the
// execution/evidence parity matrix.
var exactClosurePlaceholders = map[string]struct{}{
	"TBD":            {},
	"TODO":           {},
	"UNKNOWN":        {},
	"RUNNING":        {},
	"TO BE RECORDED": {},
}

// embeddedClosurePlaceholders mirrors the closure
// package's embedded placeholder markers. Any drift is
// caught by the parity matrix.
var embeddedClosurePlaceholders = []string{
	"(SEE GIT REV-PARSE)",
	"<COMMIT>",
	"<TREE>",
	"<HASH>",
}

// containsClosurePlaceholder mirrors the closure package's
// placeholder detection so the leaf and the closure runner
// reject the same set of values. The function is case-
// insensitive and whitespace-trimmed.
func containsClosurePlaceholder(value string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if _, found := exactClosurePlaceholders[normalized]; found {
		return true
	}
	for _, marker := range embeddedClosurePlaceholders {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

// ----------------------------------------------------------------------------
// DecodeAndValidateFull
// ----------------------------------------------------------------------------

// DecodeAndValidateFull is the B2-R4 single complete
// authority for Plan Contract v1 decoding and semantic
// validation. It composes the bounded syntactic decoder
// (DecodeBytes) with the full semantic pass (ValidateFull).
//
// The function is the only entry point both the closure
// runner and the evidence package use for the F:P contract.
// No closure-package or evidence-package code path may
// bypass it; doing so would re-introduce the second
// authority B2-R3 left behind.
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

// ValidateFullMap enforces the full Plan Contract v1
// semantic invariants on the supplied parsed JSON root.
// Callers that already have a parsed root (for example, the
// closure runner after its bounded syntactic decode) may
// invoke this directly without re-decoding the bytes.
//
// The function returns a typed *DecodeError so callers can
// distinguish failure categories by Code.
func ValidateFullMap(obj map[string]any) error {
	// contract_version MUST be present, must be a JSON
	// number, and MUST equal ContractVersionV1 (1).
	rawVersion, ok := obj["contract_version"]
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: "contract_version is required",
		}
	}
	n, ok := rawVersion.(json.Number)
	if !ok {
		return &DecodeError{
			Code:    "invalid_type",
			Message: "contract_version must be a JSON number",
		}
	}
	iv, err := n.Int64()
	if err != nil {
		return &DecodeError{
			Code:    "invalid_type",
			Message: fmt.Sprintf("contract_version %q is not an integer", n.String()),
		}
	}
	if int(iv) != ContractVersionV1 {
		return &DecodeError{
			Code:    "unsupported_version",
			Message: fmt.Sprintf("contract_version %d is not supported (only %d is)", iv, ContractVersionV1),
		}
	}

	// act_id MUST match the canonical pattern and MUST
	// not contain any closure placeholder substring.
	actID, ok := obj["act_id"].(string)
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: "act_id is required",
		}
	}
	if !actIDPattern.MatchString(actID) || containsClosurePlaceholder(actID) {
		return &DecodeError{
			Code:    "invalid_act_id",
			Message: fmt.Sprintf("act_id %q is invalid", actID),
		}
	}

	// baseline.commit_oid and baseline.tree_oid MUST be
	// valid 40- or 64-character lowercase hex OIDs and
	// MUST not be placeholders.
	baseline, ok := obj["baseline"].(map[string]any)
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: "baseline is required",
		}
	}
	if err := validateBaselineMap(baseline); err != nil {
		return err
	}

	// execution.mode MUST be present and in the closed
	// enum { serial_fail_fast } (whitespace and "" are
	// rejected). The PlanExecution field is a pointer so
	// absent vs. empty is distinguishable.
	execution, ok := obj["execution"].(map[string]any)
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: "execution is required",
		}
	}
	if err := validateExecutionMap(execution); err != nil {
		return err
	}

	// checks MUST be a non-empty array and MUST NOT
	// exceed MaxChecks.
	checksAny, ok := obj["checks"]
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: "checks is required",
		}
	}
	checks, ok := checksAny.([]any)
	if !ok {
		return &DecodeError{
			Code:    "invalid_type",
			Message: "checks is not an array",
		}
	}
	if len(checks) == 0 {
		return &DecodeError{
			Code:    "missing_field",
			Message: "checks must be non-empty",
		}
	}
	if len(checks) > MaxChecks {
		return &DecodeError{
			Code:    "too_many_checks",
			Message: fmt.Sprintf("checks count %d exceeds %d", len(checks), MaxChecks),
		}
	}
	seenIDs := map[string]struct{}{}
	for i, rawCheck := range checks {
		if err := validateCheckMap(i, rawCheck, seenIDs); err != nil {
			return err
		}
	}

// artifacts (optional) MUST NOT exceed MaxArtifacts
// and each entry MUST be well-formed. JSON null is
// treated as "no artifacts" so the typed Plan's nil
// slice round-trips cleanly through the leaf.
	if rawArtifacts, ok := obj["artifacts"]; ok && rawArtifacts != nil {
		artifacts, ok := rawArtifacts.([]any)
		if !ok {
			return &DecodeError{
				Code:    "invalid_type",
				Message: "artifacts is not an array",
			}
		}
		if len(artifacts) > MaxArtifacts {
			return &DecodeError{
				Code:    "too_many_artifacts",
				Message: fmt.Sprintf("artifacts count %d exceeds %d", len(artifacts), MaxArtifacts),
			}
		}
		seenArtifactIDs := map[string]struct{}{}
		for i, rawArtifact := range artifacts {
			if err := validateArtifactMap(i, rawArtifact, seenArtifactIDs); err != nil {
				return err
			}
		}
	}

	// policy (optional): each declared field MUST be a
	// boolean pointer in the wire shape (so the strict
	// decoder can distinguish present from absent).
	if rawPolicy, ok := obj["policy"]; ok {
		policy, ok := rawPolicy.(map[string]any)
		if !ok {
			return &DecodeError{
				Code:    "invalid_type",
				Message: "policy is not an object",
			}
		}
		if err := validatePolicyMap(policy); err != nil {
			return err
		}
	}

	// runner_authority (optional): well-formed when
	// present. subject_exact forbids a tool block;
	// tool_release_exact requires one.
	if rawRA, ok := obj["runner_authority"]; ok {
		ra, ok := rawRA.(map[string]any)
		if !ok {
			return &DecodeError{
				Code:    "invalid_type",
				Message: "runner_authority is not an object",
			}
		}
		if err := validateRunnerAuthorityMap(ra); err != nil {
			return err
		}
	}

	return nil
}

// ----------------------------------------------------------------------------
// Per-section helpers
// ----------------------------------------------------------------------------

// validateBaselineMap enforces the baseline OID rules.
// The placeholder check runs first so the diagnostic
// surfaces a clear "placeholder identity" message even when
// the value also fails the hex-pattern check.
func validateBaselineMap(baseline map[string]any) error {
	for _, field := range []string{"commit_oid", "tree_oid"} {
		v, ok := baseline[field].(string)
		if !ok {
			return &DecodeError{
				Code:    "missing_field",
				Message: fmt.Sprintf("baseline.%s is required", field),
			}
		}
		if containsClosurePlaceholder(v) {
			return &DecodeError{
				Code:    "baseline_oid_placeholder",
				Message: fmt.Sprintf("baseline.%s %q contains a closure placeholder", field, v),
			}
		}
		if !oidPattern.MatchString(v) {
			return &DecodeError{
				Code:    "invalid_baseline_oid",
				Message: fmt.Sprintf("baseline.%s %q is not a valid 40- or 64-character hex OID", field, v),
			}
		}
	}
	return nil
}

// validateExecutionMap enforces the execution.mode rules.
// The closure package recognises a single closed enum
// (serial_fail_fast); whitespace and empty values are
// rejected so the typed Plan's pointer-distinguished
// absent vs. empty categories remain meaningful.
func validateExecutionMap(execution map[string]any) error {
	rawMode, ok := execution["mode"]
	if !ok || rawMode == nil {
		return &DecodeError{
			Code:    "missing_field",
			Message: "execution.mode is required",
		}
	}
	mode, ok := rawMode.(string)
	if !ok {
		return &DecodeError{
			Code:    "invalid_type",
			Message: "execution.mode is not a string",
		}
	}
	trimmed := strings.TrimSpace(mode)
	if trimmed == "" {
		return &DecodeError{
			Code:    "invalid_mode",
			Message: "execution.mode is empty or whitespace",
		}
	}
	if trimmed != "serial_fail_fast" {
		return &DecodeError{
			Code:    "invalid_mode",
			Message: fmt.Sprintf("execution.mode %q is not in the closed enum (only serial_fail_fast)", trimmed),
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
			Code:    "invalid_type",
			Message: fmt.Sprintf("checks[%d] is not an object", index),
		}
	}

	id, ok := check["id"].(string)
	if !ok || id == "" {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("checks[%d].id is required", index),
		}
	}
	if !itemIDPattern.MatchString(id) || containsClosurePlaceholder(id) {
		return &DecodeError{
			Code:    "invalid_check_id",
			Message: fmt.Sprintf("checks[%d].id %q is invalid", index, id),
		}
	}
// The Message wording mirrors the closure package's typed
// validator so existing tests that substring-match on the
// human-readable error text continue to pass. The Code is
// the canonical failure token for switch-based callers.
	if _, dup := seenIDs[id]; dup {
		return &DecodeError{
			Code:    "duplicate_check_id",
			Message: fmt.Sprintf("duplicate check id %q at checks[%d]", id, index),
		}
	}
	seenIDs[id] = struct{}{}

	mode, _ := check["mode"].(string)
	switch mode {
	case "run":
		if err := validateRunnableCheckMap(index, check); err != nil {
			return err
		}
	case "exclude":
		if err := validateExcludeCheckMap(index, check); err != nil {
			return err
		}
	default:
		return &DecodeError{
			Code:    "invalid_mode",
			Message: fmt.Sprintf("checks[%d].mode %q is not run|exclude", index, mode),
		}
	}
	return nil
}

// validateRunnableCheckMap enforces the run-mode check
// rules: argv non-empty, working_directory present,
// timeout_seconds in [1, 600], environment present and
// well-formed, no reason.
func validateRunnableCheckMap(index int, check map[string]any) error {
	argvAny, ok := check["argv"]
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("checks[%d].argv is required", index),
		}
	}
	argv, ok := argvAny.([]any)
	if !ok {
		return &DecodeError{
			Code:    "invalid_type",
			Message: fmt.Sprintf("checks[%d].argv is not an array", index),
		}
	}
	if len(argv) == 0 {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("checks[%d].argv is empty", index),
		}
	}
	if len(argv) > MaxArgvElements {
		return &DecodeError{
			Code:    "invalid_argv_count",
			Message: fmt.Sprintf("checks[%d].argv count %d exceeds %d", index, len(argv), MaxArgvElements),
		}
	}
	for argIndex, raw := range argv {
		arg, ok := raw.(string)
		if !ok {
			return &DecodeError{
				Code:    "invalid_type",
				Message: fmt.Sprintf("checks[%d].argv[%d] is not a string", index, argIndex),
			}
		}
		if arg == "" || strings.ContainsRune(arg, 0) || containsClosurePlaceholder(arg) {
			return &DecodeError{
				Code:    "invalid_argv_element",
				Message: fmt.Sprintf("checks[%d].argv[%d] %q is invalid", index, argIndex, arg),
			}
		}
	}

	wd, ok := check["working_directory"].(string)
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("checks[%d].working_directory is required", index),
		}
	}
	if err := validateRepositoryRelativePath(wd, true, false); err != nil {
		return &DecodeError{
			Code:    "invalid_working_directory",
			Message: fmt.Sprintf("checks[%d].working_directory %q is invalid: %s", index, wd, err),
		}
	}

	rawTimeout, ok := check["timeout_seconds"]
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("checks[%d].timeout_seconds is required", index),
		}
	}
	timeoutN, ok := rawTimeout.(json.Number)
	if !ok {
		return &DecodeError{
			Code:    "invalid_type",
			Message: fmt.Sprintf("checks[%d].timeout_seconds is not a number", index),
		}
	}
	timeout, err := timeoutN.Int64()
	if err != nil {
		return &DecodeError{
			Code:    "invalid_type",
			Message: fmt.Sprintf("checks[%d].timeout_seconds is not an integer", index),
		}
	}
	if timeout < 1 || timeout > MaxCheckTimeoutSeconds {
		return &DecodeError{
			Code:    "invalid_timeout",
			Message: fmt.Sprintf("checks[%d].timeout_seconds %d is not in [1, %d]", index, timeout, MaxCheckTimeoutSeconds),
		}
	}

	envAny, ok := check["environment"]
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("checks[%d].environment is required", index),
		}
	}
	env, ok := envAny.(map[string]any)
	if !ok {
		return &DecodeError{
			Code:    "invalid_type",
			Message: fmt.Sprintf("checks[%d].environment is not an object", index),
		}
	}
	if len(env) > MaxEnvironmentEntries {
		return &DecodeError{
			Code:    "too_many_env_entries",
			Message: fmt.Sprintf("checks[%d].environment count %d exceeds %d", index, len(env), MaxEnvironmentEntries),
		}
	}
	for name, rawValue := range env {
		if !environmentNamePattern.MatchString(name) {
			return &DecodeError{
				Code:    "invalid_env_name",
				Message: fmt.Sprintf("checks[%d].environment key %q is invalid", index, name),
			}
		}
		value, ok := rawValue.(string)
		if !ok || strings.ContainsRune(value, 0) {
			return &DecodeError{
				Code:    "invalid_env_value",
				Message: fmt.Sprintf("checks[%d].environment[%q] is invalid", index, name),
			}
		}
	}

	if reason, ok := check["reason"].(string); ok && reason != "" {
		return &DecodeError{
			Code:    "runnable_check_with_reason",
			Message: fmt.Sprintf("checks[%d] is run-mode but carries a reason", index),
		}
	}

	return nil
}

// validateExcludeCheckMap enforces the exclude-mode check
// rules: reason present and well-formed, no run-only
// fields (argv, working_directory, timeout_seconds,
// environment).
func validateExcludeCheckMap(index int, check map[string]any) error {
	reason, ok := check["reason"].(string)
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("checks[%d].reason is required for exclude mode", index),
		}
	}
	if strings.TrimSpace(reason) == "" || strings.ContainsAny(reason, "\r\n") || len(reason) > 240 || containsClosurePlaceholder(reason) {
		return &DecodeError{
			Code:    "invalid_reason",
			Message: fmt.Sprintf("checks[%d].reason %q is invalid", index, reason),
		}
	}
	if v, ok := check["argv"]; ok {
		if arr, ok := v.([]any); ok && len(arr) > 0 {
			return &DecodeError{
				Code:    "exclude_with_run_only_field",
				Message: fmt.Sprintf("checks[%d] is exclude-mode but carries argv", index),
			}
		}
	}
	if v, ok := check["working_directory"].(string); ok && v != "" {
		return &DecodeError{
			Code:    "exclude_with_run_only_field",
			Message: fmt.Sprintf("checks[%d] is exclude-mode but carries working_directory", index),
		}
	}
	if v, ok := check["timeout_seconds"]; ok {
		if n, ok := v.(json.Number); ok {
			if iv, err := n.Int64(); err == nil && iv != 0 {
				return &DecodeError{
					Code:    "exclude_with_run_only_field",
					Message: fmt.Sprintf("checks[%d] is exclude-mode but carries timeout_seconds", index),
				}
			}
		}
	}
	if v, ok := check["environment"]; ok {
		if env, ok := v.(map[string]any); ok && len(env) > 0 {
			return &DecodeError{
				Code:    "exclude_with_run_only_field",
				Message: fmt.Sprintf("checks[%d] is exclude-mode but carries environment", index),
			}
		}
	}
	return nil
}

// validateArtifactMap enforces per-artifact rules.
func validateArtifactMap(index int, raw any, seenIDs map[string]struct{}) error {
	artifact, ok := raw.(map[string]any)
	if !ok {
		return &DecodeError{
			Code:    "invalid_type",
			Message: fmt.Sprintf("artifacts[%d] is not an object", index),
		}
	}
	id, ok := artifact["id"].(string)
	if !ok || id == "" {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("artifacts[%d].id is required", index),
		}
	}
	if !itemIDPattern.MatchString(id) || containsClosurePlaceholder(id) {
		return &DecodeError{
			Code:    "invalid_artifact_id",
			Message: fmt.Sprintf("artifacts[%d].id %q is invalid", index, id),
		}
	}
// Message wording mirrors the closure package's typed
// validator so substring-matching tests continue to pass.
	if _, dup := seenIDs[id]; dup {
		return &DecodeError{
			Code:    "duplicate_artifact_id",
			Message: fmt.Sprintf("duplicate artifact id %q at artifacts[%d]", id, index),
		}
	}
	seenIDs[id] = struct{}{}

	path, ok := artifact["path"].(string)
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("artifacts[%d].path is required", index),
		}
	}
	if err := validateRepositoryRelativePath(path, false, false); err != nil {
		return &DecodeError{
			Code:    "invalid_artifact_path",
			Message: fmt.Sprintf("artifacts[%d].path %q is invalid: %s", index, path, err),
		}
	}

	if _, ok := artifact["required"]; !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("artifacts[%d].required is required", index),
		}
	}

	rawMax, ok := artifact["max_bytes"]
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: fmt.Sprintf("artifacts[%d].max_bytes is required", index),
		}
	}
	maxN, ok := rawMax.(json.Number)
	if !ok {
		return &DecodeError{
			Code:    "invalid_type",
			Message: fmt.Sprintf("artifacts[%d].max_bytes is not a number", index),
		}
	}
	maxIv, err := maxN.Int64()
	if err != nil || maxIv <= 0 {
		return &DecodeError{
			Code:    "invalid_max_bytes",
			Message: fmt.Sprintf("artifacts[%d].max_bytes %s is not a positive integer", index, maxN.String()),
		}
	}

	mediaType, ok := artifact["media_type"].(string)
	if !ok || strings.TrimSpace(mediaType) == "" || containsClosurePlaceholder(mediaType) {
		return &DecodeError{
			Code:    "invalid_media_type",
			Message: fmt.Sprintf("artifacts[%d].media_type is invalid", index),
		}
	}

	return nil
}

// validatePolicyMap enforces the policy-field shape. Each
// required field MUST be present (the typed Plan's *bool
// pointer fields distinguish present from absent) and MUST
// be a JSON boolean. Required fields mirror the closure
// package's missingPlanPolicyFields contract.
func validatePolicyMap(policy map[string]any) error {
	required := []string{
		"require_clean_before",
		"require_clean_after",
		"forbid_tracked_full_digests",
		"require_diff_check",
	}
	for _, key := range required {
		v, ok := policy[key]
		if !ok {
			return &DecodeError{
				Code:    "missing_field",
				Message: fmt.Sprintf("policy.%s is required", key),
			}
		}
		if _, ok := v.(bool); !ok {
			return &DecodeError{
				Code:    "invalid_policy_constraint",
				Message: fmt.Sprintf("policy.%s is not a boolean", key),
			}
		}
	}
	for key := range policy {
		found := false
		for _, req := range required {
			if key == req {
				found = true
				break
			}
		}
		if !found {
			return &DecodeError{
				Code:    "invalid_policy_field",
				Message: fmt.Sprintf("policy.%s is not a known policy field", key),
			}
		}
	}
	return nil
}

// validateRunnerAuthorityMap enforces the runner_authority
// shape. mode MUST be subject_exact or tool_release_exact;
// tool_release_exact requires a tool block.
func validateRunnerAuthorityMap(ra map[string]any) error {
	mode, ok := ra["mode"].(string)
	if !ok {
		return &DecodeError{
			Code:    "missing_field",
			Message: "runner_authority.mode is required",
		}
	}
// hasToolBlock reports whether the runner_authority block
// carries a non-null tool object. JSON null is treated as
// "no tool" because the typed Plan's *ToolAuthority is a
// pointer and Go's JSON decoder reads null as nil.
	switch mode {
	case "subject_exact":
		if hasToolBlock(ra) {
			return &DecodeError{
				Code:    "subject_exact_with_tool",
				Message: "runner_authority.subject_exact must not carry a tool block",
			}
		}
	case "tool_release_exact":
		tool, ok := toolBlock(ra)
		if !ok {
			return &DecodeError{
				Code:    "missing_field",
				Message: "runner_authority.tool_release_exact requires a tool block",
			}
		}
		revision, ok := tool["revision"].(string)
		if !ok || !oidPattern.MatchString(revision) || containsClosurePlaceholder(revision) {
			return &DecodeError{
				Code:    "invalid_tool_revision",
				Message: "runner_authority.tool.revision is not a valid OID",
			}
		}
		sha, ok := tool["binary_sha256"].(string)
		if !ok || len(sha) != 64 {
			return &DecodeError{
				Code:    "invalid_tool_sha256",
				Message: "runner_authority.tool.binary_sha256 is not a 64-character hex string",
			}
		}
	default:
		return &DecodeError{
			Code:    "invalid_runner_authority_mode",
			Message: fmt.Sprintf("runner_authority.mode %q is not subject_exact|tool_release_exact", mode),
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

// validateRepositoryRelativePath is the canonical plan path
// rule. The rule rejects empty paths, absolute paths,
// null bytes, placeholders, and lexically unclean paths.
// The allowDot flag mirrors the closure package helper.
func validateRepositoryRelativePath(path string, allowDot bool, allowAbs bool) error {
	_ = allowAbs
	if path == "" || strings.ContainsRune(path, 0) || containsClosurePlaceholder(path) {
		return fmt.Errorf("must be a non-empty repository-relative path")
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("must be a non-empty repository-relative path")
	}
	clean := filepath.Clean(path)
	if clean == "." && allowDot {
		return nil
	}
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("must not escape the repository")
	}
	if clean != path {
		return fmt.Errorf("must be lexically clean")
	}
	return nil
}
