// Directive detection and clause segmentation.
//
// This file implements the small deterministic grammar used to
// classify persistent agent-context prose. It recognises bounded
// directive prefixes (always, please, then, next, now, you must,
// before X,, after X,, for X,, ...), normalises them away, and
// then splits the remainder into normative clauses bounded by
// semi-colons, but/however, and coordinating "and" (only when more
// than one protected operation is present in the clause).
//
// The scanner does NOT attempt to understand general English. It
// applies a deliberately restricted grammar.

package agentcontext

import (
	"regexp"
	"strings"
)

// imperativePrefixes are simple space-bounded directive prefixes
// that the scanner strips from the start of a unit.
var imperativePrefixes = []string{
	"always ",
	"please ",
	"then ",
	"next, ",
	"next ",
	"now ",
	"finally ",
	"first, ",
	"first ",
	"you must ",
	"you should ",
	"agents must ",
	"agents should ",
	"also ",
	"plus ",
}

// beforeAfterForPrefixPatterns match bounded phrases such as
// "before finishing, " or "after tests, " or "for final validation, ".
// The pattern requires a comma after the bounded phrase.
var (
	beforePrefixPattern = regexp.MustCompile(`(?i)^before\s+[^,]+,\s+`)
	afterPrefixPattern  = regexp.MustCompile(`(?i)^after\s+[^,]+,\s+`)
	forPrefixPattern   = regexp.MustCompile(`(?i)^for\s+[^,]+,\s+`)
)

// stripDirectivePrefix strips a single bounded directive prefix from
// the start of lowerUnit and returns the remaining text plus a flag
// indicating whether a prefix was stripped. If no prefix matches,
// lowerUnit is returned unchanged with stripped=false.
func stripDirectivePrefix(lowerUnit string) (string, bool) {
	for _, p := range imperativePrefixes {
		if strings.HasPrefix(lowerUnit, p) {
			return strings.TrimSpace(lowerUnit[len(p):]), true
		}
	}
	if loc := beforePrefixPattern.FindStringIndex(lowerUnit); loc != nil {
		return strings.TrimSpace(lowerUnit[loc[1]:]), true
	}
	if loc := afterPrefixPattern.FindStringIndex(lowerUnit); loc != nil {
		return strings.TrimSpace(lowerUnit[loc[1]:]), true
	}
	if loc := forPrefixPattern.FindStringIndex(lowerUnit); loc != nil {
		return strings.TrimSpace(lowerUnit[loc[1]:]), true
	}
	return lowerUnit, false
}

// stripAllDirectivePrefixes iteratively strips directive prefixes
// until no more match. Handles cases like "You must always run make gate".
func stripAllDirectivePrefixes(lowerUnit string) string {
	for i := 0; i < 8; i++ {
		next, stripped := stripDirectivePrefix(lowerUnit)
		if !stripped || next == lowerUnit {
			return lowerUnit
		}
		lowerUnit = next
	}
	return lowerUnit
}

// clauseBoundaryPattern matches clause boundaries (";", " but ",
// " however, ", " however ", " and then ", " then "). The capture
// group is the entire matched separator.
var clauseBoundaryPattern = regexp.MustCompile(`(?i)\s*;\s*|\s+but\s+|\s+however,?\s+|\s+and then\s+|\s+then\s+`)

// splitCoarseClauses splits the unit into coarse clauses using
// semi-colon, but, however, and then, then as boundaries.
func splitCoarseClauses(lowerUnit string) []string {
	parts := clauseBoundaryPattern.Split(lowerUnit, -1)
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// coordinationSplitPattern matches coordinating "and" used to join
// protected operations within a single clause.
var coordinationSplitPattern = regexp.MustCompile(`\s+and\s+`)

// countDirectiveComponents returns the number of directive
// components in a clause: positional protected-op forms PLUS any
// first-word imperative verb that maps to a protected operation.
func countDirectiveComponents(lowerClause string) int {
	count := len(findProtectedOccurrences(lowerClause))
	words := strings.Fields(lowerClause)
	if len(words) > 0 {
		if _, ok := imperativeVerbToProtectedOp(words[0]); ok {
			count++
		}
	}
	return count
}

// splitCoordination splits a clause on " and " when the clause
// contains more than one part AND at least TWO parts carry a
// directive component. Splitting on " and " within a single
// coordination unit that happens to list files (e.g. "commit
// both at a.json and b.md") would otherwise destroy the unit's
// guard binding. The rule therefore only splits when at least two
// parts each carry a directive component, which is the structural
// signature of two independent operations.
func splitCoordination(lowerClause string) []string {
	parts := coordinationSplitPattern.Split(lowerClause, -1)
	if len(parts) <= 1 {
		return []string{lowerClause}
	}
	cleanParts := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		cleanParts = append(cleanParts, p)
	}
	partsWithComponent := 0
	for _, p := range cleanParts {
		if countDirectiveComponents(p) > 0 {
			partsWithComponent++
		}
	}
	if partsWithComponent < 2 {
		return []string{lowerClause}
	}
	return cleanParts
}

// nextWordIsDescriptivePredicate reports whether the next word in
// lowerUnit after the given form is a descriptive-copula verb.
func nextWordIsDescriptivePredicate(lowerUnit, form string) bool {
	if !strings.HasPrefix(lowerUnit, form) {
		return false
	}
	after := strings.TrimSpace(lowerUnit[len(form):])
	if after == "" {
		return false
	}
	words := strings.Fields(after)
	if len(words) == 0 {
		return false
	}
	next := words[0]
	for _, d := range descriptiveVerbs {
		if next == d {
			return true
		}
	}
	return false
}

// isDirective reports whether the unit (after directive-prefix
// stripping) expresses a directive rather than descriptive prose.
//
// A unit is directive if:
//   - it starts with a protected-op form at a word boundary AND
//     the next word is not a descriptive-copula verb, OR
//   - its first word is a recognized imperative verb.
//
// The unit MUST already be lower-cased and whitespace-normalized.
func isDirective(lowerUnit string) bool {
	words := strings.Fields(lowerUnit)
	if len(words) == 0 {
		return false
	}

	// Try protected-op form at start.
	for _, op := range ProtectedOps {
		for _, form := range op.Forms {
			if !formMatchesAtStart(lowerUnit, form) {
				continue
			}
			if nextWordIsDescriptivePredicate(lowerUnit, form) {
				return false
			}
			return true
		}
	}

	// First word is a recognized imperative verb.
	first := words[0]
	for _, v := range imperativeVerbs {
		if first == v {
			return true
		}
	}
	return false
}

// formMatchesAtStart is retained here for syntactic locality with
// the directive detector. It is identical to the version in
// prose.go (kept for clarity of the directive dispatch).
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
