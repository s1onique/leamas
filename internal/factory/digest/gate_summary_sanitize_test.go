// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestSanitizeLine tests the line sanitizer.
func TestSanitizeLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"simple", "hello world", "hello world"},
		{"newline", "hello\nworld", "hello world"},
		{"crlf", "hello\r\nworld", "hello world"},
		{"tab", "hello\tworld", "hello world"},
		{"multiple spaces", "hello   world", "hello world"},
		{"leading space", "  hello", "hello"},
		{"trailing space", "hello  ", "hello"},
		{"embedded equals", "key=value", "key=value"},
		{"unicode", "héllo wörld", "héllo wörld"},
		{"long text truncated", strings.Repeat("x", 300), strings.Repeat("x", 240)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeLine(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeLine(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestSanitizeLinePreservesUTF8 tests that multi-byte UTF-8 is not split.
func TestSanitizeLinePreservesUTF8(t *testing.T) {
	t.Parallel()

	// Test boundary cases with multi-byte runes
	testCases := []struct {
		name  string
		input string
	}{
		{"emoji at 240 boundary", strings.Repeat("x", 238) + "😀"}, // 238 + 4 = 242 > 240
		{"é at 240 boundary", strings.Repeat("é", 120) + "😀"},     // 240 + 4 = 244 > 240
		{"within limit", strings.Repeat("é", 80) + "😀"},           // 160 + 4 = 164 < 240
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeLine(tc.input)
			if !utf8.ValidString(got) {
				t.Errorf("sanitizeLine produced invalid UTF-8: %q", got)
			}
			if len(got) > 240 {
				t.Errorf("sanitizeLine exceeded 240 bytes: %d", len(got))
			}
		})
	}
}
