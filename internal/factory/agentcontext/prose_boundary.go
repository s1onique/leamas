// Hard sentence boundary detection.
//
// Hard normative boundaries separate independent instructions.
// They are consumed as part of the unital scan, AFTER the normalizer
// has stripped fenced code blocks and the authority contract block.
// The boundary detector is intentionally simple:
//
//   .   at end-of-string, OR followed by whitespace + uppercase
//       (start of a new sentence). NOT consumed when the dot
//       separates characters inside a file extension (e.g. ".json",
//       ".md") or version string (e.g. "v1.2").
//   !   always a boundary.
//   ?   always a boundary.
//   ;   always a boundary.
//
// The boundary rewrite preserves the boundary character on the
// preceding unit and inserts a single newline so the downstream
// scanner can split on newline.

package agentcontext

import "strings"

// isFileExtensionBoundary reports whether the period at position
// pos in s is a file-extension dot (e.g. ".json", ".md"). A file
// extension has lowercase letters immediately after the dot and is
// not followed by whitespace+uppercase.
func isFileExtensionBoundary(s string, pos int) bool {
	if pos < 0 || pos >= len(s) || s[pos] != '.' {
		return false
	}
	// Look ahead for the file extension pattern: letters/lowercase.
	rest := s[pos+1:]
	if rest == "" {
		return false
	}
	// A file extension must start with a letter. If immediately
	// followed by whitespace or uppercase, it's a sentence boundary.
	c := rest[0]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
		return false
	}
	// Must not be followed by whitespace+uppercase (which would be
	// a sentence boundary).
	for i := 1; i < len(rest); i++ {
		if rest[i] == ' ' || rest[i] == '\t' {
			// Check next char is uppercase => sentence boundary.
			if i+1 < len(rest) {
				nc := rest[i+1]
				if nc >= 'A' && nc <= 'Z' {
					return false
				}
			}
			// Otherwise, the rest is a lowercase extension followed
			// by whitespace — likely a file extension followed by
			// a new word. We treat this as a boundary (sentence
			// boundary because the period is at sentence end).
			return false
		}
	}
	// No whitespace found after the letters — looks like a file
	// extension or version. Treat as NOT a boundary.
	return true
}

// splitIntoUnits splits a paragraph into normative sentence/clause
// units. Hard boundaries are:
//
//   .   at end-of-string, OR followed by whitespace + uppercase
//       (start of a new sentence). NOT a boundary when the dot
//       separates characters inside a file extension (".json",
//       ".md") or version string ("v1.2").
//   !   always a boundary.
//   ?   always a boundary.
//   ;   always a boundary.
func splitIntoUnits(paragraph string) []string {
	if paragraph == "" {
		return nil
	}
	var units []string
	runes := []rune(paragraph)
	start := 0
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		var isBoundary bool
		switch c {
		case '!', '?', ';':
			isBoundary = true
		case '.':
			// A period is a boundary when followed by whitespace
			// + uppercase, or at end of string. It is NOT a
			// boundary when it is part of a file extension
			// (e.g. ".json").
			if i == len(runes)-1 {
				isBoundary = true
			} else {
				// Look ahead for whitespace+uppercase.
				j := i + 1
				for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t') {
					j++
				}
				if j < len(runes) {
					nc := runes[j]
					if nc >= 'A' && nc <= 'Z' {
						isBoundary = true
					}
				}
				// If next non-space char is lowercase, the period
				// is part of a file extension like ".json".
				// Not a boundary.
			}
			// Explicit file-extension check: if the period is NOT
			// followed by whitespace+uppercase and the next chars
			// are lowercase letters, treat as file extension.
			if !isBoundary {
				// Build a Go string from the runes slice for
				// the file-extension check.
				para := string(runes)
				if isFileExtensionBoundary(para, i) {
					// File extension: not a boundary.
				} else {
					// End of string or other case: still not
					// a boundary unless we've already set it.
				}
			}
		}
		if isBoundary {
			chunk := strings.TrimSpace(string(runes[start : i+1]))
			if chunk != "" {
				units = append(units, chunk)
			}
			// Continue past the boundary character.
			start = i + 1
		}
	}
	if start < len(runes) {
		chunk := strings.TrimSpace(string(runes[start:]))
		if chunk != "" {
			units = append(units, chunk)
		}
	}
	return units
}
