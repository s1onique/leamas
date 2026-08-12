package agentcontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CanonicalContractBlock is the common contract block used by both
// AGENTS.md and .clinerules/leamas.md valid fixtures. It encodes the
// doctrinally required defaults.
const CanonicalContractBlock = `<!-- LEAMAS:AUTHORITY-CONTRACT:BEGIN -->
authority_source=act
edit=act
focused_verification=act
affected_verification=act
expensive_verification=explicit_exact
commit=explicit
push=explicit
tag=explicit
history_rewrite=forbidden
checkpoint_is_git_publication=false
unauthorized_expensive_result=NOT_RUN
frozen_closure_plan_authority=true
<!-- LEAMAS:AUTHORITY-CONTRACT:END -->
`

// validAgentsMD returns a fixture that satisfies every layer of
// verification: structured contract, prose guards, presence anchors,
// and line limit.
func validAgentsMD() string {
	return `# AGENTS.md

## Project

Leamas is a local-first, web-first, Go-only, single-binary verification witness.

## Read First

- docs/doctrine/agent-assisted-development.md
- docs/doctrine/agent-authority-delegation.md
- docs/doctrine/go-only.md

## Non-Negotiable Rules

- No Python anywhere.
- Bash is glue only.
- Do not force-push.

## Authority Delegation

Follow the current ACT's explicit authority.

Do not infer commit authority from permission to edit or test.
Do not infer push authority from permission to deploy.
Do not infer tag stewardship from permission to deploy.

Do not run make factorize unless explicitly authorized by the current ACT.
Do not run make gate-dupcode unless explicitly authorized by the current ACT.
Do not run make gate unless explicitly authorized by the current ACT.

When not authorized, report the command as not run.

Editor checkpoints, restore points, and Compare operations are not Git commits and do not grant Git publication authority.

Successful tests do not imply commit authority. Commit only when the ACT delegates this kind of action.

` + CanonicalContractBlock + `

## Required Verification

Focused checks explicitly required by the ACT remain allowed.

When Go code exists or changes, also run:

- go test ./...
- go vet ./...
- CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
`
}

// validClineMD returns a fixture that satisfies every layer of
// verification for .clinerules/leamas.md.
func validClineMD() string {
	return `# Cline Rules for Leamas

Follow AGENTS.md first.

## Language Boundary

- No Python anywhere.
- Bash only for tiny glue and Git hooks.

## Execution Authority

The current ACT is authoritative.

Never run make factorize, make gate-dupcode, or make gate in Cline/editor context unless the current ACT explicitly authorizes that exact command.

When not authorized, report it as not run.

Do not infer commit authority from permission to edit or test.
Do not infer push authority from permission to edit or test.
Do not infer tag stewardship from permission to edit or test.
Do not infer history-rewrite authority from permission to edit or test.

Cline checkpoints are not Git commits and do not grant Git publication authority.

` + CanonicalContractBlock + `

## Verification

A refusal from an expensive gate is not run, not pass, and must never be reported as successful verification.

## Git Safety

Do not force-push. Prefer forward corrective commits.
`
}

// writeFixture writes content to relPath under tmp and ensures parent
// directories exist.
func writeFixture(t *testing.T, tmp, relPath, content string) {
	t.Helper()
	full := filepath.Join(tmp, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

// writeValidFixtures writes a valid AGENTS.md, .clinerules/leamas.md,
// and the policy doc under tmp.
func writeValidFixtures(t *testing.T, tmp string) {
	t.Helper()
	writeFixture(t, tmp, "AGENTS.md", validAgentsMD())
	writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), validClineMD())
	writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"),
		"# Agent Context Files\nLeamas uses checked-in agent context files.\n")
}

func TestCheckRepo_Valid(t *testing.T) {
	tmp := t.TempDir()
	writeValidFixtures(t, tmp)

	findings, err := CheckRepo(tmp)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}
	if len(findings) != 0 {
		var msgs []string
		for _, f := range findings {
			msgs = append(msgs, f.Path+": "+f.Kind+": "+f.Message)
		}
		t.Fatalf("expected no findings, got: %s", strings.Join(msgs, "\n"))
	}
}

func TestCheckRepo_MissingAgentsMD(t *testing.T) {
	tmp := t.TempDir()
	writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), validClineMD())
	writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"), "policy")

	findings, err := CheckRepo(tmp)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}

	foundMissing := false
	for _, f := range findings {
		if f.Kind == "missing" && strings.HasSuffix(f.Path, "AGENTS.md") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Fatalf("expected missing finding for AGENTS.md, got: %+v", findings)
	}
}

func TestCheckRepo_MissingClineRules(t *testing.T) {
	tmp := t.TempDir()
	writeFixture(t, tmp, "AGENTS.md", validAgentsMD())
	writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"), "policy")

	findings, err := CheckRepo(tmp)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}

	foundMissing := false
	for _, f := range findings {
		if f.Kind == "missing" && strings.HasSuffix(f.Path, filepath.Join(".clinerules", "leamas.md")) {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Fatalf("expected missing finding for .clinerules/leamas.md, got: %+v", findings)
	}
}

func TestCheckRepo_AgentsMDTooLong(t *testing.T) {
	tmp := t.TempDir()
	writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), validClineMD())
	writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"), "policy")

	var sb strings.Builder
	sb.WriteString(validAgentsMD())
	for i := 0; i < 200; i++ {
		sb.WriteString("filler line to push over the line limit\n")
	}
	writeFixture(t, tmp, "AGENTS.md", sb.String())

	findings, err := CheckRepo(tmp)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "too_long" && strings.HasSuffix(f.Path, "AGENTS.md") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected too_long finding for AGENTS.md, got: %+v", findings)
	}
}

func TestCheckRepo_ClineRulesTooLong(t *testing.T) {
	tmp := t.TempDir()
	writeFixture(t, tmp, "AGENTS.md", validAgentsMD())
	writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"), "policy")

	var sb strings.Builder
	sb.WriteString(validClineMD())
	for i := 0; i < 200; i++ {
		sb.WriteString("filler line to push over the line limit\n")
	}
	writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), sb.String())

	findings, err := CheckRepo(tmp)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "too_long" && strings.HasSuffix(f.Path, filepath.Join(".clinerules", "leamas.md")) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected too_long finding for .clinerules/leamas.md, got: %+v", findings)
	}
}

func TestCheckRepo_MissingPolicyDoc(t *testing.T) {
	tmp := t.TempDir()
	writeFixture(t, tmp, "AGENTS.md", validAgentsMD())
	writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), validClineMD())

	findings, err := CheckRepo(tmp)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "missing" && strings.HasSuffix(f.Path, filepath.Join("docs", "factory", "agent-context-files.md")) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing finding for policy doc, got: %+v", findings)
	}
}
