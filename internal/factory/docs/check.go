// Package docs provides verification for factory documentation.
package docs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// RequiredFactoryDocs lists all required factory documentation files.
var RequiredFactoryDocs = []string{
	"docs/adr/0001-local-first-single-binary.md",
	"docs/adr/0002-go-only-for-v0.md",
	"docs/adr/0003-web-first-local-cockpit.md",
	"docs/adr/0004-no-oidc-until-shared-rig.md",
	"docs/adr/0005-not-an-llm-gateway.md",
	"docs/adr/0006-filesystem-run-bundles.md",
	"docs/adr/README.md",
	"docs/templates/act.md",
	"docs/templates/adr.md",
	"docs/templates/close-report.md",
	"docs/templates/reviewer-prompt.md",
	"docs/templates/epic.md",
	"docs/acts/.gitkeep",
	"docs/epics/.gitkeep",
	"docs/factory/tooling-boundaries.md",
	"docs/factory/llm-friendliness.md",
	"docs/factory/agent-context-files.md",
	"docs/factory/git-safety.md",
}

// RequiredADRSections lists required sections in ADR files.
var RequiredADRSections = []string{
	"## Status",
	"## Context",
	"## Decision",
}

// CheckInventory verifies all required factory docs exist.
func CheckInventory(root string) []checks.Finding {
	var findings []checks.Finding

	for _, file := range RequiredFactoryDocs {
		path := filepath.Join(root, file)
		if !checks.FileExists(path) {
			findings = append(findings, checks.Finding{
				Path:     file,
				Kind:     "missing",
				Message:  "required factory document not found",
				Severity: checks.SeverityError,
			})
		}
	}

	return findings
}

// CheckADRStructure verifies ADR files have required sections.
func CheckADRStructure(root string) []checks.Finding {
	var findings []checks.Finding

	adrDir := filepath.Join(root, "docs", "adr")

	// List all ADR files (0*.md, excluding README.md)
	entries, err := os.ReadDir(adrDir)
	if err != nil {
		if os.IsNotExist(err) {
			findings = append(findings, checks.Finding{
				Path:     "docs/adr",
				Kind:     "missing",
				Message:  "ADR directory not found",
				Severity: checks.SeverityError,
			})
			return findings
		}
		findings = append(findings, checks.Finding{
			Path:     "docs/adr",
			Kind:     "error",
			Message:  "cannot read ADR directory: " + err.Error(),
			Severity: checks.SeverityError,
		})
		return findings
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "README.md" {
			continue
		}
		// Only check ADR files (0*.md)
		if !strings.HasPrefix(name, "0") || !strings.HasSuffix(name, ".md") {
			continue
		}

		path := filepath.Join(adrDir, name)
		data := checks.ReadFile(path)
		if data == nil {
			findings = append(findings, checks.Finding{
				Path:     filepath.Join("docs/adr", name),
				Kind:     "missing",
				Message:  "cannot read ADR file",
				Severity: checks.SeverityError,
			})
			continue
		}

		content := string(data)
		for _, section := range RequiredADRSections {
			if !strings.Contains(content, section) {
				findings = append(findings, checks.Finding{
					Path:     filepath.Join("docs/adr", name),
					Kind:     "missing_section",
					Message:  "missing required section: " + section,
					Severity: checks.SeverityError,
				})
			}
		}
	}

	return findings
}

// CheckRepo runs all factory docs checks.
func CheckRepo(root string) []checks.Finding {
	var findings []checks.Finding

	findings = append(findings, CheckInventory(root)...)
	findings = append(findings, CheckADRStructure(root)...)

	checks.SortFindings(findings)
	return findings
}
