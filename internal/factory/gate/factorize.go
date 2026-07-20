// Package gate provides the factorize runner with wall-clock timings.
//
// Timings are deliberately excluded from JSON, digest, fingerprint,
// and snapshot contracts; they live only in the interactive text
// progress output of `leamas factory factorize`. The clock is injected
// so tests can assert exact elapsed-time formatting without sleeping.
//
// When LEAMAS_FACTORIZE_METRICS_FILE is set, machine-readable per-verifier
// metrics are written atomically to the specified path after the run.
package gate

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/s1onique/leamas/internal/factory/checks"
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
// duration, and writes one status line to out:
//
//	"name: OK: 0.14s"
//	"name: FAILED: 0.91s"
//
// The duration is always reported, even on failure, so slow failing
// checks remain visible. The function returns the findings produced
// by the check so the caller can render detail lines when needed.
//
// If metrics is non-nil, resource usage is also collected for the check.
func runCheck(
	out io.Writer,
	clk clock,
	name string,
	check func() []checks.Finding,
	metrics *MetricsCollection,
	ordinal int,
	root string,
) []checks.Finding {
	started := clk.Now()
	findings := check()
	elapsed := clk.Now().Sub(started)

	status := "OK"
	if len(findings) > 0 {
		status = "FAILED"
	}

	fmt.Fprintf(out, "  %s: %s: %.2fs\n", name, status, elapsed.Seconds())

	// Collect metrics if enabled
	if metrics != nil {
		rusage := collectRusage()
		// Determine cache observation based on verifier
		cacheObs := classifyCacheObservation(name, findings)
		// Get executable path and environment for fingerprinting
		execPath, _ := os.Executable()
		env := os.Environ()
		metrics.AddCheck(name, ordinal, findings, elapsed, rusage, root, cacheObs, []string{name}, env, execPath)
	}

	return findings
}

// classifyCacheObservation determines the cache classification for a verifier.
func classifyCacheObservation(name string, findings []checks.Finding) string {
	// Verifiers that run go test with -count=1 (test-result caching disabled)
	goTestVerifiers := map[string]bool{
		"dupcode":          true,
		"dupcode-baseline": true,
	}
	if goTestVerifiers[name] {
		return "go_test_result_cache=disabled;go_build_cache=relevant"
	}
	// Verifiers that benefit from Go build caching (but don't use -count=1)
	// These typically invoke go build, go vet, or similar
	goBuildRelevant := map[string]bool{
		"static-binary":   true,
		"executable-contract-first": true,
		"exec-gate":       true,
		"go-coverage":     true,
	}
	if goBuildRelevant[name] {
		return "go_test_result_cache=not-applicable;go_build_cache=relevant"
	}
	return "go_test_result_cache=not-applicable;go_build_cache=not-applicable"
}

// printFailureFindings renders the verbose failure block (one
// "path: kind: message" line per finding) for a failed check. Kept
// separate from runCheck so runCheck owns only the single status line.
func printFailureFindings(out io.Writer, name string, findings []checks.Finding) {
	fmt.Fprintf(out, "\n--- %s FAILED ---\n", name)
	for _, f := range findings {
		fmt.Fprintf(out, "  %s: %s: %s\n", f.Path, f.Kind, f.Message)
	}
}

// runFactorize is the testable form of RunFactorize. It runs the
// supplied verifiers against root, prints per-check and total
// wall-clock timings to out, and returns 0 on full success, 1 on
// any failed check. Execution order is the same ascending-name order
// used by the public RunFactorize; timings appear only in this text
// output and never in JSON, digest, or evidence artifacts.
//
// When metrics is non-nil and LEAMAS_FACTORIZE_METRICS_FILE is set,
// machine-readable metrics are written atomically to the specified path.
func runFactorize(
	out io.Writer,
	clk clock,
	root string,
	verifiers []Verifier,
	metrics *MetricsCollection,
) int {
	// Sort a local copy so we never mutate the caller's slice.
	sorted := make([]Verifier, len(verifiers))
	copy(sorted, verifiers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	startedAt := clk.Now()
	failed := false

	// Start metrics collection if enabled
	if metrics != nil {
		metrics.StartRun()
	}

	ordinal := 1
	for _, v := range sorted {
		findings := runCheck(out, clk, v.Name, func() []checks.Finding {
			return v.Run(root)
		}, metrics, ordinal, root)
		if len(findings) > 0 {
			failed = true
			printFailureFindings(out, v.Name, findings)
		}
		ordinal++
	}

	elapsed := clk.Now().Sub(startedAt)

	// Finalize metrics if enabled
	if metrics != nil {
		rusage := collectRusage()
		subject := MetricsSubject{
			WorktreeState: "clean",
		}
		status := "pass"
		if failed {
			status = "fail"
		}
		if err := metrics.FinalizeRun(status, exitCode(failed), elapsed, rusage, subject, "controlled-warm", 1); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to write metrics: %v\n", err)
		}
	}

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
