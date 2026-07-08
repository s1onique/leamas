package redact

import (
	"strings"
	"testing"
)

func TestRedactAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"OpenAI key", `api_key='sk-1234567890abcdefghij'`, "[REDACTED]"},
		{"Bearer token", `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9`, "Bearer [REDACTED]"},
		{"GitHub token", `ghp_1234567890abcdefghijklmnopqrstuvwxyzAB`, "ghp_[REDACTED]"},
		{"AWS key", `AKIAIOSFODNN7EXAMPLE`, "AKIA[REDACTED]"},
		{"generic secret", `password=mysecretpassword123`, "[REDACTED]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactDefault(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected %q in result, got %q", tt.contains, result)
			}
		})
	}
}

func TestRedactPreservesStructure(t *testing.T) {
	input := `{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "hello"}],
  "api_key": "sk-1234567890abcdefghijklmnopqrstuvwxyz"
}`
	result := RedactDefault(input)

	// Should still have valid JSON structure
	if !strings.Contains(result, `"model"`) {
		t.Error("should preserve model field")
	}
	if !strings.Contains(result, `"messages"`) {
		t.Error("should preserve messages field")
	}
	if strings.Contains(result, "sk-1234567890") {
		t.Error("should redact the API key")
	}
}

func TestRedactNonSecret(t *testing.T) {
	input := "This is a normal log message without any secrets."
	result := RedactDefault(input)
	if result != input {
		t.Error("should not modify non-secret content")
	}
}

func TestIsSecretPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// These match default patterns
		{"sk-1234567890abcdefghij", true},     // OpenAI key (sk- prefix + 20+ chars)
		{"sk-ant-1234567890abcdefghij", true}, // Anthropic key (sk-ant- prefix)
		// These don't match the strict patterns in IsSecretPattern
		{"hello", false},                          // Too short
		{"normal_variable_name", false},           // Not a secret pattern
		{"ghp_1234567890abcdefghijklmnop", false}, // GitHub tokens need full 36 chars
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsSecretPattern(tt.input)
			if got != tt.expected {
				t.Errorf("IsSecretPattern(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRedactHash(t *testing.T) {
	// Long hex strings should be redacted
	input := `commit: abcdef1234567890abcdef1234567890abcdef12`
	result := RedactDefault(input)

	if strings.Contains(result, "abcdef1234567890abcdef1234567890abcdef12") {
		t.Error("should redact long hex hash")
	}
}
