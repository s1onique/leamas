// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_runner_inner.go isolates the
// orchestration body of the v2 runner from
// RunClosureProtocolV2WithDeps so the caller-state drift
// check (LIFECYCLE-INVARIANTS01) can run on every return
// path without duplicating the inner logic.
//
// Splitting this from closure_protocol_v2_runner.go keeps
// the runner entry-point under the LLM-friendly 400-line
// threshold while preserving the single closure over the
// descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.
//
// R2A (MAC-CANARY-READINESS01-R2A): the inner runner returns
// an unpublished v2RunCandidate rather than writing the
// manifest. Publication is the outer runner's responsibility
// and only happens after the after-state authority passes.

import (
	"context"
	"fmt"
	"os"
)

// runClosureProtocolV2Inner is the orchestration body of the
// v2 runner, isolated from the caller-state drift check so
// the drift check can run on every return path without
// duplicating the inner logic.
//
// The inner authority may load and validate the frozen plan,
// execute the subject checks, and construct the manifest
// candidate. It MUST NOT write req.ManifestOutput; the outer
// runner's publication barrier is the only path that may
// publish the candidate bytes.
//
// LIFECYCLE-INVARIANTS01 wires this function as a pure
// helper; callers should continue to use
// RunClosureProtocolV2WithDeps.
func runClosureProtocolV2Inner(ctx context.Context, req V2Request, deps V2RunnerDeps, callerBefore v2CallerState, topology executionTopology) (v2RunCandidate, error) {
	_ = callerBefore
	// Phase 3 (CORRECTION01): detached output locations.
	if err := EnforceDetachedV2Outputs(req); err != nil {
		return v2RunCandidate{}, err
	}
	clean, cleanErr := workingTreeClean(ctx, deps.Git, req.RepositoryRoot)
	if cleanErr != nil {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("inspect caller worktree: %s", cleanErr.Error()),
			"caller_worktree", cleanErr.Error())
	}
	if !clean {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeCallerWorktreeDirty,
			"caller worktree must be clean before v2 run",
			"caller_worktree", "")
	}
	callerHead, err := runGitValue(ctx, deps.Git, req.RepositoryRoot, "rev-parse", "HEAD^{commit}")
	if err != nil {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("read caller HEAD: %s", err.Error()),
			"caller_head", err.Error())
	}
	facts, err := deps.Topology.ResolveTopology(ctx, req.RepositoryRoot, req.SubjectCommit, req.FreezeCommit)
	if err != nil {
		return v2RunCandidate{}, err
	}
	if !facts.SubjectResolved || !facts.FreezeResolved {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("topology resolution incomplete: subject=%v freeze=%v relation=%s",
				facts.SubjectResolved, facts.FreezeResolved, string(facts.Classify())),
			"topology", string(facts.Classify()))
	}
	subjectCommit := facts.SubjectCommitValue()
	freezeCommit := facts.FreezeCommitValue()
	subjectTree, err := runGitValue(ctx, deps.Git, req.RepositoryRoot, "rev-parse", "--verify", "--end-of-options", subjectCommit+"^{tree}")
	if err != nil {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("resolve subject tree: %s", err.Error()),
			"subject_tree", err.Error())
	}
	freezeTree, err := runGitValue(ctx, deps.Git, req.RepositoryRoot, "rev-parse", "--verify", "--end-of-options", freezeCommit+"^{tree}")
	if err != nil {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("resolve freeze tree: %s", err.Error()),
			"freeze_tree", err.Error())
	}
	outcome := dispatchClosureTopology(req.ClosureProtocolVersion, topology, facts)
	if !outcome.Accepted {
		return v2RunCandidate{}, &V2Error{Diags: V2Diagnostics{{
			Code:         outcome.Code,
			Message:      outcome.Message,
			PropertyName: "topology",
			Detail:       string(outcome.Relation),
		}}}
	}
	frozen, err := deps.Loader.LoadFrozenPlan(ctx, req.RepositoryRoot, freezeCommit, req.PlanPath)
	if err != nil {
		return v2RunCandidate{}, err
	}
	// Phase 3 (PATH-AUTHORITY01): working-plan assertion. The
	// runner compares the working-tree bytes at
	// req.OptionalWorkingPlanAssertion against the frozen
	// bytes loaded from F:P and rejects on mismatch before
	// any executor call. The detached-path check above
	// guarantees the working-plan path is outside every
	// forbidden root.
	if req.OptionalWorkingPlanAssertion != "" {
		if err := enforceWorkingPlanAssertion(req.OptionalWorkingPlanAssertion, frozen); err != nil {
			return v2RunCandidate{}, err
		}
	}
	// Phase 2 (CORRECTION01): authoritative frozen-plan
	// validation. parsePlanBytes returns (Plan, []byte, error)
	// for the canonical Plan shape; semantic validation lives
	// in ValidateV2PlanComposition. We refuse to execute a
	// plan that fails either stage.
	plan, _, err := parsePlanBytes(frozen.Bytes)
	if err != nil {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeFrozenPlanNotBlob,
			fmt.Sprintf("parse frozen plan: %s", err.Error()),
			"plan_bytes", err.Error())
	}
	if !PlanContractVersion(plan.ContractVersion).IsSupported() {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeUnsupportedPlanContractVersion,
			fmt.Sprintf("frozen plan contract version %d is not supported", plan.ContractVersion),
			"plan_contract_version", "")
	}
	if err := ValidateV2VersionCombination(PlanContractVersion(plan.ContractVersion), req.ClosureProtocolVersion); err != nil {
		return v2RunCandidate{}, err
	}
	if PlanContractVersion(plan.ContractVersion) != PlanContractVersion(req.PlanContractVersion) {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeUnsupportedPlanProtocolComb,
			fmt.Sprintf("request plan contract version %d does not match frozen plan version %d",
				req.PlanContractVersion, plan.ContractVersion),
			"plan_contract_version", fmt.Sprintf("request=%d frozen=%d", req.PlanContractVersion, plan.ContractVersion))
	}
	// Phase 1 (VALID-PLAN-AUTHORITY01): authoritative
	// frozen-plan validation with hard rejection. The runner
	// MUST refuse to execute a plan that fails any of the
	// parse / structural / semantic / composed validation
	// stages. The validation runs against the EXACT frozen
	// bytes loaded from F:P and retains every nested plan
	// diagnostic in the typed V2Error.
	if _, err := ValidateFrozenPlanV2(frozen.Bytes); err != nil {
		return v2RunCandidate{}, err
	}
	if err := os.MkdirAll(req.EvidenceDirectory, 0o700); err != nil {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeGitOperationFailed,
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
		return v2RunCandidate{}, err
	}
	if execResult.ObservedTree != subjectTree {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeExecutionTreeMismatch,
			fmt.Sprintf("executor observed tree %s does not match subject tree %s",
				execResult.ObservedTree, subjectTree),
			"execution_tree", execResult.ObservedTree)
	}
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
		BinaryIdentity:         deps.BinaryIdentity,
		PlanChecks:             plan.Checks,
		ExecutionResults:       execResult.CheckResults,
		Evidence:               execResult.Evidence,
	})
	if err != nil {
		return v2RunCandidate{}, err
	}
	data, err := V2ManifestRender(manifest)
	if err != nil {
		return v2RunCandidate{}, NewV2ErrorWith(V2CodeManifestWriteFailed,
			fmt.Sprintf("render manifest: %s", err.Error()),
			"manifest_output", err.Error())
	}
	candidate := v2RunCandidate{Manifest: manifest, ManifestBytes: data}
	// R2A candidate observer: invoked once, after the
	// candidate is constructed and rendered. The observer is
	// invocation-local; tests use a counting observer to
	// assert the candidate was constructed exactly once.
	if deps.CandidateObserver != nil {
		deps.CandidateObserver.CandidateConstructed(manifest, data)
	}
	// The inner runner MUST NOT publish the manifest. The
	// outer runner is the only path that may call
	// AtomicWriteV2Manifest, and only after the after-state
	// authority passes.
	return candidate, nil
}
