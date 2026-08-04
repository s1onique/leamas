package closure

import (
	"strings"
	"testing"
)

// TestValidatePlanSemanticMatrixFull_Part3 covers:
// - checks_0_timeout_seconds
// - checks_0_environment
// - checks_exclusion/*
// - checks_run/with_reason
// - checks/duplicate_id
func TestValidatePlanSemanticMatrixFull_Part3(t *testing.T) {
	validateSourceCase(t, buildSourceMatrixCasesPart3())
}

// buildSourceMatrixCasesPart3 returns test cases for Part 3.
func buildSourceMatrixCasesPart3() []testCase {
	return []testCase{
		// =================================================================
		// CHECKS - TIMEOUT (run mode)
		// =================================================================

		{
			name:      "checks_0_timeout_seconds/zero",
			mutate:    func(p *Plan) { p.Checks[0].TimeoutSeconds = 0 },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/timeout_seconds",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be between 1 and",
				},
			},
		},
		{
			name:      "checks_0_timeout_seconds/negative",
			mutate:    func(p *Plan) { p.Checks[0].TimeoutSeconds = -1 },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/timeout_seconds",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be between 1 and",
				},
			},
		},
		{
			name:      "checks_0_timeout_seconds/exceeds_max",
			mutate:    func(p *Plan) { p.Checks[0].TimeoutSeconds = MaxCheckTimeoutSeconds + 1 },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/timeout_seconds",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be between 1 and",
				},
			},
		},

		// =================================================================
		// CHECKS - ENVIRONMENT (run mode)
		// =================================================================

		{
			name:      "checks_0_environment/nil",
			mutate:    func(p *Plan) { p.Checks[0].Environment = nil },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/environment",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be an object with at most",
				},
			},
		},
		{
			name:      "checks_0_environment/invalid_key_starts_with_digit",
			mutate:    func(p *Plan) { p.Checks[0].Environment = map[string]string{"1INVALID": "value"} },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/environment/1INVALID",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "contains invalid entry",
				},
			},
		},
		{
			name:      "checks_0_environment/invalid_key_special_chars",
			mutate:    func(p *Plan) { p.Checks[0].Environment = map[string]string{"INVALID-KEY": "value"} },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/environment/INVALID-KEY",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "contains invalid entry",
				},
			},
		},

		// =================================================================
		// CHECKS - EXCLUSION MODE
		// =================================================================

		{
			name: "checks_exclusion/reason_missing",
			mutate: func(p *Plan) {
				p.Checks = []PlanCheck{
					{ID: "legacy-check", Mode: CheckModeExclude, Reason: ""},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/reason",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is required and must be compact final prose",
				},
			},
		},
		{
			name: "checks_exclusion/reason_placeholder",
			mutate: func(p *Plan) {
				p.Checks = []PlanCheck{
					{ID: "legacy-check", Mode: CheckModeExclude, Reason: "TODO"},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/reason",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is required and must be compact final prose",
				},
			},
		},
		{
			name: "checks_exclusion/reason_with_newline",
			mutate: func(p *Plan) {
				p.Checks = []PlanCheck{
					{ID: "legacy-check", Mode: CheckModeExclude, Reason: "Reason with\nnewline"},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/reason",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is required and must be compact final prose",
				},
			},
		},
		{
			name: "checks_exclusion/reason_with_carriage_return",
			mutate: func(p *Plan) {
				p.Checks = []PlanCheck{
					{ID: "legacy-check", Mode: CheckModeExclude, Reason: "Reason with\rreturn"},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/reason",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is required and must be compact final prose",
				},
			},
		},
		{
			name: "checks_exclusion/reason_too_long",
			mutate: func(p *Plan) {
				p.Checks = []PlanCheck{
					{ID: "legacy-check", Mode: CheckModeExclude, Reason: strings.Repeat("a", 241)},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/reason",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is required and must be compact final prose",
				},
			},
		},
		{
			name: "checks_exclusion/with_argv",
			mutate: func(p *Plan) {
				p.Checks = []PlanCheck{
					{ID: "legacy-check", Mode: CheckModeExclude, Reason: "Legacy check", Argv: []string{"echo", "test"}},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/argv",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "contains forbidden field",
				},
			},
		},
		{
			name: "checks_exclusion/with_working_directory",
			mutate: func(p *Plan) {
				p.Checks = []PlanCheck{
					{ID: "legacy-check", Mode: CheckModeExclude, Reason: "Legacy check", WorkingDirectory: "."},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/working_directory",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "contains forbidden field",
				},
			},
		},
		{
			name: "checks_exclusion/with_timeout",
			mutate: func(p *Plan) {
				p.Checks = []PlanCheck{
					{ID: "legacy-check", Mode: CheckModeExclude, Reason: "Legacy check", TimeoutSeconds: 300},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/timeout_seconds",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "contains forbidden field",
				},
			},
		},
		{
			name: "checks_exclusion/with_environment",
			mutate: func(p *Plan) {
				p.Checks = []PlanCheck{
					{ID: "legacy-check", Mode: CheckModeExclude, Reason: "Legacy check", Environment: map[string]string{"KEY": "val"}},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/environment",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "contains forbidden field",
				},
			},
		},

		// =================================================================
		// CHECKS - RUN MODE WITH REASON (invalid)
		// =================================================================

		{
			name:      "checks_run/with_reason",
			mutate:    func(p *Plan) { p.Checks[0].Reason = "This is a reason" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/reason",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "runnable check contains exclusion reason",
				},
			},
		},

		// =================================================================
		// CHECKS - DUPLICATE ID
		// =================================================================

		{
			name: "checks/duplicate_id",
			mutate: func(p *Plan) {
				p.Checks = []PlanCheck{
					{ID: "duplicate-check", Mode: CheckModeRun, Argv: []string{"echo", "test1"}, WorkingDirectory: ".", TimeoutSeconds: 300, Environment: map[string]string{}},
					{ID: "duplicate-check", Mode: CheckModeRun, Argv: []string{"echo", "test2"}, WorkingDirectory: ".", TimeoutSeconds: 300, Environment: map[string]string{}},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/1/id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "duplicate check id",
				},
			},
		},
	}
}
