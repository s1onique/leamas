package closure

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestClosureVerifyAndRenderThousandChecksWithinBound(t *testing.T) {
	plan := Plan{
		ContractVersion: ContractVersionV1,
		ActID:           "ACT-LEAMAS-PERF",
		Baseline:        Baseline{CommitOID: fullTreeOID, TreeOID: fullCommitOID},
		Execution:       PlanExecution{Mode: ExecutionSerialFailFast},
		Checks:          make([]PlanCheck, 1000),
		Artifacts:       nil,
		Policy:          PlanPolicy{RequireCleanBefore: boolPtr(true), RequireCleanAfter: boolPtr(true), ForbidTrackedFullDigests: boolPtr(true), RequireDiffCheck: boolPtr(true)},
	}
	manifest := passingManifest()
	manifest.ActID = plan.ActID
	manifest.Plan.Path = "docs/closure-plans/ACT-LEAMAS-PERF.json"
	manifest.Subject.CommitOID = plan.Baseline.CommitOID
	manifest.Subject.TreeOID = plan.Baseline.TreeOID
	manifest.Repository.HeadCommitOID = plan.Baseline.CommitOID
	manifest.Repository.HeadTreeOID = plan.Baseline.TreeOID
	manifest.Artifacts = nil
	manifest.ExcludedChecks = nil
	manifest.Checks = make([]CheckResult, 1000)
	manifest.DetachedEvidence = make([]EvidenceRecord, 0, 2001)
	for index := range 1000 {
		id := fmt.Sprintf("check-%04d", index)
		argv := []string{"go", "test", fmt.Sprintf("./case/%04d", index)}
		plan.Checks[index] = PlanCheck{ID: id, Mode: CheckModeRun, Argv: argv, WorkingDirectory: ".", TimeoutSeconds: 60, Environment: map[string]string{}}
		manifest.Checks[index] = CheckResult{
			CheckID: id, SubjectTreeOID: plan.Baseline.TreeOID, Argv: argv, WorkingDirectory: ".", OverriddenEnvironment: []string{},
			StartedAtUTC: "2026-07-23T07:00:00Z", FinishedAtUTC: "2026-07-23T07:00:00.001Z", DurationMS: 1,
			ExitCode: intPtr(0), Status: CheckStatusPass, StdoutSHA256: strings.Repeat("a", 64), StderrSHA256: strings.Repeat("b", 64), CleanupStatus: CleanupPass,
		}
		manifest.DetachedEvidence = append(manifest.DetachedEvidence,
			EvidenceRecord{LogicalName: id + ".stdout", MediaType: "text/plain; charset=utf-8", SHA256: strings.Repeat("a", 64), Availability: "detached"},
			EvidenceRecord{LogicalName: id + ".stderr", MediaType: "text/plain; charset=utf-8", SHA256: strings.Repeat("b", 64), Availability: "detached"},
		)
	}
	manifest.DetachedEvidence = append(manifest.DetachedEvidence,
		EvidenceRecord{LogicalName: "runner.diagnostics", MediaType: "application/json", SHA256: strings.Repeat("c", 64), Availability: "detached"})

	started := time.Now()
	if err := VerifyManifestAgainstPlan(manifest, plan); err != nil {
		t.Fatal(err)
	}
	report, err := Render(manifest, plan)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("verification and rendering took %s, limit 2s", elapsed)
	}
	if len(report) > MaxReportBytes {
		t.Fatalf("report bytes = %d", len(report))
	}
}
