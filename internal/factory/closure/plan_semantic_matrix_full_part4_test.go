package closure

import (
	"testing"
)

// TestValidatePlanSemanticMatrixFull_Part4 covers:
// - artifacts_0_id
// - artifacts_0_path
// - artifacts_0_required
// - artifacts_0_max_bytes
// - artifacts_0_media_type
// - artifacts_0_role
// - artifacts/duplicate_id
// - artifacts/exceeds_max
// - policy_require_clean_before
// - policy_require_clean_after
// - policy_require_diff_check
func TestValidatePlanSemanticMatrixFull_Part4(t *testing.T) {
	validateSourceCase(t, buildSourceMatrixCasesPart4())
}

// buildSourceMatrixCasesPart4 returns test cases for Part 4.
func buildSourceMatrixCasesPart4() []testCase {
	return []testCase{
		// =================================================================
		// ARTIFACTS - ID
		// =================================================================

		{
			name:      "artifacts_0_id/placeholder",
			mutate:    func(p *Plan) { p.Artifacts[0].ID = "TODO" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid",
				},
			},
		},
		{
			name:      "artifacts_0_id/empty",
			mutate:    func(p *Plan) { p.Artifacts[0].ID = "" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid",
				},
			},
		},
		{
			name:      "artifacts_0_id/invalid_chars",
			mutate:    func(p *Plan) { p.Artifacts[0].ID = "Invalid-ID" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid",
				},
			},
		},

		// =================================================================
		// ARTIFACTS - PATH
		// =================================================================

		{
			name:      "artifacts_0_path/empty",
			mutate:    func(p *Plan) { p.Artifacts[0].Path = "" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/path",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be a non-empty repository-relative path",
				},
			},
		},
		{
			name:      "artifacts_0_path/absolute",
			mutate:    func(p *Plan) { p.Artifacts[0].Path = "/absolute/path" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/path",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be a non-empty repository-relative path",
				},
			},
		},
		{
			name:      "artifacts_0_path/placeholder",
			mutate:    func(p *Plan) { p.Artifacts[0].Path = "TODO" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/path",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be a non-empty repository-relative path",
				},
			},
		},
		{
			name:      "artifacts_0_path/parent_escape",
			mutate:    func(p *Plan) { p.Artifacts[0].Path = "../escape" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/path",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be a non-empty repository-relative path",
				},
			},
		},

		// =================================================================
		// ARTIFACTS - REQUIRED
		// =================================================================

		{
			name:      "artifacts_0_required/missing",
			mutate:    func(p *Plan) { p.Artifacts[0].Required = nil },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/required",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordRequired,
					MessageSubstr: "is missing",
				},
			},
		},

		// =================================================================
		// ARTIFACTS - MAX_BYTES
		// =================================================================

		{
			name:      "artifacts_0_max_bytes/zero",
			mutate:    func(p *Plan) { p.Artifacts[0].MaxBytes = 0 },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/max_bytes",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be positive",
				},
			},
		},
		{
			name:      "artifacts_0_max_bytes/negative",
			mutate:    func(p *Plan) { p.Artifacts[0].MaxBytes = -1 },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/max_bytes",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "must be positive",
				},
			},
		},

		// =================================================================
		// ARTIFACTS - MEDIA_TYPE
		// =================================================================

		{
			name:      "artifacts_0_media_type/empty",
			mutate:    func(p *Plan) { p.Artifacts[0].MediaType = "" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/media_type",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid",
				},
			},
		},
		{
			name:      "artifacts_0_media_type/whitespace",
			mutate:    func(p *Plan) { p.Artifacts[0].MediaType = "   " },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/media_type",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid",
				},
			},
		},
		{
			name:      "artifacts_0_media_type/placeholder",
			mutate:    func(p *Plan) { p.Artifacts[0].MediaType = "TODO" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/media_type",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "is invalid",
				},
			},
		},

		// =================================================================
		// ARTIFACTS - ROLE
		// =================================================================

		{
			name:      "artifacts_0_role/invalid",
			mutate:    func(p *Plan) { p.Artifacts[0].Role = "invalid_role" },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/0/role",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordEnum,
					MessageSubstr: "is invalid",
				},
			},
		},

		// =================================================================
		// ARTIFACTS - DUPLICATE ID
		// =================================================================

		{
			name: "artifacts/duplicate_id",
			mutate: func(p *Plan) {
				trueVal := true
				p.Artifacts = []PlanArtifact{
					{ID: "duplicate-art", Path: "bin/a", Required: &trueVal, MaxBytes: 100, MediaType: "application/octet-stream"},
					{ID: "duplicate-art", Path: "bin/b", Required: &trueVal, MaxBytes: 100, MediaType: "application/octet-stream"},
				}
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts/1/id",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordType,
					MessageSubstr: "duplicate artifact id",
				},
			},
		},

		// =================================================================
		// ARTIFACTS - EXCEEDS MAX COUNT
		// =================================================================

		{
			name: "artifacts/exceeds_max",
			mutate: func(p *Plan) {
				trueVal := true
				artifacts := make([]PlanArtifact, MaxArtifacts+1)
				for i := range artifacts {
					artifacts[i] = PlanArtifact{
						ID:        "artifact",
						Path:      "path/to/file",
						Required:  &trueVal,
						MaxBytes:  100,
						MediaType: "application/octet-stream",
					}
				}
				p.Artifacts = artifacts
			},
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/artifacts",
					Code:          PlanCodeSemanticConstraintFailed,
					Keyword:       KeywordMinItems,
					MessageSubstr: "artifacts count exceeds",
				},
			},
		},

		// =================================================================
		// POLICY - REQUIRED FIELDS (uses PlanPolicyRequiredError)
		// =================================================================

		{
			name:      "policy_require_clean_before/missing",
			mutate:    func(p *Plan) { p.Policy.RequireCleanBefore = nil },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/policy/require_clean_before",
					Code:          PlanCodeRequiredPropertyMissing,
					Keyword:       KeywordRequired,
					PropertyName:  "require_clean_before",
					MessageSubstr: "missing required policy field",
				},
			},
		},
		{
			name:      "policy_require_clean_after/missing",
			mutate:    func(p *Plan) { p.Policy.RequireCleanAfter = nil },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/policy/require_clean_after",
					Code:          PlanCodeRequiredPropertyMissing,
					Keyword:       KeywordRequired,
					PropertyName:  "require_clean_after",
					MessageSubstr: "missing required policy field",
				},
			},
		},
		{
			name:      "policy_require_diff_check/missing",
			mutate:    func(p *Plan) { p.Policy.RequireDiffCheck = nil },
			wantCount: 1,
			wantDiags: []wantDiag{
				{
					InstancePath:  "/policy/require_diff_check",
					Code:          PlanCodeRequiredPropertyMissing,
					Keyword:       KeywordRequired,
					PropertyName:  "require_diff_check",
					MessageSubstr: "missing required policy field",
				},
			},
		},
	}
}

// buildPlainMatrixCases constructs test cases that return plain fmt.Errorf errors.
func buildPlainMatrixCases() []plainTestCase {
	return []plainTestCase{
		// =================================================================
		// POLICY_PROFILE - returns plain fmt.Errorf
		// =================================================================

		{
			name: "policy_profile/unknown",
			mutate: func(p *Plan) {
				p.PolicyProfile = "unknown-profile"
			},
			wantMessageSubstr: "is unknown",
		},
		{
			name: "policy_profile/not_implemented",
			mutate: func(p *Plan) {
				p.PolicyProfile = "indeep-act-v1"
			},
			wantMessageSubstr: "not yet implemented",
		},
		{
			name: "policy_profile/missing_check",
			mutate: func(p *Plan) {
				p.PolicyProfile = PolicyProfileLeamasActV1
				p.Checks = []PlanCheck{
					{ID: "compile", Mode: CheckModeRun, Argv: []string{"go", "build", "./..."}, WorkingDirectory: ".", TimeoutSeconds: 300, Environment: map[string]string{}},
				}
			},
			wantMessageSubstr: "missing or non-matching check",
		},
	}
}
