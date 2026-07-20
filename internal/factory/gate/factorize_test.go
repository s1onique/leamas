// Package gate provides tests for the factorize runner's wall-clock
// timing instrumentation. These tests inject a deterministic clock so
// they can assert exact elapsed-time formatting without sleeping.
package gate

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// fakeClock returns successive timestamps from a fixed slice. Each
// Now() call advances the cursor; tests supply exactly the times they
// expect runCheck / runFactorize to observe. If Now() is called more
// times than expected, fakeClock fails the test with a diagnostic
// rather than panicking with an opaque index-out-of-range error.
type fakeClock struct {
	t     *testing.T
	times []time.Time
	next  int
}

func (c *fakeClock) Now() time.Time {
	c.t.Helper()
	if c.next >= len(c.times) {
		c.t.Fatalf(
			"fakeClock.Now() called %d times; only %d timestamps supplied",
			c.next+1,
			len(c.times),
		)
	}
	value := c.times[c.next]
	c.next++
	return value
}

// newFakeClock builds a fakeClock from a base time and a list of
// incremental durations. The first Now() returns base; each subsequent
// Now() returns the previous timestamp plus the corresponding delta.
// This mirrors how time advances in a real run, so a duration like
// "100ms then 50ms then 20ms" yields 100ms, 150ms, and 170ms.
func newFakeClock(t *testing.T, base time.Time, deltas ...time.Duration) *fakeClock {
	t.Helper()
	times := make([]time.Time, len(deltas)+1)
	times[0] = base
	for i, d := range deltas {
		times[i+1] = times[i].Add(d)
	}
	return &fakeClock{t: t, times: times}
}

// TestRunCheck_PrintsElapsedTimeOnSuccess asserts the success status
// line format: "  <name>: OK: <seconds>s". The injected clock yields
// a deterministic 1.234s delta so the test never sleeps.
func TestRunCheck_PrintsElapsedTimeOnSuccess(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 1234*time.Millisecond)

	var out bytes.Buffer

	findings := runCheck(&out, clk, "docs", func() []checks.Finding {
		return nil
	}, nil, 1, ".")
	if len(findings) != 0 {
		t.Fatalf("runCheck() findings=%v, want empty", findings)
	}

	const want = "  docs: OK: 1.23s\n"
	if got := out.String(); got != want {
		t.Fatalf("runCheck() output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

// TestRunCheck_PrintsElapsedTimeOnFailure asserts the failure status
// line format: "  <name>: FAILED: <seconds>s". The injected clock
// yields a deterministic 1.234s delta so the test never sleeps. The
// check function returns one finding; runCheck must surface it so the
// caller can render detail lines.
func TestRunCheck_PrintsElapsedTimeOnFailure(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 1234*time.Millisecond)

	var out bytes.Buffer

	findings := runCheck(&out, clk, "docs", func() []checks.Finding {
		return []checks.Finding{
			{Path: "p", Kind: "k", Message: "m"},
		}
	}, nil, 1, ".")
	if len(findings) != 1 {
		t.Fatalf("runCheck() findings=%v, want 1", findings)
	}

	const want = "  docs: FAILED: 1.23s\n"
	if got := out.String(); got != want {
		t.Fatalf("runCheck() output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

// TestRunCheck_FormatsSubSecondDurations asserts the formatter always
// emits two decimal places even when the elapsed time is well below
// one second. Guards against accidental %v or integer-only formatting.
func TestRunCheck_FormatsSubSecondDurations(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 7*time.Millisecond)

	var out bytes.Buffer

	runCheck(&out, clk, "docs", func() []checks.Finding { return nil }, nil, 1, ".")

	const want = "  docs: OK: 0.01s\n"
	if got := out.String(); got != want {
		t.Fatalf("runCheck() output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

// TestRunFactorize_PrintsTotalOnSuccess exercises the full success
// path: per-check timing lines, sorted execution order, and a final
// summary line that includes the total wall-clock duration. The
// success summary preserves the existing PASSED vocabulary to avoid
// breaking external scripts that depend on it.
func TestRunFactorize_PrintsTotalOnSuccess(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	// 6 calls: runFactorize start, alpha start, alpha end, beta start,
	// beta end, runFactorize end. Verifier durations: alpha=100ms,
	// beta=50ms; total = 150ms.
	clk := newFakeClock(t, start,
		0,                    // alpha start (no delay)
		100*time.Millisecond, // alpha end (delta since alpha start: 100ms)
		0,                    // beta start (no delay since alpha end)
		50*time.Millisecond,  // beta end (delta since beta start: 50ms)
		0,                    // runFactorize end (no delay since beta end)
	)

	var out bytes.Buffer
	// Deliberately unsorted: alpha < beta, but registered as beta, alpha
	// to prove runFactorize applies the same sort-by-name contract
	// used by RunFactorize.
	verifiers := []Verifier{
		{Name: "beta", Run: func(string) []checks.Finding { return nil }},
		{Name: "alpha", Run: func(string) []checks.Finding { return nil }},
	}

	code := runFactorize(&out, clk, ".", verifiers, nil)
	if code != 0 {
		t.Fatalf("runFactorize() code=%d, want 0", code)
	}

	const want = "" +
		"  alpha: OK: 0.10s\n" +
		"  beta: OK: 0.05s\n" +
		"\n*** FACTORIZE PASSED: 0.15s ***\n"
	if got := out.String(); got != want {
		t.Fatalf("runFactorize() output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

// TestRunFactorize_PrintsFailureAndTotalOnError exercises the failure
// path: failed check emits FAILED with timing, detail lines follow,
// final summary prints FACTORIZE FAILED with total duration.
func TestRunFactorize_PrintsFailureAndTotalOnError(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	// 6 calls: total = 200ms, alpha = 100ms, beta = 100ms.
	clk := newFakeClock(t, start,
		0,
		100*time.Millisecond,
		0,
		100*time.Millisecond,
		0,
	)

	var out bytes.Buffer
	verifiers := []Verifier{
		{Name: "alpha", Run: func(string) []checks.Finding { return nil }},
		{Name: "beta", Run: func(string) []checks.Finding {
			return []checks.Finding{
				{Path: "p", Kind: "k", Message: "m"},
			}
		}},
	}

	code := runFactorize(&out, clk, ".", verifiers, nil)
	if code != 1 {
		t.Fatalf("runFactorize() code=%d, want 1", code)
	}

	const want = "" +
		"  alpha: OK: 0.10s\n" +
		"  beta: FAILED: 0.10s\n" +
		"\n--- beta FAILED ---\n" +
		"  p: k: m\n" +
		"\n*** FACTORIZE FAILED: 0.20s ***\n"
	if got := out.String(); got != want {
		t.Fatalf("runFactorize() output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

// TestRunFactorize_PreservesExitCodeOnFailure asserts that a single
// failed verifier produces exit code 1 (matching RunFactorize's
// pre-existing contract). Multiple failed verifiers do not change
// the exit code.
func TestRunFactorize_PreservesExitCodeOnFailure(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start,
		0,
		0,
		0,
		0,
		0,
	)
	verifiers := []Verifier{
		{Name: "alpha", Run: func(string) []checks.Finding {
			return []checks.Finding{{Path: "p", Kind: "k", Message: "m"}}
		}},
		{Name: "beta", Run: func(string) []checks.Finding {
			return []checks.Finding{{Path: "q", Kind: "k", Message: "m"}}
		}},
	}

	code := runFactorize(&bytes.Buffer{}, clk, ".", verifiers, nil)
	if code != 1 {
		t.Fatalf("runFactorize() code=%d, want 1", code)
	}
}

// TestRunFactorize_SortsByName asserts that verifiers are run in
// ascending name order regardless of registration order. This is the
// same contract as the previous RunFactorize; the timing change must
// not break it.
func TestRunFactorize_SortsByName(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start,
		0, 0, // zeta
		0, 0, // alpha
		0, 0, // mu
		0, // end
	)
	verifiers := []Verifier{
		{Name: "mu", Run: func(string) []checks.Finding { return nil }},
		{Name: "alpha", Run: func(string) []checks.Finding { return nil }},
		{Name: "zeta", Run: func(string) []checks.Finding { return nil }},
	}

	var out bytes.Buffer
	runFactorize(&out, clk, ".", verifiers, nil)

	got := out.String()
	wantSubstrs := []string{"  alpha: OK:", "  mu: OK:", "  zeta: OK:"}
	for _, s := range wantSubstrs {
		if !strings.Contains(got, s) {
			t.Fatalf("runFactorize() output missing %q in %q", s, got)
		}
	}
	// Verify the order: alpha before mu before zeta.
	alphaIdx := strings.Index(got, "alpha: OK:")
	muIdx := strings.Index(got, "mu: OK:")
	zetaIdx := strings.Index(got, "zeta: OK:")
	if !(alphaIdx < muIdx && muIdx < zetaIdx) {
		t.Fatalf("runFactorize() execution order wrong: alpha=%d mu=%d zeta=%d output=%q",
			alphaIdx, muIdx, zetaIdx, got)
	}
}
