// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// range_mode_bypass_scenarios_test.go contains scenario tests for
// ACT-LEAMAS-FACTORY-TARGETED-DIGEST-RANGE-MODE-BYPASS-CORRECTION01.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExplicitRangeWithDirtyWorktree_DoesNotUseEmptyTree verifies that
// a normal successful range path does NOT use the empty tree comparison.
// This proves the correct behavior for a valid range; the actual error
// branch is tested by TestRangeDiffErrorBranch_EmptyTreeFallbackForbidden.
func TestExplicitRangeWithDirtyWorktree_DoesNotUseEmptyTree(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content\n"), 0644); err != nil {
		t.Fatalf("write file.txt: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "initial")

	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("content2\n"), 0644); err != nil {
		t.Fatalf("write file2.txt: %v", err)
	}
	runGit(t, dir, "add", "file2.txt")
	runGit(t, dir, "commit", "-m", "second commit")

	if err := os.MkdirAll(filepath.Join(dir, ".factory"), 0755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".factory", "gate-summary.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write gate-summary.json: %v", err)
	}

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate should succeed for valid range, got: %v", err)
	}

	if strings.Contains(out, "4b825dc642cb6eb9a060e54bf8d69288fbee4904") {
		t.Errorf("digest should not reference empty tree SHA")
	}
}

// TestExplicitRangeStagedDirty_ModeStaysRange tests explicit range with staged changes.
func TestExplicitRangeStagedDirty_ModeStaysRange(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("v1\n"), 0644); err != nil {
		t.Fatalf("write file1.txt: %v", err)
	}
	runGit(t, dir, "add", "file1.txt")
	runGit(t, dir, "commit", "-m", "commit 1")

	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("v2\n"), 0644); err != nil {
		t.Fatalf("write file2.txt: %v", err)
	}
	runGit(t, dir, "add", "file2.txt")
	runGit(t, dir, "commit", "-m", "commit 2")

	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("staged content\n"), 0644); err != nil {
		t.Fatalf("write staged.txt: %v", err)
	}
	runGit(t, dir, "add", "staged.txt")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(out, "Mode: range") {
		t.Errorf("expected Mode: range, got:\n%s", extractHeader(out))
	}

	if strings.Contains(out, "staged.txt") {
		t.Errorf("staged.txt should not appear in range digest")
	}
}

// TestExplicitRangeBothStagedAndUnstaged_ModeStaysRange
func TestExplicitRangeBothStagedAndUnstaged_ModeStaysRange(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("v1\n"), 0644); err != nil {
		t.Fatalf("write file1.txt: %v", err)
	}
	runGit(t, dir, "add", "file1.txt")
	runGit(t, dir, "commit", "-m", "commit 1")

	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("v2\n"), 0644); err != nil {
		t.Fatalf("write file2.txt: %v", err)
	}
	runGit(t, dir, "add", "file2.txt")
	runGit(t, dir, "commit", "-m", "commit 2")

	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("staged\n"), 0644); err != nil {
		t.Fatalf("write staged.txt: %v", err)
	}
	runGit(t, dir, "add", "staged.txt")

	if err := os.WriteFile(filepath.Join(dir, "unstaged.txt"), []byte("unstaged\n"), 0644); err != nil {
		t.Fatalf("write unstaged.txt: %v", err)
	}

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(out, "Mode: range") {
		t.Errorf("expected Mode: range, got:\n%s", extractHeader(out))
	}

	if strings.Contains(out, "staged.txt") || strings.Contains(out, "unstaged.txt") {
		t.Errorf("staged/unstaged files should not appear in range digest")
	}
}

// TestNoExplicitRangeDirtyWorktree_UsesDirtyMode verifies that when NO
// explicit range is supplied and worktree is dirty, the digest uses dirty mode.
func TestNoExplicitRangeDirtyWorktree_UsesDirtyMode(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("v1\n"), 0644); err != nil {
		t.Fatalf("write file.txt: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "commit 1")

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("v1\nv2\n"), 0644); err != nil {
		t.Fatalf("modify file.txt: %v", err)
	}

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeAuto,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(out, "Mode: dirty") {
		t.Errorf("expected Mode: dirty, got:\n%s", extractHeader(out))
	}
}

// TestInvalidRange_FailsClosed verifies that an invalid range spec produces an error.
func TestInvalidRange_FailsClosed(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content\n"), 0644); err != nil {
		t.Fatalf("write file.txt: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "initial")

	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty\n"), 0644); err != nil {
		t.Fatalf("write dirty.txt: %v", err)
	}

	_, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "does-not-exist..HEAD",
	})
	if err == nil {
		t.Fatalf("expected error for invalid range, got nil")
	}

	if !strings.Contains(err.Error(), "does-not-exist") && !strings.Contains(err.Error(), "range") {
		t.Errorf("error should mention the invalid range, got: %v", err)
	}
}

// TestRangeAddedFile tests that legitimate additions within a range work.
func TestRangeAddedFile(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("existing\n"), 0644); err != nil {
		t.Fatalf("write existing.txt: %v", err)
	}
	runGit(t, dir, "add", "existing.txt")
	runGit(t, dir, "commit", "-m", "initial")

	if err := os.WriteFile(filepath.Join(dir, "added.txt"), []byte("new content\n"), 0644); err != nil {
		t.Fatalf("write added.txt: %v", err)
	}
	runGit(t, dir, "add", "added.txt")
	runGit(t, dir, "commit", "-m", "add new file")

	if err := os.MkdirAll(filepath.Join(dir, ".factory"), 0755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".factory", "dirty.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write dirty.json: %v", err)
	}

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(out, "A  added.txt") {
		t.Errorf("added.txt should appear as added in manifest")
	}
	if !strings.Contains(out, "Mode: range") {
		t.Errorf("expected Mode: range")
	}
}

// TestRangeDeletedFile tests that legitimate deletions within a range work.
func TestRangeDeletedFile(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("existing\n"), 0644); err != nil {
		t.Fatalf("write existing.txt: %v", err)
	}
	runGit(t, dir, "add", "existing.txt")
	runGit(t, dir, "commit", "-m", "initial")

	runGit(t, dir, "rm", "existing.txt")
	runGit(t, dir, "commit", "-m", "delete file")

	if err := os.MkdirAll(filepath.Join(dir, ".factory"), 0755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".factory", "dirty.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write dirty.json: %v", err)
	}

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(out, "D  existing.txt") {
		t.Errorf("existing.txt should appear as deleted in manifest")
	}
	if !strings.Contains(out, "Mode: range") {
		t.Errorf("expected Mode: range")
	}
}
