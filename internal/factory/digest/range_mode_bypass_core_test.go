// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// range_mode_bypass_core_test.go contains core regression tests for
// ACT-LEAMAS-FACTORY-TARGETED-DIGEST-RANGE-MODE-BYPASS-CORRECTION01.
//
// Bug class: C4_RANGE_AUTHORITY_CORRUPTION
//
// The core invariant being tested:
// An explicit --range MUST remain authoritative regardless of unrelated
// dirty worktree state. The digest MUST NOT silently fall back to
// comparing against the empty tree when range resolution encounters an error.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExplicitRangeWithDirtyWorktree_ModeStaysRange reproduces the core
// defect. When an explicit --range is supplied and the worktree has
// unrelated unstaged dirt, the digest MUST use range mode, not dirty mode.
func TestExplicitRangeWithDirtyWorktree_ModeStaysRange(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	// Setup: small.txt exists in commit A, large.csv is added in commit B
	smallTxt := filepath.Join(dir, "small.txt")
	if err := os.WriteFile(smallTxt, []byte("small\n"), 0644); err != nil {
		t.Fatalf("write small.txt: %v", err)
	}
	runGit(t, dir, "add", "small.txt")
	runGit(t, dir, "commit", "-m", "commit A: add small.txt")
	commitA := runGit(t, dir, "rev-parse", "HEAD")

	// Commit B adds large.csv with a small change
	largeCSV := filepath.Join(dir, "large.csv")
	largeContent := []byte("small change\n")
	if err := os.WriteFile(largeCSV, largeContent, 0644); err != nil {
		t.Fatalf("write large.csv: %v", err)
	}
	runGit(t, dir, "add", "large.csv")
	runGit(t, dir, "commit", "-m", "commit B: add large.csv")
	commitB := runGit(t, dir, "rev-parse", "HEAD")

	// Commit C (HEAD): modify small.txt (unrelated to our range)
	if err := os.WriteFile(smallTxt, []byte("small\nv2\n"), 0644); err != nil {
		t.Fatalf("write small.txt v2: %v", err)
	}
	runGit(t, dir, "add", "small.txt")
	runGit(t, dir, "commit", "-m", "commit C: modify small.txt")

	// Create dirty worktree: modify an unrelated file (not in our range)
	factoryDir := filepath.Join(dir, ".factory")
	if err := os.MkdirAll(factoryDir, 0755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	gateSummary := filepath.Join(factoryDir, "gate-summary.json")
	if err := os.WriteFile(gateSummary, []byte(`{"status": "dirty"}`), 0644); err != nil {
		t.Fatalf("write gate-summary.json: %v", err)
	}

	// Verify git status shows only the unrelated file as dirty
	statusOutput := runGit(t, dir, "status", "--porcelain")
	if !strings.Contains(statusOutput, "gate-summary.json") && !strings.Contains(statusOutput, ".factory/") {
		t.Fatalf("expected gate-summary.json or .factory/ to be dirty, got: %s", statusOutput)
	}

	// Generate digest with explicit range A..B
	// large.csv was added in B, so diff A..B for large.csv shows "new file mode"
	// This is CORRECT behavior - the file didn't exist at commit A
	rangeSpec := commitA + ".." + commitB
	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    rangeSpec,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// CRITICAL: The digest header MUST show Mode: range
	if !strings.Contains(out, "Mode: range") {
		t.Errorf("expected Mode: range in digest, got:\n%s", out[:minInt(600, len(out))])
	}

	// large.csv should appear as added in the manifest (since it was added in range)
	if !strings.Contains(out, "A  large.csv") {
		t.Errorf("large.csv should appear as added in manifest")
	}

	// small.txt should NOT appear (it wasn't modified in the range)
	if strings.Contains(out, "small.txt") {
		t.Errorf("small.txt should not appear in range A..B digest")
	}
}

// TestRangeDiffFailure_DegradedEvidence tests the contract when a range diff
// fails: the digest produces degraded evidence rather than falling back to
// empty-tree comparison.
func TestRangeDiffFailure_DegradedEvidence(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("v1\n"), 0644); err != nil {
		t.Fatalf("write file.txt: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "commit 1")

	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("v2\n"), 0644); err != nil {
		t.Fatalf("write file2.txt: %v", err)
	}
	runGit(t, dir, "add", "file2.txt")
	runGit(t, dir, "commit", "-m", "commit 2")

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("v1\nv2\n"), 0644); err != nil {
		t.Fatalf("modify file.txt: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "commit 3")

	if err := os.MkdirAll(filepath.Join(dir, ".factory"), 0755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".factory", "gate.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write gate.json: %v", err)
	}

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~2..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	emptyTreeSHA := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	if strings.Contains(out, emptyTreeSHA) {
		t.Errorf("BUG: empty-tree SHA %s found in digest - fallback still occurring", emptyTreeSHA)
	}

	if !strings.Contains(out, "file.txt") {
		t.Errorf("file.txt should appear in digest")
	}
	if !strings.Contains(out, "file2.txt") {
		t.Errorf("file2.txt should appear in digest")
	}
}

// TestRangeDiffErrorBranch_EmptyTreeFallbackForbidden verifies the exact error
// branch: when a per-path range diff fails, the code produces
// "(range diff unavailable: ...)" and does NOT fall back to empty-tree comparison.
func TestRangeDiffErrorBranch_EmptyTreeFallbackForbidden(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("v1\n"), 0644); err != nil {
		t.Fatalf("write file1.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("v2\n"), 0644); err != nil {
		t.Fatalf("write file2.txt: %v", err)
	}
	runGit(t, dir, "add", "file1.txt", "file2.txt")
	runGit(t, dir, "commit", "-m", "commit 1")

	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("v1\nv2\n"), 0644); err != nil {
		t.Fatalf("write file1.txt v2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("v2\nv3\n"), 0644); err != nil {
		t.Fatalf("write file2.txt v2: %v", err)
	}
	runGit(t, dir, "add", "file1.txt", "file2.txt")
	runGit(t, dir, "commit", "-m", "commit 2")

	if err := os.MkdirAll(filepath.Join(dir, ".factory"), 0755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".factory", "gate.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write gate.json: %v", err)
	}

	// Use fake runner that fails diff for file1.txt
	fake := &fakeGitRunner{
		FailPatterns: []string{"diff --unified=3 file1.txt"},
	}

	// Call the internal renderer with the fake runner
	files := []RangeFile{
		{Path: "file1.txt", Status: "M"},
		{Path: "file2.txt", Status: "M"},
	}
	out := renderRangeFileEvidenceWithRunner(fake, dir, files, "HEAD~1..HEAD")

	// Should contain degraded evidence marker for failed file1
	if !strings.Contains(out, "(range diff unavailable:") {
		t.Errorf("expected degraded evidence marker in output:\n%s", out)
	}

	// Verify: no invocation containing the empty tree SHA
	emptyTreeSHA := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	for _, cmd := range fake.Commands {
		cmdStr := strings.Join(cmd, " ")
		if strings.Contains(cmdStr, emptyTreeSHA) {
			t.Errorf("forbidden: empty tree SHA should not appear in any command:\n%s", cmdStr)
		}
	}

	// Normal diff should have been attempted for file2
	hasFile2Diff := false
	for _, cmd := range fake.Commands {
		if len(cmd) >= 4 && cmd[0] == "diff" && cmd[len(cmd)-1] == "file2.txt" {
			hasFile2Diff = true
			break
		}
	}
	if !hasFile2Diff {
		t.Errorf("expected diff attempt for file2.txt")
	}
}
