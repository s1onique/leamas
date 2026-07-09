// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
)

// toolchainRegex matches toolchain directive lines.
var toolchainRegex = regexp.MustCompile(`(?m)^toolchain\s+(\S+)`)

// extractToolchain extracts toolchain name from go.mod content.
func extractToolchain(content []byte) string {
	match := toolchainRegex.FindStringSubmatch(string(content))
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

// goModWithToolchain wraps modfile.File to add toolchain extraction.
type goModWithToolchain struct {
	*modfile.File
	ToolchainName string
}

// parseGoMod parses go.mod content, extracting toolchain via regex as fallback.
func parseGoMod(filename string, content []byte) (*goModWithToolchain, error) {
	// First try Parse (captures toolchain and replace directives)
	f, err := modfile.Parse(filename, content, nil)
	if err == nil {
		toolchainName := ""
		if f.Toolchain != nil {
			toolchainName = f.Toolchain.Name
		}
		return &goModWithToolchain{File: f, ToolchainName: toolchainName}, nil
	}

	// Fall back to ParseLax
	f, err = modfile.ParseLax(filename, content, nil)
	if err != nil {
		return nil, err
	}

	toolchainName := extractToolchain(content)
	return &goModWithToolchain{File: f, ToolchainName: toolchainName}, nil
}

// getGoModAtCommit reads go.mod content at a specific commit.
func getGoModAtCommit(repoRoot, ref string) (*goModWithToolchain, error) {
	content, err := getFileContentAtCommit(repoRoot, "go.mod", ref)
	if err != nil {
		return nil, err
	}
	return parseGoMod("go.mod", content)
}

// getWorktreeGoMod reads go.mod from the worktree.
func getWorktreeGoMod(repoRoot string) (*goModWithToolchain, error) {
	content, err := getWorktreeFileContent(repoRoot, "go.mod")
	if err != nil {
		return nil, err
	}
	return parseGoMod("go.mod", content)
}

// getIndexGoMod reads go.mod from the git index.
func getIndexGoMod(repoRoot string) (*goModWithToolchain, error) {
	content, err := getIndexFileContent(repoRoot, "go.mod")
	if err != nil {
		return nil, err
	}
	return parseGoMod("go.mod", content)
}

// getGoSumAtCommit reads go.sum content at a specific commit.
func getGoSumAtCommit(repoRoot, ref string) (map[string]string, error) {
	content, err := getFileContentAtCommit(repoRoot, "go.sum", ref)
	if err != nil {
		return nil, err
	}
	return parseGoSum(content), nil
}

// getWorktreeGoSum reads go.sum from the worktree.
func getWorktreeGoSum(repoRoot string) (map[string]string, error) {
	content, err := getWorktreeFileContent(repoRoot, "go.sum")
	if err != nil {
		return nil, err
	}
	return parseGoSum(content), nil
}

// getIndexGoSum reads go.sum from the git index.
func getIndexGoSum(repoRoot string) (map[string]string, error) {
	content, err := getIndexFileContent(repoRoot, "go.sum")
	if err != nil {
		return nil, err
	}
	return parseGoSum(content), nil
}

// parseGoSum parses go.sum content into a map of module@version -> full_line.
// Full line preservation ensures we can detect checksum changes for same module+version.
func parseGoSum(content []byte) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// Key: module path + version, Value: full line (for hash preservation)
			key := parts[0] + " " + parts[1]
			result[key] = line
		}
	}
	return result
}

// goModuleFiles returns the files this section tracks.
func goModuleFiles() []string {
	return []string{"go.mod", "go.sum"}
}

// hasGoModuleFiles checks if any of the changed files are go.mod or go.sum.
func hasGoModuleFiles(paths []string) bool {
	for _, p := range paths {
		if p == "go.mod" || p == "go.sum" {
			return true
		}
	}
	return false
}

// mapsEqual compares two maps for equality.
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}
