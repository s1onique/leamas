// Protected-operation occurrence detection.
//
// This file provides positional occurrence detection for protected
// operations. Each occurrence carries the operation kind, the matched
// form, and the byte offsets in the (lower-cased, whitespace-normalized)
// unit. Word boundaries are enforced at both ends of a match so that
// "git tag" cannot match across "git tags" and "make gate" cannot
// match across "make gatekeeper".

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
	// Left boundary
	if start > 0 && isOccurrenceWordChar(lowerUnit[start-1]) {
		return false
	}
	// Right boundary
	end := start + len(form)
	if end < len(lowerUnit) && isOccurrenceWordChar(lowerUnit[end]) {
		return false
	}
	return true
}

// findProtectedOccurrences returns all positional protected-operation
// occurrences in the unit, sorted by start position. The unit MUST
// already be lower-cased and whitespace-normalized.
func findProtectedOccurrences(lowerUnit string) []ProtectedOccurrence {
	var occurrences []ProtectedOccurrence
	seen := make(map[ProtectedOpKind]bool)
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
					if !seen[op.Kind] {
						occurrences = append(occurrences, ProtectedOccurrence{
							Op:    op.Kind,
							Form:  form,
							Start: absStart,
							End:   absStart + len(form),
						})
						seen[op.Kind] = true
					}
				}
				pos = absStart + len(form)
				if pos >= len(lowerUnit) {
					break
				}
			}
		}
	}
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
