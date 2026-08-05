// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"strings"
	"testing"
)

func TestV2CheckResultMappingIsPlanOrderedAndModePreserving(t *testing.T) {
	tree := strings.Repeat("2", 40)
	plans := []PlanCheck{
		{ID: "second-run", Mode: CheckModeRun},
		{ID: "documented-exclusion", Mode: CheckModeExclude, Reason: "not applicable"},
		{ID: "first-run", Mode: CheckModeRun},
	}
	results := []CheckResult{
		completedV2ExecutionResult("first-run", tree, 0, 17),
		completedV2ExecutionResult("second-run", tree, 0, 9),
	}
	evidence := append(v2ResultEvidence("first-run"), v2ResultEvidence("second-run")...)

	mapped, err := buildV2ManifestCheckResults(tree, plans, results, evidence)
	if err != nil {
		t.Fatalf("buildV2ManifestCheckResults: %v", err)
	}
	if len(mapped) != 3 {
		t.Fatalf("mapped length=%d, want 3", len(mapped))
	}
	wantIDs := []string{"second-run", "documented-exclusion", "first-run"}
	wantModes := []string{CheckModeRun, CheckModeExclude, CheckModeRun}
	for i := range mapped {
		if mapped[i].ID != wantIDs[i] || mapped[i].Mode != wantModes[i] {
			t.Fatalf("mapped[%d]=%+v, want id=%s mode=%s", i, mapped[i], wantIDs[i], wantModes[i])
		}
	}
	if mapped[0].Outcome != CheckStatusPass || mapped[0].ExitCode == nil || *mapped[0].ExitCode != 0 || mapped[0].DurationMS != 9 {
		t.Fatalf("run result fields not preserved: %+v", mapped[0])
	}
	if mapped[0].ExecutionClassification != "completed" {
		t.Fatalf("execution classification=%q, want completed", mapped[0].ExecutionClassification)
	}
	if len(mapped[0].Evidence) != 2 || mapped[0].Evidence[0].LogicalName != "second-run.stdout" || mapped[0].Evidence[1].LogicalName != "second-run.stderr" {
		t.Fatalf("evidence references not preserved in stream order: %+v", mapped[0].Evidence)
	}
	excluded := mapped[1]
	if excluded.Outcome != "excluded" || excluded.ExitCode != nil || excluded.DurationMS != 0 ||
		excluded.ExecutionClassification != "excluded_by_plan" || excluded.Detail != "not applicable" || len(excluded.Evidence) != 0 {
		t.Fatalf("exclude result invalid: %+v", excluded)
	}
}

func TestV2CheckResultMappingRejectsNonBijections(t *testing.T) {
	tree := strings.Repeat("2", 40)
	basePlans := []PlanCheck{{ID: "run", Mode: CheckModeRun}, {ID: "exclude", Mode: CheckModeExclude, Reason: "n/a"}}
	baseResults := []CheckResult{completedV2ExecutionResult("run", tree, 0, 1)}
	baseEvidence := v2ResultEvidence("run")
	cases := []struct {
		name     string
		plans    []PlanCheck
		results  []CheckResult
		evidence []EvidenceRecord
	}{
		{
			name:     "duplicate plan ID",
			plans:    []PlanCheck{{ID: "run", Mode: CheckModeRun}, {ID: "run", Mode: CheckModeExclude, Reason: "n/a"}},
			results:  baseResults,
			evidence: baseEvidence,
		},
		{
			name:  "unknown plan mode",
			plans: []PlanCheck{{ID: "run", Mode: "mystery"}},
		},
		{
			name:     "unknown execution result ID",
			plans:    basePlans,
			results:  []CheckResult{completedV2ExecutionResult("other", tree, 0, 1)},
			evidence: v2ResultEvidence("other"),
		},
		{
			name:  "duplicate result ID",
			plans: basePlans,
			results: []CheckResult{
				completedV2ExecutionResult("run", tree, 0, 1),
				completedV2ExecutionResult("run", tree, 0, 2),
			},
			evidence: baseEvidence,
		},
		{
			name:  "missing run result",
			plans: basePlans,
		},
		{
			name:  "invalid exclude result",
			plans: basePlans,
			results: []CheckResult{
				completedV2ExecutionResult("run", tree, 0, 1),
				completedV2ExecutionResult("exclude", tree, 0, 1),
			},
			evidence: append(v2ResultEvidence("run"), v2ResultEvidence("exclude")...),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mapped, err := buildV2ManifestCheckResults(tree, tc.plans, tc.results, tc.evidence)
			if len(mapped) != 0 {
				t.Fatalf("mapping failure returned partial results: %+v", mapped)
			}
			assertV2ManifestCode(t, err, V2CodeCheckResultMappingInvalid)
		})
	}
}

func TestV2CheckResultMappingRejectsInvalidExecutionAuthority(t *testing.T) {
	tree := strings.Repeat("2", 40)
	plans := []PlanCheck{{ID: "run", Mode: CheckModeRun}}
	cases := []struct {
		name     string
		mutate   func(*CheckResult)
		evidence []EvidenceRecord
	}{
		{name: "wrong subject tree", mutate: func(r *CheckResult) { r.SubjectTreeOID = strings.Repeat("9", 40) }, evidence: v2ResultEvidence("run")},
		{name: "negative duration", mutate: func(r *CheckResult) { r.DurationMS = -1 }, evidence: v2ResultEvidence("run")},
		{name: "unknown outcome", mutate: func(r *CheckResult) { r.Status = "unknown" }, evidence: v2ResultEvidence("run")},
		{name: "executed result missing exit", mutate: func(r *CheckResult) { r.ExitCode = nil }, evidence: v2ResultEvidence("run")},
		{name: "missing evidence", mutate: func(*CheckResult) {}, evidence: nil},
		{name: "unknown evidence", mutate: func(*CheckResult) {}, evidence: append(v2ResultEvidence("run"), EvidenceRecord{LogicalName: "other.stdout"})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := completedV2ExecutionResult("run", tree, 0, 1)
			tc.mutate(&result)
			_, err := buildV2ManifestCheckResults(tree, plans, []CheckResult{result}, tc.evidence)
			assertV2ManifestCode(t, err, V2CodeCheckResultMappingInvalid)
		})
	}
}

func completedV2ExecutionResult(id, tree string, exit, duration int) CheckResult {
	return CheckResult{
		CheckID: id, SubjectTreeOID: tree, DurationMS: int64(duration), ExitCode: &exit,
		Status: CheckStatusPass, CleanupStatus: CleanupPass,
		StdoutSHA256: strings.Repeat("a", 64), StderrSHA256: strings.Repeat("b", 64),
		StdoutByteCount: 3, StderrByteCount: 4, OutputBytesObserved: 7,
	}
}

func v2ResultEvidence(id string) []EvidenceRecord {
	return []EvidenceRecord{
		{LogicalName: id + ".stdout", MediaType: "text/plain; charset=utf-8", SHA256: strings.Repeat("a", 64), ByteCount: 3, Availability: "detached"},
		{LogicalName: id + ".stderr", MediaType: "text/plain; charset=utf-8", SHA256: strings.Repeat("b", 64), ByteCount: 4, Availability: "detached"},
	}
}
