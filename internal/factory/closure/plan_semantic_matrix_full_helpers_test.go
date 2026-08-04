package closure

import (
	"errors"
	"strings"
	"testing"
)

// testCase represents a test case that returns planDiagnosticSource.
type testCase struct {
	name      string
	mutate    func(*Plan)
	wantCount int
	wantDiags []wantDiag
}

// wantDiag represents the expected diagnostic for a case.
type wantDiag struct {
	InstancePath  string
	Code          PlanValidationCode
	Keyword       PlanValidationKeyword
	PropertyName  string
	MessageSubstr string
}

// validateSourceCase validates a case that returns planDiagnosticSource.
func validateSourceCase(t *testing.T, cases []testCase) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := validSemanticPlanFixture(t)
			tc.mutate(&plan)

			err := ValidatePlan(plan)
			if err == nil {
				t.Fatalf("ValidatePlan() = nil, want error for %s", tc.name)
			}

			// Assert errors.As(planDiagnosticSource) = true
			var source planDiagnosticSource
			if !errors.As(err, &source) {
				t.Fatalf("errors.As(err, &planDiagnosticSource) = false for %s", tc.name)
			}

			diags := source.PlanDiagnostics()
			if len(diags) != tc.wantCount {
				t.Fatalf("got %d diagnostics, want %d for %s", len(diags), tc.wantCount, tc.name)
			}

			for i, want := range tc.wantDiags {
				if i >= len(diags) {
					t.Fatalf("not enough diagnostics: index %d not found", i)
				}
				d := diags[i]

				if d.InstancePath != want.InstancePath {
					t.Errorf("InstancePath = %q, want %q for %s", d.InstancePath, want.InstancePath, tc.name)
				}
				if d.Code != want.Code {
					t.Errorf("Code = %v, want %v for %s", d.Code, want.Code, tc.name)
				}
				if d.Keyword != want.Keyword {
					t.Errorf("Keyword = %v, want %v for %s", d.Keyword, want.Keyword, tc.name)
				}
				if want.PropertyName != "" && d.PropertyName != want.PropertyName {
					t.Errorf("PropertyName = %q, want %q for %s", d.PropertyName, want.PropertyName, tc.name)
				}
				if !strings.Contains(d.Message, want.MessageSubstr) {
					t.Errorf("Message = %q, want substring %q for %s", d.Message, want.MessageSubstr, tc.name)
				}
			}
		})
	}
}
