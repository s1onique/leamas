// Package checks provides shared types and utilities for Factory verification.
package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Severity represents the severity level of a finding.
type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warn"
)

// Finding represents a single verification finding.
type Finding struct {
	Path     string
	Kind     string
	Message  string
	Severity Severity
}

// Result represents a verification result.
type Result struct {
	Name     string
	Findings []Finding
}

// HasErrors returns true if findings contain any errors.
func HasErrors(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// SortFindings sorts findings deterministically by path, then kind, then message.
func SortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		return findings[i].Message < findings[j].Message
	})
}

// PrintResult prints findings for a check and returns exit code.
func PrintResult(name string, findings []Finding) int {
	SortFindings(findings)
	fmt.Printf("%s\n", name)
	for _, f := range findings {
		fmt.Printf("  %s: %s: %s\n", f.Path, f.Kind, f.Message)
	}
	if HasErrors(findings) {
		return 1
	}
	return 0
}

// PathInDir checks if a path is within a directory.
func PathInDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	// If relative path starts with "..", it's outside
	return !strings.HasPrefix(rel, "..")
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads a file and returns its content. Returns nil if file doesn't exist.
func ReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

// CountMeaningfulBashLOC counts non-blank, non-comment lines in Bash content.
func CountMeaningfulBashLOC(content string) int {
	count := 0
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		count++
	}
	return count
}
