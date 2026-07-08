// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultAutoDirtyUsesDirtyMode tests auto mode with dirty working tree.
func TestDefaultAutoDirtyUsesDirtyMode(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	trackedFile := filepath.Join(tmpDir, "tracked.txt")
	if err := os.WriteFile(trackedFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("failed to write tracked file: %v", err)
	}
	runGit(t, tmpDir, "add", "tracked.txt")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	if err := os.WriteFile(trackedFile, []byte("initial content\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify tracked file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeAuto,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "Mode: dirty") {
		t.Error("auto mode should resolve to dirty mode when working tree has changes")
	}
	if !strings.Contains(content, "Resolved from: auto") {
		t.Error("auto mode should include Resolved from: auto")
	}
	if !strings.Contains(content, "Reason: working tree has changes") {
		t.Error("auto mode should include reason for dirty resolution")
	}
}

// TestDefaultAutoCleanUsesPreviousCommitRange tests auto mode with clean working tree.
func TestDefaultAutoCleanUsesPreviousCommitRange(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("first file\n"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	runGit(t, tmpDir, "add", "file1.txt")
	runGit(t, tmpDir, "commit", "-m", "first commit")

	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("second file\n"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	runGit(t, tmpDir, "add", "file2.txt")
	runGit(t, tmpDir, "commit", "-m", "second commit")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeAuto,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "Mode: range") {
		t.Error("auto mode should resolve to range mode when working tree is clean")
	}
	if !strings.Contains(content, "Range: HEAD~1..HEAD") {
		t.Error("auto mode should use HEAD~1..HEAD range")
	}
	if !strings.Contains(content, "Resolved from: auto") {
		t.Error("auto mode should include Resolved from: auto")
	}
	if !strings.Contains(content, "file2.txt") {
		t.Error("should include file2.txt from the last commit")
	}
}

// TestExplicitDirtyStillWorks tests explicit dirty mode.
func TestExplicitDirtyStillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	trackedFile := filepath.Join(tmpDir, "tracked.txt")
	if err := os.WriteFile(trackedFile, []byte("initial\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "tracked.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	if err := os.WriteFile(trackedFile, []byte("initial\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "Mode: dirty") {
		t.Error("explicit dirty mode should show Mode: dirty")
	}
	if strings.Contains(content, "Resolved from:") {
		t.Error("explicit dirty mode should not show Resolved from")
	}
}

// TestExplicitStagedStillWorks tests explicit staged mode.
func TestExplicitStagedStillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	trackedFile := filepath.Join(tmpDir, "staged.txt")
	if err := os.WriteFile(trackedFile, []byte("initial\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "staged.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	if err := os.WriteFile(trackedFile, []byte("initial\nstaged change\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}
	runGit(t, tmpDir, "add", "staged.txt")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeStaged,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "Mode: staged") {
		t.Error("explicit staged mode should show Mode: staged")
	}
	if strings.Contains(content, "Resolved from:") {
		t.Error("explicit staged mode should not show Resolved from")
	}
}

// TestCleanRepoWithoutParentHandledHonestly tests initial commit error.
func TestCleanRepoWithoutParentHandledHonestly(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	file := filepath.Join(tmpDir, "initial.txt")
	if err := os.WriteFile(file, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "initial.txt")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	_, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeAuto,
	})

	if err == nil {
		t.Error("expected error for clean repo with only one commit")
	}
	if !strings.Contains(err.Error(), "only one commit") {
		t.Error("error should mention single commit limitation")
	}
}

// TestFilenamesWithSpacesAreHandled tests files with spaces.
func TestFilenamesWithSpacesAreHandled(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	fileWithSpaces := filepath.Join(tmpDir, "file with spaces.txt")
	if err := os.WriteFile(fileWithSpaces, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "file with spaces.txt")
	runGit(t, tmpDir, "commit", "-m", "add file with spaces")

	if err := os.WriteFile(fileWithSpaces, []byte("content\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "file with spaces.txt") {
		t.Error("should handle file with spaces in name")
	}
}

// TestRangeMode tests explicit range mode.
func TestRangeMode(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("first\n"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	runGit(t, tmpDir, "add", "file1.txt")
	runGit(t, tmpDir, "commit", "-m", "first")

	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("second\n"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	runGit(t, tmpDir, "add", "file2.txt")
	runGit(t, tmpDir, "commit", "-m", "second")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "Mode: range") {
		t.Error("range mode should show Mode: range")
	}
	if strings.Contains(content, "Resolved from: auto") {
		t.Error("explicit range mode should not show 'Resolved from: auto'")
	}
}

// TestDetectRepoRoot tests repo root detection.
func TestDetectRepoRoot(t *testing.T) {
	repoRoot, err := DetectRepoRoot()
	if err != nil {
		t.Fatalf("DetectRepoRoot failed: %v", err)
	}

	if repoRoot == "" {
		t.Error("repo root should not be empty")
	}
}

// Helper functions

func initGit(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed in %s: %v", args, dir, err)
	}
}
