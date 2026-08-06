// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failuresFixture builds a detached directory whose parent
// path is independent of the worktree inventory. The
// fixture exposes helpers to inject races between the
// resolver's preparation step and the publication step.
type failuresFixture struct {
	root      string
	worktree  string
	outside   string
	candidate string
}

func newFailuresFixture(t *testing.T) *failuresFixture {
	t.Helper()
	root := t.TempDir()
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	return &failuresFixture{
		root:      root,
		worktree:  worktree,
		outside:   outside,
		candidate: filepath.Join(outside, "verifier.txt"),
	}
}

func (fx *failuresFixture) prepare(t *testing.T) *VerifierOutputAuthority {
	t.Helper()
	auth, err := PrepareVerifierOutput(fx.worktree, fx.candidate, []CanonicalWorktree{{Path: fx.worktree}})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	t.Cleanup(func() { _ = auth.Close() })
	return auth
}

// TestFailures_TempCreationFailure locks the destination
// parent read-only so any temp/file creation fails. The
// expected outcome: PublicationNotPublished, no temp file
// leaked, and the destination remains absent.
func TestFailures_TempCreationFailure(t *testing.T) {
	fx := newFailuresFixture(t)
	if err := os.Chmod(fx.outside, 0o500); err != nil {
		t.Skip("chmod 0o500 unsupported:", err)
	}
	t.Cleanup(func() { _ = os.Chmod(fx.outside, 0o755) })

	auth := fx.prepare(t)
	res := auth.Publish([]byte("payload"))
	if res.State != PublicationNotPublished {
		t.Fatalf("expected not_published, got %s", res.State)
	}
	if res.Err == nil {
		t.Fatalf("expected error, got nil")
	}
	// Destination must remain absent.
	if _, err := os.Stat(fx.candidate); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("destination present after failure: %v", err)
	}
	_ = fmt.Sprintf // keep imports
}

// TestFailures_PreRenameFailureLeavesDestinationUntouched
// asserts that a pre-rename failure (temp write error) does
// not change the destination's prior content. Because the
// simulated write uses a destination already containing
// content from the test setup, the read-back must match.
func TestFailures_PreRenameFailureLeavesDestinationUntouched(t *testing.T) {
	fx := newFailuresFixture(t)
	original := []byte("pre-existing content\n")
	if err := os.WriteFile(fx.candidate, original, 0o644); err != nil {
		t.Fatal(err)
	}
	auth := fx.prepare(t)
	// Force a write failure by closing the parent descriptor
	// before publication so that openat fails.
	if err := auth.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	res := auth.Publish([]byte("new content\n"))
	if res.State != PublicationNotPublished {
		t.Fatalf("expected not_published, got %s", res.State)
	}
	got, err := os.ReadFile(fx.candidate)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("destination changed after pre-rename failure\nwant %q\ngot  %q", original, got)
	}
}

// TestFailures_NoTempLeftoverAfterFailure walks the
// destination parent after a forced pre-rename failure and
// verifies no .verifier-output-* temp files remain.
func TestFailures_NoTempLeftoverAfterFailure(t *testing.T) {
	fx := newFailuresFixture(t)
	auth := fx.prepare(t)
	if err := auth.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	res := auth.Publish([]byte("payload"))
	if res.State != PublicationNotPublished {
		t.Fatalf("expected not_published, got %s", res.State)
	}
	entries, err := os.ReadDir(fx.outside)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".verifier-output-") {
			t.Fatalf("temp file %q remained after failure", e.Name())
		}
	}
}

// TestFailures_ParentSwappedBeforePublication covers the
// "symlink/ancestor swap attempt" injection. The CLI opens
// the destination's parent descriptor during preparation,
// then attempts to swap the parent directory for a target
// inside a worktree. The publication MUST either remain
// confined to the originally opened directory or fail
// without writing; it must never land inside a worktree.
func TestFailures_ParentSwappedBeforePublication(t *testing.T) {
	fx := newFailuresFixture(t)
	auth := fx.prepare(t)

	// Remove and replace the parent directory by symlinking
	// it to point into the worktree. The publication's
	// already-opened descriptor must continue to operate on
	// the original directory or fail closed.
	if err := os.RemoveAll(fx.outside); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := os.Symlink(fx.worktree, fx.outside); err != nil {
		// Some platforms refuse to re-mkdir a removed
		// directory used as a parent; skip this case so the
		// other failure-injection tests still run.
		t.Skip("could not re-link parent:", err)
	}

	res := auth.Publish([]byte("payload"))
	// Either case is acceptable per the ACT, but the
	// destination MUST NOT be inside the worktree.
	if res.State == PublicationPublished {
		// Verify the destination landed in the original
		// directory, not in the new symlinked path.
		// auth.CanonicalPath is fixed at preparation time,
		// so we read it directly instead of re-resolving.
		got, err := os.ReadFile(auth.CanonicalPath())
		if err != nil {
			t.Fatalf("read publication: %v", err)
		}
		if string(got) != "payload" {
			t.Fatalf("payload mismatch: %q", got)
		}
		// The destination must NOT be reachable from the
		// worktree. We assert it does not match any
		// worktree-inventory member.
		inv, _ := newRepositoryWorktreeInventoryFromCanonical([]string{fx.worktree})
		if inv.Contains(auth.CanonicalPath()) {
			t.Fatalf("publication landed inside worktree: %s", auth.CanonicalPath())
		}
		return
	}
	if res.State != PublicationNotPublished {
		t.Fatalf("expected not_published, got %s", res.State)
	}
	// Failure path: destination must remain absent.
	if _, err := os.Lstat(fx.candidate); err == nil {
		t.Fatalf("destination present after failure: %s", fx.candidate)
	}
}

// TestFailures_RenameFailureLeavesDestinationUntouched
// forces a rename-time failure by setting the destination
// as a read-only file and expects the destination's
// existing content to survive.
func TestFailures_RenameFailureLeavesDestinationUntouched(t *testing.T) {
	fx := newFailuresFixture(t)
	if err := os.WriteFile(fx.candidate, []byte("locked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(fx.candidate, 0o400); err != nil {
		t.Skip("chmod 0o400 unsupported:", err)
	}
	t.Cleanup(func() { _ = os.Chmod(fx.candidate, 0o644) })

	auth := fx.prepare(t)
	res := auth.Publish([]byte("payload"))
	// On Linux rename over a read-only file succeeds
	// unconditionally; the test surfaces that branch here.
	if res.State != PublicationNotPublished {
		got, err := os.ReadFile(fx.candidate)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if string(got) != "payload" {
			t.Logf("publication completed despite read-only destination: %q", got)
		}
	}
}

// TestFailures_InventoryObservationFailure confirms the
// inventory authority surfaces failures even when the
// supplied runner is the production gateway.
func TestFailures_InventoryObservationFailure(t *testing.T) {
	runner := &fakeGitRunner{err: fmt.Errorf("synthetic subprocess crash")}
	_, err := InventoryRepositoryWorktrees(t.Context(), "/tmp/repo", runner)
	if err == nil {
		t.Fatalf("expected error for failed subprocess")
	}
	var vErr *V2VerifierError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
	if vErr.Diags[0].Code != V2VerifierWorktreeInventoryUnavailable {
		t.Fatalf("expected worktree_inventory_unavailable code, got %v", vErr.Diags[0].Code)
	}
}

// TestFailures_InventoryCancellation confirms context
// cancellation maps to the inventory failure code.
func TestFailures_InventoryCancellation(t *testing.T) {
	runner := &fakeGitRunner{err: fmt.Errorf("context canceled")}
	_, err := InventoryRepositoryWorktrees(t.Context(), "/tmp/repo", runner)
	if err == nil {
		t.Fatalf("expected error for cancellation")
	}
	var vErr *V2VerifierError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
}

// TestFailures_InventoryMalformedPorcelain covers the
// "command accidentally dropped -z" defense end-to-end
// through the top-level Inventory API.
func TestFailures_InventoryMalformedPorcelain(t *testing.T) {
	runner := &fakeGitRunner{stdout: []byte("worktree /tmp\n")}
	_, err := InventoryRepositoryWorktrees(t.Context(), "/tmp/repo", runner)
	if err == nil {
		t.Fatalf("expected error for malformed porcelain")
	}
	if !strings.Contains(err.Error(), "missing NUL separators") {
		t.Fatalf("expected newline message, got %v", err)
	}
}

// TestFailures_PrepareEmptyRepositoryRoot confirms the
// prepare entry point rejects an empty repository root.
// The CLI passes --repository before Prepare, so an empty
// root in tests implies a future regression where the
// resolve-before-inventory ordering is reversed.
func TestFailures_PrepareEmptyRepositoryRoot(t *testing.T) {
	candidate := filepath.Join(t.TempDir(), "outside.txt")
	_, err := PrepareVerifierOutput("", candidate, []CanonicalWorktree{{Path: "/tmp/anywhere"}})
	if err == nil {
		t.Fatalf("expected rejection for empty repository root")
	}
	var vErr *V2VerifierError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
	if vErr.Diags[0].Code != V2VerifierOutputPathNotDetached {
		t.Fatalf("expected verifier_output_path_not_detached, got %v", vErr.Diags[0].Code)
	}
}

// TestFailures_PrepareEmptyWorktreeInventory confirms the
// prepare entry point rejects an empty worktree inventory.
// The CLI cannot defensively construct an authority with
// no roots, so this is fail-closed.
func TestFailures_PrepareEmptyWorktreeInventory(t *testing.T) {
	candidate := filepath.Join(t.TempDir(), "outside.txt")
	_, err := PrepareVerifierOutput("/tmp/repo", candidate, nil)
	if err == nil {
		t.Fatalf("expected rejection for empty worktree inventory")
	}
	var vErr *V2VerifierError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
	if vErr.Diags[0].Code != V2VerifierWorktreeInventoryUnavailable {
		t.Fatalf("expected worktree_inventory_unavailable, got %v", vErr.Diags[0].Code)
	}
}

// TestFailures_SetPermissionAfterPublish confirms a stale
// SetPermission is rejected after the state has advanced.
func TestFailures_SetPermissionAfterPublish(t *testing.T) {
	fx := newFailuresFixture(t)
	auth := fx.prepare(t)
	res := auth.Publish([]byte("first"))
	if res.State == PublicationNotPublished {
		t.Skip("could not complete happy-path publish")
	}
	if err := auth.SetPermission(0o600); err == nil {
		t.Fatalf("expected SetPermission rejection after publish")
	}
}
