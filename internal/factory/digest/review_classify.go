// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"
	"strings"
)

// classifyFile determines the classification of a file based on its path.
func classifyFile(path string) string {
	if isTestFile(path) {
		return "test"
	}
	if isDocFile(path) {
		return "doc"
	}
	if isConfigFile(path) {
		return "config"
	}
	return "source"
}

// isTestFile returns true if the path matches test file patterns.
func isTestFile(path string) bool {
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	if strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "_test.ts") ||
		strings.HasSuffix(base, "_test.tsx") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".test.tsx") {
		return true
	}

	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
		return true
	}

	if dir == "tests" || strings.HasPrefix(dir, "tests/") {
		return true
	}

	return false
}

// isDocFile returns true if the path matches doc file patterns.
func isDocFile(path string) bool {
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	if strings.HasSuffix(base, ".md") ||
		strings.HasSuffix(base, ".adoc") ||
		strings.HasSuffix(base, ".rst") {
		return true
	}

	if dir == "docs" || strings.HasPrefix(dir, "docs/") {
		return true
	}

	return false
}

// isConfigFile returns true if the path matches config file patterns.
func isConfigFile(path string) bool {
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	if base == "Makefile" ||
		strings.HasSuffix(base, ".mk") ||
		strings.HasSuffix(base, ".yaml") ||
		strings.HasSuffix(base, ".yml") ||
		strings.HasSuffix(base, ".toml") ||
		strings.HasSuffix(base, ".json") {
		return true
	}

	if dir == ".github" || strings.HasPrefix(dir, ".github/") {
		return true
	}

	if base == ".gitlab-ci.yml" {
		return true
	}

	return false
}

// isGeneratedFileAtPath checks if a file has the canonical generated-file marker.
func isGeneratedFileAtPath(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return isGeneratedFileContent(string(data))
}

// isGeneratedFileContent checks if content has the canonical generated-file marker.
func isGeneratedFileContent(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			if generatedMarker.MatchString(trimmed) {
				return true
			}
			continue
		}
		return false
	}
	return false
}

// isBinaryFileAtPath checks if a file appears to be binary.
func isBinaryFileAtPath(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}
