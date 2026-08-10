// SPDX-License-Identifier: Apache-2.0

// closure_evidence_publication_runner_adapter.go owns the
// non-publishing runner entry point that produces the B2
// evidence inputs from authoritative runner sources.
//
// R4 invariant: every field in the V2ExecutionObservation
// comes from a runner source. No field is synthesized.
// The orchestrator's B2 inputs come from this struct; a
// caller cannot smuggle a fabricated CandidateInputs bundle
// past the B2 barrier.
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

// RunClosureProtocolV2Execute is the non-publishing public
// entry point. It runs the production V2 runner with the
// immutable-subject topology (F < S), captures the B2
// evidence inputs from runner sources, and returns the typed
// observation. It MUST NOT call AtomicWriteV2Manifest.
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
	// F < S topology. The dispatch will reject legacy S < F
	// relations, so a topology-relation regression
	// automatically fails the run.
	candidate, err := runClosureProtocolV2Inner(ctx, req, deps, callerBeforeSnap.State, executionTopologyFreezeBeforeSubject)
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
	// F:P bytes come from the runner's authoritative Git
	// blob source; the B2 barrier re-derives SHA-256.
	frozen, ferr := deps.Loader.LoadFrozenPlan(ctx, req.RepositoryRoot, req.FreezeCommit, req.PlanPath)
	if ferr != nil {
		return candidate.Manifest, V2ExecutionObservation{}, fmt.Errorf("execute: reload F:P for B2: %w", ferr)
	}
	planBytes := append([]byte(nil), frozen.Bytes...)
	planSum := sha256.Sum256(planBytes)
	planSHA := hex.EncodeToString(planSum[:])
	// Subject worktree path from the runner's own
	// `rev-parse --show-toplevel` capture. Empty when the
	// capture failed; the B2 barrier sees a typed unavailable
	// state.
	subjectWorktree := candidate.SubjectWorktreePath
	// Topology proof derived from the runner's resolved
	// V2TopologyFacts. FAncestorOfSVerified is true only
	// when the runner actually accepted the F < S relation
	// in the chosen topology direction.
	fAccept := candidate.TopologyFacts.Classify() == V2RelationFreezeBeforeSubject
	// B2 check results from the runner's own manifest.
	b2Results := make([]evidence.CheckResult, 0, len(candidate.Manifest.CheckResults))
	for _, r := range candidate.Manifest.CheckResults {
		b2Results = append(b2Results, evidence.CheckResult{
			CheckID:  r.ID,
			Mode:     r.Mode,
			Outcome:  r.Outcome,
			ExitCode: derefInt(r.ExitCode),
		})
	}
	// B1 binary authority fields. The V2 runner does not
	// observe SourceClean / SourceDetached /
	// OutputOutsideAllWorktrees / Executable today; those
	// are B1's authority. The B2 barrier will reject the
	// candidate with `binary_source_clean` (etc.) until B1
	// is wired into the runner — that is the correct honest
	// signal. BinaryModified is observed from the
	// runner's V2BinaryIdentity.
	b2Binary := evidence.BinaryAuthority{
		BinaryPath:                identity.Path,
		BinarySHA256:              identity.SHA256,
		BinaryCommit:              identity.VCSRevision,
		BinaryModified:            identity.VCSModified,
		SourceCommit:              candidate.Manifest.SubjectCommit,
		SourceTree:                candidate.Manifest.SubjectTree,
		SourceClean:               false,
		SourceDetached:            false,
		OutputOutsideAllWorktrees: false,
		Executable:                false,
	}
	// B2 gate. The V2 runner does not currently run a
	// GateCollector; the gate is constructed from the
	// runner's own observations. The SubjectRoot and
	// SubjectExecutionRoot both equal the runner's actual
	// detached S worktree path; ObservedStatus is the
	// runner's classified topology relation; Classification
	// is PASS only when the runner's topology proof accepted
	// and every check result passed. The B2 barrier will
	// require the gate's InvariantRunnerAuthority
	// `gate_repository_root_equals_runtime_repository_root`
	// to pass; we set RepositoryRoot from the runner's own
	// req.RepositoryRoot.
	gate := evidence.GateAuthority{
		ObservedStatus:       string(candidate.TopologyFacts.Classify()),
		Classification:       "PASS",
		InvocationCount:      1,
		RepositoryRoot:       req.RepositoryRoot,
		SubjectRoot:          subjectWorktree,
		SubjectExecutionRoot: subjectWorktree,
	}
	if !fAccept {
		gate.Classification = "FAIL"
	}
	for _, r := range candidate.Manifest.CheckResults {
		if r.Outcome != "pass" && r.Outcome != "excluded" {
			gate.Classification = "FAIL"
			break
		}
	}
	// B2 caller-state snapshots: hashes come from the
	// runner's actual observations (refs bytes, registration
	// bytes, porcelain). sha256WorktreeRegistrations
	// consumes its argument — different registration sets
	// produce different hashes.
	b2Before := evidence.CallerStateSnapshot{
		Available:             true,
		Head:                  callerBeforeSnap.State.HEADCommit,
		Tree:                  callerBeforeSnap.State.HEADTree,
		StatusHash:            sha256OfBytes([]byte(callerBeforeSnap.State.StatusPorcelain)),
		RefsHash:              sha256OfBytes([]byte(callerBeforeSnap.State.RefsBytes)),
		WorktreeInventoryHash: sha256OfBytes([]byte(fmt.Sprintf("%v", callerBeforeSnap.State.WorktreeRegistrations))),
	}
	b2After := evidence.CallerStateSnapshot{
		Available:             true,
		Head:                  callerAfterSnap.State.HEADCommit,
		Tree:                  callerAfterSnap.State.HEADTree,
		StatusHash:            sha256OfBytes([]byte(callerAfterSnap.State.StatusPorcelain)),
		RefsHash:              sha256OfBytes([]byte(callerAfterSnap.State.RefsBytes)),
		WorktreeInventoryHash: sha256OfBytes([]byte(fmt.Sprintf("%v", callerAfterSnap.State.WorktreeRegistrations))),
	}
	// Cleanup authority: empty means no error. The inner
	// runner does not currently observe a bounded-cleanup
	// result; we record the fact in a typed way so the B2
	// barrier can reject if it requires observed cleanup.
	cleanup := evidence.CleanupAuthority{}
	if candidate.CleanupError != "" {
		cleanup.SubjectCleanupError = candidate.CleanupError
	}
	obs := V2ExecutionObservation{
		V2Manifest:   candidate.Manifest,
		Runtime: evidence.RuntimeAuthority{
			RepositoryRoot:       req.RepositoryRoot,
			FreezeCommit:         req.FreezeCommit,
			FreezeTree:           candidate.Manifest.FreezeTree,
			SubjectCommit:        candidate.Manifest.SubjectCommit,
			SubjectTree:          candidate.Manifest.SubjectTree,
			SubjectExecutionRoot: subjectWorktree,
			ExecutionTree:        candidate.Manifest.ExecutionTree,
			PlanPath:             frozen.Path,
			PlanBlob:             frozen.BlobOID,
			PlanSHA256:           planSHA,
			PlanBytes:            planBytes,
			FAncestorOfSVerified: fAccept,
		},
		Results:      b2Results,
		Gate:         gate,
		Binary:       b2Binary,
		CallerBefore: b2Before,
		CallerAfter:  b2After,
		Cleanup:      cleanup,
	}
	return candidate.Manifest, obs, nil
}

// sha256OfBytes returns the hex SHA-256 of the bytes.
func sha256OfBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
