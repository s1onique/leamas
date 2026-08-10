// SPDX-License-Identifier: Apache-2.0

// closure_evidence_publication_orchestrator.go owns the
// end-to-end wiring from the V2 runner into the B2 publication
// barrier and the B3 durable publication authority.
//
// The orchestrator is the single entry point that:
//
//  1. runs the V2 runner through a non-publishing entry point
//     that returns the authoritative V2ExecutionObservation;
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
// The orchestrator does NOT accept a CandidateInputs bundle
// from the caller. The bundle is derived from the
// V2ExecutionObservation so a caller cannot smuggle a
// fabricated COMPLETE input past the B2 barrier.
package closure

import (
	"context"
	"fmt"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// V2ExecuteFunc is the non-publishing runner entry point. The
// orchestrator calls Run(ctx, req, identity) to obtain the
// runner's authoritative V2ExecutionObservation.
type V2ExecuteFunc func(ctx context.Context, req V2Request, identity V2BinaryIdentity) (V2Manifest, V2ExecutionObservation, error)

// EvidencePublicationOrchestrator is the B2+B3 wiring.
type EvidencePublicationOrchestrator struct {
	Runner              V2ExecuteFunc
	RepositoryRoot      string
	EvidenceDestination string
	Worktrees           []CanonicalWorktree
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

// PublishEvidence runs the non-publishing runner, derives the
// B2 candidate from the runner's authoritative observation,
// crosses the B2 barrier, and publishes the durable pair.
//
// The function is total: every error path returns a typed
// error. The B3 publication authority reports the partial
// state to the caller via `result` even on failure.
func (o *EvidencePublicationOrchestrator) PublishEvidence(ctx context.Context, req V2Request, identity V2BinaryIdentity) (V2Manifest, EvidencePublicationResult, error) {
	if o == nil || o.Runner == nil {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("orchestrator: missing runner")
	}
	if o.RepositoryRoot == "" {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("orchestrator: repository root is required")
	}
	if o.EvidenceDestination == "" {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("orchestrator: evidence destination is required")
	}
	_, obs, err := o.Runner(ctx, req, identity)
	if err != nil {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, err
	}
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
