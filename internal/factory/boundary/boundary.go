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
//
// CLI runtime files:
//   - cmd/leamas/cockpit.go: CLI cockpit serve command
//   - cmd/leamas/witness.go: CLI witness proxy command
package boundary

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// PackagePolicy defines import constraints for a protected package.
type PackagePolicy struct {
	Name              string
	Dir               string
	AllowedImports    map[string]bool
	ForbiddenImports  map[string]string
	ForbiddenContains []string // Ordered list for deterministic checking
}

// FilePolicy defines import constraints for a specific CLI runtime file.
type FilePolicy struct {
	Name              string
	File              string
	AllowedImports    map[string]bool
	ForbiddenImports  map[string]string
	ForbiddenContains []string // Ordered list for deterministic checking
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

// Provider/control-plane substrings forbidden for Hulk (ordered for determinism).
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

// Provider/control-plane substrings forbidden for Witness (ordered for determinism).
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

// Auth/provider/control-plane substrings forbidden for Cockpit (ordered for determinism).
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

// CLI runtime allowed imports for cockpit and witness command files.
var cliRuntimeAllowedImports = map[string]bool{
	"context":   true,
	"errors":    true,
	"flag":      true,
	"fmt":       true,
	"io":        true,
	"net":       true,
	"net/http":  true,
	"os":        true,
	"os/signal": true,
	"strconv":   true,
	"strings":   true,
	"syscall":   true,
	"time":      true,
}

// CLI runtime forbidden imports.
var cliRuntimeForbiddenImports = map[string]string{
	"database/sql":  "CLI runtime must not import database packages",
	"os/exec":       "CLI runtime must not import process execution",
	"embed":         "CLI runtime must not import embed",
	"html/template": "CLI runtime must not import HTML templates",
	"text/template": "CLI runtime must not import text templates",
}

// CLI runtime allowed internal imports.
var cliRuntimeAllowedInternal = map[string]bool{
	"github.com/s1onique/leamas/internal/web/cockpit":   true,
	"github.com/s1onique/leamas/internal/witness/proxy": true,
}

// CLI runtime forbidden internal imports.
var cliRuntimeForbiddenInternal = map[string]string{
	"github.com/s1onique/leamas/internal/hulk/runbundle":     "CLI runtime must not import Hulk runbundle package",
	"github.com/s1onique/leamas/internal/hulk/claimevidence": "CLI runtime must not import Hulk claimevidence package",
}

// Provider/control-plane substrings forbidden for CLI runtime (ordered for determinism).
var cliRuntimeForbiddenContains = []string{
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

// filePolicies defines CLI runtime files with their import constraints.
var filePolicies = []FilePolicy{
	{
		Name:              "cockpit-cli-runtime",
		File:              "cmd/leamas/cockpit.go",
		AllowedImports:    cliRuntimeAllowedImports,
		ForbiddenImports:  cliRuntimeForbiddenImports,
		ForbiddenContains: cliRuntimeForbiddenContains,
	},
	{
		Name:              "witness-cli-runtime",
		File:              "cmd/leamas/witness.go",
		AllowedImports:    cliRuntimeAllowedImports,
		ForbiddenImports:  cliRuntimeForbiddenImports,
		ForbiddenContains: cliRuntimeForbiddenContains,
	},
}

// policies defines the protected packages and their import constraints.
var policies = []PackagePolicy{
	{
		Name:              "hulk-runbundle",
		Dir:               "internal/hulk/runbundle",
		AllowedImports:    hulkAllowedImports,
		ForbiddenImports:  hulkForbiddenImports,
		ForbiddenContains: hulkForbiddenContains,
	},
	{
		Name:              "hulk-claimevidence",
		Dir:               "internal/hulk/claimevidence",
		AllowedImports:    hulkAllowedImports,
		ForbiddenImports:  hulkForbiddenImports,
		ForbiddenContains: hulkForbiddenContains,
	},
	{
		Name:              "witness-proxy",
		Dir:               "internal/witness/proxy",
		AllowedImports:    witnessAllowedImports,
		ForbiddenImports:  witnessForbiddenImports,
		ForbiddenContains: witnessForbiddenContains,
	},
	{
		Name:              "web-cockpit",
		Dir:               "internal/web/cockpit",
		AllowedImports:    cockpitAllowedImports,
		ForbiddenImports:  cockpitForbiddenImports,
		ForbiddenContains: cockpitForbiddenContains,
	},
}

// forbiddenContainsReason generates a deterministic reason for a forbidden substring.
func forbiddenContainsReason(substring string) string {
	return "imports provider/control-plane package containing: " + substring
}

// Check verifies import boundaries for all protected packages.
func Check(repoRoot string) Result {
	var allFindings []Finding

	// Check protected packages
	for _, policy := range policies {
		dirPath := filepath.Join(repoRoot, policy.Dir)

		// Check if protected directory exists
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			// Report missing protected directory as a finding
			allFindings = append(allFindings, Finding{
				File:   policy.Dir,
				Import: "(missing directory)",
				Reason: "protected package directory does not exist",
			})
			continue
		}

		findings := checkPackage(policy, dirPath, repoRoot)
		allFindings = append(allFindings, findings...)
	}

	// Check CLI runtime files
	for _, policy := range filePolicies {
		filePath := filepath.Join(repoRoot, policy.File)

		// Check if CLI runtime file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			allFindings = append(allFindings, Finding{
				File:   policy.File,
				Import: "(missing file)",
				Reason: "expected CLI runtime file does not exist",
			})
			continue
		}

		findings := checkCLIFile(policy, filePath, repoRoot)
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
