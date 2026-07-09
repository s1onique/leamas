package claim

import (
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// TestStoreAddEvidenceToClaim tests adding evidence to a claim.
func TestStoreAddEvidenceToClaim(t *testing.T) {
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
	claim, err := NewClaim("claim-test-001", bundle.ID, "Test claim", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}
	if err := store.WriteClaim(claim); err != nil {
		t.Fatalf("WriteClaim failed: %v", err)
	}

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

	updatedClaim, err := store.AddEvidenceToClaim("claim-test-001", "evidence-test-001", time.Now)
	if err != nil {
		t.Fatalf("AddEvidenceToClaim failed: %v", err)
	}

	if len(updatedClaim.EvidenceIDs) != 1 {
		t.Errorf("len(EvidenceIDs) = %d, want 1", len(updatedClaim.EvidenceIDs))
	}
	if updatedClaim.EvidenceIDs[0] != "evidence-test-001" {
		t.Errorf("EvidenceIDs[0] = %v, want evidence-test-001", updatedClaim.EvidenceIDs[0])
	}
}

// TestStoreAddEvidenceToClaimIsIdempotent tests that adding evidence is idempotent.
func TestStoreAddEvidenceToClaimIsIdempotent(t *testing.T) {
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
	claim, err := NewClaim("claim-test-001", bundle.ID, "Test claim", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}
	if err := store.WriteClaim(claim); err != nil {
		t.Fatalf("WriteClaim failed: %v", err)
	}

	// Add evidence twice
	_, err = store.AddEvidenceToClaim("claim-test-001", "evidence-test-001", time.Now)
	if err != nil {
		t.Fatalf("AddEvidenceToClaim first call failed: %v", err)
	}

	_, err = store.AddEvidenceToClaim("claim-test-001", "evidence-test-001", time.Now)
	if err != nil {
		t.Fatalf("AddEvidenceToClaim second call failed: %v", err)
	}

	// Read claim and verify only one evidence
	readClaim, err := store.ReadClaim("claim-test-001")
	if err != nil {
		t.Fatalf("ReadClaim failed: %v", err)
	}

	if len(readClaim.EvidenceIDs) != 1 {
		t.Errorf("len(EvidenceIDs) = %d, want 1 (idempotent)", len(readClaim.EvidenceIDs))
	}
}
