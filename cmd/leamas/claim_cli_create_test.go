package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// Claim create tests
// ============================================================================

func TestWitnessClaimCreateRequiresRunID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-smoke01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{"--root", tmp, "--id", "claim-test", "--statement", "Test claim"}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaimCreate)

	if code == 0 {
		t.Error("expected non-zero exit when --run-id is missing")
	}
	if !strings.Contains(stderr, "--run-id") && !strings.Contains(stderr, "requires --run-id") {
		t.Errorf("stderr should mention --run-id requirement, got: %s", stderr)
	}
}

func TestWitnessClaimCreateRequiresID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-smoke01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{"--root", tmp, "--run-id", runID, "--statement", "Test claim"}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaimCreate)

	if code == 0 {
		t.Error("expected non-zero exit when --id is missing")
	}
	if !strings.Contains(stderr, "--id") && !strings.Contains(stderr, "requires --id") {
		t.Errorf("stderr should mention --id requirement, got: %s", stderr)
	}
}

func TestWitnessClaimCreateRequiresStatement(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-smoke01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{"--root", tmp, "--run-id", runID, "--id", "claim-test"}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaimCreate)

	if code == 0 {
		t.Error("expected non-zero exit when --statement is missing")
	}
	if !strings.Contains(stderr, "--statement") && !strings.Contains(stderr, "requires --statement") {
		t.Errorf("stderr should mention --statement requirement, got: %s", stderr)
	}
}

func TestWitnessClaimCreateCreatesClaim(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-create01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "claim-gate-passed",
		"--statement", "The gate passed",
	}

	stdout, stderr, code := captureRunBundleOutput(args, runWitnessClaimCreate)

	if code != 0 {
		t.Fatalf("create failed with code %d, stderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr on success, got: %s", stderr)
	}
	if !strings.Contains(stdout, "claim-gate-passed") {
		t.Errorf("stdout should contain claim ID, got: %s", stdout)
	}
	if !strings.Contains(stdout, runID) {
		t.Errorf("stdout should contain run ID, got: %s", stdout)
	}

	claimPath := filepath.Join(tmp, runID, "claims", "claim-gate-passed.json")
	if _, err := os.Stat(claimPath); os.IsNotExist(err) {
		t.Errorf("claim file should exist: %s", claimPath)
	}
}

func TestWitnessClaimCreateJSONOutput(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-json01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "claim-test-json",
		"--statement", "Test claim",
		"--json",
	}

	stdout, stderr, code := captureRunBundleOutput(args, runWitnessClaimCreate)

	if code != 0 {
		t.Fatalf("create with --json failed with code %d, stderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr on success, got: %s", stderr)
	}

	var output struct {
		OK      bool   `json:"ok"`
		RunID   string `json:"run_id"`
		ClaimID string `json:"claim_id"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("stdout should be valid JSON: %v\noutput: %s", err, stdout)
	}
	if !output.OK {
		t.Error("expected ok=true in JSON output")
	}
	if output.ClaimID != "claim-test-json" {
		t.Errorf("claim_id = %q, want %q", output.ClaimID, "claim-test-json")
	}
	if output.Path == "" {
		t.Error("path should not be empty")
	}
}

func TestWitnessClaimCreateRejectsInvalidClaimID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-smoke01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	testCases := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"no prefix", "test-20260101"},
		{"traversal", "claim-../etc"},
		{"path separator", "claim-2026/01/01"},
		{"absolute", "/claim-absolute"},
		{"missing suffix", "claim-"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"--root", tmp, "--run-id", runID, "--id", tc.id, "--statement", "Test"}
			_, stderr, code := captureRunBundleOutput(args, runWitnessClaimCreate)

			if code == 0 {
				t.Errorf("expected non-zero exit for invalid ID %q", tc.id)
			}
			if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "claim ID") {
				t.Errorf("stderr should mention invalid claim ID, got: %s", stderr)
			}
		})
	}
}

func TestWitnessClaimCreateRejectsInvalidRunID(t *testing.T) {
	tmp := t.TempDir()
	args := []string{
		"--root", tmp,
		"--run-id", "bad-run-id",
		"--id", "claim-test",
		"--statement", "Test",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaimCreate)

	if code == 0 {
		t.Error("expected non-zero exit for invalid run ID")
	}
	if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "run ID") {
		t.Errorf("stderr should mention invalid run ID, got: %s", stderr)
	}
}

func TestWitnessClaimCreateRejectsMissingRunBundle(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-does-not-exist"
	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "claim-test",
		"--statement", "Test",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaimCreate)

	if code == 0 {
		t.Error("expected non-zero exit for missing run bundle")
	}
	if !strings.Contains(stderr, "not found") && !strings.Contains(stderr, "run bundle") {
		t.Errorf("stderr should mention run bundle not found, got: %s", stderr)
	}
}
