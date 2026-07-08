package doctrine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

func TestCheckInventory(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Test with missing files
	findings := CheckInventory(tmpDir)
	if len(findings) != len(RequiredDoctrineFiles) {
		t.Errorf("expected %d findings for empty dir, got %d", len(RequiredDoctrineFiles), len(findings))
	}

	// Create one of the required files
	requiredPath := filepath.Join(tmpDir, "docs/doctrine/agent-assisted-development.md")
	if err := os.MkdirAll(filepath.Dir(requiredPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(requiredPath, []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should have one less error
	findings = CheckInventory(tmpDir)
	if len(findings) != len(RequiredDoctrineFiles)-1 {
		t.Errorf("expected %d findings, got %d", len(RequiredDoctrineFiles)-1, len(findings))
	}
}

func TestCheckAgentContracts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a doctrine file with missing sections
	docsDir := filepath.Join(tmpDir, "docs/doctrine")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// File without Agent Contract sections
	incompleteFile := filepath.Join(docsDir, "local-first.md")
	if err := os.WriteFile(incompleteFile, []byte("# Local First\n\nJust some content."), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckAgentContracts(tmpDir)
	if len(findings) != len(RequiredAgentContractSections) {
		t.Errorf("expected %d findings for incomplete file, got %d", len(RequiredAgentContractSections), len(findings))
	}

	// Create file with all sections
	completeFile := filepath.Join(docsDir, "agent-assisted-development.md")
	content := `## Agent Contract

### Always
- Do something

### Never
- Don't do something

### Ask / Escalate
- Ask if unsure

### Verification Hooks
- Some hook
`
	if err := os.WriteFile(completeFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	findings = CheckAgentContracts(tmpDir)
	// CheckAgentContracts skips files that don't exist.
	// Only local-first.md exists and is incomplete, so we expect 5 findings (all its missing sections)
	if len(findings) != len(RequiredAgentContractSections) {
		t.Errorf("expected %d findings for incomplete local-first.md, got %d", len(RequiredAgentContractSections), len(findings))
	}
}

func TestCheckSpecialContent(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with missing file
	findings := CheckSpecialContent(tmpDir)
	if len(findings) == 0 {
		t.Error("expected findings for missing special content files")
	}

	// Create not-a-gateway.md with required content
	notGateway := filepath.Join(tmpDir, "docs/doctrine/not-a-gateway.md")
	content := `# Not a Gateway

Leamas is not an LLM gateway, provider router, or model control plane.

## Agent Contract

### Always
- Be local

### Never
- Don't route

### Ask / Escalate
- Ask if unsure

### Verification Hooks
- None

## Local Witness Proxy

Leamas may implement a local witness proxy.
`
	if err := os.MkdirAll(filepath.Dir(notGateway), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(notGateway, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	findings = CheckSpecialContent(tmpDir)
	// After adding not-a-gateway.md, we satisfy 2 of its checks (local witness proxy + provider router)
	// Still need to check: provider router (already satisfied), model control plane (already satisfied),
	// verification-witness (2 checks), README (1 check)
	// Total special checks: 6, not-a-gateway has 3 checks (local witness proxy, provider router, model control plane)
	// After adding not-a-gateway with all 3 checks satisfied: 6 - 3 = 3 remaining
	expectedFindings := len(SpecialChecks) - 3 // not-a-gateway has 3 special checks
	if len(findings) != expectedFindings {
		t.Errorf("expected %d findings after adding not-a-gateway, got %d", expectedFindings, len(findings))
	}
}

func TestCheckRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create all required doctrine files with proper content
	for _, file := range RequiredDoctrineFiles {
		path := filepath.Join(tmpDir, file)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create file with Agent Contract sections
		content := `# Doctrine

## Agent Contract

### Always
- Always do this

### Never
- Never do that

### Ask / Escalate
- Ask if unsure

### Verification Hooks
- hook1
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Add special content for not-a-gateway and verification-witness
	notGateway := filepath.Join(tmpDir, "docs/doctrine/not-a-gateway.md")
	notGatewayContent := `# Not a Gateway

## Agent Contract

### Always
- Be local

### Never
- Don't route

### Ask / Escalate
- Ask

### Verification Hooks
- None

local witness proxy
provider router
model control plane
`
	if err := os.WriteFile(notGateway, []byte(notGatewayContent), 0644); err != nil {
		t.Fatal(err)
	}

	verificationWitness := filepath.Join(tmpDir, "docs/doctrine/verification-witness.md")
	vwContent := `# Verification Witness

## Agent Contract

### Always
- Verify

### Never
- Don't prove

### Ask / Escalate
- Ask

### Verification Hooks
- None

Separate observation from evaluation
LLM output as proof
`
	if err := os.WriteFile(verificationWitness, []byte(vwContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Add README link
	readme := filepath.Join(tmpDir, "docs/doctrine/README.md")
	readmeContent := `# Doctrine

## Agent Contract

### Always
- Always

### Never
- Never

### Ask / Escalate
- Ask

### Verification Hooks
- None

See [Agent Assisted Development](agent-assisted-development.md)
`
	if err := os.WriteFile(readme, []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckRepo(tmpDir)
	if len(findings) != 0 {
		for _, f := range findings {
			t.Logf("unexpected finding: %s: %s: %s", f.Path, f.Kind, f.Message)
		}
		t.Errorf("expected no findings for complete doctrine, got %d", len(findings))
	}
}

func TestHasErrors(t *testing.T) {
	var findings []checks.Finding

	if checks.HasErrors(findings) {
		t.Error("empty findings should not have errors")
	}

	findings = append(findings, checks.Finding{
		Kind:     "test",
		Severity: checks.SeverityWarn,
	})

	if checks.HasErrors(findings) {
		t.Error("warning-only findings should not have errors")
	}

	findings = append(findings, checks.Finding{
		Kind:     "test",
		Severity: checks.SeverityError,
	})

	if !checks.HasErrors(findings) {
		t.Error("findings with error severity should have errors")
	}
}
