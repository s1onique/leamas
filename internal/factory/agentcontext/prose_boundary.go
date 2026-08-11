// Hard sentence boundary detection.
//
// Hard normative boundaries separate independent instructions.
// They are consumed as part of the unital scan, AFTER the normalizer
// has stripped fenced code blocks and the authority contract block.
// The boundary detector is intentionally simple:
//
//   .   at end-of-input, OR followed by whitespace. NOT a boundary
//       when the dot is followed by alphanumeric characters (i.e.,
//       internal to a file extension like ".json" or ".md" or a
//       version like "v1.2").
//   !   always a boundary.
//   ?   always a boundary.
//   ;   always a boundary.
//
// Detection of the next-character case is no longer used:
// boundaries are case-independent. "Do not force-push. commit
// completed work." must split into two normative units just like
// "Do not force-push. Commit completed work." does.

package agentcontext

import "strings"

// splitIntoUnits splits a paragraph into normative sentence/clause
// units. Hard boundaries are:
//
//   .   at end-of-input, OR followed by whitespace. NOT a boundary
//       when the dot is followed by alphanumeric characters (file
//       extension, version).
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
			// A period is a hard boundary when:
			//   - it is at end of input, OR
			//   - it is followed by whitespace.
			//
			// A period is NOT a boundary when followed by
			// alphanumeric characters (file extension like
			// ".json" or version like "v1.2").
			if i == len(runes)-1 {
				isBoundary = true
			} else if i+1 < len(runes) && (runes[i+1] == ' ' || runes[i+1] == '\t' || runes[i+1] == '\n' || runes[i+1] == '\r') {
				isBoundary = true
			} else {
				// Period followed by alphanumeric: file extension
				// or version. NOT a boundary.
			}
		}
		if isBoundary {
			chunk := strings.TrimSpace(string(runes[start : i+1]))
			if chunk != "" {
				units = append(units, chunk)
			}
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
