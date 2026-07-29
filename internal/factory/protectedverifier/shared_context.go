// SPDX-License-Identifier: Apache-2.0

package protectedverifier

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// NOTE: This file imports dupcode directly because it IS the protectedverifier adapter.
// The dupcode import is the only exception - all other code must go through protectedverifier.

// DupcodeAnalysisProvider manages a single-scan context for dupcode analysis.
// This is the ONLY production entry point for shared dupcode analysis context.
type DupcodeAnalysisProvider struct {
	mu       sync.Mutex
	state    providerState
	input    DupcodeInput
	analysis *DupcodeAnalysis
	analyzer DupcodeAnalyzer
}

type providerState int

const (
	providerStateEmpty providerState = iota
	providerStateConsuming
	providerStateConsumed
)

// DupcodeInput represents the input configuration for a dupcode analysis.
type DupcodeInput struct {
	Root      string
	MinLines  int
	MinTokens int
	Config    dupcode.Config
}

// DupcodeAnalysis represents the immutable result of a dupcode scan.
type DupcodeAnalysis struct {
	Findings    []dupcode.Finding
	Occurrences []dupcode.Occurrence
	Config      dupcode.Config
}

// NewDupcodeAnalysisProvider creates a provider with the given input and analyzer.
// The analyzer must be supplied explicitly; the package holds no global
// analyzer state and never falls back to a default. The same analyzer is
// bound to this provider's lifetime and is not visible to other providers.
func NewDupcodeAnalysisProvider(input DupcodeInput, analyzer DupcodeAnalyzer) *DupcodeAnalysisProvider {
	if analyzer == nil {
		// Fail closed: no global analyzer may be used.
		panic("protectedverifier: DupcodeAnalyzer must be injected; no global analyzer is available")
	}
	return &DupcodeAnalysisProvider{
		state:    providerStateEmpty,
		input:    input,
		analyzer: analyzer,
	}
}

// ConsumedBy performs the analysis and returns the result.
// The analysis is performed exactly once, with subsequent calls returning the cached result.
func (p *DupcodeAnalysisProvider) ConsumedBy(name string, input DupcodeInput) (*DupcodeAnalysis, error) {
	if input.Root != p.input.Root || input.MinLines != p.input.MinLines || input.MinTokens != p.input.MinTokens {
		return nil, fmt.Errorf("dupcode analysis input mismatch for %s: "+
			"got root=%s minLines=%d minTokens=%d, want root=%s minLines=%d minTokens=%d",
			name, input.Root, input.MinLines, input.MinTokens,
			p.input.Root, p.input.MinLines, p.input.MinTokens)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.state {
	case providerStateEmpty:
		p.state = providerStateConsuming
		findings, err := p.analyzer(input.Root, input.Config)
		if err != nil {
			p.state = providerStateEmpty
			return nil, err
		}

		var occurrences []dupcode.Occurrence
		for _, f := range findings {
			occurrences = append(occurrences, f.Occurrences...)
		}

		p.analysis = &DupcodeAnalysis{
			Findings:    findings,
			Occurrences: occurrences,
			Config:      input.Config,
		}
		p.state = providerStateConsumed
		return p.analysis, nil

	case providerStateConsuming:
		return nil, fmt.Errorf("dupcode analysis: concurrent scan detected (programming error)")

	case providerStateConsumed:
		return p.analysis, nil

	default:
		return nil, fmt.Errorf("dupcode analysis: unexpected state %v", p.state)
	}
}

// DupcodeAnalysisContext provides the shared analysis context for factorize.
type DupcodeAnalysisContext struct {
	provider *DupcodeAnalysisProvider
}

// NewDupcodeAnalysisContext creates a new analysis context with the given provider.
func NewDupcodeAnalysisContext(provider *DupcodeAnalysisProvider) *DupcodeAnalysisContext {
	return &DupcodeAnalysisContext{provider: provider}
}

// Provider returns the underlying analysis provider.
func (c *DupcodeAnalysisContext) Provider() *DupcodeAnalysisProvider {
	return c.provider
}

// DupcodeVerifierFactory creates verifiers that share the analysis context.
type DupcodeVerifierFactory struct {
	context *DupcodeAnalysisContext
}

// NewDupcodeVerifierFactory creates a factory for dupcode verifiers.
func NewDupcodeVerifierFactory(context *DupcodeAnalysisContext) *DupcodeVerifierFactory {
	return &DupcodeVerifierFactory{context: context}
}

// SharedDupCodeVerifier returns a verifier function that uses the shared
// analysis context. All raw dupcode operations are invoked through the
// DupcodeRunner adapter; the returned closure does NOT import dupcode
// symbols directly to invoke them.
func (f *DupcodeVerifierFactory) SharedDupCodeVerifier() func(string) []checks.Finding {
	return func(root string) []checks.Finding {
		return runSharedDupcodeVerify(f.context, root)
	}
}

// SharedDupcodeBaselineVerifier returns a verifier function that uses the
// shared analysis context. All raw dupcode operations are invoked through
// the DupcodeRunner adapter.
func (f *DupcodeVerifierFactory) SharedDupcodeBaselineVerifier() func(string) []checks.Finding {
	return func(root string) []checks.Finding {
		return runSharedDupcodeBaseline(f.context, root)
	}
}

// runSharedDupcodeVerify is a named function (not a closure literal) that
// performs the shared-context dupcode verify logic. It exists as a named
// function so the policy scanner can resolve caller identity to a real
// declaration rather than a func@line:col literal.
func runSharedDupcodeVerify(ctx *DupcodeAnalysisContext, root string) []checks.Finding {
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

	// Use the adapter for the protected LoadBaseline operation.
	runner := NewDupcodeRunner()
	baseline, err := runner.LoadBaseline(fullBaselinePath)
	if err != nil {
		return []checks.Finding{
			{Path: baselinePath, Kind: "baseline_error", Message: fmt.Sprintf("failed to load baseline: %v", err), Severity: checks.SeverityError},
		}
	}

	cfg := dupcode.DefaultConfig()
	cfg.Root = root
	cfg.MinLines = baseline.Thresholds.MinLines
	cfg.MinTokens = baseline.Thresholds.MinTokens

	input := DupcodeInput{
		Root:      root,
		MinLines:  cfg.MinLines,
		MinTokens: cfg.MinTokens,
		Config:    cfg,
	}

	analysis, err := ctx.Provider().ConsumedBy("dupcode", input)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error", Message: fmt.Sprintf("duplicate code scan failed: %v", err), Severity: checks.SeverityError},
		}
	}

	report := dupcode.Report{
		Findings: analysis.Findings,
		Thresholds: dupcode.BaselineThresholds{
			MinLines:  cfg.MinLines,
			MinTokens: cfg.MinTokens,
		},
	}

	// Use the adapter for the protected CompareToBaseline operation.
	result := runner.CompareToBaseline(report, baseline)
	return convertCompareResult(result)
}

// runSharedDupcodeBaseline is a named function (not a closure literal) that
// performs the shared-context dupcode-baseline verify logic.
func runSharedDupcodeBaseline(ctx *DupcodeAnalysisContext, root string) []checks.Finding {
	policy := dupcode.DefaultBaselinePolicy()
	policy.Path = ".factory/dupcode-baseline.json"
	if root != "." && root != "" {
		policy.Path = filepath.Join(root, policy.Path)
	}

	validation, err := dupcode.ValidateBaselineArtifact(root, policy)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "baseline_error", Message: fmt.Sprintf("baseline validation failed: %v", err), Severity: checks.SeverityError},
		}
	}

	cfg := dupcode.DefaultConfig()
	cfg.Root = root
	cfg.MinLines = validation.Baseline.Thresholds.MinLines
	cfg.MinTokens = validation.Baseline.Thresholds.MinTokens

	input := DupcodeInput{
		Root:      root,
		MinLines:  cfg.MinLines,
		MinTokens: cfg.MinTokens,
		Config:    cfg,
	}

	analysis, err := ctx.Provider().ConsumedBy("dupcode-baseline", input)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error", Message: fmt.Sprintf("duplicate code scan failed: %v", err), Severity: checks.SeverityError},
		}
	}

	report := dupcode.Report{
		Findings: analysis.Findings,
		Thresholds: dupcode.BaselineThresholds{
			MinLines:  cfg.MinLines,
			MinTokens: cfg.MinTokens,
		},
	}

	driftPolicy := dupcode.DefaultBaselinePolicy()
	driftPolicy.Path = policy.Path
	driftFindings := dupcode.CheckBaselineDriftFromReport(root, validation.Baseline, report, driftPolicy)

	var findings []checks.Finding
	for _, df := range driftFindings {
		findings = append(findings, checks.Finding{
			Path:     df.Path,
			Kind:     df.Kind,
			Message:  df.Message,
			Severity: checks.SeverityError,
		})
	}
	return findings
}

// convertCompareResult converts dupcode comparison results to gate findings.
func convertCompareResult(result dupcode.CompareResult) []checks.Finding {
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
