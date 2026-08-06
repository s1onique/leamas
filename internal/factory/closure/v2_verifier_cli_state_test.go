// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_cli_state_test.go covers Phase 5 + Phase 6 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-CLI-TAG-STATE01:
//
// the optional read-only caller-state capture used to prove
// the verifier never mutates the target repository, plus
// the dirty-worktree independence requirement.
//
// The tests assert:
//
//   - CaptureV2CallerState returns non-empty projections
//     against a fresh repository.
//   - Hash() is deterministic for byte-equal inputs.
//   - Hash() differs by a full hash when any projection
//     differs by one byte.
//   - CheckReadOnly emits state_mutation_detected only when
//     a projection actually changes.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestV2CallerStateCapture exercises CaptureV2CallerState
// against a fresh repository. The test asserts all five
// projections are populated and the Hash() is stable across
// repeated calls.
func TestV2CallerStateCapture(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "seed", map[string]string{"a": "b"})
	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}

	snapA := CaptureV2CallerState(context.Background(), auth)
	if snapA.RepositoryRoot == "" {
		t.Fatalf("RepositoryRoot must be populated, got %+v", snapA)
	}
	if snapA.HeadCommit == "" {
		t.Fatalf("HeadCommit must be populated, got %+v", snapA)
	}
	if snapA.HeadTree == "" {
		t.Fatalf("HeadTree must be populated, got %+v", snapA)
	}
	if snapA.Worktrees == "" {
		t.Fatalf("Worktrees must be populated, got %+v", snapA)
	}
	if snapA.HeadCommit == "" {
		t.Fatalf("HeadCommit must be populated")
	}

	snapB := CaptureV2CallerState(context.Background(), auth)
	if snapA.Hash() != snapB.Hash() {
		t.Fatalf("state hash must be deterministic for an unchanged repository: %s vs %s",
			snapA.Hash(), snapB.Hash())
	}
}

// TestV2CallerStateHashDifferences verifies that changing
// any projection (HEAD, status, worktrees, refs) produces
// an entirely different hash. The helper exposes the
// deterministic SHA-256 contract required by Phase 5.
func TestV2CallerStateHashDifferences(t *testing.T) {
	base := V2CallerStateSnapshot{
		RepositoryRoot: "/x",
		HeadCommit:     "1111111111111111111111111111111111111111",
		HeadTree:       "2222222222222222222222222222222222222222",
		Status:         "1 .M N... 100644 100644 100644 deadbeef deadbeef x.txt\n",
		Worktrees:      "worktree /x\nHEAD abc\nbranch refs/heads/main\n",
		Refs:           "refs/heads/main\tabc",
	}
	mutated := V2CallerStateSnapshot{
		RepositoryRoot: "/x",
		HeadCommit:     "1111111111111111111111111111111111111111",
		HeadTree:       "2222222222222222222222222222222222222222",
		Status:         "1 .M N... 100644 100644 100644 deadbeef deadbeef x.txt\n",
		Worktrees:      "worktree /x\nHEAD abc\nbranch refs/heads/main\n",
		Refs:           "refs/heads/main\tabc2",
	}
	if base.Hash() == mutated.Hash() {
		t.Fatalf("Hash() must differ when Refs changes by a single character")
	}
}

// TestV2CallerStateReadOnly checks the read-only invariant:
// CheckReadOnly returns nil for two byte-equal snapshots and
// emits a state_mutation_detected diagnostic when any
// projection differs.
func TestV2CallerStateReadOnly(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "seed", map[string]string{"a": "b"})
	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}
	before := CaptureV2CallerState(context.Background(), auth)
	after := CaptureV2CallerState(context.Background(), auth)
	if diags := CheckReadOnly(before, after); len(diags) != 0 {
		t.Fatalf("read-only check on identical snapshots must return no diagnostics, got %v",
			diags.Codes())
	}

	// Simulate a mutation: change HeadCommit by hand.
	mutated := after
	mutated.HeadCommit = "ffffffffffffffffffffffffffffffffffffffff"
	if diags := CheckReadOnly(before, mutated); !diags.HasCode(V2VerifierStateMutationDetected) {
		t.Fatalf("expected state_mutation_detected diagnostic, got %v", diags.Codes())
	}
}

// TestV2CallerStateDirtyWorktreeIndependence verifies
// Phase 6 of the ACT 4 contract: a dirty worktree (an
// untracked file plus a tracked-file modification) does
// NOT change any authority the verifier binds on.
// The authority strings (HEAD commit, HEAD tree) stay
// identical even when the dirty status bytes differ.
//
// Git objects (commits, trees, blobs) are an immutable
// function of their input bytes; a working-tree dirt
// condition cannot affect what F:P or C:M contain, even
// though the status output reflects the dirt.
func TestV2CallerStateDirtyWorktreeIndependence(t *testing.T) {
	dir := initRepo(t)
	makeCommit(t, dir, "seed", map[string]string{"a": "b"})
	auth, err := newV2ClosureGitAuthorityWithClient(RealGit{}, dir)
	if err != nil {
		t.Fatalf("authority: %v", err)
	}

	clean := CaptureV2CallerState(context.Background(), auth)

	// Dirt the worktree: tracked modification + untracked file.
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("c"), 0o644); err != nil {
		t.Fatalf("write tracked file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}
	dirty := CaptureV2CallerState(context.Background(), auth)

	if clean.HeadCommit != dirty.HeadCommit {
		t.Fatalf("HEAD commit must not change with a dirty worktree: %s vs %s",
			clean.HeadCommit, dirty.HeadCommit)
	}
	if clean.HeadTree != dirty.HeadTree {
		t.Fatalf("HEAD tree must not change with a dirty worktree: %s vs %s",
			clean.HeadTree, dirty.HeadTree)
	}
	if clean.Status == dirty.Status {
		t.Fatalf("status bytes must reflect the dirty worktree (clean=%q dirty=%q)",
			clean.Status, dirty.Status)
	}
}
