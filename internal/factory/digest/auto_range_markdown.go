// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"strings"
)

// parseImplementationRangeTable extracts the BASE (and optional
// Subject) commits from the structured "## Implementation Range"
// markdown table. The parser tolerates both long and short OIDs and
// ignores parenthetical annotations such as "9276bce (whitespace
// fix)".
//
// Returns false when no BASE row is found.
func parseImplementationRangeTable(md string) (base, subject string, ok bool) {
	const headerMarker = "## Implementation Range"
	idx := strings.Index(md, headerMarker)
	if idx < 0 {
		return "", "", false
	}
	tail := md[idx+len(headerMarker):]
	if next := strings.Index(tail, "\n## "); next >= 0 {
		tail = tail[:next]
	}
	rows := parseMarkdownTableRows(tail)
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		identity := strings.ToUpper(strings.TrimSpace(row[0]))
		commit := strings.TrimSpace(row[1])
		oid := extractFirstOID(commit)
		if oid == "" {
			continue
		}
		switch identity {
		case "BASE", "BASELINE":
			base = oid
		case "SUBJECT", "SUBJECT (HEAD)", "SUBJECT_HEAD", "HEAD", "SUBJECT(HEAD)":
			subject = oid
		}
	}
	if base == "" {
		return "", "", false
	}
	return base, subject, true
}

// parseMarkdownTableRows returns the data rows of the first markdown
// table in the input. Skips the header and separator rows.
func parseMarkdownTableRows(s string) [][]string {
	var rows [][]string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			if len(rows) > 0 {
				break
			}
			continue
		}
		cells := splitMarkdownRow(line)
		if len(cells) == 1 {
			continue
		}
		allSeparators := true
		for _, c := range cells {
			t := strings.TrimSpace(c)
			if t != "" && !isMarkdownSeparator(t) {
				allSeparators = false
				break
			}
		}
		if allSeparators {
			continue
		}
		rows = append(rows, cells)
	}
	return rows
}

// splitMarkdownRow splits a markdown table row on '|' boundaries,
// trimming cell whitespace.
func splitMarkdownRow(line string) []string {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// isMarkdownSeparator returns true when the cell looks like a table
// separator (e.g. "---", ":---:", "-----").
func isMarkdownSeparator(cell string) bool {
	if cell == "" {
		return false
	}
	for _, r := range cell {
		if r != '-' && r != ':' && r != ' ' {
			return false
		}
	}
	return true
}

// extractFirstOID returns the first hex SHA-like substring in the
// input that is at least 7 characters long.
func extractFirstOID(s string) string {
	for _, m := range shortOIDPattern.FindAllString(s, -1) {
		if len(m) >= 7 {
			return m
		}
	}
	return ""
}
