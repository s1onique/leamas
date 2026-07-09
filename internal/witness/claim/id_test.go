package claim

import (
	"errors"
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
// Note: prefix is checked first, so inputs without "claim-" prefix return ErrClaimIDNoPrefix.
func TestValidateClaimIDRejectsUnsafeIDs(t *testing.T) {
	unsafeIDs := []struct {
		id       string
		expected error
	}{
		{"", ErrEmptyID},
		{".", ErrClaimIDNoPrefix},                 // checked before reserved names
		{"..", ErrClaimIDNoPrefix},                // checked before reserved names
		{"../escape", ErrClaimIDNoPrefix},         // checked before traversal
		{"/absolute", ErrClaimIDNoPrefix},         // checked before not-local
		{"claim/bad", ErrClaimIDNoPrefix},         // checked before not-local
		{"claim with spaces", ErrClaimIDNoPrefix}, // checked before invalid char
		{"claim:bad", ErrClaimIDNoPrefix},         // checked before invalid char
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
		if !errors.Is(err, tc.expected) {
			t.Errorf("ValidateClaimID(%q) error = %v, want %v", tc.id, err, tc.expected)
		}
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
// Note: prefix is checked first, so inputs without "evidence-" prefix return ErrEvidenceIDNoPrefix.
func TestValidateEvidenceIDRejectsUnsafeIDs(t *testing.T) {
	unsafeIDs := []struct {
		id       string
		expected error
	}{
		{"", ErrEmptyID},
		{".", ErrEvidenceIDNoPrefix},                    // checked before reserved names
		{"..", ErrEvidenceIDNoPrefix},                   // checked before reserved names
		{"../escape", ErrEvidenceIDNoPrefix},            // checked before traversal
		{"/absolute", ErrEvidenceIDNoPrefix},            // checked before not-local
		{"evidence/bad", ErrEvidenceIDNoPrefix},         // checked before not-local
		{"evidence with spaces", ErrEvidenceIDNoPrefix}, // checked before invalid char
		{"evidence:bad", ErrEvidenceIDNoPrefix},         // checked before invalid char
		{"bad-prefix", ErrEvidenceIDNoPrefix},
		{"EVIDENCE-UPPERCASE", ErrEvidenceIDNoPrefix},
		{"evidence-", ErrIDMissingSuffix}, // missing suffix after prefix
	}

	// Generate a too-long ID (more than 128 chars)
	longID := "evidence-"
	for i := 0; i < 130; i++ {
		longID += "x"
	}
	unsafeIDs = append(unsafeIDs, struct {
		id       string
		expected error
	}{longID, ErrIDTooLong})

	for _, tc := range unsafeIDs {
		err := ValidateEvidenceID(EvidenceID(tc.id))
		if err == nil {
			t.Errorf("ValidateEvidenceID(%q) should have returned error, got nil", tc.id)
			continue
		}
		if !errors.Is(err, tc.expected) {
			t.Errorf("ValidateEvidenceID(%q) error = %v, want %v", tc.id, err, tc.expected)
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
			continue
		}
		if !errors.Is(err, tc.expected) {
			t.Errorf("ValidateRelativePath(%q) error = %v, want %v", tc.path, err, tc.expected)
		}
	}
}
