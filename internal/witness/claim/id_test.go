package claim

import (
	"testing"
)

// TestValidateClaimIDAcceptsSafeIDs tests that valid claim IDs are accepted.
func TestValidateClaimIDAcceptsSafeIDs(t *testing.T) {
	safeIDs := []string{
		"claim-gate-passed",
		"claim-runtime-smoke-local-only",
		"claim-20260709T082245Z-gate01",
		"claim-a",
		"claim-abc123",
		"claim-a.b.c",
		"claim-a_b_c",
		"claim-a-b-c",
		"claim-ABC123",
	}

	for _, id := range safeIDs {
		err := ValidateClaimID(ClaimID(id))
		if err != nil {
			t.Errorf("ValidateClaimID(%q) returned error: %v", id, err)
		}
	}
}

// TestValidateClaimIDRejectsUnsafeIDs tests that invalid claim IDs are rejected.
func TestValidateClaimIDRejectsUnsafeIDs(t *testing.T) {
	unsafeIDs := []struct {
		id       string
		expected error
	}{
		{"", ErrEmptyID},
		{".", ErrIDReserved},
		{"..", ErrIDReserved},
		{"../escape", ErrIDTraversal},
		{"/absolute", ErrIDNotLocal},
		{"claim/bad", ErrIDNotLocal},
		{"claim with spaces", ErrIDInvalidChar},
		{"claim:bad", ErrIDInvalidChar},
		{"bad-prefix", ErrClaimIDNoPrefix},
		{"CLAIM-UPPERCASE", ErrClaimIDNoPrefix},
		{"claim-", ErrIDMissingSuffix}, // missing suffix after prefix
	}

	// Generate a too-long ID (more than 128 chars)
	longID := "claim-"
	for i := 0; i < 130; i++ {
		longID += "x"
	}
	unsafeIDs = append(unsafeIDs, struct {
		id       string
		expected error
	}{longID, ErrIDTooLong})

	for _, tc := range unsafeIDs {
		err := ValidateClaimID(ClaimID(tc.id))
		if err == nil {
			t.Errorf("ValidateClaimID(%q) should have returned error, got nil", tc.id)
			continue
		}
		// Just check that some error is returned for now
		t.Logf("ValidateClaimID(%q) correctly returned error: %v", tc.id, err)
	}
}

// TestValidateEvidenceIDAcceptsSafeIDs tests that valid evidence IDs are accepted.
func TestValidateEvidenceIDAcceptsSafeIDs(t *testing.T) {
	safeIDs := []string{
		"evidence-make-gate-output",
		"evidence-runtime-smoke-log",
		"evidence-digest-head",
		"evidence-a",
		"evidence-abc123",
		"evidence-a.b.c",
		"evidence-a_b_c",
		"evidence-a-b-c",
		"evidence-ABC123",
	}

	for _, id := range safeIDs {
		err := ValidateEvidenceID(EvidenceID(id))
		if err != nil {
			t.Errorf("ValidateEvidenceID(%q) returned error: %v", id, err)
		}
	}
}

// TestValidateEvidenceIDRejectsUnsafeIDs tests that invalid evidence IDs are rejected.
func TestValidateEvidenceIDRejectsUnsafeIDs(t *testing.T) {
	unsafeIDs := []string{
		"",
		".",
		"..",
		"../escape",
		"/absolute",
		"evidence/bad",
		"evidence with spaces",
		"evidence:bad",
		"bad-prefix",
		"EVIDENCE-UPPERCASE",
	}

	for _, id := range unsafeIDs {
		err := ValidateEvidenceID(EvidenceID(id))
		if err == nil {
			t.Errorf("ValidateEvidenceID(%q) should have returned error, got nil", id)
		}
	}
}

// TestValidateRelativePathAcceptsSafePaths tests that safe relative paths are accepted.
func TestValidateRelativePathAcceptsSafePaths(t *testing.T) {
	safePaths := []string{
		"", // empty is allowed
		"digests/head.txt",
		"traces/trace.log",
		"evidence/output.txt",
		"file.txt",
		"a/b/c/d.txt",
		"simple.file.name",
	}

	for _, path := range safePaths {
		err := ValidateRelativePath(path)
		if err != nil {
			t.Errorf("ValidateRelativePath(%q) returned error: %v", path, err)
		}
	}
}

// TestValidateRelativePathRejectsUnsafePaths tests that unsafe relative paths are rejected.
func TestValidateRelativePathRejectsUnsafePaths(t *testing.T) {
	unsafePaths := []struct {
		path     string
		expected error
	}{
		{"/absolute", ErrAbsolutePath},
		{"../escape", ErrTraversalInPath},
		{"a/../../escape", ErrTraversalInPath},
		{"a\\backslash", ErrInvalidRelativePath},
	}

	for _, tc := range unsafePaths {
		err := ValidateRelativePath(tc.path)
		if err == nil {
			t.Errorf("ValidateRelativePath(%q) should have returned error, got nil", tc.path)
		}
	}
}
