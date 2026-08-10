// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

// V2RunnerDeps captures the dependencies the runner needs.
// The defaults match the production wiring.
type V2RunnerDeps struct {
	Git               gitClient
	Topology          V2TopologyResolver
	Loader            V2FrozenPlanLoader
	Executor          V2SubjectExecutor
	Commands          commandExecutor
	Now               func() time.Time
	BinaryIdentity    V2BinaryIdentity
	SnapshotFn        V2RunnerSnapshotFunc
	CandidateObserver V2CandidateObserver
	// Topology is the internal execution-topology selector.
	// The two public entry points lock this in; the field
	// is part of the unexported deps so external Go callers
	// cannot smuggle the runtime-context mode into the
	// generic V2 runner.
	// (TopologyMode is removed; the runtime-context entry point
	// passes the topology directly into the unexported runner
	// helper so external Go callers cannot smuggle it.)
}

// DefaultV2RunnerDeps returns the production dependency wiring
// using RealGit and the default executors. Callers may override
// any field before invoking RunClosureProtocolV2WithDeps.
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
// the manifest. This is the LEGACY entry point: it uses the
// default topology (V2 = S < F). The runtime-context command
// uses RunClosureProtocolRuntimeContext instead.
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
	return runClosureProtocolV2WithDepsAndTopology(ctx, req, deps, executionTopologyDefault)
}

// runClosureProtocolV2WithDepsAndTopology is the single private
// sealed runner. The two public entry points lock the topology
// via this seam so external Go callers cannot smuggle the
// runtime-context mode into the request surface. There MUST be
// no other orchestration path: validation, dependency defaults,
// caller BEFORE/AFTER, inner-error preservation, drift detection,
// and manifest publication all live here.
func runClosureProtocolV2WithDepsAndTopology(ctx context.Context, req V2Request, deps V2RunnerDeps, topology executionTopology) (V2Manifest, error) {
	if err := ValidateV2Request(req); err != nil {
		return V2Manifest{}, err
	}
	if req.ClosureProtocolVersion != ClosureProtocolV2 {
		return V2Manifest{}, NewV2ErrorWith(V2CodeUnsupportedClosureProtocolVersion,
			fmt.Sprintf("v2 entry point requires closure_protocol_version=%q, got %q", string(ClosureProtocolV2), string(req.ClosureProtocolVersion)),
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
	// Capture before snapshot
	callerBefore := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseBefore)
	if !callerBefore.Available {
		return V2Manifest{}, &V2Error{Diags: callerBefore.Diagnostics}
	}
	candidate, err := runClosureProtocolV2Inner(ctx, req, deps, callerBefore.State, topology)
	if err != nil {
		callerAfter := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseAfter)
		post := mergeAfterWithBeforeDiagnostics(callerBefore, callerAfter)
		if v2err, ok := err.(*V2Error); ok {
			v2err.Diags = append(v2err.Diags, post...)
			return V2Manifest{}, v2err
		}
		inner := buildInnerFallbackDiagnostic(err)
		wrapped := &V2Error{
			Diags: V2Diagnostics{inner},
			Cause: err,
		}
		if len(post) > 0 {
			wrapped.Diags = append(wrapped.Diags, post...)
		}
		return V2Manifest{}, wrapped
	}
	callerAfter := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseAfter)
	if !callerAfter.Available {
		return V2Manifest{}, &V2Error{Diags: callerAfter.Diagnostics}
	}
	if post := callerBefore.State.Diff(callerAfter.State); len(post) > 0 {
		return V2Manifest{}, &V2Error{Diags: post}
	}
	if err := AtomicWriteV2Manifest(req.ManifestOutput, candidate.ManifestBytes); err != nil {
		return V2Manifest{}, err
	}
	return candidate.Manifest, nil
}

// mergeAfterDiagnostics returns diagnostics appropriate for the
// supplied after-state snapshot. It is used on the inner-failure
// path so callers always see caller-state corruption even when
// the inner runner already failed.
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
// before-vs-after drift diagnostics in canonical Diff order.
func mergeAfterWithBeforeDiagnostics(before, after v2CallerStateSnapshot) V2Diagnostics {
	var out V2Diagnostics
	if !after.Available {
		out = append(out, after.Diagnostics...)
		return out
	}
	out = append(out, before.State.Diff(after.State)...)
	return out
}

// buildInnerFallbackDiagnostic builds the deterministic first
// diagnostic representing an inner-failure error that was not
// already a typed *V2Error. The original error is preserved on
// the wrapping V2Error's Cause field so errors.Is and errors.As
// can reach it.
func buildInnerFallbackDiagnostic(err error) V2Diagnostic {
	if err == nil {
		return V2Diagnostic{
			Code:         V2CodeExecutionFailed,
			Message:      "inner runner returned a nil error",
			PropertyName: "inner_failure",
		}
	}
	return V2Diagnostic{
		Code:         V2CodeExecutionFailed,
		Message:      fmt.Sprintf("inner runner returned non-typed error: %s", err.Error()),
		PropertyName: "inner_failure",
		Detail:       err.Error(),
	}
}

// runClosureProtocolV2Inner is implemented in closure_protocol_v2_runner_inner.go.

// EnforceDetachedV2Outputs canonicalises the supplied
// evidence directory, manifest output, and working-plan
// assertion, then verifies each is detached from every
// forbidden root.
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
