// Package dupcode provides tests for baseline artifact content validation.
package dupcode

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestValidateBaselineArtifact_MalformedJSON tests that malformed JSON returns appropriate finding.
func TestValidateBaselineArtifact_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	setupGitRepo(t, ctx, tmpDir)

	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	if err := os.MkdirAll(filepath.Dir(baselinePath), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(baselinePath, []byte("not valid json{"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	gitAdd(t, ctx, tmpDir, ".factory/dupcode-baseline.json")

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	validation, err := ValidateBaselineArtifact(tmpDir, policy)
	if err != nil {
		t.Fatalf("ValidateBaselineArtifact returned unexpected error: %v", err)
	}

	if validation.UsableForDrift {
		t.Error("expected baseline to NOT be usable for drift (malformed)")
	}

	found := false
	for _, f := range validation.Findings {
		if f.Kind == "invalid_dupcode_baseline" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected invalid_dupcode_baseline finding for malformed JSON")
	}
}

// TestValidateBaselineArtifact_SchemaMismatch tests that schema mismatch returns appropriate finding.
func TestValidateBaselineArtifact_SchemaMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	setupGitRepo(t, ctx, tmpDir)

	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	writeBaseline(t, baselinePath, Baseline{
		SchemaVersion:    99,
		AlgorithmVersion: AlgorithmVersion,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds:       BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings:         []BaselineFinding{},
	})

	gitAdd(t, ctx, tmpDir, ".factory/dupcode-baseline.json")

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	validation, err := ValidateBaselineArtifact(tmpDir, policy)
	if err != nil {
		t.Fatalf("ValidateBaselineArtifact returned unexpected error: %v", err)
	}

	if validation.UsableForDrift {
		t.Error("expected baseline to NOT be usable for drift (schema mismatch)")
	}

	found := false
	for _, f := range validation.Findings {
		if f.Kind == "invalid_dupcode_baseline" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected invalid_dupcode_baseline finding for schema mismatch")
	}
}

// TestValidateBaselineArtifact_AlgorithmMismatch tests that algorithm version mismatch returns appropriate finding.
func TestValidateBaselineArtifact_AlgorithmMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	setupGitRepo(t, ctx, tmpDir)

	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	writeBaseline(t, baselinePath, Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: 99,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds:       BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings:         []BaselineFinding{},
	})

	gitAdd(t, ctx, tmpDir, ".factory/dupcode-baseline.json")

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	validation, err := ValidateBaselineArtifact(tmpDir, policy)
	if err != nil {
		t.Fatalf("ValidateBaselineArtifact returned unexpected error: %v", err)
	}

	if validation.UsableForDrift {
		t.Error("expected baseline to NOT be usable for drift (algorithm mismatch)")
	}

	found := false
	for _, f := range validation.Findings {
		if f.Kind == "invalid_dupcode_baseline" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected invalid_dupcode_baseline finding for algorithm mismatch")
	}
}

// TestValidateBaselineArtifact_ThresholdMismatch tests that threshold mismatch is non-terminal.
func TestValidateBaselineArtifact_ThresholdMismatch(t *testing.T) {
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

	gitAdd(t, ctx, tmpDir, ".factory/dupcode-baseline.json")

	policy := BaselinePolicy{
		Path:      ".factory/dupcode-baseline.json",
		MinLines:  99,
		MinTokens: 999,
	}

	validation, err := ValidateBaselineArtifact(tmpDir, policy)
	if err != nil {
		t.Fatalf("ValidateBaselineArtifact returned unexpected error: %v", err)
	}

	if !validation.UsableForDrift {
		t.Error("expected baseline to be usable for drift (threshold mismatch is non-terminal)")
	}

	found := false
	for _, f := range validation.Findings {
		if f.Kind == "threshold_policy_mismatch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected threshold_policy_mismatch finding")
	}
}

// TestValidateBaselineArtifact_MissingAlgorithmVersion tests that missing algorithm version is terminal.
func TestValidateBaselineArtifact_MissingAlgorithmVersion(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	setupGitRepo(t, ctx, tmpDir)

	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	writeBaseline(t, baselinePath, Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: 0,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds:       BaselineThresholds{MinLines: 40, MinTokens: 400},
		Findings:         []BaselineFinding{},
	})

	gitAdd(t, ctx, tmpDir, ".factory/dupcode-baseline.json")

	policy := DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"

	validation, err := ValidateBaselineArtifact(tmpDir, policy)
	if err != nil {
		t.Fatalf("ValidateBaselineArtifact returned unexpected error: %v", err)
	}

	if validation.UsableForDrift {
		t.Error("expected baseline to NOT be usable for drift (missing algorithm version)")
	}

	found := false
	for _, f := range validation.Findings {
		if f.Kind == "invalid_dupcode_baseline" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected invalid_dupcode_baseline finding for missing algorithm version")
	}
}
