// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/version"
)

// Integration test: verify digest output has contract header

func TestDigestOutput_HasContractHeader(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	file := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "test.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Modify the file to create dirty state
	if err := os.WriteFile(file, []byte("content\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify contract header is at the top
	if !strings.HasPrefix(content, "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 1") {
		t.Error("digest should start with contract header")
	}

	// Verify body content is preserved
	if !strings.Contains(content, "# Targeted digest") {
		t.Error("digest body should be preserved after header")
	}
}

func TestDigestOutput_HasContractHeader_Staged(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	file := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "test.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Make and stage a change
	if err := os.WriteFile(file, []byte("content\nstaged\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}
	runGit(t, tmpDir, "add", "test.txt")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeStaged,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.HasPrefix(content, "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 1") {
		t.Error("staged digest should start with contract header")
	}
	if !strings.Contains(content, "DIGEST_MODE: staged") {
		t.Error("staged digest should have correct mode")
	}
}

func TestDigestOutput_HasContractHeader_Range(t *testing.T) {
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

	if !strings.HasPrefix(content, "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 1") {
		t.Error("range digest should start with contract header")
	}
	if !strings.Contains(content, "DIGEST_MODE: range") {
		t.Error("range digest should have correct mode")
	}
}

func TestDigestOutput_AutoModeReportsEffectiveMode(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	file := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "test.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Make a change to trigger dirty mode
	if err := os.WriteFile(file, []byte("content\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeAuto,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Auto mode with dirty tree should report dirty mode
	if !strings.Contains(content, "DIGEST_MODE: dirty") {
		t.Error("auto mode should report effective mode (dirty) in contract header")
	}
}

func TestDigestOutput_VersionMetadataPopulated(t *testing.T) {
	// Save and restore version state
	oldVersion := version.Version
	oldCommit := version.Commit
	oldBuildTime := version.BuildTime
	t.Cleanup(func() {
		version.Version = oldVersion
		version.Commit = oldCommit
		version.BuildTime = oldBuildTime
	})

	// Inject test version metadata
	version.Version = "1.2.3"
	version.Commit = "test123"
	version.BuildTime = "2026-01-01T00:00:00Z"

	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	file := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "test.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	if err := os.WriteFile(file, []byte("content\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "LEAMAS_VERSION: 1.2.3") {
		t.Errorf("digest should contain injected version, got: %s", content)
	}
	if !strings.Contains(content, "LEAMAS_COMMIT: test123") {
		t.Errorf("digest should contain injected commit, got: %s", content)
	}
}
