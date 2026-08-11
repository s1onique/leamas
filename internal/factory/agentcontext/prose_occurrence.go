// Protected-operation occurrence detection.
//
// This file provides positional occurrence detection for protected
// operations. Each occurrence carries the operation kind, the matched
// form, and the byte offsets in the (lower-cased, whitespace-normalized)
// unit. Word boundaries are enforced at both ends of a match so that
// "git tag" cannot match across "git tags" and "make gate" cannot
// match across "make gatekeeper".
//
// CORRECTION05: occurrence collection is canonical. When multiple
// protected-op forms overlap in the same textual span (e.g. "commit
// completed work" matches both "commit completed work" AND "commit"),
// the longest non-overlapping match is retained per operation kind.
// This prevents aliases from manufacturing duplicate logical
// occurrences.

package agentcontext

import (
	"sort"
	"strings"
)

// ProtectedOccurrence is one positional mention of a protected
// operation in a unit.
type ProtectedOccurrence struct {
	Op    ProtectedOpKind
	Form  string
	Start int // byte offset in normalized unit
	End   int // byte offset (exclusive)
}

// isOccurrenceWordChar reports whether c is a character that would
// make a match extend across a word boundary (i.e. NOT a boundary).
func isOccurrenceWordChar(c byte) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	if c == '_' || c == '-' {
		return true
	}
	return false
}

// formMatchesAtRange reports whether the form matches the substring
// at [start, start+len(form)) AND is bracketed by word boundaries.
func formMatchesAtRange(lowerUnit string, form string, start int) bool {
	if start < 0 || start+len(form) > len(lowerUnit) {
		return false
	}
	if lowerUnit[start:start+len(form)] != form {
		return false
	}
	if start > 0 && isOccurrenceWordChar(lowerUnit[start-1]) {
		return false
	}
	end := start + len(form)
	if end < len(lowerUnit) && isOccurrenceWordChar(lowerUnit[end]) {
		return false
	}
	return true
}

// findProtectedOccurrences returns every canonical positional
// protected-operation occurrence in the unit, sorted by start
// position. Overlapping aliases (e.g. "commit completed work" and
// "commit" at the same position) are collapsed to the longest form.
// The unit MUST already be lower-cased and whitespace-normalized.
func findProtectedOccurrences(lowerUnit string) []ProtectedOccurrence {
	// Collect all candidate matches, then resolve overlaps.
	type candidate struct {
		op   ProtectedOpKind
		form string
		start int
		end   int
	}
	var candidates []candidate
	for _, op := range ProtectedOps {
		for _, form := range op.Forms {
			pos := 0
			for {
				idx := strings.Index(lowerUnit[pos:], form)
				if idx == -1 {
					break
				}
				absStart := pos + idx
				if formMatchesAtRange(lowerUnit, form, absStart) {
					candidates = append(candidates, candidate{
						op:    op.Kind,
						form:  form,
						start: absStart,
						end:   absStart + len(form),
					})
				}
				pos = absStart + len(form)
				if pos >= len(lowerUnit) {
					break
				}
			}
		}
	}

	// Sort by start ascending, then by length descending, then by
	// operation kind for stability.
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].start != candidates[j].start {
			return candidates[i].start < candidates[j].start
		}
		li := candidates[i].end - candidates[i].start
		lj := candidates[j].end - candidates[j].start
		if li != lj {
			return li > lj
		}
		return candidates[i].op < candidates[j].op
	})

	// Resolve overlaps:
	//   - Candidates are sorted by start ascending, then length
	//     descending (longest match first).
	//   - For each candidate, if it overlaps an already-accepted
	//     occurrence AND shares the same operation kind, it is
	//     dropped as an alias.
	//   - Cross-operation-kind overlaps are also dropped (the
	//     earlier/longer candidate wins).
	//   - Aliases at the same span are deduplicated to the longest
	//     form.
	var occurrences []ProtectedOccurrence
	for _, c := range candidates {
		overlaps := false
		for _, o := range occurrences {
			if !(c.end <= o.Start || c.start >= o.End) {
				overlaps = true
				break
			}
		}
		if overlaps {
			continue
		}
		occurrences = append(occurrences, ProtectedOccurrence{
			Op:    c.op,
			Form:  c.form,
			Start: c.start,
			End:   c.end,
		})
	}
	// Final sort by start, then by op.
	sort.Slice(occurrences, func(i, j int) bool {
		if occurrences[i].Start != occurrences[j].Start {
			return occurrences[i].Start < occurrences[j].Start
		}
		return occurrences[i].Op < occurrences[j].Op
	})
	return occurrences
}

// occurrencesByOp groups occurrences by op kind.
func occurrencesByOp(occs []ProtectedOccurrence) map[ProtectedOpKind]int {
	out := make(map[ProtectedOpKind]int, len(occs))
	for _, o := range occs {
		out[o.Op]++
	}
	return out
}
