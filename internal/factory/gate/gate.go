// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// RunGate runs all verifiers and Go toolchain checks.
// All verifier execution is routed through the central dispatcher which performs
// authority validation before invoking the verifier.
func RunGate(root string) int {
	verifiers := AllVerifiers()

	// Fail closed if registry has invalid metadata
	if err := ValidateVerifiers(verifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	// Validate each verifier's authority metadata
	for _, v := range verifiers {
		if err := v.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
			return 1
		}
	}

	sort.Slice(verifiers, func(i, j int) bool {
		return verifiers[i].Name < verifiers[j].Name
	})

	// Create dispatcher for all verifiers
	dispatcher := verifierdispatch.NewDispatcher(verifiers)
	ctx := context.Background()

	failed := false

	for _, v := range verifiers {
		var findings []checks.Finding

		// Route through dispatcher - authority validation happens inside Dispatch
		request := verifierdispatch.Request{
			VerifierID: v.Name,
			Operation:  verifierauthority.OperationVerify,
			Root:       root,
		}

		// RunnerFactory: invoked only after authority validation passes
		runnerFactory := func() func(root string) []checks.Finding {
			return v.Run
		}

		result := dispatcher.Dispatch(ctx, request, runnerFactory)

		if len(result.Findings) > 0 {
			findings = result.Findings
		}

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

// FastVerifiers returns verifiers that run in the fast lane.
func FastVerifiers() []registry.Verifier {
	var result []registry.Verifier
	for _, v := range AllVerifiers() {
		if v.Lane == registry.VerifierLaneFast {
			result = append(result, v)
		}
	}
	return result
}

// DupcodeVerifiers returns verifiers that run in the dupcode lane.
func DupcodeVerifiers() []registry.Verifier {
	var result []registry.Verifier
	for _, v := range AllVerifiers() {
		if v.Lane == registry.VerifierLaneDupcode {
			result = append(result, v)
		}
	}
	return result
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
//
// Factorize uses a shared dupcode analysis context to ensure that both
// "dupcode" and "dupcode-baseline" verifiers perform only one scan of the
// repository during a single factorize invocation.
//
// All verifier execution is routed through the central dispatcher which performs
// authority validation before invoking the verifier.
func RunFactorize(root string) int {
	// Validate registry metadata upfront
	allVerifiers := AllVerifiers()

	if err := ValidateVerifiers(allVerifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	// Validate each verifier's authority metadata upfront
	for _, v := range allVerifiers {
		if err := v.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
			return 1
		}
	}

	fastVerifiers, dupcodeVerifiers, err := PartitionVerifiers(allVerifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	// Validate registry metadata
	if err := ValidateVerifiers(fastVerifiers); err != nil {
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

	// Create dispatcher for fast verifiers
	dispatcher := verifierdispatch.NewDispatcher(fastVerifiers)
	ctx := context.Background()

	failed := false

	for _, v := range fastVerifiers {
		request := verifierdispatch.Request{
			VerifierID: v.Name,
			Operation:  verifierauthority.OperationVerify,
			Root:       root,
		}

		runnerFactory := func() func(root string) []checks.Finding {
			return v.Run
		}

		result := dispatcher.Dispatch(ctx, request, runnerFactory)

		var findings []checks.Finding
		if len(result.Findings) > 0 {
			findings = result.Findings
		}

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

	// Validate registry metadata
	if err := ValidateVerifiers(fastVerifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	// Validate each verifier's authority metadata upfront
	for _, v := range fastVerifiers {
		if err := v.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
			return 1
		}
	}

	sort.Slice(fastVerifiers, func(i, j int) bool {
		return fastVerifiers[i].Name < fastVerifiers[j].Name
	})

	// Report skipped verifiers
	for _, v := range dupcodeVerifiers {
		fmt.Printf("  %s: SKIP: expensive verifier lane; run make gate-dupcode\n", v.Name)
	}

	// Create dispatcher for fast verifiers
	dispatcher := verifierdispatch.NewDispatcher(fastVerifiers)
	ctx := context.Background()

	failed := false

	for _, v := range fastVerifiers {
		request := verifierdispatch.Request{
			VerifierID: v.Name,
			Operation:  verifierauthority.OperationVerify,
			Root:       root,
		}

		runnerFactory := func() func(root string) []checks.Finding {
			return v.Run
		}

		result := dispatcher.Dispatch(ctx, request, runnerFactory)

		var findings []checks.Finding
		if len(result.Findings) > 0 {
			findings = result.Findings
		}

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
// Dupcode is a CI-only verifier lane. All verifier execution is routed through
// the central dispatcher which performs authority validation before invoking any verifier.
func RunGateDupcode(root string) int {
	allVerifiers := AllVerifiers()
	_, dupcodeVerifiers, err := PartitionVerifiers(allVerifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	// Validate registry metadata
	if err := ValidateVerifiers(dupcodeVerifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	// Validate each verifier's authority metadata upfront
	for _, v := range dupcodeVerifiers {
		if err := v.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
			return 1
		}
	}

	sort.Slice(dupcodeVerifiers, func(i, j int) bool {
		return dupcodeVerifiers[i].Name < dupcodeVerifiers[j].Name
	})

	// Create dispatcher for dupcode verifiers
	dispatcher := verifierdispatch.NewDispatcher(dupcodeVerifiers)
	ctx := context.Background()

	failed := false

	for _, v := range dupcodeVerifiers {
		request := verifierdispatch.Request{
			VerifierID: v.Name,
			Operation:  verifierauthority.OperationVerify,
			Root:       root,
		}

		runnerFactory := func() func(root string) []checks.Finding {
			return v.Run
		}

		result := dispatcher.Dispatch(ctx, request, runnerFactory)

		var findings []checks.Finding
		if len(result.Findings) > 0 {
			findings = result.Findings
		}

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
