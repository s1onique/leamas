// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"
	"strings"
)

// Anchor represents a workflow anchor to include in digest output.
type Anchor struct {
	ID      string // Unique identifier (e.g., "epic-123", "act-456")
	Type    string // "epic", "act", "adr", "ticket"
	Summary string // Brief summary
	URL     string // Optional link
}

// AnchorsConfig represents the digest anchors configuration.
type AnchorsConfig struct {
	Anchors []Anchor
}

// DefaultAnchorsPath returns the default anchors config path.
func DefaultAnchorsPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".leamas", "anchors.toml")
}

// LoadAnchors loads anchors from the default config path.
func LoadAnchors(repoRoot string) (*AnchorsConfig, error) {
	path := DefaultAnchorsPath(repoRoot)
	return LoadAnchorsFrom(path)
}

// LoadAnchorsFrom loads anchors from a specific file path.
// Returns nil config if file doesn't exist.
func LoadAnchorsFrom(path string) (*AnchorsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config, not an error
		}
		return nil, err
	}

	// Simple TOML parsing for anchors
	// Format:
	// [[anchors]]
	// id = "epic-123"
	// type = "epic"
	// summary = "Important epic"
	//
	// [[anchors]]
	// id = "act-456"
	// type = "act"
	// summary = "Action item"
	config := &AnchorsConfig{}

	// Simple line-by-line parsing
	lines := splitLines(string(data))
	var currentAnchor *Anchor

	for _, line := range lines {
		line = trimLine(line)

		if line == `[[anchors]]` {
			if currentAnchor != nil {
				config.Anchors = append(config.Anchors, *currentAnchor)
			}
			currentAnchor = &Anchor{}
			continue
		}

		if currentAnchor == nil {
			continue
		}

		// Parse key = "value" lines
		if len(line) > 4 && line[len(line)-1] == '"' {
			eqIdx := -1
			for i := 0; i < len(line); i++ {
				if line[i] == '=' {
					eqIdx = i
					break
				}
			}
			if eqIdx > 0 {
				key := trimLine(line[:eqIdx])
				value := trimLine(line[eqIdx+1:])
				value = trimQuotes(value)

				switch key {
				case "id":
					currentAnchor.ID = value
				case "type":
					currentAnchor.Type = value
				case "summary":
					currentAnchor.Summary = value
				case "url":
					currentAnchor.URL = value
				}
			}
		}
	}

	if currentAnchor != nil {
		config.Anchors = append(config.Anchors, *currentAnchor)
	}

	return config, nil
}

// RenderAnchors renders anchors for digest output.
func RenderAnchors(config *AnchorsConfig) string {
	if config == nil || len(config.Anchors) == 0 {
		return "No workflow anchors configured.\n"
	}

	var sb strings.Builder
	sb.WriteString("| ID | Type | Summary | URL |\n")
	sb.WriteString("|----|------|---------|-----|\n")

	for _, anchor := range config.Anchors {
		url := "-"
		if anchor.URL != "" {
			url = anchor.URL
		}
		sb.WriteString("| ")
		sb.WriteString(anchor.ID)
		sb.WriteString(" | ")
		sb.WriteString(anchor.Type)
		sb.WriteString(" | ")
		sb.WriteString(anchor.Summary)
		sb.WriteString(" | ")
		sb.WriteString(url)
		sb.WriteString(" |\n")
	}

	return sb.String()
}

func splitLines(s string) []string {
	var lines []string
	var current string
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func trimLine(s string) string {
	// Trim whitespace
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
