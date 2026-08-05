package closure

import (
	"math"
	"testing"
)

// plan_run_execution_fields_mutation_test.go owns the
// mutation-sensitivity tests for the run-mode
// `working_directory` and `timeout_seconds` fields. Splitting
// it from the main differential test file keeps both under
// the LLM-friendly 400-line threshold.

// TestDifferentialMutationSensitivity mutates isolated copies of
// the generated schema and proves the differential evaluator's
// outcome changes. The runtime side is not mutated; the test
// proves the schema evaluator is bound to JSONSchema() output.
func TestDifferentialMutationSensitivity(t *testing.T) {
	type tc struct {
		name    string
		mutate  func(schema map[string]any)
		value   any
		timeout any
		wantStd bool
		wantExt bool
	}
	beyondNative := math.MaxInt64
	cases := []tc{
		{
			name: "minLength_1_to_2",
			mutate: func(s map[string]any) {
				workingDirectorySchemaMap(t, s)["minLength"] = 2
			},
			value:   ".",
			timeout: 60,
			wantStd: false,
			wantExt: false,
		},
		{
			name: "minimum_1_to_2",
			mutate: func(s map[string]any) {
				timeoutSecondsSchemaMap(t, s)["minimum"] = 2
			},
			value:   ".",
			timeout: 1,
			wantStd: false,
			wantExt: false,
		},
		{
			name: "maximum_600_to_599",
			mutate: func(s map[string]any) {
				timeoutSecondsSchemaMap(t, s)["maximum"] = 599
			},
			value:   ".",
			timeout: 600,
			wantStd: false,
			wantExt: false,
		},
		{
			name: "pattern_to_alternative",
			mutate: func(s map[string]any) {
				workingDirectorySchemaMap(t, s)["pattern"] = `^a$`
			},
			value:   ".",
			timeout: 60,
			wantStd: false,
			wantExt: false,
		},
		{
			name: "allow_parent_segments_true",
			mutate: func(s map[string]any) {
				pathPolicyMap(t, s)["allow_parent_segments"] = true
			},
			value:   "../escape",
			timeout: 60,
			wantStd: true, // standard pattern matches ".."
			wantExt: true, // extension accepts (allow_parent_segments=true)
		},
		{
			name: "require_lexically_clean_false",
			mutate: func(s map[string]any) {
				pathPolicyMap(t, s)["require_lexically_clean"] = false
			},
			value:   "foo/./bar",
			timeout: 60,
			wantStd: true,  // standard pattern matches
			wantExt: false, // allow_dot still false (default); single-dot still rejected
		},
		{
			name: "separator_unsupported",
			mutate: func(s map[string]any) {
				pathPolicyMap(t, s)["separator"] = "\\"
			},
			value:   "foo",
			timeout: 60,
			wantStd: true,
			wantExt: false,
		},
		{
			name: "allow_dot_false",
			mutate: func(s map[string]any) {
				pathPolicyMap(t, s)["allow_dot"] = false
			},
			value:   ".",
			timeout: 60,
			wantStd: true,
			wantExt: false,
		},
		{
			name: "timeout_beyond_native",
			mutate: func(s map[string]any) {
				timeoutSecondsSchemaMap(t, s)["maximum"] = int(beyondNative)
			},
			value:   ".",
			timeout: beyondNative,
			wantStd: true,
			wantExt: true,
		},
	}
	for _, m := range cases {
		m := m
		t.Run(m.name, func(t *testing.T) {
			schema, err := JSONSchema()
			if err != nil {
				t.Fatalf("JSONSchema() error = %v", err)
			}
			m.mutate(schema)
			instance := map[string]any{
				"contract_version": float64(1),
				"act_id":           "ACT-LEAMAS-DIFF-MUTATION",
				"baseline": map[string]any{
					"commit_oid": "1111111111111111111111111111111111111111",
					"tree_oid":   "2222222222222222222222222222222222222222",
				},
				"execution": map[string]any{"mode": "serial_fail_fast"},
				"checks": []any{
					map[string]any{
						"id":                "m",
						"mode":              "run",
						"argv":              []any{"true"},
						"working_directory": m.value,
						"timeout_seconds":   m.timeout,
						"environment":       map[string]any{},
					},
				},
				"artifacts": []any{},
				"policy": map[string]any{
					"require_clean_before":        true,
					"require_clean_after":         true,
					"forbid_tracked_full_digests": true,
					"require_diff_check":          true,
				},
			}
			standard := evaluateWithSchemaStandard(schema, instance)
			extension := evaluateWithSchemaExtensionAware(schema, instance)
			if standard.Accept != m.wantStd {
				t.Fatalf("%s: standard accept=%v, want %v (issues=%v)",
					m.name, standard.Accept, m.wantStd, standard.Issues)
			}
			if extension.Accept != m.wantExt {
				t.Fatalf("%s: extension accept=%v, want %v (issues=%v)",
					m.name, extension.Accept, m.wantExt, extension.Issues)
			}
		})
	}
}
