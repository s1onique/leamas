// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
	"github.com/s1onique/leamas/internal/factory/registry"
)

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
//
// The analyzer passed in here is reserved for tests. Production callers
// should use the post-authority binder path (factorizeVerifiersWithAnalyzer)
// which constructs the analyzer inside an adapter wrapper, so the raw
// dupcode.CheckRepo function value is never captured directly in this
// package.
func FactorizeVerifiersWithDupcodeContext(root string) ([]registry.Verifier, error) {
	return factorizeVerifiersWithAnalyzer(root, nil)
}

// factorizeVerifiersWithAnalyzer wires a binder-local analyzer into the
// shared analysis context. The analyzer is injected (not pulled from a global)
// and is the only path through which the shared context performs a scan.
//
// Production callers MUST pass the analyzer returned by
// protectedverifier.NewAnalyzerFromAdapter (constructed post-authority
// inside the factory closure). Tests may pass any DupcodeAnalyzer.
func factorizeVerifiersWithAnalyzer(root string, analyzer protectedverifier.DupcodeAnalyzer) ([]registry.Verifier, error) {
	if analyzer == nil {
		analyzer = protectedverifier.NewAnalyzerFromAdapter()
	}

	// Narrow metadata-only read for threshold discovery — does not invoke
	// the protected LoadBaseline operation. Setup-time metadata only.
	minLines := protectedverifier.PolicyMinLines
	minTokens := protectedverifier.PolicyMinTokens

	baselinePath := ".factory/dupcode-baseline.json"
	if root != "." && root != "" {
		baselinePath = filepath.Join(root, baselinePath)
	}

	if min, tok, err := protectedverifier.ReadBaselineThresholds(baselinePath); err == nil {
		minLines = min
		minTokens = tok
	}

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

	sharedDupcodeVerifier := factory.SharedDupCodeVerifier()
	sharedDupcodeBaselineVerifier := factory.SharedDupcodeBaselineVerifier()

	verifiers := AllVerifiers()
	return replaceDupcodeVerifierRuns(verifiers, sharedDupcodeVerifier, sharedDupcodeBaselineVerifier)
}

// factorizeVerifiersWithDupcodeAnalyzer is retained for backward
// compatibility with existing tests that inject a custom analyzer
// directly. New code MUST use factorizeVerifiersWithAnalyzer instead so
// that the analyzer is always constructed via the adapter wrapper in
// production. The bare dupcode.CheckRepo reference is permitted here only
// because this function exists solely for test injection.
//
// Deprecated: prefer factorizeVerifiersWithAnalyzer with a nil analyzer,
// which installs the post-authority adapter wrapper.
func factorizeVerifiersWithDupcodeAnalyzer(root string, analyzer protectedverifier.DupcodeAnalyzer) ([]registry.Verifier, error) {
	return factorizeVerifiersWithAnalyzer(root, analyzer)
}

// Compile-time guard that the dupcode package is still imported (this file
// documents why that import is required by the legacy analyzer path).
var _ = dupcode.PolicyMinLines

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

	out := append([]registry.Verifier(nil), verifiers...)
	out[dupcodeIndex].Run = dupcodeRun
	out[baselineIndex].Run = baselineRun
	return out, nil
}
