package claim

import (
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// TestNewEvidenceRejectsBadKind tests that NewEvidence rejects invalid kind.
func TestNewEvidenceRejectsBadKind(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	_, err := NewEvidence(
		"evidence-test-001",
		runID,
		EvidenceKind("invalid"),
		EvidenceRolePrimary,
		"Test title",
		now,
	)
	if err == nil {
		t.Error("NewEvidence with invalid kind should fail")
	}
	if err != ErrInvalidKind {
		t.Errorf("Error = %v, want %v", err, ErrInvalidKind)
	}
}

// TestNewEvidenceRejectsBadRole tests that NewEvidence rejects invalid role.
func TestNewEvidenceRejectsBadRole(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	_, err := NewEvidence(
		"evidence-test-001",
		runID,
		EvidenceKindCommandOutput,
		EvidenceRole("invalid"),
		"Test title",
		now,
	)
	if err == nil {
		t.Error("NewEvidence with invalid role should fail")
	}
	if err != ErrInvalidRole {
		t.Errorf("Error = %v, want %v", err, ErrInvalidRole)
	}
}
