// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_candidate.go owns the unpublished v2
// manifest candidate. The candidate is the typed contract
// between the inner runner (which executes and constructs the
// manifest) and the outer runner (which publishes the manifest
// only after the caller-state authority passes).
//
// Splitting construction from publication is the core barrier
// introduced by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2A:
// no manifest bytes may reach the on-disk path before the
// after-state observation succeeds and proves no caller drift
// or leaked worktree registration.
//
// Splitting this from closure_protocol_v2_runner_inner.go
// keeps the inner runner under the LLM-friendly 400-line
// threshold while preserving the single closure over the
// descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"context"
	"sync/atomic"
)

// V2SnapshotPhase names the position in the runner sequence
// at which the caller-state snapshot was captured. Tests use
// the explicit phase identifier to fail a specific snapshot
// without relying on call-count approximations.
//
// V2SnapshotPhaseBefore is captured before the inner execution.
//
// V2SnapshotPhaseAfter is captured after the inner execution
// returns; the outer runner uses it to refuse to publish the
// manifest when the after observation fails.
type V2SnapshotPhase string

const (
	// V2SnapshotPhaseBefore is the before-snapshot phase.
	V2SnapshotPhaseBefore V2SnapshotPhase = "before"
	// V2SnapshotPhaseAfter is the after-snapshot phase.
	V2SnapshotPhaseAfter V2SnapshotPhase = "after"
)

// v2RunCandidate is the unpublished result of the inner
// runner. It carries both the validated manifest and the
// deterministic byte rendering that the outer runner must
// publish atomically only after the after-state authority
// passes.
//
// Manifest holds the typed record consumed by downstream v2
// manifest decoders. ManifestBytes holds the JSON bytes that
// will be written verbatim to req.ManifestOutput by the outer
// runner. The two MUST agree byte-for-byte; the outer
// publication barrier writes ManifestBytes and returns the
// Manifest as the success-side return value.
type v2RunCandidate struct {
	Manifest      V2Manifest
	ManifestBytes []byte
}

// V2RunnerSnapshotFunc captures the caller-state observation
// for a given snapshot phase. Production wiring calls
// snapshotCallerState(ctx, git, repoRoot). Tests inject a
// fake that returns a non-Available v2CallerStateSnapshot
// when phase == V2SnapshotPhaseAfter to prove the outer runner
// refuses to publish the manifest.
//
// The signature intentionally mirrors snapshotCallerState
// plus the phase identifier so tests can target the exact
// boundary without resorting to command-count approximations
// over the underlying git client.
type V2RunnerSnapshotFunc func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot

// defaultV2RunnerSnapshotFunc routes the v2 runner snapshot
// observation through the production snapshotCallerState.
// Tests may override deps.SnapshotFn to target the exact
// snapshot phase.
func defaultV2RunnerSnapshotFunc(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
	_ = phase
	return snapshotCallerState(ctx, git, repoRoot)
}

// V2CandidateObserver is the invocation-local observer the
// runner calls when the inner authority constructs a candidate
// manifest. Tests can inject a counting observer to prove the
// candidate was constructed exactly once. Production callers
// may pass nil and the runner skips the observer call.
//
// CandidateConstructed is invoked once per successful inner
// run, after V2ManifestRender has produced the deterministic
// bytes. The observer MUST NOT mutate the candidate.
type V2CandidateObserver interface {
	CandidateConstructed(V2Manifest, []byte)
}

// noopCandidateObserver is the production-default observer
// used when no candidate observer is injected.
type noopCandidateObserver struct{}

func (noopCandidateObserver) CandidateConstructed(V2Manifest, []byte) {}

// countingCandidateObserver is a test observer that records
// every CandidateConstructed call. Tests assert the exact
// number of constructions and that the bytes were produced.
type countingCandidateObserver struct {
	calls     int32
	last      V2Manifest
	lastBytes []byte
}

// CandidateConstructed records the construction event.
func (o *countingCandidateObserver) CandidateConstructed(m V2Manifest, b []byte) {
	atomic.AddInt32(&o.calls, 1)
	o.last = m
	o.lastBytes = b
}

// Calls returns the cumulative count of construction events.
func (o *countingCandidateObserver) Calls() int {
	return int(atomic.LoadInt32(&o.calls))
}
