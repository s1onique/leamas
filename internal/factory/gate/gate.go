// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// noopSampler is a sampler that always succeeds with zero values.
type noopSampler struct{}

func (n *noopSampler) Sample() (verifierdispatch.ResourceSnapshot, error) {
	return verifierdispatch.ResourceSnapshot{}, nil
}

// platformSamplerAdapter wraps gate's PlatformSampler to implement verifierdispatch.ResourceSampler.
type platformSamplerAdapter struct {
	inner ResourceSampler
}

func (a *platformSamplerAdapter) Sample() (verifierdispatch.ResourceSnapshot, error) {
	snap, err := a.inner.Sample()
	if err != nil {
		return verifierdispatch.ResourceSnapshot{}, err
	}
	return verifierdispatch.ResourceSnapshot{
		UserCPU:   snap.UserCPU,
		SystemCPU: snap.SystemCPU,
		MaxRSSKB:  snap.ProcessMaxRSSKB,
	}, nil
}

// FastVerifiers returns verifiers that run in the fast lane.
func FastVerifiers() []registry.Verifier {
	var result []registry.Verifier
	for _, v := range AllVerifiers() {
		if v.Lane == registry.VerifierLaneFast && v.Scope == registry.InvocationGate {
			result = append(result, v)
		}
	}
	return result
}

// DupcodeVerifiers returns verifiers that run in the dupcode lane.
// Command-only definitions are excluded so they cannot leak into gate /
// factorize selection.
func DupcodeVerifiers() []registry.Verifier {
	var result []registry.Verifier
	for _, v := range AllVerifiers() {
		if v.Lane == registry.VerifierLaneDupcode && v.Scope == registry.InvocationGate {
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
//
// Command-only definitions (e.g. dupcode-update-baseline) are excluded from
// RunGate discovery; they are reachable only via typed command dispatch.
func RunGate(root string) int {
	verifiers := GateVerifiers()

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
// Authorization and execution are bound: the factory creates the shared context
// ONLY after authorization passes, and all runners execute exactly once.
func RunFactorize(root string) int {
	// Phase 1: Build the base verifier registry for authorization.
	// Command-only definitions are excluded from factorize selection; they
	// are reachable only via typed command dispatch.
	verifiers := GateVerifiers()

	// Sort by name for alphabetical order (preserving established factorize contract)
	sort.Slice(verifiers, func(i, j int) bool {
		return verifiers[i].Name < verifiers[j].Name
	})

	// Fail closed if registry has invalid metadata
	if err := ValidateVerifiers(verifiers); err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	// Create dispatcher
	dispatcher, err := verifierdispatch.NewDispatcher(verifiers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory verifier registry: %v\n", err)
		return 1
	}

	ctx := context.Background()
	observer := &verifierdispatch.DefaultContextObserver{}

	// Build ProfileRequests for exactly the authorized verifiers (alphabetical order)
	requests := make([]verifierdispatch.ProfileRequest, 0, len(verifiers))
	for _, v := range verifiers {
		requests = append(requests, verifierdispatch.ProfileRequest{
			VerifierID: v.Name,
			Operation:  verifierauthority.OperationVerify,
		})
	}

	// Phase 2: Authorize AND bind (factory creates shared context AFTER authorization passes)
	// The factory is ONLY invoked after authorization succeeds.
	binding, err := dispatcher.AuthorizeAndBindProfile(ctx, root, requests, observer,
		func(authorized []verifierdispatch.VerifierMetadata) ([]verifierdispatch.FactoryRunner, error) {
			// Create the shared dupcode context AFTER authorization
			// This is the expensive operation that should only happen when authorized
			factorizeVerifiers, err := FactorizeVerifiersWithDupcodeContext(root)
			if err != nil {
				return nil, err
			}

			// Build a map of Run functions from the factorize verifiers
			runMap := make(map[string]func(string) []checks.Finding)
			for _, v := range factorizeVerifiers {
				runMap[v.Name] = v.Run
			}

			// Build factory runners: ID + Run only (metadata comes from dispatcher)
			factoryRunners := make([]verifierdispatch.FactoryRunner, 0, len(authorized))
			for _, v := range authorized {
				run, ok := runMap[v.Name]
				if !ok {
					return nil, fmt.Errorf("factory: no run function for authorized verifier %s", v.Name)
				}
				factoryRunners = append(factoryRunners, verifierdispatch.FactoryRunner{
					VerifierID: v.Name,
					Run:        run,
				})
			}
			return factoryRunners, nil
		})

	if err != nil {
		fmt.Fprintf(os.Stderr, "factory authorization: %v\n", err)
		return 1
	}

	// Print denials if any
	profile := binding.Profile()
	if len(profile.Denials()) > 0 {
		printAuthorizationDenials(profile.Denials())
		return 1
	}

	// Factory contract violation - no runners bound
	if len(binding.Runners()) == 0 {
		fmt.Fprintf(os.Stderr, "factory: no runners bound for authorized inventory\n")
		return 1
	}

	var mc *MetricsCollectionV3
	var sampler verifierdispatch.ResourceSampler

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
		for _, meta := range binding.Runners() {
			mc.ExpectedVerifierIDs = append(mc.ExpectedVerifierIDs, meta.Name)
		}

		sampler = &platformSamplerAdapter{inner: NewPlatformSampler()}
	} else {
		// Use a no-op sampler when metrics are disabled
		sampler = &noopSampler{}
	}

	// Track total factorize duration including verifier execution
	totalStart := time.Now()

	// Phase 3: Execute bound runners exactly once with real timing and resource sampling
	records, err := binding.Execute(sampler)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory execution: %v\n", err)
		return 1
	}

	totalElapsed := time.Since(totalStart)

	// Process execution records and print results with real timing
	profile = binding.Profile()
	exitCode := processExecutionRecords(os.Stdout, profile, records, mc, totalElapsed)

	// Fail-closed: metrics finalization errors cause factorize to fail
	if mc != nil {
		if err := mc.Finalize(exitCode != 0); err != nil {
			fmt.Fprintf(os.Stderr, "factory metrics finalization: %v\n", err)
			return 1
		}
	}

	return exitCode
}
