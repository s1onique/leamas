package closure

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

// plan_contract_validation_helpers.go centralises the small
// structural-validator helpers: contract-version recovery, integer
// coercion, slice membership, deterministic ordering, and the
// compact rendering routines used by tests and future CLI
// diagnostics.

// recoverContractVersion extracts the contract version from raw,
// returning 0 if it is missing or wrong-typed. Callers MUST NOT
// treat 0 as a successful v1 detection.
func recoverContractVersion(raw any, fallback int) int {
	if raw == nil {
		return 0
	}
	asMap, ok := raw.(map[string]any)
	if !ok {
		return 0
	}
	v, present := asMap["contract_version"]
	if !present {
		return 0
	}
	switch n := v.(type) {
	case json.Number:
		if i, err := strconv.ParseInt(string(n), 10, 64); err == nil {
			return int(i)
		}
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// integerValue returns the integer value of a JSON number-shaped
// value or (0, false) if the value is not representable as one.
func integerValue(value any) (int, bool) {
	switch n := value.(type) {
	case json.Number:
		if i, err := strconv.ParseInt(string(n), 10, 64); err == nil {
			return int(i), true
		}
	case float64:
		if n == float64(int64(n)) {
			return int(n), true
		}
	case int:
		return n, true
	}
	return 0, false
}

// stringInSlice reports whether list contains the literal value.
// The function name avoids the containsString collision with the
// test-side helper in run_v2_authority_integration_test.go.
func stringInSlice(list []string, value string) bool {
	for _, candidate := range list {
		if candidate == value {
			return true
		}
	}
	return false
}

// sortDiagnostics orders the diagnostics deterministically. The key
// is (Code, InstancePath, PropertyName, Message) so parity between
// successive runs is bit-identical.
func sortDiagnostics(diagnostics []PlanValidationError) {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		if diagnostics[i].InstancePath != diagnostics[j].InstancePath {
			return diagnostics[i].InstancePath < diagnostics[j].InstancePath
		}
		if diagnostics[i].PropertyName != diagnostics[j].PropertyName {
			return diagnostics[i].PropertyName < diagnostics[j].PropertyName
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
}

// PlanValidationResultString renders a single PlanValidationError as
// a compact, deterministic human-readable string. The format is
//
//	/instance/path:
//	  code
//	  keyword=type
//	  message
//
// and is suitable for tests, logs, and CLI error output.
func (e PlanValidationError) String() string {
	var b strings.Builder
	b.WriteString(e.InstancePath)
	b.WriteString(":\n")
	b.WriteString("  ")
	b.WriteString(string(e.Code))
	b.WriteString("\n  keyword=")
	b.WriteString(string(e.Keyword))
	b.WriteString("\n  ")
	b.WriteString(e.Message)
	if len(e.AcceptedValues) > 0 {
		b.WriteString("\n  accepted=")
		b.WriteString(strings.Join(e.AcceptedValues, ","))
	}
	return b.String()
}

// FormatValidationDiagnostics renders a PlanValidationResult as the
// concatenation of every error's compact form. When the result is
// valid the function returns the empty string.
func FormatValidationDiagnostics(result PlanValidationResult) string {
	if result.Valid {
		return ""
	}
	parts := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		parts = append(parts, e.String())
	}
	return strings.Join(parts, "\n")
}
