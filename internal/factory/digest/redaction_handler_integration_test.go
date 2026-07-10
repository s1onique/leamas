// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"strings"
	"testing"
)

// TestRedactDigestWithPolicy_PolicyMetadataVisible verifies that per-file
// REDACTION_POLICY metadata is visible in the output for both source and non-source files.
func TestRedactDigestWithPolicy_PolicyMetadataVisible(t *testing.T) {
	// Digest with both source and non-source files
	digestContent := `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 2
LEAMAS_VERSION: dev

=== tests/test.py ===
Metadata: tracked, staged present: yes, unstaged present: no

password = "secret123"
token = "abc123"

=== config/app.log ===
Metadata: tracked, staged present: yes, unstaged present: no

password=secret123
token=abc123`

	result, warnings := RedactDigestWithPolicy(digestContent)

	// Verify source file has policy metadata
	if !strings.Contains(result, "=== tests/test.py ===") {
		t.Error("source file marker should be preserved")
	}

	// Source file should have REDACTION_POLICY metadata with class=source
	if !strings.Contains(result, "REDACTION_POLICY:") {
		t.Error("source file should have REDACTION_POLICY metadata")
	}
	if !strings.Contains(result, "class=source") {
		t.Error("source file should have class=source")
	}
	if !strings.Contains(result, "decision=preserve_and_warn") {
		t.Error("source file should have decision=preserve_and_warn")
	}
	if !strings.Contains(result, "redaction_applied=false") {
		t.Error("source file should have redaction_applied=false")
	}

	// Non-source file should also have policy metadata
	if !strings.Contains(result, "=== config/app.log ===") {
		t.Error("non-source file marker should be preserved")
	}
	if !strings.Contains(result, "class=non_source") {
		t.Error("non-source file should have class=non_source")
	}
	if !strings.Contains(result, "decision=redact") {
		t.Error("non-source file should have decision=redact")
	}
	if !strings.Contains(result, "redaction_applied=true") {
		t.Error("non-source file should have redaction_applied=true")
	}

	// Source content should be preserved
	if !strings.Contains(result, `password = "secret123"`) {
		t.Error("source file content should be preserved")
	}

	// Non-source content should be redacted
	if !strings.Contains(result, "[REDACTED]") {
		t.Error("non-source content should be redacted")
	}

	// Warnings should be generated for source file
	if len(warnings) == 0 {
		t.Error("expected warnings for source file")
	}

	// SOURCE_SECRET_WARNINGS should be present when secrets are detected
	if !strings.Contains(result, "SOURCE_SECRET_WARNINGS:") {
		t.Error("source file with secrets should have SOURCE_SECRET_WARNINGS section")
	}
}

// TestRedactDigestWithPolicy_NoMalformedRedaction verifies that source files
// never contain malformed redaction patterns. Source files are preserved byte-faithfully.
func TestRedactDigestWithPolicy_NoMalformedRedaction(t *testing.T) {
	// Source file with patterns that look like they should be redacted
	// These must be PRESERVED, not mutated to malformed patterns
	pythonDigest := `=== tests/test.py ===
Metadata: tracked, staged present: yes, unstaged present: no

def process(s):
    pass

process("password=secret123")
process("secret=mysecret")
process("token=abc123xyz")
process("api_key=abc123xyz")`

	result, _ := RedactDigestWithPolicy(pythonDigest)

	// Source content must be preserved as-is
	if !strings.Contains(result, `"password=secret123"`) {
		t.Error("original password pattern should be preserved")
	}

	// Malformed patterns should NEVER appear in source files
	malformedPatterns := []string{
		"password=[REDACTED])",
		"secret=[REDACTED])",
		"token=[REDACTED])",
		"api_key=[REDACTED])",
	}
	for _, pattern := range malformedPatterns {
		if strings.Contains(result, pattern) {
			t.Errorf("malformed pattern %q should NEVER appear in source file", pattern)
		}
	}
}

// TestRedactDigestWithPolicy_EnvFile verifies .env files are redacted.
func TestRedactDigestWithPolicy_EnvFile(t *testing.T) {
	envContent := `=== .env ===
Metadata: tracked, staged present: yes, unstaged present: no

PASSWORD=secret123
API_KEY=abc123xyz
TOKEN=abc123
SOME_VAR=normal_value`

	result, _ := RedactDigestWithPolicy(envContent)

	if !strings.Contains(result, "[REDACTED]") {
		t.Error(".env file should be redacted")
	}
	if strings.Contains(result, "secret123") {
		t.Error("password value should be redacted in .env")
	}
}

// TestRedactDigestWithPolicy_JSONConfig verifies JSON configs are redacted.
func TestRedactDigestWithPolicy_JSONConfig(t *testing.T) {
	jsonContent := `=== config.json ===
Metadata: tracked, staged present: yes, unstaged present: no

{
  "api_key": "sk-1234567890abcdefghijklmnopqrstuv",
  "password": "superSecretPassword123",
  "github_token": "ghp_1234567890abcdefghijklmnopqrstuvwxyzAB"
}`

	result, _ := RedactDigestWithPolicy(jsonContent)

	if !strings.Contains(result, "[REDACTED]") {
		t.Error("JSON config should be redacted")
	}
	if strings.Contains(result, "sk-1234567890") {
		t.Error("OpenAI key should be redacted in JSON")
	}
}

// TestRedactDigestWithPolicy_GoSource verifies Go source files are preserved.
func TestRedactDigestWithPolicy_GoSource(t *testing.T) {
	goContent := `=== auth.go ===
Metadata: tracked, staged present: yes, unstaged present: no

func authenticate(password string) error {
    return nil
}

apiKey := "sk-test123456789"
token := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"`

	result, warnings := RedactDigestWithPolicy(goContent)

	if !strings.Contains(result, `"sk-test123456789"`) {
		t.Error("Go source should preserve api key literal")
	}
	if !strings.Contains(result, "password string") {
		t.Error("Go source should preserve password parameter")
	}
	if len(warnings) == 0 {
		t.Error("expected source secret warnings for Go file")
	}
}

// TestSimpleRedactDigest verifies simple redaction applies to everything.
func TestSimpleRedactDigest(t *testing.T) {
	digest := `password=secret123
token=Bearer abc123def456
api_key=sk-1234567890abcdefghijklmnopqrstuv`

	result := SimpleRedactDigest(digest)

	if !strings.Contains(result, "[REDACTED]") {
		t.Error("simple redaction should apply to all content")
	}
	if strings.Contains(result, "secret123") {
		t.Error("simple redaction should redact password value")
	}
}

// TestRedactDigestWithPolicy_MetadataOrder verifies REDACTION_POLICY appears before content.
func TestRedactDigestWithPolicy_MetadataOrder(t *testing.T) {
	// Plain generated digest fixture (before policy decoration)
	plainDigest := `=== internal/factory/digest/redaction_handler.go ===
Metadata: untracked, staged present: no, unstaged present: yes

--- untracked file content ---
// Package digest provides targeted digest generation for Git repositories.
package digest

const SourcePatternBearerToken = "source.bearer_token"`

	result, _ := RedactDigestWithPolicy(plainDigest)

	// Find positions of key markers
	fileMarkerPos := strings.Index(result, "=== internal/factory/digest/redaction_handler.go ===")
	metadataPos := strings.Index(result, "Metadata: untracked")
	redactionPolicyPos := strings.Index(result, "REDACTION_POLICY:")
	untrackedContentPos := strings.Index(result, "--- untracked file content ---")
	sourceStartPos := strings.Index(result, "// Package digest")

	// Verify ordering: file marker -> Metadata -> REDACTION_POLICY -> content marker -> source
	if fileMarkerPos == -1 {
		t.Error("file marker should be present")
	}
	if metadataPos == -1 {
		t.Error("Metadata line should be present")
	}
	if redactionPolicyPos == -1 {
		t.Error("REDACTION_POLICY should be present")
	}
	if untrackedContentPos == -1 {
		t.Error("untracked file content marker should be present")
	}
	if sourceStartPos == -1 {
		t.Error("source content should begin with real source line")
	}

	// Verify REDACTION_POLICY comes before content marker
	if redactionPolicyPos > untrackedContentPos {
		t.Error("REDACTION_POLICY should appear BEFORE content marker")
	}

	// Verify content marker comes before actual source
	if untrackedContentPos > sourceStartPos {
		t.Error("content marker should appear BEFORE actual source content")
	}

	// Verify no Metadata after content marker
	metadataAfterContent := strings.Index(result[untrackedContentPos:], "Metadata:")
	if metadataAfterContent != -1 {
		t.Error("Metadata line should NOT appear after content marker")
	}
}

// TestRedactDigestWithPolicy_TrackedFileMetadataOrder verifies ordering for tracked files.
func TestRedactDigestWithPolicy_TrackedFileMetadataOrder(t *testing.T) {
	plainDigest := `=== internal/factory/digest/digest.go ===
Metadata: tracked, staged present: yes, unstaged present: no

--- staged diff ---
diff --git a/internal/factory/digest/digest.go b/internal/factory/digest/digest.go
package digest

func main() {}`

	result, _ := RedactDigestWithPolicy(plainDigest)

	// Find positions
	fileMarkerPos := strings.Index(result, "=== internal/factory/digest/digest.go ===")
	metadataPos := strings.Index(result, "Metadata: tracked")
	redactionPolicyPos := strings.Index(result, "REDACTION_POLICY:")
	diffStartPos := strings.Index(result, "--- staged diff ---")
	sourceStartPos := strings.Index(result, "package digest")

	// Verify all markers are present
	if fileMarkerPos == -1 {
		t.Error("file marker should be present")
	}
	if metadataPos == -1 {
		t.Error("Metadata should be present")
	}
	if redactionPolicyPos == -1 {
		t.Error("REDACTION_POLICY should be present")
	}
	if diffStartPos == -1 {
		t.Error("diff marker should be present")
	}
	if sourceStartPos == -1 {
		t.Error("source content should be present")
	}

	// Verify ordering: file marker -> metadata -> REDACTION_POLICY -> diff -> source
	if fileMarkerPos > metadataPos {
		t.Error("file marker should appear BEFORE metadata")
	}
	if metadataPos > redactionPolicyPos {
		t.Error("metadata should appear BEFORE REDACTION_POLICY")
	}
	if redactionPolicyPos > diffStartPos {
		t.Error("REDACTION_POLICY should appear BEFORE diff marker")
	}
	if diffStartPos > sourceStartPos {
		t.Error("diff marker should appear BEFORE source content")
	}
}

// TestRedactDigestWithPolicy_RedactionPolicyOnce verifies policy appears exactly once per file.
func TestRedactDigestWithPolicy_RedactionPolicyOnce(t *testing.T) {
	digest := `=== test.go ===
Metadata: tracked, staged present: yes, unstaged present: no

--- staged diff ---
package main

=== test.py ===
Metadata: tracked, staged present: yes, unstaged present: no

--- staged diff ---
print("hello")`

	result, _ := RedactDigestWithPolicy(digest)

	// Count REDACTION_POLICY occurrences
	count := strings.Count(result, "REDACTION_POLICY:")
	if count != 2 {
		t.Errorf("expected REDACTION_POLICY to appear exactly 2 times, got %d", count)
	}
}
