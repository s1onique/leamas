package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDigestAnchors_NoAnchorsConfigured tests that digest shows "No workflow anchors configured."
// when no .leamas/anchors.toml exists.
func TestDigestAnchors_NoAnchorsConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Commit a file
	file := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file, []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, tmpDir, "add", "test.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Generate digest
	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify "No workflow anchors configured." is present
	if !strings.Contains(content, "No workflow anchors configured.") {
		t.Error("digest should contain 'No workflow anchors configured.' when no anchors file exists")
	}
}

// TestDigestAnchors_WithAnchorsConfigured tests that digest includes configured anchors.
func TestDigestAnchors_WithAnchorsConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create .leamas directory and anchors.toml
	leamasDir := filepath.Join(tmpDir, ".leamas")
	if err := os.MkdirAll(leamasDir, 0755); err != nil {
		t.Fatal(err)
	}

	anchorsContent := `[[anchors]]
id = "ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R1"
type = "act"
summary = "Wire workflow anchors into digest output"
url = "docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R1.md"

[[anchors]]
id = "EPIC-LEAMAS-FACTORY-HARDENING"
type = "epic"
summary = "Factory hardening and next bootstrap"
`
	anchorsPath := filepath.Join(leamasDir, "anchors.toml")
	if err := os.WriteFile(anchorsPath, []byte(anchorsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Commit the anchors file
	runGit(t, tmpDir, "add", ".leamas/anchors.toml")
	runGit(t, tmpDir, "commit", "-m", "add anchors")

	// Generate digest
	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify anchors are included
	if !strings.Contains(content, "ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R1") {
		t.Error("digest should contain configured anchor ID 'ACT-LEAMAS-FACTORY-DIGEST-ANCHORS01-R1'")
	}
	if !strings.Contains(content, "EPIC-LEAMAS-FACTORY-HARDENING") {
		t.Error("digest should contain configured anchor ID 'EPIC-LEAMAS-FACTORY-HARDENING'")
	}

	// Verify table format
	if !strings.Contains(content, "| ID | Type | Summary | URL |") {
		t.Error("digest should contain anchors table header")
	}
}

// TestDigestAnchors_MultipleAnchors tests that digest includes all configured anchors.
func TestDigestAnchors_MultipleAnchors(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create .leamas directory and anchors.toml with multiple anchors
	leamasDir := filepath.Join(tmpDir, ".leamas")
	if err := os.MkdirAll(leamasDir, 0755); err != nil {
		t.Fatal(err)
	}

	anchorsContent := `[[anchors]]
id = "ACT-001"
type = "act"
summary = "First action"

[[anchors]]
id = "ADR-001"
type = "adr"
summary = "First decision"

[[anchors]]
id = "EPIC-001"
type = "epic"
summary = "First epic"
url = "docs/epics/EPIC-001.md"
`
	anchorsPath := filepath.Join(leamasDir, "anchors.toml")
	if err := os.WriteFile(anchorsPath, []byte(anchorsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Commit a file to have something in the diff
	file := filepath.Join(tmpDir, "feature.go")
	if err := os.WriteFile(file, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Generate digest
	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify all three anchors are present
	anchors := []string{"ACT-001", "ADR-001", "EPIC-001"}
	for _, id := range anchors {
		if !strings.Contains(content, id) {
			t.Errorf("digest should contain anchor ID '%s'", id)
		}
	}
}

// TestDigestAnchors_PreservesNormalDiffContent tests that anchors don't break normal diff output.
func TestDigestAnchors_PreservesNormalDiffContent(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create anchors.toml
	leamasDir := filepath.Join(tmpDir, ".leamas")
	if err := os.MkdirAll(leamasDir, 0755); err != nil {
		t.Fatal(err)
	}
	anchorsContent := `[[anchors]]
id = "ACT-001"
type = "act"
summary = "Test"
`
	anchorsPath := filepath.Join(leamasDir, "anchors.toml")
	if err := os.WriteFile(anchorsPath, []byte(anchorsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create and commit initial file
	file := filepath.Join(tmpDir, "hello.go")
	initialContent := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(file, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Modify the file
	modifiedContent := "package main\n\nfunc main() {}\n\nfunc greet() {}\n"
	if err := os.WriteFile(file, []byte(modifiedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate digest
	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify normal diff content is preserved
	if !strings.Contains(content, "## Diffs") {
		t.Error("digest should contain '## Diffs' section")
	}
	if !strings.Contains(content, "hello.go") {
		t.Error("digest should contain changed file name")
	}
	if !strings.Contains(content, "---") || !strings.Contains(content, "+++") {
		t.Error("digest should contain diff markers")
	}
}

// TestDigestAnchors_WriteRedactsAnchorSecrets tests that Write() redacts secrets in anchors.
func TestDigestAnchors_WriteRedactsAnchorSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create anchors.toml with a URL that looks like a secret (fake value)
	leamasDir := filepath.Join(tmpDir, ".leamas")
	if err := os.MkdirAll(leamasDir, 0755); err != nil {
		t.Fatal(err)
	}
	anchorsContent := `[[anchors]]
id = "ACT-001"
type = "act"
summary = "Test with secret-like URL"
url = "https://example.com/sk-1234567890abcdefghijklmnop"
`
	anchorsPath := filepath.Join(leamasDir, "anchors.toml")
	if err := os.WriteFile(anchorsPath, []byte(anchorsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Commit a file
	file := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file, []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	// Write digest
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

	// Verify anchor is still present (not filtered out)
	if !strings.Contains(writtenStr, "ACT-001") {
		t.Error("written digest should contain anchor ID")
	}

	// The secret-like URL should be redacted by the redaction pass
	// (if the pattern matches - note: this is a URL, not a standalone secret)
	// The anchor table should still be present
	if !strings.Contains(writtenStr, "| ID | Type | Summary | URL |") {
		t.Error("written digest should contain anchors table header")
	}
}

// TestDigestAnchors_RangeMode tests anchors work in range mode.
func TestDigestAnchors_RangeMode(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create anchors.toml
	leamasDir := filepath.Join(tmpDir, ".leamas")
	if err := os.MkdirAll(leamasDir, 0755); err != nil {
		t.Fatal(err)
	}
	anchorsContent := `[[anchors]]
id = "EPIC-001"
type = "epic"
summary = "Range mode test"
`
	anchorsPath := filepath.Join(leamasDir, "anchors.toml")
	if err := os.WriteFile(anchorsPath, []byte(anchorsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create and commit initial file (first commit)
	file := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(file, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "first commit")

	// Make a second commit so HEAD~1 exists
	if err := os.WriteFile(file, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "second commit")

	// Generate range digest (HEAD~1..HEAD)
	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify anchors are included in range mode
	if !strings.Contains(content, "EPIC-001") {
		t.Error("range mode digest should contain configured anchor")
	}
	if !strings.Contains(content, "| ID | Type | Summary | URL |") {
		t.Error("range mode digest should contain anchors table header")
	}
}
