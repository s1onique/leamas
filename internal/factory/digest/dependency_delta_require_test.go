// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDependencyDelta_RangeMode tests DEPENDENCY_DELTA in range mode.
func TestDependencyDelta_RangeMode(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	goMod1 := `module github.com/example/mymodule

go 1.21

require foo v1.0.0
`
	goSum1 := `foo v1.0.0 h1
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod1), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.sum"), []byte(goSum1), 0644); err != nil {
		t.Fatalf("failed to write go.sum: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial module")

	goMod2 := `module github.com/example/mymodule

go 1.21

require (
	foo v1.0.0
	bar v1.2.0
)
`
	goSum2 := `foo v1.0.0 h1
bar v1.2.0 h2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod2), 0644); err != nil {
		t.Fatalf("failed to update go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.sum"), []byte(goSum2), 0644); err != nil {
		t.Fatalf("failed to update go.sum: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add bar dependency")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "## DEPENDENCY_DELTA") {
		t.Error("digest output missing DEPENDENCY_DELTA section")
	}
	if !strings.Contains(output, "source_status=present") {
		t.Error("expected source_status=present")
	}
	if !strings.Contains(output, "go_mod_changed=true") {
		t.Error("expected go_mod_changed=true")
	}
	if !strings.Contains(output, "requires_added=1") {
		t.Errorf("expected requires_added=1, got: %s", extractField(output, "requires_added"))
	}
	if !strings.Contains(output, "bar v1.2.0") {
		t.Error("expected bar v1.2.0 in requires_added list")
	}
}

// TestDependencyDelta_NoGoFiles tests empty delta when no go files changed.
func TestDependencyDelta_NoGoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial commit")

	readmeFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Readme"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add readme")

	if err := os.WriteFile(readmeFile, []byte("# Readme\n\nUpdated"), 0644); err != nil {
		t.Fatalf("failed to update README: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "update readme")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "## DEPENDENCY_DELTA") {
		t.Error("digest output missing DEPENDENCY_DELTA section")
	}
	if !strings.Contains(output, "source_status=absent") {
		t.Error("expected source_status=absent when no go files changed")
	}
	if !strings.Contains(output, "go_mod_changed=false") {
		t.Error("expected go_mod_changed=false")
	}
}

// TestDependencyDelta_RequiresModified tests require version modification detection.
func TestDependencyDelta_RequiresModified(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	goMod1 := `module github.com/example/mymodule

go 1.21

require foo v1.0.0
`
	goSum1 := `foo v1.0.0 h1
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod1), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.sum"), []byte(goSum1), 0644); err != nil {
		t.Fatalf("failed to write go.sum: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial module")

	goMod2 := `module github.com/example/mymodule

go 1.21

require foo v1.1.0
`
	goSum2 := `foo v1.1.0 h2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod2), 0644); err != nil {
		t.Fatalf("failed to update go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.sum"), []byte(goSum2), 0644); err != nil {
		t.Fatalf("failed to update go.sum: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "upgrade foo")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(output, "requires_modified=1") {
		t.Errorf("expected requires_modified=1, got: %s", extractField(output, "requires_modified"))
	}
	if !strings.Contains(output, "foo v1.0.0 -> v1.1.0") {
		t.Error("expected foo v1.0.0 -> v1.1.0 in requires_modified")
	}
}

// TestDependencyDelta_RequiresRemoved tests require removal detection.
func TestDependencyDelta_RequiresRemoved(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	goMod1 := `module github.com/example/mymodule

go 1.21

require foo v1.0.0
require bar v1.2.0
`
	goSum1 := `foo v1.0.0 h1
bar v1.2.0 h2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod1), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.sum"), []byte(goSum1), 0644); err != nil {
		t.Fatalf("failed to write go.sum: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial module")

	goMod2 := `module github.com/example/mymodule

go 1.21

require foo v1.0.0
`
	goSum2 := `foo v1.0.0 h1
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod2), 0644); err != nil {
		t.Fatalf("failed to update go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.sum"), []byte(goSum2), 0644); err != nil {
		t.Fatalf("failed to update go.sum: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "remove bar dependency")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(output, "requires_removed=1") {
		t.Errorf("expected requires_removed=1, got: %s", extractField(output, "requires_removed"))
	}
	if !strings.Contains(output, "bar v1.2.0") {
		t.Error("expected bar v1.2.0 in requires_removed list")
	}
}
