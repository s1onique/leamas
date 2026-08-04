package closure

import (
	"testing"
)

// validSemanticPlanFixture returns a valid Plan that passes ValidatePlan.
// This is the canonical fixture for semantic path matrix tests.
func validSemanticPlanFixture(t *testing.T) Plan {
	t.Helper()
	trueVal := true
	falseVal := false
	return Plan{
		ContractVersion: ContractVersionV1,
		ActID:           "ACT-LEAMAS-FACTORY-CLOSURE01",
		Baseline: Baseline{
			CommitOID: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b3", // 40 chars
			TreeOID:   "b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b3c3", // 40 chars
		},
		Execution: PlanExecution{
			Mode: func() *ExecutionMode { m := ExecutionModeSerialFailFast; return &m }(),
		},
		Checks: []PlanCheck{
			{
				ID:               "compile",
				Mode:             CheckModeRun,
				Argv:             []string{"go", "build", "./..."},
				WorkingDirectory: ".",
				TimeoutSeconds:   300,
				Environment:      map[string]string{"CGO_ENABLED": "0"},
			},
			{
				ID:     "legacy-check",
				Mode:   CheckModeExclude,
				Reason: "Legacy check superseded by compile.",
			},
		},
		Artifacts: []PlanArtifact{
			{
				ID:        "bin-leamas",
				Path:      "bin/leamas",
				Required:  &trueVal,
				MaxBytes:  100_000_000,
				MediaType: "application/octet-stream",
				Role:      ArtifactRoleGeneratedOutput,
			},
		},
		Policy: PlanPolicy{
			RequireCleanBefore:       &trueVal,
			RequireCleanAfter:        &trueVal,
			ForbidTrackedFullDigests: &falseVal,
			RequireDiffCheck:         &trueVal,
		},
		PolicyProfile: "factory",
		RunnerBinding: "leamas",
		RunnerAuthority: &RunnerAuthority{
			Mode: RunnerAuthoritySubjectExact,
			Tool: nil, // Not required for subject_exact
		},
	}
}
