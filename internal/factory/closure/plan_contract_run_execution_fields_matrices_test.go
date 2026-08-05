package closure

import (
	"testing"
)

// plan_contract_run_execution_fields_matrices_test.go owns the
// value-matrix parity tests for the run-mode `working_directory`
// and `timeout_seconds` fields. Splitting it from the main parity
// suite keeps every file under the LLM-friendly 400-line threshold
// while every matrix case remains reviewable in one screen.

func TestRunExecutionFieldsTimeoutMatrix(t *testing.T) {
	type want struct {
		wantValid bool
		keyword   PlanValidationKeyword
	}
	cases := []struct {
		name      string
		value     any
		wantValid bool
		// structuralExpected is true when the structural walker
		// must reject the case; otherwise the semantic walker
		// owns the rejection.
		structuralExpected bool
		want               want
	}{
		{
			name:               "absent",
			value:              nil,
			wantValid:          false,
			structuralExpected: true,
			want:               want{keyword: KeywordRequired},
		},
		{
			name:               "zero",
			value:              0,
			wantValid:          false,
			structuralExpected: true,
			want:               want{keyword: KeywordMinimum},
		},
		{
			name:      "lower_bound_one",
			value:     1,
			wantValid: true,
			want:      want{},
		},
		{
			name:      "upper_bound_six_hundred",
			value:     600,
			wantValid: true,
			want:      want{},
		},
		{
			name:               "exceeds_max",
			value:              601,
			wantValid:          false,
			structuralExpected: true,
			want:               want{keyword: KeywordMaximum},
		},
		{
			name:               "negative",
			value:              -1,
			wantValid:          false,
			structuralExpected: true,
			want:               want{keyword: KeywordMinimum},
		},
		{
			name:               "wrong_json_type_string",
			value:              "60",
			wantValid:          false,
			structuralExpected: true,
			want:               want{keyword: KeywordType},
		},
		{
			name:               "wrong_json_type_float",
			value:              60.5,
			wantValid:          false,
			structuralExpected: true,
			want:               want{keyword: KeywordType},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data := applyRunExecutionParityMutations(t, func(check map[string]any) {
				if tc.value == nil {
					delete(check, "timeout_seconds")
				} else {
					check["timeout_seconds"] = tc.value
				}
			})
			structResult := ValidatePlanStructural(data)
			if tc.structuralExpected && structResult.Valid {
				t.Fatalf("%s: structural must reject; got valid", tc.name)
			}
			composed := ValidatePlanComposed(data)
			if composed.Valid != tc.wantValid {
				t.Fatalf("%s: composed.Valid = %v, want %v (semantic=%+v)",
					tc.name, composed.Valid, tc.wantValid, composed.SemanticErrors)
			}
			if !tc.structuralExpected {
				return
			}
			diag := findDiagnosticAt(structResult.Errors, "/checks/0/timeout_seconds")
			if diag == nil {
				t.Fatalf("%s: expected diagnostic at /checks/0/timeout_seconds; got %+v",
					tc.name, structResult.Errors)
			}
			if diag.Keyword != tc.want.keyword {
				t.Fatalf("%s: keyword = %q, want %q", tc.name, diag.Keyword, tc.want.keyword)
			}
			if diag.PropertyName != "timeout_seconds" {
				t.Fatalf("%s: property_name = %q, want timeout_seconds", tc.name, diag.PropertyName)
			}
		})
	}
}

// TestRunExecutionFieldsSchemaParity proves the emitted JSON
// Schema declares the same constraints the runtime enforces. The
// parity probe walks the schema for /checks/[]/working_directory
// and /checks/[]/timeout_seconds and asserts each constraint is
// present and accurate.
