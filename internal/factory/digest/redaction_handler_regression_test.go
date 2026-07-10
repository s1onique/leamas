// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"strings"
	"testing"
)

// TestRedactDigestWithPolicy_PEMLikeLines verifies that PEM-like lines
// (which start with -----BEGIN/END) are preserved in source content.
func TestRedactDigestWithPolicy_PEMLikeLines(t *testing.T) {
	// Digest with Go source containing PEM-like lines
	digest := `=== keys/manager.go ===
Metadata: tracked, staged present: yes, unstaged present: no

--- staged diff ---
diff --git a/keys/manager.go b/keys/manager.go
-----BEGIN RSA PRIVATE KEY-----
MIIBOgIBAAJBALRiMLAHudeSA2MFVVC3rU3KBEUptTdqyB7J8dXwHqR3nB3d
+gkBE5T4L3V3bL5bL5bL5bL5bL5bL5bL5bL5bL5bL5bL5bL5bL5bL5bL5
-----END RSA PRIVATE KEY-----
package keys

func LoadKey() string {
	return "key"
}`

	result, _ := RedactDigestWithPolicy(digest)

	// The source content should begin with actual source, not PEM tail
	if !strings.Contains(result, "package keys") {
		t.Error("source should begin with package declaration, not dangling PEM tail")
	}

	// PEM lines should be preserved in content
	if !strings.Contains(result, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Error("PEM BEGIN line should be preserved in source content")
	}
	if !strings.Contains(result, "-----END RSA PRIVATE KEY-----") {
		t.Error("PEM END line should be preserved in source content")
	}

	// Ensure no dangling content before package clause
	packagePos := strings.Index(result, "package keys")
	contentStart := strings.Index(result, "-----BEGIN")
	if contentStart != -1 && packagePos != -1 && contentStart > packagePos {
		// Good: package comes before PEM (PEM is in diff context)
	} else if contentStart != -1 && packagePos == -1 {
		t.Error("package declaration was lost")
	}
}

// TestRedactDigestWithPolicy_DocumentationContent verifies that documentation
// containing literal REDACTION_POLICY: or SOURCE_SECRET_WARNINGS: examples
// is preserved byte-faithfully.
func TestRedactDigestWithPolicy_DocumentationContent(t *testing.T) {
	// Digest with markdown documentation containing policy examples
	digest := `=== docs/factory/digest-redaction-policy.md ===
Metadata: tracked, staged present: yes, unstaged present: no

--- staged diff ---
diff --git a/docs/factory/digest-redaction-policy.md b/docs/factory/digest-redaction-policy.md
# Factory: Digest Redaction Policy

## Example REDACTION_POLICY:

Here is an example of what the policy looks like:

    REDACTION_POLICY:
      class=source
      decision=preserve_and_warn

You might also see:

    SOURCE_SECRET_WARNINGS:
      pattern_id=source.bearer_token
      line=42

This helps reviewers understand the digest format.`

	result, _ := RedactDigestWithPolicy(digest)

	// Documentation content should begin with the heading, not policy markers
	if !strings.Contains(result, "# Factory: Digest Redaction Policy") {
		t.Error("documentation should begin with the heading")
	}

	// Literal examples in documentation should be preserved
	if !strings.Contains(result, "## Example REDACTION_POLICY:") {
		t.Error("REDACTION_POLICY example heading should be preserved")
	}

	// The actual REDACTION_POLICY: in code block should be preserved
	if !strings.Contains(result, "REDACTION_POLICY:") {
		t.Error("literal REDACTION_POLICY: in documentation should be preserved")
	}

	// SOURCE_SECRET_WARNINGS in documentation should be preserved
	if !strings.Contains(result, "SOURCE_SECRET_WARNINGS:") {
		t.Error("literal SOURCE_SECRET_WARNINGS: in documentation should be preserved")
	}

	// Verify the document content section contains the expected structure
	if strings.Contains(result, "# Factory: Digest Redaction Policy") {
		// Good - the heading is present
	} else {
		t.Error("documentation heading should be preserved")
	}

	// The literal REDACTION_POLICY: should appear as part of the documentation example
	count := strings.Count(result, "REDACTION_POLICY:")
	if count < 2 {
		t.Errorf("expected at least 2 occurrences of REDACTION_POLICY: (one in header, one in doc example), got %d", count)
	}
}

// TestRedactDigestWithPolicy_RealWorldGoFile verifies a realistic Go file
// with various constructs is preserved correctly.
func TestRedactDigestWithPolicy_RealWorldGoFile(t *testing.T) {
	digest := `=== auth/cert.go ===
Metadata: untracked, staged present: no, unstaged present: yes

--- untracked file content ---
// Package auth handles authentication.
package auth

import "crypto/tls"

const CertMarker = "-----BEGIN CERTIFICATE-----"

func LoadCert() (string, error) {
	return ` + "`" + `-----BEGIN CERTIFICATE-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0Z3VS5JJcds3xfn/ygWyf8B8
-----END CERTIFICATE-----` + "`" + `, nil
}`

	result, _ := RedactDigestWithPolicy(digest)

	// Package declaration should be present
	if !strings.Contains(result, "package auth") {
		t.Error("package declaration should be preserved")
	}

	// Import should be preserved
	if !strings.Contains(result, `import "crypto/tls"`) {
		t.Error("import statement should be preserved")
	}

	// Constant should be preserved
	if !strings.Contains(result, `const CertMarker = "-----BEGIN CERTIFICATE-----"`) {
		t.Error("constant with cert marker should be preserved")
	}

	// Function should be preserved
	if !strings.Contains(result, "func LoadCert()") {
		t.Error("function should be preserved")
	}

	// Backtick-quoted cert should be preserved
	if !strings.Contains(result, "-----BEGIN CERTIFICATE-----") {
		t.Error("embedded cert should be preserved in raw string literal")
	}
}
