// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sentinel content to verify full file is included
const sentinelContent = "SENTINEL_AT_END_EOF_CONTENT"

// largeFileContent creates content that exceeds the old MaxPreviewLines (200) limit.
func largeFileContent() string {
	var sb strings.Builder
	// Write 250 lines to exceed MaxPreviewLines (200)
	for i := 0; i < 250; i++ {
		sb.WriteString("line ")
		sb.WriteString(strings.Repeat("x", 50))
		sb.WriteString("\n")
	}
	// Add sentinel at the end
	sb.WriteString(sentinelContent)
	sb.WriteString("\n")
	return sb.String()
}

// TestDigest_UntrackedFile_IncludesFullContent verifies that untracked files
// are not truncated in digest output. This is a regression test for
// ACT-LEAMAS-FACTORY-DIGEST-FULL-FILE-CONTEXT01.
func TestDigest_UntrackedFile_IncludesFullContent(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create a large untracked file
	file := filepath.Join(tmpDir, "large.txt")
	content := largeFileContent()
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Generate digest
	digest, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify sentinel content is present (proves full file is included)
	if !strings.Contains(digest, sentinelContent) {
		t.Errorf("digest should contain sentinel content from end of file")
	}

	// Verify no truncation marker
	if strings.Contains(digest, "(truncated)") {
		t.Error("digest should not contain truncation marker for untracked files")
	}

	// Verify label says "file content" not "file preview"
	if strings.Contains(digest, "--- untracked file preview ---") {
		t.Error("digest should use 'file content' label, not 'file preview'")
	}
	if !strings.Contains(digest, "--- untracked file content ---") {
		t.Error("digest should contain 'untracked file content' label")
	}
}

// TestDigest_UntrackedFile_LargeBytes verifies that untracked
// files at the per-file cap boundary (just under
// MaxPerFileBytes) are still rendered fully.
//
// ACT-LEAMAS-TARGETED-DIGEST-RECURSIVE-EVIDENCE-GUARD01:
// untracked files exceeding MaxPerFileBytes (64 KiB) are now
// rendered as bounded stubs, NOT in full. This test asserts
// the new bounded semantics for files at/above the cap.
//
// Files under MaxPerFileBytes (the ~12 KB file in
// TestDigest_UntrackedFile_IncludesFullContent) continue to
// render fully.
func TestDigest_UntrackedFile_LargeBytes(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create a file larger than MaxFileSizeForFull (1 MiB)
	// so the size gate fires.
	file := filepath.Join(tmpDir, "large_bytes.txt")
	var sb strings.Builder
	sb.WriteString("START_MARKER\n")
	for i := 0; i < 40_000; i++ {
		sb.WriteString(strings.Repeat("ABCDEFGHIJ", 100))
		sb.WriteString("\n")
	}
	sb.WriteString("END_MARKER\n")
	content := sb.String()
	if int64(len(content)) <= MaxFileSizeForFull {
		t.Fatalf("test setup wrong: content size %d must "+
			"exceed MaxFileSizeForFull=%d",
			len(content), MaxFileSizeForFull)
	}

	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	digest, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// The full body must NOT be embedded (file is > 1 MiB).
	if strings.Contains(digest, content) {
		t.Errorf("digest should NOT contain full %d-byte "+
			"body", len(content))
	}
	// A bounded preview / metadata must be present.
	if !strings.Contains(digest, "LARGE_FILE_EVIDENCE_BOUNDED") &&
		!strings.Contains(digest, "Classification: BOUNDED_BODY") {
		t.Errorf("digest should classify large untracked " +
			"file as bounded")
	}
	// Identity (size) must be visible.
	if !strings.Contains(digest, "Bytes: ") {
		t.Errorf("digest should report file size in bounded stub")
	}
}

// TestDigest_StagedFile_IncludesUntruncatedDiff verifies staged file diff is shown without truncation.
func TestDigest_StagedFile_IncludesUntruncatedDiff(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create and commit initial file
	file := filepath.Join(tmpDir, "staged.txt")
	initialContent := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(file, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "staged.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Stage a large change
	var sb strings.Builder
	for i := 0; i < 300; i++ {
		sb.WriteString("new line content\n")
	}
	sb.WriteString(sentinelContent + "\n")
	stagedContent := sb.String()

	if err := os.WriteFile(file, []byte(stagedContent), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}
	runGit(t, tmpDir, "add", "staged.txt")

	digest, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeStaged,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify sentinel in staged diff
	if !strings.Contains(digest, sentinelContent) {
		t.Error("digest should contain sentinel content from staged change")
	}

	// Verify staged diff section exists
	if !strings.Contains(digest, "--- staged diff ---") {
		t.Error("digest should contain staged diff section")
	}
}

// TestDigest_UnstagedFile_IncludesUntruncatedDiff verifies unstaged file diff is shown without truncation.
func TestDigest_UnstagedFile_IncludesUntruncatedDiff(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create and commit initial file
	file := filepath.Join(tmpDir, "unstaged.txt")
	initialContent := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(file, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "unstaged.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Make unstaged changes
	var sb strings.Builder
	for i := 0; i < 300; i++ {
		sb.WriteString("modified line\n")
	}
	sb.WriteString(sentinelContent + "\n")
	modifiedContent := sb.String()

	if err := os.WriteFile(file, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	digest, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify sentinel in unstaged diff
	if !strings.Contains(digest, sentinelContent) {
		t.Error("digest should contain sentinel content from unstaged change")
	}

	// Verify unstaged diff section exists
	if !strings.Contains(digest, "--- unstaged diff ---") {
		t.Error("digest should contain unstaged diff section")
	}
}

// TestReadFileFull_Basic verifies the ReadFileFull function works correctly.
func TestReadFileFull_Basic(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with normal file
	file := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	result, isBinary := ReadFileFull(file)
	if isBinary {
		t.Error("text file should not be detected as binary")
	}
	if result != content {
		t.Errorf("ReadFileFull returned wrong content, got: %q", result)
	}
}

// TestReadFileFull_Binary verifies binary files are detected.
func TestReadFileFull_Binary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file with null bytes
	file := filepath.Join(tmpDir, "binary.bin")
	content := []byte{0x00, 0x01, 0x02, 0x03}
	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	result, isBinary := ReadFileFull(file)
	if !isBinary {
		t.Error("file with null bytes should be detected as binary")
	}
	if result != "" {
		t.Errorf("binary file should return empty content, got: %q", result)
	}
}

// TestReadFileFull_Large verifies large files are fully read.
func TestReadFileFull_Large(t *testing.T) {
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "large.txt")
	largeContent := largeFileContent()
	if err := os.WriteFile(file, []byte(largeContent), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	result, isBinary := ReadFileFull(file)
	if isBinary {
		t.Error("large text file should not be detected as binary")
	}
	if !strings.Contains(result, sentinelContent) {
		t.Error("ReadFileFull should include content past old MaxPreviewLines limit")
	}
	if !strings.HasSuffix(result, "\n") {
		t.Error("ReadFileFull should ensure trailing newline")
	}
}

// TestReadFileFull_Missing verifies missing file handling.
func TestReadFileFull_Missing(t *testing.T) {
	result, isBinary := ReadFileFull("/nonexistent/file.txt")
	if isBinary {
		t.Error("missing file should not be binary")
	}
	if result != "(file not present)\n" {
		t.Errorf("missing file should return error message, got: %q", result)
	}
}
