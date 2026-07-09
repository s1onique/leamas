package claim

import (
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// TestNewClaimDefaults tests that NewClaim sets correct defaults.
func TestNewClaimDefaults(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	claim, err := NewClaim("claim-test-001", runID, "Test claim statement", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}

	// Check schema version
	if claim.SchemaVersion != ClaimSchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", claim.SchemaVersion, ClaimSchemaVersion)
	}

	// Check ID
	if claim.ID != "claim-test-001" {
		t.Errorf("ID = %q, want %q", claim.ID, "claim-test-001")
	}

	// Check RunID
	if claim.RunID != runID {
		t.Errorf("RunID = %v, want %v", claim.RunID, runID)
	}

	// Check statement
	if claim.Statement != "Test claim statement" {
		t.Errorf("Statement = %q, want %q", claim.Statement, "Test claim statement")
	}

	// Check default status
	if claim.Status != ClaimStatusOpen {
		t.Errorf("Status = %q, want %q", claim.Status, ClaimStatusOpen)
	}

	// Check default verdict
	if claim.Verdict != VerdictUnreviewed {
		t.Errorf("Verdict = %q, want %q", claim.Verdict, VerdictUnreviewed)
	}

	// Check evidence IDs is non-nil empty slice
	if claim.EvidenceIDs == nil {
		t.Error("EvidenceIDs is nil, want non-nil empty slice")
	}
	if len(claim.EvidenceIDs) != 0 {
		t.Errorf("len(EvidenceIDs) = %d, want 0", len(claim.EvidenceIDs))
	}

	// Check timestamps
	if !claim.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", claim.CreatedAt, now)
	}
	if !claim.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", claim.UpdatedAt, now)
	}
}

// TestValidateClaimRejectsEmptyStatement tests that empty statement fails.
func TestValidateClaimRejectsEmptyStatement(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	_, err := NewClaim("claim-test-001", runID, "", now)
	if err == nil {
		t.Error("NewClaim with empty statement should fail")
	}
	if err != ErrEmptyStatement {
		t.Errorf("Error = %v, want %v", err, ErrEmptyStatement)
	}
}

// TestValidateClaimRejectsBadStatus tests that invalid status fails.
func TestValidateClaimRejectsBadStatus(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	claim, err := NewClaim("claim-test-001", runID, "Test statement", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}

	// Set invalid status
	claim.Status = ClaimStatus("invalid")
	err = claim.Validate()
	if err == nil {
		t.Error("Validate with invalid status should fail")
	}
}

// TestValidateClaimRejectsBadVerdict tests that invalid verdict fails.
func TestValidateClaimRejectsBadVerdict(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	claim, err := NewClaim("claim-test-001", runID, "Test statement", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}

	// Set invalid verdict
	claim.Verdict = Verdict("invalid")
	err = claim.Validate()
	if err == nil {
		t.Error("Validate with invalid verdict should fail")
	}
}

// TestValidateClaimRejectsBadEvidenceID tests that invalid evidence ID fails.
func TestValidateClaimRejectsBadEvidenceID(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	claim, err := NewClaim("claim-test-001", runID, "Test statement", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}

	// Add invalid evidence ID
	claim.EvidenceIDs = []EvidenceID{"invalid-evidence"}
	err = claim.Validate()
	if err == nil {
		t.Error("Validate with invalid evidence ID should fail")
	}
}

// TestClaimStrictDecodeRejectsUnknownFields tests that unknown JSON fields are rejected.
func TestClaimStrictDecodeRejectsUnknownFields(t *testing.T) {
	// JSON with unknown field "extra_field"
	jsonWithUnknownField := `{
		"schema_version": "leamas.claim.v1",
		"id": "claim-test-001",
		"run_id": "run-20260709T071704Z-smoke01",
		"created_at": "2026-07-09T07:17:04Z",
		"updated_at": "2026-07-09T07:17:04Z",
		"statement": "Test statement",
		"status": "open",
		"verdict": "unreviewed",
		"evidence_ids": [],
		"extra_field": "should be rejected"
	}`

	_, err := StrictDecodeClaim([]byte(jsonWithUnknownField))
	if err == nil {
		t.Error("StrictDecodeClaim should reject unknown fields")
	}
}

// TestClaimRoundTripJSON tests that claims can be marshaled and unmarshaled.
func TestClaimRoundTripJSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	original, err := NewClaim("claim-test-001", runID, "Test statement", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}

	// Marshal
	data, err := MarshalClaimJSON(original)
	if err != nil {
		t.Fatalf("MarshalClaimJSON failed: %v", err)
	}

	// Unmarshal
	decoded, err := StrictDecodeClaim(data)
	if err != nil {
		t.Fatalf("StrictDecodeClaim failed: %v", err)
	}

	// Validate decoded claim
	if err := decoded.Validate(); err != nil {
		t.Fatalf("Decoded claim validation failed: %v", err)
	}

	// Verify fields match
	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %v, want %v", decoded.ID, original.ID)
	}
	if decoded.Statement != original.Statement {
		t.Errorf("Statement mismatch: got %v, want %v", decoded.Statement, original.Statement)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status mismatch: got %v, want %v", decoded.Status, original.Status)
	}
}

// TestClaimAddEvidence tests adding evidence to a claim.
func TestClaimAddEvidence(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	claim, err := NewClaim("claim-test-001", runID, "Test statement", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}

	// Add valid evidence
	added, err := claim.AddEvidence("evidence-test-001")
	if err != nil {
		t.Fatalf("AddEvidence failed: %v", err)
	}
	if !added {
		t.Error("AddEvidence should return true for new evidence")
	}
	if len(claim.EvidenceIDs) != 1 {
		t.Errorf("len(EvidenceIDs) = %d, want 1", len(claim.EvidenceIDs))
	}

	// Add same evidence again (should be idempotent)
	added, err = claim.AddEvidence("evidence-test-001")
	if err != nil {
		t.Fatalf("AddEvidence second call failed: %v", err)
	}
	if added {
		t.Error("AddEvidence should return false for duplicate evidence")
	}
	if len(claim.EvidenceIDs) != 1 {
		t.Errorf("len(EvidenceIDs) = %d, want 1 (idempotent)", len(claim.EvidenceIDs))
	}

	// Add invalid evidence ID
	_, err = claim.AddEvidence("invalid-evidence")
	if err == nil {
		t.Error("AddEvidence with invalid ID should fail")
	}
}

// TestClaimHasEvidence tests checking for evidence.
func TestClaimHasEvidence(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	claim, err := NewClaim("claim-test-001", runID, "Test statement", now)
	if err != nil {
		t.Fatalf("NewClaim failed: %v", err)
	}

	if claim.HasEvidence("evidence-test-001") {
		t.Error("HasEvidence should return false for non-existent evidence")
	}

	claim.EvidenceIDs = []EvidenceID{"evidence-test-001"}

	if !claim.HasEvidence("evidence-test-001") {
		t.Error("HasEvidence should return true for existent evidence")
	}
}

// TestIsValidClaimStatus tests status validation helper.
func TestIsValidClaimStatus(t *testing.T) {
	validStatuses := []ClaimStatus{
		ClaimStatusOpen,
		ClaimStatusSupported,
		ClaimStatusRejected,
		ClaimStatusUnknown,
	}

	for _, s := range validStatuses {
		if !IsValidClaimStatus(s) {
			t.Errorf("IsValidClaimStatus(%q) = false, want true", s)
		}
	}

	if IsValidClaimStatus("invalid") {
		t.Error("IsValidClaimStatus(\"invalid\") = true, want false")
	}
}

// TestIsValidVerdict tests verdict validation helper.
func TestIsValidVerdict(t *testing.T) {
	validVerdicts := []Verdict{
		VerdictUnreviewed,
		VerdictPass,
		VerdictFail,
		VerdictMixed,
	}

	for _, v := range validVerdicts {
		if !IsValidVerdict(v) {
			t.Errorf("IsValidVerdict(%q) = false, want true", v)
		}
	}

	if IsValidVerdict("invalid") {
		t.Error("IsValidVerdict(\"invalid\") = true, want false")
	}
}

// TestClaimValidationErrorMessages tests that error messages are descriptive.
func TestClaimValidationErrorMessages(t *testing.T) {
	// Test empty statement error
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	_, err := NewClaim("claim-test-001", runID, "", now)
	if err == nil || !strings.Contains(err.Error(), "statement") {
		t.Errorf("Error should mention 'statement', got: %v", err)
	}

	// Test invalid run ID error
	_, err = NewClaim("claim-test-001", runbundle.RunID("invalid"), "Test", now)
	if err == nil || !strings.Contains(err.Error(), "run") {
		t.Errorf("Error should mention 'run', got: %v", err)
	}
}
