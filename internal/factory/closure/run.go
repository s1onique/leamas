package closure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type RunOptions struct {
	PlanPath            string
	Subject             string
	EvidenceDirectory   string
	ManifestOutput      string
	RepositoryDirectory string
}

type runDependencies struct {
	Git      gitClient
	Commands commandExecutor
	Runner   runnerIdentityProvider
	Now      func() time.Time
}

type detachedDiagnostics struct {
	ContractVersion     int    `json:"contract_version"`
	ActID               string `json:"act_id"`
	PatchHygieneOutput  string `json:"patch_hygiene_output"`
	ClosurePolicyOutput string `json:"closure_policy_output"`
}

func RunClosure(ctx context.Context, options RunOptions) (Manifest, []byte, error) {
	return runClosureWithDependencies(ctx, options, runDependencies{
		Git:      realGitClient{},
		Commands: boundedCommandExecutor{},
		Runner:   currentRunnerIdentity{},
		Now:      time.Now,
	})
}

func runClosureWithDependencies(ctx context.Context, options RunOptions, dependencies runDependencies) (Manifest, []byte, error) {
	if options.PlanPath == "" || options.Subject == "" || options.EvidenceDirectory == "" || options.ManifestOutput == "" {
		return Manifest{}, nil, fmt.Errorf("plan, subject, evidence directory, and manifest output are required")
	}
	if options.RepositoryDirectory == "" {
		options.RepositoryDirectory = "."
	}
	plan, planBytes, err := LoadPlan(options.PlanPath)
	if err != nil {
		return Manifest{}, nil, err
	}
	snapshot, err := snapshotSubject(ctx, dependencies.Git, options.RepositoryDirectory, options.Subject)
	if err != nil {
		return Manifest{}, nil, err
	}
	planPath, err := canonicalPlanPath(snapshot.RepositoryRoot, options.PlanPath, plan.ActID)
	if err != nil {
		return Manifest{}, nil, err
	}
	if err := validateBaselineIdentity(ctx, dependencies.Git, snapshot.RepositoryRoot, plan.Baseline); err != nil {
		return Manifest{}, nil, err
	}
	evidenceDirectory, err := prepareEvidenceDirectory(snapshot.RepositoryRoot, options.EvidenceDirectory)
	if err != nil {
		return Manifest{}, nil, err
	}
	if err := requireDetachedOutput(snapshot.RepositoryRoot, options.ManifestOutput); err != nil {
		return Manifest{}, nil, err
	}
	runner, err := dependencies.Runner.Identity()
	if err != nil {
		return Manifest{}, nil, err
	}
	checks, evidence, err := executeChecks(ctx, checkExecutionRequest{
		RepositoryRoot:    snapshot.RepositoryRoot,
		EvidenceDirectory: evidenceDirectory,
		SubjectTreeOID:    snapshot.SubjectTreeOID,
		Checks:            plan.Checks,
		Now:               dependencies.Now,
	}, dependencies.Commands)
	if err != nil {
		return Manifest{}, nil, err
	}
	artifacts := collectRepositoryArtifacts(snapshot.RepositoryRoot, plan.Artifacts)
	patchHygiene, patchDiagnostics := evaluateRequiredPatchHygiene(ctx, dependencies.Git, snapshot.RepositoryRoot, plan)
	closurePolicy, policyDiagnostics := evaluateRequiredClosurePolicy(ctx, dependencies.Git, snapshot.RepositoryRoot, plan, snapshot.SubjectCommitOID)
	cleanAfter, cleanErr := workingTreeClean(ctx, dependencies.Git, snapshot.RepositoryRoot)
	if cleanErr != nil {
		cleanAfter = false
	}
	diagnosticRecord, err := writeRunnerDiagnostics(evidenceDirectory, plan.ActID, patchDiagnostics, policyDiagnostics)
	if err != nil {
		return Manifest{}, nil, err
	}
	evidence = append(evidence, diagnosticRecord)
	manifest := assembleManifest(plan, planBytes, planPath, snapshot, runner, checks, artifacts, evidence, patchHygiene, closurePolicy, cleanAfter)
	verdict, err := DeriveVerdict(manifest, plan)
	if err != nil {
		return Manifest{}, nil, err
	}
	manifest.Verdict = verdict
	data, err := MarshalManifest(manifest, plan)
	if err != nil {
		return Manifest{}, nil, err
	}
	if err := WriteManifest(options.ManifestOutput, data); err != nil {
		return Manifest{}, nil, err
	}
	return manifest, data, nil
}

func assembleManifest(
	plan Plan,
	planBytes []byte,
	planPath string,
	snapshot subjectSnapshot,
	runner RunnerIdentity,
	checks []CheckResult,
	artifacts []ArtifactResult,
	evidence []EvidenceRecord,
	patch PatchHygiene,
	policy ClosurePolicyResult,
	cleanAfter bool,
) Manifest {
	return Manifest{
		ContractVersion: ContractVersionV1,
		ActID:           plan.ActID,
		Plan:            ManifestPlanRef{SHA256: SHA256Hex(planBytes), Path: planPath},
		Subject:         ManifestSubject{CommitOID: snapshot.SubjectCommitOID, TreeOID: snapshot.SubjectTreeOID},
		Runner:          runner,
		Repository: RepositoryIdentity{
			Root: ".", RemoteURL: snapshot.RemoteURL, Branch: snapshot.Branch,
			HeadCommitOID: snapshot.HeadCommitOID, HeadTreeOID: snapshot.HeadTreeOID,
			OriginMainCommitOID: snapshot.OriginMainCommitOID, AheadBy: snapshot.AheadBy, BehindBy: snapshot.BehindBy,
			WorkingTreeCleanBefore: snapshot.Clean, WorkingTreeCleanAfter: cleanAfter,
		},
		Checks:           checks,
		Artifacts:        artifacts,
		DetachedEvidence: evidence,
		PatchHygiene:     patch,
		ClosurePolicy:    policy,
		ExcludedChecks:   excludedCheckResults(plan.Checks, snapshot.SubjectTreeOID),
		Verdict:          VerdictFail,
	}
}

func excludedCheckResults(checks []PlanCheck, subjectTree string) []ExcludedCheck {
	results := make([]ExcludedCheck, 0)
	for _, check := range checks {
		if check.Mode == CheckModeExclude {
			results = append(results, ExcludedCheck{CheckID: check.ID, SubjectTreeOID: subjectTree, Reason: check.Reason})
		}
	}
	return results
}

func writeRunnerDiagnostics(directory, actID string, patch, policy []byte) (EvidenceRecord, error) {
	data, err := json.MarshalIndent(detachedDiagnostics{
		ContractVersion:     ContractVersionV1,
		ActID:               actID,
		PatchHygieneOutput:  string(patch),
		ClosurePolicyOutput: string(policy),
	}, "", "  ")
	if err != nil {
		return EvidenceRecord{}, fmt.Errorf("marshal runner diagnostics: %w", err)
	}
	data = append(data, '\n')
	return writeDetachedBytes(directory, "runner.diagnostics", "application/json", data)
}

func requireDetachedOutput(root, output string) error {
	if !filepath.IsAbs(output) {
		return fmt.Errorf("manifest output must be an absolute detached path")
	}
	parent := filepath.Dir(output)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create manifest output parent: %w", err)
	}
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return fmt.Errorf("resolve manifest output parent: %w", err)
	}
	inside, err := pathInside(filepath.Join(resolvedParent, filepath.Base(output)), root)
	if err != nil {
		return err
	}
	if inside {
		return fmt.Errorf("manifest output must be outside the Git worktree")
	}
	return nil
}
