package claimevidence

import (
	"strings"
	"testing"
)

// TestMissingSourceIDFails tests that missing source ID fails validation.
func TestMissingSourceIDFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Sources: []Source{
			{
				ID:      SourceID(""),
				Kind:    SourceKindHuman,
				Summary: "Test source",
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Missing source ID should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Source.ID" && strings.Contains(f.Message, "non-empty") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing source ID, got: %v", result.Findings)
	}
}

// TestDuplicateSourceIDsFail tests that duplicate source IDs fail validation.
func TestDuplicateSourceIDsFail(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Sources: []Source{
			{ID: SourceID("source-001"), Kind: SourceKindHuman, Summary: "Source 1"},
			{ID: SourceID("source-002"), Kind: SourceKindHuman, Summary: "Source 2"},
			{ID: SourceID("source-001"), Kind: SourceKindAgent, Summary: "Source 3"},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Duplicate source IDs should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Source.ID" && strings.Contains(f.Message, "Duplicate source ID") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about duplicate source ID, got: %v", result.Findings)
	}
}

// TestInvalidSourceKindFails tests that invalid source kind fails validation.
func TestInvalidSourceKindFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Sources: []Source{
			{
				ID:      SourceID("source-001"),
				Kind:    SourceKind("invalid_kind"),
				Summary: "Test source",
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Invalid source kind should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Source.Kind" && strings.Contains(f.Message, "Invalid source kind") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about invalid source kind, got: %v", result.Findings)
	}
}

// TestMissingSourceSummaryFails tests that missing source summary fails validation.
func TestMissingSourceSummaryFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Sources: []Source{
			{
				ID:      SourceID("source-001"),
				Kind:    SourceKindHuman,
				Summary: "",
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Missing source summary should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Source.Summary" && strings.Contains(f.Message, "non-empty") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing source summary, got: %v", result.Findings)
	}
}

// TestArtifactSourceWithoutArtifactIDFails tests that artifact source without ArtifactID fails.
func TestArtifactSourceWithoutArtifactIDFails(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Sources: []Source{
			{
				ID:         SourceID("source-001"),
				Kind:       SourceKindArtifact,
				Summary:    "Test source",
				ArtifactID: "",
			},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Artifact source without ArtifactID should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "Source.ArtifactID" && strings.Contains(f.Message, "non-empty for SourceKindArtifact") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing ArtifactID for artifact source, got: %v", result.Findings)
	}
}
