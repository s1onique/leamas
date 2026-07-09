package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// ============================================================================
// Attach evidence idempotent tests
// ============================================================================

func TestWitnessClaimAttachEvidenceIsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach05"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	claimArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "claim-test-idempotent",
		"--statement", "Test claim",
	}
	if code := runWitnessClaimCreate(claimArgs); code != 0 {
		t.Fatalf("failed to create claim: %d", code)
	}

	evidenceArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test-idempotent",
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
		"--claim-id", "claim-test-idempotent",
		"--evidence-id", "evidence-test-idempotent",
	}
	_, stderr, code := captureRunBundleOutput(attachArgs, runWitnessClaimAttachEvidence)
	if code != 0 {
		t.Fatalf("first attach-evidence failed with code %d, stderr: %s", code, stderr)
	}

	stdout, stderr, code := captureRunBundleOutput(attachArgs, runWitnessClaimAttachEvidence)
	if code != 0 {
		t.Fatalf("second attach-evidence (idempotent) failed with code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "already attached") {
		t.Errorf("stdout should mention already attached for idempotent call, got: %s", stdout)
	}
}

func TestWitnessClaimAttachEvidenceJSONIdempotentOutput(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-attach09"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if code := runWitnessRunBundleCreate(runBundleArgs); code != 0 {
		t.Fatalf("failed to create run bundle: %d", code)
	}

	claimArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "claim-test-json2",
		"--statement", "Test claim",
	}
	if code := runWitnessClaimCreate(claimArgs); code != 0 {
		t.Fatalf("failed to create claim: %d", code)
	}

	evidenceArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test-json2",
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
		"--claim-id", "claim-test-json2",
		"--evidence-id", "evidence-test-json2",
	}

	if code := runWitnessClaimAttachEvidence(attachArgs); code != 0 {
		t.Fatalf("first attach-evidence failed: %d", code)
	}

	attachArgs = append(attachArgs, "--json")
	stdout, stderr, code := captureRunBundleOutput(attachArgs, runWitnessClaimAttachEvidence)

	if code != 0 {
		t.Fatalf("idempotent attach-evidence with --json failed with code %d, stderr: %s", code, stderr)
	}

	var output struct {
		OK         bool   `json:"ok"`
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
	if output.Attached {
		t.Error("expected attached=false for idempotent (already attached) call")
	}
}
