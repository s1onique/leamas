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
func FactorizeVerifiersWithDupcodeContext(root string) ([]registry.Verifier, error) {
	return factorizeVerifiersWithDupcodeAnalyzer(root, dupcode.CheckRepo)
}

// factorizeVerifiersWithDupcodeAnalyzer wires a binder-local analyzer into the
// shared analysis context. The analyzer is injected (not pulled from a global)
// and is the only path through which the shared context performs a scan.
//
// Production callers pass dupcode.CheckRepo as the analyzer. Tests pass a fake
// analyzer that returns canned findings without touching the filesystem.
func factorizeVerifiersWithDupcodeAnalyzer(root string, analyzer protectedverifier.DupcodeAnalyzer) ([]registry.Verifier, error) {
	if analyzer == nil {
		return nil, fmt.Errorf("factorize: dupcode analyzer must be injected; no global analyzer is available")
	}

	minLines := protectedverifier.PolicyMinLines
	minTokens := protectedverifier.PolicyMinTokens

	baselinePath := ".factory/dupcode-baseline.json"
	if root != "." && root != "" {
		baselinePath = filepath.Join(root, baselinePath)
	}

	// Baseline metadata read for threshold discovery. This is configuration
	// loading only — no scan is performed here. The actual scan happens
	// later through the bound runner after authority admission.
	if checks.FileExists(baselinePath) {
		if baseline, err := dupcode.LoadBaseline(baselinePath); err == nil {
			minLines = baseline.Thresholds.MinLines
			minTokens = baseline.Thresholds.MinTokens
		}
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
