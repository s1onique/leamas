// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// clock abstracts the time source used to measure check and total
// durations. Production code uses systemClock; tests inject a fake.
type clock interface {
	Now() time.Time
}

// systemClock is the production clock; it delegates to time.Now.
type systemClock struct{}

// Now returns the current wall-clock time.
func (systemClock) Now() time.Time { return time.Now() }

// runCheck executes a single verifier check, measures its wall-clock
// duration, and writes one status line to out.
//
// Resource usage is sampled before and after the verifier when a sampler is provided.
// Both samples must succeed or the check fails.
func runCheck(
	out io.Writer,
	clk clock,
	verifier registry.Verifier,
	metrics *MetricsCollectionV3,
	ordinal int,
	root string,
	sampler ResourceSampler,
) ([]checks.Finding, error) {
	// Sample before verifier execution if sampler provided
	var before ResourceSnapshot
	if sampler != nil {
		var err error
		before, err = sampler.Sample()
		if err != nil {
			return nil, fmt.Errorf("resource sample before %s: %w", verifier.Name, err)
		}
	}

	started := clk.Now()
	findings := verifier.Run(root)
	elapsed := clk.Now().Sub(started)

	status := "OK"
	if len(findings) > 0 {
		status = "FAILED"
	}

	fmt.Fprintf(out, "  %s: %s: %.2fs\n", verifier.Name, status, elapsed.Seconds())

	// Sample after verifier execution if sampler provided
	if sampler != nil {
		after, err := sampler.Sample()
		if err != nil {
			return findings, fmt.Errorf("resource sample after %s: %w", verifier.Name, err)
		}

		if metrics != nil {
			env := os.Environ()
			if err := metrics.AddCheckWithResources(
				verifier,
				ordinal,
				findings,
				elapsed,
				after.UserCPU-before.UserCPU,
				after.SystemCPU-before.SystemCPU,
				after.ProcessMaxRSSKB,
				root,
				env,
			); err != nil {
				return findings, fmt.Errorf("metrics collection for %s: %w", verifier.Name, err)
			}
		}
	}

	return findings, nil
}

// printFailureFindings renders the verbose failure block.
func printFailureFindings(out io.Writer, name string, findings []checks.Finding) {
	fmt.Fprintf(out, "\n--- %s FAILED ---\n", name)
	for _, f := range findings {
		fmt.Fprintf(out, "  %s: %s: %s\n", f.Path, f.Kind, f.Message)
	}
}

// runFactorize is the testable form of RunFactorize.
func runFactorize(
	out io.Writer,
	clk clock,
	root string,
	verifiers []registry.Verifier,
	metrics *MetricsCollectionV3,
	sampler ResourceSampler,
) int {
	sorted := make([]registry.Verifier, len(verifiers))
	copy(sorted, verifiers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	startedAt := clk.Now()
	failed := false

	ordinal := 1
	for _, v := range sorted {
		findings, err := runCheck(out, clk, v, metrics, ordinal, root, sampler)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			failed = true
		}
		if len(findings) > 0 {
			failed = true
			printFailureFindings(out, v.Name, findings)
		}
		ordinal++
	}

	elapsed := clk.Now().Sub(startedAt)

	_ = elapsed // retained for future metrics enrichment

	if failed {
		fmt.Fprintf(out, "\n*** FACTORIZE FAILED: %.2fs ***\n", elapsed.Seconds())
		return 1
	}

	fmt.Fprintf(out, "\n*** FACTORIZE PASSED: %.2fs ***\n", elapsed.Seconds())
	return 0
}

// exitCode returns the appropriate exit code for the factorize run.
func exitCode(failed bool) int {
	if failed {
		return 1
	}
	return 0
}

// processExecutionRecords processes pre-computed execution records and prints results.
// This is called after binding.Execute() has already run all verifiers with truthful timing.
func processExecutionRecords(
	out io.Writer,
	profile *verifierdispatch.AuthorizedProfile,
	records []verifierdispatch.ExecutionRecord,
	metrics *MetricsCollectionV3,
	totalElapsed time.Duration,
) int {
	failed := false

	ordinal := 1
	for _, record := range records {
		status := "OK"
		if len(record.Findings) > 0 {
			status = "FAILED"
		}
		fmt.Fprintf(out, "  %s: %s: %.2fs\n", record.Metadata.Name, status, record.Duration.Seconds())

		if len(record.Findings) > 0 {
			failed = true
			printFailureFindings(out, record.Metadata.Name, record.Findings)
		}

		// Metrics collection using real timing from Execute()
		if metrics != nil {
			env := os.Environ()
			// Build a registry.Verifier from metadata for metrics compatibility
			verifier := registry.Verifier{
				Name:      record.Metadata.Name,
				Lane:      record.Metadata.Lane,
				Authority: record.Metadata.Authority,
				Cache:     record.Metadata.Cache,
				Execution: registry.ExecutionDefinition{
					Kind:             record.Metadata.Kind,
					ImplementationID: record.Metadata.ImplID,
					EnvVars:          record.Metadata.EnvVars,
				},
			}
			if err := metrics.AddCheckWithResources(
				verifier,
				ordinal,
				record.Findings,
				record.Duration,
				0, // CPU delta not available from binding.Execute()
				0, // System CPU delta not available
				0, // RSS not available
				profile.RepositoryRoot(),
				env,
			); err != nil {
				fmt.Fprintf(os.Stderr, "error: metrics collection for %s: %v\n", record.Metadata.Name, err)
				failed = true
			}
		}
		ordinal++
	}

	if failed {
		fmt.Fprintf(out, "\n*** FACTORIZE FAILED: %.2fs ***\n", totalElapsed.Seconds())
		return 1
	}

	fmt.Fprintf(out, "\n*** FACTORIZE PASSED: %.2fs ***\n", totalElapsed.Seconds())
	return 0
}
