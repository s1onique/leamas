// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_orchestrator.go implements the single
// end-to-end entry point required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-CLI-TAG-STATE01
// (Phase 1 + Phase 2 + Phase 5 + Phase 6).
//
// The orchestrator composes the foundation, topology, and
// manifest modules into one canonical V2ClosureVerification:
//
//  1. Run ValidateV2ClosureVerifyRequest on the typed
//     request. Any diagnostic short-circuits the run with a
//     non-empty Diagnostics slice and Valid=false.
//  2. Bind the repository authority to RepositoryRoot and
//     enforce the SHA-1 object-format policy.
//  3. Resolve the topology (S < F < C) and capture the
//     subject / freeze / closure tree OIDs.
//  4. Resolve the frozen-plan authority at F:P and the
//     committed-manifest authority at C:M.
//  5. Compute the optional mutable-manifest assertion when
//     the request supplies bytes.
//  6. Resolve the optional annotated-tag assertion when
//     the request supplies --expected-tag.
//  7. Verify the manifest identity / bijection / success
//     against the topology anchor and the frozen-plan
//     inventory.
//  8. If --capture-caller-state is set (test surface), take
//     an after-capture and run CheckReadOnly to prove the
//     verifier was non-mutating.
//
// The result is fed into NewV2ClosureVerification which
// computes the canonical Valid flag from the four required
// booleans and the diagnostics slice.
//
// The orchestrator is the ONLY public entry the CLI uses to
// reach the verifier; tests assert on the function's
// observable behaviour rather than on the internal helpers.

import "context"

// V2VerifierOrchestrator composes every verifier phase
// behind a single deterministic entry point. The type
// holds the resolvers so production wiring supplies one
// orchestrator per verifier invocation, while tests can
// swap individual resolver fields to exercise specific
// rejection paths.
type V2VerifierOrchestrator struct {
	TopologyResolver V2ClosureTopologyResolver
	ManifestVerifier V2ManifestIdentityVerifier
}

// NewV2VerifierOrchestrator constructs a production
// orchestrator wired to the canonical resolvers.
func NewV2VerifierOrchestrator() V2VerifierOrchestrator {
	return V2VerifierOrchestrator{
		TopologyResolver: NewV2ClosureTopologyResolver(),
		ManifestVerifier: NewV2ManifestIdentityVerifier(),
	}
}

// V2RunRequest is the orchestrator's input bundle. The
// CaptureCallerState flag controls the optional Phase 5
// before/after read-only check used by tests; production
// CLI invocations leave it false.
type V2RunRequest struct {
	Request            V2ClosureVerifyRequest
	CaptureCallerState bool
}

// V2RunResult is the orchestrator's outcome. It carries the
// closed verification result plus the before/after state
// snapshots when CaptureCallerState is true. The fields are
// always populated; callers may ignore the snapshots when
// the capture is disabled.
type V2RunResult struct {
	Verification V2ClosureVerification
	StateBefore  V2CallerStateSnapshot
	StateAfter   V2CallerStateSnapshot
	StateDiags   V2VerifierDiagnostics
}

// Run executes every phase of the verifier end-to-end. The
// function returns a non-empty V2RunResult even on
// failure; only Valid distinguishes pass from fail. The
// function never panics, never panics on a malformed
// request, and never writes outside the supplied report
// sink.
//
// When state capture is enabled, the function snapshots
// caller state once before the verification sequence and
// once after, then computes CheckReadOnly; a state
// mutation produces a state_mutation_detected diagnostic
// but NEVER gates verification success — the verifier
// itself remains read-only regardless of the capture
// result, and a capture failure cannot cause false
// rejection.
func (o V2VerifierOrchestrator) Run(
	ctx context.Context,
	authority V2ClosureGitAuthority,
	runReq V2RunRequest,
) V2RunResult {
	result := V2RunResult{}

	// Phase A: optional caller-state capture BEFORE any
	// verification (so a fast topology miss does not
	// change the reported state).
	if runReq.CaptureCallerState && authority != nil {
		result.StateBefore = CaptureV2CallerState(ctx, authority)
	}

	// Phase B: request validation. Any non-empty diagnostic
	// short-circuits the run with Invalid result.
	if requestDiags := ValidateV2ClosureVerifyRequest(runReq.Request); len(requestDiags) > 0 {
		result.Verification = buildInvalidResultForRequest(runReq.Request, requestDiags)
		result.StateAfter, result.StateDiags = captureAfterIfNeeded(ctx, authority, runReq.CaptureCallerState, result.StateBefore)
		return result
	}

	// Phase C: object-format enforcement. A failure here
	// short-circuits because no OID validation can run on
	// a sha256 repository.
	if authority == nil {
		diags := V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierRepositoryUnavailable,
			"git authority is nil",
		)}
		result.Verification = buildInvalidResultForRequest(runReq.Request, diags)
		result.StateAfter, result.StateDiags = captureAfterIfNeeded(ctx, authority, runReq.CaptureCallerState, result.StateBefore)
		return result
	}
	if err := EnforceV2VerifierObjectFormatPolicy(authority); err != nil {
		diags := collectV2VerifierDiags(err)
		result.Verification = buildInvalidResultForRequest(runReq.Request, diags)
		result.StateAfter, result.StateDiags = captureAfterIfNeeded(ctx, authority, runReq.CaptureCallerState, result.StateBefore)
		return result
	}

	// Phase D: topology.
	var topologyFacts V2ClosureTopologyFacts
	if o.TopologyResolver != nil {
		topologyFacts, _ = o.TopologyResolver.ResolveTopology(ctx, authority, runReq.Request)
	}

	// Phase E: frozen-plan authority at F:P.
	frozenPlan, err := ResolveV2FrozenPlanAuthority(ctx, authority, runReq.Request.FreezeCommit, runReq.Request.PlanPath)
	if err != nil {
		result.Verification = assembleVerification(
			runReq.Request,
			topologyFacts,
			frozenPlan,
			V2CommittedManifestAuthority{},
			V2OptionalManifestAssertion{},
			V2VerifierTagAssertion{},
			V2ManifestIdentityFacts{},
			collectV2VerifierDiags(err),
		)
		result.StateAfter, result.StateDiags = captureAfterIfNeeded(ctx, authority, runReq.CaptureCallerState, result.StateBefore)
		return result
	}

	if !topologyFacts.Relation.IsAccepted() {
		topologyDiags := topologyFacts.Diagnostics
		if len(topologyDiags) == 0 && frozenPlan.BlobOID != "" {
			// Topology accepted the S/F/C triple but
			// the frozen plan still failed; surface
			// the frozen-plan diagnostics.
			topologyDiags = frozenPlan.Diagnostics
		} else if len(topologyDiags) == 0 {
			topologyDiags = frozenPlan.Diagnostics
		}
		result.Verification = assembleVerification(
			runReq.Request,
			topologyFacts,
			frozenPlan,
			V2CommittedManifestAuthority{},
			V2OptionalManifestAssertion{},
			V2VerifierTagAssertion{},
			V2ManifestIdentityFacts{},
			topologyDiags,
		)
		result.StateAfter, result.StateDiags = captureAfterIfNeeded(ctx, authority, runReq.CaptureCallerState, result.StateBefore)
		return result
	}

	// Phase F: committed-manifest authority at C:M.
	committedManifest, err := ResolveV2CommittedManifestAuthority(ctx, authority, runReq.Request.ClosureCommit, runReq.Request.ManifestPath)
	if err != nil {
		result.Verification = assembleVerification(
			runReq.Request,
			topologyFacts,
			frozenPlan,
			committedManifest,
			V2OptionalManifestAssertion{},
			V2VerifierTagAssertion{},
			V2ManifestIdentityFacts{},
			collectV2VerifierDiags(err),
		)
		result.StateAfter, result.StateDiags = captureAfterIfNeeded(ctx, authority, runReq.CaptureCallerState, result.StateBefore)
		return result
	}
	if len(committedManifest.Diagnostics) > 0 {
		result.Verification = assembleVerification(
			runReq.Request,
			topologyFacts,
			frozenPlan,
			committedManifest,
			V2OptionalManifestAssertion{},
			V2VerifierTagAssertion{},
			V2ManifestIdentityFacts{},
			committedManifest.Diagnostics,
		)
		result.StateAfter, result.StateDiags = captureAfterIfNeeded(ctx, authority, runReq.CaptureCallerState, result.StateBefore)
		return result
	}
	if len(frozenPlan.Diagnostics) > 0 {
		result.Verification = assembleVerification(
			runReq.Request,
			topologyFacts,
			frozenPlan,
			committedManifest,
			V2OptionalManifestAssertion{},
			V2VerifierTagAssertion{},
			V2ManifestIdentityFacts{},
			frozenPlan.Diagnostics,
		)
		result.StateAfter, result.StateDiags = captureAfterIfNeeded(ctx, authority, runReq.CaptureCallerState, result.StateBefore)
		return result
	}

	// Phase G: optional mutable-manifest assertion.
	optionalAssertion := AssertV2OptionalManifestAssertion(committedManifest, runReq.Request.OptionalManifestAssertion)

	// Phase H: manifest identity / bijection / success.
	var manifestFacts V2ManifestIdentityFacts
	if o.ManifestVerifier != nil {
		manifestFacts, _ = o.ManifestVerifier.VerifyManifestIdentity(
			committedManifest.RawBytes,
			frozenPlan,
			committedManifest,
			topologyFacts.Topology,
		)
	}

	// Phase I: optional annotated-tag assertion. Always
	// keyed on the resolved closure commit so a substituted
	// C never desynchronises the tag wiring.
	var tagAssertion V2VerifierTagAssertion
	var tagMetadataObs V2ClosureTagMetadataObservation
	if runReq.Request.HasExpectedTag() {
		tagAssertion = ResolveV2ClosureTagAssertion(ctx, authority, runReq.Request.ExpectedTagName, topologyFacts.Topology.ClosureCommit)
		// Phase I-2 (CORRECTION02C): when --expected-tag is
		// supplied, read the annotated tag-object bytes and
		// bind the metadata to the externally supplied
		// S/F/C/P/M. The observation is recorded even when
		// the structural assertion rejects the tag, so the
		// caller sees a stable shape regardless of branch.
		if tagAssertion.Annotated && tagAssertion.Found {
			tagMetadataObs = ResolveV2ClosureTagMetadataObservation(
				ctx,
				authority,
				tagAssertion.Target,
				runReq.Request.ExpectedTagName,
				runReq.Request.SubjectCommit,
				runReq.Request.FreezeCommit,
				runReq.Request.ClosureCommit,
				runReq.Request.PlanPath,
				runReq.Request.ManifestPath,
				runReq.Request.ClosureProtocolVersion,
				runReq.Request.PlanContractVersion,
			)
		}
	}

	// Phase J: assemble verdict. The tag-metadata observation
	// only contributes diagnostics when the caller supplied
	// --expected-tag; otherwise the observation is the zero
	// value and contributes nothing.
	combinedDiags := combineDiagnostics(topologyFacts, frozenPlan, committedManifest, optionalAssertion, tagAssertion, manifestFacts, tagMetadataObs)
	result.Verification = assembleVerification(
		runReq.Request,
		topologyFacts,
		frozenPlan,
		committedManifest,
		optionalAssertion,
		tagAssertion,
		manifestFacts,
		combinedDiags,
	)
	result.StateAfter, result.StateDiags = captureAfterIfNeeded(ctx, authority, runReq.CaptureCallerState, result.StateBefore)
	if len(result.StateDiags) > 0 {
		result.Verification.Diagnostics = append(result.Verification.Diagnostics, result.StateDiags...)
	}
	return result
}

// captureAfterIfNeeded optionally captures the caller
// state after a phase has run, then returns the
// after-snapshot plus the CheckReadOnly diagnostics. The
// helper short-circuits when the caller did not opt into
// capture, when authority is nil, or when the before-snapshot
// is zero-valued.
func captureAfterIfNeeded(
	ctx context.Context,
	authority V2ClosureGitAuthority,
	capture bool,
	before V2CallerStateSnapshot,
) (V2CallerStateSnapshot, V2VerifierDiagnostics) {
	if !capture || authority == nil {
		return V2CallerStateSnapshot{}, nil
	}
	after := CaptureV2CallerState(ctx, authority)
	return after, CheckReadOnly(before, after)
}

// combineDiagnostics concatenates the supplied per-phase
// diagnostics slices in deterministic order. The helper is
// the single source of truth for verifier-level
// diagnostics ordering so tests can assert on a closed list.
func combineDiagnostics(
	topology V2ClosureTopologyFacts,
	frozenPlan V2FrozenPlanAuthority,
	committedManifest V2CommittedManifestAuthority,
	optionalAssertion V2OptionalManifestAssertion,
	tagAssertion V2VerifierTagAssertion,
	manifestFacts V2ManifestIdentityFacts,
	tagMetadata V2ClosureTagMetadataObservation,
) V2VerifierDiagnostics {
	combined := V2VerifierDiagnostics{}
	combined = append(combined, topology.Diagnostics...)
	combined = append(combined, frozenPlan.Diagnostics...)
	combined = append(combined, committedManifest.Diagnostics...)
	if optionalAssertion.Supplied {
		combined = append(combined, optionalAssertion.Diagnostics...)
	}
	if tagAssertion.Expected {
		combined = append(combined, tagAssertion.Diagnostics...)
		combined = append(combined, tagMetadata.Diagnostics...)
	}
	combined = append(combined, manifestFacts.Diagnostics...)
	return combined
}

// topologyAccepted reports whether the topology phase
// reached the canonical S < F < C relation.
func topologyAccepted(facts V2ClosureTopologyFacts) bool {
	if len(facts.Diagnostics) > 0 {
		return false
	}
	if facts.Relation == "" {
		return false
	}
	return facts.Relation.IsAccepted()
}

// buildInvalidResultForRequest constructs an Invalid
// verification result from the request fields plus the
// supplied request-level diagnostics. The result keeps the
// subject / freeze / closure strings the caller supplied so
// the CLI can render them in failure messages even when no
// Git observation has happened.
func buildInvalidResultForRequest(req V2ClosureVerifyRequest, requestDiags V2VerifierDiagnostics) V2ClosureVerification {
	build := V2VerificationBuild{
		ClosureProtocolVersion: req.ClosureProtocolVersion,
		PlanContractVersion:    req.PlanContractVersion,
		RepositoryRoot:         req.RepositoryRoot,
		SubjectCommit:          req.SubjectCommit,
		FreezeCommit:           req.FreezeCommit,
		ClosureCommit:          req.ClosureCommit,
		PlanPath:               req.PlanPath,
		ManifestPath:           req.ManifestPath,
		Diagnostics:            requestDiags,
	}
	return NewV2ClosureVerification(build)
}

// assembleVerification combines every per-phase product
// into a single V2ClosureVerification. The verdict booleans
// follow the canonical predicate defined in ACT 3:
//
//	TopologyValid   := relation accepted
//	ManifestValid   := manifest identity valid AND binary
//	                   identity valid AND no committed
//	                   manifest diagnostics
//	ResultSetValid  := bijection valid AND success valid
//
// When --expected-tag is supplied, the verdict requires
// the tag assertion to have passed; otherwise tag
// diagnostics are recorded as informational evidence only.
// When --working-manifest-assertion is supplied, the
// optional mutable-manifest assertion is recorded as
// informational evidence only — a mismatch never overrides
// the C:M binding.
//
// The metadata observation drives the verdict through the
// combinedDiags slice: when --expected-tag is supplied the
// orchestrator adds the metadata diagnostics to the
// combined slice, and a non-empty metadata-mismatch
// diagnostic short-circuits tagValid.
func assembleVerification(
	req V2ClosureVerifyRequest,
	topology V2ClosureTopologyFacts,
	frozenPlan V2FrozenPlanAuthority,
	committedManifest V2CommittedManifestAuthority,
	optionalAssertion V2OptionalManifestAssertion,
	tagAssertion V2VerifierTagAssertion,
	manifestFacts V2ManifestIdentityFacts,
	combinedDiags V2VerifierDiagnostics,
) V2ClosureVerification {

	tagValid := true
	if tagAssertion.Expected {
		tagValid = tagAssertion.Found && tagAssertion.Annotated &&
			tagAssertion.Target == topology.Topology.ClosureCommit &&
			len(tagAssertion.Diagnostics) == 0 &&
			!combinedDiags.HasCode(V2VerifierClosureTagMetadataMismatch)
	}
	_ = optionalAssertion // Optional assertion never gates verdict.

	topologyValid := topologyAccepted(topology) && len(frozenPlan.Diagnostics) == 0
	manifestValid := manifestFacts.ManifestIdentityValid &&
		manifestFacts.BinaryIdentityValid &&
		len(committedManifest.Diagnostics) == 0 &&
		tagValid
	resultSetValid := manifestFacts.BijectionValid && manifestFacts.SuccessValid

	build := V2VerificationBuild{
		ClosureProtocolVersion: req.ClosureProtocolVersion,
		PlanContractVersion:    req.PlanContractVersion,
		RepositoryRoot:         req.RepositoryRoot,
		SubjectCommit:          firstNonEmpty(topology.Topology.SubjectCommit, req.SubjectCommit),
		SubjectTree:            topology.Topology.SubjectTree,
		FreezeCommit:           firstNonEmpty(topology.Topology.FreezeCommit, req.FreezeCommit),
		FreezeTree:             topology.Topology.FreezeTree,
		ClosureCommit:          firstNonEmpty(topology.Topology.ClosureCommit, req.ClosureCommit),
		ClosureTree:            topology.Topology.ClosureTree,
		PlanPath:               firstNonEmpty(frozenPlan.Path, req.PlanPath),
		PlanBlob:               frozenPlan.BlobOID,
		PlanSHA256:             frozenPlan.BlobSHA256,
		ManifestPath:           firstNonEmpty(committedManifest.Path, req.ManifestPath),
		ManifestBlob:           committedManifest.BlobOID,
		ManifestSHA256:         committedManifest.BlobSHA256,
		TopologyValid:          topologyValid,
		ManifestValid:          manifestValid,
		ResultSetValid:         resultSetValid,
		Diagnostics:            combinedDiags,
	}
	return NewV2ClosureVerification(build)
}

// firstNonEmpty returns the first non-empty string from the
// supplied pair. Used to keep the verdict stable when the
// topology resolver echoes the request fields back into the
// typed result.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// collectV2VerifierDiags unwraps a V2VerifierError to its
// diagnostics slice. Plain errors fall back to a wrapped
// generic diagnostic so the caller still gets a typed
// verdict.
func collectV2VerifierDiags(err error) V2VerifierDiagnostics {
	if err == nil {
		return nil
	}
	if v, ok := err.(*V2VerifierError); ok && v != nil {
		return append(V2VerifierDiagnostics{}, v.Diags...)
	}
	return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
		V2VerifierRepositoryUnavailable,
		err.Error(),
	)}
}
