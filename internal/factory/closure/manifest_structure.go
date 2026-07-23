package closure

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

func validateManifestStructure(manifest Manifest) error {
	if manifest.ContractVersion != ContractVersionV1 {
		return fmt.Errorf("unsupported closure manifest contract_version %d", manifest.ContractVersion)
	}
	if !actIDPattern.MatchString(manifest.ActID) || containsClosurePlaceholder(manifest.ActID) {
		return fmt.Errorf("invalid manifest act_id %q", manifest.ActID)
	}
	if err := validateSHA256("plan.sha256", manifest.Plan.SHA256); err != nil {
		return err
	}
	if err := validateRepositoryRelativePath(manifest.Plan.Path, false); err != nil {
		return fmt.Errorf("plan.path: %w", err)
	}
	expectedPlanPath := "docs/closure-plans/" + manifest.ActID + ".json"
	if manifest.Plan.Path != expectedPlanPath {
		return fmt.Errorf("plan.path must be canonical %q", expectedPlanPath)
	}
	if err := validateOID("subject.commit_oid", manifest.Subject.CommitOID); err != nil {
		return err
	}
	if err := validateOID("subject.tree_oid", manifest.Subject.TreeOID); err != nil {
		return err
	}
	if err := validateRunnerIdentity(manifest.Runner); err != nil {
		return err
	}
	if err := validateRepositoryIdentity(manifest.Repository); err != nil {
		return err
	}
	if len(manifest.Checks) > MaxChecks || len(manifest.ExcludedChecks) > MaxChecks {
		return fmt.Errorf("manifest check count exceeds %d", MaxChecks)
	}
	if len(manifest.Artifacts) > MaxArtifacts {
		return fmt.Errorf("manifest artifact count exceeds %d", MaxArtifacts)
	}
	for i, result := range manifest.Checks {
		if err := validateCheckResult(i, result); err != nil {
			return err
		}
	}
	for i, artifact := range manifest.Artifacts {
		if err := validateArtifactResult(i, artifact); err != nil {
			return err
		}
	}
	if err := validateEvidenceRecords(manifest.DetachedEvidence); err != nil {
		return err
	}
	if err := validateSimpleStatus("patch_hygiene.status", manifest.PatchHygiene.Status); err != nil {
		return err
	}
	if manifest.PatchHygiene.DiagnosticCount < 0 {
		return fmt.Errorf("patch_hygiene.diagnostic_count is negative")
	}
	if err := validateSimpleStatus("closure_policy.tracked_full_digest_status", manifest.ClosurePolicy.TrackedFullDigestStatus); err != nil {
		return err
	}
	if manifest.ClosurePolicy.DiagnosticCount < 0 {
		return fmt.Errorf("closure_policy.diagnostic_count is negative")
	}
	if manifest.Verdict != VerdictPass && manifest.Verdict != VerdictFail {
		return fmt.Errorf("invalid manifest verdict %q", manifest.Verdict)
	}
	return nil
}

func validateRunnerIdentity(runner RunnerIdentity) error {
	if strings.TrimSpace(runner.LeamasVersion) == "" || containsClosurePlaceholder(runner.LeamasVersion) {
		return fmt.Errorf("runner.leamas_version is invalid")
	}
	if err := validateSHA256("runner.binary_sha256", runner.BinarySHA256); err != nil {
		return err
	}
	return validateOID("runner.vcs_revision", runner.VCSRevision)
}

func validateRepositoryIdentity(repository RepositoryIdentity) error {
	if repository.Root != "." {
		return fmt.Errorf("repository.root must be %q", ".")
	}
	if strings.TrimSpace(repository.Branch) == "" || containsClosurePlaceholder(repository.Branch) {
		return fmt.Errorf("repository.branch is invalid")
	}
	if err := validateOID("repository.head_commit_oid", repository.HeadCommitOID); err != nil {
		return err
	}
	if err := validateOID("repository.head_tree_oid", repository.HeadTreeOID); err != nil {
		return err
	}
	if repository.OriginMainCommitOID != "" {
		if err := validateOID("repository.origin_main_commit_oid", repository.OriginMainCommitOID); err != nil {
			return err
		}
	}
	if repository.RemoteURL != "" {
		if strings.ContainsAny(repository.RemoteURL, "\r\n") || strings.Contains(repository.RemoteURL, "://") && remoteURLHasUserInfo(repository.RemoteURL) {
			return fmt.Errorf("repository.remote_url contains credentials or is malformed")
		}
	}
	if repository.AheadBy != nil && *repository.AheadBy < 0 || repository.BehindBy != nil && *repository.BehindBy < 0 {
		return fmt.Errorf("repository ahead/behind counts must be non-negative")
	}
	return nil
}

func remoteURLHasUserInfo(raw string) bool {
	parsed, err := url.Parse(raw)
	return err != nil || parsed.User != nil
}

func validateCheckResult(index int, result CheckResult) error {
	prefix := fmt.Sprintf("checks[%d]", index)
	if !itemIDPattern.MatchString(result.CheckID) || containsClosurePlaceholder(result.CheckID) {
		return fmt.Errorf("%s.check_id is invalid", prefix)
	}
	if err := validateOID(prefix+".subject_tree_oid", result.SubjectTreeOID); err != nil {
		return err
	}
	if len(result.Argv) == 0 || len(result.Argv) > MaxArgvElements {
		return fmt.Errorf("%s.argv is invalid", prefix)
	}
	if err := validateRepositoryRelativePath(result.WorkingDirectory, true); err != nil {
		return fmt.Errorf("%s.working_directory: %w", prefix, err)
	}
	if !sort.StringsAreSorted(result.OverriddenEnvironment) {
		return fmt.Errorf("%s.overridden_environment is not sorted", prefix)
	}
	for _, name := range result.OverriddenEnvironment {
		if !environmentNamePattern.MatchString(name) {
			return fmt.Errorf("%s.overridden_environment contains invalid name", prefix)
		}
	}
	if result.DurationMS < 0 || result.StdoutByteCount < 0 || result.StderrByteCount < 0 || result.OutputBytesObserved < 0 {
		return fmt.Errorf("%s contains a negative measurement", prefix)
	}
	switch result.Status {
	case CheckStatusPass, CheckStatusFail:
		if result.ExitCode == nil || result.StartedAtUTC == "" || result.FinishedAtUTC == "" {
			return fmt.Errorf("%s executed result is incomplete", prefix)
		}
		if err := validateUTCInterval(result.StartedAtUTC, result.FinishedAtUTC); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
		if err := validateSHA256(prefix+".stdout_sha256", result.StdoutSHA256); err != nil {
			return err
		}
		if err := validateSHA256(prefix+".stderr_sha256", result.StderrSHA256); err != nil {
			return err
		}
		if result.CleanupStatus != CleanupPass && result.CleanupStatus != CleanupFailed {
			return fmt.Errorf("%s.cleanup_status is invalid", prefix)
		}
	case CheckStatusNotRun:
		if result.ExitCode != nil || result.StartedAtUTC != "" || result.FinishedAtUTC != "" ||
			result.StdoutSHA256 != "" || result.StderrSHA256 != "" || result.DurationMS != 0 ||
			result.StdoutByteCount != 0 || result.StderrByteCount != 0 || result.OutputBytesObserved != 0 ||
			result.OutputTruncated || result.OutputIncomplete || result.ExecutionErrorCode != "" || result.CleanupStatus != CleanupNotRequired {
			return fmt.Errorf("%s not-run result contains execution evidence", prefix)
		}
	default:
		return fmt.Errorf("%s.status is invalid", prefix)
	}
	return nil
}

func validateUTCInterval(startRaw, finishRaw string) error {
	start, err := time.Parse(time.RFC3339Nano, startRaw)
	if err != nil || !strings.HasSuffix(startRaw, "Z") {
		return fmt.Errorf("started_at_utc is not canonical UTC")
	}
	finish, err := time.Parse(time.RFC3339Nano, finishRaw)
	if err != nil || !strings.HasSuffix(finishRaw, "Z") {
		return fmt.Errorf("finished_at_utc is not canonical UTC")
	}
	if finish.Before(start) {
		return fmt.Errorf("finished_at_utc precedes started_at_utc")
	}
	return nil
}

func validateSHA256(field, value string) error {
	if containsClosurePlaceholder(value) || len(value) != 64 {
		return fmt.Errorf("%s must be a lowercase SHA-256", field)
	}
	for _, char := range value {
		if char < '0' || char > '9' && char < 'a' || char > 'f' {
			return fmt.Errorf("%s must be a lowercase SHA-256", field)
		}
	}
	return nil
}

func validateSimpleStatus(field, status string) error {
	if status != CheckStatusPass && status != CheckStatusFail {
		return fmt.Errorf("%s is invalid", field)
	}
	return nil
}
