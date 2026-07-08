package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/redact"
)

// TestDigestRedactsSecrets tests that digest output redacts secrets.
func TestDigestRedactsSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create and commit a clean baseline file
	cleanFile := filepath.Join(tmpDir, "config.json")
	cleanContent := `{
  "name": "test-project",
  "version": "1.0.0"
}`
	if err := os.WriteFile(cleanFile, []byte(cleanContent), 0644); err != nil {
		t.Fatalf("failed to write clean file: %v", err)
	}
	runGit(t, tmpDir, "add", "config.json")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	// Modify tracked file to include fake secrets
	secretContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "api_key": "sk-1234567890abcdefghijklmnopqrstuv",
  "password": "superSecretPassword123",
  "github_token": "ghp_1234567890abcdefghijklmnopqrstuvwxyzAB"
}`
	if err := os.WriteFile(cleanFile, []byte(secretContent), 0644); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	// Generate dirty digest
	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Apply redaction (simulating what Write() does)
	redacted := redact.RedactDigest(content)

	// Verify secrets are redacted
	if strings.Contains(redacted, "sk-1234567890") {
		t.Error("OpenAI API key should be redacted")
	}
	if strings.Contains(redacted, "superSecretPassword123") {
		t.Error("password should be redacted")
	}
	if strings.Contains(redacted, "ghp_1234567890") {
		t.Error("GitHub token should be redacted")
	}

	// Verify [REDACTED] marker is present
	if !strings.Contains(redacted, "[REDACTED]") {
		t.Error("redacted output should contain [REDACTED] marker")
	}

	// Verify file path is preserved
	if !strings.Contains(redacted, "config.json") {
		t.Error("file path should be preserved")
	}

	// Verify diff structure is preserved
	if !strings.Contains(redacted, "---") || !strings.Contains(redacted, "+++") {
		t.Error("diff structure should be preserved")
	}
}

// TestDigestPreservesGitCommitHashes tests that commit hashes are NOT redacted.
func TestDigestPreservesGitCommitHashes(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create and commit a file
	file := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file, []byte("initial\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "test.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Modify the file
	if err := os.WriteFile(file, []byte("initial\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Generate digest with commit hash reference
	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Apply redaction
	redacted := redact.RedactDigest(content)

	// The critical test: long hex strings that look like commit hashes should NOT be redacted
	// This is a 40-char hex string similar to a git commit hash
	testCommitHash := "abc123def456789012345678901234567890abcd"
	if strings.Contains(redacted, "[REDACTED_HASH]") {
		t.Error("Generic long hex strings (like commit hashes) should NOT be redacted")
	}
	// Verify that such a hash would NOT be redacted if it appeared
	hashInput := "commit: " + testCommitHash
	hashResult := redact.RedactDigest(hashInput)
	if strings.Contains(hashResult, "[REDACTED") {
		t.Errorf("Git commit hash should not be redacted, got: %s", hashResult)
	}
}

// TestDigestWriteAppliesRedaction tests that Write() function applies redaction.
func TestDigestWriteAppliesRedaction(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create and commit a clean baseline
	cleanFile := filepath.Join(tmpDir, "app.properties")
	cleanContent := "app.name=test\n"
	if err := os.WriteFile(cleanFile, []byte(cleanContent), 0644); err != nil {
		t.Fatalf("failed to write clean file: %v", err)
	}
	runGit(t, tmpDir, "add", "app.properties")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Modify file with fake secrets
	secretContent := "app.name=test\napi.key=sk-1234567890abcdefghijklmnopqrstuvwxyz\n"
	if err := os.WriteFile(cleanFile, []byte(secretContent), 0644); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	// Write digest to file (this applies redaction)
	outputPath := filepath.Join(tmpDir, "digest.md")
	err := Write(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
		Output:   outputPath,
	})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read back the written digest
	writtenContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read written digest: %v", err)
	}

	writtenStr := string(writtenContent)

	// Verify secrets are redacted
	if strings.Contains(writtenStr, "sk-1234567890") {
		t.Error("OpenAI API key should be redacted in written output")
	}
	if !strings.Contains(writtenStr, "[REDACTED]") {
		t.Error("redacted output should contain [REDACTED] marker")
	}

	// Verify file path is preserved
	if !strings.Contains(writtenStr, "app.properties") {
		t.Error("file path should be preserved in written output")
	}
}
