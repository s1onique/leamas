// Package gate provides the factorize runner with wall-clock timings.
//
// Timings are deliberately excluded from JSON, digest, fingerprint,
// and snapshot contracts; they live only in the interactive text
// progress output of `leamas factory factorize`. The clock is injected
// so tests can assert exact elapsed-time formatting without sleeping.
package gate

import (
	"fmt"
	"io"
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
func runCheck(
	out io.Writer,
	clk clock,
	name string,
	check func() []checks.Finding,
) []checks.Finding {
	started := clk.Now()
	findings := check()
	elapsed := clk.Now().Sub(started)

	status := "OK"
	if len(findings) > 0 {
		status = "FAILED"
	}

	fmt.Fprintf(out, "  %s: %s: %.2fs\n", name, status, elapsed.Seconds())
	return findings
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
func runFactorize(
	out io.Writer,
	clk clock,
	root string,
	verifiers []Verifier,
) int {
	// Sort a local copy so we never mutate the caller's slice.
	sorted := make([]Verifier, len(verifiers))
	copy(sorted, verifiers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	startedAt := clk.Now()
	failed := false

	for _, v := range sorted {
		findings := runCheck(out, clk, v.Name, func() []checks.Finding {
			return v.Run(root)
		})
		if len(findings) > 0 {
			failed = true
			printFailureFindings(out, v.Name, findings)
		}
	}

	elapsed := clk.Now().Sub(startedAt)

	if failed {
		fmt.Fprintf(out, "\n*** FACTORIZE FAILED: %.2fs ***\n", elapsed.Seconds())
		return 1
	}

	fmt.Fprintf(out, "\n*** FACTORIZE PASSED: %.2fs ***\n", elapsed.Seconds())
	return 0
}
