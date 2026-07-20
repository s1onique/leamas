// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"fmt"
	"os"
	"sort"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// ExecutionKind classifies how a verifier executes.
type ExecutionKind string

const (
	ExecutionInProcess ExecutionKind = "in-process"
	ExecutionChild     ExecutionKind = "child-process"
)

// CacheRelevance classifies whether Go build cache affects the verifier.
type CacheRelevance string

const (
	CacheRelevant      CacheRelevance = "relevant"
	CacheNotRelevant   CacheRelevance = "not-relevant"
	CacheNotApplicable CacheRelevance = "not-applicable"
)

// TestResultCacheMode classifies whether test result cache applies.
type TestResultCacheMode string

const (
	CacheModeEnabled  TestResultCacheMode = "enabled"
	CacheModeDisabled TestResultCacheMode = "disabled"
	CacheModeNA       TestResultCacheMode = "not-applicable"
)

// ExecutionDefinition captures the authoritative execution metadata for a verifier.
type ExecutionDefinition struct {
	Kind        ExecutionKind
	LogicalArgv []string
	EnvVars     []string
}

// CacheSemantics captures the authoritative cache behavior for a verifier.
type CacheSemantics struct {
	GoBuildCache      CacheRelevance
	GoTestResultCache TestResultCacheMode
}

// Verifier represents a Factory verifier with its authoritative metadata.
type Verifier struct {
	Name      string
	Run       func(root string) []checks.Finding
	Execution ExecutionDefinition
	Cache     CacheSemantics
}

// RunGate runs all verifiers and Go toolchain checks.
func RunGate(root string) int {
	failed := false
	verifiers := AllVerifiers()
	sort.Slice(verifiers, func(i, j int) bool {
		return verifiers[i].Name < verifiers[j].Name
	})

	for _, v := range verifiers {
		findings := v.Run(root)
		if len(findings) > 0 {
			failed = true
			fmt.Printf("\n--- %s FAILED ---\n", v.Name)
			for _, f := range findings {
				fmt.Printf("  %s: %s: %s\n", f.Path, f.Kind, f.Message)
			}
		} else {
			fmt.Printf("  %s: OK\n", v.Name)
		}
	}

	runToolchainChecks(root, &failed)

	if failed {
		fmt.Printf("\n*** GATE FAILED ***\n")
		return 1
	}

	fmt.Printf("\n*** GATE PASSED ***\n")
	return 0
}

// RunFactorize runs all Factory policy verifiers without toolchain checks.
func RunFactorize(root string) int {
	var mc *MetricsCollection
	if shouldCollectMetrics() {
		mc = &MetricsCollection{Path: metricsFilePath()}
	}
	return runFactorize(os.Stdout, systemClock{}, root, AllVerifiers(), mc)
}
