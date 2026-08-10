package closure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure/plancontract"
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

// DecodePlan is the legacy public entry point. It preserves the
// documented contract: parse, decode, and ValidatePlan in
// sequence. The internal composed pipeline routes the bytes
// through parseBoundedClosurePlanDocument (the single bounded
// syntactic authority) and the typed-decoder through
// decodeTypedPlan; composition observability is invocation-local
// via the compositionObserver interface in
// plan_contract_validation.go.
func DecodePlan(data []byte) (Plan, error) {
	root, parseDiagnostics := parseBoundedClosurePlanDocument(data)
	if len(parseDiagnostics) > 0 {
		return Plan{}, errorFromDiagnostics(parseDiagnostics)
	}
	plan, err := decodeTypedPlan(root)
	if err != nil {
		return Plan{}, err
	}
	if err := ValidatePlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

// decodeTypedPlan turns the already-parsed document root into a
// typed Plan. It uses the same canonical JSON encoder/decoder pair
// as the parser so no second syntactic parse occurs. The typed
// decoder uses DisallowUnknownFields so unknown JSON keys still
// surface as a typed decode error even when the structural
// validator has accepted the document.
func decodeTypedPlan(root any) (Plan, error) {
	return decodeTypedPlanWithObserver(root, noopCompositionObserver{})
}

// decodeTypedPlanWithObserver is the internal entry point the
// composed pipeline uses. The observer is invocation-local; tests
// pass a per-assertion counting observer and production passes the
// noop observer.
func decodeTypedPlanWithObserver(root any, observer compositionObserver) (Plan, error) {
	observer.TypedDecoded()
	buf, err := json.Marshal(root)
	if err != nil {
		return Plan{}, fmt.Errorf("marshal parsed plan: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(buf))
	dec.DisallowUnknownFields()
	var plan Plan
	if err := dec.Decode(&plan); err != nil {
		return Plan{}, fmt.Errorf("typed decode: %w", err)
	}
	return plan, nil
}

// errorFromDiagnostics turns a list of structural diagnostics into
// a Go error so the legacy DecodePlan preserves its (Plan{}, error)
// return contract.
func errorFromDiagnostics(diags []PlanValidationError) error {
	if len(diags) == 0 {
		return nil
	}
	return fmt.Errorf("plan rejected by structural validation: %s", diags[0].Message)
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
//
// B2-R2: the bounded syntactic decoder is the canonical
// plancontract.DecodeBytes. The closure runner and the
// evidence package both consume the same parser pass so the
// production decoder and the evidence decoder cannot diverge.
func LoadPlanFromBytes(data []byte) (Plan, []byte, error) {
	root, err := plancontract.DecodeBytes(data)
	if err != nil {
		return Plan{}, nil, fmt.Errorf("plan rejected by structural validation: %s", convertPlanContractError(err))
	}
	plan, err := decodeTypedPlan(root)
	if err != nil {
		return Plan{}, nil, fmt.Errorf("decode closure plan: %w", err)
	}
	return plan, data, nil
}

// convertPlanContractError adapts the plancontract leaf's
// typed errors to a single human-readable string that the
// closure package's legacy errorFromDiagnostics contract
// preserves. The function is a thin wrapper around the
// typed error so callers that want the type can switch on
// it; legacy callers just see the message.
func convertPlanContractError(err error) string {
	return err.Error()
}

func ValidatePlan(plan Plan) error {
	if plan.ContractVersion != ContractVersionV1 {
		return errUnsupportedContractVersion(plan.ContractVersion)
	}
	if !actIDPattern.MatchString(plan.ActID) || containsClosurePlaceholder(plan.ActID) {
		return errInvalidActID(plan.ActID)
	}
	if err := validateBaselineCommitOID(plan.Baseline.CommitOID); err != nil {
		return err
	}
	if err := validateBaselineTreeOID(plan.Baseline.TreeOID); err != nil {
		return err
	}
	if err := validatePlanExecutionMode(plan.Execution); err != nil {
		return err
	}
	if len(plan.Checks) == 0 || len(plan.Checks) > MaxChecks {
		return errInvalidChecksCount(len(plan.Checks))
	}
	if len(plan.Artifacts) > MaxArtifacts {
		return errInvalidArtifactsCount(len(plan.Artifacts))
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
	if err := ValidateRunnerAuthority(plan.RunnerAuthority); err != nil {
		return err
	}
	return nil
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
	seen := make(map[string]int, len(checks))
	for i, check := range checks {
		if !itemIDPattern.MatchString(check.ID) || containsClosurePlaceholder(check.ID) {
			return errInvalidCheckID(i, check.ID)
		}
		if _, exists := seen[check.ID]; exists {
			return errDuplicateCheckID(i, check.ID)
		}
		seen[check.ID] = i
		switch check.Mode {
		case CheckModeRun:
			if err := validateRunnableCheck(i, check); err != nil {
				return err
			}
		case CheckModeExclude:
			if strings.TrimSpace(check.Reason) == "" || strings.ContainsAny(check.Reason, "\r\n") || len(check.Reason) > 240 || containsClosurePlaceholder(check.Reason) {
				return errInvalidCheckReason(i)
			}
			if len(check.Argv) != 0 || check.WorkingDirectory != "" ||
				check.TimeoutSeconds != 0 || check.Environment != nil {
				return errExclusionWithExecutionFields(i, check)
			}
		default:
			return errUnknownCheckMode(i, string(check.Mode))
		}
	}
	return nil
}

func validateRunnableCheck(index int, check PlanCheck) error {
	if len(check.Argv) == 0 || len(check.Argv) > MaxArgvElements {
		return errInvalidCheckArgvCount(index)
	}
	for argIndex, arg := range check.Argv {
		if arg == "" || strings.ContainsRune(arg, 0) || containsClosurePlaceholder(arg) {
			return errInvalidCheckArgvElement(index, argIndex)
		}
	}
	if err := portablePathValidate(check.WorkingDirectory, true, false); err != nil {
		return errInvalidCheckWorkingDirectory(index, err)
	}
	if check.TimeoutSeconds <= 0 || check.TimeoutSeconds > MaxCheckTimeoutSeconds {
		return errInvalidCheckTimeout(index)
	}
	if check.Environment == nil || len(check.Environment) > MaxEnvironmentEntries {
		return errInvalidCheckEnvironment(index)
	}
	// Sort environment keys for deterministic validation.
	keys := make([]string, 0, len(check.Environment))
	for k := range check.Environment {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		value := check.Environment[name]
		if !environmentNamePattern.MatchString(name) || strings.ContainsRune(value, 0) {
			return errInvalidCheckEnvironmentKey(index, name)
		}
	}
	if check.Reason != "" {
		return errRunnableCheckWithReason(index)
	}
	return nil
}

func validatePlanArtifacts(artifacts []PlanArtifact) error {
	seen := make(map[string]int, len(artifacts))
	for i, artifact := range artifacts {
		if !itemIDPattern.MatchString(artifact.ID) || containsClosurePlaceholder(artifact.ID) {
			return errInvalidArtifactID(i, artifact.ID)
		}
		if _, exists := seen[artifact.ID]; exists {
			return errDuplicateArtifactID(i, artifact.ID)
		}
		seen[artifact.ID] = i
		if err := validateRepositoryRelativePath(artifact.Path, false); err != nil {
			return errInvalidArtifactPath(i)
		}
		if artifact.Required == nil {
			return errMissingArtifactRequired(i)
		}
		if artifact.MaxBytes <= 0 {
			return errInvalidArtifactMaxBytes(i)
		}
		if strings.TrimSpace(artifact.MediaType) == "" || containsClosurePlaceholder(artifact.MediaType) {
			return errInvalidArtifactMediaType(i)
		}
		role := ArtifactRoleFor(artifact)
		if !validArtifactRole(role) {
			return errInvalidArtifactRole(i, string(role))
		}
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

// validateOID validates any OID field using the generic string-based dispatch.
// This is used by manifest and runner identity validation where field identity
// is implicit from context. For plan baseline validation, use validateBaselineCommitOID
// and validateBaselineTreeOID directly for explicit field paths.
func validateOID(field, value string) error {
	if containsClosurePlaceholder(value) {
		if field == "baseline.commit_oid" {
			return errBaselineCommitOIDPlaceholder()
		}
		return errBaselineTreeOIDPlaceholder()
	}
	if !oidPattern.MatchString(value) {
		if field == "baseline.commit_oid" {
			return errInvalidBaselineCommitOID(value)
		}
		return errInvalidBaselineTreeOID(value)
	}
	return nil
}
