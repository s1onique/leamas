package closure

import (
	"fmt"
	"reflect"
	"sort"
)

func VerifyManifestAgainstPlan(manifest Manifest, plan Plan) error {
	if err := ValidatePlan(plan); err != nil {
		return fmt.Errorf("invalid referenced plan: %w", err)
	}
	if err := validateManifestStructure(manifest); err != nil {
		return err
	}
	if manifest.ActID != plan.ActID {
		return fmt.Errorf("manifest act_id does not match plan")
	}
	if err := verifySubjectBinding(manifest); err != nil {
		return err
	}
	if err := verifyCheckMatrix(manifest, plan); err != nil {
		return err
	}
	if err := verifyArtifactMatrix(manifest, plan); err != nil {
		return err
	}
	if err := verifyDetachedEvidence(manifest); err != nil {
		return err
	}
	derived, err := DeriveVerdict(manifest, plan)
	if err != nil {
		return err
	}
	if manifest.Verdict != derived {
		return fmt.Errorf("manifest verdict %q does not match mechanically derived verdict %q", manifest.Verdict, derived)
	}
	return nil
}

func verifySubjectBinding(manifest Manifest) error {
	if manifest.Repository.HeadCommitOID != manifest.Subject.CommitOID {
		return fmt.Errorf("repository HEAD commit does not bind the subject")
	}
	if manifest.Repository.HeadTreeOID != manifest.Subject.TreeOID {
		return fmt.Errorf("repository HEAD tree does not bind the subject")
	}
	return nil
}

func verifyCheckMatrix(manifest Manifest, plan Plan) error {
	seen := make(map[string]struct{}, len(plan.Checks))
	runIndex, excludedIndex := 0, 0
	for planIndex, check := range plan.Checks {
		if _, exists := seen[check.ID]; exists {
			return fmt.Errorf("duplicate check %q", check.ID)
		}
		seen[check.ID] = struct{}{}
		switch check.Mode {
		case CheckModeRun:
			if runIndex >= len(manifest.Checks) {
				return fmt.Errorf("missing check %q", check.ID)
			}
			result := manifest.Checks[runIndex]
			if result.CheckID != check.ID {
				return fmt.Errorf("check order mismatch at plan index %d: got %q, want %q", planIndex, result.CheckID, check.ID)
			}
			if err := verifyRunResult(check, result, manifest.Subject.TreeOID); err != nil {
				return err
			}
			runIndex++
		case CheckModeExclude:
			if excludedIndex >= len(manifest.ExcludedChecks) {
				return fmt.Errorf("missing excluded check %q", check.ID)
			}
			result := manifest.ExcludedChecks[excludedIndex]
			if result.CheckID != check.ID {
				return fmt.Errorf("excluded check order mismatch at plan index %d", planIndex)
			}
			if result.Reason != check.Reason || result.SubjectTreeOID != manifest.Subject.TreeOID {
				return fmt.Errorf("excluded check %q does not match plan or subject", check.ID)
			}
			excludedIndex++
		}
	}
	if runIndex != len(manifest.Checks) {
		return fmt.Errorf("duplicate or unexpected runnable check at manifest index %d", runIndex)
	}
	if excludedIndex != len(manifest.ExcludedChecks) {
		return fmt.Errorf("duplicate or unexpected excluded check at manifest index %d", excludedIndex)
	}
	return nil
}

func verifyRunResult(check PlanCheck, result CheckResult, subjectTree string) error {
	if result.SubjectTreeOID != subjectTree {
		return fmt.Errorf("check %q subject tree binding mismatch", check.ID)
	}
	if !reflect.DeepEqual(result.Argv, check.Argv) || result.WorkingDirectory != check.WorkingDirectory {
		return fmt.Errorf("check %q command does not match frozen plan", check.ID)
	}
	expectedEnvironment := make([]string, 0, len(check.Environment))
	for name := range check.Environment {
		expectedEnvironment = append(expectedEnvironment, name)
	}
	sort.Strings(expectedEnvironment)
	if !reflect.DeepEqual(result.OverriddenEnvironment, expectedEnvironment) {
		return fmt.Errorf("check %q overridden environment names do not match plan", check.ID)
	}
	return nil
}

func verifyArtifactMatrix(manifest Manifest, plan Plan) error {
	if len(manifest.Artifacts) != len(plan.Artifacts) {
		return fmt.Errorf("artifact count mismatch: got %d, want %d", len(manifest.Artifacts), len(plan.Artifacts))
	}
	for i, planned := range plan.Artifacts {
		actual := manifest.Artifacts[i]
		if actual.ArtifactID != planned.ID || actual.Path != planned.Path ||
			actual.Required != *planned.Required || actual.MediaType != planned.MediaType {
			return fmt.Errorf("artifact order or identity mismatch at index %d", i)
		}
		if actual.ByteCount > planned.MaxBytes {
			return fmt.Errorf("artifact %q exceeds planned maximum", planned.ID)
		}
	}
	return nil
}

func verifyDetachedEvidence(manifest Manifest) error {
	expected := make([]EvidenceRecord, 0, len(manifest.Checks)*2)
	for _, check := range manifest.Checks {
		if check.Status == CheckStatusNotRun {
			continue
		}
		expected = append(expected,
			EvidenceRecord{LogicalName: check.CheckID + ".stdout", MediaType: "text/plain; charset=utf-8", SHA256: check.StdoutSHA256, ByteCount: check.StdoutByteCount, Availability: "detached"},
			EvidenceRecord{LogicalName: check.CheckID + ".stderr", MediaType: "text/plain; charset=utf-8", SHA256: check.StderrSHA256, ByteCount: check.StderrByteCount, Availability: "detached"},
		)
	}
	if len(manifest.DetachedEvidence) != len(expected)+1 ||
		!reflect.DeepEqual(manifest.DetachedEvidence[:len(expected)], expected) {
		return fmt.Errorf("detached evidence does not exactly bind executed check output")
	}
	diagnostic := manifest.DetachedEvidence[len(expected)]
	if diagnostic.LogicalName != "runner.diagnostics" || diagnostic.MediaType != "application/json" || diagnostic.Availability != "detached" {
		return fmt.Errorf("detached runner diagnostics evidence is missing or malformed")
	}
	return nil
}

func DeriveVerdict(manifest Manifest, plan Plan) (string, error) {
	if err := validateManifestStructure(manifest); err != nil {
		return "", err
	}
	pass := manifest.Repository.WorkingTreeCleanBefore && manifest.Repository.WorkingTreeCleanAfter && !manifest.Runner.VCSModified
	priorFailure := false
	for _, result := range manifest.Checks {
		switch result.Status {
		case CheckStatusPass:
			if priorFailure || result.ExitCode == nil || *result.ExitCode != 0 || result.OutputTruncated || result.OutputIncomplete || result.CleanupStatus != CleanupPass {
				pass = false
			}
		case CheckStatusFail:
			if priorFailure {
				return "", fmt.Errorf("check %q ran after a prior failure", result.CheckID)
			}
			priorFailure = true
			pass = false
		case CheckStatusNotRun:
			if !priorFailure {
				return "", fmt.Errorf("check %q was not run without a prior failure", result.CheckID)
			}
			pass = false
		}
	}
	for _, artifact := range manifest.Artifacts {
		if artifact.Required && artifact.Status != ArtifactStatusPass {
			pass = false
		}
	}
	if *plan.Policy.RequireDiffCheck && manifest.PatchHygiene.Status != CheckStatusPass {
		pass = false
	}
	if *plan.Policy.ForbidTrackedFullDigests && manifest.ClosurePolicy.TrackedFullDigestStatus != CheckStatusPass {
		pass = false
	}
	if pass {
		return VerdictPass, nil
	}
	return VerdictFail, nil
}
