// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"bytes"
	"io"
	"os"
	"strings"
)

const (
	// MaxPreviewBytes is the maximum bytes to include in a file preview.
	MaxPreviewBytes = 16 * 1024 // 16 KiB
	// MaxPreviewLines is the maximum lines to include in a file preview.
	MaxPreviewLines = 200
)

// IsBinary checks if a file appears to be binary.
func IsBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	// Read first 8KB
	buf := make([]byte, 8192)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	buf = buf[:n]

	// Check for null bytes (common in binary files)
	for _, b := range buf {
		if b == 0 {
			return true
		}
	}

	return false
}

// PreviewFile reads and returns a preview of a file, bounded by MaxPreviewBytes and MaxPreviewLines.
// Returns the preview content and whether the file is binary.
func PreviewFile(path string, maxBytes, maxLines int) (string, bool) {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return "(file not present)\n", false
	}

	// Check if it's a directory
	if info.IsDir() {
		return "(directory)\n", false
	}

	// Check if binary
	if IsBinary(path) {
		return "", true
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return "(error reading file)\n", false
	}

	// Apply byte limit
	if len(content) > maxBytes {
		content = content[:maxBytes]
	}

	// Convert to string and apply line limit
	preview := string(content)
	lines := strings.Split(preview, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		preview = strings.Join(lines, "\n") + "\n(truncated)"
	}

	// Ensure trailing newline
	if !strings.HasSuffix(preview, "\n") {
		preview += "\n"
	}

	return preview, false
}

// PreviewFileWithEncoding reads a file and returns a preview with proper encoding handling.
func PreviewFileWithEncoding(path string) (string, bool) {
	return PreviewFile(path, MaxPreviewBytes, MaxPreviewLines)
}

// CleanPreview ensures preview output is consistent and safe.
func CleanPreview(content string) string {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Remove very long lines (common in minified/binary data)
	var sb strings.Builder
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if len(line) > 1000 {
			line = line[:1000] + "..."
		}
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// BytesPreview returns a preview limited by byte count only.
func BytesPreview(path string, maxBytes int) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	buf := &bytes.Buffer{}
	limited := io.LimitReader(f, int64(maxBytes))
	_, err = io.Copy(buf, limited)
	if err != nil {
		return "", false
	}

	// Check if we hit the limit
	info, _ := f.Stat()
	if info.Size() > int64(maxBytes) {
		return buf.String() + "\n(truncated)", false
	}

	return buf.String(), false
}

// ReadFileFull reads and returns the full content of a file without truncation.
// This is the preferred method for digest output where complete file context is required.
// Returns the content and whether the file is binary.
func ReadFileFull(path string) (string, bool) {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return "(file not present)\n", false
	}

	// Check if it's a directory
	if info.IsDir() {
		return "(directory)\n", false
	}

	// Check if binary
	if IsBinary(path) {
		return "", true
	}

	// Read full file content
	content, err := os.ReadFile(path)
	if err != nil {
		return "(error reading file)\n", false
	}

	// Ensure trailing newline
	result := string(content)
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result, false
}
