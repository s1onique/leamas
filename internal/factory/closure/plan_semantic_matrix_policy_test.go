package closure

import (
	"errors"
	"testing"
)

// TestValidatePlanSemanticMatrixPolicy tests semantic errors for policy properties.
func TestValidatePlanSemanticMatrixPolicy(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*Plan)
		wantCount int
		wantPath  string
		wantCode  PlanValidationCode
		wantKw    PlanValidationKeyword
	}{
		// Policy - individual fields
		{
			name:      "policy_require_clean_before missing",
			mutate:    func(p *Plan) { p.Policy.RequireCleanBefore = nil },
			wantCount: 1, wantPath: "/policy/require_clean_before",
			wantCode: PlanCodeRequiredPropertyMissing, wantKw: KeywordRequired,
		},
		{
			name:      "policy_require_diff_check missing",
			mutate:    func(p *Plan) { p.Policy.RequireDiffCheck = nil },
			wantCount: 1, wantPath: "/policy/require_diff_check",
			wantCode: PlanCodeRequiredPropertyMissing, wantKw: KeywordRequired,
		},
		{
			name:      "policy_require_clean_after missing",
			mutate:    func(p *Plan) { p.Policy.RequireCleanAfter = nil },
			wantCount: 1, wantPath: "/policy/require_clean_after",
			wantCode: PlanCodeRequiredPropertyMissing, wantKw: KeywordRequired,
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
