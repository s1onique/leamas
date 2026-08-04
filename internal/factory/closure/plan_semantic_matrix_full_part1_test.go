package closure

import (
	"testing"
)

// TestValidatePlanSemanticMatrixFull_Part1 covers:
// - contract_version
// - act_id
// - baseline_commit_oid
// - baseline_tree_oid
func TestValidatePlanSemanticMatrixFull_Part1(t *testing.T) {
	validateSourceCase(t, buildSourceMatrixCasesPart1())
}

// buildSourceMatrixCasesPart1 returns test cases for Part 1.
func buildSourceMatrixCasesPart1() []testCase {
	return []testCase{
		// =================================================================
		// CONTRACT VERSION
		// =================================================================

		{
			name:      "contract_version/unsupported_zero",
			mutate:    func(p *Plan) { p.ContractVersion = 0 },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/contract_version",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordConst,
					MessageSubstr: "unsupported closure plan contract_version",
				},
			},
		},
		{
			name:      "contract_version/unsupported_two",
			mutate:    func(p *Plan) { p.ContractVersion = 2 },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/contract_version",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordConst,
					MessageSubstr: "unsupported closure plan contract_version",
				},
			},
		},
		{
			name:      "contract_version/unsupported_negative",
			mutate:    func(p *Plan) { p.ContractVersion = -1 },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/contract_version",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordConst,
					MessageSubstr: "unsupported closure plan contract_version",
				},
			},
		},

		// =================================================================
		// ACT_ID
		// =================================================================

		{
			name:      "act_id/placeholder_todo",
			mutate:    func(p *Plan) { p.ActID = "TODO" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/act_id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "invalid act_id",
				},
			},
		},
		{
			name:      "act_id/placeholder_tbd",
			mutate:    func(p *Plan) { p.ActID = "TBD" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/act_id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "invalid act_id",
				},
			},
		},
		{
			name:      "act_id/placeholder_unknown",
			mutate:    func(p *Plan) { p.ActID = "UNKNOWN" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/act_id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "invalid act_id",
				},
			},
		},
		{
			name:      "act_id/empty",
			mutate:    func(p *Plan) { p.ActID = "" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/act_id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "invalid act_id",
				},
			},
		},
		{
			name:      "act_id/lowercase_prefix",
			mutate:    func(p *Plan) { p.ActID = "act-lowercase" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/act_id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "invalid act_id",
				},
			},
		},
		{
			name:      "act_id/too_short",
			mutate:    func(p *Plan) { p.ActID = "ACT-A" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/act_id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "invalid act_id",
				},
			},
		},
		{
			name:      "act_id/invalid_chars",
			mutate:    func(p *Plan) { p.ActID = "ACT_LOWERCASE" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/act_id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "invalid act_id",
				},
			},
		},

		// =================================================================
		// BASELINE COMMIT OID
		// =================================================================

		{
			name:      "baseline_commit_oid/placeholder",
			mutate:    func(p *Plan) { p.Baseline.CommitOID = "TODO" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/baseline/commit_oid",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "contains a closure placeholder",
				},
			},
		},
		{
			name:      "baseline_commit_oid/short",
			mutate:    func(p *Plan) { p.Baseline.CommitOID = "abc123" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/baseline/commit_oid",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be a full lowercase Git OID",
				},
			},
		},
		{
			name:      "baseline_commit_oid/invalid_chars",
			mutate:    func(p *Plan) { p.Baseline.CommitOID = "g1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/baseline/commit_oid",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be a full lowercase Git OID",
				},
			},
		},
		{
			name:      "baseline_commit_oid/uppercase",
			mutate:    func(p *Plan) { p.Baseline.CommitOID = "ABCDEF0123456789ABCDEF0123456789ABCDEF01" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/baseline/commit_oid",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be a full lowercase Git OID",
				},
			},
		},

		// =================================================================
		// BASELINE TREE OID
		// =================================================================

		{
			name:      "baseline_tree_oid/placeholder",
			mutate:    func(p *Plan) { p.Baseline.TreeOID = "TODO" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/baseline/tree_oid",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "contains a closure placeholder",
				},
			},
		},
		{
			name:      "baseline_tree_oid/short",
			mutate:    func(p *Plan) { p.Baseline.TreeOID = "abc123" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/baseline/tree_oid",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be a full lowercase Git OID",
				},
			},
		},
	}
}
