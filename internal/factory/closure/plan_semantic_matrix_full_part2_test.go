package closure

import (
	"testing"
)

// TestValidatePlanSemanticMatrixFull_Part2 covers:
// - execution_mode
// - checks/empty, checks/zero_length, checks/exceeds_max
// - checks_0_id/*
// - checks_0_mode/*
// - checks_0_argv/*
// - checks_0_working_directory/*
func TestValidatePlanSemanticMatrixFull_Part2(t *testing.T) {
	validateSourceCase(t, buildSourceMatrixCasesPart2())
}

// buildSourceMatrixCasesPart2 returns test cases for Part 2.
func buildSourceMatrixCasesPart2() []testCase {
	return []testCase{
		// =================================================================
		// EXECUTION MODE
		// =================================================================

		{
			name:      "execution_mode/missing",
			mutate:    func(p *Plan) { p.Execution.Mode = nil },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/execution/mode",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordEnum,
					MessageSubstr: "is required and must be one of",
				},
			},
		},
		{
			name:      "execution_mode/empty",
			mutate:    func(p *Plan) { empty := ExecutionMode(""); p.Execution.Mode = &empty },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/execution/mode",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordEnum,
					MessageSubstr: "is empty",
				},
			},
		},
		{
			name:      "execution_mode/whitespace",
			mutate:    func(p *Plan) { ws := ExecutionMode("   "); p.Execution.Mode = &ws },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/execution/mode",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordEnum,
					MessageSubstr: "is whitespace-only",
				},
			},
		},
		{
			name:      "execution_mode/unknown",
			mutate:    func(p *Plan) { unknown := ExecutionMode("unknown_mode"); p.Execution.Mode = &unknown },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/execution/mode",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordEnum,
					MessageSubstr: "is not a supported execution mode",
				},
			},
		},
		{
			name:      "execution_mode/invalid_value",
			mutate:    func(p *Plan) { inv := ExecutionMode("parallel"); p.Execution.Mode = &inv },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/execution/mode",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordEnum,
					MessageSubstr: "is not a supported execution mode",
				},
			},
		},

		// =================================================================
		// CHECKS - COUNT
		// =================================================================

		{
			name:      "checks/empty",
			mutate:    func(p *Plan) { p.Checks = nil },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordMinItems,
					MessageSubstr: "must be between 1 and",
				},
			},
		},
		{
			name:      "checks/zero_length",
			mutate:    func(p *Plan) { p.Checks = []PlanCheck{} },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordMinItems,
					MessageSubstr: "must be between 1 and",
				},
			},
		},
		{
			name: "checks/exceeds_max",
			mutate: func(p *Plan) {
				checks := make([]PlanCheck, MaxChecks+1)
				for i := range checks {
					checks[i] = PlanCheck{ID: "check", Mode: CheckModeRun, Argv: []string{"echo"}, WorkingDirectory: ".", TimeoutSeconds: 300}
				}
				p.Checks = checks
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordMinItems,
					MessageSubstr: "must be between 1 and",
				},
			},
		},

		// =================================================================
		// CHECKS - ID
		// =================================================================

		{
			name:      "checks_0_id/placeholder",
			mutate:    func(p *Plan) { p.Checks[0].ID = "TODO" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid",
				},
			},
		},
		{
			name:      "checks_0_id/empty",
			mutate:    func(p *Plan) { p.Checks[0].ID = "" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid",
				},
			},
		},
		{
			name:      "checks_0_id/invalid_chars",
			mutate:    func(p *Plan) { p.Checks[0].ID = "Invalid-ID" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid",
				},
			},
		},
		{
			name:      "checks_0_id/starts_with_hyphen",
			mutate:    func(p *Plan) { p.Checks[0].ID = "-invalid" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid",
				},
			},
		},

		// =================================================================
		// CHECKS - MODE
		// =================================================================

		{
			name:      "checks_0_mode/missing",
			mutate:    func(p *Plan) { p.Checks[0].Mode = "" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/mode",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordEnum,
					MessageSubstr: "has unknown mode",
				},
			},
		},
		{
			name:      "checks_0_mode/invalid",
			mutate:    func(p *Plan) { p.Checks[0].Mode = "unknown_mode" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/mode",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordEnum,
					MessageSubstr: "has unknown mode",
				},
			},
		},

		// =================================================================
		// CHECKS - ARGV (run mode)
		// =================================================================

		{
			name:      "checks_0_argv/empty",
			mutate:    func(p *Plan) { p.Checks[0].Argv = nil },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/argv",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordMinItems,
					MessageSubstr: "must be between 1 and",
				},
			},
		},
		{
			name:      "checks_0_argv/zero_length",
			mutate:    func(p *Plan) { p.Checks[0].Argv = []string{} },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/argv",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordMinItems,
					MessageSubstr: "must be between 1 and",
				},
			},
		},
		{
			name:      "checks_0_argv/placeholder_element",
			mutate:    func(p *Plan) { p.Checks[0].Argv[0] = "TODO" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/argv/0",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid or contains a placeholder",
				},
			},
		},
		{
			name:      "checks_0_argv/empty_element",
			mutate:    func(p *Plan) { p.Checks[0].Argv[0] = "" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/argv/0",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid or contains a placeholder",
				},
			},
		},
		{
			name:      "checks_0_argv/null_byte",
			mutate:    func(p *Plan) { p.Checks[0].Argv[0] = "go\x00build" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/argv/0",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid or contains a placeholder",
				},
			},
		},

		// =================================================================
		// CHECKS - WORKING DIRECTORY (run mode)
		// =================================================================

		{
			name:      "checks_0_working_directory/empty",
			mutate:    func(p *Plan) { p.Checks[0].WorkingDirectory = "" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/working_directory",
					Code:          PlanCodePathPolicyViolation,
					Keyword:       KeywordPathPolicy,
					MessageSubstr: "must be a non-empty repository-relative path",
				},
			},
		},
		{
			name:      "checks_0_working_directory/absolute",
			mutate:    func(p *Plan) { p.Checks[0].WorkingDirectory = "/absolute/path" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/working_directory",
					Code:          PlanCodePathPolicyViolation,
					Keyword:       KeywordPathPolicy,
					MessageSubstr: "must be a non-empty repository-relative path",
				},
			},
		},
		{
			name:      "checks_0_working_directory/parent_escape",
			mutate:    func(p *Plan) { p.Checks[0].WorkingDirectory = "../escape" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/working_directory",
					Code:          PlanCodePathPolicyViolation,
					Keyword:       KeywordPathPolicy,
					MessageSubstr: "must not escape the repository",
				},
			},
		},
		{
			name:      "checks_0_working_directory/placeholder",
			mutate:    func(p *Plan) { p.Checks[0].WorkingDirectory = "TODO" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/checks/0/working_directory",
					Code:          PlanCodePathPolicyViolation,
					Keyword:       KeywordPathPolicy,
					MessageSubstr: "must be a non-empty repository-relative path",
				},
			},
		},
	}
}
