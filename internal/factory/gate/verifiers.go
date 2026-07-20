// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"fmt"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/agentcontext"
	"github.com/s1onique/leamas/internal/factory/boundary"
	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/coverage"
	"github.com/s1onique/leamas/internal/factory/docs"
	"github.com/s1onique/leamas/internal/factory/doctrine"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/execgate"
	"github.com/s1onique/leamas/internal/factory/forbidden"
	"github.com/s1onique/leamas/internal/factory/githooks"
	"github.com/s1onique/leamas/internal/factory/github"
	"github.com/s1onique/leamas/internal/factory/language"
	"github.com/s1onique/leamas/internal/factory/llmfriendly"
	"github.com/s1onique/leamas/internal/factory/staticbinary"
	"github.com/s1onique/leamas/internal/factory/tooling"
)

// AllVerifiers returns all Factory policy verifiers (for factorize).
func AllVerifiers() []Verifier {
	return []Verifier{
		{Name: "agent-context", Run: agentContextVerifier, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"agent-context"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "doctrine", Run: doctrine.CheckRepo, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"doctrine"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "doctrine-agent-contracts", Run: doctrine.CheckRepo, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"doctrine-agent-contracts"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "docs", Run: docs.CheckRepo, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"docs"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "domain-boundaries", Run: boundary.CheckRepo, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"domain-boundaries"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "dupcode-baseline", Run: dupcodeBaselineVerifier, Execution: ExecutionDefinition{
			Kind: ExecutionChild, LogicalArgv: []string{"leamas", "factory", "verify", "dupcode-baseline"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED", "GOCACHE"},
		}, Cache: CacheSemantics{GoBuildCache: CacheRelevant, GoTestResultCache: CacheModeDisabled}},
		{Name: "dupcode", Run: dupCodeVerifier, Execution: ExecutionDefinition{
			Kind: ExecutionChild, LogicalArgv: []string{"leamas", "factory", "verify", "dupcode"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED", "GOCACHE"},
		}, Cache: CacheSemantics{GoBuildCache: CacheRelevant, GoTestResultCache: CacheModeDisabled}},
		{Name: "exec-gate", Run: execgate.CheckRepo, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"exec-gate"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "executable-contract-first", Run: doctrine.CheckExecutableContractFirst, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"executable-contract-first"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "forbidden-patterns", Run: forbidden.CheckRepo, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"forbidden-patterns"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "git-hooks", Run: gitHooksVerifier, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"git-hooks"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "language", Run: language.CheckRepo, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"language"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "llm-friendly", Run: llmFriendlyVerifier, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"llm-friendly"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "static-binary", Run: staticbinary.CheckRepo, Execution: ExecutionDefinition{
			Kind: ExecutionChild, LogicalArgv: []string{"leamas", "factory", "verify", "static-binary"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED", "GOCACHE"},
		}, Cache: CacheSemantics{GoBuildCache: CacheRelevant, GoTestResultCache: CacheModeNA}},
		{Name: "tooling-boundaries", Run: tooling.CheckRepo, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, LogicalArgv: []string{"tooling-boundaries"}, EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
	}
}

func llmFriendlyVerifier(root string) []checks.Finding {
	cfg := llmfriendly.DefaultConfig()
	findings, _ := llmfriendly.CheckRepo(root, cfg)
	return convertLLMFriendlyFindings(findings)
}

func agentContextVerifier(root string) []checks.Finding {
	findings, _ := agentcontext.CheckRepo(root)
	return convertAgentContextFindings(findings)
}

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

func dupcodeBaselineVerifier(root string) []checks.Finding {
	policy := dupcode.DefaultBaselinePolicy()
	findings, err := dupcode.VerifyBaseline(root, policy)
	if err != nil {
		return []checks.Finding{{Path: policy.Path, Kind: "baseline_verification_error", Message: fmt.Sprintf("baseline verification failed: %v", err), Severity: checks.SeverityError}}
	}
	return findings
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

// dupCodeVerifier is a placeholder - actual implementation in dupcode package.
func dupCodeVerifier(root string) []checks.Finding {
	return []checks.Finding{}
}

// CheckCoverage is the exported wrapper for coverage verification.
func CheckCoverage(root string) []checks.Finding {
	return coverageVerifier(root)
}

// coverageVerifier checks a pre-existing coverage profile against a threshold.
func coverageVerifier(root string) []checks.Finding {
	var findings []checks.Finding
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
	return findings
}
