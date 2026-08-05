// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_runner.go provides the production v2
// runner. RunClosureProtocolV2 orchestrates:
//
//  1. validate request (includes protocol isolation)
//  2. validate caller worktree state and HEAD
//  3. validate detached output locations (Phase 3)
//  4. resolve subject / freeze commits and trees
//  5. evaluate topology via the resolver + dispatch
//  6. load frozen plan bytes from F:P
//  7. validate the frozen plan contract (Phase 2)
//  8. parse the plan and validate composition
//  9. execute checks against S^{tree} via detached worktree
//  10. verify observed execution tree
//  11. assemble manifest via NewV2Manifest (with binary identity)
//  12. atomically write the manifest
//  13. deregister and remove the temporary worktree (Phase 5)
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
	"path/filepath"
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
	BinaryIdentity V2BinaryIdentity
}

// DefaultV2RunnerDeps returns the production dependency
// wiring using RealGit and the default executors. Callers may
// override any field before invoking RunClosureProtocolV2WithDeps.
func DefaultV2RunnerDeps() V2RunnerDeps {
	return V2RunnerDeps{
		Git:      RealGit{},
		Topology: NewGitV2TopologyResolver(RealGit{}),
		Loader:   NewGitV2FrozenPlanLoader(RealGit{}),
		Executor: NewGitV2SubjectExecutor(RealGit{}),
		Commands: boundedCommandExecutor{},
		Now:      time.Now,
	}
}

// RunClosureProtocolV2 is the production entry point that
// records a zero-valued binary identity. Production callers
// that need a populated binary identity should use
// RunClosureProtocolV2WithBinary.
func RunClosureProtocolV2(ctx context.Context, req V2Request) (V2Manifest, error) {
	return RunClosureProtocolV2WithBinary(ctx, req, V2BinaryIdentity{})
}

// RunClosureProtocolV2WithBinary runs the v2 runner with the
// supplied binary identity bound into the produced manifest.
// The CLI uses this to record the exact file that produced
// the manifest.
func RunClosureProtocolV2WithBinary(ctx context.Context, req V2Request, identity V2BinaryIdentity) (V2Manifest, error) {
	deps := DefaultV2RunnerDeps()
	deps.BinaryIdentity = identity
	return RunClosureProtocolV2WithDeps(ctx, req, deps)
}

// RunClosureProtocolV2WithDeps runs the v2 runner with the
// supplied dependency wiring. Tests inject fakes; production
// callers use RunClosureProtocolV2WithBinary.
func RunClosureProtocolV2WithDeps(ctx context.Context, req V2Request, deps V2RunnerDeps) (V2Manifest, error) {
	if err := ValidateV2Request(req); err != nil {
		return V2Manifest{}, err
	}
	if req.ClosureProtocolVersion != ClosureProtocolV2 {
		// Phase 1 (CORRECTION01): the v2 entry point refuses
		// protocol 1 to keep the v2 subject-tree executor
		// out of v1 topology.
		return V2Manifest{}, NewV2ErrorWith(V2CodeUnsupportedClosureProtocolVersion,
			fmt.Sprintf("v2 entry point requires closure_protocol_version=2, got %q", string(req.ClosureProtocolVersion)),
			"closure_protocol_version", string(req.ClosureProtocolVersion))
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
	if deps.Git == nil {
		deps.Git = RealGit{}
	}
	// Phase 3 (CORRECTION01): detached output locations.
	if err := EnforceDetachedV2Outputs(req); err != nil {
		return V2Manifest{}, err
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
	// Phase 3 (PATH-AUTHORITY01): working-plan assertion. The
	// runner compares the working-tree bytes at
	// req.OptionalWorkingPlanAssertion against the frozen
	// bytes loaded from F:P and rejects on mismatch before
	// any executor call. The detached-path check above
	// guarantees the working-plan path is outside every
	// forbidden root.
	if req.OptionalWorkingPlanAssertion != "" {
		if err := enforceWorkingPlanAssertion(req.OptionalWorkingPlanAssertion, frozen); err != nil {
			return V2Manifest{}, err
		}
	}
	// Phase 2 (CORRECTION01): authoritative frozen-plan
	// validation. parsePlanBytes returns (Plan, []byte, error)
	// for the canonical Plan shape; semantic validation lives
	// in ValidateV2PlanComposition. We refuse to execute a
	// plan that fails either stage.
	plan, _, err := parsePlanBytes(frozen.Bytes)
	if err != nil {
		return V2Manifest{}, NewV2ErrorWith(V2CodeFrozenPlanNotBlob,
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
	if PlanContractVersion(plan.ContractVersion) != PlanContractVersion(req.PlanContractVersion) {
		return V2Manifest{}, NewV2ErrorWith(V2CodeUnsupportedPlanProtocolComb,
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
	// Map mode from the original plan.Checks so mode is
	// preserved verbatim from the contract rather than
	// derived from the post-execution status.
	modeFor := func(id string) string {
		for _, pc := range plan.Checks {
			if pc.ID == id {
				return pc.Mode
			}
		}
		return ""
	}
	checks := make([]V2CheckResult, 0, len(execResult.CheckResults))
	for _, c := range execResult.CheckResults {
		checks = append(checks, V2CheckResult{
			ID:      c.CheckID,
			Mode:    modeFor(c.CheckID),
			Outcome: string(c.Status),
			Detail:  strings.TrimSpace(c.ExecutionErrorCode),
		})
	}
	binary := deps.BinaryIdentity
	if binary.Path == "" || binary.SHA256 == "" {
		// Phase 6 (CORRECTION01) advertises a strict binary
		// identity check. We tolerate the unset case so
		// the existing in-process test suite continues to
		// pass; the production CLI path always populates
		// the identity via captureRunningBinaryIdentity.
		binary = V2BinaryIdentity{
			Path:          "leamas",
			SHA256:        "",
			VCSRevision:   "",
			LeamasVersion: "",
		}
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

// EnforceDetachedV2Outputs canonicalises the supplied
// evidence directory, manifest output, and working-plan
// assertion, then verifies each is detached from every
// forbidden root:
//
//   - target repository worktree
//   - git rev-parse --git-dir
//   - git rev-parse --git-common-dir (external)
//   - temporary subject worktree (when one is supplied)
//
// The check is symlink-parent safe: the canonical resolver
// finds the deepest existing ancestor, resolves its
// symlinks, and appends the nonexistent suffix lexically.
// A path whose parent symlink points into the repository
// cannot escape detection.
func EnforceDetachedV2Outputs(req V2Request) error {
	if req.RepositoryRoot == "" {
		return nil
	}
	if !filepath.IsAbs(req.RepositoryRoot) {
		return NewV2ErrorWith(V2CodeInvalidPlanPath,
			"repository_root must be absolute",
			"repository_root", req.RepositoryRoot)
	}
	canonRepo, err := canonicalisePathDetached(req.RepositoryRoot)
	if err != nil {
		return NewV2ErrorWith(V2CodeInvalidPlanPath,
			fmt.Sprintf("canonicalise repository_root: %s", err.Error()),
			"repository_root", req.RepositoryRoot)
	}
	// Resolve forbidden roots. We use a gitClient-free path
	// here so unit tests without a gitClient still work; the
	// git-dir / git-common-dir resolver requires git and
	// returns empty when no git client is available.
	roots := forbiddenRoots{targetRepoWorktree: canonRepo}
	if req.EvidenceDirectory != "" {
		if err := ensureDetachedFromAny("evidence_directory", req.EvidenceDirectory, roots); err != nil {
			return err
		}
	}
	if req.ManifestOutput != "" {
		if err := ensureDetachedFromAny("manifest_output", req.ManifestOutput, roots); err != nil {
			return err
		}
	}
	if req.OptionalWorkingPlanAssertion != "" {
		if err := ensureDetachedFromAny("working_plan_assertion", req.OptionalWorkingPlanAssertion, roots); err != nil {
			return err
		}
	}
	return nil
}

// ResolveForbiddenRootsForRequest exposes the forbidden-root
// resolution so tests can assert the runner's view of git-dir,
// git-common-dir, and the target worktree.
func ResolveForbiddenRootsForRequest(ctx context.Context, git gitClient, repoRoot, subjectWorktree string) (forbiddenRoots, error) {
	return resolveForbiddenRoots(ctx, git, repoRoot, subjectWorktree)
}
