// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
	"github.com/s1onique/leamas/internal/factory/registry"
)

// factorizeDupcodeDeps is an immutable, invocation-local capability set.
// Construction stores these functions but does not invoke them.
type factorizeDupcodeDeps struct {
	ReadThresholds func(string) (int, int, error)
	NewAnalyzer    func() protectedverifier.DupcodeAnalyzer
	NewProvider    func(
		protectedverifier.DupcodeInput,
		protectedverifier.DupcodeAnalyzer,
	) *protectedverifier.DupcodeAnalysisProvider
}

func productionFactorizeDupcodeDeps() factorizeDupcodeDeps {
	return factorizeDupcodeDeps{
		ReadThresholds: readFactorizeDupcodeThresholds,
		NewAnalyzer:    newFactorizeDupcodeAnalyzer,
		NewProvider:    protectedverifier.NewDupcodeAnalysisProvider,
	}
}

// readFactorizeDupcodeThresholds is the named production adapter edge. The
// function is captured data-only and invoked by lifecycle initialization.
func readFactorizeDupcodeThresholds(path string) (int, int, error) {
	return protectedverifier.ReadBaselineThresholds(path)
}

// newFactorizeDupcodeAnalyzer is the named production adapter edge. Capturing
// this wrapper does not construct the analyzer; initialization invokes it.
func newFactorizeDupcodeAnalyzer() protectedverifier.DupcodeAnalyzer {
	return protectedverifier.NewAnalyzerFromAdapter()
}

func (d factorizeDupcodeDeps) validate() error {
	switch {
	case d.ReadThresholds == nil:
		return fmt.Errorf("factorize dupcode dependency ReadThresholds is nil")
	case d.NewAnalyzer == nil:
		return fmt.Errorf("factorize dupcode dependency NewAnalyzer is nil")
	case d.NewProvider == nil:
		return fmt.Errorf("factorize dupcode dependency NewProvider is nil")
	default:
		return nil
	}
}

// factorizeAnalyzerConstructionError is the stable fail-closed result when an
// analyzer factory returns no analyzer.
type factorizeAnalyzerConstructionError struct{}

func (factorizeAnalyzerConstructionError) Error() string {
	return "factorize dupcode analyzer construction returned nil"
}

// factorizeProviderConstructionError is the stable fail-closed result when a
// provider factory returns no provider.
type factorizeProviderConstructionError struct{}

func (factorizeProviderConstructionError) Error() string {
	return "factorize dupcode provider construction returned nil"
}

// factorizeDupcodeLifecycle owns one lazy shared analysis per factorize call.
type factorizeDupcodeLifecycle struct {
	once sync.Once

	root         string
	baselinePath string
	deps         factorizeDupcodeDeps

	dupcodeRun  func(string) []checks.Finding
	baselineRun func(string) []checks.Finding
	initErr     error
}

func newFactorizeDupcodeLifecycle(root string, deps factorizeDupcodeDeps) *factorizeDupcodeLifecycle {
	baselinePath := ".factory/dupcode-baseline.json"
	if root != "." && root != "" {
		baselinePath = filepath.Join(root, baselinePath)
	}
	return &factorizeDupcodeLifecycle{
		root:         root,
		baselinePath: baselinePath,
		deps:         deps,
	}
}

// initialize performs every protected setup step. sync.Once publishes all
// initialized fields before any concurrent caller continues.
func (l *factorizeDupcodeLifecycle) initialize() {
	minLines, minTokens, err := l.deps.ReadThresholds(l.baselinePath)
	if err != nil {
		l.initErr = fmt.Errorf("read dupcode thresholds: %w", err)
		return
	}
	if minLines <= 0 || minTokens <= 0 {
		l.initErr = fmt.Errorf("invalid dupcode thresholds: minLines=%d minTokens=%d", minLines, minTokens)
		return
	}

	analyzer := l.deps.NewAnalyzer()
	if analyzer == nil {
		l.initErr = factorizeAnalyzerConstructionError{}
		return
	}

	cfg := protectedverifier.DefaultConfig()
	cfg.Root = l.root
	cfg.MinLines = minLines
	cfg.MinTokens = minTokens
	input := protectedverifier.DupcodeInput{
		Root:      l.root,
		MinLines:  minLines,
		MinTokens: minTokens,
		Config:    cfg,
	}
	provider := l.deps.NewProvider(input, analyzer)
	if provider == nil {
		l.initErr = factorizeProviderConstructionError{}
		return
	}

	context := protectedverifier.NewDupcodeAnalysisContext(provider)
	factory := protectedverifier.NewDupcodeVerifierFactory(context)
	l.dupcodeRun = factory.SharedDupCodeVerifier()
	l.baselineRun = factory.SharedDupcodeBaselineVerifier()
}

func (l *factorizeDupcodeLifecycle) run(name string) []checks.Finding {
	if name != "dupcode" && name != "dupcode-baseline" {
		return []checks.Finding{{
			Path:     "dupcode",
			Kind:     "factorize_internal_invariant",
			Message:  fmt.Sprintf("unknown factorize dupcode lifecycle consumer: %q", name),
			Severity: checks.SeverityError,
		}}
	}

	// Dependency panics intentionally propagate. sync.Once therefore marks a
	// panicking initialization complete; panic containment remains a separate
	// policy decision and is not silently introduced at this boundary.
	l.once.Do(l.initialize)
	if l.initErr != nil {
		return []checks.Finding{{
			Path:     "dupcode",
			Kind:     "dupcode_error",
			Message:  fmt.Sprintf("duplicate code initialization failed: %v", l.initErr),
			Severity: checks.SeverityError,
		}}
	}
	switch name {
	case "dupcode":
		return l.dupcodeRun(l.root)
	case "dupcode-baseline":
		return l.baselineRun(l.root)
	default:
		panic("unreachable factorize dupcode lifecycle consumer")
	}
}

// FactorizeVerifiersWithDupcodeContext constructs one data-only factorize
// registry. Protected setup starts only when an admitted returned Run closure
// executes.
func FactorizeVerifiersWithDupcodeContext(root string) ([]registry.Verifier, error) {
	return factorizeVerifiersWithDeps(root, productionFactorizeDupcodeDeps())
}

func factorizeVerifiersWithDeps(root string, deps factorizeDupcodeDeps) ([]registry.Verifier, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	lifecycle := newFactorizeDupcodeLifecycle(root, deps)
	return replaceDupcodeVerifierRuns(
		AllVerifiers(),
		func(string) []checks.Finding { return lifecycle.run("dupcode") },
		func(string) []checks.Finding { return lifecycle.run("dupcode-baseline") },
	)
}

// factorizeVerifiersWithAnalyzer is retained as a test compatibility seam. It
// binds an injected analyzer lazily; nil selects production dependencies.
func factorizeVerifiersWithAnalyzer(
	root string,
	analyzer protectedverifier.DupcodeAnalyzer,
) ([]registry.Verifier, error) {
	deps := productionFactorizeDupcodeDeps()
	if analyzer != nil {
		deps.NewAnalyzer = func() protectedverifier.DupcodeAnalyzer { return analyzer }
	}
	return factorizeVerifiersWithDeps(root, deps)
}

// Deprecated: use factorizeVerifiersWithDeps for lifecycle integration tests.
func factorizeVerifiersWithDupcodeAnalyzer(
	root string,
	analyzer protectedverifier.DupcodeAnalyzer,
) ([]registry.Verifier, error) {
	return factorizeVerifiersWithAnalyzer(root, analyzer)
}

// replaceDupcodeVerifierRuns returns a copy with exactly two Run replacements.
// Missing or duplicate canonical identities fail atomically.
func replaceDupcodeVerifierRuns(
	verifiers []registry.Verifier,
	dupcodeRun func(string) []checks.Finding,
	baselineRun func(string) []checks.Finding,
) ([]registry.Verifier, error) {
	dupcodeIndex := -1
	baselineIndex := -1
	dupcodeCount := 0
	baselineCount := 0

	for i := range verifiers {
		switch verifiers[i].Name {
		case "dupcode":
			dupcodeCount++
			dupcodeIndex = i
		case "dupcode-baseline":
			baselineCount++
			baselineIndex = i
		}
	}
	if dupcodeCount != 1 || baselineCount != 1 {
		return nil, fmt.Errorf(
			"shared dupcode registry replacement requires exactly one entry each: dupcode=%d dupcode-baseline=%d",
			dupcodeCount,
			baselineCount,
		)
	}

	out := append([]registry.Verifier(nil), verifiers...)
	out[dupcodeIndex].Run = dupcodeRun
	out[baselineIndex].Run = baselineRun
	return out, nil
}
