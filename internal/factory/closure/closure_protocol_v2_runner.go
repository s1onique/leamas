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
//  12. capture caller-after state (publication barrier gate)
//  13. verify no caller drift or worktree registration leak
//  14. atomically write the manifest (publication barrier exit)
//  15. caller-state drift check (LIFECYCLE-INVARIANTS01)
//
// The runner MUST emit no success manifest on failure. All
// failures surface as typed V2Error values with a non-empty
// Diags list so the CLI can render structured output.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.
//
// R2A (MAC-CANARY-READINESS01-R2A): the runner now enforces a
// publication barrier between manifest construction (inner)
// and manifest publication (outer). No on-disk manifest bytes
// become visible until the after-state authority proves the
// caller state was observed successfully and is unchanged
// from the before-state snapshot.

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
	// SnapshotFn captures the caller-state observation for
	// the given snapshot phase. Production wiring uses
	// defaultV2RunnerSnapshotFunc; tests inject a fake that
	// returns a non-Available v2CallerStateSnapshot at the
	// V2SnapshotPhaseAfter boundary to prove the outer runner
	// refuses to publish the manifest.
	SnapshotFn V2RunnerSnapshotFunc
	// CandidateObserver, when non-nil, is invoked by the
	// inner runner immediately after the manifest candidate
	// is constructed and rendered. Tests inject a counting
	// observer to prove the candidate was constructed exactly
	// once. The runner skips the observer call when nil.
	CandidateObserver V2CandidateObserver
}

// DefaultV2RunnerDeps returns the production dependency
// wiring using RealGit and the default executors. Callers may
// override any field before invoking RunClosureProtocolV2WithDeps.
func DefaultV2RunnerDeps() V2RunnerDeps {
	return V2RunnerDeps{
		Git:               RealGit{},
		Topology:          NewGitV2TopologyResolver(RealGit{}),
		Loader:            NewGitV2FrozenPlanLoader(RealGit{}),
		Executor:          NewGitV2SubjectExecutor(RealGit{}),
		Commands:          boundedCommandExecutor{},
		Now:               time.Now,
		SnapshotFn:        defaultV2RunnerSnapshotFunc,
		CandidateObserver: noopCandidateObserver{},
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
// R2A (MAC-CANARY-READINESS01-R2A): the outer runner sequence
// is now:
//
//	capture before snapshot
//	run inner authority → unpublished candidate
//	capture after snapshot
//	require after Available=true
//	require no caller drift
//	require no worktree registration drift
//	atomically publish candidate bytes to ManifestOutput
//	return (manifest, nil)
//
// No manifest bytes reach the on-disk path before the
// after-state authority passes. Any failure leaves the
// manifest path absent.
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
	if deps.SnapshotFn == nil {
		deps.SnapshotFn = defaultV2RunnerSnapshotFunc
	}
	if deps.CandidateObserver == nil {
		// Production may leave the observer nil; the inner
		// runner skips the observer call when nil so the
		// barrier remains correct.
	}
	// Phase 4 (LIFECYCLE-INVARIANTS01): capture the caller
	// state BEFORE any side-effecting step. The post-run
	// snapshot is captured just before publication (success
	// path) or before the function returns (failure path) so
	// the runner can refuse to claim success when HEAD, HEAD
	// tree, status, or linked-worktree registrations drifted.
	//
	// R1 (MAC-CANARY-READINESS01-R1): the before snapshot
	// must be fail-closed. Any observation failure (HEAD
	// lookup, status, worktree inventory) produces typed
	// V2Diagnostics and the runner refuses to execute.
	//
	// R2A (MAC-CANARY-READINESS01-R2A): the snapshot
	// observation now flows through deps.SnapshotFn with an
	// explicit phase identifier so tests can target the
	// after boundary without command-count approximations.
	callerBefore := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseBefore)
	if !callerBefore.Available {
		return V2Manifest{}, &V2Error{Diags: callerBefore.Diagnostics}
	}
	candidate, err := runClosureProtocolV2Inner(ctx, req, deps, callerBefore.State)
	if err != nil {
		// Even on inner failure the runner still captures an
		// after-state observation so it can surface drift
		// diagnostics. No manifest bytes have been written;
		// the candidate is unpublished.
		//
		// R2C-R2: the inner-error path now ALWAYS processes
		// the after snapshot. If the snapshot is unavailable
		// the availability diagnostics are appended; if it
		// is available the before-vs-after drift is
		// appended. The original inner diagnostic is preserved
		// in front of the after diagnostics so the CLI sees
		// both root-cause and symptom.
		callerAfter := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseAfter)
		post := mergeAfterWithBeforeDiagnostics(callerBefore, callerAfter)
		if v2err, ok := err.(*V2Error); ok {
			v2err.Diags = append(v2err.Diags, post...)
			return V2Manifest{}, v2err
		}
		if len(post) > 0 {
			return V2Manifest{}, &V2Error{Diags: append(V2Diagnostics(nil), post...)}
		}
		return V2Manifest{}, err
	}
	// Publication barrier: capture the after-state snapshot
	// BEFORE writing the candidate bytes to disk. Any failure
	// here aborts publication.
	callerAfter := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseAfter)
	if !callerAfter.Available {
		// The after snapshot must be fail-closed: if we
		// cannot confirm the caller state after the run,
		// we MUST NOT claim clean success. The diagnostics
		// are merged into a typed V2Error so the CLI can
		// render the exact cause. The candidate manifest
		// is intentionally not written.
		return V2Manifest{}, &V2Error{Diags: callerAfter.Diagnostics}
	}
	if post := callerBefore.State.Diff(callerAfter.State); len(post) > 0 {
		// Drift detected between before and after. The
		// candidate manifest is intentionally not written.
		return V2Manifest{}, &V2Error{Diags: post}
	}
	// Publication barrier exit: atomic write happens ONLY
	// after the after-state authority passes. This is the
	// final success-side mutation.
	if err := AtomicWriteV2Manifest(req.ManifestOutput, candidate.ManifestBytes); err != nil {
		return V2Manifest{}, err
	}
	return candidate.Manifest, nil
}

// mergeAfterDiagnostics returns diagnostics appropriate for the
// supplied after-state snapshot. It is used on the inner-failure
// path so callers always see caller-state corruption even when
// the inner runner already failed.
//
// Behavior:
//   - snapshot unavailable: emit the snapshot's availability
//     diagnostics so the caller knows the after-observation
//     itself failed
//   - snapshot available but drifted from the before snapshot:
//     emit the drift diagnostics in canonical Diff order
//   - snapshot available with no drift: emit nil so the caller
//     does not synthesise diagnostics for a healthy after
//     observation
//
// The drift comparison requires the matching before-state. When
// the caller has not supplied one (e.g. the runner entry point
// before any caller snapshot was taken) the function returns
// nil rather than guess.
func mergeAfterDiagnostics(after v2CallerStateSnapshot) V2Diagnostics {
	if !after.Available {
		return after.Diagnostics
	}
	return nil
}

// mergeAfterWithBeforeDiagnostics returns the diagnostics that
// should be appended to an inner-failure error. It is the
// fail-closed companion of mergeAfterDiagnostics: it ALWAYS
// emits the supplied after-observation diagnostics plus the
// before-vs-after drift diagnostics in canonical Diff order, so
// an inner failure that coincides with caller-state drift is
// surfaced to the CLI.
//
// Diagnostic order is deterministic:
//
//  1. caller-after availability diagnostics, if any
//  2. caller-before vs caller-after drift diagnostics
func mergeAfterWithBeforeDiagnostics(before, after v2CallerStateSnapshot) V2Diagnostics {
	var out V2Diagnostics
	if !after.Available {
		out = append(out, after.Diagnostics...)
		return out
	}
	out = append(out, before.State.Diff(after.State)...)
	return out
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
