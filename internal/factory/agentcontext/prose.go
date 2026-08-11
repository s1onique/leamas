// Protected-operation prose scanner.
//
// The prose scanner rejects imperative grants of "protected"
// operations that appear in persistent agent-context prose without an
// explicit authority guard in the same paragraph. This complements
// the structured authority contract (contract.go) by enforcing the
// same rules at the prose level rather than relying on a finite
// blacklist of complete bad sentences.
//
// Model:
//
//   verb class
//   + protected operation
//   + presence/absence of authority guard
//
// The scanner detects common imperative forms (run, execute, invoke,
// use, ...) and common protected operations (make factorize, make
// gate-dupcode, make gate, repository gate, git commit, git push,
// git tag, force-push, ...). If a paragraph mentions a protected
// operation and contains no guard phrase and no negation, it is
// rejected.
//
// Fenced code blocks, the structured authority contract block, and
// inline code spans are stripped before scanning so that example
// commands and the contract itself are not misinterpreted.
//
// Before substring matching, the scanner normalizes whitespace: any
// run of internal whitespace is collapsed to a single space. This
// makes detection robust against double spaces, tabs, and other
// variations.

package agentcontext

import (
	"regexp"
	"strings"
)

// ProtectedOpKind is a stable identifier for one protected operation.
type ProtectedOpKind string

const (
	OpMakeFactorize  ProtectedOpKind = "make_factorize"
	OpMakeGateDupcode ProtectedOpKind = "make_gate_dupcode"
	OpMakeGate       ProtectedOpKind = "make_gate"
	OpRepositoryGate ProtectedOpKind = "repository_gate"
	OpGitCommit      ProtectedOpKind = "git_commit"
	OpGitPush        ProtectedOpKind = "git_push"
	OpGitTag         ProtectedOpKind = "git_tag"
	OpForcePush      ProtectedOpKind = "force_push"
	OpHistoryRewrite ProtectedOpKind = "history_rewrite"
)

// ProtectedOp binds one logical operation to the textual forms it
// may appear in. Forms are matched case-insensitively as substrings
// against normalized paragraph text.
type ProtectedOp struct {
	Kind  ProtectedOpKind
	Forms []string
}

// ProtectedOps is the canonical list of protected operations.
var ProtectedOps = []ProtectedOp{
	{Kind: OpMakeFactorize, Forms: []string{"make factorize"}},
	{Kind: OpMakeGateDupcode, Forms: []string{"make gate-dupcode"}},
	{Kind: OpMakeGate, Forms: []string{"make gate"}},
	{Kind: OpRepositoryGate, Forms: []string{"repository gate"}},
	{Kind: OpGitCommit, Forms: []string{"git commit", "commit the changes", "commit completed work", "create a commit", "make a commit"}},
	{Kind: OpGitPush, Forms: []string{"git push", "push the commit", "push changes", "push successful work"}},
	{Kind: OpGitTag, Forms: []string{"git tag", "tag the commit", "create a tag", "tag successful", "tag every", "tag all"}},
	{Kind: OpForcePush, Forms: []string{"force-push", "force push"}},
	{Kind: OpHistoryRewrite, Forms: []string{"rewrite history", "rebase", "amend a commit"}},
}

// GuardPhraseKind is a stable identifier for one authority-guard
// phrase.
type GuardPhraseKind string

const (
	GuardExplicitlyAuthorizedByACT GuardPhraseKind = "explicitly_authorized_by_act"
	GuardWhenDelegatedByACT        GuardPhraseKind = "when_delegated_by_act"
	GuardOnlyWhenACTAuth           GuardPhraseKind = "only_when_act_authorizes"
	GuardUnlessACTAuthorizes       GuardPhraseKind = "unless_act_authorizes"
	GuardWhenACTDelegates          GuardPhraseKind = "when_act_delegates"
	GuardOnlyIfDelegated           GuardPhraseKind = "only_if_delegated"
)

// GuardPhrase binds one logical guard to its textual forms.
type GuardPhrase struct {
	Kind  GuardPhraseKind
	Forms []string
}

// GuardPhrases is the canonical list of authority-guard phrases.
var GuardPhrases = []GuardPhrase{
	{Kind: GuardExplicitlyAuthorizedByACT, Forms: []string{
		"explicitly authorized by the current act",
		"explicitly authorized by the act",
		"explicitly authorizes that exact command",
		"explicitly authorizes",
		"explicitly authorized",
	}},
	{Kind: GuardWhenDelegatedByACT, Forms: []string{
		"when delegated by the current act",
		"when delegated by the act",
		"only when delegated",
		"when delegated",
		"only when the act delegates",
		"when the act delegates",
		"the current act delegates",
		"current act delegates",
		"the act delegates",
	}},
	{Kind: GuardOnlyWhenACTAuth, Forms: []string{
		"only when the current act authorizes",
		"only when the act authorizes",
		"only when that verification tier is explicitly authorized",
		"only when the act delegates",
		"only when delegated",
	}},
	{Kind: GuardUnlessACTAuthorizes, Forms: []string{
		"unless the current act explicitly authorizes",
		"unless the act explicitly authorizes",
		"unless explicitly authorized",
		"unless the current act authorizes",
		"unless the act authorizes",
	}},
	{Kind: GuardWhenACTDelegates, Forms: []string{
		"when the act delegates",
		"the act delegates",
		"only when the act delegates",
		"current act delegates",
		"the current act delegates",
		"delegates commit authority",
		"delegates tag authority",
		"delegated by the current act",
		"delegated by the act",
	}},
	{Kind: GuardOnlyIfDelegated, Forms: []string{
		"only if delegated",
		"only when delegated",
		"only when that verification tier is explicitly authorized",
	}},
}

// negationPatterns are textual NEGATION forms.
var negationPatterns = []string{
	"do not ",
	"don't ",
	"never ",
	"is not ",
	"are not ",
	"isn't ",
	"aren't ",
}

// imperativeVerbPattern matches common imperative verbs. Used as a hint.
var imperativeVerbPattern = regexp.MustCompile(`(?i)\b(run|execute|invoke|use|perform|do|run the|execute the|invoke the|use the)\b`)

// whitespaceCollapse collapses runs of internal whitespace into a single
// space and lowercases the string. This is the primary normalization
// applied before scanning.
func whitespaceCollapse(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				sb.WriteByte(' ')
			}
			prevSpace = true
			continue
		}
		prevSpace = false
		sb.WriteRune(r)
	}
	return strings.TrimSpace(sb.String())
}

// stripExcludedRegions removes fenced code blocks, the authority
// contract block, and inline backtick code spans from the prose
// before scanning.
func stripExcludedRegions(content string) string {
	beginIdx := strings.Index(content, ContractBeginMarker)
	endIdx := strings.Index(content, ContractEndMarker)
	if beginIdx != -1 && endIdx != -1 && endIdx > beginIdx {
		content = content[:beginIdx] + content[endIdx+len(ContractEndMarker):]
	} else if beginIdx != -1 {
		content = content[:beginIdx]
	}

	var sb2 strings.Builder
	inFence := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		sb2.WriteString(line)
		sb2.WriteString("\n")
	}
	content = sb2.String()

	content = stripInlineCode(content)

	return content
}

// stripInlineCode replaces content inside single-backtick spans with
// spaces (preserving offsets and newlines).
func stripInlineCode(content string) string {
	var sb strings.Builder
	sb.Grow(len(content))
	inCode := false
	for i := 0; i < len(content); i++ {
		c := content[i]
		if c == '`' {
			inCode = !inCode
			sb.WriteByte(' ')
			continue
		}
		if inCode {
			sb.WriteByte(' ')
		} else {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// SplitParagraphs splits content into paragraph units.
func SplitParagraphs(content string) []string {
	parts := strings.Split(content, "\n\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// ParagraphHasImperativeVerb reports whether a paragraph contains at
// least one imperative verb form.
func ParagraphHasImperativeVerb(lowerPara string) bool {
	return imperativeVerbPattern.MatchString(lowerPara)
}

// paragraphMentionsProtectedOp reports whether the normalized
// paragraph contains any protected-operation form.
func paragraphMentionsProtectedOp(lowerPara string) (ProtectedOpKind, string) {
	for _, op := range ProtectedOps {
		for _, form := range op.Forms {
			if strings.Contains(lowerPara, form) {
				return op.Kind, form
			}
		}
	}
	return "", ""
}

// paragraphHasGuard reports whether the normalized paragraph contains
// at least one guard phrase.
func paragraphHasGuard(lowerPara string) bool {
	for _, guard := range GuardPhrases {
		for _, form := range guard.Forms {
			if strings.Contains(lowerPara, form) {
				return true
			}
		}
	}
	return false
}

// paragraphHasNegation reports whether the normalized paragraph
// contains at least one negation form.
func paragraphHasNegation(lowerPara string) bool {
	for _, neg := range negationPatterns {
		if strings.Contains(lowerPara, neg) {
			return true
		}
	}
	return false
}

// ProseFinding is a single unguarded occurrence.
type ProseFinding struct {
	Path    string
	Kind    string
	Op      ProtectedOpKind
	Excerpt string
}

// FindUnguardedProtectedOps scans content for unguarded imperative
// grants of protected operations.
func FindUnguardedProtectedOps(path, content string) []ProseFinding {
	cleaned := stripExcludedRegions(content)
	var findings []ProseFinding
	for _, para := range SplitParagraphs(cleaned) {
		normalized := whitespaceCollapse(strings.ToLower(para))
		if normalized == "" {
			continue
		}
		op, _ := paragraphMentionsProtectedOp(normalized)
		if op == "" {
			continue
		}
		if paragraphHasGuard(normalized) {
			continue
		}
		if paragraphHasNegation(normalized) {
			continue
		}
		findings = append(findings, ProseFinding{
			Path:    path,
			Kind:    "unguarded_protected_operation",
			Op:      op,
			Excerpt: truncate(para, 160),
		})
	}
	return findings
}

// truncate returns at most n runes of s, ending with "..." if it was
// actually truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}