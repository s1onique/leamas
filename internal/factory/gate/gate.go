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

// VerifierLane classifies a verifier into a specific execution lane.
type VerifierLane string

const (
	VerifierLaneFast    VerifierLane = "fast"
	VerifierLaneDupcode VerifierLane = "dupcode"
)

// ExecutionDefinition captures the authoritative execution metadata for a verifier.
type ExecutionDefinition struct {
	Kind             ExecutionKind
	ImplementationID string
	EnvVars          []string
}

// CacheSemantics captures the authoritative cache behavior for a verifier.
type CacheSemantics struct {
	GoBuildCache      CacheRelevance      `json:"go_build_cache"`
	GoTestResultCache TestResultCacheMode `json:"go_test_result_cache"`
}

// Verifier represents a Factory verifier with its authoritative metadata.
type Verifier struct {
	Name      string
	Run       func(root string) []checks.Finding
	Lane      VerifierLane
	Execution ExecutionDefinition
	Cache     CacheSemantics
}

// RunGate runs all verifiers and Go toolchain checks.
func RunGate(root string) int {
	verifiers := AllVerifiers()

	// Fail closed if registry has invalid metadata
	if err := ValidateVerifiers(verifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	sort.Slice(verifiers, func(i, j int) bool {
		return verifiers[i].Name < verifiers[j].Name
	})

	failed := false

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
	verifiers := AllVerifiers()

	// Fail closed if registry has invalid metadata
	if err := ValidateVerifiers(verifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	var mc *MetricsCollection
	if shouldCollectMetrics() {
		mc = &MetricsCollection{Path: metricsFilePath()}
	}
	return runFactorize(os.Stdout, systemClock{}, root, verifiers, mc)
}

// RunGateFast runs the gate in fast mode. It executes only fast-lane verifiers
// and explicitly skips dupcode-lane verifiers with honest SKIP messages.
func RunGateFast(root string) int {
	allVerifiers := AllVerifiers()
	fastVerifiers, dupcodeVerifiers, err := PartitionVerifiers(allVerifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	sort.Slice(fastVerifiers, func(i, j int) bool {
		return fastVerifiers[i].Name < fastVerifiers[j].Name
	})

	// Report skipped verifiers
	for _, v := range dupcodeVerifiers {
		fmt.Printf("  %s: SKIP: expensive verifier lane; run make gate-dupcode\n", v.Name)
	}

	failed := false

	for _, v := range fastVerifiers {
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

	// Run toolchain checks in fast mode (excludes dupcode package tests)
	runToolchainChecksFast(root, &failed)

	if failed {
		fmt.Printf("\n*** GATE FAILED ***\n")
		return 1
	}

	fmt.Printf("\n*** GATE PASSED ***\n")
	return 0
}

// RunGateDupcode runs the dupcode lane with exactly the duplicate-code verifiers.
func RunGateDupcode(root string) int {
	allVerifiers := AllVerifiers()
	_, dupcodeVerifiers, err := PartitionVerifiers(allVerifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	sort.Slice(dupcodeVerifiers, func(i, j int) bool {
		return dupcodeVerifiers[i].Name < dupcodeVerifiers[j].Name
	})

	failed := false

	for _, v := range dupcodeVerifiers {
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

	// Run dupcode package tests
	RunDupcodeToolchain(root, &failed)

	if failed {
		fmt.Printf("\n*** GATE FAILED ***\n")
		return 1
	}

	fmt.Printf("\n*** GATE PASSED ***\n")
	return 0
}

// FastVerifiers returns verifiers that run in the fast lane.
func FastVerifiers() []Verifier {
	var result []Verifier
	for _, v := range AllVerifiers() {
		if v.Lane == VerifierLaneFast {
			result = append(result, v)
		}
	}
	return result
}

// DupcodeVerifiers returns verifiers that run in the dupcode lane.
func DupcodeVerifiers() []Verifier {
	var result []Verifier
	for _, v := range AllVerifiers() {
		if v.Lane == VerifierLaneDupcode {
			result = append(result, v)
		}
	}
	return result
}
