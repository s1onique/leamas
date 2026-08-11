// Presence anchors: simple, non-authority textual requirements that
// the agent-context files MUST satisfy. Presence anchors exist ONLY
// for orthogonal concerns such as doctrine references, language
// boundary, and line limits; they MUST NOT be used to express
// execution authority. Authority is enforced by contract.go and
// prose.go.

package agentcontext

import "strings"

// PresenceAnchor is a simple textual requirement that a persistent
// agent-context file MUST contain. Presence anchors are
// case-insensitive substring matches.
type PresenceAnchor struct {
	ID      string
	Phrase  string
	Comment string
}

// AgentsMDPresenceAnchors are the canonical presence requirements
// for AGENTS.md. They cover doctrine, language, line limit, and
// non-authority housekeeping; they explicitly do NOT cover
// execution authority.
var AgentsMDPresenceAnchors = []PresenceAnchor{
	{ID: "doctrine_ref", Phrase: "docs/doctrine/agent-assisted-development.md", Comment: "doctrine reference"},
	{ID: "authority_delegation_ref", Phrase: "docs/doctrine/agent-authority-delegation.md", Comment: "authority-delegation doctrine reference"},
	{ID: "contract_block_required", Phrase: "LEAMAS:AUTHORITY-CONTRACT:BEGIN", Comment: "structured authority contract begin marker"},
	{ID: "contract_block_end_required", Phrase: "LEAMAS:AUTHORITY-CONTRACT:END", Comment: "structured authority contract end marker"},
	{ID: "no_python", Phrase: "no python", Comment: "no-Python language rule"},
	{ID: "bash_is_glue", Phrase: "bash is glue", Comment: "Bash-is-glue language rule"},
	{ID: "do_not_force_push", Phrase: "do not force-push", Comment: "do-not-force-push Git safety rule"},
}

// ClineMDPresenceAnchors are the canonical presence requirements
// for .clinerules/leamas.md. Same scope as AgentsMDPresenceAnchors.
var ClineMDPresenceAnchors = []PresenceAnchor{
	{ID: "agents_md_ref", Phrase: "agents.md", Comment: "AGENTS.md reference"},
	{ID: "contract_block_required", Phrase: "LEAMAS:AUTHORITY-CONTRACT:BEGIN", Comment: "structured authority contract begin marker"},
	{ID: "contract_block_end_required", Phrase: "LEAMAS:AUTHORITY-CONTRACT:END", Comment: "structured authority contract end marker"},
	{ID: "no_python", Phrase: "no python", Comment: "no-Python language rule"},
	{ID: "bash_only", Phrase: "bash only", Comment: "Bash-only language rule"},
	{ID: "do_not_force_push", Phrase: "do not force-push", Comment: "do-not-force-push Git safety rule"},
}

// AgentsMDMaxLines is the canonical line limit for AGENTS.md.
const AgentsMDMaxLines = 160

// ClineMDMaxLines is the canonical line limit for .clinerules/leamas.md.
const ClineMDMaxLines = 120

// MissingPresenceAnchors returns the presence anchors that are NOT
// satisfied by content (case-insensitive substring match).
func MissingPresenceAnchors(content string, anchors []PresenceAnchor) []PresenceAnchor {
	lower := strings.ToLower(content)
	var missing []PresenceAnchor
	for _, a := range anchors {
		if !strings.Contains(lower, strings.ToLower(a.Phrase)) {
			missing = append(missing, a)
		}
	}
	return missing
}
