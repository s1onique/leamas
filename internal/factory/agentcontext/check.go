// Package agentcontext provides verification for agent instruction files.
// It ensures AGENTS.md and .clinerules/leamas.md exist and enforce the
// agent authority-delegation contract defined in
// docs/doctrine/agent-authority-delegation.md.
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

// CheckRepo verifies that agent context files exist and enforce the
// authority-delegation contract. Findings are sorted deterministically
// by (Path, Kind).
func CheckRepo(root string) ([]Finding, error) {
	var findings []Finding

	findings = append(findings, checkFile(root, "AGENTS.md", AgentsMDRequiredAnchors, AgentsMDForbiddenAnchors, 160)...)
	findings = append(findings, checkFile(root, filepath.Join(".clinerules", "leamas.md"), ClineMDRequiredAnchors, ClineMDForbiddenAnchors, 120)...)
	findings = append(findings, checkPolicyDoc(root)...)

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Kind < findings[j].Kind
	})

	return findings, nil
}

// checkFile enforces the canonical anchor contract against a single
// persistent agent-context file. It emits one finding per missing
// required anchor, one per forbidden (unguarded) anchor, one for
// missing files, and one for size violations.
func checkFile(root, relPath string, required, forbidden []Anchor, maxLines int) []Finding {
	var findings []Finding
	path := filepath.Join(root, relPath)

	data, err := os.ReadFile(path)
	if err != nil {
		return []Finding{{Path: path, Kind: "missing", Message: relPath + " not found"}}
	}
	content := string(data)
	lower := strings.ToLower(content)

	for _, anchor := range required {
		if !strings.Contains(lower, anchor.Phrase) {
			findings = append(findings, Finding{
				Path:    path,
				Kind:    "missing_authority",
				Message: fmt.Sprintf("missing required authority anchor: %s", anchor.ID),
			})
		}
	}

	for _, anchor := range forbidden {
		if strings.Contains(lower, anchor.Phrase) {
			findings = append(findings, Finding{
				Path:    path,
				Kind:    "unguarded_authority",
				Message: fmt.Sprintf("unguarded authority instruction: %s", anchor.ID),
			})
		}
	}

	if lineCount := countLines(content); lineCount > maxLines {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "too_long",
			Message: fmt.Sprintf("%d lines > %d (LLM context bloat)", lineCount, maxLines),
		})
	}

	return findings
}

// checkPolicyDoc verifies docs/factory/agent-context-files.md exists.
func checkPolicyDoc(root string) []Finding {
	path := filepath.Join(root, "docs", "factory", "agent-context-files.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Finding{{Path: path, Kind: "missing", Message: "docs/factory/agent-context-files.md not found"}}
	}
	return nil
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
