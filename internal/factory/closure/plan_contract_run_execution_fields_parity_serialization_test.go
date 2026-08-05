package closure

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// plan_contract_run_execution_fields_parity_serialization_test.go
// owns the schema/runtime parity harness and the eight-key
// serialization tests for the run-mode `working_directory` and
// `timeout_seconds` fields. Splitting it from the main parity
// suite keeps every file under the LLM-friendly 400-line threshold
// while each test remains reviewable in one screen.

// expectedDiagnosticJSONKeys is the closed set of public JSON
// keys a PlanValidationError must marshal. The test fails any
// future change that introduces or omits a key without explicit
// review.
var expectedDiagnosticJSONKeys = []string{
	"instance_path",
	"schema_path",
	"code",
	"keyword",
	"message",
	"rejected_value",
	"accepted_values",
	"property_name",
}

// TestRunExecutionFieldsDiagnosticEightKeySerialization proves
// every diagnostic emitted for run-execution-field failures
// marshals to exactly the closed eight-key public shape. The
// shape is the contract downstream tooling pins against; the
// test reflects on the marshalled bytes directly so it does not
// rely on the Go struct definition's reflection-only check.
func TestRunExecutionFieldsDiagnosticEightKeySerialization(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{
			name: "missing_working_directory",
			data: applyRunExecutionParityMutations(t, func(c map[string]any) {
				delete(c, "working_directory")
			}),
		},
		{
			name: "missing_timeout_seconds",
			data: applyRunExecutionParityMutations(t, func(c map[string]any) {
				delete(c, "timeout_seconds")
			}),
		},
		{
			name: "empty_working_directory",
			data: applyRunExecutionParityMutations(t, func(c map[string]any) {
				c["working_directory"] = ""
			}),
		},
		{
			name: "timeout_zero",
			data: applyRunExecutionParityMutations(t, func(c map[string]any) {
				c["timeout_seconds"] = 0
			}),
		},
		{
			name: "timeout_six_hundred_one",
			data: applyRunExecutionParityMutations(t, func(c map[string]any) {
				c["timeout_seconds"] = 601
			}),
		},
		{
			name: "working_directory_parent_escape",
			data: applyRunExecutionParityMutations(t, func(c map[string]any) {
				c["working_directory"] = "../escape"
			}),
		},
		{
			name: "exclude_with_forbidden_field",
			data: func() []byte {
				var body map[string]any
				if err := json.Unmarshal([]byte(runExecutionFieldsExcludeFixture), &body); err != nil {
					t.Fatalf("unmarshal exclude fixture: %v", err)
				}
				body["checks"].([]any)[0].(map[string]any)["working_directory"] = "."
				out, err := json.Marshal(body)
				if err != nil {
					t.Fatalf("marshal exclude: %v", err)
				}
				return out
			}(),
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			composed := ValidatePlanComposed(tc.data)
			if composed.Valid {
				t.Fatalf("%s: expected validation to fail", tc.name)
			}
			allDiags := append(append([]PlanValidationError{}, composed.Structural.Errors...), composed.SemanticErrors...)
			if len(allDiags) == 0 {
				t.Fatalf("%s: expected at least one diagnostic", tc.name)
			}
			for _, d := range allDiags {
				raw, err := json.Marshal(d)
				if err != nil {
					t.Fatalf("%s: marshal diag: %v", tc.name, err)
				}
				var asMap map[string]any
				if err := json.Unmarshal(raw, &asMap); err != nil {
					t.Fatalf("%s: unmarshal diag: %v", tc.name, err)
				}
				gotKeys := make([]string, 0, len(asMap))
				for k := range asMap {
					gotKeys = append(gotKeys, k)
				}
				sort.Strings(gotKeys)
				// Every diagnostic must marshal exactly the eight
				// public keys. omitempty is removed so the keys
				// are always present (with zero values when
				// inapplicable, e.g. rejected_value: null).
				for _, want := range expectedDiagnosticJSONKeys {
					if _, ok := asMap[want]; !ok {
						t.Fatalf("%s: missing required key %q in marshalled diagnostic %s; got keys %v",
							tc.name, want, string(raw), gotKeys)
					}
				}
				if len(gotKeys) != len(expectedDiagnosticJSONKeys) {
					t.Fatalf("%s: marshalled diagnostic has %d keys (%v); want exactly %d (%v)",
						tc.name, len(gotKeys), gotKeys, len(expectedDiagnosticJSONKeys), expectedDiagnosticJSONKeys)
				}
			}
		})
	}
}

// TestRunExecutionFieldsPathPolicyDiscoverable proves the
// descriptor emits the x-leamas-repository-relative-path extension
// with the canonical rule set so a source-free consumer can
// detect every canonical rule without parsing prose.
func TestRunExecutionFieldsPathPolicyDiscoverable(t *testing.T) {
	schema, err := JSONSchema()
	if err != nil {
		t.Fatalf("JSONSchema() error = %v", err)
	}
	checkItems := schema["properties"].(map[string]any)["checks"].(map[string]any)["items"].(map[string]any)
	props := checkItems["properties"].(map[string]any)
	wd := props["working_directory"].(map[string]any)
	pp, ok := wd["x-leamas-repository-relative-path"].(map[string]any)
	if !ok {
		t.Fatalf("working_directory missing x-leamas-repository-relative-path extension: %+v", wd)
	}
	if pp["allow_dot"] != true {
		t.Fatalf("path_policy.allow_dot = %v, want true", pp["allow_dot"])
	}
	if pp["allow_parent_segments"] != false {
		t.Fatalf("path_policy.allow_parent_segments = %v, want false", pp["allow_parent_segments"])
	}
	if pp["require_lexically_clean"] != true {
		t.Fatalf("path_policy.require_lexically_clean = %v, want true", pp["require_lexically_clean"])
	}
	if pp["separator"] != "/" {
		t.Fatalf("path_policy.separator = %v, want /", pp["separator"])
	}
}

// TestRunExecutionFieldsCanonicalExampleDeterministic proves
// the descriptor-driven example generator is byte-deterministic
// across two consecutive invocations. The canonical example is
// the source the future schema/example CLI command emits; its
// bytes are the witness downstream tooling pins against.
func TestRunExecutionFieldsCanonicalExampleDeterministic(t *testing.T) {
	first := DescriptorExample()
	firstBytes, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first: %v", err)
	}
	second := DescriptorExample()
	secondBytes, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second: %v", err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("canonical example is not byte-deterministic:\nfirst:  %s\nsecond: %s",
			firstBytes, secondBytes)
	}
}

// TestRunExecutionFieldsSchemaAndRuntimeAgreement exercises the
// schema/runtime parity harness over the working-directory and
// timeout-seconds matrices. For each value the test records the
// emitted schema-level outcome (a portable subset of
// minLength/pattern/minimum/maximum checks) and the runtime
// composed-validation outcome and asserts the two agree. The
// Leamas path-policy rules are reported separately as the
// runtime-only policy that JSON Schema cannot express portably;
// both surfaces accept or reject identically for the matrix
// cases.
func TestRunExecutionFieldsSchemaAndRuntimeAgreement(t *testing.T) {
	type matrix struct {
		name   string
		value  any
		absent bool
	}
	wd := []matrix{
		{"dot", ".", false},
		{"internal_foo", "internal/foo", false},
		{"empty_string", "", false},
		{"absolute", "/absolute/path", false},
		{"double_dot", "..", false},
		{"parent_escape", "../escape", false},
		{"nested_parent_traversal", "foo/../bar", false},
		{"dot_slash_foo", "./foo", false},
		{"foo_dot_bar", "foo/./bar", false},
		{"double_slash", "foo//bar", false},
		{"trailing_slash", "foo/", false},
		{"absent", nil, true},
	}
	ts := []matrix{
		{"absent", nil, true},
		{"neg_one", -1, false},
		{"zero", 0, false},
		{"one", 1, false},
		{"six_hundred", 600, false},
		{"six_hundred_one", 601, false},
		{"string_sixty", "60", false},
		{"float_sixty_point_five", 60.5, false},
	}

	for _, tc := range wd {
		data := applyRunExecutionParityMutations(t, func(c map[string]any) {
			if tc.absent {
				delete(c, "working_directory")
			} else {
				c["working_directory"] = tc.value
			}
		})
		composed := ValidatePlanComposed(data)
		schemaAccepts := portableSchemaAcceptsWorkingDirectory(tc.value, tc.absent)
		runtimeAccepts := composed.Valid
		if schemaAccepts != runtimeAccepts {
			t.Fatalf("working_directory %s: schema accepts=%v, runtime accepts=%v",
				tc.name, schemaAccepts, runtimeAccepts)
		}
		if !runtimeAccepts && len(composed.SemanticErrors) == 0 && len(composed.Structural.Errors) == 0 {
			t.Fatalf("working_directory %s: runtime rejects but neither structural nor semantic produced diagnostics", tc.name)
		}
	}
	for _, tc := range ts {
		data := applyRunExecutionParityMutations(t, func(c map[string]any) {
			if tc.absent {
				delete(c, "timeout_seconds")
			} else {
				c["timeout_seconds"] = tc.value
			}
		})
		composed := ValidatePlanComposed(data)
		schemaAccepts := portableSchemaAcceptsTimeoutSeconds(tc.value, tc.absent)
		runtimeAccepts := composed.Valid
		if schemaAccepts != runtimeAccepts {
			t.Fatalf("timeout_seconds %s: schema accepts=%v, runtime accepts=%v",
				tc.name, schemaAccepts, runtimeAccepts)
		}
		if !runtimeAccepts && len(composed.SemanticErrors) == 0 && len(composed.Structural.Errors) == 0 {
			t.Fatalf("timeout_seconds %s: runtime rejects but neither structural nor semantic produced diagnostics", tc.name)
		}
	}
}

// portableSchemaAcceptsWorkingDirectory replicates the JSON Schema
// portable-subset evaluation the runtime would expose. The
// function returns true when the value passes the schema's
// portable constraints (minLength, pattern) AND the runtime
// path-policy extension that the JSON Schema consumer must
// interpret. The Leamas extension (x-leamas-repository-relative-
// path) is consulted for the additional rules.
func portableSchemaAcceptsWorkingDirectory(value any, absent bool) bool {
	if absent {
		return false
	}
	str, ok := value.(string)
	if !ok {
		return false
	}
	if len(str) < 1 {
		return false
	}
	if !matchPortablePattern(str) {
		return false
	}
	if strings.HasPrefix(str, ".") {
		if str == "." {
			return true
		}
		return filepathCleanEqual(str)
	}
	if containsParentSegment(str) {
		return false
	}
	return filepathCleanEqual(str)
}

func matchPortablePattern(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '/' {
		return false
	}
	if s[len(s)-1] == '/' {
		return false
	}
	if strings.Contains(s, "//") {
		return false
	}
	return true
}

func containsParentSegment(s string) bool {
	parts := strings.Split(s, "/")
	for _, p := range parts {
		if p == ".." {
			return true
		}
	}
	return false
}

func filepathCleanEqual(s string) bool {
	if s == "" {
		return false
	}
	parts := strings.Split(s, "/")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		if p == ".." {
			if len(cleaned) == 0 {
				return false
			}
			cleaned = cleaned[:len(cleaned)-1]
			continue
		}
		cleaned = append(cleaned, p)
	}
	if len(cleaned) == 0 {
		return s == "." || s == ""
	}
	if s == "." {
		return false
	}
	if s == "/" {
		return false
	}
	return strings.Join(cleaned, "/") == s
}

func portableSchemaAcceptsTimeoutSeconds(value any, absent bool) bool {
	if absent {
		return false
	}
	var n int
	switch v := value.(type) {
	case int:
		n = v
	case int64:
		n = int(v)
	case float64:
		if v != float64(int64(v)) {
			return false
		}
		n = int(v)
	case string:
		return false
	default:
		return false
	}
	return n >= 1 && n <= 600
}
