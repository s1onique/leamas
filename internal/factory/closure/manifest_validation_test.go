package closure

import (
	"encoding/json"
	"strings"
	"testing"
)

func boolPtr(value bool) *bool { return &value }
func intPtr(value int) *int    { return &value }

func canonicalPlan() Plan {
	return Plan{
		ContractVersion: ContractVersionV1,
		ActID:           "ACT-LEAMAS-TEST01",
		Baseline:        Baseline{CommitOID: fullCommitOID, TreeOID: fullTreeOID},
		Execution:       PlanExecution{Mode: ExecutionSerialFailFast},
		Checks: []PlanCheck{
			{ID: "focused", Mode: CheckModeRun, Argv: []string{"go", "test", "./..."}, WorkingDirectory: ".", TimeoutSeconds: 60, Environment: map[string]string{}},
			{ID: "dupcode", Mode: CheckModeExclude, Reason: "No dupcode-owned source changed."},
		},
		Artifacts: []PlanArtifact{{ID: "summary", Path: ".factory/summary.json", Required: boolPtr(true), MaxBytes: 1024, MediaType: "application/json"}},
		Policy:    PlanPolicy{RequireCleanBefore: boolPtr(true), RequireCleanAfter: boolPtr(true), ForbidTrackedFullDigests: boolPtr(true), RequireDiffCheck: boolPtr(true)},
	}
}

func passingManifest() Manifest {
	return Manifest{
		ContractVersion: ContractVersionV1,
		ActID:           "ACT-LEAMAS-TEST01",
		Plan:            ManifestPlanRef{SHA256: strings.Repeat("a", 64), Path: "docs/closure-plans/ACT-LEAMAS-TEST01.json"},
		Subject:         ManifestSubject{CommitOID: fullCommitOID, TreeOID: fullTreeOID},
		Runner:          RunnerIdentity{LeamasVersion: "0.1.0", BinarySHA256: strings.Repeat("b", 64), VCSRevision: fullCommitOID, VCSModified: false},
		Repository:      RepositoryIdentity{Root: ".", Branch: "main", HeadCommitOID: fullCommitOID, HeadTreeOID: fullTreeOID, WorkingTreeCleanBefore: true, WorkingTreeCleanAfter: true},
		Checks: []CheckResult{{
			CheckID: "focused", SubjectTreeOID: fullTreeOID, Argv: []string{"go", "test", "./..."}, WorkingDirectory: ".",
			OverriddenEnvironment: []string{}, StartedAtUTC: "2026-07-23T07:00:00Z", FinishedAtUTC: "2026-07-23T07:00:01Z",
			DurationMS: 1000, ExitCode: intPtr(0), Status: CheckStatusPass,
			StdoutSHA256: strings.Repeat("c", 64), StderrSHA256: strings.Repeat("d", 64), CleanupStatus: CleanupPass,
		}},
		Artifacts: []ArtifactResult{{ArtifactID: "summary", Path: ".factory/summary.json", Required: true, MediaType: "application/json", Status: ArtifactStatusPass, SHA256: strings.Repeat("e", 64), ByteCount: 10}},
		DetachedEvidence: []EvidenceRecord{
			{LogicalName: "focused.stdout", MediaType: "text/plain; charset=utf-8", SHA256: strings.Repeat("c", 64), Availability: "detached"},
			{LogicalName: "focused.stderr", MediaType: "text/plain; charset=utf-8", SHA256: strings.Repeat("d", 64), Availability: "detached"},
			{LogicalName: "runner.diagnostics", MediaType: "application/json", SHA256: strings.Repeat("f", 64), Availability: "detached"},
		},
		PatchHygiene:   PatchHygiene{Status: CheckStatusPass},
		ClosurePolicy:  ClosurePolicyResult{TrackedFullDigestStatus: CheckStatusPass},
		ExcludedChecks: []ExcludedCheck{{CheckID: "dupcode", SubjectTreeOID: fullTreeOID, Reason: "No dupcode-owned source changed."}},
		Verdict:        VerdictPass,
	}
}

func TestClosureManifestContainsEveryPlanCheckExactlyOnce(t *testing.T) {
	if err := VerifyManifestAgainstPlan(passingManifest(), canonicalPlan()); err != nil {
		t.Fatalf("VerifyManifestAgainstPlan() error = %v", err)
	}
}

func TestClosureManifestPreservesPlanOrder(t *testing.T) {
	plan := canonicalPlan()
	plan.Checks = append([]PlanCheck{
		{ID: "first", Mode: CheckModeRun, Argv: []string{"true"}, WorkingDirectory: ".", TimeoutSeconds: 1, Environment: map[string]string{}},
	}, plan.Checks...)
	manifest := passingManifest()
	first := manifest.Checks[0]
	first.CheckID = "first"
	first.Argv = []string{"true"}
	manifest.Checks = append([]CheckResult{first}, manifest.Checks...)
	manifest.Checks[0], manifest.Checks[1] = manifest.Checks[1], manifest.Checks[0]
	if err := VerifyManifestAgainstPlan(manifest, plan); err == nil || !strings.Contains(err.Error(), "order") {
		t.Fatalf("error = %v, want order failure", err)
	}
}

func TestClosureManifestRejectsMissingCheck(t *testing.T) {
	manifest := passingManifest()
	manifest.Checks = nil
	assertManifestError(t, manifest, "missing")
}

func TestClosureManifestRejectsDuplicateCheck(t *testing.T) {
	manifest := passingManifest()
	manifest.Checks = append(manifest.Checks, manifest.Checks[0])
	assertManifestError(t, manifest, "duplicate")
}

func TestClosureManifestRejectsUserSuppliedVerdictMismatch(t *testing.T) {
	manifest := passingManifest()
	manifest.Verdict = VerdictFail
	assertManifestError(t, manifest, "verdict")
}

func TestClosureVerdictPass(t *testing.T) {
	got, err := DeriveVerdict(passingManifest(), canonicalPlan())
	if err != nil || got != VerdictPass {
		t.Fatalf("DeriveVerdict() = %q, %v", got, err)
	}
}

func TestClosureVerdictFailsOnRequiredCheck(t *testing.T) {
	manifest := passingManifest()
	manifest.Checks[0].Status = CheckStatusFail
	manifest.Checks[0].ExitCode = intPtr(1)
	assertDerivedFail(t, manifest)
}

func TestClosureVerdictFailsOnMissingArtifact(t *testing.T) {
	manifest := passingManifest()
	manifest.Artifacts[0].Status = ArtifactStatusMissing
	manifest.Artifacts[0].SHA256 = ""
	manifest.Artifacts[0].ByteCount = 0
	manifest.Artifacts[0].Diagnostic = "required artifact is missing"
	assertDerivedFail(t, manifest)
}

func TestClosureVerdictFailsOnPatchHygiene(t *testing.T) {
	manifest := passingManifest()
	manifest.PatchHygiene = PatchHygiene{Status: CheckStatusFail, DiagnosticCount: 1}
	assertDerivedFail(t, manifest)
}

func TestClosureVerdictFailsOnCleanupFailure(t *testing.T) {
	manifest := passingManifest()
	manifest.Checks[0].CleanupStatus = CleanupFailed
	assertDerivedFail(t, manifest)
}

func TestClosureManifestContainsNoAbsolutePaths(t *testing.T) {
	manifest := passingManifest()
	manifest.Artifacts[0].Path = "/tmp/summary.json"
	assertManifestError(t, manifest, "path")
}

func TestClosureManifestContainsNoRawOutput(t *testing.T) {
	data, err := json.Marshal(passingManifest())
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"stdout_sha256":`, `"stdout":"secret","stdout_sha256":`, 1))
	if _, err := DecodeManifest(data); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("DecodeManifest() error = %v", err)
	}
}

func TestClosureManifestContainsNoFutureIdentityFields(t *testing.T) {
	data, err := json.Marshal(passingManifest())
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"verdict":`, `"closure_commit_oid":"`+fullCommitOID+`","verdict":`, 1))
	if _, err := DecodeManifest(data); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("DecodeManifest() error = %v", err)
	}
}

func assertManifestError(t *testing.T, manifest Manifest, want string) {
	t.Helper()
	err := VerifyManifestAgainstPlan(manifest, canonicalPlan())
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(want)) {
		t.Fatalf("error = %v, want containing %q", err, want)
	}
}

func assertDerivedFail(t *testing.T, manifest Manifest) {
	t.Helper()
	manifest.Verdict = VerdictFail
	got, err := DeriveVerdict(manifest, canonicalPlan())
	if err != nil || got != VerdictFail {
		t.Fatalf("DeriveVerdict() = %q, %v", got, err)
	}
}
