// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGenerateDirtyDigest tests dirty mode digest generation.
func TestGenerateDirtyDigest(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	trackedFile := filepath.Join(tmpDir, "tracked.txt")
	if err := os.WriteFile(trackedFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("failed to write tracked file: %v", err)
	}
	runGit(t, tmpDir, "add", "tracked.txt")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	if err := os.WriteFile(trackedFile, []byte("initial content\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify tracked file: %v", err)
	}

	untrackedFile := filepath.Join(tmpDir, "untracked.md")
	if err := os.WriteFile(untrackedFile, []byte("# Untracked\n\nContent here.\n"), 0644); err != nil {
		t.Fatalf("failed to write untracked file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "# Targeted digest") {
		t.Error("missing header")
	}
	if !strings.Contains(content, "Mode: dirty") {
		t.Error("missing mode")
	}
	if !strings.Contains(content, "tracked.txt") {
		t.Error("missing tracked file")
	}
	if !strings.Contains(content, "untracked.md") {
		t.Error("missing untracked file")
	}
	if !strings.Contains(content, "--- unstaged diff ---") {
		t.Error("missing unstaged diff")
	}
	if !strings.Contains(content, "--- untracked file preview ---") {
		t.Error("missing untracked preview")
	}
}

// TestGenerateStagedDigest tests staged mode digest generation.
func TestGenerateStagedDigest(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

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

	if err := os.WriteFile(trackedFile, []byte("initial\nstaged change\nunstaged change\n"), 0644); err != nil {
		t.Fatalf("failed to modify file again: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeStaged,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "Mode: staged") {
		t.Error("missing staged mode")
	}
	if !strings.Contains(content, "staged.txt") {
		t.Error("missing staged file")
	}
	if !strings.Contains(content, "--- staged diff ---") {
		t.Error("missing staged diff")
	}
	if strings.Contains(content, "unstaged change") {
		t.Error("staged digest should not include unstaged-only changes")
	}
}

// TestStagedDigestExcludesUnstagedOnly verifies staged mode excludes unstaged-only changes.
func TestStagedDigestExcludesUnstagedOnly(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	trackedFile := filepath.Join(tmpDir, "only-unstaged.txt")
	if err := os.WriteFile(trackedFile, []byte("initial\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "only-unstaged.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	if err := os.WriteFile(trackedFile, []byte("initial\nunstaged-only\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeStaged,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if strings.Contains(content, "only-unstaged.txt") {
		t.Error("staged digest should not include unstaged-only changes")
	}
}

// TestUntrackedFilesExcludedFromIgnored verifies ignored files are excluded.
func TestUntrackedFilesExcludedFromIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	gitignore := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("ignored.txt\n"), 0644); err != nil {
		t.Fatalf("failed to write .gitignore: %v", err)
	}
	runGit(t, tmpDir, "add", ".gitignore")
	runGit(t, tmpDir, "commit", "-m", "add gitignore")

	ignoredFile := filepath.Join(tmpDir, "ignored.txt")
	if err := os.WriteFile(ignoredFile, []byte("ignored content\n"), 0644); err != nil {
		t.Fatalf("failed to write ignored file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if strings.Contains(content, "ignored.txt") {
		t.Error("digest should not include ignored files")
	}
}

// TestWriteDigest tests the Write function.
func TestWriteDigest(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	outputFile := filepath.Join(tmpDir, "build", "digest.txt")

	err := Write(Options{
		RepoRoot: tmpDir,
		Mode:     ModeStaged,
		Output:   outputFile,
	})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# Targeted digest") {
		t.Error("output file missing header")
	}
}

// TestBinaryUntrackedFile tests that binary files get summarized.
func TestBinaryUntrackedFile(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	binaryFile := filepath.Join(tmpDir, "binary.dat")
	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	if err := os.WriteFile(binaryFile, binaryContent, 0644); err != nil {
		t.Fatalf("failed to write binary file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "(binary file)") {
		t.Error("binary file should be summarized")
	}
}

// TestEmptyDigest tests handling of no changes.
func TestEmptyDigest(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	trackedFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(trackedFile, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "file.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "No changed files found") {
		t.Error("should indicate no changed files")
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

// TestPreviewFile tests file preview functionality.
func TestPreviewFile(t *testing.T) {
	tmpDir := t.TempDir()

	textFile := filepath.Join(tmpDir, "test.txt")
	content := strings.Repeat("line\n", 300)
	if err := os.WriteFile(textFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	preview, isBinary := PreviewFile(textFile, 1024, 200)
	if isBinary {
		t.Error("text file should not be binary")
	}
	if len(preview) > 1024 {
		t.Error("preview should respect byte limit")
	}

	binaryFile := filepath.Join(tmpDir, "binary.bin")
	binaryData := []byte{0x00, 0x01, 0x02, 0x00}
	if err := os.WriteFile(binaryFile, binaryData, 0644); err != nil {
		t.Fatalf("failed to write binary file: %v", err)
	}

	isBinary = IsBinary(binaryFile)
	if !isBinary {
		t.Error("should detect binary file")
	}
}

// TestDigestTimestampFormat verifies timestamp is RFC3339 format.
func TestDigestTimestampFormat(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	lines := strings.Split(content, "\n")
	var foundTimestamp bool
	for _, line := range lines {
		if strings.HasPrefix(line, "Generated at:") {
			foundTimestamp = true
			ts := strings.TrimPrefix(line, "Generated at:")
			ts = strings.TrimSpace(ts)
			_, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				t.Errorf("timestamp not in RFC3339 format: %s", ts)
			}
			break
		}
	}
	if !foundTimestamp {
		t.Error("missing generated timestamp")
	}
}

// TestChangedFileOrdering verifies files are sorted correctly.
func TestChangedFileOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	files := map[string]string{
		"zebra.txt":    "tracked",
		"apple.txt":    "tracked",
		"untracked.md": "untracked",
		"banana.md":    "untracked",
	}

	for name, status := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("content\n"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
		if status == "tracked" {
			runGit(t, tmpDir, "add", name)
		}
	}
	runGit(t, tmpDir, "commit", "-m", "add tracked")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	changedIdx := strings.Index(content, "## Changed files")
	diffsIdx := strings.Index(content, "## Diffs")
	if changedIdx == -1 || diffsIdx == -1 {
		t.Fatal("missing sections")
	}

	changedSection := content[changedIdx:diffsIdx]

	applePos := strings.Index(changedSection, "apple.txt")
	zebraPos := strings.Index(changedSection, "zebra.txt")
	bananaPos := strings.Index(changedSection, "banana.md")
	untrackedPos := strings.Index(changedSection, "untracked.md")

	if applePos > bananaPos {
		t.Error("tracked files should come before untracked")
	}
	if zebraPos > bananaPos {
		t.Error("tracked files should come before untracked")
	}
	if applePos > zebraPos {
		t.Error("tracked files should be alphabetically sorted")
	}

	if bananaPos == -1 || untrackedPos == -1 {
		t.Error("both untracked files should be in changed section")
	}
}

// Helper functions

func initGitRepo(t *testing.T, dir string) {
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
