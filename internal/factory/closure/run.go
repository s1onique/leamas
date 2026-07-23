package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RunOptions struct {
	PlanPath            string
	PlanFreeze          string
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

type resolvedFreeze struct {
	FreezeCommit  string
	PlanPath      string
	PlanBlobOID   string
	PlanSHA256    string
	SubjectCommit string
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
	resolved, err := resolvePlanFreeze(ctx, dependencies.Git, options.PlanFreeze, snapshot.RepositoryRoot, planPath, planBytes, snapshot.SubjectCommitOID)
	if err != nil {
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
	manifest := assembleManifest(plan, planBytes, planPath, resolved, snapshot, runner, checks, artifacts, evidence, patchHygiene, closurePolicy, cleanAfter)
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

func resolvePlanFreeze(
	ctx context.Context,
	git gitClient,
	freezeArg, repositoryRoot, planPath string,
	planBytes []byte,
	subjectCommit string,
) (resolvedFreeze, error) {
	if freezeArg == "" {
		return resolvedFreeze{}, fmt.Errorf("--plan-freeze is required (format <commit>:<path>)")
	}
	commit, pathInRepo, err := splitFreezeArgument(freezeArg)
	if err != nil {
		return resolvedFreeze{}, fmt.Errorf("--plan-freeze: %w", err)
	}
	if !oidPattern.MatchString(commit) || strings.HasPrefix(commit, "-") {
		return resolvedFreeze{}, fmt.Errorf("--plan-freeze: invalid commit OID %q", commit)
	}
	if commit == subjectCommit {
		return resolvedFreeze{}, fmt.Errorf("freeze commit %s must differ from subject commit %s", commit, subjectCommit)
	}
	ancestorResult := git.Run(ctx, repositoryRoot, "merge-base", "--is-ancestor", commit, subjectCommit)
	if ancestorResult.Err != nil || ancestorResult.ExitCode != 0 {
		return resolvedFreeze{}, fmt.Errorf("freeze commit %s is not an ancestor of subject %s", commit, subjectCommit)
	}
	canonicalPlan := relativePlanPath(planPath)
	if pathInRepo != canonicalPlan {
		return resolvedFreeze{}, fmt.Errorf("freeze plan path %q does not match executed plan path %q", pathInRepo, canonicalPlan)
	}
	blobResult := git.Run(ctx, repositoryRoot, "cat-file", "blob", commit+":"+canonicalPlan)
	if blobResult.Err != nil || blobResult.ExitCode != 0 {
		return resolvedFreeze{}, fmt.Errorf("read frozen plan blob %s:%s: %s", commit, canonicalPlan, sanitizeDiagnostic(string(blobResult.Stderr)))
	}
	frozenBytes := blobResult.Stdout
	sum := sha256.Sum256(frozenBytes)
	freezeSHA := hex.EncodeToString(sum[:])
	frozenBlobOID, err := runGitValue(ctx, git, repositoryRoot, "rev-parse", "--verify", "--end-of-options", commit+":"+canonicalPlan)
	if err != nil {
		return resolvedFreeze{}, fmt.Errorf("resolve frozen plan blob oid: %w", err)
	}
	var frozenPlan Plan
	if err := decodeStrictBounded(frozenBytes, MaxPlanBytes, &frozenPlan); err != nil {
		return resolvedFreeze{}, fmt.Errorf("parse frozen plan blob: %w", err)
	}
	var executedPlan Plan
	if err := decodeStrictBounded(planBytes, MaxPlanBytes, &executedPlan); err != nil {
		return resolvedFreeze{}, fmt.Errorf("parse executed plan: %w", err)
	}
	if frozenPlan.ActID != executedPlan.ActID || frozenPlan.RunnerBinding != executedPlan.RunnerBinding {
		return resolvedFreeze{}, fmt.Errorf(
			"executed plan metadata does not match frozen plan: "+
				"executed_act_id=%q frozen_act_id=%q executed_binding=%q frozen_binding=%q",
			executedPlan.ActID, frozenPlan.ActID, executedPlan.RunnerBinding, frozenPlan.RunnerBinding)
	}
	if frozenPlan.PolicyProfile != "" && frozenPlan.PolicyProfile != executedPlan.PolicyProfile {
		return resolvedFreeze{}, fmt.Errorf("executed plan policy profile does not match frozen plan: executed=%q frozen=%q", executedPlan.PolicyProfile, frozenPlan.PolicyProfile)
	}
	executionSum := sha256.Sum256(planBytes)
	executionSHA := hex.EncodeToString(executionSum[:])
	if freezeSHA != executionSHA {
		return resolvedFreeze{}, fmt.Errorf("frozen plan bytes do not match executed plan bytes: frozen=%s executed=%s", freezeSHA, executionSHA)
	}
	return resolvedFreeze{
		FreezeCommit:  commit,
		PlanPath:      canonicalPlan,
		PlanBlobOID:   frozenBlobOID,
		PlanSHA256:    freezeSHA,
		SubjectCommit: subjectCommit,
	}, nil
}

func splitFreezeArgument(value string) (commit, path string, err error) {
	for index := 0; index < len(value); index++ {
		if value[index] == ':' {
			return value[:index], value[index+1:], nil
		}
	}
	return "", "", fmt.Errorf("missing <commit>:<path> separator")
}

func relativePlanPath(planPath string) string {
	if strings.HasPrefix(planPath, "docs/") {
		return planPath
	}
	for index := 0; index < len(planPath); index++ {
		if planPath[index] == '/' {
			return planPath[index+1:]
		}
	}
	return planPath
}

func _useStrings() {
	_ = strings.HasPrefix
}

func assembleManifest(
	plan Plan,
	planBytes []byte,
	planPath string,
	resolved resolvedFreeze,
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
		PlanFreeze: ManifestPlanFreeze{
			FreezeCommit:  resolved.FreezeCommit,
			PlanPath:      resolved.PlanPath,
			PlanBlobOID:   resolved.PlanBlobOID,
			PlanSHA256:    resolved.PlanSHA256,
			SubjectCommit: resolved.SubjectCommit,
		},
		Subject: ManifestSubject{CommitOID: snapshot.SubjectCommitOID, TreeOID: snapshot.SubjectTreeOID},
		Runner:  runner,
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
