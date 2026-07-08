// Package githooks provides verification for Git hook installation and configuration.
// It ensures pre-push hooks are properly installed and configured to prevent force-pushes.
package githooks

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Finding represents a single Git hooks verification finding.
type Finding struct {
	Path    string
	Kind    string
	Message string
}

// CheckRepo verifies Git hooks are properly installed and configured.
func CheckRepo(root string) ([]Finding, error) {
	var findings []Finding

	findings = append(findings, checkPrePushHook(root)...)
	findings = append(findings, checkHookInstaller(root)...)
	findings = append(findings, checkNoBashVerifier(root)...)
	findings = append(findings, checkHooksPath(root)...)

	// Sort findings deterministically by path, then kind
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Kind < findings[j].Kind
	})

	return findings, nil
}

// checkPrePushHook verifies githooks/pre-push exists, is executable, and has required content.
func checkPrePushHook(root string) []Finding {
	var findings []Finding
	path := filepath.Join(root, "githooks", "pre-push")

	// Check existence
	data, err := os.ReadFile(path)
	if err != nil {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "missing",
			Message: "githooks/pre-push not found",
		})
		return findings
	}

	// Check executable
	info, err := os.Stat(path)
	if err != nil {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "error",
			Message: fmt.Sprintf("cannot stat: %v", err),
		})
		return findings
	}

	// Check if executable (mode & 0111)
	if info.Mode()&0111 == 0 {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "not_executable",
			Message: "hook must be executable",
		})
	}

	// Check required content patterns
	content := string(data)
	requiredPatterns := []struct {
		pattern string
		desc    string
	}{
		{"refs/heads/main", "protected branch main"},
		{"refs/heads/master", "protected branch master"},
		{"refs/heads/release/", "protected branch release/*"},
		{"merge-base --is-ancestor", "non-fast-forward detection"},
		{"refusing non-fast-forward", "non-fast-forward error message"},
		{"refusing to delete protected branch", "deletion error message"},
	}

	for _, req := range requiredPatterns {
		if !strings.Contains(content, req.pattern) {
			findings = append(findings, Finding{
				Path:    path,
				Kind:    "missing_content",
				Message: fmt.Sprintf("missing required content: %s", req.desc),
			})
		}
	}

	// Check LOC limit (≤50 meaningful lines)
	loc := countMeaningfulLOC(content)
	if loc > 50 {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "too_long",
			Message: fmt.Sprintf("%d meaningful LOC > 50", loc),
		})
	}

	return findings
}

// checkHookInstaller verifies scripts/install_git_hooks.sh exists and has required content.
func checkHookInstaller(root string) []Finding {
	var findings []Finding
	path := filepath.Join(root, "scripts", "install_git_hooks.sh")

	// Check existence
	data, err := os.ReadFile(path)
	if err != nil {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "missing",
			Message: "scripts/install_git_hooks.sh not found",
		})
		return findings
	}

	// Check executable
	info, err := os.Stat(path)
	if err != nil {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "error",
			Message: fmt.Sprintf("cannot stat: %v", err),
		})
		return findings
	}

	if info.Mode()&0111 == 0 {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "not_executable",
			Message: "installer must be executable",
		})
	}

	// Check required content patterns
	content := string(data)
	requiredPatterns := []struct {
		pattern string
		desc    string
	}{
		{"core.hooksPath githooks", "sets core.hooksPath to githooks"},
		{"chmod +x githooks/pre-push", "makes hook executable"},
	}

	for _, req := range requiredPatterns {
		if !strings.Contains(content, req.pattern) {
			findings = append(findings, Finding{
				Path:    path,
				Kind:    "missing_content",
				Message: fmt.Sprintf("missing required content: %s", req.desc),
			})
		}
	}

	// Check LOC limit (≤50 meaningful lines)
	loc := countMeaningfulLOC(content)
	if loc > 50 {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "too_long",
			Message: fmt.Sprintf("%d meaningful LOC > 50", loc),
		})
	}

	return findings
}

// checkNoBashVerifier ensures scripts/verify_git_hooks.sh does not exist.
func checkNoBashVerifier(root string) []Finding {
	var findings []Finding
	path := filepath.Join(root, "scripts", "verify_git_hooks.sh")

	if _, err := os.Stat(path); err == nil {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "forbidden",
			Message: "Bash verifier scripts are forbidden; use Go verifier",
		})
	}

	return findings
}

// checkHooksPath verifies core.hooksPath is set to githooks.
func checkHooksPath(root string) []Finding {
	var findings []Finding

	cmd := exec.Command("git", "config", "--get", "core.hooksPath")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		// git config returns exit code 1 when key doesn't exist
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			findings = append(findings, Finding{
				Path:    ".git/config",
				Kind:    "hooks_path",
				Message: "core.hooksPath not configured",
			})
			return findings
		}
		findings = append(findings, Finding{
			Path:    ".git/config",
			Kind:    "error",
			Message: fmt.Sprintf("failed to read core.hooksPath: %v", err),
		})
		return findings
	}

	hooksPath := strings.TrimSpace(string(output))
	if hooksPath != "githooks" {
		findings = append(findings, Finding{
			Path:    ".git/config",
			Kind:    "hooks_path",
			Message: fmt.Sprintf("core.hooksPath must be githooks, got: %s", hooksPath),
		})
	}

	return findings
}

// countMeaningfulLOC counts non-blank, non-comment lines in Bash content.
func countMeaningfulLOC(content string) int {
	count := 0
	scanner := bufio.NewScanner(bytes.NewReader([]byte(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comment-only lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		count++
	}
	return count
}
