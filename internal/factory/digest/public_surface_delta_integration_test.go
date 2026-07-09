// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPublicSurfaceDelta_Integration tests PUBLIC_SURFACE_DELTA section in digest output.
func TestPublicSurfaceDelta_Integration(t *testing.T) {
	// Create temp git repo for integration test
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Create initial commit (empty)
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial commit")

	// Create a Go file with exported symbols
	pkgDir := filepath.Join(tmpDir, "pkg", "example")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	goFile := filepath.Join(pkgDir, "example.go")
	goContent := `// Package example provides an example API.
package example

import "context"

// ExportedFunc is an exported function.
func ExportedFunc(ctx context.Context) error {
	return nil
}

// ExportedType is an exported type.
type ExportedType struct {
	// Name is an exported field.
	Name string
}

// Method is a method on ExportedType.
func (e *ExportedType) Method() {}

// UnexportedFunc is not exported.
func unexportedFunc() {}

// unexportedType is not exported.
type unexportedType struct{}
`
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	// Create a CLI command file under cmd/leamas (matches real leamas CLI)
	cmdDir := filepath.Join(tmpDir, "cmd", "leamas")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}

	cmdFile := filepath.Join(cmdDir, "main.go")
	cmdContent := `package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "leamas",
	Short: "Leamas CLI",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("v1.0.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
`
	if err := os.WriteFile(cmdFile, []byte(cmdContent), 0644); err != nil {
		t.Fatalf("failed to write cmd file: %v", err)
	}

	// Stage and commit
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add example package and CLI")

	// Add more exports in a second commit
	newGoContent := goContent + `
// NewExportedFunc is a newly added exported function.
func NewExportedFunc() {}

// AnotherExportedType is another exported type.
type AnotherExportedType struct{}
`
	if err := os.WriteFile(goFile, []byte(newGoContent), 0644); err != nil {
		t.Fatalf("failed to update go file: %v", err)
	}

	// Also add a new command
	newCmdFile := filepath.Join(cmdDir, "newcmd.go")
	newCmdContent := `package main

import "github.com/spf13/cobra"

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "New command",
	Run: func(cmd *cobra.Command, args []string) {},
}

func init() {
	rootCmd.AddCommand(newCmd)
}
`
	if err := os.WriteFile(newCmdFile, []byte(newCmdContent), 0644); err != nil {
		t.Fatalf("failed to write new cmd file: %v", err)
	}

	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add new exports")

	// Generate digest for the range
	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify PUBLIC_SURFACE_DELTA section exists
	if !strings.Contains(output, "## PUBLIC_SURFACE_DELTA") {
		t.Error("digest output missing PUBLIC_SURFACE_DELTA section")
	}

	// Verify packages_changed count
	if !strings.Contains(output, "packages_changed=") {
		t.Error("missing packages_changed field")
	}

	// Verify symbols_added count
	if !strings.Contains(output, "symbols_added=") {
		t.Error("missing symbols_added field")
	}

	// Verify cli_commands_changed count
	if !strings.Contains(output, "cli_commands_changed=") {
		t.Error("missing cli_commands_changed field")
	}

	// Verify symbols_added is > 0 (we added NewExportedFunc and AnotherExportedType)
	if !strings.Contains(output, "symbols_added=2") {
		t.Errorf("expected symbols_added=2, got: %s", extractField(output, "symbols_added"))
	}

	// Verify packages section has content
	if !strings.Contains(output, "pkg.example") {
		t.Error("expected pkg.example in packages list")
	}

	t.Logf("PUBLIC_SURFACE_DELTA section present in digest output")
}

// extractField extracts a field value from digest output.
func extractField(output, fieldName string) string {
	prefix := fieldName + "="
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}

// TestPublicSurfaceDelta_EmptyChanges verifies empty section when no public surface changes.
func TestPublicSurfaceDelta_EmptyChanges(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial commit")

	// Create only doc files
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("failed to create docs dir: %v", err)
	}
	docFile := filepath.Join(docsDir, "readme.md")
	if err := os.WriteFile(docFile, []byte("# Readme"), 0644); err != nil {
		t.Fatalf("failed to write doc file: %v", err)
	}

	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add docs")

	// Add more docs
	if err := os.WriteFile(docFile, []byte("# Readme\n\nUpdated"), 0644); err != nil {
		t.Fatalf("failed to update doc file: %v", err)
	}

	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "update docs")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify PUBLIC_SURFACE_DELTA section exists
	if !strings.Contains(output, "## PUBLIC_SURFACE_DELTA") {
		t.Error("digest output missing PUBLIC_SURFACE_DELTA section")
	}

	// Verify counts are all zero
	if !strings.Contains(output, "packages_changed=0") {
		t.Error("expected packages_changed=0")
	}
	if !strings.Contains(output, "symbols_added=0") {
		t.Error("expected symbols_added=0")
	}
	if !strings.Contains(output, "symbols_removed=0") {
		t.Error("expected symbols_removed=0")
	}
	if !strings.Contains(output, "symbols_modified=0") {
		t.Error("expected symbols_modified=0")
	}
	if !strings.Contains(output, "cli_commands_changed=0") {
		t.Error("expected cli_commands_changed=0")
	}
}
