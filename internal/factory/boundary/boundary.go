// Package boundary provides verification for domain boundary import policies.
//
// This verifier ensures that intentionally constrained internal packages maintain
// their declared scope by checking import boundaries.
//
// Protected packages:
//   - internal/hulk/runbundle: Pure domain logic
//   - internal/hulk/claimevidence: Pure domain logic
//   - internal/witness/proxy: Local HTTP witness proxy seed
//   - internal/web/cockpit: Local read-only web cockpit seed
package boundary

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// PackagePolicy defines import constraints for a protected package.
type PackagePolicy struct {
	Name              string
	Dir               string
	AllowedImports    map[string]bool
	ForbiddenImports  map[string]string
	ForbiddenContains map[string]string
}

// Finding represents a boundary violation.
type Finding struct {
	File   string
	Import string
	Reason string
}

// Result contains all findings from boundary verification.
type Result struct {
	Findings []Finding
}

// OK returns true if there are no findings.
func (r Result) OK() bool {
	return len(r.Findings) == 0
}

// Standard library imports allowed for Hulk pure domain packages.
var hulkAllowedImports = map[string]bool{
	"sort":    true,
	"strings": true,
}

// Standard library imports forbidden for Hulk pure domain packages.
var hulkForbiddenImports = map[string]string{
	"context":       "Hulk domain core must not import context",
	"time":          "Hulk domain core must not import time",
	"os":            "Hulk domain core must not import os",
	"io":            "Hulk domain core must not import io",
	"io/fs":         "Hulk domain core must not import io/fs",
	"path/filepath": "Hulk domain core must not import path/filepath",
	"net":           "Hulk domain core must not import net",
	"net/http":      "Hulk domain core must not import HTTP/network packages",
	"net/url":       "Hulk domain core must not import network packages",
	"database/sql":  "Hulk domain core must not import database packages",
	"os/exec":       "Hulk domain core must not import process execution",
	"sync":          "Hulk domain core must not import sync primitives",
	"embed":         "Hulk domain core must not import embed",
	"encoding/json": "Hulk domain core must not import encoding packages",
}

// Provider/control-plane substrings forbidden for Hulk.
var hulkForbiddenContains = []string{
	"openai",
	"anthropic",
	"litellm",
	"ollama",
	"gemini",
	"bedrock",
	"azure",
	"oauth",
	"oidc",
	"jwt",
	"session",
	"cookie",
	"sqlite",
	"postgres",
	"mysql",
}

// Standard library imports allowed for Witness proxy seed.
var witnessAllowedImports = map[string]bool{
	"errors":            true,
	"net/http":          true,
	"net/http/httputil": true,
	"net/url":           true,
	"strings":           true,
	"sync":              true,
	"time":              true,
}

// Standard library imports forbidden for Witness proxy seed.
var witnessForbiddenImports = map[string]string{
	"database/sql":  "Witness proxy must not import database packages",
	"os":            "Witness proxy must not import os",
	"io/fs":         "Witness proxy must not import filesystem packages",
	"path/filepath": "Witness proxy must not import path packages",
	"os/exec":       "Witness proxy must not import process execution",
	"embed":         "Witness proxy must not import embed",
	"encoding/json": "Witness proxy must not import encoding packages",
	"html/template": "Witness proxy must not import template packages",
	"text/template": "Witness proxy must not import template packages",
}

// Provider/control-plane substrings forbidden for Witness.
var witnessForbiddenContains = []string{
	"openai",
	"anthropic",
	"litellm",
	"ollama",
	"gemini",
	"bedrock",
	"azure",
	"oauth",
	"oidc",
	"jwt",
	"session",
	"cookie",
	"sqlite",
	"postgres",
	"mysql",
}

// Standard library imports allowed for Web cockpit seed.
var cockpitAllowedImports = map[string]bool{
	"embed":         true,
	"encoding/json": true,
	"fmt":           true,
	"net/http":      true,
	"strings":       true,
}

// Standard library imports forbidden for Web cockpit seed.
var cockpitForbiddenImports = map[string]string{
	"database/sql":      "Web cockpit must not import database packages",
	"os":                "Web cockpit must not import os",
	"io/fs":             "Web cockpit must not import filesystem packages",
	"path/filepath":     "Web cockpit must not import path packages",
	"os/exec":           "Web cockpit must not import process execution",
	"net/http/httputil": "Web cockpit must not import reverse proxy utilities",
	"net/url":           "Web cockpit must not import URL packages",
	"sync":              "Web cockpit must not import sync primitives",
	"time":              "Web cockpit must not import time packages",
	"html/template":     "Web cockpit must not import HTML templates",
	"text/template":     "Web cockpit must not import text templates",
}

// Auth/provider/control-plane substrings forbidden for Cockpit.
var cockpitForbiddenContains = []string{
	"openai",
	"anthropic",
	"litellm",
	"ollama",
	"gemini",
	"bedrock",
	"azure",
	"oauth",
	"oidc",
	"jwt",
	"session",
	"cookie",
	"sqlite",
	"postgres",
	"mysql",
}

// policies defines the protected packages and their import constraints.
var policies = []PackagePolicy{
	{
		Name:              "hulk-runbundle",
		Dir:               "internal/hulk/runbundle",
		AllowedImports:    hulkAllowedImports,
		ForbiddenImports:  hulkForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(hulkForbiddenContains),
	},
	{
		Name:              "hulk-claimevidence",
		Dir:               "internal/hulk/claimevidence",
		AllowedImports:    hulkAllowedImports,
		ForbiddenImports:  hulkForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(hulkForbiddenContains),
	},
	{
		Name:              "witness-proxy",
		Dir:               "internal/witness/proxy",
		AllowedImports:    witnessAllowedImports,
		ForbiddenImports:  witnessForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(witnessForbiddenContains),
	},
	{
		Name:              "web-cockpit",
		Dir:               "internal/web/cockpit",
		AllowedImports:    cockpitAllowedImports,
		ForbiddenImports:  cockpitForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(cockpitForbiddenContains),
	},
}

func forbiddenContainsToMap(list []string) map[string]string {
	m := make(map[string]string)
	for _, s := range list {
		m[s] = "imports provider/control-plane package containing: " + s
	}
	return m
}

// Check verifies import boundaries for all protected packages.
func Check(repoRoot string) Result {
	var allFindings []Finding

	for _, policy := range policies {
		dirPath := filepath.Join(repoRoot, policy.Dir)
		findings := checkPackage(policy, dirPath)
		allFindings = append(allFindings, findings...)
	}

	// Sort findings for deterministic order
	sort.Slice(allFindings, func(i, j int) bool {
		if allFindings[i].File != allFindings[j].File {
			return allFindings[i].File < allFindings[j].File
		}
		if allFindings[i].Import != allFindings[j].Import {
			return allFindings[i].Import < allFindings[j].Import
		}
		return allFindings[i].Reason < allFindings[j].Reason
	})

	return Result{Findings: allFindings}
}

// checkPackage scans a single package directory for boundary violations.
func checkPackage(policy PackagePolicy, dirPath string) []Finding {
	var findings []Finding

	fset := token.NewFileSet()

	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			// Skip testdata and vendor directories
			name := d.Name()
			if name == "testdata" || name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan .go files, skip test files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileFindings := checkFile(policy, path, fset)
		findings = append(findings, fileFindings...)
		return nil
	})

	if err != nil {
		return findings
	}

	return findings
}

// checkFile parses a single Go file and checks its imports.
func checkFile(policy PackagePolicy, filePath string, fset *token.FileSet) []Finding {
	var findings []Finding

	f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return findings
	}

	relPath, _ := filepath.Rel(".", filePath)

	for _, imp := range f.Imports {
		if imp.Path == nil {
			continue
		}

		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check if import is explicitly forbidden
		if reason, forbidden := policy.ForbiddenImports[importPath]; forbidden {
			findings = append(findings, Finding{
				File:   relPath,
				Import: importPath,
				Reason: reason,
			})
			continue
		}

		// Check if import path contains forbidden substrings
		for substring, reason := range policy.ForbiddenContains {
			if strings.Contains(importPath, substring) {
				findings = append(findings, Finding{
					File:   relPath,
					Import: importPath,
					Reason: reason,
				})
				break
			}
		}
	}

	return findings
}

// CheckRepo returns findings as checks.Finding for integration with Factory gate.
func CheckRepo(root string) []checks.Finding {
	result := Check(root)

	cfindings := make([]checks.Finding, len(result.Findings))
	for i, f := range result.Findings {
		cfindings[i] = checks.Finding{
			Path:     f.File,
			Kind:     "boundary_violation",
			Message:  f.Import + ": " + f.Reason,
			Severity: checks.SeverityError,
		}
	}

	checks.SortFindings(cfindings)
	return cfindings
}
