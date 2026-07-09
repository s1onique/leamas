// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUniqueRangeFiles_DedupesDuplicates tests that UniqueRangeFiles removes duplicates.
func TestUniqueRangeFiles_DedupesDuplicates(t *testing.T) {
	files := []RangeFile{
		{Path: "a.txt", Status: "added"},
		{Path: "b.txt", Status: "added"},
		{Path: "a.txt", Status: "added"}, // duplicate
		{Path: "c.txt", Status: "modified"},
		{Path: "b.txt", Status: "added"}, // duplicate
	}

	result := UniqueRangeFiles(files)

	if len(result) != 3 {
		t.Errorf("expected 3 unique files, got %d", len(result))
	}

	// Verify order is preserved (first-seen)
	if result[0].Path != "a.txt" {
		t.Errorf("expected first file to be a.txt, got %s", result[0].Path)
	}
	if result[1].Path != "b.txt" {
		t.Errorf("expected second file to be b.txt, got %s", result[1].Path)
	}
	if result[2].Path != "c.txt" {
		t.Errorf("expected third file to be c.txt, got %s", result[2].Path)
	}
}

// TestUniqueRangeFiles_EmptySlice returns empty slice.
func TestUniqueRangeFiles_EmptySlice(t *testing.T) {
	result := UniqueRangeFiles([]RangeFile{})
	if len(result) != 0 {
		t.Errorf("expected 0 files, got %d", len(result))
	}
}

// TestUniqueRangeFiles_SingleElement returns same element.
func TestUniqueRangeFiles_SingleElement(t *testing.T) {
	files := []RangeFile{{Path: "only.txt", Status: "added"}}
	result := UniqueRangeFiles(files)
	if len(result) != 1 {
		t.Errorf("expected 1 file, got %d", len(result))
	}
	if result[0].Path != "only.txt" {
		t.Errorf("expected path to be only.txt, got %s", result[0].Path)
	}
}

// TestRangeDigest_DedupesAddedFilesInInventory tests that range digest de-duplicates added files in inventory.
func TestRangeDigest_DedupesAddedFilesInInventory(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create initial commit
	initialFile := filepath.Join(tmpDir, "initial.txt")
	if err := os.WriteFile(initialFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runGit(t, tmpDir, "add", "initial.txt")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	// Create second commit with added files
	added1 := filepath.Join(tmpDir, "added1.txt")
	added2 := filepath.Join(tmpDir, "added2.txt")
	if err := os.WriteFile(added1, []byte("added file 1\n"), 0644); err != nil {
		t.Fatalf("failed to write added1: %v", err)
	}
	if err := os.WriteFile(added2, []byte("added file 2\n"), 0644); err != nil {
		t.Fatalf("failed to write added2: %v", err)
	}
	runGit(t, tmpDir, "add", "added1.txt")
	runGit(t, tmpDir, "add", "added2.txt")
	runGit(t, tmpDir, "commit", "-m", "add two files")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Count occurrences of each file in "Changed files" section
	lines := strings.Split(content, "\n")
	inChangedFiles := false
	added1Count := 0
	added2Count := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "## Changed files") {
			inChangedFiles = true
			continue
		}
		if strings.HasPrefix(line, "## Diffs") {
			break
		}
		if inChangedFiles && strings.Contains(line, "added1.txt") {
			added1Count++
		}
		if inChangedFiles && strings.Contains(line, "added2.txt") {
			added2Count++
		}
	}

	if added1Count != 1 {
		t.Errorf("added1.txt should appear exactly once in inventory, got %d", added1Count)
	}
	if added2Count != 1 {
		t.Errorf("added2.txt should appear exactly once in inventory, got %d", added2Count)
	}
}

// TestRangeDigest_DedupesAddedFileDiffBlocks tests that range digest de-duplicates diff blocks.
func TestRangeDigest_DedupesAddedFileDiffBlocks(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create initial commit
	initialFile := filepath.Join(tmpDir, "initial.txt")
	if err := os.WriteFile(initialFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runGit(t, tmpDir, "add", "initial.txt")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	// Create second commit with added files
	added1 := filepath.Join(tmpDir, "added1.txt")
	added2 := filepath.Join(tmpDir, "added2.txt")
	if err := os.WriteFile(added1, []byte("added file 1\n"), 0644); err != nil {
		t.Fatalf("failed to write added1: %v", err)
	}
	if err := os.WriteFile(added2, []byte("added file 2\n"), 0644); err != nil {
		t.Fatalf("failed to write added2: %v", err)
	}
	runGit(t, tmpDir, "add", "added1.txt")
	runGit(t, tmpDir, "add", "added2.txt")
	runGit(t, tmpDir, "commit", "-m", "add two files")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Count diff block headers for each file
	added1DiffCount := strings.Count(content, "=== added1.txt ===")
	added2DiffCount := strings.Count(content, "=== added2.txt ===")

	if added1DiffCount != 1 {
		t.Errorf("added1.txt diff block should appear exactly once, got %d", added1DiffCount)
	}
	if added2DiffCount != 1 {
		t.Errorf("added2.txt diff block should appear exactly once, got %d", added2DiffCount)
	}
}

// TestRangeDigest_DedupesMultipleAddedFilesWithoutDroppingAny tests that no files are dropped during dedupe.
func TestRangeDigest_DedupesMultipleAddedFilesWithoutDroppingAny(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create initial commit
	initialFile := filepath.Join(tmpDir, "initial.txt")
	if err := os.WriteFile(initialFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runGit(t, tmpDir, "add", "initial.txt")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	// Create second commit with multiple added files
	paths := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt"}
	for _, name := range paths {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("content for "+name+"\n"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
		runGit(t, tmpDir, "add", name)
	}
	runGit(t, tmpDir, "commit", "-m", "add five files")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify all files appear exactly once in inventory
	for _, name := range paths {
		count := strings.Count(content, name+"  [added]")
		if count != 1 {
			t.Errorf("%s should appear exactly once in inventory, got %d", name, count)
		}
	}

	// Verify all files appear exactly once in diff blocks
	for _, name := range paths {
		count := strings.Count(content, "=== "+name+" ===")
		if count != 1 {
			t.Errorf("%s diff block should appear exactly once, got %d", name, count)
		}
	}
}

// TestRangeDigest_IntegrationWithRealGit tests range digest with real Git repository.
func TestRangeDigest_IntegrationWithRealGit(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Commit 1: base
	baseFile := filepath.Join(tmpDir, "base.txt")
	if err := os.WriteFile(baseFile, []byte("base content\n"), 0644); err != nil {
		t.Fatalf("failed to write base file: %v", err)
	}
	runGit(t, tmpDir, "add", "base.txt")
	runGit(t, tmpDir, "commit", "-m", "commit 1: base")

	// Commit 2: add two files
	newFile1 := filepath.Join(tmpDir, "new1.txt")
	newFile2 := filepath.Join(tmpDir, "new2.txt")
	if err := os.WriteFile(newFile1, []byte("new file 1\n"), 0644); err != nil {
		t.Fatalf("failed to write new1: %v", err)
	}
	if err := os.WriteFile(newFile2, []byte("new file 2\n"), 0644); err != nil {
		t.Fatalf("failed to write new2: %v", err)
	}
	runGit(t, tmpDir, "add", "new1.txt")
	runGit(t, tmpDir, "add", "new2.txt")
	runGit(t, tmpDir, "commit", "-m", "commit 2: add two files")

	// Run range digest for HEAD~1..HEAD
	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify mode
	if !strings.Contains(content, "Mode: range") {
		t.Error("should contain Mode: range")
	}

	// Verify each file appears exactly once in inventory
	new1InvCount := strings.Count(content, "new1.txt  [added]")
	new2InvCount := strings.Count(content, "new2.txt  [added]")
	if new1InvCount != 1 {
		t.Errorf("new1.txt should appear once in inventory, got %d", new1InvCount)
	}
	if new2InvCount != 1 {
		t.Errorf("new2.txt should appear once in inventory, got %d", new2InvCount)
	}

	// Verify each file appears exactly once in diff blocks
	new1DiffCount := strings.Count(content, "=== new1.txt ===")
	new2DiffCount := strings.Count(content, "=== new2.txt ===")
	if new1DiffCount != 1 {
		t.Errorf("new1.txt should appear once in diff blocks, got %d", new1DiffCount)
	}
	if new2DiffCount != 1 {
		t.Errorf("new2.txt should appear once in diff blocks, got %d", new2DiffCount)
	}
}
