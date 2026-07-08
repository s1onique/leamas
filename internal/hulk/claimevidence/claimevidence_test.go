package claimevidence

import (
	"testing"
)

// TestEmptyBundlePasses tests that an empty bundle is valid.
func TestEmptyBundlePasses(t *testing.T) {
	bundle := ClaimEvidenceBundle{}
	result := Validate(bundle)
	if !result.OK() {
		t.Errorf("Empty bundle should pass, got findings: %v", result.Findings)
	}
}

// TestValidMinimalBundlePasses tests that a valid minimal bundle passes.
func TestValidMinimalBundlePasses(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:         ClaimID("claim-001"),
				Kind:       ClaimKindFact,
				Status:     ClaimStatusOpen,
				Summary:    "Test claim",
				Confidence: ConfidenceMedium,
			},
		},
		Evidence: []Evidence{
			{
				ID:      EvidenceID("evidence-001"),
				Kind:    EvidenceKindDigest,
				Summary: "Test evidence",
			},
		},
		Sources: []Source{
			{
				ID:      SourceID("source-001"),
				Kind:    SourceKindHuman,
				Summary: "Test source",
			},
		},
	}
	result := Validate(bundle)
	if !result.OK() {
		t.Errorf("Valid minimal bundle should pass, got findings: %v", result.Findings)
	}
}

// TestClaimWithValidEvidenceReferencePasses tests that claims with valid evidence refs pass.
func TestClaimWithValidEvidenceReferencePasses(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:          ClaimID("claim-001"),
				Kind:        ClaimKindFact,
				Status:      ClaimStatusOpen,
				Summary:     "Test claim",
				Confidence:  ConfidenceMedium,
				EvidenceIDs: []EvidenceID{EvidenceID("evidence-001")},
			},
		},
		Evidence: []Evidence{
			{
				ID:      EvidenceID("evidence-001"),
				Kind:    EvidenceKindDigest,
				Summary: "Test evidence",
			},
		},
		Sources: []Source{
			{
				ID:      SourceID("source-001"),
				Kind:    SourceKindHuman,
				Summary: "Test source",
			},
		},
	}
	result := Validate(bundle)
	if !result.OK() {
		t.Errorf("Claim with valid evidence reference should pass, got findings: %v", result.Findings)
	}
}

// TestEvidenceWithValidSourceReferencePasses tests that evidence with valid source refs pass.
func TestEvidenceWithValidSourceReferencePasses(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Evidence: []Evidence{
			{
				ID:       EvidenceID("evidence-001"),
				Kind:     EvidenceKindDigest,
				Summary:  "Test evidence",
				SourceID: SourceID("source-001"),
			},
		},
		Sources: []Source{
			{
				ID:      SourceID("source-001"),
				Kind:    SourceKindHuman,
				Summary: "Test source",
			},
		},
	}
	result := Validate(bundle)
	if !result.OK() {
		t.Errorf("Evidence with valid source reference should pass, got findings: %v", result.Findings)
	}
}

// TestValidArtifactSourcePasses tests that valid artifact sources pass.
func TestValidArtifactSourcePasses(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Sources: []Source{
			{
				ID:         SourceID("source-001"),
				Kind:       SourceKindArtifact,
				Summary:    "Test artifact source",
				ArtifactID: ArtifactID("artifact-001"),
			},
		},
	}
	result := Validate(bundle)
	if !result.OK() {
		t.Errorf("Valid artifact source should pass, got findings: %v", result.Findings)
	}
}

// TestFindingsAreDeterministic tests that validation produces deterministic findings.
func TestFindingsAreDeterministic(t *testing.T) {
	bundle := ClaimEvidenceBundle{
		Claims: []Claim{
			{
				ID:         ClaimID("claim-001"),
				Kind:       ClaimKind("invalid_kind"),
				Status:     ClaimStatus("invalid_status"),
				Summary:    "",
				Confidence: ConfidenceLevel("invalid_confidence"),
			},
		},
	}

	result1 := Validate(bundle)
	result2 := Validate(bundle)

	if len(result1.Findings) != len(result2.Findings) {
		t.Errorf("Findings should be deterministic: len(result1.Findings)=%d, len(result2.Findings)=%d",
			len(result1.Findings), len(result2.Findings))
	}

	for i := range result1.Findings {
		if result1.Findings[i] != result2.Findings[i] {
			t.Errorf("Finding %d should be deterministic: %v vs %v",
				i, result1.Findings[i], result2.Findings[i])
		}
	}
}
