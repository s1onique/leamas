// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDependencyDelta_DirtyMode tests dirty mode comparison.
func TestDependencyDelta_DirtyMode(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	goMod1 := `module github.com/example/mymodule

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod1), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.sum"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write go.sum: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial module")

	goMod2 := `module github.com/example/mymodule

go 1.22
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod2), 0644); err != nil {
		t.Fatalf("failed to modify go.mod: %v", err)
	}

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "## DEPENDENCY_DELTA") {
		t.Error("digest output missing DEPENDENCY_DELTA section")
	}
	if !strings.Contains(output, "source_status=present") {
		t.Error("expected source_status=present in dirty mode")
	}
	if !strings.Contains(output, "go_version_changed=true") {
		t.Error("expected go_version_changed=true for dirty mode")
	}
}

// TestDependencyDelta_StagedMode tests staged mode comparison.
func TestDependencyDelta_StagedMode(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	goMod1 := `module github.com/example/mymodule

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod1), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.sum"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write go.sum: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial module")

	goMod2 := `module github.com/example/mymodule

go 1.22
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod2), 0644); err != nil {
		t.Fatalf("failed to modify go.mod: %v", err)
	}
	runGit(t, tmpDir, "add", "go.mod")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeStaged,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "## DEPENDENCY_DELTA") {
		t.Error("digest output missing DEPENDENCY_DELTA section")
	}
	if !strings.Contains(output, "go_version_changed=true") {
		t.Error("expected go_version_changed=true for staged mode")
	}
}
