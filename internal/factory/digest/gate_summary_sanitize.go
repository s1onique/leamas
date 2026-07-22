// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"strings"
	"unicode/utf8"
)

// sanitizeLine sanitizes a single line for stable rendering.
// It replaces line breaks and tabs with spaces, collapses whitespace,
// trims edges, preserves UTF-8, and truncates without splitting a rune.
func sanitizeLine(s string) string {
	if s == "" {
		return ""
	}

	// Replace CR, LF, and tab with space
	s = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '\t' {
			return ' '
		}
		return r
	}, s)

	// Collapse repeated spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}

	// Trim edges
	s = strings.TrimSpace(s)

	// Truncate to 240 bytes without splitting a UTF-8 rune
	const maxBytes = 240
	if len(s) > maxBytes {
		// Find the last valid rune boundary
		for len(s) > maxBytes {
			_, size := utf8.DecodeLastRuneInString(s)
			s = s[:len(s)-size]
		}
	}

	return s
}
