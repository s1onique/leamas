package runbundle

import (
	"strings"
	"testing"
)

func TestValidMinimalBundlePasses(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleDraft,
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
	}
	result := Validate(bundle)
	if !result.OK() {
		t.Errorf("Valid minimal bundle should pass, got findings: %v", result.Findings)
	}
}

func TestMissingBundleIDFails(t *testing.T) {
	bundle := RunBundle{
		RunID:     RunID("run-001"),
		Status:    RunBundleDraft,
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Missing bundle ID should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "RunBundle.ID" && strings.Contains(f.Message, "required") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing ID, got: %v", result.Findings)
	}
}

func TestMissingRunIDFails(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		Status:    RunBundleDraft,
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Missing run ID should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "RunBundle.RunID" && strings.Contains(f.Message, "required") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing RunID, got: %v", result.Findings)
	}
}

func TestMissingCreatedAtFails(t *testing.T) {
	bundle := RunBundle{
		ID:      RunBundleID("bundle-001"),
		RunID:   RunID("run-001"),
		Status:  RunBundleDraft,
		Summary: "Test bundle",
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Missing createdAt should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "RunBundle.CreatedAt" && strings.Contains(f.Message, "required") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing CreatedAt, got: %v", result.Findings)
	}
}

func TestMissingSummaryFails(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleDraft,
		CreatedAt: "2024-01-15T10:00:00Z",
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Missing summary should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "RunBundle.Summary" && strings.Contains(f.Message, "required") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing Summary, got: %v", result.Findings)
	}
}

func TestInvalidStatusFails(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleStatus("invalid_status"),
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Invalid status should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "RunBundle.Status" && strings.Contains(f.Message, "must be one of") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about invalid status, got: %v", result.Findings)
	}
}

func TestDuplicateArtifactIDsFail(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleDraft,
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
		Artifacts: []ArtifactRef{
			{ID: ArtifactID("artifact-1"), Kind: ArtifactLog, Path: "/path/1"},
			{ID: ArtifactID("artifact-2"), Kind: ArtifactLog, Path: "/path/2"},
			{ID: ArtifactID("artifact-1"), Kind: ArtifactDigest, Path: "/path/3"},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Duplicate artifact IDs should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "ArtifactRef.ID" && strings.Contains(f.Message, "Duplicate artifact ID") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about duplicate artifact ID, got: %v", result.Findings)
	}
}

func TestEmptyArtifactIDFails(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleDraft,
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
		Artifacts: []ArtifactRef{
			{ID: ArtifactID(""), Kind: ArtifactLog, Path: "/path/1"},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Empty artifact ID should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "ArtifactRef.ID" && strings.Contains(f.Message, "non-empty") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about empty artifact ID, got: %v", result.Findings)
	}
}

func TestInvalidArtifactKindFails(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleDraft,
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
		Artifacts: []ArtifactRef{
			{ID: ArtifactID("artifact-1"), Kind: ArtifactKind("invalid_kind"), Path: "/path/1"},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Invalid artifact kind should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "ArtifactRef.Kind" && strings.Contains(f.Message, "Invalid artifact kind") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about invalid artifact kind, got: %v", result.Findings)
	}
}

func TestEmptyArtifactPathFails(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleDraft,
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
		Artifacts: []ArtifactRef{
			{ID: ArtifactID("artifact-1"), Kind: ArtifactLog, Path: ""},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Empty artifact path should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "ArtifactRef.Path" && strings.Contains(f.Message, "non-empty") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about empty artifact path, got: %v", result.Findings)
	}
}

func TestDuplicateEvidenceIDsFail(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleDraft,
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
		Artifacts: []ArtifactRef{
			{ID: ArtifactID("artifact-1"), Kind: ArtifactLog, Path: "/path/1"},
		},
		Evidence: []EvidenceRef{
			{ID: EvidenceID("evidence-1"), ArtifactID: ArtifactID("artifact-1"), Summary: "Evidence 1"},
			{ID: EvidenceID("evidence-2"), ArtifactID: ArtifactID("artifact-1"), Summary: "Evidence 2"},
			{ID: EvidenceID("evidence-1"), ArtifactID: ArtifactID("artifact-1"), Summary: "Evidence 3"},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Duplicate evidence IDs should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "EvidenceRef.ID" && strings.Contains(f.Message, "Duplicate evidence ID") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about duplicate evidence ID, got: %v", result.Findings)
	}
}

func TestEvidenceReferencingMissingArtifactFails(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleDraft,
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
		Artifacts: []ArtifactRef{
			{ID: ArtifactID("artifact-1"), Kind: ArtifactLog, Path: "/path/1"},
		},
		Evidence: []EvidenceRef{
			{ID: EvidenceID("evidence-1"), ArtifactID: ArtifactID("nonexistent"), Summary: "Evidence 1"},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Evidence referencing missing artifact should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "EvidenceRef.ArtifactID" && strings.Contains(f.Message, "non-existent artifact") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about missing artifact reference, got: %v", result.Findings)
	}
}

func TestDuplicateClaimIDsFail(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleDraft,
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
		Claims: []ClaimRef{
			{ID: ClaimID("claim-1"), Summary: "Claim 1"},
			{ID: ClaimID("claim-2"), Summary: "Claim 2"},
			{ID: ClaimID("claim-1"), Summary: "Claim 3"},
		},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Duplicate claim IDs should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "ClaimRef.ID" && strings.Contains(f.Message, "Duplicate claim ID") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about duplicate claim ID, got: %v", result.Findings)
	}
}

func TestEmptyLimitationFails(t *testing.T) {
	bundle := RunBundle{
		ID:          RunBundleID("bundle-001"),
		RunID:       RunID("run-001"),
		Status:      RunBundleDraft,
		Summary:     "Test bundle",
		CreatedAt:   "2024-01-15T10:00:00Z",
		Limitations: []string{"Limitation 1", "", "Limitation 3"},
	}
	result := Validate(bundle)
	if result.OK() {
		t.Error("Empty limitation should fail validation")
	}
	found := false
	for _, f := range result.Findings {
		if f.Path == "RunBundle.Limitations" && strings.Contains(f.Message, "empty strings") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected finding about empty limitation, got: %v", result.Findings)
	}
}

func TestFindingsAreDeterministic(t *testing.T) {
	bundle := RunBundle{
		ID:        RunBundleID("bundle-001"),
		RunID:     RunID("run-001"),
		Status:    RunBundleStatus("invalid"),
		Summary:   "Test bundle",
		CreatedAt: "2024-01-15T10:00:00Z",
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
