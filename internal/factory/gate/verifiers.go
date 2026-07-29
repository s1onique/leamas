// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"github.com/s1onique/leamas/internal/factory/boundary"
	"github.com/s1onique/leamas/internal/factory/docs"
	"github.com/s1onique/leamas/internal/factory/doctrine"
	"github.com/s1onique/leamas/internal/factory/execgate"
	"github.com/s1onique/leamas/internal/factory/language"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/staticbinary"
	"github.com/s1onique/leamas/internal/factory/tooling"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// AllVerifiers returns all Factory policy verifiers (for factorize).
// This function uses independent dupcode verifiers and is used for
// direct commands like `leamas factory verify dupcode` and `leamas factory verify dupcode-baseline`.
// For factorize, use FactorizeVerifiersWithDupcodeContext instead.
//
// All entries carry an explicit Scope. Gate-scoped entries are the
// canonical gate / factorize population. The dupcode-update-baseline
// entry is the command-only mutation identity and is excluded from
// gate / factorize selection by SelectVerifiers / PartitionVerifiers.
func AllVerifiers() []registry.Verifier {
	return []registry.Verifier{
		{Name: "agent-context", Run: agentContextVerifier, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.agentContextVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "doctrine", Run: doctrine.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/doctrine.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "doctrine-agent-contracts", Run: doctrine.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/doctrine.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "docs", Run: docs.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/docs.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "domain-boundaries", Run: boundary.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/boundary.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "dupcode-baseline", Run: dupcodeBaselineVerifier, Lane: registry.VerifierLaneDupcode, Authority: verifierauthority.AuthorityCIExactCheckout, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.dupcodeBaselineVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		// dupcode-update-baseline is the typed command-only mutation
		// identity. It is reachable via DispatcherForVerifier but is
		// excluded from gate / factorize selection. It must NOT
		// supply a Run function: the typed binder is the only
		// execution path.
		{Name: "dupcode-update-baseline", Lane: registry.VerifierLaneDupcode, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationCommandOnly, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.DupcodeUpdateBaselineTyped", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "dupcode", Run: dupCodeVerifier, Lane: registry.VerifierLaneDupcode, Authority: verifierauthority.AuthorityCIExactCheckout, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.dupCodeVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "exec-gate", Run: execgate.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/execgate.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "executable-contract-first", Run: doctrine.CheckExecutableContractFirst, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/doctrine.CheckExecutableContractFirst", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "forbidden-patterns", Run: forbiddenPatternsVerifier, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.forbiddenPatternsVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "git-hooks", Run: gitHooksVerifier, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.gitHooksVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "language", Run: language.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/language.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "llm-friendly", Run: llmFriendlyVerifier, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/gate.llmFriendlyVerifier", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "static-binary", Run: staticbinary.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/staticbinary.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED", "GOCACHE"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheRelevant, GoTestResultCache: registry.CacheModeNA}},
		{Name: "tooling-boundaries", Run: tooling.CheckRepo, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/tooling.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
		{Name: "long-test-policy", Run: longTestPolicyVerifier, Lane: registry.VerifierLaneFast, Authority: verifierauthority.AuthorityLocalSafe, Scope: registry.InvocationGate, Execution: registry.ExecutionDefinition{
			Kind: registry.ExecutionInProcess, ImplementationID: "internal/factory/longtestpolicy.CheckRepo", EnvVars: []string{"GOFLAGS", "CGO_ENABLED"},
		}, Cache: registry.CacheSemantics{GoBuildCache: registry.CacheNotApplicable, GoTestResultCache: registry.CacheModeNA}},
	}
}

// GateVerifiers returns the gate-scoped subset of AllVerifiers. The
// command-only definitions (e.g. dupcode-update-baseline) are excluded.
// This is the canonical population for RunGate and RunFactorize.
func GateVerifiers() []registry.Verifier {
	all := AllVerifiers()
	out := make([]registry.Verifier, 0, len(all))
	for _, v := range all {
		if v.Scope == registry.InvocationGate {
			out = append(out, v)
		}
	}
	return out
}
