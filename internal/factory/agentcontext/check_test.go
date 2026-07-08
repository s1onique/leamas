package agentcontext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckRepo(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create minimal valid AGENTS.md
	agentsMD := `# AGENTS.md

## Project

Leamas is a verification tool.

## Read First

- docs/doctrine/agent-assisted-development.md
- docs/doctrine/go-only.md
- docs/factory/llm-friendliness.md

## Non-Negotiable Rules

- No Python anywhere.
- Bash is glue only.
- make factorize
- make gate
- go test ./...
- go vet ./...
- CGO_ENABLED=0 go build
- Do not force-push.

## Close Reports

Every closed ACT must record exact commands.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(agentsMD), 0644); err != nil {
		t.Fatal(err)
	}

	// Create minimal valid .clinerules/leamas.md
	clineMD := `# Cline Rules for Leamas

Follow AGENTS.md first.

## Language Boundary

- No Python anywhere.
- Bash only for tiny glue.
- make factorize
- make gate
- Do not force-push.

## Verification

Run make factorize and make gate.
`
	if err := os.MkdirAll(filepath.Join(tmpDir, ".clinerules"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".clinerules", "leamas.md"), []byte(clineMD), 0644); err != nil {
		t.Fatal(err)
	}

	// Create minimal policy doc
	policyDoc := `# Agent Context Files

Leamas uses checked-in agent context files.

## Files

| File | Purpose |
|------|---------|
| AGENTS.md | Tool-agnostic instructions |
| .clinerules/leamas.md | Cline-specific rules |
`
	if err := os.MkdirAll(filepath.Join(tmpDir, "docs", "factory"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "docs", "factory", "agent-context-files.md"), []byte(policyDoc), 0644); err != nil {
		t.Fatal(err)
	}

	// Run check
	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	// Should have no findings for valid files
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d: %+v", len(findings), findings)
	}
}

func TestCheckRepo_MissingAgentsMD(t *testing.T) {
	tmpDir := t.TempDir()

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "missing" && f.Path == filepath.Join(tmpDir, "AGENTS.md") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for missing AGENTS.md")
	}
}

func TestCheckRepo_MissingClineRules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only AGENTS.md
	agentsMD := `# AGENTS.md
Leamas.
No Python.
Bash is glue.
make factorize.
make gate.
go test ./...
go vet ./...
CGO_ENABLED=0 go build.
Do not force-push.
docs/doctrine/agent-assisted-development.md
docs/doctrine/go-only.md
docs/factory/llm-friendliness.md
`
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(agentsMD), 0644); err != nil {
		t.Fatal(err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "missing" && f.Path == filepath.Join(tmpDir, ".clinerules", "leamas.md") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for missing .clinerules/leamas.md")
	}
}

func TestCheckRepo_AgentsMDTooLong(t *testing.T) {
	tmpDir := t.TempDir()

	// Create AGENTS.md with too many lines (>160)
	lines := []string{"# AGENTS.md\n"}
	for i := 0; i < 170; i++ {
		lines = append(lines, "Line of content.\n")
	}
	content := ""
	for _, l := range lines {
		content += l
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "too_long" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for AGENTS.md too long")
	}
}

func TestCheckRepo_ClineRulesTooLong(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid AGENTS.md
	agentsMD := `# AGENTS.md
Leamas.
No Python.
Bash is glue.
make factorize.
make gate.
go test ./...
go vet ./...
CGO_ENABLED=0 go build.
Do not force-push.
docs/doctrine/agent-assisted-development.md
docs/doctrine/go-only.md
docs/factory/llm-friendliness.md
`
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(agentsMD), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .clinerules/leamas.md with too many lines (>120)
	lines := []string{"# Cline Rules for Leamas\n", "Follow AGENTS.md first.\n"}
	for i := 0; i < 125; i++ {
		lines = append(lines, "Line of content.\n")
	}
	content := ""
	for _, l := range lines {
		content += l
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".clinerules"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".clinerules", "leamas.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "too_long" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for .clinerules/leamas.md too long")
	}
}

func TestCheckRepo_MissingPolicyDoc(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid AGENTS.md and .clinerules/leamas.md
	agentsMD := `# AGENTS.md
Leamas.
No Python.
Bash is glue.
make factorize.
make gate.
go test ./...
go vet ./...
CGO_ENABLED=0 go build.
Do not force-push.
docs/doctrine/agent-assisted-development.md
docs/doctrine/go-only.md
docs/factory/llm-friendliness.md
`
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(agentsMD), 0644); err != nil {
		t.Fatal(err)
	}

	clineMD := `# Cline Rules for Leamas
Follow AGENTS.md first.
No Python.
Bash only.
make factorize.
make gate.
Do not force-push.
`
	if err := os.MkdirAll(filepath.Join(tmpDir, ".clinerules"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".clinerules", "leamas.md"), []byte(clineMD), 0644); err != nil {
		t.Fatal(err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "missing" && f.Path == filepath.Join(tmpDir, "docs", "factory", "agent-context-files.md") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for missing docs/factory/agent-context-files.md")
	}
}
