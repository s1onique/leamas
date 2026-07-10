// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"strings"
	"testing"
)

func TestRedactDigestWithPolicy_SourcePreserved(t *testing.T) {
	// This is the critical regression test: Python source should NOT be modified
	pythonContent := `=== tests/test_llm_safe_evidence_boundary.py ===
Metadata: tracked, staged present: yes, unstaged present: no

--- staged diff ---
diff --git a/tests/test_llm_safe_evidence_boundary.py b/tests/test_llm_safe_evidence_boundary.py
--- staged diff ---
def make_redacted_evidence_text(s):
    pass

make_redacted_evidence_text("password=secret123")
make_redacted_evidence_text("secret=mysecret")
make_redacted_evidence_text("token=abc123xyz")
make_redacted_evidence_text("api_key=abc123xyz")
make_redacted_evidence_text("Authorization: Bearer aaaa.bbbb.cccc")`

	result, warnings := RedactDigestWithPolicy(pythonContent)

	// Source content should be preserved exactly
	if !strings.Contains(result, `"password=secret123"`) {
		t.Error("Python source with password literal should be preserved")
	}
	if !strings.Contains(result, `"secret=mysecret"`) {
		t.Error("Python source with secret literal should be preserved")
	}
	if !strings.Contains(result, `"token=abc123xyz"`) {
		t.Error("Python source with token literal should be preserved")
	}
	if !strings.Contains(result, `"api_key=abc123xyz"`) {
		t.Error("Python source with api_key literal should be preserved")
	}
	if !strings.Contains(result, "Bearer aaaa.bbbb.cccc") {
		t.Error("Python source with Bearer token should be preserved")
	}

	// Should have warnings for source secrets
	if len(warnings) == 0 {
		t.Error("expected source secret warnings for Python file")
	}

	// Should NOT contain [REDACTED] in source content
	if strings.Contains(result, "password=[REDACTED]") {
		t.Error("Python source should NOT contain redaction markers")
	}
}

func TestRedactDigestWithPolicy_NonSourceRedacted(t *testing.T) {
	// Non-source files should still be redacted
	logContent := `=== app.log ===
Metadata: tracked, staged present: yes, unstaged present: no

--- staged diff ---
password=secret123
secret=mysecret
token=abc123xyz
api_key=abc123xyz
Authorization: Bearer aaaa.bbbb.cccc`

	result, warnings := RedactDigestWithPolicy(logContent)

	// Non-source content should be redacted
	if !strings.Contains(result, "[REDACTED]") {
		t.Error("non-source content should be redacted")
	}

	// Should not contain raw secrets
	if strings.Contains(result, "password=secret123") {
		t.Error("password should be redacted in log file")
	}
	if strings.Contains(result, "secret=mysecret") {
		t.Error("secret should be redacted in log file")
	}

	// Warnings should be empty for non-source files (redaction is applied, not warn)
	if len(warnings) > 0 {
		t.Error("non-source files should not produce source secret warnings")
	}
}

func TestRedactDigestWithPolicy_MalformedPythonPattern(t *testing.T) {
	// The specific bug pattern: malformed Python like "password=[REDACTED])"
	// should never appear in output
	buggyContent := `=== tests/test.py ===
Metadata: tracked, staged present: yes, unstaged present: no

--- staged diff ---
make_redacted_evidence_text("password=secret123")
make_redacted_evidence_text("secret=mysecret")
make_redacted_evidence_text("token=abc123xyz")
make_redacted_evidence_text("api_key=abc123xyz")`

	result, _ := RedactDigestWithPolicy(buggyContent)

	// The malformed patterns should NOT appear
	badPatterns := []string{
		"password=[REDACTED])",
		"secret=[REDACTED])",
		"token=[REDACTED])",
		"api_key=[REDACTED])",
	}

	for _, pattern := range badPatterns {
		if strings.Contains(result, pattern) {
			t.Errorf("malformed Python pattern %q should NOT appear in output", pattern)
		}
	}

	// Original source should be preserved
	if !strings.Contains(result, `"password=secret123"`) {
		t.Error("original Python source should be preserved")
	}
}

func TestRenderRedactionPolicyMetadata(t *testing.T) {
	meta := DefaultPolicyMetadata()

	output := RenderRedactionPolicyMetadata(meta)

	if !strings.Contains(output, "REDACTION_POLICY:") {
		t.Error("expected REDACTION_POLICY header")
	}
	if !strings.Contains(output, "source_redaction=warn_only") {
		t.Error("expected source_redaction=warn_only")
	}
	if !strings.Contains(output, "source_secret_scan=warn_only") {
		t.Error("expected source_secret_scan=warn_only")
	}
	if !strings.Contains(output, "non_source_redaction=redact") {
		t.Error("expected non_source_redaction=redact")
	}
}

func TestRenderFileRedactionMetadata_Source(t *testing.T) {
	meta := FileRedactionMetadata{
		Path:             "test.py",
		Class:            RedactionClassSource,
		Decision:         RedactionDecisionPreserveAndWarn,
		RedactionApplied: false,
		WarningCount:     3,
	}

	output := RenderFileRedactionMetadata(meta)

	if !strings.Contains(output, "class=source") {
		t.Error("expected class=source")
	}
	if !strings.Contains(output, "decision=preserve_and_warn") {
		t.Error("expected decision=preserve_and_warn")
	}
	if !strings.Contains(output, "redaction_applied=false") {
		t.Error("expected redaction_applied=false for source")
	}
	if !strings.Contains(output, "source_secret_scan=warn_only") {
		t.Error("expected source_secret_scan=warn_only when warnings exist")
	}
}

func TestRenderFileRedactionMetadata_NonSource(t *testing.T) {
	meta := FileRedactionMetadata{
		Path:             "app.log",
		Class:            RedactionClassNonSource,
		Decision:         RedactionDecisionRedact,
		RedactionApplied: true,
		WarningCount:     0,
	}

	output := RenderFileRedactionMetadata(meta)

	if !strings.Contains(output, "class=non_source") {
		t.Error("expected class=non_source")
	}
	if !strings.Contains(output, "decision=redact") {
		t.Error("expected decision=redact")
	}
	if !strings.Contains(output, "redaction_applied=true") {
		t.Error("expected redaction_applied=true for non-source")
	}
	// Should NOT have source_secret_scan for non-source
	if strings.Contains(output, "source_secret_scan") {
		t.Error("non-source should not have source_secret_scan")
	}
}

func TestSplitDigestSections(t *testing.T) {
	digest := `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 2
LEAMAS_VERSION: dev

# Targeted digest

=== tests/test.py ===
Metadata: tracked, staged present: yes, unstaged present: no

password = "secret123"
token = "abc123"

=== config/app.log ===
Metadata: tracked, staged present: yes, unstaged present: no

password=secret123
token=abc123`

	sections := splitDigestSections(digest)

	// Should have at least 2 sections
	if len(sections) < 2 {
		t.Fatalf("expected at least 2 sections, got %d", len(sections))
	}

	// Find the Python section
	var pySection *digestSection
	var logSection *digestSection
	for i := range sections {
		if sections[i].Path == "tests/test.py" {
			pySection = &sections[i]
		}
		if sections[i].Path == "config/app.log" {
			logSection = &sections[i]
		}
	}

	if pySection == nil {
		t.Fatal("expected to find tests/test.py section")
	}
	if logSection == nil {
		t.Fatal("expected to find config/app.log section")
	}

	if !pySection.IsFileSection {
		t.Error("test.py should be a file section")
	}
	if !logSection.IsFileSection {
		t.Error("app.log should be a file section")
	}

	if !pySection.Tracked {
		t.Error("test.py should be tracked")
	}
	if !logSection.Tracked {
		t.Error("app.log should be tracked")
	}
}

func TestCountSecretFindings(t *testing.T) {
	content := `password = "secret123"
secret = "mysecret"
token = "abc123"
api_key = "xyz789"
normal_line = "hello"`

	count := countSecretFindings(content)

	// Should count at least password, secret, token patterns
	if count < 3 {
		t.Errorf("expected at least 3 findings, got %d", count)
	}
}
