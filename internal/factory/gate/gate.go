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

func metricsFilePath() string {
	return os.Getenv("LEAMAS_FACTORIZE_METRICS_FILE")
}

func metricsScenario() string {
	return os.Getenv("LEAMAS_FACTORIZE_SCENARIO")
}

func metricsSequence() string {
	return os.Getenv("LEAMAS_FACTORIZE_SEQUENCE")
}

func shouldCollectMetrics() bool {
	return metricsFilePath() != ""
}

// RunFactorize runs all Factory policy verifiers without toolchain checks.
// When LEAMAS_FACTORIZE_METRICS_FILE is set, metrics are collected and published.
// Metrics collection failures cause factorize to exit non-zero (fail-closed).
func RunFactorize(root string) int {
	verifiers := AllVerifiers()

	// Fail closed if registry has invalid metadata
	if err := ValidateVerifiers(verifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	var mc *MetricsCollectionV3
	var sampler ResourceSampler

	// Metrics collection is enabled when the destination path is set
	if shouldCollectMetrics() {
		var err error

		// Validate scenario is provided
		scenario := metricsScenario()
		if scenario == "" {
			fmt.Fprintf(os.Stderr, "factory metrics: LEAMAS_FACTORIZE_SCENARIO required when LEAMAS_FACTORIZE_METRICS_FILE is set\n")
			return 1
		}

		// Validate sequence is provided
		sequence := metricsSequence()
		if sequence == "" {
			fmt.Fprintf(os.Stderr, "factory metrics: LEAMAS_FACTORIZE_SEQUENCE required when LEAMAS_FACTORIZE_METRICS_FILE is set\n")
			return 1
		}

		mc, err = NewMetricsCollectionV3(metricsFilePath(), scenario, sequence)
		if err != nil {
			fmt.Fprintf(os.Stderr, "factory metrics configuration: %v\n", err)
			return 1
		}

		// Collect subject identity from the repository
		identity, err := CollectSubjectIdentity(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "factory metrics: subject identity collection: %v\n", err)
			return 1
		}

		if err := ValidateSubjectIdentity(identity); err != nil {
			fmt.Fprintf(os.Stderr, "factory metrics: invalid subject identity: %v\n", err)
			return 1
		}

		mc.SetSubjectIdentity(
			identity.HeadOID,
			identity.TreeOID,
			identity.WorktreeState,
			identity.SubjectInputDigest,
		)

		// Bind expected verifier inventory for reconciliation
		for _, v := range verifiers {
			mc.ExpectedVerifierIDs = append(mc.ExpectedVerifierIDs, v.Name)
		}

		sampler = NewPlatformSampler()
	} else {
		// Use a no-op sampler when metrics are disabled
		sampler = &noopSampler{}
	}

	result := runFactorize(os.Stdout, systemClock{}, root, verifiers, mc, sampler)

	// Fail-closed: metrics finalization errors cause factorize to fail
	if mc != nil {
		if err := mc.Finalize(result != 0); err != nil {
			fmt.Fprintf(os.Stderr, "factory metrics finalization: %v\n", err)
			return 1
		}
	}

	return result
}

// noopSampler is a sampler that always succeeds with zero values.
type noopSampler struct{}

func (n *noopSampler) Sample() (ResourceSnapshot, error) {
	return ResourceSnapshot{}, nil
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
