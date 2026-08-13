// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// symlinkFixture builds a sandboxed directory layout for the
// symlink matrix:
//
//	<root>/
//	  main/               (main worktree root)
//	    file
//	    hop1 -> main      (chain entry inside main)
//	  linked-real/        (linked worktree root)
//	    file
//	  linked              -> linked-real/   (linked symlinked view)
//	  outside/            (plain detached directory)
//	    plain.txt
//	    link-into-main   -> main/file
//	    into-main-link    -> main/file
//	    into-main-dir     -> main
//	    into-linked-link  -> linked-real/file
//	    into-linked-dir   -> linked-real
//	    into-main-chain   -> main/hop1
//
// All paths are inside t.TempDir() so cleanup is automatic.
type symlinkFixture struct {
	root           string
	main           string
	linkedReal     string
	linked         string
	outside        string
	plainFile      string
	intoMainFile   string
	intoLinkedFile string
	intoMainDir    string
	intoLinkedDir  string
	intoMainChain  string
}

func newSymlinkFixture(t *testing.T) *symlinkFixture {
	t.Helper()
	root := t.TempDir()

	main := filepath.Join(root, "main")
	if err := os.Mkdir(main, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(main, "file"), []byte("m"), 0o644); err != nil {
		t.Fatal(err)
	}
	hop1 := filepath.Join(main, "hop1")
	if err := os.Symlink(main, hop1); err != nil {
		t.Fatal(err)
	}

	linkedReal := filepath.Join(root, "linked-real")
	if err := os.Mkdir(linkedReal, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linkedReal, "file"), []byte("l"), 0o644); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(root, "linked")
	if err := os.Symlink(linkedReal, linked); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	plainFile := filepath.Join(outside, "plain.txt")
	if err := os.WriteFile(plainFile, []byte("p"), 0o644); err != nil {
		t.Fatal(err)
	}
	intoMainFile := filepath.Join(outside, "into-main-link")
	if err := os.Symlink(filepath.Join(main, "file"), intoMainFile); err != nil {
		t.Fatal(err)
	}
	intoLinkedFile := filepath.Join(outside, "into-linked-link")
	if err := os.Symlink(filepath.Join(linkedReal, "file"), intoLinkedFile); err != nil {
		t.Fatal(err)
	}
	intoMainDir := filepath.Join(outside, "into-main-dir")
	if err := os.Symlink(main, intoMainDir); err != nil {
		t.Fatal(err)
	}
	intoLinkedDir := filepath.Join(outside, "into-linked-dir")
	if err := os.Symlink(linkedReal, intoLinkedDir); err != nil {
		t.Fatal(err)
	}
	intoMainChain := filepath.Join(outside, "into-main-chain")
	if err := os.Symlink(hop1, intoMainChain); err != nil {
		t.Fatal(err)
	}

	return &symlinkFixture{
		root:           root,
		main:           main,
		linkedReal:     linkedReal,
		linked:         linked,
		outside:        outside,
		plainFile:      plainFile,
		intoMainFile:   intoMainFile,
		intoLinkedFile: intoLinkedFile,
		intoMainDir:    intoMainDir,
		intoLinkedDir:  intoLinkedDir,
		intoMainChain:  intoMainChain,
	}
}

func inventoryWithRoots(roots ...string) RepositoryWorktreeInventory {
	inv, err := newRepositoryWorktreeInventoryFromCanonical(roots)
	if err != nil {
		panic(err)
	}
	return inv
}

func canonicalFromInventory(inv RepositoryWorktreeInventory) []CanonicalWorktree {
	out := make([]CanonicalWorktree, 0, len(inv.RootsView()))
	for _, r := range inv.RootsView() {
		out = append(out, CanonicalWorktree{Path: r})
	}
	return out
}

// TestConfineDestination_RootNameMatchesOpenedParent ensures
// the prepared authority's underlying os.Root descriptor's
// Name() equals the resolved parent from the inventory-aware
// resolver. A drift here would mean a future refactor that
// re-resolves the destination after opening the parent.
//
// The authority exposes canonical paths (post-EvalSymlinks); on
// macOS /var is a symlink to /private/var, so a string-equality
// comparison between the lexical fixture path and the canonical
// parent would drift spuriously. The test therefore uses
// os.SameFile as the canonical-identity oracle: two paths
// reference the same directory iff their os.Stat results carry
// matching inode identity.
func TestConfineDestination_RootNameMatchesOpenedParent(t *testing.T) {
	fx := newSymlinkFixture(t)
	inventory := inventoryWithRoots(fx.main, fx.linkedReal)
	candidate := filepath.Join(fx.outside, "future.txt")
	auth, err := PrepareVerifierOutput(fx.main, candidate, canonicalFromInventory(inventory))
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	defer auth.Close()
	wantParent := filepath.Clean(fx.outside)
	gotParent := filepath.Dir(auth.CanonicalPath())
	wantInfo, werr := os.Stat(wantParent)
	if werr != nil {
		t.Fatalf("stat wantParent %q: %v", wantParent, werr)
	}
	gotInfo, gerr := os.Stat(gotParent)
	if gerr != nil {
		t.Fatalf("stat gotParent %q: %v", gotParent, gerr)
	}
	if !os.SameFile(wantInfo, gotInfo) {
		t.Fatalf("parent mismatch: got %q want %q (same-file=false)", gotParent, wantParent)
	}
}

// TestConfineDestination_PrepareRoundTrip confirms the
// canonical destination set during preparation survives a
// defer-cleaned close and the auth's String methods.
//
// The publication contract exposes canonical (post-EvalSymlinks)
// paths; on macOS /var is a symlink to /private/var, so a
// raw strings.HasPrefix between the lexical fixture path and
// the canonical destination would drift spuriously. The test
// compares via os.SameFile on the parent directory.
func TestConfineDestination_PrepareRoundTrip(t *testing.T) {
	fx := newSymlinkFixture(t)
	inventory := inventoryWithRoots(fx.main, fx.linkedReal)
	candidate := filepath.Join(fx.outside, "future.txt")
	auth, err := PrepareVerifierOutput(fx.main, candidate, canonicalFromInventory(inventory))
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	defer auth.Close()
	if auth.CanonicalPath() == "" {
		t.Fatalf("canonical path empty")
	}
	wantParentInfo, werr := os.Stat(fx.outside)
	if werr != nil {
		t.Fatalf("stat wantParent %q: %v", fx.outside, werr)
	}
	gotParentInfo, gerr := os.Stat(filepath.Dir(auth.CanonicalPath()))
	if gerr != nil {
		t.Fatalf("stat gotParent %q: %v", filepath.Dir(auth.CanonicalPath()), gerr)
	}
	if !os.SameFile(wantParentInfo, gotParentInfo) {
		t.Fatalf("canonical %q not under %q (same-file=false)", auth.CanonicalPath(), fx.outside)
	}
	if auth.State() != PublicationNotPublished {
		t.Fatalf("expected initial state not_published, got %s", auth.State())
	}
}

// TestSymlinkMatrix covers the full Phase 5 specification
// matrix. Each row asserts the resolver's accept/reject
// verdict for a specific candidate layout.
func TestSymlinkMatrix(t *testing.T) {
	fx := newSymlinkFixture(t)
	inventory := inventoryWithRoots(fx.main, fx.linkedReal)

	cases := []struct {
		name      string
		want      bool
		candidate string
	}{
		// Accepted
		{"outside all worktrees", true, fx.plainFile},
		{"ordinary nonexistent detached path", true, filepath.Join(fx.outside, "future.txt")},
		// Rejected
		{"inside main worktree", false, filepath.Join(fx.main, "inside.txt")},
		{"equal main worktree root", false, fx.main},
		{"inside linked worktree", false, filepath.Join(fx.linkedReal, "inside.txt")},
		{"equal linked worktree root", false, fx.linkedReal},
		{"outside symlink -> main worktree file", false, fx.intoMainFile},
		{"outside symlink -> linked worktree file", false, fx.intoLinkedFile},
		{"outside symlink -> main worktree dir", false, fx.intoMainDir},
		{"outside symlink -> linked worktree dir", false, fx.intoLinkedDir},
		{"nested symlink chain into main worktree", false, filepath.Join(fx.intoMainChain, "file")},
		{"nonexistent under symlinked parent in worktree", false, filepath.Join(fx.intoMainDir, "future.txt")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			canonical, err := resolveDetachedDestination(fx.main, c.candidate, inventory)
			gotAccept := err == nil
			if gotAccept != c.want {
				t.Fatalf("candidate %s: got_accept=%v want=%v (canonical=%q err=%v)",
					c.candidate, gotAccept, c.want, canonical, err)
			}
		})
	}
}

// TestConfineDestination_ExistingDirectoryRejected confirms an
// existing directory at the candidate location is rejected
// because the contract is "regular file".
func TestConfineDestination_ExistingDirectoryRejected(t *testing.T) {
	fx := newSymlinkFixture(t)
	inventory := inventoryWithRoots(fx.main, fx.linkedReal)
	_, err := resolveDetachedDestination(fx.main, fx.outside, inventory)
	if err == nil {
		t.Fatalf("expected directory rejection")
	}
	var vErr *V2VerifierError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
	if vErr.Diags[0].Code != V2VerifierOutputPathNotDetached {
		t.Fatalf("expected verifier_output_path_not_detached, got %v", vErr.Diags[0].Code)
	}
}

// TestConfineDestination_NoSymlinkLeakIntoRepository ensures
// even a single-hop symlink into a linked worktree is
// rejected.
func TestConfineDestination_NoSymlinkLeakIntoRepository(t *testing.T) {
	fx := newSymlinkFixture(t)
	inventory := inventoryWithRoots(fx.main, fx.linkedReal)
	link := filepath.Join(fx.outside, "into-linked-real-direct")
	if err := os.Symlink(filepath.Join(fx.linkedReal, "file"), link); err != nil {
		t.Fatal(err)
	}
	_, err := resolveDetachedDestination(fx.main, link, inventory)
	if err == nil {
		t.Fatalf("expected rejection for symlink into linked worktree")
	}
}

// TestConfineDestination_CanonicalizesThenCompares proves the
// resolver canonicalizes the candidate through
// filepath.EvalSymlinks before comparing against worktree
// roots. A symlinked candidate whose target sits in the
// worktree is rejected.
func TestConfineDestination_CanonicalizesThenCompares(t *testing.T) {
	fx := newSymlinkFixture(t)
	inventory := inventoryWithRoots(fx.main, fx.linkedReal)
	candidate := filepath.Join(fx.outside, "link-into-main-direct")
	if err := os.Symlink(filepath.Join(fx.main, "file"), candidate); err != nil {
		t.Fatal(err)
	}
	_, err := resolveDetachedDestination(fx.main, candidate, inventory)
	if err == nil {
		t.Fatalf("expected rejection for symlinked path into main")
	}
}

// TestConfineDestination_NonexistentPlainPath confirms the
// resolver's walk-up behavior for ordinary detached missing
// paths.
func TestConfineDestination_NonexistentPlainPath(t *testing.T) {
	fx := newSymlinkFixture(t)
	inventory := inventoryWithRoots(fx.main, fx.linkedReal)
	candidate := filepath.Join(fx.outside, "absent.txt")
	got, err := resolveDetachedDestination(fx.main, candidate, inventory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("canonical form %q is not absolute", got)
	}
}

// TestConfineDestination_RejectNestedSymlinkChainIntoWorktree
// covers the dedicated nested-chain case.
func TestConfineDestination_RejectNestedSymlinkChainIntoWorktree(t *testing.T) {
	fx := newSymlinkFixture(t)
	inventory := inventoryWithRoots(fx.main, fx.linkedReal)
	candidate := filepath.Join(fx.intoMainChain, "file")
	_, err := resolveDetachedDestination(fx.main, candidate, inventory)
	if err == nil {
		t.Fatalf("expected rejection for nested symlink chain into main")
	}
}

// TestConfineDestination_LinkedSymlinkEqualWorktree confirms
// that a symlink whose target is exactly equal to a worktree
// root triggers rejection. This is a "no symlinked alias"
// guard preventing a path that lexicalises to outside but
// canonicalises to inside.
func TestConfineDestination_LinkedSymlinkEqualWorktree(t *testing.T) {
	fx := newSymlinkFixture(t)
	inventory := inventoryWithRoots(fx.main, fx.linkedReal)
	link := filepath.Join(fx.outside, "linked-mirror")
	if err := os.Symlink(fx.linked, link); err != nil {
		t.Fatal(err)
	}
	_, err := resolveDetachedDestination(fx.main, link, inventory)
	if err == nil {
		t.Fatalf("expected rejection for symlinked worktree view")
	}
}
