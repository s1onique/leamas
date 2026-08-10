// SPDX-License-Identifier: Apache-2.0

// closure_evidence_publication_orchestrator.go owns the
// end-to-end wiring from the V2 runner into the B2 publication
// barrier and the B3 durable publication authority.
//
// The orchestrator is the single entry point that:
//
//  1. runs the V2 runner and obtains the typed V2ExecutionObservation
//     (the runner's authoritative execution record);
//  2. derives the B2 CandidateInputs from the observation alone
//     (no caller-supplied evidence bundle);
//  3. runs the B2 publication barrier
//     (evidence.PrepareClosureEvidenceForPublication) so the
//     typed PublicationCandidate is the only object the B3
//     publisher ever sees;
//  4. opens the destination parent once via os.OpenRoot and
//     publishes the JSON + sidecar pair through the B3
//     authority (no V2 manifest shortcut, no caller-supplied
//     arbitrary bytes).
//
// On any inner / barrier / publication failure, the function
// returns a typed error and the partial state reported by the
// authority. The orchestrator never returns a V2Manifest with
// side effects on disk that the caller cannot observe.
//
// R2 invariant: the orchestrator does not accept a
// `CandidateInputs` bundle from the caller. The bundle is
// derived from the V2ExecutionObservation so a caller cannot
// smuggle a fabricated COMPLETE input past the B2 barrier.
package closure

import (
	"context"
	"fmt"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// V2ExecutionObservation is the authoritative execution record
// the V2 runner produces. Every field the orchestrator needs
// to build the B2 CandidateInputs MUST come from this struct;
// the orchestrator MUST NOT accept a complete CandidateInputs
// bundle from the caller.
type V2ExecutionObservation struct {
	// V2Manifest is the runner's own typed result.
	V2Manifest V2Manifest
	// Runtime, Results, Gate, Binary, CallerBefore, CallerAfter,
	// Cleanup are the authoritative observations.
	Runtime       evidence.RuntimeAuthority
	Results       []evidence.CheckResult
	Gate          evidence.GateAuthority
	Binary        evidence.BinaryAuthority
	CallerBefore  evidence.CallerStateSnapshot
	CallerAfter   evidence.CallerStateSnapshot
	Cleanup       evidence.CleanupAuthority
}

// EvidencePublicationOrchestrator is the B2+B3 wiring.
type EvidencePublicationOrchestrator struct {
	Runner              *V2OrchestratorHandle
	RepositoryRoot      string
	EvidenceDestination string
	Worktrees           []CanonicalWorktree
}

// V2OrchestratorHandle is the small surface the orchestrator
// needs from the V2 runner. The default wiring uses
// `RunClosureProtocolV2WithBinary`.
type V2OrchestratorHandle struct {
	// Run returns a V2ExecutionObservation. Production
	// wiring delegates to a runner that converts its
	// internal state into the typed observation.
	Run func(ctx context.Context, req V2Request, identity V2BinaryIdentity) (V2ExecutionObservation, error)
}

// deriveInputsFromObservation builds the B2 CandidateInputs
// from the runner's authoritative observation. It is the only
// path through which B2 inputs enter the orchestrator. The
// canonical B2 candidate builder re-derives the expected
// check set from `Runtime.PlanBytes` so the orchestrator
// intentionally leaves the `Plan` field zero — a caller
// cannot smuggle a fabricated check list past the barrier.
func deriveInputsFromObservation(obs V2ExecutionObservation) evidence.CandidateInputs {
	return evidence.CandidateInputs{
		Runtime:      obs.Runtime,
		Results:      obs.Results,
		Gate:         obs.Gate,
		Binary:       obs.Binary,
		CallerBefore: obs.CallerBefore,
		CallerAfter:  obs.CallerAfter,
		Cleanup:      obs.Cleanup,
	}
}

// PublishEvidence runs the V2 runner, derives the B2 candidate
// from the runner's authoritative observation, crosses the B2
// barrier, and publishes the durable pair.
//
// The function is total: every error path returns a typed
// error. The B3 publication authority reports the partial
// state to the caller via `result` even on failure.
func (o *EvidencePublicationOrchestrator) PublishEvidence(ctx context.Context, req V2Request, identity V2BinaryIdentity) (V2Manifest, EvidencePublicationResult, error) {
	if o == nil || o.Runner == nil || o.Runner.Run == nil {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("orchestrator: missing runner")
	}
	if o.RepositoryRoot == "" {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("orchestrator: repository root is required")
	}
	if o.EvidenceDestination == "" {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("orchestrator: evidence destination is required")
	}
	obs, err := o.Runner.Run(ctx, req, identity)
	if err != nil {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, err
	}
	// The B2 candidate inputs are derived from the runner's
	// authoritative observation. The caller cannot supply a
	// complete bundle; the orchestrator's B2 inputs come from
	// the runner's result alone.
	inputs := deriveInputsFromObservation(obs)
	candidate := evidence.BuildClosureEvidenceCandidate(inputs)
	prepared, err := evidence.PrepareClosureEvidenceForPublication(candidate)
	if err != nil {
		return obs.V2Manifest, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("B2 publication barrier refused candidate: %w", err)
	}
	auth, err := PrepareEvidencePublication(o.RepositoryRoot, o.EvidenceDestination, o.Worktrees)
	if err != nil {
		return obs.V2Manifest, EvidencePublicationResult{State: EvidencePublicationNotPublished}, err
	}
	defer auth.Close()
	result := auth.Publish(prepared)
	if result.Err != nil {
		return obs.V2Manifest, result, result.Err
	}
	return obs.V2Manifest, result, nil
}
