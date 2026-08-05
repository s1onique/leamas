// SPDX-License-Identifier: Apache-2.0

package closure

import "fmt"

const (
	v2CheckOutcomeExcluded        = "excluded"
	v2ExecutionCompleted          = "completed"
	v2ExecutionNonzeroExit        = "nonzero_exit"
	v2ExecutionOutputTruncated    = "output_truncated"
	v2ExecutionOutputIncomplete   = "output_incomplete"
	v2ExecutionCleanupFailed      = "cleanup_failed"
	v2ExecutionExcludedByPlan     = "excluded_by_plan"
	v2ExecutionNotRunPriorFailure = "not_run_due_to_prior_failure"
)

// buildV2ManifestCheckResults constructs exactly one manifest result per plan
// check, in plan order. Execution results are an unordered authority stream:
// every run check must occur exactly once and exclude checks must never occur.
func buildV2ManifestCheckResults(subjectTree string, plans []PlanCheck, executions []CheckResult, evidence []EvidenceRecord) ([]V2CheckResult, error) {
	if len(plans) == 0 {
		return nil, invalidV2CheckMapping("checks", "plan has no checks")
	}
	planByID := make(map[string]PlanCheck, len(plans))
	for _, plan := range plans {
		if !itemIDPattern.MatchString(plan.ID) {
			return nil, invalidV2CheckMapping("checks", fmt.Sprintf("invalid plan check ID %q", plan.ID))
		}
		if _, exists := planByID[plan.ID]; exists {
			return nil, invalidV2CheckMapping("checks", fmt.Sprintf("duplicate plan check ID %q", plan.ID))
		}
		if plan.Mode != CheckModeRun && plan.Mode != CheckModeExclude {
			return nil, invalidV2CheckMapping("checks", fmt.Sprintf("unknown mode %q for plan check %q", plan.Mode, plan.ID))
		}
		planByID[plan.ID] = plan
	}

	executionByID := make(map[string]CheckResult, len(executions))
	for _, result := range executions {
		plan, known := planByID[result.CheckID]
		if !known {
			return nil, invalidV2CheckMapping("check_results", fmt.Sprintf("unknown execution result ID %q", result.CheckID))
		}
		if plan.Mode != CheckModeRun {
			return nil, invalidV2CheckMapping("check_results", fmt.Sprintf("exclude check %q has an execution result", result.CheckID))
		}
		if _, exists := executionByID[result.CheckID]; exists {
			return nil, invalidV2CheckMapping("check_results", fmt.Sprintf("duplicate execution result ID %q", result.CheckID))
		}
		executionByID[result.CheckID] = result
	}

	evidenceByName := make(map[string]EvidenceRecord, len(evidence))
	for _, record := range evidence {
		if record.LogicalName == "" {
			return nil, invalidV2CheckMapping("evidence", "evidence logical name is empty")
		}
		if _, exists := evidenceByName[record.LogicalName]; exists {
			return nil, invalidV2CheckMapping("evidence", fmt.Sprintf("duplicate evidence %q", record.LogicalName))
		}
		if record.MediaType == "" || record.Availability == "" || record.ByteCount < 0 || !sha256Pattern.MatchString(record.SHA256) {
			return nil, invalidV2CheckMapping("evidence", fmt.Sprintf("evidence %q is incomplete", record.LogicalName))
		}
		evidenceByName[record.LogicalName] = record
	}

	mapped := make([]V2CheckResult, 0, len(plans))
	consumedEvidence := make(map[string]bool, len(evidence))
	for _, plan := range plans {
		if plan.Mode == CheckModeExclude {
			mapped = append(mapped, V2CheckResult{
				ID: plan.ID, Mode: plan.Mode, Outcome: v2CheckOutcomeExcluded,
				ExecutionClassification: v2ExecutionExcludedByPlan,
				CleanupStatus:           CleanupNotRequired, Detail: plan.Reason,
			})
			continue
		}
		executionResult, exists := executionByID[plan.ID]
		if !exists {
			return nil, invalidV2CheckMapping("check_results", fmt.Sprintf("missing execution result for run check %q", plan.ID))
		}
		result, names, err := mapV2RunResult(subjectTree, executionResult, evidenceByName)
		if err != nil {
			return nil, err
		}
		result.Mode = plan.Mode
		mapped = append(mapped, result)
		for _, name := range names {
			consumedEvidence[name] = true
		}
	}
	if len(consumedEvidence) != len(evidenceByName) {
		for name := range evidenceByName {
			if !consumedEvidence[name] {
				return nil, invalidV2CheckMapping("evidence", fmt.Sprintf("unknown evidence reference %q", name))
			}
		}
	}
	return mapped, nil
}

func mapV2RunResult(subjectTree string, in CheckResult, evidence map[string]EvidenceRecord) (V2CheckResult, []string, error) {
	if in.SubjectTreeOID != subjectTree {
		return V2CheckResult{}, nil, invalidV2CheckMapping("check_results", fmt.Sprintf("check %q subject tree does not match execution tree", in.CheckID))
	}
	if in.DurationMS < 0 {
		return V2CheckResult{}, nil, invalidV2CheckMapping("check_results", fmt.Sprintf("check %q has negative duration", in.CheckID))
	}
	out := V2CheckResult{
		ID: in.CheckID, Outcome: in.Status, DurationMS: in.DurationMS,
		CleanupStatus: in.CleanupStatus,
	}
	if in.ExitCode != nil {
		exit := *in.ExitCode
		out.ExitCode = &exit
	}
	switch in.Status {
	case CheckStatusNotRun:
		if in.ExitCode != nil || in.DurationMS != 0 || in.StdoutSHA256 != "" || in.StderrSHA256 != "" {
			return V2CheckResult{}, nil, invalidV2CheckMapping("check_results", fmt.Sprintf("not-run check %q carries execution evidence", in.CheckID))
		}
		out.ExecutionClassification = v2ExecutionNotRunPriorFailure
		if in.CleanupStatus != CleanupNotRequired {
			return V2CheckResult{}, nil, invalidV2CheckMapping("check_results", fmt.Sprintf("not-run check %q has invalid cleanup status", in.CheckID))
		}
		return out, nil, nil
	case CheckStatusPass, CheckStatusFail:
		if in.ExitCode == nil {
			return V2CheckResult{}, nil, invalidV2CheckMapping("check_results", fmt.Sprintf("executed check %q has no exit code", in.CheckID))
		}
	default:
		return V2CheckResult{}, nil, invalidV2CheckMapping("check_results", fmt.Sprintf("check %q has unknown outcome %q", in.CheckID, in.Status))
	}

	refs := make([]V2EvidenceReference, 0, 2)
	names := []string{in.CheckID + ".stdout", in.CheckID + ".stderr"}
	wantSHA := []string{in.StdoutSHA256, in.StderrSHA256}
	wantBytes := []int64{in.StdoutByteCount, in.StderrByteCount}
	for i, name := range names {
		record, exists := evidence[name]
		if !exists || record.SHA256 != wantSHA[i] || record.ByteCount != wantBytes[i] {
			return V2CheckResult{}, nil, invalidV2CheckMapping("evidence", fmt.Sprintf("evidence %q does not bind check result", name))
		}
		refs = append(refs, V2EvidenceReference(record))
	}
	out.Evidence = refs
	out.ExecutionClassification = classifyV2Execution(in)
	return out, names, nil
}

func classifyV2Execution(result CheckResult) string {
	switch {
	case result.ExecutionErrorCode != "":
		return result.ExecutionErrorCode
	case result.CleanupStatus == CleanupFailed:
		return v2ExecutionCleanupFailed
	case result.OutputIncomplete:
		return v2ExecutionOutputIncomplete
	case result.OutputTruncated:
		return v2ExecutionOutputTruncated
	case result.Status == CheckStatusFail && result.ExitCode != nil && *result.ExitCode != 0:
		return v2ExecutionNonzeroExit
	default:
		return v2ExecutionCompleted
	}
}

func invalidV2CheckMapping(property, message string) error {
	return NewV2ErrorWith(V2CodeCheckResultMappingInvalid, message, property, "")
}
