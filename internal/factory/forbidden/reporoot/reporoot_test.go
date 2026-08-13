// SPDX-License-Identifier: Apache-2.0

package reporoot_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/forbidden/reporoot"
)

func TestRootResolver_RepoRoot(t *testing.T) {
	resolver := reporoot.New()

	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot get cwd")
	}

	root, err := resolver.Resolve(cwd)
	if err != nil {
		t.Fatalf("failed to resolve root: %v", err)
	}

	gitDir := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Error("resolved root should contain .git directory")
	}
}

func TestRootResolver_Subdirectory(t *testing.T) {
	resolver := reporoot.New()

	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot get cwd")
	}

	subdir := filepath.Join(cwd, "internal")
	if _, err := os.Stat(subdir); os.IsNotExist(err) {
		t.Skip("internal directory does not exist")
	}

	root, err := resolver.Resolve(subdir)
	if err != nil {
		t.Fatalf("failed to resolve root from subdir: %v", err)
	}

	cwdRoot, err := resolver.Resolve(cwd)
	if err != nil {
		t.Fatalf("failed to resolve cwd root: %v", err)
	}

	if root != cwdRoot {
		t.Errorf("subdirectory root != cwd root: %s vs %s", root, cwdRoot)
	}
}

func TestRootResolver_Nonexistent(t *testing.T) {
	resolver := reporoot.New()

	_, err := resolver.Resolve("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestRootResolver_EmptyInput(t *testing.T) {
	resolver := reporoot.New()

	_, err := resolver.Resolve("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestRootResolver_IsWithinRepo(t *testing.T) {
	resolver := reporoot.New()

	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot get cwd")
	}

	tests := []struct {
		path     string
		repo     string
		expected bool
	}{
		{cwd, cwd, true},
		{filepath.Join(cwd, "internal"), cwd, true},
		{filepath.Join(cwd, "cmd"), cwd, true},
		{"/tmp", cwd, false},
	}

	for _, tc := range tests {
		result := resolver.IsWithinRepo(tc.path, tc.repo)
		if result != tc.expected {
			t.Errorf("IsWithinRepo(%s, %s) = %v, want %v", tc.path, tc.repo, result, tc.expected)
		}
	}
}

func TestRootResolver_SplitRepoPath(t *testing.T) {
	resolver := reporoot.New()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Skip("cannot create .git dir")
	}
	if err := os.MkdirAll(filepath.Join(dir, "internal", "factory"), 0o755); err != nil {
		t.Skip("cannot create internal/factory")
	}

	root, rel, err := resolver.SplitRepoPath(dir)
	if err != nil {
		t.Fatalf("failed to split repo root path: %v", err)
	}

	if rel != "." {
		t.Errorf("relative path for root = %q, want \".\"", rel)
	}

	subdir := filepath.Join(dir, "internal", "factory")
	root2, rel2, err := resolver.SplitRepoPath(subdir)
	if err != nil {
		t.Fatalf("failed to split subdir path: %v", err)
	}

	if root2 != root {
		t.Errorf("root from subdir != root from repo root: %s vs %s", root2, root)
	}

	expected := filepath.Join("internal", "factory")
	if rel2 != expected {
		t.Errorf("relative path = %q, want %q", rel2, expected)
	}
}

// SplitRepoPath accepts a non-existent leaf below the repo root.
// This is the legitimate "future output leaf" lifecycle: the
// repository exists, the parent of the leaf exists, and the leaf
// is intentionally absent. The resolver walks upward to the
// nearest existing prefix and re-appends the missing suffix.
func TestRootResolver_SplitRepoPath_NonexistentLeafWalksUp(t *testing.T) {
	resolver := reporoot.New()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("cannot create .git dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "internal"), 0o755); err != nil {
		t.Fatalf("cannot create internal: %v", err)
	}

	futureLeaf := filepath.Join(dir, "internal", "future-leaf.txt")
	if _, err := os.Stat(futureLeaf); !os.IsNotExist(err) {
		t.Fatalf("precondition: future leaf must not exist (stat err=%v)", err)
	}

	root, rel, err := resolver.SplitRepoPath(futureLeaf)
	if err != nil {
		t.Fatalf("nonexistent leaf should walk up, got error: %v", err)
	}
	if root == "" {
		t.Fatalf("root must be set even when leaf is absent")
	}
	wantRel := filepath.Join("internal", "future-leaf.txt")
	if rel != wantRel {
		t.Fatalf("relative path = %q, want %q", rel, wantRel)
	}
}

// SplitRepoPath accepts a symlinked ancestor. The canonical
// form is resolved through EvalSymlinks so the relPath comes
// out in canonical coordinates rather than as ".." segments.
func TestRootResolver_SplitRepoPath_SymlinkedAncestor(t *testing.T) {
	resolver := reporoot.New()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("cannot create .git dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "internal", "factory"), 0o755); err != nil {
		t.Fatalf("cannot create internal/factory: %v", err)
	}

	// Build a symlink that aliases the repo root from another
	// location; the resolver must canonicalize through the
	// alias and produce a sensible relPath.
	linkRoot := t.TempDir()
	alias := filepath.Join(linkRoot, "alias")
	if err := os.Symlink(dir, alias); err != nil {
		t.Fatalf("symlink alias: %v", err)
	}

	subdirViaAlias := filepath.Join(alias, "internal", "factory")
	root, rel, err := resolver.SplitRepoPath(subdirViaAlias)
	if err != nil {
		t.Fatalf("symlinked ancestor should canonicalize, got error: %v", err)
	}
	if root == "" {
		t.Fatalf("root must be set")
	}
	wantRel := filepath.Join("internal", "factory")
	if rel != wantRel {
		t.Fatalf("relative path = %q, want %q (symlink must not leak into relPath)", rel, wantRel)
	}
}

// SplitRepoPath canonicalizes a symlinked leaf so the relPath
// is expressed in canonical coordinates, not in alias form.
func TestRootResolver_SplitRepoPath_CanonicalizesSymlinkedLeaf(t *testing.T) {
	resolver := reporoot.New()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("cannot create .git dir: %v", err)
	}
	real := filepath.Join(dir, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatalf("cannot create real: %v", err)
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	root, rel, err := resolver.SplitRepoPath(filepath.Join(alias, "sub"))
	if err != nil {
		t.Fatalf("symlinked leaf should canonicalize, got error: %v", err)
	}
	if root == "" {
		t.Fatalf("root must be set")
	}
	wantRel := filepath.Join("real", "sub")
	if rel != wantRel {
		t.Fatalf("relative path = %q, want %q (alias must not leak into relPath)", rel, wantRel)
	}
}

// SplitRepoPath rejects a symlink loop with a typed error. A
// loop on an existing prefix is an authority failure: the
// resolver cannot determine a canonical root, so it MUST surface
// the error rather than collapse to (root, "", nil).
func TestRootResolver_SplitRepoPath_SymlinkLoopFailsClosed(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("symlink loop test is not meaningful as root (root bypasses ELOOP)")
	}

	resolver := reporoot.New()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("cannot create .git dir: %v", err)
	}

	loopA := filepath.Join(dir, "loop-a")
	loopB := filepath.Join(dir, "loop-b")
	if err := os.Symlink(loopB, loopA); err != nil {
		t.Fatalf("symlink loop-a: %v", err)
	}
	if err := os.Symlink(loopA, loopB); err != nil {
		t.Fatalf("symlink loop-b: %v", err)
	}

	// Even though the .git lives in dir and dir is otherwise
	// normal, Resolve must walk through the loop when
	// canonicalizing absPath. The error MUST surface.
	_, _, err := resolver.SplitRepoPath(filepath.Join(loopA, "sub"))
	if err == nil {
		t.Fatalf("symlink loop must fail closed, got nil error")
	}
	if !errors.Is(err, reporoot.ErrCanonicalizeFailed) &&
		!errors.Is(err, reporoot.ErrWalkUpResolveFailed) &&
		!errors.Is(err, reporoot.ErrWalkUpLstatFailed) &&
		!errors.Is(err, reporoot.ErrSymlinkFailed) {
		t.Fatalf("expected a typed path-authority error, got %v", err)
	}
}

// SplitRepoPath rejects an empty absPath with a typed error.
func TestRootResolver_SplitRepoPath_EmptyInputFailsClosed(t *testing.T) {
	resolver := reporoot.New()

	_, _, err := resolver.SplitRepoPath("")
	if err == nil {
		t.Fatalf("empty absPath must fail closed")
	}
	if !errors.Is(err, reporoot.ErrEmptyInput) {
		t.Fatalf("expected ErrEmptyInput, got %v", err)
	}
}

// SplitRepoPath rejects a permission-denied ancestor with a
// typed error. A 0000-mode directory cannot be inspected by
// os.Lstat or os.Stat in any meaningful way, and the resolver
// MUST surface that as ErrWalkUpLstatFailed rather than walking
// upward as if the component did not exist.
func TestRootResolver_SplitRepoPath_PermissionDeniedFailsClosed(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-denied test is not meaningful as root (root bypasses DAC)")
	}

	resolver := reporoot.New()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("cannot create .git dir: %v", err)
	}

	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatalf("cannot create locked dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	// The leaf lives inside a 0000-mode directory; the resolver
	// MUST fail closed rather than silently walk upward.
	leaf := filepath.Join(locked, "future.txt")
	_, _, err := resolver.SplitRepoPath(leaf)
	if err == nil {
		t.Fatalf("permission-denied ancestor must fail closed, got nil error")
	}
	if !errors.Is(err, reporoot.ErrWalkUpLstatFailed) &&
		!errors.Is(err, reporoot.ErrWalkUpResolveFailed) &&
		!errors.Is(err, reporoot.ErrCanonicalizeFailed) {
		t.Fatalf("expected a typed path-authority error, got %v", err)
	}
}

// SplitRepoPath rejects a path whose canonicalized form (with
// missing suffix re-appended) cannot be resolved by the git-root
// lookup. The resolver MUST surface a typed path-authority error
// rather than silently collapse to (root, "", nil).
//
// On macOS the system tempdir lives under /var which is a symlink
// to /private/var, so a missing leaf below a fresh tempdir is the
// canonical case where the walk-up succeeds but the canonical
// reference passed back to Resolve still does not exist.
func TestRootResolver_SplitRepoPath_NonexistentCanonicalPathFailsClosed(t *testing.T) {
	resolver := reporoot.New()

	dir := t.TempDir()
	gone := filepath.Join(dir, "absent-leaf", "deeper-still")
	if _, err := os.Stat(gone); !os.IsNotExist(err) {
		t.Fatalf("precondition: gone must not exist (stat err=%v)", err)
	}

	_, _, err := resolver.SplitRepoPath(gone)
	if err == nil {
		t.Fatalf("absent leaf must fail closed, got nil error")
	}
	// Any typed path-authority failure is acceptable here; the
	// contract is "never silently collapse to (root, "", nil)".
	if !errors.Is(err, reporoot.ErrCanonicalizeFailed) &&
		!errors.Is(err, reporoot.ErrWalkUpResolveFailed) &&
		!errors.Is(err, reporoot.ErrWalkUpLstatFailed) &&
		!errors.Is(err, reporoot.ErrSymlinkFailed) &&
		!errors.Is(err, reporoot.ErrCanonicalizeNotExist) {
		t.Fatalf("expected a typed path-authority error, got %v", err)
	}
}

// SplitRepoPath rejects a fully nonexistent path with a typed
// error. The walk-up finds the filesystem root as the nearest
// existing prefix, but the canonicalized-with-suffix path is
// still entirely absent so Resolve fails closed.
func TestRootResolver_SplitRepoPath_FullyNonexistentFailsClosed(t *testing.T) {
	resolver := reporoot.New()

	gone := "/nonexistent-prefix-xyzzy-leamas-2588/sub/deeper"
	if _, err := os.Stat(gone); !os.IsNotExist(err) {
		t.Fatalf("precondition: gone must not exist (stat err=%v)", err)
	}

	_, _, err := resolver.SplitRepoPath(gone)
	if err == nil {
		t.Fatalf("fully nonexistent path must fail closed, got nil error")
	}
	if !errors.Is(err, reporoot.ErrCanonicalizeFailed) &&
		!errors.Is(err, reporoot.ErrWalkUpResolveFailed) &&
		!errors.Is(err, reporoot.ErrSymlinkFailed) {
		t.Fatalf("expected a typed path-authority error, got %v", err)
	}
}
