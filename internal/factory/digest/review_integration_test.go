// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegration_NewSectionsBeforeFileEvidence(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	trackedFile := filepath.Join(tmpDir, "tracked.txt")
	if err := os.WriteFile(trackedFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "tracked.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	if err := os.WriteFile(trackedFile, []byte("initial content\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(content, "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 2") {
		t.Error("expected contract version 2 in output")
	}

	manifestIdx := strings.Index(content, "## CHANGESET_MANIFEST")
	statsIdx := strings.Index(content, "## CHANGESET_STATS")
	reviewMapIdx := strings.Index(content, "## REVIEW_MAP")
	riskIdx := strings.Index(content, "## RISK_SIGNALS")
	changedIdx := strings.Index(content, "## Changed files")

	if manifestIdx == -1 {
		t.Error("expected ## CHANGESET_MANIFEST section")
	}
	if statsIdx == -1 {
		t.Error("expected ## CHANGESET_STATS section")
	}
	if reviewMapIdx == -1 {
		t.Error("expected ## REVIEW_MAP section")
	}
	if riskIdx == -1 {
		t.Error("expected ## RISK_SIGNALS section")
	}

	if manifestIdx > changedIdx && manifestIdx != -1 {
		t.Error("## CHANGESET_MANIFEST should come before ## Changed files")
	}
	if statsIdx > changedIdx && statsIdx != -1 {
		t.Error("## CHANGESET_STATS should come before ## Changed files")
	}
	if reviewMapIdx > changedIdx && reviewMapIdx != -1 {
		t.Error("## REVIEW_MAP should come before ## Changed files")
	}
	if riskIdx > changedIdx && riskIdx != -1 {
		t.Error("## RISK_SIGNALS should come before ## Changed files")
	}

	diffsIdx := strings.Index(content, "## Diffs")
	if diffsIdx == -1 {
		t.Error("expected ## Diffs section")
	}
	if diffsIdx < changedIdx {
		t.Error("## Diffs should come after ## Changed files")
	}
}

func TestIntegration_RangeModeWithNewSections(t *testing.T) {
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

	if !strings.Contains(content, "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 2") {
		t.Error("expected contract version 2 in output")
	}

	if !strings.Contains(content, "## CHANGESET_MANIFEST") {
		t.Error("expected ## CHANGESET_MANIFEST section")
	}
	if !strings.Contains(content, "## CHANGESET_STATS") {
		t.Error("expected ## CHANGESET_STATS section")
	}
	if !strings.Contains(content, "## REVIEW_MAP") {
		t.Error("expected ## REVIEW_MAP section")
	}
	if !strings.Contains(content, "## RISK_SIGNALS") {
		t.Error("expected ## RISK_SIGNALS section")
	}

	if !strings.Contains(content, "A  file2.txt") {
		t.Error("expected added file2.txt in manifest")
	}
}
