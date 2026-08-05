// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_runner.go provides the production v2
// runner. RunClosureProtocolV2 orchestrates:
//
//  1. validate request
//  2. resolve caller worktree state and HEAD
//  3. resolve subject / freeze commits and trees
//  4. evaluate topology via the resolver + dispatch
//  5. load frozen plan bytes from F:P
//  6. parse the plan contract from frozen bytes
//  7. execute checks against S^{tree} via detached worktree
//  8. verify observed execution tree
//  9. assemble manifest via NewV2Manifest
//  10. atomically write the manifest
//
// The runner MUST emit no success manifest on failure. All
// failures surface as typed V2Error values with a non-empty
// Diags list so the CLI can render structured output.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// V2RunnerDeps captures the dependencies the runner needs.
// The defaults match the production wiring.
type V2RunnerDeps struct {
	Git            gitClient
	Topology       V2TopologyResolver
	Loader         V2FrozenPlanLoader
	Executor       V2SubjectExecutor
	Commands       commandExecutor
	Now            func() time.Time
	BinaryIdentity func() V2BinaryIdentity
}

// DefaultV2RunnerDeps returns the production dependency
// wiring using RealGit and the default executors.
func DefaultV2RunnerDeps() V2RunnerDeps {
	return V2RunnerDeps{
		Git:      RealGit{},
		Topology: NewGitV2TopologyResolver(RealGit{}),
		Loader:   NewGitV2FrozenPlanLoader(RealGit{}),
		Executor: NewGitV2SubjectExecutor(RealGit{}),
		Commands: boundedCommandExecutor{},
		Now:      time.Now,
		BinaryIdentity: func() V2BinaryIdentity {
			return V2BinaryIdentity{Path: "leamas"}
		},
	}
}

// RunClosureProtocolV2 is the production entry point. The
// function returns the produced manifest on success or a
// V2Error carrying the typed diagnostics on failure.
//
// The runner never publishes a success manifest on failure.
// Cleanup of the temporary worktree is handled inside the
// executor's deferred cleanup.
func RunClosureProtocolV2(ctx context.Context, req V2Request) (V2Manifest, error) {
	deps := DefaultV2RunnerDeps()
	return RunClosureProtocolV2WithDeps(ctx, req, deps)
}

// RunClosureProtocolV2WithDeps runs the v2 runner with the
// supplied dependency wiring. Tests inject fakes; production
// callers use RunClosureProtocolV2.
func RunClosureProtocolV2WithDeps(ctx context.Context, req V2Request, deps V2RunnerDeps) (V2Manifest, error) {
	if err := ValidateV2Request(req); err != nil {
		return V2Manifest{}, err
	}
	if deps.Topology == nil {
		deps.Topology = NewGitV2TopologyResolver(deps.Git)
	}
	if deps.Loader == nil {
		deps.Loader = NewGitV2FrozenPlanLoader(deps.Git)
	}
	if deps.Executor == nil {
		deps.Executor = NewGitV2SubjectExecutor(deps.Git)
	}
	if deps.Commands == nil {
		deps.Commands = boundedCommandExecutor{}
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.BinaryIdentity == nil {
		deps.BinaryIdentity = func() V2BinaryIdentity { return V2BinaryIdentity{Path: "leamas"} }
	}
	if deps.Git == nil {
		deps.Git = RealGit{}
	}
	clean, cleanErr := workingTreeClean(ctx, deps.Git, req.RepositoryRoot)
	if cleanErr != nil {
		return V2Manifest{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("inspect caller worktree: %s", cleanErr.Error()),
			"caller_worktree", cleanErr.Error())
	}
	if !clean {
		return V2Manifest{}, NewV2ErrorWith(V2CodeCallerWorktreeDirty,
			"caller worktree must be clean before v2 run",
			"caller_worktree", "")
	}
	callerHead, err := runGitValue(ctx, deps.Git, req.RepositoryRoot, "rev-parse", "HEAD^{commit}")
	if err != nil {
		return V2Manifest{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("read caller HEAD: %s", err.Error()),
			"caller_head", err.Error())
	}
	facts, err := deps.Topology.ResolveTopology(ctx, req.RepositoryRoot, req.SubjectCommit, req.FreezeCommit)
	if err != nil {
		return V2Manifest{}, err
	}
	if !facts.SubjectResolved || !facts.FreezeResolved {
		return V2Manifest{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("topology resolution incomplete: subject=%v freeze=%v relation=%s",
				facts.SubjectResolved, facts.FreezeResolved, string(facts.Classify())),
			"topology", string(facts.Classify()))
	}
	subjectCommit := facts.SubjectCommitValue()
	freezeCommit := facts.FreezeCommitValue()
	subjectTree, err := runGitValue(ctx, deps.Git, req.RepositoryRoot, "rev-parse", "--verify", "--end-of-options", subjectCommit+"^{tree}")
	if err != nil {
		return V2Manifest{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("resolve subject tree: %s", err.Error()),
			"subject_tree", err.Error())
	}
	freezeTree, err := runGitValue(ctx, deps.Git, req.RepositoryRoot, "rev-parse", "--verify", "--end-of-options", freezeCommit+"^{tree}")
	if err != nil {
		return V2Manifest{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("resolve freeze tree: %s", err.Error()),
			"freeze_tree", err.Error())
	}
	outcome := DispatchClosureTopology(req.ClosureProtocolVersion, facts)
	if !outcome.Accepted {
		return V2Manifest{}, &V2Error{Diags: V2Diagnostics{{
			Code:         outcome.Code,
			Message:      outcome.Message,
			PropertyName: "topology",
			Detail:       string(outcome.Relation),
		}}}
	}
	frozen, err := deps.Loader.LoadFrozenPlan(ctx, req.RepositoryRoot, freezeCommit, req.PlanPath)
	if err != nil {
		return V2Manifest{}, err
	}
	workingBytes, err := ReadOptionalWorkingPlan(req.OptionalWorkingPlanAssertion)
	if err != nil {
		return V2Manifest{}, NewV2ErrorWith(V2CodeRequestIncomplete,
			fmt.Sprintf("read working plan: %s", err.Error()),
			"working_plan", err.Error())
	}
	if err := CompareToWorkingPlan(frozen, workingBytes); err != nil {
		return V2Manifest{}, err
	}
	plan, _, err := parsePlanBytes(frozen.Bytes)
	if err != nil {
		return V2Manifest{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("parse frozen plan: %s", err.Error()),
			"plan_bytes", err.Error())
	}
	if !PlanContractVersion(plan.ContractVersion).IsSupported() {
		return V2Manifest{}, NewV2ErrorWith(V2CodeUnsupportedPlanContractVersion,
			fmt.Sprintf("frozen plan contract version %d is not supported", plan.ContractVersion),
			"plan_contract_version", "")
	}
	if err := ValidateV2VersionCombination(PlanContractVersion(plan.ContractVersion), req.ClosureProtocolVersion); err != nil {
		return V2Manifest{}, err
	}
	if err := os.MkdirAll(req.EvidenceDirectory, 0o700); err != nil {
		return V2Manifest{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("mkdir evidence dir: %s", err.Error()),
			"evidence_directory", err.Error())
	}
	execResult, err := deps.Executor.ExecuteSubjectChecks(ctx, V2ExecuteRequest{
		RepositoryRoot:  req.RepositoryRoot,
		SubjectCommit:   subjectCommit,
		SubjectTree:     subjectTree,
		EvidenceDir:     req.EvidenceDirectory,
		Checks:          plan.Checks,
		CommandExecutor: deps.Commands,
		Now:             deps.Now,
	})
	if err != nil {
		return V2Manifest{}, err
	}
	if execResult.ObservedTree != subjectTree {
		return V2Manifest{}, NewV2ErrorWith(V2CodeExecutionTreeMismatch,
			fmt.Sprintf("executor observed tree %s does not match subject tree %s",
				execResult.ObservedTree, subjectTree),
			"execution_tree", execResult.ObservedTree)
	}
	checks := make([]V2CheckResult, 0, len(execResult.CheckResults))
	for _, c := range execResult.CheckResults {
		checks = append(checks, V2CheckResult{
			ID:      c.CheckID,
			Mode:    c.Status,
			Outcome: c.Status,
			Detail:  strings.TrimSpace(c.ExecutionErrorCode),
		})
	}
	binary := deps.BinaryIdentity()
	manifest, err := NewV2Manifest(V2ManifestBuild{
		ClosureProtocolVersion: req.ClosureProtocolVersion,
		PlanContractVersion:    PlanContractVersion(plan.ContractVersion),
		SubjectCommit:          subjectCommit,
		SubjectTree:            subjectTree,
		FreezeCommit:           freezeCommit,
		FreezeTree:             freezeTree,
		PlanPath:               frozen.Path,
		PlanBlob:               frozen.BlobOID,
		PlanSHA256:             frozen.SHA256,
		PlanBytes:              frozen.Bytes,
		ExecutionTree:          subjectTree,
		CallerHead:             callerHead,
		BinaryIdentity:         binary,
		CheckResults:           checks,
	})
	if err != nil {
		return V2Manifest{}, err
	}
	data, err := V2ManifestRender(manifest)
	if err != nil {
		return V2Manifest{}, NewV2ErrorWith(V2CodeManifestWriteFailed,
			fmt.Sprintf("render manifest: %s", err.Error()),
			"manifest_output", err.Error())
	}
	if err := AtomicWriteV2Manifest(req.ManifestOutput, data); err != nil {
		return V2Manifest{}, err
	}
	return manifest, nil
}
