// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// range_mode_bypass_helpers_test.go contains shared test helpers for
// regression tests of ACT-LEAMAS-FACTORY-TARGETED-DIGEST-RANGE-MODE-BYPASS-CORRECTION01.
package digest

import "strings"

// extractHeader extracts the digest header (Mode, Range, etc.)
func extractHeader(digest string) string {
	lines := strings.Split(digest, "\n")
	var header []string
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "# ") {
			header = append(header, line)
		}
		if strings.HasPrefix(line, "Mode:") || strings.HasPrefix(line, "Range:") {
			header = append(header, line)
		}
		if len(header) > 0 && strings.HasPrefix(line, "## CHANGESET") {
			break
		}
	}
	return strings.Join(header, "\n")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func extractDiffSection(digest, filename string) string {
	idx := strings.Index(digest, "=== "+filename+" ===")
	if idx == -1 {
		idx = strings.Index(digest, "=== ")
		for idx != -1 {
			endIdx := strings.Index(digest[idx:], " ===")
			if endIdx != -1 && strings.Contains(digest[idx:idx+endIdx], filename) {
				break
			}
			idx = strings.Index(digest[idx+4:], "=== ")
			if idx != -1 {
				idx += 4
			}
		}
	}
	if idx == -1 {
		return ""
	}
	endIdx := strings.Index(digest[idx+1:], "===")
	if endIdx == -1 {
		return digest[idx:]
	}
	return digest[idx : idx+endIdx+3]
}
