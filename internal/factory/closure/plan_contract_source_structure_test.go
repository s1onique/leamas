package closure

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// plan_contract_source_structure_test.go enforces the package's
// own LLM-friendly invariants at test-time. The directive ACT
// requires splitting long inline fixtures into small focused
// files; this file proves that outcome and guards against
// future regressions.

// maxLineLength mirrors the LLM-friendly gate threshold (240
// chars per line). Any test file in the closure package whose
// lines exceed this threshold trips the test.
const maxLineLength = 240

// TestSourceStructureNoLongLines walks every Go file in the
// closure package directory and asserts that no line exceeds
// the LLM-friendly line-length threshold. This proves the
// "split long inline fixtures into small focused files"
// requirement is mechanically satisfied.
func TestSourceStructureNoLongLines(t *testing.T) {
	files, err := filepath.Glob("./*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no Go files found in closure package directory")
	}
	for _, path := range files {
		scanFile(t, path)
	}
}

// TestSourceStructureNoAllowlistInLLMFriendly asserts the
// LLM-friendly verifier has not grown an allowlist entry for
// any closure-package source file. The existing
// isCanonicalClosurePlan exemption is restricted to
// docs/closure-plans/*.json; adding a similar exemption that
// names any closure-package Go file would mask a real
// LLM-friendliness regression.
func TestSourceStructureNoAllowlistInLLMFriendly(t *testing.T) {
	data, err := os.ReadFile("../llmfriendly/check.go")
	if err != nil {
		t.Fatalf("read llmfriendly/check.go: %v", err)
	}
	content := string(data)
	markers := []string{
		"internal/factory/closure/",
		"plan_contract_",
		"closure package",
	}
	for _, marker := range markers {
		if strings.Contains(content, marker) {
			t.Fatalf("llmfriendly/check.go mentions %q; that is a forbidden allowlist for the closure package", marker)
		}
	}
}

// scanFile reads path and asserts every line is at most
// maxLineLength runes long. The line counter and offending
// snippet are reported in the failure message.
func scanFile(t *testing.T, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	const bufferSize = 1024 * 1024
	scanner.Buffer(make([]byte, bufferSize), bufferSize)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if len(line) <= maxLineLength {
			continue
		}
		t.Fatalf("%s line %d: %d chars (max %d): %s",
			path, lineNum, len(line), maxLineLength, truncate(line, 80))
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("%s scan: %v", path, err)
	}
}

// truncate returns the first n characters of s plus an ellipsis
// when truncation happened.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
