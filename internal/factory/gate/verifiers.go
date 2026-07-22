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
	"github.com/s1onique/leamas/internal/factory/longtestpolicy"
	"github.com/s1onique/leamas/internal/factory/staticbinary"
	"github.com/s1onique/leamas/internal/factory/tooling"
)

// AllVerifiers returns all Factory policy verifiers (for factorize).
// This function uses independent dupcode verifiers and is used for
// direct commands like `leamas factory verify dupcode` and `leamas factory verify dupcode-baseline`.
// For factorize, use FactorizeVerifiersWithDupcodeContext instead.
func AllVerifiers() []Verifier {
	return []Verifier{
		{Name: "agent-context", Run: agentContextVerifier, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/gate.agentContextVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "doctrine", Run: doctrine.CheckRepo, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/doctrine.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "doctrine-agent-contracts", Run: doctrine.CheckRepo, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/doctrine.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "docs", Run: docs.CheckRepo, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/docs.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "domain-boundaries", Run: boundary.CheckRepo, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/boundary.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "dupcode-baseline", Run: dupcodeBaselineVerifier, Lane: VerifierLaneDupcode, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/gate.dupcodeBaselineVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "dupcode", Run: dupCodeVerifier, Lane: VerifierLaneDupcode, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/gate.dupCodeVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "exec-gate", Run: execgate.CheckRepo, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/execgate.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "executable-contract-first", Run: doctrine.CheckExecutableContractFirst, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/doctrine.CheckExecutableContractFirst", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "forbidden-patterns", Run: forbidden.CheckRepo, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/forbidden.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "git-hooks", Run: gitHooksVerifier, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/gate.gitHooksVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "language", Run: language.CheckRepo, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/language.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "llm-friendly", Run: llmFriendlyVerifier, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/gate.llmFriendlyVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "static-binary", Run: staticbinary.CheckRepo, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/staticbinary.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED", "GOCACHE"},
		}, Cache: CacheSemantics{GoBuildCache: CacheRelevant, GoTestResultCache: CacheModeNA}},
		{Name: "tooling-boundaries", Run: tooling.CheckRepo, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/tooling.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
		{Name: "long-test-policy", Run: longTestPolicyVerifier, Lane: VerifierLaneFast, Execution: ExecutionDefinition{
			Kind: ExecutionInProcess, ImplementationID: "internal/factory/longtestpolicy.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: CacheSemantics{GoBuildCache: CacheNotApplicable, GoTestResultCache: CacheModeNA}},
	}
}

// FactorizeVerifiersWithDupcodeContext returns all Factory policy verifiers for
// a factorize invocation. The dupcode and dupcode-baseline verifiers share a single
// analysis context so only one repository scan is performed.
//
// This function derives from AllVerifiers() and only replaces the Run functions
// for dupcode and dupcode-baseline. This ensures metadata (name, lane, execution,
// cache, environment) stays in sync with the canonical registry.
//
// This function is used by RunFactorize. For direct commands like
// `leamas factory verify dupcode`, use AllVerifiers instead which performs
// independent scans per verifier.
func FactorizeVerifiersWithDupcodeContext(root string) ([]Verifier, error) {
	// Determine the effective dupcode thresholds from the baseline (if it exists)
	minLines := dupcode.PolicyMinLines
	minTokens := dupcode.PolicyMinTokens

	baselinePath := ".factory/dupcode-baseline.json"
	if root != "." && root != "" {
		baselinePath = filepath.Join(root, baselinePath)
	}

	// Try to load baseline thresholds if baseline exists
	if checks.FileExists(baselinePath) {
		if baseline, err := dupcode.LoadBaseline(baselinePath); err == nil {
			minLines = baseline.Thresholds.MinLines
			minTokens = baseline.Thresholds.MinTokens
		}
	}

	// Create shared analysis context with complete config
	cfg := dupcode.DefaultConfig()
	cfg.Root = root
	cfg.MinLines = minLines
	cfg.MinTokens = minTokens
	provider := NewDupcodeAnalysisProvider(newDupcodeInput(cfg), nil) // nil uses default dupcode.CheckRepo

	ctx := NewDupcodeAnalysisContext(provider)
	factory := NewDupcodeVerifierFactory(ctx)

	// Create shared dupcode verifiers
	sharedDupcodeVerifier := factory.SharedDupCodeVerifier()
	sharedDupcodeBaselineVerifier := factory.SharedDupcodeBaselineVerifier()

	// Derive from AllVerifiers and only replace the Run functions for dupcode verifiers
	verifiers := AllVerifiers()
	replacedDupcode := false
	replacedBaseline := false
	for i := range verifiers {
		switch verifiers[i].Name {
		case "dupcode":
			verifiers[i].Run = sharedDupcodeVerifier
			replacedDupcode = true
		case "dupcode-baseline":
			verifiers[i].Run = sharedDupcodeBaselineVerifier
			replacedBaseline = true
		}
	}

	// Fail-closed: both registry entries must be replaced
	if !replacedDupcode || !replacedBaseline {
		return nil, fmt.Errorf(
			"shared dupcode registry replacement incomplete: dupcode=%t dupcode-baseline=%t",
			replacedDupcode, replacedBaseline,
		)
	}

	return verifiers, nil
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
