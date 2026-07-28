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

// noopSampler is a sampler that always succeeds with zero values.
type noopSampler struct{}

func (n *noopSampler) Sample() (ResourceSnapshot, error) {
	return ResourceSnapshot{}, nil
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

// RunGate runs all verifiers and Go toolchain checks.
// All verifier execution is routed through the central dispatcher which performs
// authority validation before invoking the verifier.
func RunGate(root string) int {
	verifiers := AllVerifiers()

	if err := ValidateVerifiers(verifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	for _, v := range verifiers {
		if err := v.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
			return 1
		}
	}

	sort.Slice(verifiers, func(i, j int) bool {
		return verifiers[i].Name < verifiers[j].Name
	})

	dispatcher, err := verifierdispatch.NewDispatcher(verifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	ctx := context.Background()
	observer := &verifierdispatch.DefaultContextObserver{}
	failed := false

	for _, v := range verifiers {
		var findings []checks.Finding

		request := verifierdispatch.Request{
			VerifierID: v.Name,
			Operation:  verifierauthority.OperationVerify,
			Root:       root,
		}

		runnerFactory := func() func(root string) []checks.Finding {
			return v.Run
		}

		result := dispatcher.Dispatch(ctx, request, observer, runnerFactory)

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

// RunFactorize runs all Factory policy verifiers without toolchain checks.
func RunFactorize(root string) int {
	allVerifiers := AllVerifiers()

	if err := ValidateVerifiers(allVerifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

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

	if err := ValidateVerifiers(fastVerifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	sort.Slice(fastVerifiers, func(i, j int) bool {
		return fastVerifiers[i].Name < fastVerifiers[j].Name
	})

	for _, v := range dupcodeVerifiers {
		fmt.Printf("  %s: SKIP: expensive verifier lane; run make gate-dupcode\n", v.Name)
	}

	dispatcher, err := verifierdispatch.NewDispatcher(fastVerifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	ctx := context.Background()
	observer := &verifierdispatch.DefaultContextObserver{}
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

		result := dispatcher.Dispatch(ctx, request, observer, runnerFactory)

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

	runToolchainChecksFast(root, &failed)

	if failed {
		fmt.Printf("\n*** GATE FAILED ***\n")
		return 1
	}

	fmt.Printf("\n*** GATE PASSED ***\n")
	return 0
}

// RunGateFast runs the gate in fast mode. Executes only fast-lane verifiers.
func RunGateFast(root string) int {
	allVerifiers := AllVerifiers()
	fastVerifiers, dupcodeVerifiers, err := PartitionVerifiers(allVerifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	if err := ValidateVerifiers(fastVerifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	for _, v := range fastVerifiers {
		if err := v.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
			return 1
		}
	}

	sort.Slice(fastVerifiers, func(i, j int) bool {
		return fastVerifiers[i].Name < fastVerifiers[j].Name
	})

	for _, v := range dupcodeVerifiers {
		fmt.Printf("  %s: SKIP: expensive verifier lane; run make gate-dupcode\n", v.Name)
	}

	dispatcher, err := verifierdispatch.NewDispatcher(fastVerifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	ctx := context.Background()
	observer := &verifierdispatch.DefaultContextObserver{}
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

		result := dispatcher.Dispatch(ctx, request, observer, runnerFactory)

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

	if err := ValidateVerifiers(dupcodeVerifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	for _, v := range dupcodeVerifiers {
		if err := v.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
			return 1
		}
	}

	sort.Slice(dupcodeVerifiers, func(i, j int) bool {
		return dupcodeVerifiers[i].Name < dupcodeVerifiers[j].Name
	})

	dispatcher, err := verifierdispatch.NewDispatcher(dupcodeVerifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	ctx := context.Background()
	observer := &verifierdispatch.DefaultContextObserver{}
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

		result := dispatcher.Dispatch(ctx, request, observer, runnerFactory)

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

	RunDupcodeToolchain(root, &failed)

	if failed {
		fmt.Printf("\n*** GATE FAILED ***\n")
		return 1
	}

	fmt.Printf("\n*** GATE PASSED ***\n")
	return 0
}
