// SPDX-License-Identifier: Apache-2.0

package closure

// v2_publication_barrier_r2a_helpers_test.go owns the shared
// helpers for the R2A publication-barrier test files:
//
//   - phaseAwareSnapshotFn
//   - unavailableAfterSnapshotFn
//   - driftAfterSnapshotFn
//   - mutateCommit
//   - assertManifestAbsent
//   - assertCandidateConstructedOnce
//   - r2aRunnerDeps
//
// Splitting these helpers from the matrix and success tests
// keeps every file under the LLM-friendly 400-line threshold
// while preserving a clear R2A boundary in the test corpus.

import (
	"context"
	"os"
	"testing"
)

// phaseAwareSnapshotFn returns a V2RunnerSnapshotFunc that
// calls the production snapshotCallerState for the BEFORE
// phase and the supplied afterFn for the AFTER phase. Tests
// use it to inject deterministic after-snapshot failures or
// drift without resorting to git command-count approximations.
func phaseAwareSnapshotFn(afterFn V2RunnerSnapshotFunc) V2RunnerSnapshotFunc {
	return func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
		if phase == V2SnapshotPhaseAfter {
			return afterFn(ctx, git, repoRoot, phase)
		}
		return snapshotCallerState(ctx, git, repoRoot)
	}
}

// unavailableAfterSnapshotFn returns a V2RunnerSnapshotFunc
// that fails the AFTER snapshot with the supplied diagnostic
// code. The BEFORE snapshot still observes the real state.
func unavailableAfterSnapshotFn(t *testing.T, code V2DiagnosticCode, message string) V2RunnerSnapshotFunc {
	t.Helper()
	return func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
		if phase != V2SnapshotPhaseAfter {
			return snapshotCallerState(ctx, git, repoRoot)
		}
		_ = ctx
		_ = git
		_ = repoRoot
		return v2CallerStateSnapshot{
			Available: false,
			Diagnostics: V2Diagnostics{{
				Code:         code,
				Message:      message,
				PropertyName: "caller_state",
			}},
		}
	}
}

// driftAfterSnapshotFn returns a V2RunnerSnapshotFunc that
// captures the real AFTER state but mutates one of the four
// observations to simulate drift detected between BEFORE and
// AFTER.
//
//	property == "head_commit"     -> after.HEADCommit differs
//	property == "head_tree"       -> after.HEADTree differs
//	property == "status"          -> after.StatusPorcelain is non-empty
//	property == "worktree_leaked" -> an extra worktree registration
func driftAfterSnapshotFn(t *testing.T, property string) V2RunnerSnapshotFunc {
	t.Helper()
	return func(ctx context.Context, git gitClient, repoRoot string, phase V2SnapshotPhase) v2CallerStateSnapshot {
		if phase != V2SnapshotPhaseAfter {
			return snapshotCallerState(ctx, git, repoRoot)
		}
		_ = ctx
		snap := snapshotCallerState(ctx, git, repoRoot)
		switch property {
		case "head_commit":
			snap.State.HEADCommit = mutateCommit(snap.State.HEADCommit)
		case "head_tree":
			snap.State.HEADTree = mutateCommit(snap.State.HEADTree)
		case "status":
			snap.State.StatusPorcelain = "?? leaked-untracked\n"
		case "worktree_leaked":
			snap.State.WorktreeRegistrations = append(snap.State.WorktreeRegistrations,
				v2WorktreeRegistration{Path: "/tmp/leaked-worktree", Hash: snap.State.HEADCommit})
		}
		return snap
	}
}

// mutateCommit flips the last byte of a 40-char commit hex
// to produce a syntactically valid but distinct OID. Tests
// use this to simulate HEAD or HEAD^{tree} drift between
// BEFORE and AFTER snapshots.
func mutateCommit(oid string) string {
	if len(oid) == 0 {
		return ""
	}
	b := []byte(oid)
	if b[len(b)-1] == '0' {
		b[len(b)-1] = '1'
	} else {
		b[len(b)-1] = '0'
	}
	return string(b)
}

// assertManifestAbsent fails the test if the manifest path
// exists or is non-empty.
func assertManifestAbsent(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err == nil {
		t.Fatalf("manifest must be absent at %s but exists (size=%d)", path, info.Size())
	}
	if !os.IsNotExist(err) {
		t.Fatalf("manifest stat at %s returned unexpected error: %v", path, err)
	}
}

// assertCandidateConstructedOnce fails the test unless the
// observer recorded exactly one construction event.
func assertCandidateConstructedOnce(t *testing.T, obs *countingCandidateObserver) {
	t.Helper()
	if obs.Calls() != 1 {
		t.Fatalf("expected exactly one CandidateConstructed call, got %d", obs.Calls())
	}
	if len(obs.lastBytes) == 0 {
		t.Fatalf("candidate observer captured empty bytes")
	}
	if obs.last.ClosureProtocolVersion != ClosureProtocolV2 {
		t.Fatalf("candidate manifest bound wrong protocol: %s", obs.last.ClosureProtocolVersion)
	}
}

// r2aRunnerDeps builds the V2RunnerDeps for an R2A test with
// the supplied SnapshotFn and CandidateObserver. The deps
// use the production binary identity so the inner runner can
// construct a manifest candidate when the after-snapshot
// authority passes.
func r2aRunnerDeps(t *testing.T, snap V2RunnerSnapshotFunc, obs V2CandidateObserver) V2RunnerDeps {
	t.Helper()
	deps := DefaultV2RunnerDeps()
	deps.BinaryIdentity = newV2TestBinaryIdentity(t)
	deps.SnapshotFn = snap
	deps.CandidateObserver = obs
	return deps
}
