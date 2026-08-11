// Prose normalization helpers.
//
// This file contains the deterministic normalization primitives
// used by the prose scanner. They are kept separate from the
// scanner (prose.go) so that the scanner stays focused on the
// imperative-vs-protected-operation classification while the
// normalization rules remain a single coherent unit.

package agentcontext

import (
	"regexp"
	"strings"
)

// stripFencedCodeBlocks removes fenced code blocks from content.
// Fenced blocks are delimited by lines starting with ```.
func stripFencedCodeBlocks(content string) string {
	var sb strings.Builder
	inFence := false
	hadContent := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if hadContent {
			sb.WriteByte('\n')
		}
		sb.WriteString(line)
		hadContent = true
	}
	return sb.String()
}

// stripContractBlock removes the structured authority contract
// block from content.
func stripContractBlock(content string) string {
	beginIdx := strings.Index(content, ContractBeginMarker)
	endIdx := strings.Index(content, ContractEndMarker)
	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		return content
	}
	return content[:beginIdx] + content[endIdx+len(ContractEndMarker):]
}

// stripBacktickDelimiters removes backtick delimiters from inline
// code spans while PRESERVING the wrapped content. `Run `make gate“.
// becomes "Run make gate." (backticks gone, content retained).
func stripBacktickDelimiters(content string) string {
	var sb strings.Builder
	sb.Grow(len(content))
	for i := 0; i < len(content); i++ {
		c := content[i]
		if c == '`' {
			continue
		}
		sb.WriteByte(c)
	}
	return sb.String()
}

// whitespaceCollapse collapses runs of internal whitespace into a
// single space and trims the result.
func whitespaceCollapse(s string) string {
	var sb strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				sb.WriteByte(' ')
			}
			prevSpace = true
			continue
		}
		prevSpace = false
		sb.WriteRune(r)
	}
	return strings.TrimSpace(sb.String())
}

// splitBulletLines splits a paragraph into individual Markdown
// bullet/list-item lines. Each returned line has its bullet prefix
// already stripped.
func splitBulletLines(paragraph string) []string {
	var lines []string
	for _, raw := range strings.Split(paragraph, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lines = append(lines, stripBulletPrefix(line))
	}
	return lines
}

var bulletPrefixPattern = regexp.MustCompile(`^([-*+]|\d+\.)\s+`)

func stripBulletPrefix(line string) string {
	if loc := bulletPrefixPattern.FindStringIndex(line); loc != nil {
		return strings.TrimSpace(line[loc[1]:])
	}
	return line
}

// splitIntoUnits splits a paragraph into normative sentence/clause
// units. Units are separated by sentence-ending punctuation
// (period, semicolon, exclamation, question mark) optionally
// followed by whitespace.
func splitIntoUnits(paragraph string) []string {
	re := regexp.MustCompile(`[.!?;]+\s*`)
	parts := re.Split(paragraph, -1)
	var units []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		units = append(units, p)
	}
	return units
}

// splitParagraphs splits content into paragraph units.
func splitParagraphs(content string) []string {
	parts := strings.Split(content, "\n\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// truncate returns at most n runes of s, ending with "..." if it
// was actually truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
