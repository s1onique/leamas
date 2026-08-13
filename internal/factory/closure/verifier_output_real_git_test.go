// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInventoryRepositoryWorktrees_RealGitPorcelainZAndNewlinePath(t *testing.T) {
	root := t.TempDir()
	runGitTestCommand(t, root, "init", "-q")
	runGitTestCommand(t, root, "config", "user.email", "test@example.invalid")
	runGitTestCommand(t, root, "config", "user.name", "Leamas Test")
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitTestCommand(t, root, "add", "README")
	runGitTestCommand(t, root, "commit", "-qm", "fixture")
	linked := filepath.Join(t.TempDir(), "linked\nworktree")
	runGitTestCommand(t, root, "worktree", "add", "--detach", "-q", linked)

	cmd := exec.Command("git", "-C", root, "worktree", "list", "--porcelain", "-z")
	raw, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("\x00\x00")) {
		t.Fatalf("real git output lacks record framing: %q", raw)
	}
	inv, err := InventoryRepositoryWorktrees(context.Background(), root, &fakeGitRunner{stdout: raw})
	if err != nil {
		t.Fatal(err)
	}
	// The inventory contract stores canonical (post-EvalSymlinks)
	// worktree roots. On macOS /var is a symlink to /private/var,
	// so the canonical form of `linked` may differ from its lexical
	// t.TempDir()-derived spelling. Compare via canonical identity
	// so the test exercises the production contract rather than
	// the platform-specific lexical form.
	canonicalLinked, lerr := filepath.EvalSymlinks(linked)
	if lerr != nil {
		t.Fatalf("EvalSymlinks(%q): %v", linked, lerr)
	}
	found := false
	for _, path := range inv.RootsView() {
		if path == canonicalLinked {
			found = true
		}
	}
	if !found {
		t.Fatalf("newline-containing linked worktree missing: %q", inv.RootsView())
	}
}

func runGitTestCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v (%s)", args, err, out)
	}
}
