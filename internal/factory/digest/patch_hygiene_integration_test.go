// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchHygiene_Integration(t *testing.T) {
	// Create a temporary git repo
	tmpDir := t.TempDir()

	// Initialize git repo
	_, exitCode := RunGitWithExitCode(tmpDir, []string{"init"})
	if exitCode != 0 && exitCode != -1 {
		t.Skip("git not available")
	}

	// Configure git user
	RunGit(tmpDir, []string{"config", "user.email", "test@example.com"})
	RunGit(tmpDir, []string{"config", "user.name", "Test"})

	// Create initial file and commit
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644)
	RunGit(tmpDir, []string{"add", "test.go"})
	RunGit(tmpDir, []string{"commit", "-m", "initial"})

	// Test 1: Dirty mode detects trailing whitespace
	t.Run("DirtyMode_DetectsTrailingWhitespace", func(t *testing.T) {
		os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n   \n"), 0644)

		ph := RunPatchHygieneDirty(tmpDir)
		if ph.GitDiffCheck != PatchHygieneFail {
			t.Errorf("expected fail, got %s", ph.GitDiffCheck)
		}
	})

	// Test 2: Staged mode detects trailing whitespace
	t.Run("StagedMode_DetectsTrailingWhitespace", func(t *testing.T) {
		// Write with trailing whitespace and stage it
		os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n   \n"), 0644)
		RunGit(tmpDir, []string{"add", "test.go"})

		ph := RunPatchHygiene(tmpDir, "--cached")
		if ph.GitDiffCheck != PatchHygieneFail {
			t.Errorf("expected fail, got %s", ph.GitDiffCheck)
		}
	})

	// Test 3: Range mode detects trailing whitespace
	t.Run("RangeMode_DetectsTrailingWhitespace", func(t *testing.T) {
		RunGit(tmpDir, []string{"commit", "-m", "no ws"})

		os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n   \n"), 0644)
		RunGit(tmpDir, []string{"add", "test.go"})
		RunGit(tmpDir, []string{"commit", "-m", "with ws"})

		ph := RunPatchHygiene(tmpDir, "HEAD~1..HEAD")
		if ph.GitDiffCheck != PatchHygieneFail {
			t.Errorf("expected fail, got %s", ph.GitDiffCheck)
		}
	})

	// Test 4: Clean patch renders pass
	t.Run("CleanPatch_RendersPass", func(t *testing.T) {
		RunGit(tmpDir, []string{"commit", "--allow-empty", "-m", "clean commit"})

		ph := RunPatchHygiene(tmpDir, "HEAD~1..HEAD")
		if ph.GitDiffCheck != PatchHygienePass {
			t.Errorf("expected pass, got %s", ph.GitDiffCheck)
		}
	})
}

func TestPatchHygiene_SectionPosition(t *testing.T) {
	// Test that PATCH_HYGIENE appears after RISK_SIGNALS and before Changed files
	tmpDir := t.TempDir()

	_, exitCode := RunGitWithExitCode(tmpDir, []string{"init"})
	if exitCode != 0 && exitCode != -1 {
		t.Skip("git not available")
	}

	RunGit(tmpDir, []string{"config", "user.email", "test@example.com"})
	RunGit(tmpDir, []string{"config", "user.name", "Test"})

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644)
	RunGit(tmpDir, []string{"add", "test.go"})
	RunGit(tmpDir, []string{"commit", "-m", "initial"})

	// Generate digest
	content, err := RenderRangeDigestWithResolved(tmpDir, []RangeFile{}, &ResolvedMode{
		Mode:   ModeRange,
		Range:  "HEAD~1..HEAD",
		Reason: "test",
	})
	if err != nil {
		t.Fatalf("failed to render digest: %v", err)
	}

	// Check section positions
	riskIdx := strings.Index(content, "## RISK_SIGNALS")
	patchIdx := strings.Index(content, "## PATCH_HYGIENE")
	changedIdx := strings.Index(content, "## Changed files")

	if riskIdx < 0 || patchIdx < 0 || changedIdx < 0 {
		t.Fatal("missing required sections")
	}

	if patchIdx <= riskIdx {
		t.Error("PATCH_HYGIENE should come after RISK_SIGNALS")
	}

	if changedIdx <= patchIdx {
		t.Error("## Changed files should come after PATCH_HYGIENE")
	}
}
