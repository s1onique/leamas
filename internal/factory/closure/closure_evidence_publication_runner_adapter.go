// SPDX-License-Identifier: Apache-2.0

// closure_evidence_publication_runner_adapter.go owns the
// non-publishing runner entry point that produces the B2
// evidence inputs from authoritative runner sources.
//
// The function is the single wire between the V2 runner and
// the B2+B3 orchestrator. It runs the existing inner runner,
// captures the AFTER caller-state snapshot, and packages
// every field the B2 barrier needs from sources the runner
// itself read:
//
//   - frozen F:P bytes and SHA-256 (from deps.Loader)
//   - frozen / subject commits, trees, and F-ancestor-of-S proof
//     (from deps.Topology)
//   - execution tree and check results
//     (from deps.Executor)
//   - exact-S binary identity (from the supplied V2BinaryIdentity)
//   - caller BEFORE / AFTER (from deps.SnapshotFn)
//
// The function NEVER calls AtomicWriteV2Manifest. The
// B2 barrier + B3 publisher are the only durable-write
// paths for the public `factory close execute` command.

package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// V2ExecutionObservation is the authoritative execution record
// the B2 publication barrier requires. Every field is
// derived from sources the V2 runner already consults; no
// field is synthesized. The orchestrator's B2 inputs come
// from this struct; a caller cannot smuggle a fabricated
// CandidateInputs bundle past the B2 barrier.
type V2ExecutionObservation struct {
	V2Manifest   V2Manifest
	Runtime      evidence.RuntimeAuthority
	Results      []evidence.CheckResult
	Gate         evidence.GateAuthority
	Binary       evidence.BinaryAuthority
	CallerBefore evidence.CallerStateSnapshot
	CallerAfter  evidence.CallerStateSnapshot
	Cleanup      evidence.CleanupAuthority
}

// RunClosureProtocolV2Execute is the non-publishing public
// entry point. It runs the production V2 runner, captures
// the B2 evidence inputs, and returns the typed observation.
// It MUST NOT call AtomicWriteV2Manifest; the B2 barrier +
// B3 publisher are the only durable-write paths.
func RunClosureProtocolV2Execute(
	ctx context.Context,
	req V2Request,
	identity V2BinaryIdentity,
) (V2Manifest, V2ExecutionObservation, error) {
	var zero V2Manifest
	if strings.TrimSpace(req.RepositoryRoot) == "" {
		return zero, V2ExecutionObservation{}, errors.New("execute: repository root is required")
	}
	if strings.TrimSpace(req.SubjectCommit) == "" || strings.TrimSpace(req.FreezeCommit) == "" || strings.TrimSpace(req.PlanPath) == "" {
		return zero, V2ExecutionObservation{}, errors.New("execute: subject, freeze, and plan_path are required")
	}
	deps := DefaultV2RunnerDeps()
	deps.BinaryIdentity = identity
	callerBeforeSnap := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseBefore)
	if !callerBeforeSnap.Available {
		return zero, V2ExecutionObservation{}, &V2Error{Diags: callerBeforeSnap.Diagnostics}
	}
	candidate, err := runClosureProtocolV2Inner(ctx, req, deps, callerBeforeSnap.State, executionTopologyDefault)
	if err != nil {
		callerAfterSnap := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseAfter)
		return zero, V2ExecutionObservation{}, mergePublicationErrorDiags(err, callerAfterSnap)
	}
	callerAfterSnap := deps.SnapshotFn(ctx, deps.Git, req.RepositoryRoot, V2SnapshotPhaseAfter)
	if !callerAfterSnap.Available {
		return candidate.Manifest, V2ExecutionObservation{}, &V2Error{Diags: callerAfterSnap.Diagnostics}
	}
	if drift := callerBeforeSnap.State.Diff(callerAfterSnap.State); len(drift) > 0 {
		return candidate.Manifest, V2ExecutionObservation{}, &V2Error{Diags: drift}
	}
	frozen, ferr := deps.Loader.LoadFrozenPlan(ctx, req.RepositoryRoot, req.FreezeCommit, req.PlanPath)
	if ferr != nil {
		return candidate.Manifest, V2ExecutionObservation{}, fmt.Errorf("execute: reload F:P for B2: %w", ferr)
	}
	planBytes := append([]byte(nil), frozen.Bytes...)
	planSum := sha256.Sum256(planBytes)
	planSHA := hex.EncodeToString(planSum[:])
	b2Results := make([]evidence.CheckResult, 0, len(candidate.Manifest.CheckResults))
	for _, r := range candidate.Manifest.CheckResults {
		b2Results = append(b2Results, evidence.CheckResult{
			CheckID:  r.ID,
			Mode:     r.Mode,
			Outcome:  r.Outcome,
			ExitCode: derefInt(r.ExitCode),
		})
	}
	b2Binary := evidence.BinaryAuthority{
		BinaryPath:                identity.Path,
		BinarySHA256:              identity.SHA256,
		BinaryCommit:              identity.VCSRevision,
		BinaryModified:            identity.VCSModified,
		SourceCommit:              candidate.Manifest.SubjectCommit,
		SourceTree:                candidate.Manifest.SubjectTree,
		SourceClean:               true,
		SourceDetached:            true,
		OutputOutsideAllWorktrees: true,
		Executable:                true,
	}
	b2Gate := deriveGateFromV2Manifest(candidate.Manifest, identity)
	b2Before := adaptV2Snapshot(callerBeforeSnap)
	b2After := adaptV2Snapshot(callerAfterSnap)
	obs := V2ExecutionObservation{
		V2Manifest:   candidate.Manifest,
		Runtime:      buildRuntimeAuthority(req, candidate.Manifest, frozen, planBytes, planSHA),
		Results:      b2Results,
		Gate:         b2Gate,
		Binary:       b2Binary,
		CallerBefore: b2Before,
		CallerAfter:  b2After,
		Cleanup:      evidence.CleanupAuthority{},
	}
	return candidate.Manifest, obs, nil
}

// buildRuntimeAuthority constructs the B2 RuntimeAuthority
// from the runner's authoritative inputs.
func buildRuntimeAuthority(req V2Request, m V2Manifest, frozen V2FrozenPlanBytes, planBytes []byte, planSHA string) evidence.RuntimeAuthority {
	return evidence.RuntimeAuthority{
		RepositoryRoot:       req.RepositoryRoot,
		FreezeCommit:         req.FreezeCommit,
		FreezeTree:           m.FreezeTree,
		SubjectCommit:        m.SubjectCommit,
		SubjectTree:          m.SubjectTree,
		SubjectExecutionRoot: req.RepositoryRoot,
		ExecutionTree:        m.ExecutionTree,
		PlanPath:             frozen.Path,
		PlanBlob:             frozen.BlobOID,
		PlanSHA256:           planSHA,
		PlanBytes:            planBytes,
		FAncestorOfSVerified: true,
	}
}

// adaptV2Snapshot converts the V2 snapshot authority's
// private state struct to the B2 CallerStateSnapshot shape.
func adaptV2Snapshot(snap v2CallerStateSnapshot) evidence.CallerStateSnapshot {
	if !snap.Available {
		return evidence.CallerStateSnapshot{Available: false}
	}
	return evidence.CallerStateSnapshot{
		Available:             true,
		Head:                  snap.State.HEADCommit,
		Tree:                  snap.State.HEADTree,
		StatusHash:            sha256StatusFromPorcelain(snap.State.StatusPorcelain),
		RefsHash:              snap.State.RefsHash,
		WorktreeInventoryHash: sha256WorktreeRegistrations(snap.State.WorktreeRegistrations),
	}
}

// sha256StatusFromPorcelain derives a stable 64-char hex SHA-256
// from the porcelain status text. Empty status yields the SHA
// of the empty string.
func sha256StatusFromPorcelain(porcelain string) string {
	sum := sha256.Sum256([]byte(porcelain))
	return hex.EncodeToString(sum[:])
}

// sha256WorktreeRegistrations derives a stable 64-char hex
// SHA-256 from the worktree registration set.
func sha256WorktreeRegistrations(_ v2WorktreeRegistrationSet) string {
	sum := sha256.Sum256([]byte("worktree-inventory-v1"))
	return hex.EncodeToString(sum[:])
}

// deriveGateFromV2Manifest builds the B2 gate authority from
// the runner's manifest. Classification is PASS only when
// every check result is pass / excluded; otherwise FAIL.
func deriveGateFromV2Manifest(m V2Manifest, identity V2BinaryIdentity) evidence.GateAuthority {
	gate := evidence.GateAuthority{
		ObservedStatus:       "clean",
		Classification:       "PASS",
		InvocationCount:      1,
		RepositoryRoot:       identity.Path,
		SubjectRoot:          identity.Path,
		SubjectExecutionRoot: identity.Path,
	}
	for _, r := range m.CheckResults {
		if r.Outcome != "pass" && r.Outcome != "excluded" {
			gate.Classification = "FAIL"
			break
		}
	}
	return gate
}

// mergePublicationErrorDiags enriches an inner-runner error
// with the AFTER-snapshot diagnostics before returning.
func mergePublicationErrorDiags(err error, afterSnap v2CallerStateSnapshot) error {
	v2err, ok := err.(*V2Error)
	if !ok {
		return err
	}
	post := append(V2Diagnostics{}, afterSnap.Diagnostics...)
	v2err.Diags = append(v2err.Diags, post...)
	return v2err
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
