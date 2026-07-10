// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPublicSurfaceDelta_IntraPackageFileSplit verifies that moving a symbol between
// files in the same package does not produce false removals.
func TestPublicSurfaceDelta_IntraPackageFileSplit(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial commit")

	pkgDir := filepath.Join(tmpDir, "internal", "factory", "digest")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	digestGo := filepath.Join(pkgDir, "digest.go")
	digestContent := `// Package digest provides digest generation.
package digest

// ChangedFile represents a changed file.
type ChangedFile struct {
	Path string
}

// GetDirtyFiles returns dirty files.
func GetDirtyFiles(repoRoot string) ([]ChangedFile, error) {
	return nil, nil
}

// GetStagedFiles returns staged files.
func GetStagedFiles(repoRoot string) ([]ChangedFile, error) {
	return nil, nil
}

// Generate generates a digest.
func Generate(opts Options) (string, error) {
	return "", nil
}

// Write writes a digest.
func Write(digest string) error {
	return nil
}

// Options holds digest options.
type Options struct{}

// RenderDigest renders a digest.
func RenderDigest() string {
	return ""
}
`
	if err := os.WriteFile(digestGo, []byte(digestContent), 0644); err != nil {
		t.Fatalf("failed to write digest.go: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add digest package")

	fileOpsGo := filepath.Join(pkgDir, "file_operations.go")
	fileOpsContent := `// Package digest provides digest generation.
package digest

// ChangedFile represents a changed file.
type ChangedFile struct {
	Path string
}

// GetDirtyFiles returns dirty files.
func GetDirtyFiles(repoRoot string) ([]ChangedFile, error) {
	return nil, nil
}

// GetStagedFiles returns staged files.
func GetStagedFiles(repoRoot string) ([]ChangedFile, error) {
	return nil, nil
}
`
	digestContentAfter := `// Package digest provides digest generation.
package digest

// Generate generates a digest.
func Generate(opts Options) (string, error) {
	return "", nil
}

// Write writes a digest.
func Write(digest string) error {
	return nil
}

// Options holds digest options.
type Options struct{}

// RenderDigest renders a digest.
func RenderDigest() string {
	return ""
}
`
	if err := os.WriteFile(digestGo, []byte(digestContentAfter), 0644); err != nil {
		t.Fatalf("failed to update digest.go: %v", err)
	}
	if err := os.WriteFile(fileOpsGo, []byte(fileOpsContent), 0644); err != nil {
		t.Fatalf("failed to write file_operations.go: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "split digest.go into file_operations.go")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	falseRemovals := []string{
		"internal.factory.digest.ChangedFile(type)",
		"internal.factory.digest.GetDirtyFiles(func)",
		"internal.factory.digest.GetStagedFiles(func)",
		"internal.factory.digest.Generate(func)",
		"internal.factory.digest.Write(func)",
		"internal.factory.digest.Options(type)",
		"internal.factory.digest.RenderDigest(func)",
	}
	for _, fr := range falseRemovals {
		if strings.Contains(output, "- "+fr) {
			t.Errorf("False removal: %s", fr)
		}
	}
	if !strings.Contains(output, "symbols_removed=0") {
		t.Errorf("Expected symbols_removed=0, got: %s", extractField(output, "symbols_removed"))
	}
}

// TestPublicSurfaceDelta_ExportedFieldSurvivesTypeRelocation verifies that exported
// fields survive when a type is moved to a different file.
func TestPublicSurfaceDelta_ExportedFieldSurvivesTypeRelocation(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial commit")

	pkgDir := filepath.Join(tmpDir, "pkg", "example")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	typesGo := filepath.Join(pkgDir, "types.go")
	typesContent := `package example

// MyStruct is a struct with exported fields.
type MyStruct struct {
	Name string
	Value int
}
`
	if err := os.WriteFile(typesGo, []byte(typesContent), 0644); err != nil {
		t.Fatalf("failed to write types.go: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add types.go")

	newFile := filepath.Join(pkgDir, "relocated.go")
	relocatedContent := `package example

// MyStruct is a struct with exported fields.
type MyStruct struct {
	Name string
	Value int
}
`
	emptyContent := `package example

// Empty file after relocating MyStruct
`
	if err := os.WriteFile(typesGo, []byte(emptyContent), 0644); err != nil {
		t.Fatalf("failed to update types.go: %v", err)
	}
	if err := os.WriteFile(newFile, []byte(relocatedContent), 0644); err != nil {
		t.Fatalf("failed to write relocated.go: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "relocate MyStruct to new file")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "symbols_removed=0") {
		t.Errorf("Expected symbols_removed=0, got: %s", extractField(output, "symbols_removed"))
	}
	if !strings.Contains(output, "symbols_added=0") {
		t.Errorf("Expected symbols_added=0, got: %s", extractField(output, "symbols_added"))
	}
	if !strings.Contains(output, "symbols_modified=0") {
		t.Errorf("Expected symbols_modified=0, got: %s", extractField(output, "symbols_modified"))
	}
}
