// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAnchors_MissingFile(t *testing.T) {
	// Create a temp directory without anchors.toml
	tmpDir := t.TempDir()

	config, err := LoadAnchors(tmpDir)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if config != nil {
		t.Fatalf("expected nil config for missing file, got: %v", config)
	}
}

func TestLoadAnchors_OneAnchor(t *testing.T) {
	tmpDir := t.TempDir()

	// Create anchors.toml with one anchor
	content := `[[anchors]]
id = "EPIC-001"
type = "epic"
summary = "Test epic"
url = "docs/epics/EPIC-001.md"
`
	anchorsPath := filepath.Join(tmpDir, ".leamas", "anchors.toml")
	if err := os.MkdirAll(filepath.Dir(anchorsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchorsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadAnchors(tmpDir)
	if err != nil {
		t.Fatalf("failed to load anchors: %v", err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if len(config.Anchors) != 1 {
		t.Fatalf("expected 1 anchor, got: %d", len(config.Anchors))
	}
	if config.Anchors[0].ID != "EPIC-001" {
		t.Errorf("expected ID 'EPIC-001', got: %s", config.Anchors[0].ID)
	}
}

func TestLoadAnchors_MultipleAnchorsInOrder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create anchors.toml with multiple anchors
	content := `[[anchors]]
id = "ACT-001"
type = "act"
summary = "First act"

[[anchors]]
id = "EPIC-001"
type = "epic"
summary = "Second epic"
url = "docs/epics/EPIC-001.md"

[[anchors]]
id = "ADR-001"
type = "adr"
summary = "Third adr"
`
	anchorsPath := filepath.Join(tmpDir, ".leamas", "anchors.toml")
	if err := os.MkdirAll(filepath.Dir(anchorsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchorsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadAnchors(tmpDir)
	if err != nil {
		t.Fatalf("failed to load anchors: %v", err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if len(config.Anchors) != 3 {
		t.Fatalf("expected 3 anchors, got: %d", len(config.Anchors))
	}

	// Verify order is preserved
	expectedIDs := []string{"ACT-001", "EPIC-001", "ADR-001"}
	for i, expected := range expectedIDs {
		if config.Anchors[i].ID != expected {
			t.Errorf("anchor %d: expected ID '%s', got: '%s'", i, expected, config.Anchors[i].ID)
		}
	}
}

func TestLoadAnchors_MissingURL(t *testing.T) {
	tmpDir := t.TempDir()

	// Create anchors.toml without URL
	content := `[[anchors]]
id = "ACT-001"
type = "act"
summary = "Test act"
`
	anchorsPath := filepath.Join(tmpDir, ".leamas", "anchors.toml")
	if err := os.MkdirAll(filepath.Dir(anchorsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchorsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadAnchors(tmpDir)
	if err != nil {
		t.Fatalf("failed to load anchors: %v", err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if config.Anchors[0].URL != "" {
		t.Errorf("expected empty URL, got: %s", config.Anchors[0].URL)
	}
}

// TestLoadAnchors_MalformedMissingClosingQuote tests that missing closing quote returns error.
func TestLoadAnchors_MalformedMissingClosingQuote(t *testing.T) {
	tmpDir := t.TempDir()

	// Missing closing quote on id value
	content := `[[anchors]]
id = "ACT-001
type = "act"
summary = "Missing closing quote"
`
	anchorsPath := filepath.Join(tmpDir, ".leamas", "anchors.toml")
	if err := os.MkdirAll(filepath.Dir(anchorsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchorsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadAnchors(tmpDir)
	if err == nil {
		t.Fatal("expected error for malformed config with missing closing quote")
	}
	if !errors.Is(err, ErrMalformedAnchors) {
		t.Errorf("expected ErrMalformedAnchors, got: %v", err)
	}
}

// TestLoadAnchors_MalformedUnknownKey tests that unknown keys return error.
func TestLoadAnchors_MalformedUnknownKey(t *testing.T) {
	tmpDir := t.TempDir()

	content := `[[anchors]]
id = "ACT-001"
type = "act"
summary = "Test act"
unknown = "field"
`
	anchorsPath := filepath.Join(tmpDir, ".leamas", "anchors.toml")
	if err := os.MkdirAll(filepath.Dir(anchorsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchorsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadAnchors(tmpDir)
	if err == nil {
		t.Fatal("expected error for malformed config with unknown key")
	}
	if !errors.Is(err, ErrMalformedAnchors) {
		t.Errorf("expected ErrMalformedAnchors, got: %v", err)
	}
}

// TestLoadAnchors_MalformedWrongSection tests that wrong section names return error.
func TestLoadAnchors_MalformedWrongSection(t *testing.T) {
	tmpDir := t.TempDir()

	content := `[[wrong_section]]
id = "ACT-001"
`
	anchorsPath := filepath.Join(tmpDir, ".leamas", "anchors.toml")
	if err := os.MkdirAll(filepath.Dir(anchorsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchorsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadAnchors(tmpDir)
	if err == nil {
		t.Fatal("expected error for malformed config with wrong section")
	}
	if !errors.Is(err, ErrMalformedAnchors) {
		t.Errorf("expected ErrMalformedAnchors, got: %v", err)
	}
}

// TestLoadAnchors_MalformedContentOutsideAnchorsBlock tests that content outside anchors block returns error.
func TestLoadAnchors_MalformedContentOutsideAnchorsBlock(t *testing.T) {
	tmpDir := t.TempDir()

	content := `some random text
[[anchors]]
id = "ACT-001"
`
	anchorsPath := filepath.Join(tmpDir, ".leamas", "anchors.toml")
	if err := os.MkdirAll(filepath.Dir(anchorsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchorsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadAnchors(tmpDir)
	if err == nil {
		t.Fatal("expected error for malformed config with content outside anchors block")
	}
	if !errors.Is(err, ErrMalformedAnchors) {
		t.Errorf("expected ErrMalformedAnchors, got: %v", err)
	}
}

// TestLoadAnchors_ValidWithComments tests that comments are allowed.
func TestLoadAnchors_ValidWithComments(t *testing.T) {
	tmpDir := t.TempDir()

	content := `# This is a comment
[[anchors]]
id = "ACT-001"
type = "act"
summary = "Test act"
# Another comment
`
	anchorsPath := filepath.Join(tmpDir, ".leamas", "anchors.toml")
	if err := os.MkdirAll(filepath.Dir(anchorsPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchorsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadAnchors(tmpDir)
	if err != nil {
		t.Fatalf("failed to load anchors with comments: %v", err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if len(config.Anchors) != 1 {
		t.Fatalf("expected 1 anchor, got: %d", len(config.Anchors))
	}
	if config.Anchors[0].ID != "ACT-001" {
		t.Errorf("expected ID 'ACT-001', got: %s", config.Anchors[0].ID)
	}
}

func TestRenderAnchors_EmptyConfig(t *testing.T) {
	result := RenderAnchors(nil)
	expected := "No workflow anchors configured.\n"
	if result != expected {
		t.Errorf("expected '%s', got: '%s'", expected, result)
	}

	result = RenderAnchors(&AnchorsConfig{})
	if result != expected {
		t.Errorf("expected '%s' for empty config, got: '%s'", expected, result)
	}
}

func TestRenderAnchors_OneAnchor(t *testing.T) {
	config := &AnchorsConfig{
		Anchors: []Anchor{
			{ID: "ACT-001", Type: "act", Summary: "Test act", URL: "docs/acts/ACT-001.md"},
		},
	}

	result := RenderAnchors(config)

	// Check table header
	if !contains(result, "| ID | Type | Summary | URL |") {
		t.Error("missing table header")
	}
	if !contains(result, "| ACT-001 | act | Test act |") {
		t.Error("missing anchor row")
	}
	if !contains(result, "docs/acts/ACT-001.md") {
		t.Error("missing URL in row")
	}
}

func TestRenderAnchors_MultipleAnchors(t *testing.T) {
	config := &AnchorsConfig{
		Anchors: []Anchor{
			{ID: "ACT-001", Type: "act", Summary: "First act"},
			{ID: "EPIC-001", Type: "epic", Summary: "Second epic", URL: "docs/epics/EPIC-001.md"},
		},
	}

	result := RenderAnchors(config)

	// Check both anchors are present
	if !contains(result, "ACT-001") {
		t.Error("missing ACT-001")
	}
	if !contains(result, "EPIC-001") {
		t.Error("missing EPIC-001")
	}
}

func TestRenderAnchors_MissingURLRendersDash(t *testing.T) {
	config := &AnchorsConfig{
		Anchors: []Anchor{
			{ID: "ACT-001", Type: "act", Summary: "Test act"},
		},
	}

	result := RenderAnchors(config)

	// Missing URL should render as "-"
	if !contains(result, "| ACT-001 | act | Test act | - |") {
		t.Errorf("expected dash for missing URL, got: %s", result)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
