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
// When LEAMAS_FACTORIZE_METRICS_FILE is set, metrics are collected and published.
// Metrics collection failures cause factorize to exit non-zero (fail-closed).
//
// Factorize uses a shared dupcode analysis context to ensure that both
// "dupcode" and "dupcode-baseline" verifiers perform only one scan of the
// repository during a single factorize invocation.
//
// All verifier execution is routed through AuthorizeAndRunProfile which binds
// authorization to execution: the factory is ONLY invoked after authorization passes,
// and the returned runners must match exactly the authorized inventory.
func RunFactorize(root string) int {
	// Phase 1: Build verifier registry with shared dupcode context
	// This must happen BEFORE creating the dispatcher so the Run functions
	// are bound to the shared context (not the independent ones from AllVerifiers).
	verifiers, err := FactorizeVerifiersWithDupcodeContext(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	// Fail closed if registry has invalid metadata
	if err := ValidateVerifiers(verifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	// Build map for quick lookup of Run functions
	runMap := make(map[string]func(string) []checks.Finding)
	for _, v := range verifiers {
		runMap[v.Name] = v.Run
	}

	// Phase 2: Create dispatcher and authorize the exact verifier inventory
	dispatcher, err := verifierdispatch.NewDispatcher(verifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	ctx := context.Background()
	observer := &verifierdispatch.DefaultContextObserver{}

	// Build ProfileRequests for exactly the authorized verifiers
	requests := make([]verifierdispatch.ProfileRequest, 0, len(verifiers))
	authorizedIDs := make([]string, 0, len(verifiers))
	for _, v := range verifiers {
		requests = append(requests, verifierdispatch.ProfileRequest{
			VerifierID: v.Name,
			Operation:  verifierauthority.OperationVerify,
		})
		authorizedIDs = append(authorizedIDs, v.Name)
	}

	// The factory is ONLY invoked after authorization passes.
	// It creates bound runners for the exact authorized inventory.
	factory := func(authorized []*registry.Verifier) ([]verifierdispatch.BoundProfileRunner, error) {
		// Build bound runners in canonical authorized-request order
		runners := make([]verifierdispatch.BoundProfileRunner, 0, len(authorized))
		for _, v := range authorized {
			run, ok := runMap[v.Name]
			if !ok {
				// Should not happen if registry is consistent
				return nil, fmt.Errorf("factory: no run function for authorized verifier %s", v.Name)
			}
			runners = append(runners, verifierdispatch.BoundProfileRunner{
				VerifierID: v.Name,
				Run:        run,
			})
		}
		return runners, nil
	}

	result, err := dispatcher.AuthorizeAndRunProfile(ctx, root, requests, observer, factory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory authorization: %v\n", err)
		return 1
	}

	// Print denials if any
	if len(result.Profile.Denials()) > 0 {
		printAuthorizationDenials(result.Profile.Denials())
		return 1
	}

	// Factory contract violation - no runners executed
	if !result.AllRun {
		fmt.Fprintf(os.Stderr, "factory: runner set did not match authorized inventory\n")
		return 1
	}

	var mc *MetricsCollectionV3
	var sampler ResourceSampler

	// Metrics collection is enabled when the destination path is set
	if shouldCollectMetrics() {
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

		// Collect subject identity from the repository (exclude metrics destination)
		identity, err := CollectSubjectIdentity(root, metricsFilePath())
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
		for _, id := range authorizedIDs {
			mc.ExpectedVerifierIDs = append(mc.ExpectedVerifierIDs, id)
		}

		sampler = NewPlatformSampler()
	} else {
		// Use a no-op sampler when metrics are disabled
		sampler = &noopSampler{}
	}

	// Extract runners from result for execution
	runners := make([]verifierdispatch.BoundProfileRunner, 0, len(result.Findings))
	for id, _ := range result.Findings {
		if run, ok := runMap[id]; ok {
			runners = append(runners, verifierdispatch.BoundProfileRunner{
				VerifierID: id,
				Run:        run,
			})
		}
	}

	exitCode := runFactorizeWithRunners(os.Stdout, systemClock{}, result.Profile, runners, mc, sampler)

	// Fail-closed: metrics finalization errors cause factorize to fail
	if mc != nil {
		if err := mc.Finalize(exitCode != 0); err != nil {
			fmt.Fprintf(os.Stderr, "factory metrics finalization: %v\n", err)
			return 1
		}
	}

	return exitCode
}
