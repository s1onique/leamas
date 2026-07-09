package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// Evidence create tests
// ============================================================================

func TestWitnessEvidenceCreateRequiresRunID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-ev01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--id", "evidence-test",
		"--kind", "command_output",
		"--role", "primary",
		"--title", "Test evidence",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessEvidenceCreate)

	if code == 0 {
		t.Error("expected non-zero exit when --run-id is missing")
	}
	if !strings.Contains(stderr, "--run-id") && !strings.Contains(stderr, "requires --run-id") {
		t.Errorf("stderr should mention --run-id requirement, got: %s", stderr)
	}
}

func TestWitnessEvidenceCreateRequiresID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-ev02"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--kind", "command_output",
		"--role", "primary",
		"--title", "Test evidence",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessEvidenceCreate)

	if code == 0 {
		t.Error("expected non-zero exit when --id is missing")
	}
	if !strings.Contains(stderr, "--id") && !strings.Contains(stderr, "requires --id") {
		t.Errorf("stderr should mention --id requirement, got: %s", stderr)
	}
}

func TestWitnessEvidenceCreateRequiresKind(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-ev03"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test",
		"--role", "primary",
		"--title", "Test evidence",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessEvidenceCreate)

	if code == 0 {
		t.Error("expected non-zero exit when --kind is missing")
	}
	if !strings.Contains(stderr, "--kind") && !strings.Contains(stderr, "requires --kind") {
		t.Errorf("stderr should mention --kind requirement, got: %s", stderr)
	}
}

func TestWitnessEvidenceCreateRequiresRole(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-ev04"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test",
		"--kind", "command_output",
		"--title", "Test evidence",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessEvidenceCreate)

	if code == 0 {
		t.Error("expected non-zero exit when --role is missing")
	}
	if !strings.Contains(stderr, "--role") && !strings.Contains(stderr, "requires --role") {
		t.Errorf("stderr should mention --role requirement, got: %s", stderr)
	}
}

func TestWitnessEvidenceCreateRequiresTitle(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-ev05"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test",
		"--kind", "command_output",
		"--role", "primary",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessEvidenceCreate)

	if code == 0 {
		t.Error("expected non-zero exit when --title is missing")
	}
	if !strings.Contains(stderr, "--title") && !strings.Contains(stderr, "requires --title") {
		t.Errorf("stderr should mention --title requirement, got: %s", stderr)
	}
}

func TestWitnessEvidenceCreateRejectsBadKind(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-ev06"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test",
		"--kind", "invalid_kind",
		"--role", "primary",
		"--title", "Test evidence",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessEvidenceCreate)

	if code == 0 {
		t.Error("expected non-zero exit for invalid kind")
	}
	if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "kind") {
		t.Errorf("stderr should mention invalid kind, got: %s", stderr)
	}
}

func TestWitnessEvidenceCreateRejectsBadRole(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-ev07"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test",
		"--kind", "command_output",
		"--role", "invalid_role",
		"--title", "Test evidence",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessEvidenceCreate)

	if code == 0 {
		t.Error("expected non-zero exit for invalid role")
	}
	if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "role") {
		t.Errorf("stderr should mention invalid role, got: %s", stderr)
	}
}

func TestWitnessEvidenceCreateRejectsUnsafeRelativePath(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-ev08"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	testCases := []struct {
		name string
		path string
	}{
		{"absolute", "/absolute/path"},
		{"traversal", "../escape"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{
				"--root", tmp,
				"--run-id", runID,
				"--id", "evidence-test",
				"--kind", "command_output",
				"--role", "primary",
				"--title", "Test evidence",
				"--relative-path", tc.path,
			}
			_, stderr, code := captureRunBundleOutput(args, runWitnessEvidenceCreate)

			if code == 0 {
				t.Errorf("expected non-zero exit for invalid relative path %q", tc.path)
			}
			if !strings.Contains(stderr, "relative path") {
				t.Errorf("stderr should mention relative path error, got: %s", stderr)
			}
		})
	}
}

func TestWitnessEvidenceCreateCreatesEvidence(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-ev09"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-make-gate-output",
		"--kind", "command_output",
		"--role", "primary",
		"--title", "Make gate output",
		"--relative-path", "verifier-results/make-gate.txt",
		"--summary", "Exit code 0",
	}

	stdout, stderr, code := captureRunBundleOutput(args, runWitnessEvidenceCreate)

	if code != 0 {
		t.Fatalf("create failed with code %d, stderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr on success, got: %s", stderr)
	}
	if !strings.Contains(stdout, "evidence-make-gate-output") {
		t.Errorf("stdout should contain evidence ID, got: %s", stdout)
	}

	evidencePath := filepath.Join(tmp, runID, "evidence", "evidence-make-gate-output.json")
	if _, err := os.Stat(evidencePath); os.IsNotExist(err) {
		t.Errorf("evidence file should exist: %s", evidencePath)
	}
}

func TestWitnessEvidenceCreateJSONOutput(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-ev10"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test-json",
		"--kind", "command_output",
		"--role", "primary",
		"--title", "Test evidence",
		"--json",
	}

	stdout, stderr, code := captureRunBundleOutput(args, runWitnessEvidenceCreate)

	if code != 0 {
		t.Fatalf("create with --json failed with code %d, stderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr on success, got: %s", stderr)
	}

	var output struct {
		OK         bool   `json:"ok"`
		RunID      string `json:"run_id"`
		EvidenceID string `json:"evidence_id"`
		Path       string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("stdout should be valid JSON: %v\noutput: %s", err, stdout)
	}
	if !output.OK {
		t.Error("expected ok=true in JSON output")
	}
	if output.EvidenceID != "evidence-test-json" {
		t.Errorf("evidence_id = %q, want %q", output.EvidenceID, "evidence-test-json")
	}
	if output.Path == "" {
		t.Error("path should not be empty")
	}
}
