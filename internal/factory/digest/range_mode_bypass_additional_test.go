// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// range_mode_bypass_additional_test.go contains additional tests for
// ACT-LEAMAS-FACTORY-TARGETED-DIGEST-RANGE-MODE-BYPASS-CORRECTION01.
package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLargeFileSmallDelta verifies that a large file with a small committed
// delta is handled correctly with unrelated worktree dirt.
// This test includes the amplification bound: rendered evidence must be within
// 10x of the raw git diff size (floor 64KB). This prevents a regression to
// the ~2,675x amplification that occurred when the empty-tree fallback was active.
func TestLargeFileSmallDelta(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	largeFile := filepath.Join(dir, "large.csv")
	var largeContent strings.Builder
	largeContent.Grow(1024 * 1024)
	for i := 0; i < 20000; i++ {
		largeContent.WriteString(fmt.Sprintf("%05d,long,string,value,here,with,more,fields\n", i))
	}
	if err := os.WriteFile(largeFile, []byte(largeContent.String()), 0644); err != nil {
		t.Fatalf("write large.csv: %v", err)
	}
	runGit(t, dir, "add", "large.csv")
	runGit(t, dir, "commit", "-m", "add large file")

	content := largeContent.String()
	content = strings.Replace(content, "00000,", "00000M,", 1)
	if err := os.WriteFile(largeFile, []byte(content), 0644); err != nil {
		t.Fatalf("modify large.csv: %v", err)
	}
	runGit(t, dir, "add", "large.csv")
	runGit(t, dir, "commit", "-m", "tiny change to large file")

	if err := os.MkdirAll(filepath.Join(dir, ".factory"), 0755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".factory", "gate.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write gate.json: %v", err)
	}

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(out, "=== large.csv ===") {
		t.Errorf("large.csv should appear in digest")
	}
	if strings.Contains(out, "new file mode") {
		t.Errorf("BUG: large.csv rendered as new file against empty tree")
	}
	if strings.Contains(out, "--- /dev/null") {
		t.Errorf("BUG: diff shows /dev/null as old state")
	}

	// Amplification bound: extract the large.csv diff section and compare to raw git diff
	largeCSVSection := extractDiffSection(out, "large.csv")
	rawDiff := runGit(t, dir, "diff", "--unified=3", "HEAD~1..HEAD", "--", "large.csv")

	// The bound: max(10x raw, 64KB floor)
	// Prevents regression to ~2,675x amplification seen in the original incident
	limit := max(len(rawDiff)*10, 64*1024)
	if len(largeCSVSection) > limit {
		t.Fatalf(
			"large.csv evidence inflated: rendered=%d raw=%d limit=%d",
			len(largeCSVSection),
			len(rawDiff),
			limit,
		)
	}
}

// TestAutoModeWithExplicitRange_ResolverPath exercises ModeAuto with explicit Range.
func TestAutoModeWithExplicitRange_ResolverPath(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("v1\n"), 0644); err != nil {
		t.Fatalf("write file.txt: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "commit 1")

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("v2\n"), 0644); err != nil {
		t.Fatalf("modify file.txt: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "commit 2")

	if err := os.MkdirAll(filepath.Join(dir, ".factory"), 0755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".factory", "gate.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write gate.json: %v", err)
	}

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeAuto,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate with ModeAuto + Range failed: %v", err)
	}

	if !strings.Contains(out, "Mode: range") {
		t.Errorf("expected Mode: range with explicit Range")
	}
	if !strings.Contains(out, "RESOLUTION_SOURCE: explicit_cli") {
		t.Errorf("expected RESOLUTION_SOURCE: explicit_cli")
	}
}

// TestExplicitRangeResolver_IsCleanField verifies IsClean field.
func TestExplicitRangeResolver_IsCleanField(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("v1\n"), 0644); err != nil {
		t.Fatalf("write file.txt: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "commit 1")

	if err := os.MkdirAll(filepath.Join(dir, ".factory"), 0755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".factory", "gate.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write gate.json: %v", err)
	}
	resolved, err := ResolveAutoModeExplicitRange(dir, "", "HEAD~1..HEAD")
	if err != nil {
		t.Fatalf("ResolveAutoModeExplicitRange failed: %v", err)
	}
	if resolved.Mode != ModeRange {
		t.Errorf("expected ModeRange, got %s", resolved.Mode)
	}
	if resolved.IsClean {
		t.Errorf("IsClean should be false when dirty")
	}

	os.RemoveAll(filepath.Join(dir, ".factory"))

	resolved2, err := ResolveAutoModeExplicitRange(dir, "", "HEAD~1..HEAD")
	if err != nil {
		t.Fatalf("ResolveAutoModeExplicitRange failed (clean): %v", err)
	}
	if resolved2.Mode != ModeRange {
		t.Errorf("expected ModeRange, got %s", resolved2.Mode)
	}
	if !resolved2.IsClean {
		t.Errorf("IsClean should be true when clean")
	}
}
