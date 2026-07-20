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
	verifier Verifier,
	metrics *MetricsCollection,
	ordinal int,
	root string,
) ([]checks.Finding, error) {
	started := clk.Now()
	findings := verifier.Run(root)
	elapsed := clk.Now().Sub(started)

	status := "OK"
	if len(findings) > 0 {
		status = "FAILED"
	}

	fmt.Fprintf(out, "  %s: %s: %.2fs\n", verifier.Name, status, elapsed.Seconds())

	if metrics != nil {
		rusage := collectRusage()
		env := os.Environ()
		if err := metrics.AddCheck(verifier, ordinal, findings, elapsed, rusage, root, env); err != nil {
			return findings, fmt.Errorf("metrics collection for %s: %w", verifier.Name, err)
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
	verifiers []Verifier,
	metrics *MetricsCollection,
) int {
	sorted := make([]Verifier, len(verifiers))
	copy(sorted, verifiers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	startedAt := clk.Now()
	failed := false

	if metrics != nil {
		metrics.StartRun()
	}

	ordinal := 1
	for _, v := range sorted {
		findings, err := runCheck(out, clk, v, metrics, ordinal, root)
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

	if metrics != nil {
		rusage := collectRusage()
		subject := MetricsSubject{WorktreeState: "clean"}
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
