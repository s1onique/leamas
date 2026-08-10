// SPDX-License-Identifier: Apache-2.0

// closure_evidence_publication_runner_adapter.go owns the
// non-publishing runner entry point that produces the B2
// evidence inputs from authoritative runner sources.
//
// R5 invariant: every field in V2ExecutionObservation
// comes from a runner source. No field is synthesized.
// The orchestrator's B2 inputs come from this struct; a
// caller cannot smuggle a fabricated CandidateInputs bundle
// past the B2 barrier.
package closure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// V2ExecutionObservation is the authoritative execution record
// the B2 publication barrier requires. Every field is
// derived from sources the V2 runner already consults; no
// field is synthesized.
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
	frozen, ferr := deps.Loader.LoadFrozenPlan(ctx, req.RepositoryRoot, req.FreezeCommit, req.PlanPath)
	if ferr != nil {
		return candidate.Manifest, V2ExecutionObservation{}, fmt.Errorf("execute: reload F:P for B2: %w", ferr)
	}
	planBytes := append([]byte(nil), frozen.Bytes...)
	planSum := sha256.Sum256(planBytes)
	planSHA := hex.EncodeToString(planSum[:])
	subjectWorktree := candidate.SubjectWorktreePath
	fAccept := candidate.TopologyFacts.Classify() == V2RelationFreezeBeforeSubject
	b2Results := make([]evidence.CheckResult, 0, len(candidate.Manifest.CheckResults))
	for _, r := range candidate.Manifest.CheckResults {
		b2Results = append(b2Results, evidence.CheckResult{
			CheckID:  r.ID,
			Mode:     r.Mode,
			Outcome:  r.Outcome,
			ExitCode: derefInt(r.ExitCode),
		})
	}
	// B1 binary authority. SourceClean/SourceDetached/
	// OutputOutsideAllWorktrees/Executable remain false
	// until B1 is wired into the V2 runner; the B2 barrier
	// will reject the candidate with the corresponding
	// `binary_*` predicate (correct honest signal).
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
	// runner's own observations. SubjectRoot and
	// SubjectExecutionRoot both equal the runner's actual
	// detached S worktree path; ObservedStatus is the
	// runner's classified topology relation; Classification
	// is PASS only when the runner's topology proof accepted
	// and every check result passed; InvocationCount is 1
	// (the production gate is captured exactly once).
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
	// runner's actual observations. An empty observation
	// (e.g. a clean porcelain status) is a valid
	// observation whose authority hash is the SHA-256 of
	// zero bytes — never an empty string.
	b2Before := evidence.CallerStateSnapshot{
		Available:             true,
		Head:                  callerBeforeSnap.State.HEADCommit,
		Tree:                  callerBeforeSnap.State.HEADTree,
		StatusHash:            sha256OfBytes([]byte(callerBeforeSnap.State.StatusPorcelain)),
		RefsHash:              sha256OfBytes([]byte(callerBeforeSnap.State.RefsBytes)),
		WorktreeInventoryHash: sha256OfWorktreeInventory(callerBeforeSnap.State.WorktreeRegistrations),
	}
	b2After := evidence.CallerStateSnapshot{
		Available:             true,
		Head:                  callerAfterSnap.State.HEADCommit,
		Tree:                  callerAfterSnap.State.HEADTree,
		StatusHash:            sha256OfBytes([]byte(callerAfterSnap.State.StatusPorcelain)),
		RefsHash:              sha256OfBytes([]byte(callerAfterSnap.State.RefsBytes)),
		WorktreeInventoryHash: sha256OfWorktreeInventory(callerAfterSnap.State.WorktreeRegistrations),
	}
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

// sha256OfBytes returns the hex SHA-256 of the bytes. An
// empty byte stream is a valid observation whose authority
// hash is the SHA-256 of zero bytes — never an empty string.
func sha256OfBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// sha256OfWorktreeInventory hashes a canonical, versioned
// serialization of the runner's worktree registration set.
// Entries are sorted by their path bytes so different
// insertion orders produce the same hash (the canonical
// authority), and different sets produce different hashes.
// This is the durable worktree-inventory authority hash, not
// a debugging representation.
func sha256OfWorktreeInventory(invs v2WorktreeRegistrationSet) string {
	entries := make([]string, 0, len(invs))
	for _, e := range invs {
		entries = append(entries, e.Path)
	}
	sort.Strings(entries)
	var buf bytes.Buffer
	buf.WriteString("worktree-inventory-v1\n")
	for _, e := range entries {
		buf.WriteString(e)
		buf.WriteByte(0)
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
