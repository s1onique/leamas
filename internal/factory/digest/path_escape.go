// Package digest provides targeted digest generation for Git repositories.
//
// Path escaping for rendered digest sections.
//
// Git's `--name-status -z` output preserves paths verbatim, including
// bytes that would break a line-oriented Markdown rendering: tab,
// carriage return, line feed, backslash, and other control characters.
// The digest renders one record per line, so any of these bytes in a
// path would split a single manifest or changed-files entry across
// multiple visual lines and silently corrupt the digest.
//
// This file defines the canonical `PathEscape` form used everywhere a
// path is rendered into the digest. The encoding is a small,
// well-defined superset of the C `\\` / `\t` / `\n` / `\r` style used
// by Git's own UI: control bytes (0x00..0x1f, 0x7f) become `\xNN`,
// backslash becomes `\\`, tab becomes `\t`, newline becomes `\n`, and
// carriage return becomes `\r`. Printable UTF-8 (which includes the
// common leading-dash, space, and Unicode cases) passes through
// unchanged.
//
// The encoding is symmetric: ParseEscapedPath inverts it exactly.
package digest

import (
	"fmt"
	"strings"
)

// PathEscape returns the canonical escaped form of `path` for digest
// rendering. Bytes that would break the line-oriented Markdown
// rendering are escaped; printable UTF-8 (including spaces, tabs
// without a NUL/LF/CR pair, Unicode, and leading dashes) passes
// through unchanged.
//
// The function is a pure formatter and does not read or write any
// repository state. Callers may invoke it from renderers and tests.
func PathEscape(path string) string {
	if path == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(path))
	for i := 0; i < len(path); i++ {
		c := path[i]
		switch c {
		case '\\':
			b.WriteString(`\\`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if c < 0x20 || c == 0x7f {
				fmt.Fprintf(&b, `\x%02x`, c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

// ParseEscapedPath inverts PathEscape. It is the inverse function for
// the round-trip properties exercised by the test suite.
func ParseEscapedPath(escaped string) (string, error) {
	var b strings.Builder
	b.Grow(len(escaped))
	for i := 0; i < len(escaped); i++ {
		c := escaped[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		// Escape sequence: consume the backslash plus the token.
		if i+1 >= len(escaped) {
			return "", fmt.Errorf("trailing backslash at offset %d", i)
		}
		next := escaped[i+1]
		switch next {
		case '\\':
			b.WriteByte('\\')
			i++
		case 't':
			b.WriteByte('\t')
			i++
		case 'n':
			b.WriteByte('\n')
			i++
		case 'r':
			b.WriteByte('\r')
			i++
		case 'x':
			if i+3 >= len(escaped) {
				return "", fmt.Errorf("truncated hex escape at offset %d", i)
			}
			var hi, lo byte
			if v, ok := hexNibble(escaped[i+2]); ok {
				hi = v
			} else {
				return "", fmt.Errorf("invalid hex escape at offset %d", i)
			}
			if v, ok := hexNibble(escaped[i+3]); ok {
				lo = v
			} else {
				return "", fmt.Errorf("invalid hex escape at offset %d", i)
			}
			b.WriteByte((hi << 4) | lo)
			i += 3
		default:
			return "", fmt.Errorf("unknown escape \\%c at offset %d", next, i)
		}
	}
	return b.String(), nil
}

func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}
