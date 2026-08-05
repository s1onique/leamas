// SPDX-License-Identifier: Apache-2.0

package closure

// v2_publication_barrier_r2a_matrix_test.go proves the
// after-state publication barrier under the four
// after-unavailable cases and the four after-drift cases
// required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2A.
//
// For every row the test asserts:
//
//   - inner execution completed
//   - candidate manifest constructed exactly once
//   - after observation failed or detected drift
//   - typed diagnostic exact
//   - returned error non-nil
//   - final manifest absent
//
// Splitting this from v2_publication_barrier_r2a_success_test.go
// keeps every file under the LLM-friendly 400-line threshold.

import (
	"context"
	"testing"
)

// TestV2RunnerPublicationBarrier_AfterHeadUnavailable proves
// the outer runner refuses to publish the manifest when the
// AFTER HEAD^{commit} observation fails.
//
// Pre-correction (R1) the inner runner wrote the manifest
// before the after-snapshot failed; the test would have
// observed the manifest on disk. Post-correction the inner
// runner returns an unpublished candidate; the outer runner
// refuses to write; the manifest path is absent.
func TestV2RunnerPublicationBarrier_AfterHeadUnavailable(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	observer := &countingCandidateObserver{}
	deps := r2aRunnerDeps(t,
		unavailableAfterSnapshotFn(t, V2CodeCallerStateUnavailable,
			"caller state observation failed: HEAD lookup: simulated"),
		observer)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after HEAD failure must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerStateUnavailable) {
		t.Fatalf("expected caller_state_unavailable, got %v", v2err.Diags.Codes())
	}
	assertCandidateConstructedOnce(t, observer)
	assertManifestAbsent(t, req.ManifestOutput)
}

// TestV2RunnerPublicationBarrier_AfterHeadTreeUnavailable
// proves the outer runner refuses to publish the manifest
// when the AFTER HEAD^{tree} observation fails.
func TestV2RunnerPublicationBarrier_AfterHeadTreeUnavailable(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	observer := &countingCandidateObserver{}
	deps := r2aRunnerDeps(t,
		unavailableAfterSnapshotFn(t, V2CodeCallerStateUnavailable,
			"caller state observation failed: HEAD^{tree} lookup: simulated"),
		observer)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after HEAD^{tree} failure must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerStateUnavailable) {
		t.Fatalf("expected caller_state_unavailable, got %v", v2err.Diags.Codes())
	}
	assertCandidateConstructedOnce(t, observer)
	assertManifestAbsent(t, req.ManifestOutput)
}

// TestV2RunnerPublicationBarrier_AfterStatusUnavailable proves
// the outer runner refuses to publish the manifest when the
// AFTER status observation fails.
func TestV2RunnerPublicationBarrier_AfterStatusUnavailable(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	observer := &countingCandidateObserver{}
	deps := r2aRunnerDeps(t,
		unavailableAfterSnapshotFn(t, V2CodeCallerStateUnavailable,
			"caller state observation failed: status: simulated"),
		observer)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after status failure must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerStateUnavailable) {
		t.Fatalf("expected caller_state_unavailable, got %v", v2err.Diags.Codes())
	}
	assertCandidateConstructedOnce(t, observer)
	assertManifestAbsent(t, req.ManifestOutput)
}

// TestV2RunnerPublicationBarrier_AfterWorktreeInventoryUnavailable
// proves the outer runner refuses to publish the manifest when
// the AFTER worktree inventory observation fails.
//
// The typed diagnostic set MUST include both
// worktree_inventory_unavailable (the precise failing
// observation) and caller_state_unavailable (the aggregate
// after-state verdict).
func TestV2RunnerPublicationBarrier_AfterWorktreeInventoryUnavailable(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	observer := &countingCandidateObserver{}
	deps := r2aRunnerDeps(t,
		phaseAwareSnapshotFn(func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
			_ = ctx
			_ = git
			_ = repoRoot
			_ = phase
			return v2CallerStateSnapshot{
				Available: false,
				Diagnostics: V2Diagnostics{
					{
						Code:         V2CodeWorktreeInventoryUnavailable,
						Message:      "worktree inventory observation failed: simulated",
						PropertyName: "worktree_inventory",
					},
					{
						Code:         V2CodeCallerStateUnavailable,
						Message:      "caller state observation failed: worktree inventory unavailable",
						PropertyName: "caller_state",
					},
				},
			}
		}),
		observer)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after worktree-inventory failure must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeWorktreeInventoryUnavailable) {
		t.Fatalf("expected worktree_inventory_unavailable, got %v", v2err.Diags.Codes())
	}
	if !v2err.Diags.HasCode(V2CodeCallerStateUnavailable) {
		t.Fatalf("expected caller_state_unavailable propagated, got %v", v2err.Diags.Codes())
	}
	assertCandidateConstructedOnce(t, observer)
	assertManifestAbsent(t, req.ManifestOutput)
}

// TestV2RunnerPublicationBarrier_AfterHeadDrift proves the
// outer runner refuses to publish the manifest when HEAD
// changed between BEFORE and AFTER snapshots.
func TestV2RunnerPublicationBarrier_AfterHeadDrift(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	observer := &countingCandidateObserver{}
	deps := r2aRunnerDeps(t, driftAfterSnapshotFn(t, "head_commit"), observer)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after HEAD drift must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerHeadChanged) {
		t.Fatalf("expected caller_head_changed, got %v", v2err.Diags.Codes())
	}
	assertCandidateConstructedOnce(t, observer)
	assertManifestAbsent(t, req.ManifestOutput)
}

// TestV2RunnerPublicationBarrier_AfterHeadTreeDrift proves the
// outer runner refuses to publish the manifest when HEAD^{tree}
// changed between BEFORE and AFTER snapshots.
func TestV2RunnerPublicationBarrier_AfterHeadTreeDrift(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	observer := &countingCandidateObserver{}
	deps := r2aRunnerDeps(t, driftAfterSnapshotFn(t, "head_tree"), observer)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after HEAD^{tree} drift must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerTreeChanged) {
		t.Fatalf("expected caller_tree_changed, got %v", v2err.Diags.Codes())
	}
	assertCandidateConstructedOnce(t, observer)
	assertManifestAbsent(t, req.ManifestOutput)
}

// TestV2RunnerPublicationBarrier_AfterStatusDirtyDrift proves
// the outer runner refuses to publish the manifest when the
// caller worktree becomes dirty between BEFORE and AFTER.
func TestV2RunnerPublicationBarrier_AfterStatusDirtyDrift(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	observer := &countingCandidateObserver{}
	deps := r2aRunnerDeps(t, driftAfterSnapshotFn(t, "status"), observer)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after status drift must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeCallerWorktreeDirtyAfter) {
		t.Fatalf("expected caller_worktree_dirty_after, got %v", v2err.Diags.Codes())
	}
	assertCandidateConstructedOnce(t, observer)
	assertManifestAbsent(t, req.ManifestOutput)
}

// TestV2RunnerPublicationBarrier_AfterWorktreeRegistrationLeaked
// proves the outer runner refuses to publish the manifest when
// a new linked-worktree registration appeared between BEFORE
// and AFTER.
func TestV2RunnerPublicationBarrier_AfterWorktreeRegistrationLeaked(t *testing.T) {
	req := v2FailClosedRunnerRequest(t)
	observer := &countingCandidateObserver{}
	deps := r2aRunnerDeps(t, driftAfterSnapshotFn(t, "worktree_leaked"), observer)
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("after worktree leak must reject")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeWorktreeRegistrationLeaked) {
		t.Fatalf("expected worktree_registration_leaked, got %v", v2err.Diags.Codes())
	}
	assertCandidateConstructedOnce(t, observer)
	assertManifestAbsent(t, req.ManifestOutput)
}
