package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// ============================================================================
// Attach evidence tests
// ============================================================================

func TestWitnessClaimAttachEvidenceRequiresRunID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--claim-id", "claim-test",
		"--evidence-id", "evidence-test",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaimAttachEvidence)

	if code == 0 {
		t.Error("expected non-zero exit when --run-id is missing")
	}
	if !strings.Contains(stderr, "--run-id") && !strings.Contains(stderr, "requires --run-id") {
		t.Errorf("stderr should mention --run-id requirement, got: %s", stderr)
	}
}

func TestWitnessClaimAttachEvidenceRequiresClaimID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach02"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--evidence-id", "evidence-test",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaimAttachEvidence)

	if code == 0 {
		t.Error("expected non-zero exit when --claim-id is missing")
	}
	if !strings.Contains(stderr, "--claim-id") && !strings.Contains(stderr, "requires --claim-id") {
		t.Errorf("stderr should mention --claim-id requirement, got: %s", stderr)
	}
}

func TestWitnessClaimAttachEvidenceRequiresEvidenceID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach03"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--claim-id", "claim-test",
	}
	_, stderr, code := captureRunBundleOutput(args, runWitnessClaimAttachEvidence)

	if code == 0 {
		t.Error("expected non-zero exit when --evidence-id is missing")
	}
	if !strings.Contains(stderr, "--evidence-id") && !strings.Contains(stderr, "requires --evidence-id") {
		t.Errorf("stderr should mention --evidence-id requirement, got: %s", stderr)
	}
}

func TestWitnessClaimAttachEvidenceLinksEvidence(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach04"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	claimArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "claim-test-attach",
		"--statement", "Test claim",
	}
	if code := runWitnessClaimCreate(claimArgs); code != 0 {
		t.Fatalf("failed to create claim: %d", code)
	}

	evidenceArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test-attach",
		"--kind", "command_output",
		"--role", "primary",
		"--title", "Test evidence",
	}
	if code := runWitnessEvidenceCreate(evidenceArgs); code != 0 {
		t.Fatalf("failed to create evidence: %d", code)
	}

	attachArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--claim-id", "claim-test-attach",
		"--evidence-id", "evidence-test-attach",
	}
	stdout, stderr, code := captureRunBundleOutput(attachArgs, runWitnessClaimAttachEvidence)

	if code != 0 {
		t.Fatalf("attach-evidence failed with code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "attached") {
		t.Errorf("stdout should mention attached, got: %s", stdout)
	}
	if !strings.Contains(stdout, "claim-test-attach") {
		t.Errorf("stdout should contain claim ID, got: %s", stdout)
	}
	if !strings.Contains(stdout, "evidence-test-attach") {
		t.Errorf("stdout should contain evidence ID, got: %s", stdout)
	}
}

func TestWitnessClaimAttachEvidenceRejectsMissingClaim(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach06"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	evidenceArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-missing-claim",
		"--kind", "command_output",
		"--role", "primary",
		"--title", "Test evidence",
	}
	if code := runWitnessEvidenceCreate(evidenceArgs); code != 0 {
		t.Fatalf("failed to create evidence: %d", code)
	}

	attachArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--claim-id", "claim-does-not-exist",
		"--evidence-id", "evidence-missing-claim",
	}
	_, stderr, code := captureRunBundleOutput(attachArgs, runWitnessClaimAttachEvidence)

	if code == 0 {
		t.Error("expected non-zero exit for missing claim")
	}
	if !strings.Contains(stderr, "claim") && !strings.Contains(stderr, "not found") {
		t.Errorf("stderr should mention claim not found, got: %s", stderr)
	}
}

func TestWitnessClaimAttachEvidenceRejectsMissingEvidence(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach07"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	claimArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "claim-missing-evidence",
		"--statement", "Test claim",
	}
	if code := runWitnessClaimCreate(claimArgs); code != 0 {
		t.Fatalf("failed to create claim: %d", code)
	}

	attachArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--claim-id", "claim-missing-evidence",
		"--evidence-id", "evidence-does-not-exist",
	}
	_, stderr, code := captureRunBundleOutput(attachArgs, runWitnessClaimAttachEvidence)

	if code == 0 {
		t.Error("expected non-zero exit for missing evidence")
	}
	if !strings.Contains(stderr, "evidence") && !strings.Contains(stderr, "not found") {
		t.Errorf("stderr should mention evidence not found, got: %s", stderr)
	}
}

func TestWitnessClaimAttachEvidenceJSONOutput(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach08"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	claimArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "claim-test-json",
		"--statement", "Test claim",
	}
	if code := runWitnessClaimCreate(claimArgs); code != 0 {
		t.Fatalf("failed to create claim: %d", code)
	}

	evidenceArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test-json",
		"--kind", "command_output",
		"--role", "primary",
		"--title", "Test evidence",
	}
	if code := runWitnessEvidenceCreate(evidenceArgs); code != 0 {
		t.Fatalf("failed to create evidence: %d", code)
	}

	attachArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--claim-id", "claim-test-json",
		"--evidence-id", "evidence-test-json",
		"--json",
	}
	stdout, stderr, code := captureRunBundleOutput(attachArgs, runWitnessClaimAttachEvidence)

	if code != 0 {
		t.Fatalf("attach-evidence with --json failed with code %d, stderr: %s", code, stderr)
	}

	var output struct {
		OK         bool   `json:"ok"`
		RunID      string `json:"run_id"`
		ClaimID    string `json:"claim_id"`
		EvidenceID string `json:"evidence_id"`
		Attached   bool   `json:"attached"`
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
	if output.EvidenceID != "evidence-test-json" {
		t.Errorf("evidence_id = %q, want %q", output.EvidenceID, "evidence-test-json")
	}
	if !output.Attached {
		t.Error("expected attached=true for first attachment")
	}
}

func TestWitnessClaimAttachEvidenceRejectsInvalidClaimID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach10"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	evidenceArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-invalid-claim",
		"--kind", "command_output",
		"--role", "primary",
		"--title", "Test evidence",
	}
	if code := runWitnessEvidenceCreate(evidenceArgs); code != 0 {
		t.Fatalf("failed to create evidence: %d", code)
	}

	attachArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--claim-id", "invalid-claim-id",
		"--evidence-id", "evidence-invalid-claim",
	}
	_, stderr, code := captureRunBundleOutput(attachArgs, runWitnessClaimAttachEvidence)

	if code == 0 {
		t.Error("expected non-zero exit for invalid claim ID")
	}
	if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "claim ID") {
		t.Errorf("stderr should mention invalid claim ID, got: %s", stderr)
	}
}

func TestWitnessClaimAttachEvidenceRejectsInvalidEvidenceID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach11"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	claimArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "claim-invalid-evidence",
		"--statement", "Test claim",
	}
	if code := runWitnessClaimCreate(claimArgs); code != 0 {
		t.Fatalf("failed to create claim: %d", code)
	}

	attachArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--claim-id", "claim-invalid-evidence",
		"--evidence-id", "invalid-evidence-id",
	}
	_, stderr, code := captureRunBundleOutput(attachArgs, runWitnessClaimAttachEvidence)

	if code == 0 {
		t.Error("expected non-zero exit for invalid evidence ID")
	}
	if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "evidence ID") {
		t.Errorf("stderr should mention invalid evidence ID, got: %s", stderr)
	}
}
