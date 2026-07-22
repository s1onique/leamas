// Package dupcode provides tests for baseline artifact validation.
package dupcode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

// TestValidateBaselineArtifact_ValidTrackedArtifact tests validation of a valid, tracked baseline.
func TestValidateBaselineArtifact_ValidTrackedArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	setupGitRepo(t, ctx, tmpDir)

	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	writeBaseline(t, baselinePath, Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: AlgorithmVersion,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds:       BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings: []BaselineFinding{
			{
				Fingerprint: "002ec5ff009cad28f7e278c01749ac4268d1ed3a1325a86df39db87d7c909edb",
				TokenCount:  400,
				LineCount:   75,
				Occurrences: []BaselineOccurrence{
					{Path: "foo.go", StartLine: 10, EndLine: 50},
				},
			},
		},
	})

	gitAdd(t, ctx, tmpDir, ".factory/dupcode-baseline.json")

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	validation, err := ValidateBaselineArtifact(tmpDir, policy)
	if err != nil {
		t.Fatalf("ValidateBaselineArtifact failed: %v", err)
	}

	if !validation.UsableForDrift {
		t.Error("expected baseline to be usable for drift")
	}
	if validation.Baseline.SchemaVersion != 1 {
		t.Errorf("baseline SchemaVersion = %d, want 1", validation.Baseline.SchemaVersion)
	}
	if len(validation.Findings) != 0 {
		t.Errorf("unexpected findings: %#v", validation.Findings)
	}
}

// TestValidateBaselineArtifact_Missing tests that missing baseline returns appropriate finding.
func TestValidateBaselineArtifact_Missing(t *testing.T) {
	tmpDir := t.TempDir()

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	validation, err := ValidateBaselineArtifact(tmpDir, policy)
	if err != nil {
		t.Fatalf("ValidateBaselineArtifact returned unexpected error: %v", err)
	}

	if validation.UsableForDrift {
		t.Error("expected baseline to NOT be usable for drift (missing)")
	}

	found := false
	for _, f := range validation.Findings {
		if f.Kind == "missing_dupcode_baseline" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected missing_dupcode_baseline finding")
	}
}

// TestValidateBaselineArtifact_Untracked tests that untracked baseline returns appropriate finding.
func TestValidateBaselineArtifact_Untracked(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	setupGitRepo(t, ctx, tmpDir)

	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	writeBaseline(t, baselinePath, Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: AlgorithmVersion,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds:       BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings:         []BaselineFinding{},
	})

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	validation, err := ValidateBaselineArtifact(tmpDir, policy)
	if err != nil {
		t.Fatalf("ValidateBaselineArtifact returned unexpected error: %v", err)
	}

	if validation.UsableForDrift {
		t.Error("expected baseline to NOT be usable for drift (untracked)")
	}

	found := false
	for _, f := range validation.Findings {
		if f.Kind == "untracked_dupcode_baseline" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected untracked_dupcode_baseline finding")
	}
}

// TestValidateBaselineArtifact_Symlink tests that symlink baseline returns appropriate finding.
func TestValidateBaselineArtifact_Symlink(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	setupGitRepo(t, ctx, tmpDir)

	factoryDir := filepath.Join(tmpDir, ".factory")
	if err := os.MkdirAll(factoryDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	targetPath := filepath.Join(tmpDir, "target.json")
	if err := os.WriteFile(targetPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("write target failed: %v", err)
	}

	symlinkPath := filepath.Join(factoryDir, "dupcode-baseline.json")
	if err := os.Symlink(targetPath, symlinkPath); err != nil {
		t.Fatalf("symlink failed: %v", err)
	}

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	validation, err := ValidateBaselineArtifact(tmpDir, policy)
	if err != nil {
		t.Fatalf("ValidateBaselineArtifact returned unexpected error: %v", err)
	}

	if validation.UsableForDrift {
		t.Error("expected baseline to NOT be usable for drift (symlink)")
	}

	found := false
	for _, f := range validation.Findings {
		if f.Kind == "symlink_not_allowed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected symlink_not_allowed finding")
	}
}

// TestValidateBaselineArtifact_NonRegular tests that non-regular file returns appropriate finding.
func TestValidateBaselineArtifact_NonRegular(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	setupGitRepo(t, ctx, tmpDir)

	factoryDir := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	if err := os.MkdirAll(factoryDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	validation, err := ValidateBaselineArtifact(tmpDir, policy)
	if err != nil {
		t.Fatalf("ValidateBaselineArtifact returned unexpected error: %v", err)
	}

	if validation.UsableForDrift {
		t.Error("expected baseline to NOT be usable for drift (non-regular)")
	}

	found := false
	for _, f := range validation.Findings {
		if f.Kind == "invalid_baseline_type" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected invalid_baseline_type finding")
	}
}

// Helper functions

func setupGitRepo(t *testing.T, ctx context.Context, tmpDir string) {
	t.Helper()
	if _, err := runGitCmd(ctx, tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if _, err := runGitCmd(ctx, tmpDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if _, err := runGitCmd(ctx, tmpDir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}
	if _, err := runGitCmd(ctx, tmpDir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
}

func gitAdd(t *testing.T, ctx context.Context, dir, path string) {
	t.Helper()
	if _, err := runGitCmd(ctx, dir, "add", path); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
}

func runGitCmd(ctx context.Context, dir, name string, args ...string) (string, error) {
	result, err := execution.RunGit(ctx, dir, append([]string{name}, args...)...)
	if err != nil {
		return string(result.Stderr), err
	}
	return string(result.Stdout), nil
}

func writeBaseline(t *testing.T, path string, baseline Baseline) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
