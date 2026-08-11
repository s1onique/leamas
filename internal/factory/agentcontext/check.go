// Package agentcontext provides verification for agent instruction
// files. It ensures AGENTS.md and .clinerules/leamas.md exist,
// include the structured authority contract, agree on shared
// authority semantics, and do not contain unguarded imperative
// grants of protected operations.
package agentcontext

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
//
// The check is composed of three layers:
//  1. structured contract (contract.go): authoritative machine-readable
//     contract that MUST be present, well-formed, and semantically
//     valid in both files, with shared semantics equal;
//  2. guarded prose scanner (prose.go): rejects unguarded imperative
//     grants of protected operations in human-readable prose;
//  3. presence anchors (presence.go): orthogonal non-authority
//     requirements such as doctrine references and line limits.
func CheckRepo(root string) ([]Finding, error) {
	var findings []Finding

	agentsContent, agentsExists, agentsFindings := checkAuthorityFile(root, "AGENTS.md", AgentsMDPresenceAnchors, AgentsMDMaxLines)
	findings = append(findings, agentsFindings...)

	clinePath := filepath.Join(".clinerules", "leamas.md")
	clineContent, clineExists, clineFindings := checkAuthorityFile(root, clinePath, ClineMDPresenceAnchors, ClineMDMaxLines)
	findings = append(findings, clineFindings...)

	// Cross-file consistency: shared contract semantics MUST agree.
	if agentsExists && clineExists {
		findings = append(findings, checkSharedContractAgreement(root, agentsContent, clineContent)...)
	}

	findings = append(findings, checkPolicyDoc(root)...)

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Kind < findings[j].Kind
	})

	return findings, nil
}

// checkAuthorityFile runs the full verification chain against one
// persistent agent-context file.
func checkAuthorityFile(root, relPath string, presenceAnchors []PresenceAnchor, maxLines int) (string, bool, []Finding) {
	path := filepath.Join(root, relPath)
	var findings []Finding

	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, []Finding{{Path: path, Kind: "missing", Message: relPath + " not found"}}
	}
	content := string(data)

	// 1) Structured contract.
	contract, contractErr := ParseContractBlock(content)
	if contractErr != nil {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "contract_malformed",
			Message: fmt.Sprintf("structured authority contract is malformed: %s", contractErr.Error()),
		})
	} else if !contract.IsValidContractSemantics() {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "contract_semantics_invalid",
			Message: "structured authority contract values violate doctrinal defaults",
		})
	}

	// 2) Prose scanner.
	proseFindings := FindUnguardedProtectedOps(path, content)
	for _, pf := range proseFindings {
		findings = append(findings, Finding{
			Path:    pf.Path,
			Kind:    pf.Kind,
			Message: fmt.Sprintf("unguarded %s in paragraph: %s", pf.Op, pf.Excerpt),
		})
	}

	// 3) Presence anchors.
	for _, a := range MissingPresenceAnchors(content, presenceAnchors) {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "missing_presence",
			Message: fmt.Sprintf("missing presence anchor: %s (%s)", a.ID, a.Comment),
		})
	}

	// 4) Line limit.
	if lines := countLines(content); lines > maxLines {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "too_long",
			Message: fmt.Sprintf("%d lines > %d (LLM context bloat)", lines, maxLines),
		})
	}

	return content, true, findings
}

// checkSharedContractAgreement parses the contracts in both files
// and emits findings when their shared semantics disagree.
func checkSharedContractAgreement(root, agentsContent, clineContent string) []Finding {
	var findings []Finding

	agentsContract, agentsErr := ParseContractBlock(agentsContent)
	clineContract, clineErr := ParseContractBlock(clineContent)

	if agentsErr != nil || clineErr != nil {
		// Per-file contract_malformed findings already emitted.
		return findings
	}
	if !SharedSemanticsEqual(agentsContract, clineContract) {
		findings = append(findings, Finding{
			Path:    filepath.Join(root, "AGENTS.md"),
			Kind:    "contract_shared_semantics_mismatch",
			Message: "shared authority semantics disagree between AGENTS.md and .clinerules/leamas.md",
		})
		findings = append(findings, Finding{
			Path:    filepath.Join(root, ".clinerules", "leamas.md"),
			Kind:    "contract_shared_semantics_mismatch",
			Message: "shared authority semantics disagree between AGENTS.md and .clinerules/leamas.md",
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
