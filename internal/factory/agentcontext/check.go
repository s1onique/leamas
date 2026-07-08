// Package agentcontext provides verification for agent instruction files.
// It ensures AGENTS.md and .clinerules/leamas.md exist and contain required content.
package agentcontext

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Finding represents a single agent context verification finding.
type Finding struct {
	Path    string
	Kind    string
	Message string
}

// CheckRepo verifies that agent context files exist and contain required content.
func CheckRepo(root string) ([]Finding, error) {
	var findings []Finding

	// Check AGENTS.md
	findings = append(findings, checkAgentsMD(root)...)

	// Check .clinerules/leamas.md
	findings = append(findings, checkClineRules(root)...)

	// Check docs/factory/agent-context-files.md exists
	findings = append(findings, checkPolicyDoc(root)...)

	// Sort findings deterministically by path
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Kind < findings[j].Kind
	})

	return findings, nil
}

// checkAgentsMD verifies AGENTS.md exists and contains required content.
func checkAgentsMD(root string) []Finding {
	var findings []Finding
	path := filepath.Join(root, "AGENTS.md")

	// Check existence
	data, err := os.ReadFile(path)
	if err != nil {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "missing",
			Message: "AGENTS.md not found",
		})
		return findings
	}

	content := string(data)
	lower := strings.ToLower(content)

	// Required content checks for AGENTS.md
	requiredContent := []struct {
		pattern string
		desc    string
	}{
		{"docs/doctrine/agent-assisted-development.md", "agent-assisted-development.md reference"},
		{"docs/doctrine/go-only.md", "go-only.md reference"},
		{"docs/factory/llm-friendliness.md", "llm-friendliness.md reference"},
		{"no python", "No Python rule"},
		{"bash is glue", "Bash is glue rule"},
		{"make factorize", "make factorize instruction"},
		{"make gate", "make gate instruction"},
		{"go test ./...", "go test instruction"},
		{"go vet ./...", "go vet instruction"},
		{"cgo_enabled=0 go build", "CGO_ENABLED=0 go build instruction"},
		{"do not force-push", "Do not force-push rule"},
	}

	for _, req := range requiredContent {
		if !strings.Contains(lower, req.pattern) {
			findings = append(findings, Finding{
				Path:    path,
				Kind:    "missing_content",
				Message: fmt.Sprintf("missing required content: %s", req.desc),
			})
		}
	}

	// Check line count limit (<=160 lines)
	lineCount := countLines(content)
	if lineCount > 160 {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "too_long",
			Message: fmt.Sprintf("%d lines > 160 (LLM context bloat)", lineCount),
		})
	}

	return findings
}

// checkClineRules verifies .clinerules/leamas.md exists and contains required content.
func checkClineRules(root string) []Finding {
	var findings []Finding
	path := filepath.Join(root, ".clinerules", "leamas.md")

	// Check existence
	data, err := os.ReadFile(path)
	if err != nil {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "missing",
			Message: ".clinerules/leamas.md not found",
		})
		return findings
	}

	content := string(data)
	lower := strings.ToLower(content)

	// Required content checks for .clinerules/leamas.md
	requiredContent := []struct {
		pattern string
		desc    string
	}{
		{"agents.md", "AGENTS.md reference"},
		{"no python", "No Python rule"},
		{"bash only", "Bash only rule"},
		{"make factorize", "make factorize instruction"},
		{"make gate", "make gate instruction"},
		{"do not force-push", "Do not force-push rule"},
	}

	for _, req := range requiredContent {
		if !strings.Contains(lower, req.pattern) {
			findings = append(findings, Finding{
				Path:    path,
				Kind:    "missing_content",
				Message: fmt.Sprintf("missing required content: %s", req.desc),
			})
		}
	}

	// Check line count limit (<=120 lines)
	lineCount := countLines(content)
	if lineCount > 120 {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "too_long",
			Message: fmt.Sprintf("%d lines > 120 (LLM context bloat)", lineCount),
		})
	}

	return findings
}

// checkPolicyDoc verifies docs/factory/agent-context-files.md exists.
func checkPolicyDoc(root string) []Finding {
	var findings []Finding
	path := filepath.Join(root, "docs", "factory", "agent-context-files.md")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "missing",
			Message: "docs/factory/agent-context-files.md not found",
		})
	}

	return findings
}

// countLines counts the number of lines in content.
func countLines(content string) int {
	scanner := bufio.NewScanner(bytes.NewReader([]byte(content)))
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines
}
