// Package llmfriendly provides LLM-friendliness verification for repositories.
// It ensures files are reviewable by LLMs, agents, and humans by rejecting
// oversized, dense, minified, or generated-looking committed files.
package llmfriendly

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

// Config holds thresholds for LLM-friendliness checks.
type Config struct {
	// MaxBytes is the maximum file size in bytes (default 64 KiB).
	MaxBytes int64
	// MaxLines is the maximum number of lines in a text file (default 400).
	MaxLines int
	// MaxLineLength is the maximum allowed line length (default 240 chars).
	MaxLineLength int
	// MinifiedLineLength is the threshold for detecting minified lines (default 1000 chars).
	MinifiedLineLength int
}

// Finding represents a single LLM-friendliness violation.
type Finding struct {
	Path    string
	Kind    string
	Message string
}

// DefaultConfig returns the default LLM-friendliness configuration.
func DefaultConfig() Config {
	return Config{
		MaxBytes:           64 * 1024, // 64 KiB
		MaxLines:           400,
		MaxLineLength:      240,
		MinifiedLineLength: 1000,
	}
}

// CheckRepo checks all Git-visible files in the repository for LLM-friendliness.
func CheckRepo(root string, cfg Config) ([]Finding, error) {
	files, err := listGitFiles(root)
	if err != nil {
		return nil, fmt.Errorf("listing git files: %w", err)
	}

	var findings []Finding
	for _, file := range files {
		f, err := checkFile(filepath.Join(root, file), cfg)
		if err != nil {
			// Skip files we can't read
			continue
		}
		findings = append(findings, f...)
	}

	// Sort findings deterministically by path
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Path < findings[j].Path
	})

	return findings, nil
}

// listGitFiles returns a list of files tracked by Git (cached and untracked).
func listGitFiles(root string) ([]string, error) {
	// Get cached (tracked) files
	cached, err := runGitCommand(root, "ls-files", "--cached")
	if err != nil {
		return nil, err
	}

	// Get untracked files (excluding ignored)
	others, err := runGitCommand(root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}

	// Combine and dedupe
	all := append(cached, others...)
	seen := make(map[string]bool)
	var unique []string
	for _, f := range all {
		// Skip if already seen or if it's a directory-like entry
		if seen[f] || f == "" {
			continue
		}
		seen[f] = true
		unique = append(unique, f)
	}

	// Sort for deterministic output
	sort.Strings(unique)
	return unique, nil
}

// runGitCommand runs a git command and returns its output lines.
func runGitCommand(root string, args ...string) ([]string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Git may return non-zero for untracked files, which is OK
			if exitErr.ExitCode() == 128 {
				return nil, nil
			}
		}
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// checkFile checks a single file for LLM-friendliness violations.
func checkFile(path string, cfg Config) ([]Finding, error) {
	// Check structural ignores (defensive, in case git didn't catch them)
	ignoredDirs := []string{".git", "build", "bin", "vendor"}
	for _, dir := range ignoredDirs {
		if strings.HasPrefix(path, dir+"/") || path == dir {
			return nil, nil
		}
	}

	// Check path-based ignores for large data files that are inherently large
	ignoredPaths := []string{
		".factory/dupcode-baseline.json",
		".factory/coverage.out",
	}
	for _, p := range ignoredPaths {
		if path == p || strings.HasSuffix(path, p) {
			return nil, nil
		}
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Skip directories
	if info.IsDir() {
		return nil, nil
	}

	// Check file size
	if info.Size() > cfg.MaxBytes {
		return []Finding{{
			Path:    path,
			Kind:    "too_large",
			Message: fmt.Sprintf("%d > %d bytes", info.Size(), cfg.MaxBytes),
		}}, nil
	}

	// Detect and skip binary files
	if isBinary(path) {
		return nil, nil
	}

	// Open and check contents
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return scanTextFile(path, file, cfg)
}

// isBinary detects if a file is binary by checking for NUL bytes.
func isBinary(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first 8KB to check for NUL bytes
	buf := make([]byte, 8192)
	n, err := file.Read(buf)
	if err != nil {
		return false
	}

	// Check for NUL byte in the first n bytes
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}

// scanTextFile scans a text file for LLM-friendliness violations.
func scanTextFile(path string, file *os.File, cfg Config) ([]Finding, error) {
	var findings []Finding
	lineNum := 0
	maxLineLen := 0
	minifiedDetected := false

	// Use a large scanner buffer for long line detection
	const maxTokenSize = 64 * 1024 // 64KB buffer
	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lineLen := len(line)

		// Track max line length
		if lineLen > maxLineLen {
			maxLineLen = lineLen
		}

		// Check for long lines
		if lineLen > cfg.MaxLineLength {
			findings = append(findings, Finding{
				Path:    path,
				Kind:    "long_line",
				Message: fmt.Sprintf("line %d: %d > %d chars", lineNum, lineLen, cfg.MaxLineLength),
			})
		}

		// Check for minified-looking lines (only in certain file types)
		if isMinifiableFile(path) && lineLen > cfg.MinifiedLineLength {
			findings = append(findings, Finding{
				Path:    path,
				Kind:    "minified_line",
				Message: fmt.Sprintf("line %d: %d > %d chars (minified)", lineNum, lineLen, cfg.MinifiedLineLength),
			})
			minifiedDetected = true
			// Don't break - still need to count lines
		}
	}

	if err := scanner.Err(); err != nil {
		// Scanner errors are non-fatal; continue with what we have
	}

	// Check total line count (only if we haven't already flagged minified lines)
	if lineNum > cfg.MaxLines && !minifiedDetected {
		findings = append(findings, Finding{
			Path:    path,
			Kind:    "too_many_lines",
			Message: fmt.Sprintf("%d > %d", lineNum, cfg.MaxLines),
		})
	}

	return findings, nil
}

// isMinifiableFile returns true if the file type is typically minified.
func isMinifiableFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	minifiableExts := []string{".js", ".css", ".html", ".json", ".xml", ".svg", ".min.js", ".min.css"}
	for _, e := range minifiableExts {
		if ext == e {
			return true
		}
	}
	return false
}
