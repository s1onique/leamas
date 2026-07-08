package claimevidence

import "testing"

// TestHelperValidityFunctions tests all helper validity functions.
func TestHelperValidityFunctions(t *testing.T) {
	// Test IsValidClaimStatus
	if !IsValidClaimStatus(ClaimStatusOpen) {
		t.Error("IsValidClaimStatus should return true for ClaimStatusOpen")
	}
	if !IsValidClaimStatus(ClaimStatusSupported) {
		t.Error("IsValidClaimStatus should return true for ClaimStatusSupported")
	}
	if !IsValidClaimStatus(ClaimStatusRefuted) {
		t.Error("IsValidClaimStatus should return true for ClaimStatusRefuted")
	}
	if !IsValidClaimStatus(ClaimStatusUnknown) {
		t.Error("IsValidClaimStatus should return true for ClaimStatusUnknown")
	}
	if IsValidClaimStatus(ClaimStatus("invalid")) {
		t.Error("IsValidClaimStatus should return false for invalid status")
	}

	// Test IsValidClaimKind
	if !IsValidClaimKind(ClaimKindFact) {
		t.Error("IsValidClaimKind should return true for ClaimKindFact")
	}
	if !IsValidClaimKind(ClaimKindInterpretation) {
		t.Error("IsValidClaimKind should return true for ClaimKindInterpretation")
	}
	if !IsValidClaimKind(ClaimKindRisk) {
		t.Error("IsValidClaimKind should return true for ClaimKindRisk")
	}
	if !IsValidClaimKind(ClaimKindLimitation) {
		t.Error("IsValidClaimKind should return true for ClaimKindLimitation")
	}
	if IsValidClaimKind(ClaimKind("invalid")) {
		t.Error("IsValidClaimKind should return false for invalid kind")
	}

	// Test IsValidEvidenceKind
	if !IsValidEvidenceKind(EvidenceKindDigest) {
		t.Error("IsValidEvidenceKind should return true for EvidenceKindDigest")
	}
	if !IsValidEvidenceKind(EvidenceKindLog) {
		t.Error("IsValidEvidenceKind should return true for EvidenceKindLog")
	}
	if !IsValidEvidenceKind(EvidenceKindProof) {
		t.Error("IsValidEvidenceKind should return true for EvidenceKindProof")
	}
	if !IsValidEvidenceKind(EvidenceKindCloseReport) {
		t.Error("IsValidEvidenceKind should return true for EvidenceKindCloseReport")
	}
	if !IsValidEvidenceKind(EvidenceKindObservation) {
		t.Error("IsValidEvidenceKind should return true for EvidenceKindObservation")
	}
	if !IsValidEvidenceKind(EvidenceKindOther) {
		t.Error("IsValidEvidenceKind should return true for EvidenceKindOther")
	}
	if IsValidEvidenceKind(EvidenceKind("invalid")) {
		t.Error("IsValidEvidenceKind should return false for invalid kind")
	}

	// Test IsValidSourceKind
	if !IsValidSourceKind(SourceKindArtifact) {
		t.Error("IsValidSourceKind should return true for SourceKindArtifact")
	}
	if !IsValidSourceKind(SourceKindHuman) {
		t.Error("IsValidSourceKind should return true for SourceKindHuman")
	}
	if !IsValidSourceKind(SourceKindAgent) {
		t.Error("IsValidSourceKind should return true for SourceKindAgent")
	}
	if !IsValidSourceKind(SourceKindVerifier) {
		t.Error("IsValidSourceKind should return true for SourceKindVerifier")
	}
	if IsValidSourceKind(SourceKind("invalid")) {
		t.Error("IsValidSourceKind should return false for invalid kind")
	}

	// Test IsValidConfidence
	if !IsValidConfidence(ConfidenceLow) {
		t.Error("IsValidConfidence should return true for ConfidenceLow")
	}
	if !IsValidConfidence(ConfidenceMedium) {
		t.Error("IsValidConfidence should return true for ConfidenceMedium")
	}
	if !IsValidConfidence(ConfidenceHigh) {
		t.Error("IsValidConfidence should return true for ConfidenceHigh")
	}
	if IsValidConfidence(ConfidenceLevel("invalid")) {
		t.Error("IsValidConfidence should return false for invalid confidence")
	}
}

// TestConstructors tests all constructor functions.
func TestConstructors(t *testing.T) {
	// Test NewClaim
	claim := NewClaim(ClaimID("claim-001"), ClaimKindFact, "Test claim")
	if claim.ID != "claim-001" {
		t.Errorf("NewClaim ID mismatch: got %s, want claim-001", claim.ID)
	}
	if claim.Kind != ClaimKindFact {
		t.Errorf("NewClaim Kind mismatch: got %s, want fact", claim.Kind)
	}
	if claim.Status != ClaimStatusOpen {
		t.Errorf("NewClaim Status mismatch: got %s, want open", claim.Status)
	}
	if claim.Summary != "Test claim" {
		t.Errorf("NewClaim Summary mismatch: got %s, want Test claim", claim.Summary)
	}
	if claim.Confidence != ConfidenceMedium {
		t.Errorf("NewClaim Confidence mismatch: got %s, want medium", claim.Confidence)
	}

	// Test NewEvidence
	evidence := NewEvidence(EvidenceID("evidence-001"), EvidenceKindDigest, "Test evidence")
	if evidence.ID != "evidence-001" {
		t.Errorf("NewEvidence ID mismatch: got %s, want evidence-001", evidence.ID)
	}
	if evidence.Kind != EvidenceKindDigest {
		t.Errorf("NewEvidence Kind mismatch: got %s, want digest", evidence.Kind)
	}
	if evidence.Summary != "Test evidence" {
		t.Errorf("NewEvidence Summary mismatch: got %s, want Test evidence", evidence.Summary)
	}

	// Test NewSource
	source := NewSource(SourceID("source-001"), SourceKindHuman, "Test source")
	if source.ID != "source-001" {
		t.Errorf("NewSource ID mismatch: got %s, want source-001", source.ID)
	}
	if source.Kind != SourceKindHuman {
		t.Errorf("NewSource Kind mismatch: got %s, want human", source.Kind)
	}
	if source.Summary != "Test source" {
		t.Errorf("NewSource Summary mismatch: got %s, want Test source", source.Summary)
	}
}
