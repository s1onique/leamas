package claimevidence

import (
	"strings"
	"testing"
)

// TestMissingClaimIDFails tests that missing claim ID fails validation.
func TestMissingClaimIDFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:         ClaimID(""),
				Kind:       ClaimKindFact,
				Status:     ClaimStatusOpen,
				Summary:    "Test claim",
				Confidence: ConfidenceMedium,
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Missing claim ID should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Claim.ID" && strings.Contains(f.Message, "non-empty") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing claim ID, got: %v", result.Findings)
	}
}

// TestDuplicateClaimIDsFail tests that duplicate claim IDs fail validation.
func TestDuplicateClaimIDsFail(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:         ClaimID("claim-001"),
				Kind:       ClaimKindFact,
				Status:     ClaimStatusOpen,
				Summary:    "Claim 1",
				Confidence: ConfidenceMedium,
			},
			{
				ID:         ClaimID("claim-002"),
				Kind:       ClaimKindFact,
				Status:     ClaimStatusOpen,
				Summary:    "Claim 2",
				Confidence: ConfidenceMedium,
			},
			{
				ID:         ClaimID("claim-001"),
				Kind:       ClaimKindFact,
				Status:     ClaimStatusOpen,
				Summary:    "Claim 3",
				Confidence: ConfidenceMedium,
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Duplicate claim IDs should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Claim.ID" && strings.Contains(f.Message, "Duplicate claim ID") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about duplicate claim ID, got: %v", result.Findings)
	}
}

// TestInvalidClaimKindFails tests that invalid claim kind fails validation.
func TestInvalidClaimKindFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:         ClaimID("claim-001"),
				Kind:       ClaimKind("invalid_kind"),
				Status:     ClaimStatusOpen,
				Summary:    "Test claim",
				Confidence: ConfidenceMedium,
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Invalid claim kind should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Claim.Kind" && strings.Contains(f.Message, "Invalid claim kind") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about invalid claim kind, got: %v", result.Findings)
	}
}

// TestInvalidClaimStatusFails tests that invalid claim status fails validation.
func TestInvalidClaimStatusFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:         ClaimID("claim-001"),
				Kind:       ClaimKindFact,
				Status:     ClaimStatus("invalid_status"),
				Summary:    "Test claim",
				Confidence: ConfidenceMedium,
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Invalid claim status should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Claim.Status" && strings.Contains(f.Message, "Invalid claim status") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about invalid claim status, got: %v", result.Findings)
	}
}

// TestMissingClaimSummaryFails tests that missing claim summary fails validation.
func TestMissingClaimSummaryFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:         ClaimID("claim-001"),
				Kind:       ClaimKindFact,
				Status:     ClaimStatusOpen,
				Summary:    "",
				Confidence: ConfidenceMedium,
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Missing claim summary should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Claim.Summary" && strings.Contains(f.Message, "non-empty") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing claim summary, got: %v", result.Findings)
	}
}

// TestInvalidConfidenceFails tests that invalid confidence fails validation.
func TestInvalidConfidenceFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:         ClaimID("claim-001"),
				Kind:       ClaimKindFact,
				Status:     ClaimStatusOpen,
				Summary:    "Test claim",
				Confidence: ConfidenceLevel("invalid_confidence"),
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Invalid confidence should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Claim.Confidence" && strings.Contains(f.Message, "Invalid confidence") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about invalid confidence, got: %v", result.Findings)
	}
}

// TestClaimReferencesMissingEvidenceFails tests that referencing missing evidence fails.
func TestClaimReferencesMissingEvidenceFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:          ClaimID("claim-001"),
				Kind:        ClaimKindFact,
				Status:      ClaimStatusOpen,
				Summary:     "Test claim",
				Confidence:  ConfidenceMedium,
				EvidenceIDs: []EvidenceID{EvidenceID("nonexistent-evidence")},
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Claim referencing missing evidence should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Claim.EvidenceIDs" && strings.Contains(f.Message, "non-existent evidence") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing evidence reference, got: %v", result.Findings)
	}
}

// TestEmptyClaimLimitationFails tests that empty claim limitation fails validation.
func TestEmptyClaimLimitationFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:          ClaimID("claim-001"),
				Kind:        ClaimKindFact,
				Status:      ClaimStatusOpen,
				Summary:     "Test claim",
				Confidence:  ConfidenceMedium,
				Limitations: []string{"Limitation 1", "", "Limitation 3"},
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Empty claim limitation should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Claim.Limitations" && strings.Contains(f.Message, "empty strings") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about empty limitation, got: %v", result.Findings)
	}
}
