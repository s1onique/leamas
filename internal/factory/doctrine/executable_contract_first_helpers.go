package doctrine

import "strings"

// extractECFMarkedBlock extracts the content between the ECF markers.
// Applies same normalization as canonical content for fair comparison.
// Normalizes first (CRLF→LF, trim trailing newline), then handles leading newline.
func extractECFMarkedBlock(content string) string {
	beginIdx := strings.Index(content, ecfBeginMarker)
	endIdx := strings.Index(content, ecfEndMarker)

	if beginIdx == -1 || endIdx == -1 {
		return ""
	}

	start := beginIdx + len(ecfBeginMarker)
	if start >= endIdx {
		return ""
	}

	block := content[start:endIdx]
	// Normalize first (same as canonical).
	block = normalizeECFContent(block)
	// Strip leading newline for consistency (the marker ends with ">  " and we add "\n").
	if len(block) > 0 && block[0] == '\n' {
		block = block[1:]
	}
	return block
}

// normalizeECFContent normalizes line endings and trims a single final
// newline, while preserving all other whitespace exactly as the
// projection wrote it. This keeps exact byte-for-byte comparison with
// the canonical source possible.
func normalizeECFContent(content string) string {
	// Convert CRLF to LF.
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// Strip at most one trailing newline.
	if strings.HasSuffix(content, "\n") {
		content = content[:len(content)-1]
	}
	return content
}

// hasNestedECFMarks checks for nested marker blocks within the content.
func hasNestedECFMarks(content string) bool {
	// Find the outer markers.
	outerBegin := strings.Index(content, ecfBeginMarker)
	outerEnd := strings.Index(content, ecfEndMarker)

	if outerBegin == -1 || outerEnd == -1 {
		return false
	}

	// Check if there are inner markers between outer markers.
	innerContent := content[outerBegin+len(ecfBeginMarker) : outerEnd]
	innerBegin := strings.Index(innerContent, ecfBeginMarker)
	innerEnd := strings.Index(innerContent, ecfEndMarker)

	return innerBegin != -1 || innerEnd != -1
}

// countOccurrences counts non-overlapping occurrences of substr in s.
func countOccurrences(s, substr string) int {
	if substr == "" {
		return 0
	}
	count := 0
	for {
		idx := strings.Index(s, substr)
		if idx == -1 {
			break
		}
		count++
		s = s[idx+len(substr):]
	}
	return count
}
