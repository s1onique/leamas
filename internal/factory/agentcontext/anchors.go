// Anchor definitions for the agent-context authority-delegation contract.
//
// The anchors.go file is the single source of truth for what
// persistent agent context files MUST contain (required) and MUST NOT
// contain (forbidden / unguarded). The check.go file enforces them
// deterministically against AGENTS.md and .clinerules/leamas.md.
//
// Phrases are matched case-insensitively against the whole file
// content. Anchors are intentionally explicit lower-case substrings so
// the verifier cannot be fooled by adversarial phrasing such as
// "Always run make gate." vs "Do not run make gate unless explicitly
// authorized.".

package agentcontext

// Anchor is one canonical lower-case phrase that the agent-context
// verifier checks for in a persistent agent-context file.
//
// Required = true  => the phrase MUST appear (lowercase substring).
// Required = false => the phrase MUST NOT appear (forbidden).
type Anchor struct {
	// ID is a stable identifier used in finding messages and tests.
	ID string
	// Phrase is the lowercase substring checked against the file.
	Phrase string
	// Required is true for required anchors and false for forbidden
	// (unguarded) anchors.
	Required bool
}

// AgentsMDRequiredAnchors are the canonical required anchors for
// AGENTS.md. They encode the authority-delegation contract that the
// tool-agnostic agent contract MUST express.
var AgentsMDRequiredAnchors = []Anchor{
	{ID: "doctrine_ref", Phrase: "docs/doctrine/agent-assisted-development.md", Required: true},
	{ID: "authority_delegation_ref", Phrase: "docs/doctrine/agent-authority-delegation.md", Required: true},
	{ID: "no_python", Phrase: "no python", Required: true},
	{ID: "bash_is_glue", Phrase: "bash is glue", Required: true},
	{ID: "follow_current_act", Phrase: "follow the current act", Required: true},
	{ID: "do_not_infer_commit_authority", Phrase: "do not infer commit authority from permission to edit or test", Required: true},
	{ID: "do_not_infer_push_authority", Phrase: "do not infer push authority from permission to commit", Required: true},
	{ID: "do_not_infer_tag_authority", Phrase: "do not infer tag authority from permission to commit", Required: true},
	{ID: "no_factorize_unless_authorized", Phrase: "do not run make factorize unless explicitly authorized", Required: true},
	{ID: "no_dupcode_unless_authorized", Phrase: "do not run make gate-dupcode unless explicitly authorized", Required: true},
	{ID: "no_gate_unless_authorized", Phrase: "do not run make gate unless explicitly authorized", Required: true},
	{ID: "not_run_is_valid", Phrase: "report the command as not run", Required: true},
	{ID: "checkpoint_distinguished", Phrase: "checkpoint", Required: true},
	{ID: "frozen_plan_authority", Phrase: "factory closure plan", Required: true},
	{ID: "gate_fast_required", Phrase: "gate-fast", Required: true},
	{ID: "go_test_umbrella", Phrase: "go test ./...", Required: true},
	{ID: "go_vet_umbrella", Phrase: "go vet ./...", Required: true},
	{ID: "go_build_trimpath", Phrase: "cgo_enabled=0 go build", Required: true},
	{ID: "do_not_force_push", Phrase: "do not force-push", Required: true},
}

// AgentsMDForbiddenAnchors are canonical forbidden (unguarded)
// anchors for AGENTS.md. If any of these appear, the file grants
// authority implicitly and MUST be rejected.
var AgentsMDForbiddenAnchors = []Anchor{
	{ID: "always_run_factorize", Phrase: "always run make factorize", Required: false},
	{ID: "always_run_dupcode", Phrase: "always run make gate-dupcode", Required: false},
	{ID: "always_run_gate", Phrase: "always run make gate", Required: false},
	{ID: "factorize_before_commit", Phrase: "make factorize before every commit", Required: false},
	{ID: "gate_before_commit", Phrase: "make gate before every commit", Required: false},
	{ID: "always_commit", Phrase: "always commit", Required: false},
	{ID: "commit_all_changes_after", Phrase: "commit all changes after", Required: false},
	{ID: "commit_when_tests_pass", Phrase: "commit when tests pass", Required: false},
	{ID: "automatically_push", Phrase: "automatically push", Required: false},
	{ID: "push_successful_work", Phrase: "push successful work", Required: false},
	{ID: "tag_after_commit", Phrase: "tag the commit automatically", Required: false},
}

// ClineMDRequiredAnchors are the canonical required anchors for
// .clinerules/leamas.md. They encode the editor/Cline-specific
// authority-delegation contract.
var ClineMDRequiredAnchors = []Anchor{
	{ID: "agents_md_ref", Phrase: "agents.md", Required: true},
	{ID: "no_python", Phrase: "no python", Required: true},
	{ID: "bash_only", Phrase: "bash only", Required: true},
	{ID: "current_act_is_authoritative", Phrase: "the current act is authoritative", Required: true},
	{ID: "no_factorize_in_cline", Phrase: "make factorize", Required: true},
	{ID: "no_dupcode_in_cline", Phrase: "make gate-dupcode", Required: true},
	{ID: "no_gate_in_cline", Phrase: "make gate", Required: true},
	{ID: "cline_explicit_authorize", Phrase: "explicitly authorizes that exact command", Required: true},
	{ID: "report_not_run_rule", Phrase: "report it as not run", Required: true},
	{ID: "do_not_infer_commit", Phrase: "do not infer git commit", Required: true},
	{ID: "checkpoint_distinguished", Phrase: "checkpoint", Required: true},
	{ID: "do_not_force_push", Phrase: "do not force-push", Required: true},
}

// ClineMDForbiddenAnchors are canonical forbidden (unguarded)
// anchors for .clinerules/leamas.md.
var ClineMDForbiddenAnchors = []Anchor{
	{ID: "always_run_factorize", Phrase: "always run make factorize", Required: false},
	{ID: "always_run_dupcode", Phrase: "always run make gate-dupcode", Required: false},
	{ID: "always_run_gate", Phrase: "always run make gate", Required: false},
	{ID: "always_commit", Phrase: "always commit", Required: false},
	{ID: "commit_all_changes_after", Phrase: "commit all changes after", Required: false},
	{ID: "commit_when_tests_pass", Phrase: "commit when tests pass", Required: false},
	{ID: "automatically_push", Phrase: "automatically push", Required: false},
	{ID: "push_successful_work", Phrase: "push successful work", Required: false},
	{ID: "force_push_successful", Phrase: "force-push successful", Required: false},
}
