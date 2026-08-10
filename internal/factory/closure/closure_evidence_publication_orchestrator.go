// SPDX-License-Identifier: Apache-2.0

// closure_evidence_publication_orchestrator.go owns the
// end-to-end wiring from the V2 runner into the B2 publication
// barrier and the B3 durable publication authority.
//
// The orchestrator is the single entry point that:
//
//  1. runs the existing V2 runner and obtains the typed
//     V2Manifest (the runner's own result; unchanged);
//  2. builds a B2 ClosureEvidence candidate from the supplied
//     inputs (the only public way to cross into B2);
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
package closure

import (
	"context"
	"fmt"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

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
	Run func(ctx context.Context, req V2Request, identity V2BinaryIdentity) (V2Manifest, error)
}

// PublishEvidence runs the V2 runner, builds the B2 candidate,
// crosses the B2 barrier, and publishes the durable pair.
//
// The function is total: every error path returns a typed
// error. The B3 publication authority reports the partial
// state to the caller via `result` even on failure.
func (o *EvidencePublicationOrchestrator) PublishEvidence(ctx context.Context, req V2Request, identity V2BinaryIdentity, inputs evidence.CandidateInputs) (V2Manifest, EvidencePublicationResult, error) {
	if o == nil || o.Runner == nil || o.Runner.Run == nil {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("orchestrator: missing runner")
	}
	if o.RepositoryRoot == "" {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("orchestrator: repository root is required")
	}
	if o.EvidenceDestination == "" {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("orchestrator: evidence destination is required")
	}
	manifest, err := o.Runner.Run(ctx, req, identity)
	if err != nil {
		return V2Manifest{}, EvidencePublicationResult{State: EvidencePublicationNotPublished}, err
	}
	candidate := evidence.BuildClosureEvidenceCandidate(inputs)
	prepared, err := evidence.PrepareClosureEvidenceForPublication(candidate)
	if err != nil {
		return manifest, EvidencePublicationResult{State: EvidencePublicationNotPublished}, fmt.Errorf("B2 publication barrier refused candidate: %w", err)
	}
	auth, err := PrepareEvidencePublication(o.RepositoryRoot, o.EvidenceDestination, o.Worktrees)
	if err != nil {
		return manifest, EvidencePublicationResult{State: EvidencePublicationNotPublished}, err
	}
	defer auth.Close()
	result := auth.Publish(prepared)
	if result.Err != nil {
		return manifest, result, result.Err
	}
	return manifest, result, nil
}
