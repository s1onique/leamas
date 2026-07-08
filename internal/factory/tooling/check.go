// Package tooling provides verification for tooling language boundaries.
package tooling

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

const maxBashLOC = 50

// CheckToolingBoundaries verifies tooling language boundaries.
func CheckToolingBoundaries(root string) []checks.Finding {
	var findings []checks.Finding

	// Check for Python files (strictly forbidden)
	findings = append(findings, checkPythonFiles(root)...)

	// Check Bash script LOC limits
	findings = append(findings, checkBashLOC(root)...)

	// Check Bash verifier logic doesn't exist
	findings = append(findings, checkNoBashVerifierLogic(root)...)

	checks.SortFindings(findings)
	return findings
}

// checkPythonFiles verifies no Python files exist.
func checkPythonFiles(root string) []checks.Finding {
	var findings []checks.Finding

	ignoredDirs := map[string]bool{
		".git":   true,
		"build":  true,
		"bin":    true,
		"vendor": true,
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if ignoredDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".py" {
			relPath, _ := filepath.Rel(root, path)
			findings = append(findings, checks.Finding{
				Path:     relPath,
				Kind:     "forbidden",
				Message:  "Python file strictly forbidden",
				Severity: checks.SeverityError,
			})
		}

		return nil
	})
	if err != nil {
		return findings
	}

	return findings
}

// checkBashLOC checks that Bash scripts are ≤50 meaningful LOC.
func checkBashLOC(root string) []checks.Finding {
	var findings []checks.Finding

	// Directories where Bash is allowed
	allowedDirs := map[string]bool{
		"scripts":  true,
		"githooks": true,
	}

	ignoredDirs := map[string]bool{
		".git":   true,
		"build":  true,
		"bin":    true,
		"vendor": true,
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if ignoredDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".sh" {
			return nil
		}

		relPath, _ := filepath.Rel(root, path)

		// Check if in allowed directory
		if !isInAllowedDir(relPath, allowedDirs) {
			findings = append(findings, checks.Finding{
				Path:     relPath,
				Kind:     "forbidden_location",
				Message:  "shell script outside allowed directories",
				Severity: checks.SeverityError,
			})
			return nil
		}

		// Check if executable
		info, err := os.Stat(path)
		if err != nil {
			return nil
		}

		if info.Mode()&0111 == 0 {
			// Not executable, skip LOC check
			return nil
		}

		// Check LOC
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		loc := checks.CountMeaningfulBashLOC(string(data))
		if loc > maxBashLOC {
			findings = append(findings, checks.Finding{
				Path:     relPath,
				Kind:     "too_long",
				Message:  "Bash script exceeds 50 meaningful LOC (" + string(rune('0'+loc/100)) + string(rune('0'+loc%100/10)) + string(rune('0'+loc%10)) + ")",
				Severity: checks.SeverityError,
			})
		}

		return nil
	})
	if err != nil {
		return findings
	}

	return findings
}

// isInAllowedDir checks if a path is in an allowed directory.
func isInAllowedDir(path string, allowed map[string]bool) bool {
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) == 0 {
		return false
	}
	return allowed[parts[0]]
}

// checkNoBashVerifierLogic ensures no Bash verifier scripts contain implementation logic.
func checkNoBashVerifierLogic(root string) []checks.Finding {
	var findings []checks.Finding

	// Patterns that indicate verifier logic in Bash
	verifierPatterns := []struct {
		pattern string
		desc    string
	}{
		{"grep -", "grep-based verification"},
		{"find .*\\.go", "find-based Go file scanning"},
		{"awk ", "awk-based processing"},
		{"sed ", "sed-based text processing"},
	}

	scriptsDir := filepath.Join(root, "scripts")
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		return findings
	}

	err := filepath.WalkDir(scriptsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".sh" {
			return nil
		}

		// Skip non-verify scripts
		name := d.Name()
		if !strings.HasPrefix(name, "verify_") {
			return nil
		}

		// Skip wrapper scripts that just delegate
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)

		// If it contains go run or exec, it might be a wrapper
		if strings.Contains(content, "go run ./cmd/leamas") || strings.Contains(content, "exec ./bin/leamas") {
			// This is a wrapper, check LOC
			loc := checks.CountMeaningfulBashLOC(content)
			if loc > maxBashLOC {
				relPath, _ := filepath.Rel(root, path)
				findings = append(findings, checks.Finding{
					Path:     relPath,
					Kind:     "too_long",
					Message:  "wrapper exceeds 50 meaningful LOC",
					Severity: checks.SeverityError,
				})
			}
			return nil
		}

		// Check for verifier implementation patterns
		for _, vp := range verifierPatterns {
			if strings.Contains(content, vp.pattern) {
				relPath, _ := filepath.Rel(root, path)
				findings = append(findings, checks.Finding{
					Path:     relPath,
					Kind:     "bash_verifier_logic",
					Message:  "Bash script contains verifier implementation: " + vp.desc,
					Severity: checks.SeverityError,
				})
				break
			}
		}

		return nil
	})
	if err != nil {
		return findings
	}

	return findings
}

// CheckRepo runs all tooling boundary checks.
func CheckRepo(root string) []checks.Finding {
	return CheckToolingBoundaries(root)
}
