// Protected-operation prose scanner.
//
// The scanner rejects imperative grants of "protected" operations
// that appear in persistent agent-context prose without a
// per-operation authority guard or negation. The model is:
//
//   imperative intent
//   + protected operation
//   + absence of operation-bound guard
//   + absence of operation-bound negation
//
// A paragraph is split into normative sentence/clause units. Each
// unit is independently classified. A guard or negation in one
// unit does NOT authorize an unrelated protected operation in
// another unit.
//
// Inline Markdown code spans preserve their content; the backtick
// delimiters are removed but the wrapped text remains in the
// scanned prose. This makes "Run make gate" and "Run `make gate`"
// classify identically. Fenced code blocks are excluded because
// they represent a separately delimited example region.
//
// The structured authority contract block is also excluded from
// scanning.
//
// A paragraph is treated as a single logical unit: soft-wrapped
// lines within a paragraph are joined with single spaces before
// sentence units are extracted. Bullets are stripped at the
// start of each line before the join.

package agentcontext

import (
	"regexp"
	"strings"
)

// ProtectedOpKind is a stable identifier for one protected operation.
type ProtectedOpKind string

const (
	OpMakeFactorize   ProtectedOpKind = "make_factorize"
	OpMakeGateDupcode ProtectedOpKind = "make_gate_dupcode"
	OpMakeGate        ProtectedOpKind = "make_gate"
	OpRepositoryGate  ProtectedOpKind = "repository_gate"
	OpGitCommit       ProtectedOpKind = "git_commit"
	OpGitPush         ProtectedOpKind = "git_push"
	OpGitTag          ProtectedOpKind = "git_tag"
	OpForcePush       ProtectedOpKind = "force_push"
	OpHistoryRewrite  ProtectedOpKind = "history_rewrite"
)

// ProtectedOp binds one logical operation to the textual forms it
// may appear in.
type ProtectedOp struct {
	Kind  ProtectedOpKind
	Forms []string
}

// ProtectedOps is the canonical list of protected operations.
// Plural forms are listed when the noun form is common in the
// codebase (e.g. "git tags"). A word-boundary check prevents the
// singular form from accidentally matching the plural.
var ProtectedOps = []ProtectedOp{
	{Kind: OpMakeFactorize, Forms: []string{"make factorize"}},
	{Kind: OpMakeGateDupcode, Forms: []string{"make gate-dupcode"}},
	{Kind: OpMakeGate, Forms: []string{"make gate"}},
	{Kind: OpRepositoryGate, Forms: []string{"repository gate"}},
	{Kind: OpGitCommit, Forms: []string{"git commit", "commit the changes", "commit completed work", "create a commit", "make a commit"}},
	{Kind: OpGitPush, Forms: []string{"git push", "push the commit", "push changes", "push successful work"}},
	{Kind: OpGitTag, Forms: []string{"git tag", "git tags", "tag the commit", "create a tag", "tag successful", "tag every", "tag all"}},
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

// imperativeVerbs is the canonical list of imperative/directive
// verbs used to recognize imperative intent in a unit.
var imperativeVerbs = []string{
	"run", "execute", "invoke", "use", "perform",
	"commit", "push", "tag", "create", "make",
	"rewrite", "amend", "rebase",
}

// descriptiveVerbs indicate a descriptive sentence. A unit that
// contains any of these ANYWHERE is treated as descriptive.
var descriptiveVerbs = []string{
	"is", "are", "was", "were", "can", "must", "may",
	"should", "would", "does", "do", "validates",
	"performs", "records", "defines", "means",
	"represents", "includes", "supports", "documents",
	"indicates", "provides", "provide", "allows",
}

// imperativeVerbPattern matches common imperative verbs.
var imperativeVerbPattern = regexp.MustCompile(`(?i)\b(run|execute|invoke|use|perform|commit|push|tag|create|make|rewrite|amend|rebase)\b`)

// ParagraphHasImperativeVerb reports whether the unit contains at
// least one imperative verb form.
func ParagraphHasImperativeVerb(lowerPara string) bool {
	return imperativeVerbPattern.MatchString(lowerPara)
}

// unitHasDescriptiveVerb reports whether the unit contains any
// descriptive verb (anywhere).
func unitHasDescriptiveVerb(lowerUnit string) bool {
	words := strings.Fields(lowerUnit)
	for _, w := range words {
		for _, d := range descriptiveVerbs {
			if w == d {
				return true
			}
		}
	}
	return false
}

// formMatchesAtStart reports whether the form matches at the start of
// the unit AND is followed by a word boundary (space, punctuation, or
// end of string). This prevents the singular form from matching the
// plural form (e.g. "git tag" must not match "git tags").
func formMatchesAtStart(lowerUnit, form string) bool {
	if !strings.HasPrefix(lowerUnit, form) {
		return false
	}
	after := lowerUnit[len(form):]
	if after == "" {
		return true
	}
	c := after[0]
	return c == ' ' || c == '\t' || c == '\n' || c == '.' || c == ',' ||
		c == ';' || c == '!' || c == '?' || c == ':'
}

// unitHasImperativeIntent reports whether the unit expresses an
// imperative directive (vs descriptive prose).
//
// The check has two phases:
//
//  1. If the unit starts with a protected-op form (followed by a
//     word boundary), the unit is imperative UNLESS at least one
//     descriptive verb appears anywhere in the unit. This catches
//     both bare commands ("make gate") and subject-complement
//     patterns ("make factorize is a Tier-3 command", "Git commit
//     authority is independent ...").
//
//  2. Otherwise, the unit is imperative if and only if its first
//     word is a recognized imperative verb.
func unitHasImperativeIntent(lowerUnit string) bool {
	words := strings.Fields(lowerUnit)
	if len(words) == 0 {
		return false
	}

	// Phase 1: protected-op form at start.
	for _, op := range ProtectedOps {
		for _, form := range op.Forms {
			if !formMatchesAtStart(lowerUnit, form) {
				continue
			}
			// Bare form is imperative.
			if lowerUnit == form {
				return true
			}
			// If the unit has any descriptive verb, treat as
			// descriptive (catches "X is Y", "X provides Y").
			if unitHasDescriptiveVerb(lowerUnit) {
				return false
			}
			// Imperative: "Run make gate", "make gate" alone.
			// Note: "make" alone as first word is in imperativeVerbs
			// but we already ruled out the form case, so falling
			// through is fine.
			return true
		}
	}

	// Phase 2: imperative verb at the start of the unit.
	first := words[0]
	for _, v := range imperativeVerbs {
		if first == v {
			return true
		}
	}
	return false
}

// protectedOpMatch is one protected operation mentioned in a unit.
type protectedOpMatch struct {
	op   ProtectedOpKind
	form string
}

// unitsMentionProtectedOps returns all protected operations in the
// unit, in deterministic insertion order.
func unitsMentionProtectedOps(lowerUnit string) []protectedOpMatch {
	seen := make(map[ProtectedOpKind]bool)
	var matches []protectedOpMatch
	for _, op := range ProtectedOps {
		for _, form := range op.Forms {
			if strings.Contains(lowerUnit, form) {
				if !seen[op.Kind] {
					matches = append(matches, protectedOpMatch{op: op.Kind, form: form})
					seen[op.Kind] = true
				}
			}
		}
	}
	return matches
}

// unitHasGuard reports whether the unit contains a guard phrase.
func unitHasGuard(lowerUnit string) bool {
	for _, guard := range GuardPhrases {
		for _, form := range guard.Forms {
			if strings.Contains(lowerUnit, form) {
				return true
			}
		}
	}
	return false
}

// unitHasNegation reports whether the unit contains a negation.
func unitHasNegation(lowerUnit string) bool {
	for _, neg := range negationPatterns {
		if strings.Contains(lowerUnit, neg) {
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

// joinParagraphLines joins soft-wrapped lines in a paragraph into
// one logical string, stripping Markdown bullet prefixes along the
// way. Each line is treated as a separate item (bullet or sentence).
func joinParagraphLines(paragraph string) string {
	var lines []string
	for _, raw := range strings.Split(paragraph, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lines = append(lines, stripBulletPrefix(line))
	}
	return strings.Join(lines, " ")
}

// FindUnguardedProtectedOps scans content for unguarded imperative
// grants of protected operations. Each paragraph is treated as a
// single logical unit (soft-wrapped lines joined), then split into
// sentence/clause units. Each unit is independently classified.
func FindUnguardedProtectedOps(path, content string) []ProseFinding {
	stripped := stripContractBlock(content)
	stripped = stripFencedCodeBlocks(stripped)
	stripped = stripBacktickDelimiters(stripped)

	var findings []ProseFinding
	for _, paragraph := range splitParagraphs(stripped) {
		joined := joinParagraphLines(paragraph)
		for _, unit := range splitIntoUnits(joined) {
			normalized := whitespaceCollapse(strings.ToLower(unit))
			if normalized == "" {
				continue
			}
			if !unitHasImperativeIntent(normalized) {
				continue
			}
			matches := unitsMentionProtectedOps(normalized)
			if len(matches) == 0 {
				continue
			}
			hasGuard := unitHasGuard(normalized)
			hasNegation := unitHasNegation(normalized)
			if hasGuard || hasNegation {
				continue
			}
			for _, m := range matches {
				findings = append(findings, ProseFinding{
					Path:    path,
					Kind:    "unguarded_protected_operation",
					Op:      m.op,
					Excerpt: truncate(unit, 160),
				})
			}
		}
	}
	return findings
}
