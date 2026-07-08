package runbundle

import (
	"testing"
)

func TestValidationResultOK(t *testing.T) {
	validResult := ValidationResult{Findings: []ValidationFinding{}}
	if !validResult.OK() {
		t.Error("ValidationResult with no findings should be OK")
	}

	invalidResult := ValidationResult{Findings: []ValidationFinding{
		{Path: "Test.Path", Message: "Test message"},
	}}
	if invalidResult.OK() {
		t.Error("ValidationResult with findings should not be OK")
	}
}

func TestNewRunBundle(t *testing.T) {
	bundle := NewRunBundle(RunBundleID("bundle-001"), RunID("run-001"), "2024-01-15T10:00:00Z", "Test summary")

	if bundle.ID != "bundle-001" {
		t.Errorf("Expected ID 'bundle-001', got '%s'", bundle.ID)
	}
	if bundle.RunID != "run-001" {
		t.Errorf("Expected RunID 'run-001', got '%s'", bundle.RunID)
	}
	if bundle.CreatedAt != "2024-01-15T10:00:00Z" {
		t.Errorf("Expected CreatedAt '2024-01-15T10:00:00Z', got '%s'", bundle.CreatedAt)
	}
	if bundle.Summary != "Test summary" {
		t.Errorf("Expected Summary 'Test summary', got '%s'", bundle.Summary)
	}
	if bundle.Status != RunBundleDraft {
		t.Errorf("Expected default Status 'draft', got '%s'", bundle.Status)
	}
}

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		status   RunBundleStatus
		expected bool
	}{
		{RunBundleDraft, true},
		{RunBundleComplete, true},
		{RunBundleInvalid, true},
		{RunBundleStatus("unknown"), false},
		{RunBundleStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := IsValidStatus(tt.status)
			if got != tt.expected {
				t.Errorf("IsValidStatus(%q) = %v, want %v", tt.status, got, tt.expected)
			}
		})
	}
}

func TestIsValidArtifactKind(t *testing.T) {
	tests := []struct {
		kind     ArtifactKind
		expected bool
	}{
		{ArtifactDigest, true},
		{ArtifactCloseReport, true},
		{ArtifactProof, true},
		{ArtifactLog, true},
		{ArtifactOther, true},
		{ArtifactKind("unknown"), false},
		{ArtifactKind(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			got := IsValidArtifactKind(tt.kind)
			if got != tt.expected {
				t.Errorf("IsValidArtifactKind(%q) = %v, want %v", tt.kind, got, tt.expected)
			}
		})
	}
}

func TestValidBundleWithArtifactsAndEvidence(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleComplete,
		Summary:   "Complete test bundle with all fields",
		CreatedAt: "2024-01-15T10:00:00Z",
		Artifacts: []ArtifactRef{
			{ID: ArtifactID("artifact-1"), Kind: ArtifactDigest, Path: "/artifacts/digest.md", Role: "main", Digest: "sha256:abc123"},
			{ID: ArtifactID("artifact-2"), Kind: ArtifactLog, Path: "/artifacts/log.txt", Role: "log", Digest: "sha256:def456"},
			{ID: ArtifactID("artifact-3"), Kind: ArtifactProof, Path: "/artifacts/proof.json", Role: "proof", Digest: "sha256:ghi789"},
		},
		Claims: []ClaimRef{
			{ID: ClaimID("claim-1"), Summary: "Build succeeded"},
			{ID: ClaimID("claim-2"), Summary: "Tests passed"},
		},
		Evidence: []EvidenceRef{
			{ID: EvidenceID("evidence-1"), ArtifactID: ArtifactID("artifact-1"), Summary: "Digest artifact evidence"},
			{ID: EvidenceID("evidence-2"), ArtifactID: ArtifactID("artifact-2"), Summary: "Log artifact evidence"},
		},
		Limitations: []string{"Limited to unit tests", "No integration tests included"},
	}

	result := Validate(bundle)
	if !result.OK() {
		t.Errorf("Valid complete bundle should pass, got findings: %v", result.Findings)
	}
}

func TestAllStatusValues(t *testing.T) {
	for _, status := range []RunBundleStatus{RunBundleDraft, RunBundleComplete, RunBundleInvalid} {
		t.Run(string(status), func(t *testing.T) {
			bundle := RunBundle{
				ID:        RunBundleID("bundle-001"),
				RunID:     RunID("run-001"),
				Status:    status,
				Summary:   "Test bundle",
				CreatedAt: "2024-01-15T10:00:00Z",
			}
			result := Validate(bundle)
			for _, f := range result.Findings {
				if f.Path == "RunBundle.Status" {
					t.Errorf("Status %q should be valid, got finding: %s", status, f.Message)
				}
			}
		})
	}
}

func TestAllArtifactKindValues(t *testing.T) {
	for _, kind := range []ArtifactKind{ArtifactDigest, ArtifactCloseReport, ArtifactProof, ArtifactLog, ArtifactOther} {
		t.Run(string(kind), func(t *testing.T) {
			bundle := RunBundle{
				ID:        RunBundleID("bundle-001"),
				RunID:     RunID("run-001"),
				Status:    RunBundleDraft,
				Summary:   "Test bundle",
				CreatedAt: "2024-01-15T10:00:00Z",
				Artifacts: []ArtifactRef{
					{ID: ArtifactID("artifact-1"), Kind: kind, Path: "/path/1"},
				},
			}
			result := Validate(bundle)
			for _, f := range result.Findings {
				if f.Path == "ArtifactRef.Kind" {
					t.Errorf("ArtifactKind %q should be valid, got finding: %s", kind, f.Message)
				}
			}
		})
	}
}
