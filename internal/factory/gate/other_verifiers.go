// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/agentcontext"
	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/coverage"
	"github.com/s1onique/leamas/internal/factory/forbidden"
	"github.com/s1onique/leamas/internal/factory/githooks"
	"github.com/s1onique/leamas/internal/factory/github"
	"github.com/s1onique/leamas/internal/factory/llmfriendly"
	"github.com/s1onique/leamas/internal/factory/longtestpolicy"
)

// llmFriendlyVerifier runs the LLM-friendly verifier.
func llmFriendlyVerifier(root string) []checks.Finding {
	cfg := llmfriendly.DefaultConfig()
	findings, _ := llmfriendly.CheckRepo(root, cfg)
	return convertLLMFriendlyFindings(findings)
}

// agentContextVerifier runs the agent context verifier.
func agentContextVerifier(root string) []checks.Finding {
	findings, _ := agentcontext.CheckRepo(root)
	return convertAgentContextFindings(findings)
}

// gitHooksVerifier runs the git hooks verifier.
func gitHooksVerifier(root string) []checks.Finding {
	findings, _ := githooks.CheckRepo(root)
	return convertGitHooksFindings(findings)
}

func convertLLMFriendlyFindings(src []llmfriendly.Finding) []checks.Finding {
	result := make([]checks.Finding, len(src))
	for i, f := range src {
		result[i] = checks.Finding{Path: f.Path, Kind: f.Kind, Message: f.Message, Severity: checks.SeverityError}
	}
	return result
}

func convertAgentContextFindings(src []agentcontext.Finding) []checks.Finding {
	result := make([]checks.Finding, len(src))
	for i, f := range src {
		result[i] = checks.Finding{Path: f.Path, Kind: f.Kind, Message: f.Message, Severity: checks.SeverityError}
	}
	return result
}

func convertGitHooksFindings(src []githooks.Finding) []checks.Finding {
	result := make([]checks.Finding, len(src))
	for i, f := range src {
		result[i] = checks.Finding{Path: f.Path, Kind: f.Kind, Message: f.Message, Severity: checks.SeverityError}
	}
	return result
}

func githubVerifier(root string) []checks.Finding {
	findings, _ := github.CheckRepo(root)
	return convertGithubFindings(findings)
}

func convertGithubFindings(src []github.Finding) []checks.Finding {
	result := make([]checks.Finding, len(src))
	for i, f := range src {
		severity := checks.SeverityError
		if f.Severity == "info" {
			severity = checks.SeverityWarn
		}
		result[i] = checks.Finding{Path: f.Path, Kind: f.Kind, Message: f.Message, Severity: severity}
	}
	return result
}

// CheckCoverage is the exported wrapper for coverage verification.
func CheckCoverage(root string) []checks.Finding {
	return coverageVerifier(root)
}

// coverageVerifier checks a pre-existing coverage profile against a threshold.
func coverageVerifier(root string) []checks.Finding {
	profilePath := ".factory/coverage.out"
	fullPath := profilePath
	if root != "." && root != "" {
		fullPath = filepath.Join(root, profilePath)
	}
	if !checks.FileExists(fullPath) {
		return []checks.Finding{{Path: profilePath, Kind: "missing_coverage_profile", Message: "coverage profile not found. Run 'make coverage' first.", Severity: checks.SeverityError}}
	}
	threshold := coverage.DefaultThreshold()
	_, err := coverage.Analyze(fullPath, threshold)
	if err != nil {
		return []checks.Finding{{Path: profilePath, Kind: "coverage_threshold_fail", Message: err.Error(), Severity: checks.SeverityError}}
	}
	return nil
}

// longTestPolicyVerifier checks that long-test policy is enforced:
// all RequireLongTest calls have registered baseline entries.
func longTestPolicyVerifier(root string) []checks.Finding {
	return longtestpolicy.CheckRepo(root)
}

// forbiddenPatternsVerifier runs the canonical AST-based dupcode bypass policy.
// This uses module-aware type checking with go/packages for accurate symbol identity resolution.
func forbiddenPatternsVerifier(root string) []checks.Finding {
	var findings []checks.Finding

	dupcodeFindings := forbidden.CanonicalCheckDupcodeBypass(root, "github.com/s1onique/leamas")
	findings = append(findings, dupcodeFindings...)

	legacyFindings := forbidden.CheckRepo(root)
	findings = append(findings, legacyFindings...)

	seen := make(map[string]bool)
	unique := findings[:0]
	for _, f := range findings {
		key := f.Path + "|" + f.Kind + "|" + f.Message
		if !seen[key] {
			seen[key] = true
			unique = append(unique, f)
		}
	}

	return unique
}

// ensure gofmt keeps the unused fmt import on a stable line.
var _ = fmt.Sprintf
