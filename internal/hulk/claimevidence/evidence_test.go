package claimevidence

import (
	"strings"
	"testing"
)

// TestMissingEvidenceIDFails tests that missing evidence ID fails validation.
func TestMissingEvidenceIDFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Evidence: []Evidence{
			{
				ID:      EvidenceID(""),
				Kind:    EvidenceKindDigest,
				Summary: "Test evidence",
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Missing evidence ID should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Evidence.ID" && strings.Contains(f.Message, "non-empty") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing evidence ID, got: %v", result.Findings)
	}
}

// TestDuplicateEvidenceIDsFail tests that duplicate evidence IDs fail validation.
func TestDuplicateEvidenceIDsFail(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Evidence: []Evidence{
			{ID: EvidenceID("evidence-001"), Kind: EvidenceKindDigest, Summary: "Evidence 1"},
			{ID: EvidenceID("evidence-002"), Kind: EvidenceKindDigest, Summary: "Evidence 2"},
			{ID: EvidenceID("evidence-001"), Kind: EvidenceKindLog, Summary: "Evidence 3"},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Duplicate evidence IDs should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Evidence.ID" && strings.Contains(f.Message, "Duplicate evidence ID") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about duplicate evidence ID, got: %v", result.Findings)
	}
}

// TestInvalidEvidenceKindFails tests that invalid evidence kind fails validation.
func TestInvalidEvidenceKindFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Evidence: []Evidence{
			{
				ID:      EvidenceID("evidence-001"),
				Kind:    EvidenceKind("invalid_kind"),
				Summary: "Test evidence",
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Invalid evidence kind should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Evidence.Kind" && strings.Contains(f.Message, "Invalid evidence kind") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about invalid evidence kind, got: %v", result.Findings)
	}
}

// TestMissingEvidenceSummaryFails tests that missing evidence summary fails validation.
func TestMissingEvidenceSummaryFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Evidence: []Evidence{
			{
				ID:      EvidenceID("evidence-001"),
				Kind:    EvidenceKindDigest,
				Summary: "",
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Missing evidence summary should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Evidence.Summary" && strings.Contains(f.Message, "non-empty") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing evidence summary, got: %v", result.Findings)
	}
}

// TestEvidenceReferencesMissingSourceFails tests that referencing missing source fails.
func TestEvidenceReferencesMissingSourceFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Evidence: []Evidence{
			{
				ID:       EvidenceID("evidence-001"),
				Kind:     EvidenceKindDigest,
				Summary:  "Test evidence",
				SourceID: SourceID("nonexistent-source"),
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Evidence referencing missing source should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Evidence.SourceID" && strings.Contains(f.Message, "non-existent source") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing source reference, got: %v", result.Findings)
	}
}
