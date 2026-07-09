package claim

import (
	"os"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// TestStoreReadClaimRejectsIDMismatch tests that reading a claim with mismatched ID fails.
func TestStoreReadClaimRejectsIDMismatch(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	bundle, err := runbundle.Create(runbundle.CreateOptions{
		Root:  root,
		RunID: "run-20260709T071704Z-smoke01",
		Now:   func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("runbundle.Create failed: %v", err)
	}

	store := NewStore(bundle)

	// Create and write a claim with ID claim-a
	claim, err := NewClaim("claim-a", bundle.ID, "Test claim", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}
	if err := store.WriteClaim(claim); err != nil {
		t.Fatalf("WriteClaim failed: %v", err)
	}

	// Tamper with the file to change ID to claim-b
	badJSON := `{
		"schema_version": "leamas.claim.v1",
		"id": "claim-b",
		"run_id": "run-20260709T071704Z-smoke01",
		"created_at": "2026-07-09T07:17:04Z",
		"updated_at": "2026-07-09T07:17:04Z",
		"statement": "Test claim",
		"status": "open",
		"verdict": "unreviewed",
		"evidence_ids": []
	}`
	path := bundle.Path + "/claims/claim-a.json"
	if err := os.WriteFile(path, []byte(badJSON), 0644); err != nil {
		t.Fatalf("Failed to write tampered claim: %v", err)
	}

	// Reading claim-a should fail because ID mismatch
	_, err = store.ReadClaim("claim-a")
	if err == nil {
		t.Error("ReadClaim should reject tampered claim with wrong ID")
	}
	if err != ErrClaimIDMismatch {
		t.Errorf("Error = %v, want %v", err, ErrClaimIDMismatch)
	}
}

// TestStoreReadEvidenceRejectsIDMismatch tests that reading evidence with mismatched ID fails.
func TestStoreReadEvidenceRejectsIDMismatch(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	bundle, err := runbundle.Create(runbundle.CreateOptions{
		Root:  root,
		RunID: "run-20260709T071704Z-smoke01",
		Now:   func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("runbundle.Create failed: %v", err)
	}

	store := NewStore(bundle)

	// Create and write evidence with ID evidence-a
	evidence, err := NewEvidence(
		"evidence-a",
		bundle.ID,
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"Test evidence",
		now,
	)
	if err != nil {
		t.Fatalf("NewEvidence failed: %v", err)
	}
	if err := store.WriteEvidence(evidence); err != nil {
		t.Fatalf("WriteEvidence failed: %v", err)
	}

	// Tamper with the file to change ID to evidence-b
	badJSON := `{
		"schema_version": "leamas.evidence.v1",
		"id": "evidence-b",
		"run_id": "run-20260709T071704Z-smoke01",
		"created_at": "2026-07-09T07:17:04Z",
		"kind": "command_output",
		"role": "primary",
		"title": "Test evidence"
	}`
	path := bundle.Path + "/evidence/evidence-a.json"
	if err := os.WriteFile(path, []byte(badJSON), 0644); err != nil {
		t.Fatalf("Failed to write tampered evidence: %v", err)
	}

	// Reading evidence-a should fail because ID mismatch
	_, err = store.ReadEvidence("evidence-a")
	if err == nil {
		t.Error("ReadEvidence should reject tampered evidence with wrong ID")
	}
	if err != ErrEvidenceIDMismatch {
		t.Errorf("Error = %v, want %v", err, ErrEvidenceIDMismatch)
	}
}

// TestStoreAddEvidenceToClaimAllowsNilClock tests that nil clock is tolerated.
func TestStoreAddEvidenceToClaimAllowsNilClock(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	bundle, err := runbundle.Create(runbundle.CreateOptions{
		Root:  root,
		RunID: "run-20260709T071704Z-smoke01",
		Now:   func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("runbundle.Create failed: %v", err)
	}

	store := NewStore(bundle)

	// Create and write a claim
	claim, err := NewClaim("claim-test-001", bundle.ID, "Test claim", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}
	if err := store.WriteClaim(claim); err != nil {
		t.Fatalf("WriteClaim failed: %v", err)
	}

	// Create and write evidence
	evidence, err := NewEvidence(
		"evidence-test-001",
		bundle.ID,
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"Test evidence",
		now,
	)
	if err != nil {
		t.Fatalf("NewEvidence failed: %v", err)
	}
	if err := store.WriteEvidence(evidence); err != nil {
		t.Fatalf("WriteEvidence failed: %v", err)
	}

	// Add evidence with nil clock - should not panic
	_, err = store.AddEvidenceToClaim("claim-test-001", "evidence-test-001", nil)
	if err != nil {
		t.Errorf("AddEvidenceToClaim with nil clock failed: %v", err)
	}
}
