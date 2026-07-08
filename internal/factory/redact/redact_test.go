package redact

import (
	"strings"
	"testing"
)

func TestRedactBearerToken(t *testing.T) {
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ"
	result := RedactDefault(input)
	if !strings.Contains(result, "Bearer [REDACTED]") {
		t.Errorf("Bearer token should be redacted, got: %s", result)
	}
}

func TestRedactOpenAIKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"sk- prefix", "api_key='sk-1234567890abcdefghijklmnop'"},
		{"sk- in code", "OPENAI_API_KEY=sk-1234567890abcdefghijklmnopqrstuvwxyz"},
		{"sk- with quotes", `"api_key": "sk-abcdefghijklmnopqrstuvwxyz"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactDefault(tt.input)
			if !strings.Contains(result, "sk-[REDACTED]") && !strings.Contains(result, "[REDACTED]") {
				t.Errorf("OpenAI key should be redacted, got: %s", result)
			}
			if strings.Contains(result, "sk-1234567890") || strings.Contains(result, "sk-abcdefghijklmnop") {
				t.Errorf("OpenAI key should not be visible, got: %s", result)
			}
		})
	}
}

func TestRedactAnthropicKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"sk-ant- prefix", "ANTHROPIC_API_KEY=sk-ant-api1234567890abcdefghijklmnopqrst"},
		{"sk-ant- in code", "api_key='sk-ant-1234567890abcdefghijklmnopqrstuvwxyz'"},
		{"sk-ant- with quotes", `"key": "sk-ant-abcdefghijklmnopqrstuvwxyz"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactDefault(tt.input)
			if !strings.Contains(result, "sk-ant-[REDACTED]") && !strings.Contains(result, "[REDACTED]") {
				t.Errorf("Anthropic key should be redacted, got: %s", result)
			}
			if strings.Contains(result, "sk-ant-api1234567890") || strings.Contains(result, "sk-ant-1234567890abcdef") {
				t.Errorf("Anthropic key should not be visible, got: %s", result)
			}
		})
	}
}

func TestRedactGitHubToken(t *testing.T) {
	input := "ghp_1234567890abcdefghijklmnopqrstuvwxyzAB"
	result := RedactDefault(input)
	if !strings.Contains(result, "ghp_[REDACTED]") {
		t.Errorf("GitHub token should be redacted, got: %s", result)
	}
	if strings.Contains(result, "ghp_1234567890abcdefghijklmnopqrstuvwxyz") {
		t.Errorf("GitHub token should not be visible, got: %s", result)
	}
}

func TestRedactAWSAccessKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"AKIA pattern", "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE", "AKIA[REDACTED_AWS_KEY]"},
		{"AKIA inline", "export AWS_SECRET_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE", "AKIA[REDACTED_AWS_KEY]"},
		{"ASIA pattern", "AWS_SESSION_TOKEN=ASIAAKIAIOSFODNN7EXAMPLE", "ASIAAKIAIOSFODNN7EXAMPLE"},
		{"ASIA inline", "aws_access_key_id: ASIA1234567890ABCDEF", "ASIA1234567890ABCDEF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactDefault(tt.input)
			// Check if the actual key pattern matches
			hasRedaction := strings.Contains(result, "[REDACTED_AWS_KEY]")
			hasAKIA := strings.Contains(result, "AKIA") && !strings.Contains(result, "AKIA[REDACTED_AWS_KEY]")
			hasASIA := strings.Contains(result, "ASIA") && !strings.Contains(result, "[REDACTED_AWS_KEY]")
			if !hasRedaction && (hasAKIA || hasASIA) {
				t.Errorf("AWS key should be redacted, got: %s", result)
			}
		})
	}
}

func TestRedactSecretVariables(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"password assignment", "password=mySecretPassword123!"},
		{"secret assignment", "secret=super_secret_value_12345"},
		{"token assignment", "token=myApiTokenValue12345"},
		{"api_key assignment", "api_key=myApiKeyValue12345"},
		{"passwd assignment", "passwd=Password123!"},
		{"pwd assignment", "pwd=Password123!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactDefault(tt.input)
			if !strings.Contains(result, "[REDACTED]") {
				t.Errorf("Secret variable should be redacted, got: %s", result)
			}
		})
	}
}

func TestRedactPrivateKey(t *testing.T) {
	input := `-----BEGIN RSA PRIVATE KEY-----
MIIBOgIBAAJBALRiMLAHudeSA2F+0TaROVvyLpvIMCXdGShDNCvPTH8p4oZx
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0Z3VS5JJcds3xfn/yg==
-----END RSA PRIVATE KEY-----`
	result := RedactDefault(input)
	if !strings.Contains(result, "-----BEGIN [REDACTED] PRIVATE KEY-----") {
		t.Errorf("Private key header should be redacted, got: %s", result)
	}
}

func TestRedactPreservesOrdinaryText(t *testing.T) {
	input := "This is a normal log message without any secrets. It contains only ordinary text and numbers like 12345."
	result := RedactDefault(input)
	if result != input {
		t.Errorf("Ordinary text should not be modified.\nInput: %s\nOutput: %s", input, result)
	}
}

func TestRedactPreservesGitCommitHash(t *testing.T) {
	// Git commit hashes are 40-character hex strings that should NOT be redacted
	// because they are important evidence for digest review
	commitHashes := []string{
		"494dd7f7356c7f712c456b7e0577c89146c45f26",
		"abc123def456789012345678901234567890abcd",
		"deadbeef1234567890abcdef1234567890abcdef",
		"0000000000000000000000000000000000000000",
	}
	for _, hash := range commitHashes {
		input := "commit: " + hash
		result := RedactDefault(input)
		if strings.Contains(result, "[REDACTED") || strings.Contains(result, "[REDACTED_HASH]") {
			t.Errorf("Git commit hash %s should NOT be redacted, got: %s", hash, result)
		}
		if !strings.Contains(result, hash) {
			t.Errorf("Git commit hash %s should be preserved, got: %s", hash, result)
		}
	}
}

func TestRedactPreservesStructure(t *testing.T) {
	input := `{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "hello"}],
  "api_key": "sk-1234567890abcdefghijklmnopqrstuvwxyz"
}`
	result := RedactDefault(input)

	// Should preserve structure
	if !strings.Contains(result, `"model"`) {
		t.Error("should preserve model field")
	}
	if !strings.Contains(result, `"messages"`) {
		t.Error("should preserve messages field")
	}
	if !strings.Contains(result, `"api_key"`) {
		t.Error("should preserve api_key key name")
	}
	// But should redact the actual key value
	if strings.Contains(result, "sk-1234567890") {
		t.Error("should redact the API key value")
	}
}

func TestIsSecretPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// These match secret patterns
		{"sk-1234567890abcdefghij", true},                  // OpenAI key
		{"sk-ant-1234567890abcdefghij", true},              // Anthropic key
		{"ghp_123456789012345678901234567890123456", true}, // GitHub token (36 chars after prefix = 40 total)
		// These don't match - not secret patterns
		{"hello", false},                              // Too short
		{"normal_variable_name", false},               // Not a secret pattern
		{"abcdef1234567890abcdef1234567890ab", false}, // Git commit hash - not redacted by design
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
