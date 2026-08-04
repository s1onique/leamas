package closure

import (
	"errors"
	"testing"
)

// TestValidatePlanSemanticMatrixRoot tests semantic errors for root plan properties.
func TestValidatePlanSemanticMatrixRoot(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*Plan)
		wantCount int
		wantPath  string
		wantCode  PlanValidationCode
		wantKw    PlanValidationKeyword
	}{
		// Root - act_id
		{
			name:      "act_id placeholder",
			mutate:    func(p *Plan) { p.ActID = "TODO" },
			wantCount: 1, wantPath: "/act_id",
			wantCode: PlanCodeSemanticConstraintFailed, wantKw: KeywordType,
		},
		// Root - execution mode
		{
			name:      "execution_mode missing",
			mutate:    func(p *Plan) { p.Execution.Mode = nil },
			wantCount: 1, wantPath: "/execution/mode",
			wantCode: PlanCodeSemanticConstraintFailed, wantKw: KeywordEnum,
		},
		{
			name:      "execution_mode empty",
			mutate:    func(p *Plan) { empty := ExecutionMode(""); p.Execution.Mode = &empty },
			wantCount: 1, wantPath: "/execution/mode",
			wantCode: PlanCodeSemanticConstraintFailed, wantKw: KeywordEnum,
		},
		{
			name:      "execution_mode unknown",
			mutate:    func(p *Plan) { unknown := ExecutionMode("unknown"); p.Execution.Mode = &unknown },
			wantCount: 1, wantPath: "/execution/mode",
			wantCode: PlanCodeSemanticConstraintFailed, wantKw: KeywordEnum,
		},
		// Root - checks
		{
			name:      "checks empty",
			mutate:    func(p *Plan) { p.Checks = nil },
			wantCount: 1, wantPath: "/checks",
			wantCode: PlanCodeSemanticConstraintFailed, wantKw: KeywordMinItems,
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
