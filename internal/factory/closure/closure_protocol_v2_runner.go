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
//  14. caller-state drift check (LIFECYCLE-INVARIANTS01)
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
	"path/filepath"
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
//
// Phase 4 (LIFECYCLE-INVARIANTS01): the runner captures the
// caller state BEFORE any side-effecting step and re-snapshots
// it on every return path. If the after snapshot disagrees with
// the before snapshot (HEAD changed, HEAD tree changed,
// `git status --porcelain=v2 --untracked-files=all` produced
// different output, or a new linked-worktree registration
// leaked), the runner emits typed diagnostics and refuses to
// claim success.
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
	// Phase 4 (LIFECYCLE-INVARIANTS01): capture the caller
	// state BEFORE any side-effecting step. The post-run
	// snapshot is captured just before the function returns
	// (success or failure) so the runner can refuse to claim
	// success when HEAD, HEAD tree, status, or
	// linked-worktree registrations drifted.
	//
	// R1 (MAC-CANARY-READINESS01-R1): the before snapshot
	// must be fail-closed. Any observation failure (HEAD
	// lookup, status, worktree inventory) produces typed
	// V2Diagnostics and the runner refuses to execute.
	callerBefore := snapshotCallerState(ctx, deps.Git, req.RepositoryRoot)
	if !callerBefore.Available {
		return V2Manifest{}, &V2Error{Diags: callerBefore.Diagnostics}
	}
	manifest, err := runClosureProtocolV2Inner(ctx, req, deps, callerBefore.State)
	callerAfter := snapshotCallerState(ctx, deps.Git, req.RepositoryRoot)
	if !callerAfter.Available {
		// The after snapshot must be fail-closed: if we
		// cannot confirm the caller state after the run,
		// we MUST NOT claim clean success. The diagnostics
		// are merged with any prior error so the CLI can
		// render the exact cause.
		post := callerAfter.Diagnostics
		if err == nil {
			return V2Manifest{}, &V2Error{Diags: post}
		}
		if v2err, ok := err.(*V2Error); ok {
			v2err.Diags = append(v2err.Diags, post...)
			return V2Manifest{}, v2err
		}
		return V2Manifest{}, &V2Error{Diags: post}
	}
	if post := callerBefore.State.Diff(callerAfter.State); len(post) > 0 {
		// Drift detected. If the inner call succeeded we
		// MUST refuse to publish the manifest and surface
		// every diagnostic; if the inner call already
		// failed we still surface the drift so the CLI
		// can render the exact cause.
		if err == nil {
			return V2Manifest{}, &V2Error{Diags: post}
		}
		if v2err, ok := err.(*V2Error); ok {
			v2err.Diags = append(v2err.Diags, post...)
			return V2Manifest{}, v2err
		}
		return V2Manifest{}, &V2Error{Diags: post}
	}
	return manifest, err
}

// runClosureProtocolV2Inner is implemented in closure_protocol_v2_runner_inner.go.

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
