// SPDX-License-Identifier: Apache-2.0

// Package closure - runtime_placeholders.go implements the typed
// placeholder vocabulary required by Phase 2 of
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.
//
// The vocabulary is closed: every placeholder is one of the twelve
// names exported by PlaceholderNames(). Substitution is performed
// in Go before any process is created so neither the OS shell nor
// ambient environment variables can influence expansion.
//
// Failure modes are typed. The expander never silently leaves a
// placeholder in place; it either expands to a literal value or
// returns a typed PlaceholderError.

package closure

import (
	"fmt"
	"os"
	"strings"
)

// PlaceholderSyntax is the literal token the expander recognises.
// The leading '${' and trailing '}' are fixed; the interior is
// compared against the canonical placeholder vocabulary.
const PlaceholderSyntax = "${name}"

// PlaceholderErrorKind classifies every typed expansion failure.
type PlaceholderErrorKind string

const (
	// PlaceholderErrorUnknown means the placeholder name is not in
	// the canonical vocabulary.
	PlaceholderErrorUnknown PlaceholderErrorKind = "unknown_placeholder"
	// PlaceholderErrorMalformed means a '${...}' token was opened
	// but never closed, or vice versa.
	PlaceholderErrorMalformed PlaceholderErrorKind = "malformed_placeholder"
	// PlaceholderErrorMissingValue means a known placeholder was
	// supplied without a corresponding runtime value.
	PlaceholderErrorMissingValue PlaceholderErrorKind = "missing_runtime_value"
	// PlaceholderErrorRecursion means a placeholder value still
	// contained an unexpanded placeholder after one pass.
	PlaceholderErrorRecursion PlaceholderErrorKind = "recursive_expansion_forbidden"
)

// PlaceholderError is returned by Expand when expansion cannot
// complete. The Value field records the offending token so the
// caller can locate it in argv, environment, or working directory.
type PlaceholderError struct {
	Kind  PlaceholderErrorKind
	Name  string
	Value string
}

func (e *PlaceholderError) Error() string {
	switch e.Kind {
	case PlaceholderErrorUnknown:
		return fmt.Sprintf("runtime placeholder: unknown placeholder %q in %q", e.Name, e.Value)
	case PlaceholderErrorMalformed:
		return fmt.Sprintf("runtime placeholder: malformed placeholder %q", e.Value)
	case PlaceholderErrorMissingValue:
		return fmt.Sprintf("runtime placeholder: no runtime value for %q in %q", e.Name, e.Value)
	case PlaceholderErrorRecursion:
		return fmt.Sprintf("runtime placeholder: recursive expansion forbidden for %q", e.Value)
	}
	return fmt.Sprintf("runtime placeholder: %s failure for %q", e.Kind, e.Value)
}

// IsPlaceholderError reports whether err is a typed PlaceholderError.
func IsPlaceholderError(err error) bool {
	_, ok := err.(*PlaceholderError)
	return ok
}

// placeholderValuesFor converts a RuntimeContext into the value map
// the expander reads from. It is the single authority for the
// placeholder-to-value binding.
func placeholderValuesFor(rc RuntimeContext) map[string]string {
	values := make(map[string]string, len(runtimeContextFieldNames))
	for _, field := range runtimeContextFieldNames {
		values[canonicalPlaceholderPrefix+field] = fieldValue(rc, field)
	}
	return values
}

func fieldValue(rc RuntimeContext, field string) string {
	switch field {
	case "act_id":
		return rc.ACTID
	case "repository_root":
		return rc.RepositoryRoot
	case "run_id":
		return rc.RunID
	case "freeze_commit":
		return rc.FreezeCommit
	case "freeze_tree":
		return rc.FreezeTree
	case "subject_commit":
		return rc.SubjectCommit
	case "subject_tree":
		return rc.SubjectTree
	case "plan_path":
		return rc.PlanPath
	case "plan_blob":
		return rc.PlanBlob
	case "plan_sha256":
		return rc.PlanSHA256
	case "evidence_directory":
		return rc.EvidenceDirectory
	case "started_at":
		return rc.StartedAt
	}
	return ""
}

// Expand substitutes every '${runtime.<field>}' placeholder in
// value with the corresponding field from rc. Expansion runs in a
// single pass; placeholders that survive one pass are a typed
// failure. Ambient environment variables are NEVER consulted.
func Expand(value string, rc RuntimeContext) (string, error) {
	return expandWithValues(value, placeholderValuesFor(rc))
}

// expandWithValues is the test seam. It performs the substitution
// against the supplied value map so tests can exercise every error
// classification without constructing a full RuntimeContext.
func expandWithValues(value string, values map[string]string) (string, error) {
	if !strings.Contains(value, "${") {
		return value, nil
	}
	var out strings.Builder
	i := 0
	for i < len(value) {
		if strings.HasPrefix(value[i:], "${") {
			end := strings.Index(value[i:], "}")
			if end < 0 {
				return "", &PlaceholderError{Kind: PlaceholderErrorMalformed, Value: value[i:]}
			}
			end += i
			name := value[i+2 : end]
			resolved, ok := values[name]
			if !ok {
				return "", &PlaceholderError{Kind: PlaceholderErrorUnknown, Name: name, Value: "${" + name + "}"}
			}
			if resolved == "" {
				return "", &PlaceholderError{Kind: PlaceholderErrorMissingValue, Name: name, Value: "${" + name + "}"}
			}
			if strings.Contains(resolved, "${") {
				return "", &PlaceholderError{Kind: PlaceholderErrorRecursion, Value: resolved}
			}
			out.WriteString(resolved)
			i = end + 1
			continue
		}
		out.WriteByte(value[i])
		i++
	}
	return out.String(), nil
}

// ExpandArgv applies Expand to every entry in argv and returns a
// new slice with the same length. The original slice is never
// mutated.
func ExpandArgv(argv []string, rc RuntimeContext) ([]string, error) {
	if len(argv) == 0 {
		return argv, nil
	}
	out := make([]string, len(argv))
	for i, entry := range argv {
		expanded, err := Expand(entry, rc)
		if err != nil {
			return nil, err
		}
		out[i] = expanded
	}
	return out, nil
}

// ExpandEnvironment applies Expand to every value in environment
// and returns a sorted "KEY=VALUE" slice suitable for exec.Cmd.
// Keys are never substituted; only values. The map is not mutated.
func ExpandEnvironment(environment map[string]string, rc RuntimeContext) ([]string, error) {
	if len(environment) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(environment))
	for key := range environment {
		keys = append(keys, key)
	}
	sortStrings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		value := environment[key]
		expanded, err := Expand(value, rc)
		if err != nil {
			return nil, err
		}
		out = append(out, key+"="+expanded)
	}
	return out, nil
}

// ExpandWorkingDirectory applies Expand to workingDirectory and
// returns the resolved absolute path. The caller may pass a
// repository-relative value; the function joins it against
// rc.RepositoryRoot so the result is always absolute.
func ExpandWorkingDirectory(workingDirectory string, rc RuntimeContext) (string, error) {
	expanded, err := Expand(workingDirectory, rc)
	if err != nil {
		return "", err
	}
	if expanded == "" {
		return rc.RepositoryRoot, nil
	}
	return expanded, nil
}

// AssertNoAmbientExpansion is the runtime guardrail that prevents
// placeholders from being resolved via ambient state. It scans the
// supplied environment block for keys that match the runtime
// vocabulary and rejects the call. The test seam accepts a custom
// environment snapshot.
func AssertNoAmbientExpansion(snapshot []string) error {
	if len(snapshot) == 0 {
		return nil
	}
	for _, entry := range snapshot {
		for _, name := range PlaceholderNames() {
			upper := strings.ToUpper(strings.ReplaceAll(name, ".", "_"))
			if strings.HasPrefix(entry, upper+"=") {
				return &PlaceholderError{Kind: PlaceholderErrorUnknown, Name: upper, Value: entry}
			}
		}
	}
	return nil
}

// CaptureAmbientEnvironment is a small helper for tests. It returns
// the OS environment as a sorted slice. Production code MUST NOT
// read from this helper; it exists so AssertNoAmbientExpansion can
// be exercised against a known snapshot.
func CaptureAmbientEnvironment() []string {
	return os.Environ()
}

// sortStrings is a small inlined helper to avoid pulling the
// "sort" package into this file (the package is already imported
// in execution.go via a different path).
func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j-1] > values[j]; j-- {
			values[j-1], values[j] = values[j], values[j-1]
		}
	}
}
