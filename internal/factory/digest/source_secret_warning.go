// Package digest provides targeted digest generation for Git repositories.
// It creates reviewable artifacts of repository changes suitable for
// agent-assisted review workflows.
package digest

import (
	"regexp"
	"strings"
)

// SourceSecretPatternID identifies a detected secret-like pattern.
type SourceSecretPatternID string

const (
	SourcePatternPasswordAssignment SourceSecretPatternID = "source.password_assignment"
	SourcePatternSecretAssignment   SourceSecretPatternID = "source.secret_assignment"
	SourcePatternTokenAssignment    SourceSecretPatternID = "source.token_assignment"
	SourcePatternAPIKeyAssignment   SourceSecretPatternID = "source.api_key_assignment"
	SourcePatternBearerToken        SourceSecretPatternID = "source.bearer_token"
	SourcePatternPEMPrivateKey      SourceSecretPatternID = "source.pem_private_key"
)

// SourceSecretConfidence indicates confidence level of the detection.
type SourceSecretConfidence string

const (
	ConfidencePattern SourceSecretConfidence = "pattern"
	ConfidenceHigh    SourceSecretConfidence = "high"
)

// SourceSecretFinding represents a detected secret-like pattern in source.
type SourceSecretFinding struct {
	Line       int
	Kind       SourceSecretPatternID
	Confidence SourceSecretConfidence
	Column     int // 0 if unknown
}

// SourceSecretWarning contains findings for a source file.
type SourceSecretWarning struct {
	Path     string
	Findings []SourceSecretFinding
}

// sourceSecretPattern represents a pattern to detect in source files.
type sourceSecretPattern struct {
	id         SourceSecretPatternID
	pattern    *regexp.Regexp
	confidence SourceSecretConfidence
}

// sourceSecretPatterns are patterns to detect secret-like literals in source.
// These patterns intentionally do NOT capture the secret value itself.
var sourceSecretPatterns = []sourceSecretPattern{
	// Password assignments: password = value, password string, password: value
	{
		id:         SourcePatternPasswordAssignment,
		pattern:    regexp.MustCompile(`(?im)\bpassword\s*(?:[=:]\s*\w*|\s+(?:string|int|bool|Value|var|const)\b)`),
		confidence: ConfidencePattern,
	},
	// Secret assignments: secret = value, secret= value, secret := value
	{
		id:         SourcePatternSecretAssignment,
		pattern:    regexp.MustCompile(`(?im)\bsecret\s*[=:]+\s*\w*`),
		confidence: ConfidencePattern,
	},
	// Token assignments: token = value, token= value, token := value
	{
		id:         SourcePatternTokenAssignment,
		pattern:    regexp.MustCompile(`(?im)\btoken\s*[=:]+\s*\w*`),
		confidence: ConfidencePattern,
	},
	// API key assignments: api_key = value, api_key= value, api_key := value
	{
		id:         SourcePatternAPIKeyAssignment,
		pattern:    regexp.MustCompile(`(?im)\b(api[_-]?key|apikey)\s*[=:]+\s*\w*`),
		confidence: ConfidencePattern,
	},
	// Bearer token format
	{
		id:         SourcePatternBearerToken,
		pattern:    regexp.MustCompile(`(?i)\bbearer\s+[a-zA-Z0-9_\-\.]{10,}`),
		confidence: ConfidencePattern,
	},
	// PEM private key header
	{
		id:         SourcePatternPEMPrivateKey,
		pattern:    regexp.MustCompile(`-----BEGIN\s+[A-Z]+\s+PRIVATE\s+KEY-----`),
		confidence: ConfidenceHigh,
	},
}

// ScanSourceForSecrets scans source content for secret-like patterns.
// Returns a SourceSecretWarning with all findings.
//
// WARNING: This function only detects PATTERNS, not actual secrets.
// False positives are expected. Reviewers should verify findings.
func ScanSourceForSecrets(path string, content string) SourceSecretWarning {
	var findings []SourceSecretFinding
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		for _, sp := range sourceSecretPatterns {
			loc := sp.pattern.FindStringIndex(line)
			if loc != nil {
				findings = append(findings, SourceSecretFinding{
					Line:       lineNum + 1, // 1-indexed
					Kind:       sp.id,
					Confidence: sp.confidence,
					Column:     loc[0] + 1, // 1-indexed
				})
			}
		}
	}

	return SourceSecretWarning{
		Path:     path,
		Findings: findings,
	}
}

// RenderSourceSecretWarnings renders source secret warning metadata.
// Returns empty string if no findings.
func RenderSourceSecretWarnings(warning SourceSecretWarning) string {
	if len(warning.Findings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("SOURCE_SECRET_WARNINGS:\n")
	for _, f := range warning.Findings {
		sb.WriteString("  - ")
		sb.WriteString("line=")
		sb.WriteString(formatInt(f.Line))
		sb.WriteString(" kind=")
		sb.WriteString(string(f.Kind))
		sb.WriteString(" confidence=")
		sb.WriteString(string(f.Confidence))
		if f.Column > 0 {
			sb.WriteString(" column=")
			sb.WriteString(formatInt(f.Column))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// formatInt converts int to string for warning output.
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + positiveIntToString(-n)
	}
	return positiveIntToString(n)
}

// positiveIntToString converts a positive int to string.
func positiveIntToString(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

// HasSourceSecretWarnings returns true if the warning contains any findings.
func (w SourceSecretWarning) HasFindings() bool {
	return len(w.Findings) > 0
}

// FindingCount returns the number of findings.
func (w SourceSecretWarning) FindingCount() int {
	return len(w.Findings)
}

// PatternIDs returns a slice of unique pattern IDs found.
func (w SourceSecretWarning) PatternIDs() []SourceSecretPatternID {
	seen := make(map[SourceSecretPatternID]bool)
	var ids []SourceSecretPatternID
	for _, f := range w.Findings {
		if !seen[f.Kind] {
			seen[f.Kind] = true
			ids = append(ids, f.Kind)
		}
	}
	return ids
}
