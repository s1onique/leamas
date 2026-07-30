package closure

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	actIDPattern           = regexp.MustCompile(`^ACT-[A-Z0-9][A-Z0-9-]{2,199}$`)
	itemIDPattern          = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	oidPattern             = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
	environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// planExecutionModePath is the canonical JSON pointer used in every
// diagnostic that names the execution-mode field. Centralising the
// string keeps the runtime, JSON Schema, and CLI subprocess tests
// aligned.
const planExecutionModePath = "/execution/mode"

func DecodePlan(data []byte) (Plan, error) {
	var plan Plan
	if err := decodeStrictBounded(data, MaxPlanBytes, &plan); err != nil {
		return Plan{}, err
	}
	if err := ValidatePlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func LoadPlan(path string) (Plan, []byte, error) {
	data, err := readBoundedFile(path, MaxPlanBytes)
	if err != nil {
		return Plan{}, nil, fmt.Errorf("read closure plan: %w", err)
	}
	plan, err := DecodePlan(data)
	if err != nil {
		return Plan{}, nil, fmt.Errorf("validate closure plan: %w", err)
	}
	return plan, data, nil
}

// LoadPlanFromBytes parses plan bytes without reading from the filesystem.
// It enforces the size bound and strict JSON syntax only; callers that need
// an executable plan must subsequently invoke ValidatePlan explicitly.
func LoadPlanFromBytes(data []byte) (Plan, []byte, error) {
	if len(data) > MaxPlanBytes {
		return Plan{}, nil, fmt.Errorf("plan exceeds maximum size: %d > %d", len(data), MaxPlanBytes)
	}
	var plan Plan
	if err := decodeStrictBounded(data, MaxPlanBytes, &plan); err != nil {
		return Plan{}, nil, fmt.Errorf("decode closure plan: %w", err)
	}
	return plan, data, nil
}

func ValidatePlan(plan Plan) error {
	if plan.ContractVersion != ContractVersionV1 {
		return fmt.Errorf("unsupported closure plan contract_version %d", plan.ContractVersion)
	}
	if !actIDPattern.MatchString(plan.ActID) || containsClosurePlaceholder(plan.ActID) {
		return fmt.Errorf("invalid act_id %q", plan.ActID)
	}
	if err := validateOID("baseline.commit_oid", plan.Baseline.CommitOID); err != nil {
		return err
	}
	if err := validateOID("baseline.tree_oid", plan.Baseline.TreeOID); err != nil {
		return err
	}
	if err := validatePlanExecutionMode(plan.Execution); err != nil {
		return err
	}
	if len(plan.Checks) == 0 || len(plan.Checks) > MaxChecks {
		return fmt.Errorf("checks count must be between 1 and %d", MaxChecks)
	}
	if len(plan.Artifacts) > MaxArtifacts {
		return fmt.Errorf("artifacts count exceeds %d", MaxArtifacts)
	}
	if err := validatePlanChecks(plan.Checks); err != nil {
		return err
	}
	if err := validatePlanArtifacts(plan.Artifacts); err != nil {
		return err
	}
	if err := validatePlanPolicy(plan.Policy); err != nil {
		return err
	}
	if err := validatePlanAuthority(plan); err != nil {
		return err
	}
	return nil
}

// validatePlanPolicy replaces the previous "all policy fields are
// required" generic message with exact per-sibling diagnostics. The
// validator reports each missing sibling under its /policy/<name>
// JSON pointer so callers can fix every gap in a single edit. The
// validator also keeps the historical "closure v1 requires clean
// worktree before and after" diagnostic, which is a semantic
// constraint rather than a missing-property diagnostic.
func validatePlanPolicy(policy PlanPolicy) error {
	missing := missingPlanPolicyFields(policy)
	if len(missing) > 0 {
		return &PlanPolicyRequiredError{Missing: missing}
	}
	if !*policy.RequireCleanBefore || !*policy.RequireCleanAfter {
		return fmt.Errorf("closure v1 requires clean worktree before and after")
	}
	return nil
}

// missingPlanPolicyFields returns the ordered list of /policy
// sibling names whose value is missing. The order is read from
// the descriptor's /policy.Required set (via PolicyFieldOrder)
// so the descriptor remains the single authority; the typed
// PlanPolicy struct's pointer fields are the value source.
func missingPlanPolicyFields(policy PlanPolicy) []string {
	values := map[string]*bool{
		"require_clean_before":        policy.RequireCleanBefore,
		"require_clean_after":         policy.RequireCleanAfter,
		"forbid_tracked_full_digests": policy.ForbidTrackedFullDigests,
		"require_diff_check":          policy.RequireDiffCheck,
	}
	order := PolicyFieldOrder()
	missing := make([]string, 0, len(order))
	for _, name := range order {
		if values[name] == nil {
			missing = append(missing, name)
		}
	}
	return missing
}

// PlanPolicyRequiredError is the typed diagnostic the policy
// validator emits when one or more policy siblings are absent. The
// error is JSON-marshallable so future CLI flags can render it
// directly. The Missing slice is the ordered list of sibling names
// that were absent.
type PlanPolicyRequiredError struct {
	Missing []string
}

// Error implements the error interface. The message format lists
// every missing sibling, separated by ", ", so consumers that print
// the error see all gaps at once.
func (e *PlanPolicyRequiredError) Error() string {
	if len(e.Missing) == 0 {
		return "policy fields missing"
	}
	return fmt.Sprintf("missing required policy field(s): %s", strings.Join(e.Missing, ", "))
}

// IsPlanPolicyRequiredError reports whether err (or any wrapped
// error) is a *PlanPolicyRequiredError. The helper lets callers
// route the diagnostic through a dedicated path without exposing
// the concrete error type.
func IsPlanPolicyRequiredError(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(*PlanPolicyRequiredError); ok {
		return true
	}
	if unwrap, ok := err.(interface{ Unwrap() error }); ok {
		return IsPlanPolicyRequiredError(unwrap.Unwrap())
	}
	return false
}

// validatePlanExecutionMode is the single, authoritative entry point
// for runtime execution-mode validation. It distinguishes every
// presence category the JSON Schema recognises:
//
//   - the property absent            → ExecutionModeMissing;
//   - the property present, ""        → ExecutionModePresentEmpty;
//   - the property present, "   "     → ExecutionModePresentWhitespace;
//   - the property present, anything else not in the closed enum
//     → ExecutionModePresentUnknown.
//
// Every category is rejected. The validator never falls back to a
// privileged default mode and never accepts an alias.
func validatePlanExecutionMode(execution PlanExecution) error {
	if execution.Mode == nil {
		return &ExecutionModeError{
			Path:      planExecutionModePath,
			Value:     "",
			Presence:  ExecutionModeMissing,
			Supported: SupportedExecutionModes(),
		}
	}
	_, err := ParseExecutionMode(planExecutionModePath, string(*execution.Mode))
	return err
}

func validatePlanChecks(checks []PlanCheck) error {
	seen := make(map[string]struct{}, len(checks))
	for i, check := range checks {
		if !itemIDPattern.MatchString(check.ID) || containsClosurePlaceholder(check.ID) {
			return fmt.Errorf("checks[%d].id is invalid", i)
		}
		if _, exists := seen[check.ID]; exists {
			return fmt.Errorf("duplicate check id %q", check.ID)
		}
		seen[check.ID] = struct{}{}
		switch check.Mode {
		case CheckModeRun:
			if err := validateRunnableCheck(i, check); err != nil {
				return err
			}
		case CheckModeExclude:
			if strings.TrimSpace(check.Reason) == "" || strings.ContainsAny(check.Reason, "\r\n") || len(check.Reason) > 240 || containsClosurePlaceholder(check.Reason) {
				return fmt.Errorf("checks[%d].reason is required and must be compact final prose", i)
			}
			if len(check.Argv) != 0 || check.WorkingDirectory != "" ||
				check.TimeoutSeconds != 0 || check.Environment != nil {
				return fmt.Errorf("checks[%d] exclusion contains execution fields", i)
			}
		default:
			return fmt.Errorf("checks[%d] has unknown mode %q", i, check.Mode)
		}
	}
	return nil
}

func validateRunnableCheck(index int, check PlanCheck) error {
	if len(check.Argv) == 0 || len(check.Argv) > MaxArgvElements {
		return fmt.Errorf("checks[%d].argv count must be between 1 and %d", index, MaxArgvElements)
	}
	for argIndex, arg := range check.Argv {
		if arg == "" || strings.ContainsRune(arg, 0) || containsClosurePlaceholder(arg) {
			return fmt.Errorf("checks[%d].argv[%d] is invalid or contains a placeholder", index, argIndex)
		}
	}
	if err := validateRepositoryRelativePath(check.WorkingDirectory, true); err != nil {
		return fmt.Errorf("checks[%d].working_directory: %w", index, err)
	}
	if check.TimeoutSeconds <= 0 || check.TimeoutSeconds > MaxCheckTimeoutSeconds {
		return fmt.Errorf("checks[%d].timeout_seconds must be between 1 and %d", index, MaxCheckTimeoutSeconds)
	}
	if check.Environment == nil || len(check.Environment) > MaxEnvironmentEntries {
		return fmt.Errorf("checks[%d].environment must be an object with at most %d entries", index, MaxEnvironmentEntries)
	}
	for name, value := range check.Environment {
		if !environmentNamePattern.MatchString(name) || strings.ContainsRune(value, 0) {
			return fmt.Errorf("checks[%d].environment contains invalid entry %q", index, name)
		}
	}
	if check.Reason != "" {
		return fmt.Errorf("checks[%d] runnable check contains exclusion reason", index)
	}
	return nil
}

func validatePlanArtifacts(artifacts []PlanArtifact) error {
	seen := make(map[string]struct{}, len(artifacts))
	for i, artifact := range artifacts {
		if !itemIDPattern.MatchString(artifact.ID) || containsClosurePlaceholder(artifact.ID) {
			return fmt.Errorf("artifacts[%d].id is invalid", i)
		}
		if _, exists := seen[artifact.ID]; exists {
			return fmt.Errorf("duplicate artifact id %q", artifact.ID)
		}
		seen[artifact.ID] = struct{}{}
		if err := validateRepositoryRelativePath(artifact.Path, false); err != nil {
			return fmt.Errorf("artifacts[%d].path: %w", i, err)
		}
		if artifact.Required == nil {
			return fmt.Errorf("artifacts[%d].required is missing", i)
		}
		if artifact.MaxBytes <= 0 {
			return fmt.Errorf("artifacts[%d].max_bytes must be positive", i)
		}
		if strings.TrimSpace(artifact.MediaType) == "" || containsClosurePlaceholder(artifact.MediaType) {
			return fmt.Errorf("artifacts[%d].media_type is invalid", i)
		}
		role := ArtifactRoleFor(artifact)
		if !validArtifactRole(role) {
			return fmt.Errorf("artifacts[%d].role %q is invalid", i, role)
		}
	}
	return nil
}

func validateOID(field, value string) error {
	if containsClosurePlaceholder(value) {
		return fmt.Errorf("%s contains a closure placeholder", field)
	}
	if !oidPattern.MatchString(value) {
		return fmt.Errorf("%s must be a full lowercase Git OID", field)
	}
	return nil
}

func validateRepositoryRelativePath(path string, allowDot bool) error {
	if path == "" || filepath.IsAbs(path) || strings.ContainsRune(path, 0) || containsClosurePlaceholder(path) {
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

func readBoundedFile(path string, limit int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	if info.Size() > int64(limit) {
		return nil, fmt.Errorf("file exceeds %d-byte limit", limit)
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("file exceeds %d-byte limit", limit)
	}
	return data, nil
}
