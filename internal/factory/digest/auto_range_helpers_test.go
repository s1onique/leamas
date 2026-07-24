// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// auto_range_helpers_test.go provides shared helpers used by the
// other auto_range_test files for cross-checking digest output
// against `git diff --name-status`.
package digest

import (
	"strings"
	"testing"
)

// gitNameStatus returns the repo-relative file paths touched by range.
func gitNameStatus(t *testing.T, dir, revRange string) []string {
	t.Helper()
	out, err := runGitValueTrimmed(dir, "diff", "--name-only", revRange)
	if err != nil {
		t.Fatalf("git diff --name-only: %v", err)
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files
}

// digestNameStatus extracts the file lines from a CHANGESET_MANIFEST block.
func digestNameStatus(content string) []string {
	idx := strings.Index(content, "## CHANGESET_MANIFEST")
	if idx < 0 {
		return nil
	}
	rest := content[idx+len("## CHANGESET_MANIFEST"):]
	end := strings.Index(rest, "## CHANGESET_STATS")
	if end >= 0 {
		rest = rest[:end]
	}
	var files []string
	for _, line := range strings.Split(rest, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			files = append(files, fields[1])
		}
	}
	return files
}

// sameFileSet returns true when two file lists contain the same paths
// irrespective of order.
func sameFileSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, x := range a {
		m[x]++
	}
	for _, x := range b {
		m[x]--
		if m[x] < 0 {
			return false
		}
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}

// GenerateOrFatal invokes Generate for use inside non-Generate helpers.
func GenerateOrFatal(t *testing.T, dir string) string {
	t.Helper()
	out, err := Generate(Options{RepoRoot: dir, Mode: ModeAuto})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return out
}
