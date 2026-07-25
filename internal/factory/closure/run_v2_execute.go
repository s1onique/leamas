// SPDX-License-Identifier: Apache-2.0

package closure

import "context"

// v2ExecuteChecks executes the frozen checks.
func v2ExecuteChecks(ctx context.Context, plan Plan, deps v2Dependencies, repoRoot, evidenceDir, subjectTree string) ([]CheckResult, []EvidenceRecord, error) {
	runnableChecks := make([]PlanCheck, 0)
	for _, check := range plan.Checks {
		if check.Mode == CheckModeRun {
			runnableChecks = append(runnableChecks, check)
		}
	}
	return executeChecks(ctx, checkExecutionRequest{
		RepositoryRoot: repoRoot, EvidenceDirectory: evidenceDir,
		SubjectTreeOID: subjectTree, Checks: runnableChecks, Now: deps.Now,
	}, deps.Commands)
}
