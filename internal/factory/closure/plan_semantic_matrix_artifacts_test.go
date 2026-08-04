package closure

import (
	"errors"
	"testing"
)

// TestValidatePlanSemanticMatrixArtifacts tests semantic errors for artifact properties.
func TestValidatePlanSemanticMatrixArtifacts(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*Plan)
		wantCount int
		wantPath  string
		wantCode  PlanValidationCode
		wantKw    PlanValidationKeyword
	}{
		// Artifacts - ID
		{
			name:      "artifacts_0_id placeholder",
			mutate:    func(p *Plan) { p.Artifacts[0].ID = "TODO" },
			wantCount: 1, wantPath: "/artifacts/0/id",
			wantCode: PlanCodeSemanticConstraintFailed, wantKw: KeywordType,
		},
		// Artifacts - path
		{
			name:      "artifacts_0_path absolute",
			mutate:    func(p *Plan) { p.Artifacts[0].Path = "/absolute/path" },
			wantCount: 1, wantPath: "/artifacts/0/path",
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
