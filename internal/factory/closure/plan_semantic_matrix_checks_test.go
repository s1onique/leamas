package closure

import (
	"errors"
	"testing"
)

// TestValidatePlanSemanticMatrixChecks tests semantic errors for check properties.
func TestValidatePlanSemanticMatrixChecks(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*Plan)
		wantCount int
		wantPath  string
		wantCode  PlanValidationCode
		wantKw    PlanValidationKeyword
	}{
		// Checks - ID
		{
			name:      "checks_0_id placeholder",
			mutate:    func(p *Plan) { p.Checks[0].ID = "TODO" },
			wantCount: 1, wantPath: "/checks/0/id",
			wantCode: PlanCodeSemanticConstraintFailed, wantKw: KeywordType,
		},
		// Checks - mode
		{
			name:      "checks_0_mode missing",
			mutate:    func(p *Plan) { p.Checks[0].Mode = "" },
			wantCount: 1, wantPath: "/checks/0/mode",
			wantCode: PlanCodeSemanticConstraintFailed, wantKw: KeywordEnum,
		},
		// Checks - argv
		{
			name:      "checks_0_argv element placeholder",
			mutate:    func(p *Plan) { p.Checks[0].Argv[0] = "TODO" },
			wantCount: 1, wantPath: "/checks/0/argv/0",
			wantCode: PlanCodeSemanticConstraintFailed, wantKw: KeywordType,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := validSemanticPlanFixture(t)
			tc.mutate(&plan)

			err := ValidatePlan(plan)
			if err == nil {
				t.Fatalf("ValidatePlan() = nil, want error")
			}

			var diags []PlanValidationError
			var source planDiagnosticSource
			if errors.As(err, &source) {
				diags = source.PlanDiagnostics()
			}
			if len(diags) != tc.wantCount {
				t.Fatalf("got %d diagnostics, want %d", len(diags), tc.wantCount)
			}
			if diags[0].InstancePath != tc.wantPath {
				t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, tc.wantPath)
			}
			if diags[0].Code != tc.wantCode {
				t.Errorf("Code = %v, want %v", diags[0].Code, tc.wantCode)
			}
			if diags[0].Keyword != tc.wantKw {
				t.Errorf("Keyword = %q, want %q", diags[0].Keyword, tc.wantKw)
			}
		})
	}
}
