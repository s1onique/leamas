package claim

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// TestStoreWriteReadClaim tests writing and reading a claim.
func TestStoreWriteReadClaim(t *testing.T) {
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
	claim, err := NewClaim("claim-test-001", bundle.ID, "Test claim statement", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}

	if err := store.WriteClaim(claim); err != nil {
		t.Fatalf("WriteClaim failed: %v", err)
	}

	readClaim, err := store.ReadClaim("claim-test-001")
	if err != nil {
		t.Fatalf("ReadClaim failed: %v", err)
	}

	if readClaim.ID != claim.ID {
		t.Errorf("ID mismatch: got %v, want %v", readClaim.ID, claim.ID)
	}
	if readClaim.Statement != claim.Statement {
		t.Errorf("Statement mismatch: got %v, want %v", readClaim.Statement, claim.Statement)
	}
	if readClaim.Status != claim.Status {
		t.Errorf("Status mismatch: got %v, want %v", readClaim.Status, claim.Status)
	}
	if readClaim.Verdict != claim.Verdict {
		t.Errorf("Verdict mismatch: got %v, want %v", readClaim.Verdict, claim.Verdict)
	}
}

// TestStoreWriteReadEvidence tests writing and reading evidence.
func TestStoreWriteReadEvidence(t *testing.T) {
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
	evidence, err := NewEvidence(
		"evidence-test-001",
		bundle.ID,
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"Test evidence title",
		now,
	)
	if err != nil {
		t.Fatalf("NewEvidence failed: %v", err)
	}
	evidence.RelativePath = "digests/head.txt"
	evidence.Summary = "Test summary"

	if err := store.WriteEvidence(evidence); err != nil {
		t.Fatalf("WriteEvidence failed: %v", err)
	}

	readEvidence, err := store.ReadEvidence("evidence-test-001")
	if err != nil {
		t.Fatalf("ReadEvidence failed: %v", err)
	}

	if readEvidence.ID != evidence.ID {
		t.Errorf("ID mismatch: got %v, want %v", readEvidence.ID, evidence.ID)
	}
	if readEvidence.Kind != evidence.Kind {
		t.Errorf("Kind mismatch: got %v, want %v", readEvidence.Kind, evidence.Kind)
	}
	if readEvidence.Role != evidence.Role {
		t.Errorf("Role mismatch: got %v, want %v", readEvidence.Role, evidence.Role)
	}
	if readEvidence.RelativePath != evidence.RelativePath {
		t.Errorf("RelativePath mismatch: got %v, want %v", readEvidence.RelativePath, evidence.RelativePath)
	}
}

// TestStoreRejectsClaimRunIDMismatch tests that claims with mismatched run IDs are rejected.
func TestStoreRejectsClaimRunIDMismatch(t *testing.T) {
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
	claim, err := NewClaim("claim-test-001", "run-different-run", "Test claim", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}

	err = store.WriteClaim(claim)
	if err == nil {
		t.Error("WriteClaim with mismatched run ID should fail")
	}
}

// TestStoreRejectsEvidenceRunIDMismatch tests that evidence with mismatched run IDs are rejected.
func TestStoreRejectsEvidenceRunIDMismatch(t *testing.T) {
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
	evidence, err := NewEvidence(
		"evidence-test-001",
		"run-different-run",
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"Test evidence",
		now,
	)
	if err != nil {
		t.Fatalf("NewEvidence failed: %v", err)
	}

	err = store.WriteEvidence(evidence)
	if err == nil {
		t.Error("WriteEvidence with mismatched run ID should fail")
	}
}

// TestStoreRejectsMissingClaim tests that reading a missing claim fails.
func TestStoreRejectsMissingClaim(t *testing.T) {
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
	_, err = store.ReadClaim("claim-nonexistent")
	if err == nil {
		t.Error("ReadClaim for nonexistent claim should fail")
	}
	if err != ErrClaimNotFound {
		t.Errorf("Error = %v, want %v", err, ErrClaimNotFound)
	}
}

// TestStoreRejectsMissingEvidence tests that reading missing evidence fails.
func TestStoreRejectsMissingEvidence(t *testing.T) {
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
	_, err = store.ReadEvidence("evidence-nonexistent")
	if err == nil {
		t.Error("ReadEvidence for nonexistent evidence should fail")
	}
	if err != ErrEvidenceNotFound {
		t.Errorf("Error = %v, want %v", err, ErrEvidenceNotFound)
	}
}

// TestStoreClaimFilePath tests that claim files are written to correct path.
func TestStoreClaimFilePath(t *testing.T) {
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

	expectedPath := filepath.Join(bundle.Path, "claims", "claim-test-001.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Claim file not found at expected path: %s", expectedPath)
	}
}

// TestStoreEvidenceFilePath tests that evidence files are written to correct path.
func TestStoreEvidenceFilePath(t *testing.T) {
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

	expectedPath := filepath.Join(bundle.Path, "evidence", "evidence-test-001.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Evidence file not found at expected path: %s", expectedPath)
	}
}
