// Package redact provides redaction utilities for digest and trace output.
// Redaction removes obvious secret patterns before output to prevent accidental exposure.
package redact

import (
	"regexp"
	"strings"
)

// SecretPattern represents a pattern to redact.
type SecretPattern struct {
	Pattern *regexp.Regexp
	Replace string
}

// DefaultPatterns returns the default secret redaction patterns.
func DefaultPatterns() []SecretPattern {
	return []SecretPattern{
		// Bearer tokens
		{Pattern: regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.]{20,}`), Replace: "Bearer [REDACTED]"},
		// OpenAI API keys (sk- prefix + 20+ chars)
		{Pattern: regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`), Replace: "sk-[REDACTED]"},
		// Anthropic API keys (sk-ant- prefix + 20+ chars)
		{Pattern: regexp.MustCompile(`(?i)sk-ant-[a-zA-Z0-9]{20,}`), Replace: "sk-ant-[REDACTED]"},
		// GitHub tokens (ghp_ prefix + 36 alphanumeric chars)
		{Pattern: regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`), Replace: "ghp_[REDACTED]"},
		// Generic secret variables (secret=, password=, token=, api_key= assignments)
		{Pattern: regexp.MustCompile(`(?i)(api[_-]?key|secret|password|passwd|pwd|token)['":\s=]+['"]?[a-zA-Z0-9_\-]{8,}['"]?`), Replace: "$1=[REDACTED]"},
		// Private keys
		{Pattern: regexp.MustCompile(`-----BEGIN [A-Z]+ PRIVATE KEY-----`), Replace: "-----BEGIN [REDACTED] PRIVATE KEY-----"},
		// AWS access keys (AKIA and ASIA patterns with exactly 16 chars after prefix)
		{Pattern: regexp.MustCompile(`(?i)\b(AKIA|ASIA)[A-Z0-9]{16}\b`), Replace: "[REDACTED_AWS_KEY]"},
	}
}

// Redact applies redaction patterns to the input string.
func Redact(input string, patterns []SecretPattern) string {
	result := input
	for _, p := range patterns {
		result = p.Pattern.ReplaceAllString(result, p.Replace)
	}
	return result
}

// RedactDefault applies default redaction patterns to input.
func RedactDefault(input string) string {
	return Redact(input, DefaultPatterns())
}

// RedactDigest redacts secrets from digest output.
// Keeps structure visible but removes sensitive values.
func RedactDigest(digest string) string {
	return RedactDefault(digest)
}

// RedactTrace redacts secrets from trace output.
func RedactTrace(trace string) string {
	return RedactDefault(trace)
}

// RedactRequest redacts secrets from API request output.
func RedactRequest(req string) string {
	return RedactDefault(req)
}

// RedactResponse redacts secrets from API response output.
func RedactResponse(resp string) string {
	return RedactDefault(resp)
}

// IsSecretPattern returns true if the string looks like a secret.
func IsSecretPattern(s string) bool {
	s = strings.TrimSpace(s)
	// Too short to be a meaningful secret
	if len(s) < 8 {
		return false
	}
	// Check against known secret-like patterns
	patterns := DefaultPatterns()
	for _, p := range patterns {
		if p.Pattern.MatchString(s) {
			return true
		}
	}
	return false
}
