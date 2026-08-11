// Protected-operation prose scanner.
//
// The scanner rejects imperative grants of "protected" operations
// that appear in persistent agent-context prose without a
// per-operation authority guard or negation. The model is:
//
//   directive intent
//   + protected operation
//   + absence of operation-bound guard
//   + absence of operation-bound negation
//
// Classification is PER OCCURRENCE:
//
//   1. The paragraph is split into coarse clauses by ";", " but ",
//      " however, ", " however ", " and then ", " then ".
//   2. Each coarse clause is split on " and " ONLY when it contains
//      more than one protected operation (failure to disambiguate
//      shared guards fails closed).
//   3. Each sub-clause is stripped of leading directive prefixes
//      (always, please, then, next, now, you must, before X,, after X,,
//      for X,, ...) and then classified by isDirective.
//   4. If a sub-clause is directive, protected-operation occurrences
//      in it are flagged UNLESS the sub-clause also contains a
//      per-occurrence guard or negation.
//
// Inline backticks preserve their content (only the delimiter is
// removed). Fenced code blocks and the authority contract block are
// excluded from scanning.

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
// may appear in.
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
		"leamas factory",
		"factory close",
		"factory protocol",
		"closure protocol v1",
		"factory workflow",
		"factory command",
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
	// Auxiliary / modal / negation-head verbs that introduce a
	// directive clause even though they are not themselves a
	// protected-op verb.
	"do",
}

// descriptiveVerbs indicate a descriptive copula sentence. A unit
// that starts with a protected-op form AND has the next word in this
// list is treated as descriptive prose.
var descriptiveVerbs = []string{
	"is", "are", "was", "were",
	"provides", "provide",
	"has", "have", "had",
	"means", "represent", "represents",
	"includes", "include",
	"supports", "support",
	"validates", "validate",
	"performs", "perform",
	"documents", "document",
	"indicates", "indicate",
	"allows", "allow",
}

// imperativeVerbPattern matches common imperative verbs.
var imperativeVerbPattern = regexp.MustCompile(`(?i)\b(run|execute|invoke|use|perform|commit|push|tag|create|make|rewrite|amend|rebase)\b`)

// ParagraphHasImperativeVerb reports whether the unit contains at
// least one imperative verb form.
func ParagraphHasImperativeVerb(lowerPara string) bool {
	return imperativeVerbPattern.MatchString(lowerPara)
}

// ProseFinding is a single unguarded occurrence.
type ProseFinding struct {
	Path    string
	Kind    string
	Op      ProtectedOpKind
	Excerpt string
}

// subClauseHasGuard reports whether the sub-clause contains any
// guard phrase.
func subClauseHasGuard(lowerSubClause string) bool {
	for _, guard := range GuardPhrases {
		for _, form := range guard.Forms {
			if strings.Contains(lowerSubClause, form) {
				return true
			}
		}
	}
	return false
}

// subClauseHasNegation reports whether the sub-clause contains any
// negation form.
func subClauseHasNegation(lowerSubClause string) bool {
	for _, neg := range negationPatterns {
		if strings.Contains(lowerSubClause, neg) {
			return true
		}
	}
	return false
}

// imperativeVerbToProtectedOp maps a directive-first imperative
// verb to the protected operation it implicitly authorizes.
func imperativeVerbToProtectedOp(verb string) (ProtectedOpKind, bool) {
	switch verb {
	case "commit", "create":
		return OpGitCommit, true
	case "push":
		return OpGitPush, true
	case "tag":
		return OpGitTag, true
	case "make":
		return OpMakeGate, true
	case "rewrite":
		return OpHistoryRewrite, true
	}
	return "", false
}

// scanSubClause returns the protected-operation occurrences that
// should be flagged in the given directive sub-clause. If the
// sub-clause contains a guard or negation, no occurrences are flagged.
func scanSubClause(lowerSubClause string) []ProtectedOpKind {
	seen := make(map[ProtectedOpKind]bool)
	var ops []ProtectedOpKind
	// Positional protected-op forms.
	for _, o := range findProtectedOccurrences(lowerSubClause) {
		if !seen[o.Op] {
			seen[o.Op] = true
			ops = append(ops, o.Op)
		}
	}
	// First-word imperative verb that is itself a protected-op verb.
	words := strings.Fields(lowerSubClause)
	if len(words) > 0 {
		if op, ok := imperativeVerbToProtectedOp(words[0]); ok {
			if !seen[op] {
				seen[op] = true
				ops = append(ops, op)
			}
		}
	}
	if len(ops) == 0 {
		return nil
	}
	if subClauseHasGuard(lowerSubClause) {
		return nil
	}
	if subClauseHasNegation(lowerSubClause) {
		return nil
	}
	return ops
}

// scanUnit walks the unit: strip directive prefixes, split into
// coarse clauses, then sub-clauses, then scan each sub-clause.
func scanUnit(lowerUnit string) []ProtectedOpKind {
	stripped := stripAllDirectivePrefixes(lowerUnit)
	if stripped == "" {
		return nil
	}
	coarseClauses := splitCoarseClauses(stripped)
	var flagged []ProtectedOpKind
	for _, clause := range coarseClauses {
		strippedClause := stripAllDirectivePrefixes(clause)
		if strippedClause == "" {
			continue
		}
		if !isDirective(strippedClause) {
			continue
		}
		subClauses := splitCoordination(strippedClause)
		for _, sub := range subClauses {
			flagged = append(flagged, scanSubClause(sub)...)
		}
	}
	return flagged
}

// FindUnguardedProtectedOps scans content for unguarded imperative
// grants of protected operations.
func FindUnguardedProtectedOps(path, content string) []ProseFinding {
	stripped := stripContractBlock(content)
	stripped = stripFencedCodeBlocks(stripped)
	stripped = stripBacktickDelimiters(stripped)

	var findings []ProseFinding
	seen := make(map[string]bool)
	for _, paragraph := range splitParagraphs(stripped) {
		joined := joinParagraphLines(paragraph)
		for _, unit := range splitIntoUnits(joined) {
			normalized := whitespaceCollapse(strings.ToLower(unit))
			if normalized == "" {
				continue
			}
			flagged := scanUnit(normalized)
			for _, op := range flagged {
				key := normalized + "|" + string(op)
				if seen[key] {
					continue
				}
				seen[key] = true
				findings = append(findings, ProseFinding{
					Path:    path,
					Kind:    "unguarded_protected_operation",
					Op:      op,
					Excerpt: truncate(unit, 160),
				})
			}
		}
	}
	return findings
}

// joinParagraphLines joins soft-wrapped lines in a paragraph into
// one logical string, stripping Markdown bullet prefixes along the
// way.
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