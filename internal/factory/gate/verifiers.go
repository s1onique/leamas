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
	"github.com/s1onique/leamas/internal/factory/execgate"
	"github.com/s1onique/leamas/internal/factory/forbidden"
	"github.com/s1onique/leamas/internal/factory/githooks"
	"github.com/s1onique/leamas/internal/factory/github"
	"github.com/s1onique/leamas/internal/factory/language"
	"github.com/s1onique/leamas/internal/factory/llmfriendly"
	"github.com/s1onique/leamas/internal/factory/longtestpolicy"
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/staticbinary"
	"github.com/s1onique/leamas/internal/factory/tooling"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// AllVerifiers returns all Factory policy verifiers (for factorize).
// This function uses independent dupcode verifiers and is used for
// direct commands like `leamas factory verify dupcode` and `leamas factory verify dupcode-baseline`.
// For factorize, use FactorizeVerifiersWithDupcodeContext instead.
func AllVerifiers() []registry.Verifier {
	return []registry.Verifier{
		{Name: "agent-context", Run: agentContextVerifier, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.agentContextVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "doctrine", Run: doctrine.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/doctrine.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "doctrine-agent-contracts", Run: doctrine.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/doctrine.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "docs", Run: docs.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/docs.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "domain-boundaries", Run: boundary.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/boundary.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "dupcode-baseline", Run: dupcodeBaselineVerifier, Lane: registry.VerifierLaneDupcode, Authority: verifierauthority.AuthorityCIExactCheckout, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.dupcodeBaselineVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "dupcode", Run: dupCodeVerifier, Lane: registry.VerifierLaneDupcode, Authority: verifierauthority.AuthorityCIExactCheckout, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.dupCodeVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "exec-gate", Run: execgate.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/execgate.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "executable-contract-first", Run: doctrine.CheckExecutableContractFirst, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/doctrine.CheckExecutableContractFirst", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "forbidden-patterns", Run: forbiddenPatternsVerifier, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.forbiddenPatternsVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "git-hooks", Run: gitHooksVerifier, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.gitHooksVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "language", Run: language.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/language.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "llm-friendly", Run: llmFriendlyVerifier, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.llmFriendlyVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "static-binary", Run: staticbinary.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/staticbinary.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED", "GOCACHE"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheRelevant, GoTestResultCache: registry.CacheModeNA}},
		{Name: "tooling-boundaries", Run: tooling.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/tooling.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "long-test-policy", Run: longTestPolicyVerifier, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/longtestpolicy.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
	}
}

// FactorizeVerifiersWithDupcodeContext returns all Factory policy verifiers for
// a factorize invocation. The dupcode and dupcode-baseline verifiers share a single
// analysis context so only one repository scan is performed.
//
// This function derives from AllVerifiers() and only replaces the Run functions
// for dupcode and dupcode-baseline. This ensures metadata (name, lane, authority, execution,
// cache, environment) stays in sync with the canonical registry.
//
// This function is used by RunFactorize. For direct commands like
// `leamas factory verify dupcode`, use AllVerifiers instead which performs
// independent scans per verifier.
func FactorizeVerifiersWithDupcodeContext(root string) ([]registry.Verifier, error) {
	return factorizeVerifiersWithDupcodeAnalyzer(root, nil)
}

// factorizeVerifiersWithDupcodeAnalyzer is the internal constructor that accepts an optional
// injected analyzer for testing the production registry wiring.
func factorizeVerifiersWithDupcodeAnalyzer(root string, analyzer protectedverifier.DupcodeAnalyzer) ([]registry.Verifier, error) {
	// Determine the effective dupcode thresholds from the baseline (if it exists)
	// This is a lightweight metadata read, not the expensive scan
	minLines := protectedverifier.PolicyMinLines
	minTokens := protectedverifier.PolicyMinTokens

	baselinePath := ".factory/dupcode-baseline.json"
	if root != "." && root != "" {
		baselinePath = filepath.Join(root, baselinePath)
	}

	// Try to load baseline thresholds if baseline exists
	if checks.FileExists(baselinePath) {
		if baseline, err := protectedverifier.LoadBaseline(baselinePath); err == nil {
			minLines = baseline.Thresholds.MinLines
			minTokens = baseline.Thresholds.MinTokens
		}
	}

	// Create shared analysis context with complete config
	// The expensive scan only happens when the verifier actually runs (after authorization)
	cfg := protectedverifier.DefaultConfig()
	cfg.Root = root
	cfg.MinLines = minLines
	cfg.MinTokens = minTokens

	input := protectedverifier.DupcodeInput{
		Root:      root,
		MinLines:  minLines,
		MinTokens: minTokens,
		Config:    cfg,
	}

	provider := protectedverifier.NewDupcodeAnalysisProvider(input, analyzer)
	ctx := protectedverifier.NewDupcodeAnalysisContext(provider)
	factory := protectedverifier.NewDupcodeVerifierFactory(ctx)

	// Create shared dupcode verifiers
	sharedDupcodeVerifier := factory.SharedDupCodeVerifier()
	sharedDupcodeBaselineVerifier := factory.SharedDupcodeBaselineVerifier()

	// Derive from AllVerifiers and only replace the Run functions for dupcode verifiers
	verifiers := AllVerifiers()
	return replaceDupcodeVerifierRuns(verifiers, sharedDupcodeVerifier, sharedDupcodeBaselineVerifier)
}

// replaceDupcodeVerifierRuns replaces the Run functions of the dupcode and
// dupcode-baseline entries in the provided registry. The replacement is
// failure-atomic: if either entry is missing the function returns an error
// and the caller's input slice is not mutated. The function is pure with
// respect to its inputs and is therefore the testable unit for the
// fail-closed registry replacement invariant.
func replaceDupcodeVerifierRuns(
	verifiers []registry.Verifier,
	dupcodeRun func(string) []checks.Finding,
	baselineRun func(string) []checks.Finding,
) ([]registry.Verifier, error) {
	dupcodeIndex := -1
	baselineIndex := -1

	for i := range verifiers {
		switch verifiers[i].Name {
		case "dupcode":
			dupcodeIndex = i
		case "dupcode-baseline":
			baselineIndex = i
		}
	}

	if dupcodeIndex < 0 || baselineIndex < 0 {
		return nil, fmt.Errorf(
			"shared dupcode registry replacement incomplete: dupcode=%t dupcode-baseline=%t",
			dupcodeIndex >= 0,
			baselineIndex >= 0,
		)
	}

	// Failure-atomic: copy the input slice before mutating so the
	// caller's input slice is never observable as partially replaced.
	out := append([]registry.Verifier(nil), verifiers...)
	out[dupcodeIndex].Run = dupcodeRun
	out[baselineIndex].Run = baselineRun
	return out, nil
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

// dupCodeVerifier is the verifier for dupcode baseline comparison.
// This is used in AllVerifiers for direct commands and as fallback.
func dupCodeVerifier(root string) []checks.Finding {
	runner := protectedverifier.NewDupcodeRunner()
	cfg := protectedverifier.DefaultConfig()

	baselinePath := ".factory/dupcode-baseline.json"
	fullBaselinePath := baselinePath
	if root != "." && root != "" {
		fullBaselinePath = filepath.Join(root, baselinePath)
	}

	if !checks.FileExists(fullBaselinePath) {
		return []checks.Finding{
			{Path: baselinePath, Kind: "missing_baseline", Message: "baseline file not found. Run 'make dupcode-baseline' to create it.", Severity: checks.SeverityError},
		}
	}

	baseline, err := runner.LoadBaseline(fullBaselinePath)
	if err != nil {
		return []checks.Finding{
			{Path: baselinePath, Kind: "baseline_error", Message: fmt.Sprintf("failed to load baseline: %v", err), Severity: checks.SeverityError},
		}
	}

	cfg.Root = root
	cfg.MinLines = baseline.Thresholds.MinLines
	cfg.MinTokens = baseline.Thresholds.MinTokens

	report, err := runner.RunCheckReport(root, cfg)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error", Message: fmt.Sprintf("duplicate code scan failed: %v", err), Severity: checks.SeverityError},
		}
	}

	result := runner.CompareToBaseline(report, baseline)
	return convertDupcodeCompareResult(result)
}

// dupcodeBaselineVerifier is the verifier for dupcode baseline verification.
// This is used in AllVerifiers for direct commands and as fallback.
func dupcodeBaselineVerifier(root string) []checks.Finding {
	runner := protectedverifier.NewDupcodeRunner()
	policy := protectedverifier.BaselinePolicy{
		Path: ".factory/dupcode-baseline.json",
	}
	if root != "." && root != "" {
		policy.Path = filepath.Join(root, policy.Path)
	}

	findings, err := runner.VerifyBaseline(root, policy)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error", Message: fmt.Sprintf("baseline verification failed: %v", err), Severity: checks.SeverityError},
		}
	}
	return findings
}

// forbiddenPatternsVerifier runs the canonical AST-based dupcode bypass policy.
func forbiddenPatternsVerifier(root string) []checks.Finding {
	// Use the canonical V2 policy with repository-wide scanning and symbol awareness
	findings := forbidden.CanonicalCheckDupcodeBypassV2(root, "github.com/s1onique/leamas")

	// Also run the legacy forbidden patterns check for completeness
	// (patterns like OIDC, OAuth, RBAC, etc.)
	legacyFindings := forbidden.CheckRepo(root)

	// Merge findings, avoiding duplicates
	existingKinds := make(map[string]bool)
	result := make([]checks.Finding, 0, len(findings)+len(legacyFindings))

	for _, f := range findings {
		key := f.Path + "|" + f.Kind + "|" + f.Message
		if !existingKinds[key] {
			existingKinds[key] = true
			result = append(result, f)
		}
	}

	for _, f := range legacyFindings {
		key := f.Path + "|" + f.Kind + "|" + f.Message
		if !existingKinds[key] {
			existingKinds[key] = true
			result = append(result, f)
		}
	}

	return result
}

// convertDupcodeCompareResult converts dupcode comparison results to gate findings.
func convertDupcodeCompareResult(result protectedverifier.CompareResult) []checks.Finding {
	var findings []checks.Finding
	if !result.HasChanges {
		return findings
	}

	for _, f := range result.NewFindings {
		findings = append(findings, checks.Finding{
			Path:     "dupcode",
			Kind:     "new_duplicate",
			Message:  fmt.Sprintf("new duplicate block (tokens: %d)", f.TokenCount),
			Severity: checks.SeverityError,
		})
	}

	for range result.WorsenedFindings {
		findings = append(findings, checks.Finding{
			Path:     "dupcode",
			Kind:     "worsened_duplicate",
			Message:  fmt.Sprintf("worsened duplicate block"),
			Severity: checks.SeverityError,
		})
	}

	return findings
}
