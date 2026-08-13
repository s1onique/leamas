// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

// fakeGitRunner is a closure-friendly in-process substitute for
// the bounded execution gateway. Tests construct one with a
// fixed (stdout, stderr, exit) tuple so each call returns the
// same canned output until the test changes the closure.
type fakeGitRunner struct {
	stdout   []byte
	stderr   []byte
	exitCode int
	err      error
	calls    int
	gotArgs  [][]string
}

func (f *fakeGitRunner) RunGit(_ context.Context, _ string, args ...string) (execution.GitResult, error) {
	f.calls++
	argCopy := make([]string, len(args))
	copy(argCopy, args)
	f.gotArgs = append(f.gotArgs, argCopy)
	if f.err != nil {
		return execution.GitResult{}, f.err
	}
	return execution.GitResult{Stdout: f.stdout, Stderr: f.stderr, ExitCode: f.exitCode}, nil
}

// singlePorcelain returns the NUL-byte representation of the
// supplied list of worktree paths. Each record is terminated
// by a double-NUL to match the on-the-wire format of
// `git worktree list --porcelain -z`.
func singlePorcelain(paths ...string) []byte {
	var sb strings.Builder
	for _, p := range paths {
		sb.WriteString("worktree ")
		sb.WriteString(p)
		sb.WriteString("\x00\x00")
	}
	return []byte(sb.String())
}

// TestInventoryRepositoryWorktrees_CommandStamped is the basic
// "the inventory authority calls the canonical command" test.
// The runner returns a successful exit code without producing
// real porcelain so the failure stops at command-stamp
// verification; full porcelain parsing is exercised by the
// other inventory tests.
func TestInventoryRepositoryWorktrees_CommandStamped(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(repoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &fakeGitRunner{stdout: nil, exitCode: 0}
	_, _ = InventoryRepositoryWorktrees(context.Background(), repoRoot, runner)
	if runner.calls != 1 {
		t.Fatalf("expected 1 git call, got %d", runner.calls)
	}
	got := runner.gotArgs[0]
	want := []string{"worktree", "list", "--porcelain", "-z"}
	if len(got) != len(want) {
		t.Fatalf("args length: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d: got %q want %q", i, got[i], want[i])
		}
	}
}

// TestInventoryRepositoryWorktrees_NilRunnerUsesDefault
// confirms the production nil-check path resolves to
// execution.RunGit. We cannot observe the call directly here,
// but we can guarantee the function does not panic.
func TestInventoryRepositoryWorktrees_NilRunnerUsesDefault(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil runner must not panic: %v", r)
		}
	}()
	if _, err := InventoryRepositoryWorktrees(context.Background(), "/nonexistent", nil); err == nil {
		// In rare sandboxes, a non-existent path MIGHT succeed
		// because /nonexistent is a string passed to RunGit
		// which forks git; we accept either branch here and
		// treat the lack of a panic as the contract.
		t.Log("default runner accepted path; no panic either way")
	}
}

// TestInventoryRepositoryWorktrees_EmptyOutputRejected confirms
// the parser never accepts an empty inventory.
func TestInventoryRepositoryWorktrees_EmptyOutputRejected(t *testing.T) {
	runner := &fakeGitRunner{}
	_, err := InventoryRepositoryWorktrees(context.Background(), "/tmp/repo", runner)
	if err == nil {
		t.Fatalf("expected error for empty output")
	}
	var vErr *V2VerifierError
	if !errors.As(err, &vErr) {
		t.Fatalf("error is not typed: %v", err)
	}
	if vErr.Diags[0].Code != V2VerifierWorktreeInventoryUnavailable {
		t.Fatalf("expected worktree_inventory_unavailable code, got %v", vErr.Diags[0].Code)
	}
}

// TestInventoryRepositoryWorktrees_SpawnFailure confirms the
// runner error path is mapped to a typed diagnostic.
func TestInventoryRepositoryWorktrees_SpawnFailure(t *testing.T) {
	runner := &fakeGitRunner{err: os.ErrPermission}
	_, err := InventoryRepositoryWorktrees(context.Background(), "/tmp/repo", runner)
	if err == nil {
		t.Fatalf("expected spawn failure")
	}
	var vErr *V2VerifierError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
	if !strings.Contains(vErr.Diags[0].Message, "permission denied") &&
		!strings.Contains(vErr.Diags[0].Message, "operation not permitted") {
		t.Fatalf("expected permission error, got %v", vErr.Diags[0].Message)
	}
}

// TestInventoryRepositoryWorktrees_Timeout simulates a
// context-deadline-exceeded by returning errCtx from the runner.
func TestInventoryRepositoryWorktrees_Timeout(t *testing.T) {
	runner := &fakeGitRunner{err: context.DeadlineExceeded}
	_, err := InventoryRepositoryWorktrees(context.Background(), "/tmp/repo", runner)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	var vErr *V2VerifierError
	if !errors.As(err, &vErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
}

// TestInventoryRepositoryWorktrees_NonZeroExit maps git's
// non-zero exit to the inventory-unavailable code.
func TestInventoryRepositoryWorktrees_NonZeroExit(t *testing.T) {
	runner := &fakeGitRunner{exitCode: 128, stderr: []byte("fatal: not a git repository")}
	_, err := InventoryRepositoryWorktrees(context.Background(), "/tmp/repo", runner)
	if err == nil {
		t.Fatalf("expected non-zero exit rejection")
	}
	if !strings.Contains(err.Error(), "exited 128") {
		t.Fatalf("expected exit-128 message, got %v", err)
	}
}

// TestInventoryRepositoryWorktrees_OutputOverflow rejects
// inventories exceeding worktreeInventoryMaxBytes.
func TestInventoryRepositoryWorktrees_OutputOverflow(t *testing.T) {
	huge := make([]byte, worktreeInventoryMaxBytes+1)
	for i := range huge {
		huge[i] = 'A'
	}
	runner := &fakeGitRunner{stdout: huge}
	_, err := InventoryRepositoryWorktrees(context.Background(), "/tmp/repo", runner)
	if err == nil {
		t.Fatalf("expected overflow rejection")
	}
	if !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("expected overflow message, got %v", err)
	}
}

// TestParseWorktreeInventory_AcceptsCanonicalPath confirms
// canonical absolute paths survive parsing.
func TestParseWorktreeInventory_AcceptsCanonicalPath(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	inv, err := parseWorktreeInventory(singlePorcelain(root, sub))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inv) != 2 {
		t.Fatalf("expected 2 roots, got %v", inv)
	}
}

// TestParseWorktreeInventory_RejectsRelativePath confirms
// the parser refuses a relative worktree path. Upstream git
// is documented to emit absolute paths; a relative path is
// forensic evidence the upstream command was hijacked.
func TestParseWorktreeInventory_RejectsRelativePath(t *testing.T) {
	_, err := parseWorktreeInventory([]byte("worktree .\x00"))
	if err == nil {
		t.Fatalf("expected rejection for relative worktree path")
	}
	if !strings.Contains(err.Error(), "relative") {
		t.Fatalf("expected relative-path error, got %v", err)
	}
}

// TestCanonicalizeWorktreeRoot_RejectsNonExistent confirms
// the helper never falls back to a lexical form for missing
// or unresolvable paths. Upstream git is expected to emit
// canonical worktree paths for live worktrees; a stale or
// prunable registration must not become an authoritative
// component of the inventory.
func TestCanonicalizeWorktreeRoot_RejectsNonExistent(t *testing.T) {
	_, err := canonicalizeWorktreeRoot("/nonexistent-root/leaf")
	if err == nil {
		t.Fatalf("expected rejection for non-existent worktree path")
	}
	if !strings.Contains(err.Error(), "unresolvable") {
		t.Fatalf("expected unresolvable-message, got %v", err)
	}
}

// TestParseWorktreeInventory_RejectsNewline carries the
// "command accidentally dropped -z" defense.
func TestParseWorktreeInventory_RejectsNewline(t *testing.T) {
	_, err := parseWorktreeInventory([]byte("worktree /tmp\n"))
	if err == nil {
		t.Fatalf("expected newline rejection")
	}
}

// TestParseWorktreeInventory_RejectsCarriageReturn is the
// Windows-safe companion to the newline rejection.
func TestParseWorktreeInventory_RejectsCarriageReturn(t *testing.T) {
	_, err := parseWorktreeInventory([]byte("worktree /tmp\r\x00"))
	if err == nil {
		t.Fatalf("expected CR rejection")
	}
}

// TestParseWorktreeInventory_RejectsMissingPrefix maps an
// upstream-future-protocol addition (e.g. "porcelain-v2") to a
// diagnostic. The CLI does not evolve the inventory format.
func TestParseWorktreeInventory_RejectsMissingPrefix(t *testing.T) {
	_, err := parseWorktreeInventory([]byte("porcelain-v2 /tmp\x00"))
	if err == nil {
		t.Fatalf("expected prefix rejection")
	}
}

// TestRepositoryWorktreeInventory_Contains confirms the
// containment check classifies both equality and strict
// descendants; non-members are rejected.
//
// The inventory stores canonical (post-EvalSymlinks) roots and
// the test compares against canonical identities. On macOS
// /var is a symlink to /private/var, so a lexical t.TempDir()
// path must be canonicalised before the containment check
// sees it; the helper does so via os.SameFile so the test
// exercises the production contract rather than the
// platform-specific lexical form.
func TestRepositoryWorktreeInventory_Contains(t *testing.T) {
	owner := t.TempDir()
	canonicalOwner, err := filepath.EvalSymlinks(owner)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", owner, err)
	}
	inv, err := newRepositoryWorktreeInventoryFromCanonical([]string{owner})
	if err != nil {
		t.Fatalf("NewRepositoryWorktreeInventory: %v", err)
	}
	mustContain := []string{
		canonicalOwner,
		filepath.Join(canonicalOwner, "file"),
		filepath.Join(canonicalOwner, "sub/dir/x"),
	}
	mustNot := []string{
		filepath.Join(filepath.Dir(canonicalOwner), "other"),
		canonicalOwner + "ed", // sibling with same prefix but not a child
		filepath.Dir(canonicalOwner),
	}
	for _, c := range mustContain {
		if !inv.Contains(c) {
			t.Fatalf("expected Contains(%q) = true", c)
		}
	}
	for _, c := range mustNot {
		if inv.Contains(c) {
			t.Fatalf("expected Contains(%q) = false", c)
		}
	}
}

// TestInventoryRoots_ReturnsCopy enforces that callers cannot
// mutate the inventory's underlying slice.
//
// The inventory stores canonical (post-EvalSymlinks) roots;
// the mutation probe therefore writes to a caller-visible
// copy and confirms the next RootsView() reflects the
// canonical form (not the lexical fixture form), proving the
// caller never escapes the inventory's internal slice.
func TestInventoryRoots_ReturnsCopy(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	if err := os.MkdirAll(a, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatal(err)
	}
	canonicalA, aerr := filepath.EvalSymlinks(a)
	if aerr != nil {
		t.Fatalf("EvalSymlinks(%q): %v", a, aerr)
	}
	canonicalB, berr := filepath.EvalSymlinks(b)
	if berr != nil {
		t.Fatalf("EvalSymlinks(%q): %v", b, berr)
	}
	inv, err := newRepositoryWorktreeInventoryFromCanonical([]string{a, b})
	if err != nil {
		t.Fatalf("NewRepositoryWorktreeInventory: %v", err)
	}
	c1 := inv.RootsView()
	c1[0] = filepath.Join(root, "c")
	c2 := inv.RootsView()
	if c2[0] != canonicalA {
		t.Fatalf("inventory was mutated through the copy: %v", c2)
	}
	if c2[1] != canonicalB {
		t.Fatalf("inventory second slot drifted: %v", c2)
	}
}

// TestCanonicalizeWorktreeRoot_NonExistentAbsentInsideTemp
// confirms the helper refuses a missing worktree path even
// when it sits under an existing parent (the parent exists
// but the immediate component does not). Upstream git is
// expected to emit canonical worktree paths only for live
// worktrees.
func TestCanonicalizeWorktreeRoot_NonExistentAbsentInsideTemp(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "no-such-dir")
	_, err := canonicalizeWorktreeRoot(missing)
	if err == nil {
		t.Fatalf("expected rejection for missing worktree path")
	}
	if !strings.Contains(err.Error(), "unresolvable") {
		t.Fatalf("expected unresolvable error, got %v", err)
	}
}

// TestCanonicalizeWorktreeRoot_SymlinkedRoot resolves a path
// whose component chain includes a symlink into the destination.
//
// canonicalizeWorktreeRoot resolves the FULL path including
// any symlinked ancestors (notably macOS /var -> /private/var).
// The contract is therefore "EvalSymlinks across the entire
// component chain", and the expected value is the canonical
// form of the target, not the lexical form of the target's
// parent.
func TestCanonicalizeWorktreeRoot_SymlinkedRoot(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	got, err := canonicalizeWorktreeRoot(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, werr := filepath.EvalSymlinks(target)
	if werr != nil {
		t.Fatalf("EvalSymlinks(%q): %v", target, werr)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// TestWorktreeInventoryCommand_MatchesSpec pins the literal
// upstream command line so future refactors cannot drift.
func TestWorktreeInventoryCommand_MatchesSpec(t *testing.T) {
	want := []string{"worktree", "list", "--porcelain", "-z"}
	got := worktreeInventoryCommand()
	if len(got) != len(want) {
		t.Fatalf("length: %v vs %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("idx %d: %q vs %q", i, got[i], want[i])
		}
	}
}
