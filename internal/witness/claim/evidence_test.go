package claim

import (
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// TestNewEvidenceDefaults tests that NewEvidence sets correct defaults.
func TestNewEvidenceDefaults(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	evidence, err := NewEvidence(
		"evidence-test-001",
		runID,
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"Test evidence title",
		now,
	)
	if err != nil {
		t.Fatalf("NewEvidence failed: %v", err)
	}

	// Check schema version
	if evidence.SchemaVersion != EvidenceSchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", evidence.SchemaVersion, EvidenceSchemaVersion)
	}

	// Check ID
	if evidence.ID != "evidence-test-001" {
		t.Errorf("ID = %q, want %q", evidence.ID, "evidence-test-001")
	}

	// Check RunID
	if evidence.RunID != runID {
		t.Errorf("RunID = %v, want %v", evidence.RunID, runID)
	}

	// Check Kind
	if evidence.Kind != EvidenceKindCommandOutput {
		t.Errorf("Kind = %q, want %q", evidence.Kind, EvidenceKindCommandOutput)
	}

	// Check Role
	if evidence.Role != EvidenceRolePrimary {
		t.Errorf("Role = %q, want %q", evidence.Role, EvidenceRolePrimary)
	}

	// Check Title
	if evidence.Title != "Test evidence title" {
		t.Errorf("Title = %q, want %q", evidence.Title, "Test evidence title")
	}

	// Check metadata is non-nil empty map
	if evidence.Metadata == nil {
		t.Error("Metadata is nil, want non-nil empty map")
	}

	// Check created_at
	if !evidence.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", evidence.CreatedAt, now)
	}
}

// TestValidateEvidenceRejectsBadKind tests that invalid kind fails.
func TestValidateEvidenceRejectsBadKind(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	evidence, err := NewEvidence(
		"evidence-test-001",
		runID,
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"Test title",
		now,
	)
	if err != nil {
		t.Fatalf("NewEvidence failed: %v", err)
	}

	// Set invalid kind
	evidence.Kind = EvidenceKind("invalid")
	err = evidence.Validate()
	if err == nil {
		t.Error("Validate with invalid kind should fail")
	}
}

// TestValidateEvidenceRejectsBadRole tests that invalid role fails.
func TestValidateEvidenceRejectsBadRole(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	evidence, err := NewEvidence(
		"evidence-test-001",
		runID,
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"Test title",
		now,
	)
	if err != nil {
		t.Fatalf("NewEvidence failed: %v", err)
	}

	// Set invalid role
	evidence.Role = EvidenceRole("invalid")
	err = evidence.Validate()
	if err == nil {
		t.Error("Validate with invalid role should fail")
	}
}

// TestValidateEvidenceRejectsEmptyTitle tests that empty title fails.
func TestValidateEvidenceRejectsEmptyTitle(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	_, err := NewEvidence(
		"evidence-test-001",
		runID,
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"",
		now,
	)
	if err == nil {
		t.Error("NewEvidence with empty title should fail")
	}
	if err != ErrEmptyTitle {
		t.Errorf("Error = %v, want %v", err, ErrEmptyTitle)
	}
}

// TestValidateEvidenceAcceptsSafeRelativePath tests that safe relative paths are accepted.
func TestValidateEvidenceAcceptsSafeRelativePath(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	safePaths := []string{
		"digests/head.txt",
		"traces/trace.log",
		"evidence/output.txt",
		"file.txt",
		"a/b/c/d.txt",
	}

	for _, path := range safePaths {
		evidence, err := NewEvidence(
			"evidence-test-001",
			runID,
			EvidenceKindCommandOutput,
			EvidenceRolePrimary,
			"Test title",
			now,
		)
		if err != nil {
			t.Fatalf("NewEvidence failed: %v", err)
		}
		evidence.RelativePath = path
		err = evidence.Validate()
		if err != nil {
			t.Errorf("Validate with relative path %q failed: %v", path, err)
		}
	}
}

// TestValidateEvidenceRejectsTraversalRelativePath tests that traversal paths are rejected.
func TestValidateEvidenceRejectsTraversalRelativePath(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	traversalPaths := []string{
		"../escape",
		"a/../../escape",
		"a/b/../../c",
	}

	for _, path := range traversalPaths {
		evidence, err := NewEvidence(
			"evidence-test-001",
			runID,
			EvidenceKindCommandOutput,
			EvidenceRolePrimary,
			"Test title",
			now,
		)
		if err != nil {
			t.Fatalf("NewEvidence failed: %v", err)
		}
		evidence.RelativePath = path
		err = evidence.Validate()
		if err == nil {
			t.Errorf("Validate with traversal path %q should fail", path)
		}
	}
}

// TestEvidenceStrictDecodeRejectsUnknownFields tests that unknown fields are rejected.
func TestEvidenceStrictDecodeRejectsUnknownFields(t *testing.T) {
	jsonWithUnknownField := `{
		"schema_version": "leamas.evidence.v1",
		"id": "evidence-test-001",
		"run_id": "run-20260709T071704Z-smoke01",
		"created_at": "2026-07-09T07:17:04Z",
		"kind": "command_output",
		"role": "primary",
		"title": "Test title",
		"extra_field": "should be rejected"
	}`

	_, err := StrictDecodeEvidence([]byte(jsonWithUnknownField))
	if err == nil {
		t.Error("StrictDecodeEvidence should reject unknown fields")
	}
}

// TestEvidenceRoundTripJSON tests that evidence can be marshaled and unmarshaled.
func TestEvidenceRoundTripJSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	original, err := NewEvidence(
		"evidence-test-001",
		runID,
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"Test title",
		now,
	)
	if err != nil {
		t.Fatalf("NewEvidence failed: %v", err)
	}
	original.RelativePath = "digests/head.txt"
	original.Summary = "Test summary"
	original.Metadata["key1"] = "value1"

	// Marshal
	data, err := MarshalEvidenceJSON(original)
	if err != nil {
		t.Fatalf("MarshalEvidenceJSON failed: %v", err)
	}

	// Unmarshal
	decoded, err := StrictDecodeEvidence(data)
	if err != nil {
		t.Fatalf("StrictDecodeEvidence failed: %v", err)
	}

	// Validate decoded evidence
	if err := decoded.Validate(); err != nil {
		t.Fatalf("Decoded evidence validation failed: %v", err)
	}

	// Verify fields match
	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %v, want %v", decoded.ID, original.ID)
	}
	if decoded.Kind != original.Kind {
		t.Errorf("Kind mismatch: got %v, want %v", decoded.Kind, original.Kind)
	}
	if decoded.Role != original.Role {
		t.Errorf("Role mismatch: got %v, want %v", decoded.Role, original.Role)
	}
	if decoded.RelativePath != original.RelativePath {
		t.Errorf("RelativePath mismatch: got %v, want %v", decoded.RelativePath, original.RelativePath)
	}
}

// TestIsValidEvidenceKind tests kind validation helper.
func TestIsValidEvidenceKind(t *testing.T) {
	validKinds := []EvidenceKind{
		EvidenceKindCommandOutput,
		EvidenceKindDigest,
		EvidenceKindLog,
		EvidenceKindFile,
		EvidenceKindTrace,
		EvidenceKindVerifierResult,
	}

	for _, k := range validKinds {
		if !IsValidEvidenceKind(k) {
			t.Errorf("IsValidEvidenceKind(%q) = false, want true", k)
		}
	}

	if IsValidEvidenceKind("invalid") {
		t.Error("IsValidEvidenceKind(\"invalid\") = true, want false")
	}
}

// TestIsValidEvidenceRole tests role validation helper.
func TestIsValidEvidenceRole(t *testing.T) {
	validRoles := []EvidenceRole{
		EvidenceRolePrimary,
		EvidenceRoleSupporting,
		EvidenceRoleContradicting,
		EvidenceRoleContext,
	}

	for _, r := range validRoles {
		if !IsValidEvidenceRole(r) {
			t.Errorf("IsValidEvidenceRole(%q) = false, want true", r)
		}
	}

	if IsValidEvidenceRole("invalid") {
		t.Error("IsValidEvidenceRole(\"invalid\") = true, want false")
	}
}

// TestEvidenceValidationErrorMessages tests that error messages are descriptive.
func TestEvidenceValidationErrorMessages(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	// Test empty title error
	_, err := NewEvidence(
		"evidence-test-001",
		runID,
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"",
		now,
	)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Errorf("Error should mention 'title', got: %v", err)
	}

	// Test invalid evidence ID error
	_, err = NewEvidence(
		"invalid-id",
		runID,
		EvidenceKindCommandOutput,
		EvidenceRolePrimary,
		"Test title",
		now,
	)
	if err == nil || !strings.Contains(err.Error(), "evidence") {
		t.Errorf("Error should mention 'evidence', got: %v", err)
	}
}

// TestEvidenceAllKindsAndRoles tests all valid kinds and roles combinations.
func TestEvidenceAllKindsAndRoles(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	runID := runbundle.RunID("run-20260709T071704Z-smoke01")

	kinds := []EvidenceKind{
		EvidenceKindCommandOutput,
		EvidenceKindDigest,
		EvidenceKindLog,
		EvidenceKindFile,
		EvidenceKindTrace,
		EvidenceKindVerifierResult,
	}

	roles := []EvidenceRole{
		EvidenceRolePrimary,
		EvidenceRoleSupporting,
		EvidenceRoleContradicting,
		EvidenceRoleContext,
	}

	for i, kind := range kinds {
		for j, role := range roles {
			id := EvidenceID("evidence-" + string(rune('a'+i)) + string(rune('0'+j)))
			evidence, err := NewEvidence(
				id,
				runID,
				kind,
				role,
				"Test title",
				now,
			)
			if err != nil {
				t.Errorf("NewEvidence(%q, %q, %q) failed: %v", id, kind, role, err)
			}
			if err := evidence.Validate(); err != nil {
				t.Errorf("Validate() for kind=%q, role=%q failed: %v", kind, role, err)
			}
		}
	}
}
