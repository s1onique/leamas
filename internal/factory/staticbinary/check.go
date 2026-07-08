// Package staticbinary provides verification for static binary build requirements.
package staticbinary

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// CheckStaticBinaryIntent verifies the repository can produce a static binary.
func CheckStaticBinaryIntent(root string) []checks.Finding {
	var findings []checks.Finding

	// Check for go.mod
	goModPath := filepath.Join(root, "go.mod")
	if !checks.FileExists(goModPath) {
		findings = append(findings, checks.Finding{
			Path:     "go.mod",
			Kind:     "missing",
			Message:  "go.mod not found",
			Severity: checks.SeverityError,
		})
	}

	// Check for cmd/leamas/main.go
	mainPath := filepath.Join(root, "cmd", "leamas", "main.go")
	if !checks.FileExists(mainPath) {
		findings = append(findings, checks.Finding{
			Path:     "cmd/leamas/main.go",
			Kind:     "missing",
			Message:  "cmd/leamas/main.go not found",
			Severity: checks.SeverityError,
		})
	}

	// Check Makefile for CGO_ENABLED=0
	makefilePath := filepath.Join(root, "Makefile")
	if checks.FileExists(makefilePath) {
		data := checks.ReadFile(makefilePath)
		if data != nil {
			content := string(data)
			if !strings.Contains(content, "CGO_ENABLED=0") {
				findings = append(findings, checks.Finding{
					Path:     "Makefile",
					Kind:     "missing_config",
					Message:  "Makefile does not contain CGO_ENABLED=0",
					Severity: checks.SeverityWarn,
				})
			}
		}
	}

	// Try to build statically
	findings = append(findings, tryStaticBuild(root)...)

	checks.SortFindings(findings)
	return findings
}

// tryStaticBuild attempts to build the binary with CGO_ENABLED=0.
func tryStaticBuild(root string) []checks.Finding {
	var findings []checks.Finding

	// Try to run static build
	cmd := exec.Command("go", "build", "-trimpath", "-o", "bin/leamas-test", "./cmd/leamas")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		findings = append(findings, checks.Finding{
			Path:     ".",
			Kind:     "build_failed",
			Message:  "static build failed: " + strings.TrimSpace(string(output)),
			Severity: checks.SeverityError,
		})
		return findings
	}

	// Clean up test binary
	testBinary := filepath.Join(root, "bin", "leamas-test")
	os.Remove(testBinary)

	// Build succeeded - return nil (no findings) on success
	// Only return findings on failure
	return nil
}

// CheckRepo runs all static binary checks.
func CheckRepo(root string) []checks.Finding {
	return CheckStaticBinaryIntent(root)
}
