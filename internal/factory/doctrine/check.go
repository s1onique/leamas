// Package doctrine provides verification for doctrine documents.
package doctrine

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// RequiredDoctrineFiles lists all required doctrine documents (files with Agent Contract).
// Note: README.md is documentation, not doctrine, so it's excluded from Agent Contract checks.
var RequiredDoctrineFiles = []string{
	"docs/doctrine/agent-assisted-development.md",
	"docs/doctrine/local-first.md",
	"docs/doctrine/web-first.md",
	"docs/doctrine/go-only.md",
	"docs/doctrine/single-binary.md",
	"docs/doctrine/no-enterprise-swamp.md",
	"docs/doctrine/not-a-gateway.md",
	"docs/doctrine/verification-witness.md",
	"docs/doctrine/factory-meta-loop.md",
}

// RequiredAgentContractSections lists required Agent Contract sections.
var RequiredAgentContractSections = []string{
	"## Agent Contract",
	"### Always",
	"### Never",
	"### Ask / Escalate",
	"### Verification Hooks",
}

// SpecialChecks defines special content checks for specific doctrine files.
var SpecialChecks = []struct {
	Path    string
	Pattern string
	Desc    string
}{
	{
		Path:    "docs/doctrine/README.md",
		Pattern: "agent-assisted-development.md",
		Desc:    "README must link agent-assisted-development.md",
	},
	{
		Path:    "docs/doctrine/not-a-gateway.md",
		Pattern: "local witness proxy",
		Desc:    "not-a-gateway must permit local witness proxy",
	},
	{
		Path:    "docs/doctrine/not-a-gateway.md",
		Pattern: "provider router",
		Desc:    "not-a-gateway must forbid provider router behavior",
	},
	{
		Path:    "docs/doctrine/not-a-gateway.md",
		Pattern: "model control plane",
		Desc:    "not-a-gateway must forbid model control plane",
	},
	{
		Path:    "docs/doctrine/verification-witness.md",
		Pattern: "Separate observation from evaluation",
		Desc:    "verification-witness must require observation/evaluation separation",
	},
	{
		Path:    "docs/doctrine/verification-witness.md",
		Pattern: "LLM output as proof",
		Desc:    "verification-witness must forbid treating LLM output as proof",
	},
}

// CheckInventory verifies all required doctrine files exist.
func CheckInventory(root string) []checks.Finding {
	var findings []checks.Finding

	for _, file := range RequiredDoctrineFiles {
		path := filepath.Join(root, file)
		if !checks.FileExists(path) {
			findings = append(findings, checks.Finding{
				Path:     file,
				Kind:     "missing",
				Message:  "required doctrine file not found",
				Severity: checks.SeverityError,
			})
		}
	}

	return findings
}

// CheckAgentContracts verifies all doctrine files have Agent Contract sections.
func CheckAgentContracts(root string) []checks.Finding {
	var findings []checks.Finding

	for _, file := range RequiredDoctrineFiles {
		path := filepath.Join(root, file)
		data := checks.ReadFile(path)
		if data == nil {
			// Skip files that don't exist - inventory check handles that
			continue
		}

		content := string(data)
		for _, section := range RequiredAgentContractSections {
			if !strings.Contains(content, section) {
				findings = append(findings, checks.Finding{
					Path:     file,
					Kind:     "missing_section",
					Message:  fmt.Sprintf("missing Agent Contract section: %s", section),
					Severity: checks.SeverityError,
				})
			}
		}
	}

	return findings
}

// CheckSpecialContent verifies special content requirements.
func CheckSpecialContent(root string) []checks.Finding {
	var findings []checks.Finding

	for _, check := range SpecialChecks {
		path := filepath.Join(root, check.Path)
		data := checks.ReadFile(path)
		if data == nil {
			findings = append(findings, checks.Finding{
				Path:     check.Path,
				Kind:     "missing",
				Message:  fmt.Sprintf("file not found for special check: %s", check.Desc),
				Severity: checks.SeverityError,
			})
			continue
		}

		content := string(data)
		if !strings.Contains(content, check.Pattern) {
			findings = append(findings, checks.Finding{
				Path:     check.Path,
				Kind:     "missing_content",
				Message:  check.Desc,
				Severity: checks.SeverityError,
			})
		}
	}

	return findings
}

// CheckRepo runs all doctrine checks.
func CheckRepo(root string) []checks.Finding {
	var findings []checks.Finding

	findings = append(findings, CheckInventory(root)...)
	findings = append(findings, CheckAgentContracts(root)...)
	findings = append(findings, CheckSpecialContent(root)...)

	checks.SortFindings(findings)
	return findings
}
