// Protected-operation prose scanner.
//
// The scanner rejects imperative grants of "protected" operations
// that appear in persistent agent-context prose without a
// per-occurrence authority guard or negation. The model is:
//
//   directive intent
//   + protected operation
//   + absence of operation-bound authority guard
//   + absence of operation-bound negation
//
// Classification is PER OCCURRENCE:
//
//   1. The paragraph is split into coarse clauses by ";", " but ",
//      " however, ", " however ", " and then ", " then ".
//   2. Each coarse clause is split on " and " ONLY when the clause
//      contains more than one directive component (failure to
//      disambiguate shared guards fails closed).
//   3. Each sub-clause is stripped of leading directive prefixes
//      (always, please, then, next, now, you must, before X,, after X,,
//      for X,, ...) and then classified by isDirective.
//   4. If a sub-clause is directive, protected-operation occurrences
//      in it are flagged UNLESS the sub-clause also contains a
//      per-sub-clause authority guard (with a named authority source)
//      or a negation.
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
// CORRECTION04: single-word forms ("commit", "push", "tag") are
// present so that operational directives like "commit completed work",
// "push the commit", "tag the commit" are detected. Word-boundary
// checks protect against false positives on nouns like "the commit" in
// descriptive prose (which is not flagged anyway because the
// directive check fails).
var ProtectedOps = []ProtectedOp{
	{Kind: OpMakeFactorize, Forms: []string{"make factorize"}},
	{Kind: OpMakeGateDupcode, Forms: []string{"make gate-dupcode"}},
	{Kind: OpMakeGate, Forms: []string{"make gate"}},
	{Kind: OpRepositoryGate, Forms: []string{"repository gate"}},
	{Kind: OpGitCommit, Forms: []string{"git commit", "commit the changes", "commit completed work", "create a commit", "make a commit", "commit"}},
	{Kind: OpGitPush, Forms: []string{"git push", "push the commit", "push changes", "push successful work", "push"}},
	{Kind: OpGitTag, Forms: []string{"git tag", "git tags", "tag the commit", "create a tag", "tag successful", "tag every", "tag all", "tag"}},
	{Kind: OpForcePush, Forms: []string{"force-push", "force push"}},
	{Kind: OpHistoryRewrite, Forms: []string{"rewrite history", "rebase", "amend a commit"}},
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
// verbs used to recognize imperative intent in a unit. These are
// intentionally limited to verbs that DIRECT an action, not verbs
// that identify a protected operation. "do" is included so that
// "do not run X" is recognised as a directive.
var imperativeVerbs = []string{
	"run", "execute", "invoke", "use", "perform",
	"do", "rewrite", "amend", "rebase",
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
var imperativeVerbPattern = regexp.MustCompile(`(?i)\b(run|execute|invoke|use|perform|rewrite|amend|rebase)\b`)

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

// scanSubClause returns the protected-operation occurrences that
// should be flagged in the given directive sub-clause. Sub-clauses
// with zero or one protected-operation occurrence are reported
// unless they are exempt by a guard or negation. Sub-clauses with
// MORE THAN ONE protected-operation occurrence fail closed
// (CORRECTION04): ambiguity in shared scope cannot be resolved
// without explicit per-operation authorization, so every occurrence
// is reported.
func scanSubClause(lowerSubClause string) []ProtectedOpKind {
	occs := findProtectedOccurrences(lowerSubClause)
	if len(occs) == 0 {
		return nil
	}
	if subClauseHasNegation(lowerSubClause) {
		return nil
	}
	if unitHasAuthoritySource(lowerSubClause) {
		return nil
	}
	ops := make([]ProtectedOpKind, 0, len(occs))
	for _, o := range occs {
		ops = append(ops, o.Op)
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
