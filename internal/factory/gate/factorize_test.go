// Package gate provides tests for the factorize runner's wall-clock
// timing instrumentation. These tests inject a deterministic clock and
// sampler so they can assert exact behavior without sleeping.
package gate

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// fakeClock returns successive timestamps from a fixed slice.
type fakeClock struct {
	t     *testing.T
	times []time.Time
	next  int
}

func (c *fakeClock) Now() time.Time {
	c.t.Helper()
	if c.next >= len(c.times) {
		c.t.Fatalf("fakeClock.Now() called %d times; only %d timestamps supplied",
			c.next+1, len(c.times))
	}
	value := c.times[c.next]
	c.next++
	return value
}

func newFakeClock(t *testing.T, base time.Time, deltas ...time.Duration) *fakeClock {
	t.Helper()
	times := make([]time.Time, len(deltas)+1)
	times[0] = base
	for i, d := range deltas {
		times[i+1] = times[i].Add(d)
	}
	return &fakeClock{t: t, times: times}
}

// fakeSampler is a deterministic sampler for testing.
type fakeSampler struct {
	snapshots []ResourceSnapshot
	next      int
	err       error
}

func newFakeSampler(snapshots ...ResourceSnapshot) *fakeSampler {
	return &fakeSampler{snapshots: snapshots, next: 0}
}

func (f *fakeSampler) Sample() (ResourceSnapshot, error) {
	if f.err != nil {
		return ResourceSnapshot{}, f.err
	}
	if f.next >= len(f.snapshots) {
		f.snapshots = append(f.snapshots, ResourceSnapshot{})
	}
	s := f.snapshots[f.next]
	f.next++
	return s, nil
}

func (f *fakeSampler) setError(err error) {
	f.err = err
}

// testVerifier creates a Verifier for testing.
func testVerifier(name string, run func(string) []checks.Finding) Verifier {
	return Verifier{
		Name: name,
		Run:  run,
		Execution: ExecutionDefinition{
			Kind:             ExecutionInProcess,
			ImplementationID: "internal/factory/gate.testVerifier",
			EnvVars:          []string{"GOFLAGS"},
		},
		Cache: CacheSemantics{
			GoBuildCache:      CacheNotApplicable,
			GoTestResultCache: CacheModeNA,
		},
	}
}

func TestRunCheck_PrintsElapsedTimeOnSuccess(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 1234*time.Millisecond)
	sampler := newFakeSampler(ResourceSnapshot{}, ResourceSnapshot{})

	var out bytes.Buffer
	v := testVerifier("docs", func(string) []checks.Finding { return nil })

	findings, err := runCheck(&out, clk, v, nil, 1, ".", sampler)
	if err != nil {
		t.Fatalf("runCheck() error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("runCheck() findings=%v, want empty", findings)
	}

	const want = "  docs: OK: 1.23s\n"
	if got := out.String(); got != want {
		t.Fatalf("runCheck() output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestRunCheck_PrintsElapsedTimeOnFailure(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 1234*time.Millisecond)
	sampler := newFakeSampler(ResourceSnapshot{}, ResourceSnapshot{})

	var out bytes.Buffer
	v := testVerifier("docs", func(string) []checks.Finding {
		return []checks.Finding{{Path: "p", Kind: "k", Message: "m"}}
	})

	findings, err := runCheck(&out, clk, v, nil, 1, ".", sampler)
	if err != nil {
		t.Fatalf("runCheck() error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("runCheck() findings=%v, want 1", findings)
	}

	const want = "  docs: FAILED: 1.23s\n"
	if got := out.String(); got != want {
		t.Fatalf("runCheck() output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestRunCheck_FormatsSubSecondDurations(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 7*time.Millisecond)
	sampler := newFakeSampler(ResourceSnapshot{}, ResourceSnapshot{})

	var out bytes.Buffer
	v := testVerifier("docs", func(string) []checks.Finding { return nil })

	runCheck(&out, clk, v, nil, 1, ".", sampler)

	const want = "  docs: OK: 0.01s\n"
	if got := out.String(); got != want {
		t.Fatalf("runCheck() output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestRunCheck_PreSampleErrorFailsCheck(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 0)
	sampler := newFakeSampler()
	sampler.setError(assertionError("sampler failed"))

	var out bytes.Buffer
	v := testVerifier("docs", func(string) []checks.Finding { return nil })

	_, err := runCheck(&out, clk, v, nil, 1, ".", sampler)
	if err == nil {
		t.Fatalf("runCheck() expected error from pre-sample failure")
	}
}

func TestRunCheck_PostSampleErrorFailsCheck(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 0)
	sampler := newFakeSampler(ResourceSnapshot{})
	sampler.setError(assertionError("sampler failed"))

	var out bytes.Buffer
	v := testVerifier("docs", func(string) []checks.Finding { return nil })

	_, err := runCheck(&out, clk, v, nil, 1, ".", sampler)
	if err == nil {
		t.Fatalf("runCheck() expected error from post-sample failure")
	}
}

func assertionError(msg string) error {
	return &testError{msg: msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

func TestRunFactorize_PrintsTotalOnSuccess(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start,
		0, 100*time.Millisecond, 0, 50*time.Millisecond, 0)

	var out bytes.Buffer
	verifiers := []Verifier{
		testVerifier("beta", func(string) []checks.Finding { return nil }),
		testVerifier("alpha", func(string) []checks.Finding { return nil }),
	}
	sampler := newFakeSampler(ResourceSnapshot{}, ResourceSnapshot{}, ResourceSnapshot{}, ResourceSnapshot{})

	code := runFactorize(&out, clk, ".", verifiers, nil, sampler)
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

func TestRunFactorize_PrintsFailureAndTotalOnError(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 0, 100*time.Millisecond, 0, 100*time.Millisecond, 0)

	var out bytes.Buffer
	verifiers := []Verifier{
		testVerifier("alpha", func(string) []checks.Finding { return nil }),
		testVerifier("beta", func(string) []checks.Finding {
			return []checks.Finding{{Path: "p", Kind: "k", Message: "m"}}
		}),
	}
	sampler := newFakeSampler(ResourceSnapshot{}, ResourceSnapshot{}, ResourceSnapshot{}, ResourceSnapshot{})

	code := runFactorize(&out, clk, ".", verifiers, nil, sampler)
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

func TestRunFactorize_PreservesExitCodeOnFailure(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 0, 0, 0, 0, 0)
	verifiers := []Verifier{
		testVerifier("alpha", func(string) []checks.Finding {
			return []checks.Finding{{Path: "p", Kind: "k", Message: "m"}}
		}),
		testVerifier("beta", func(string) []checks.Finding {
			return []checks.Finding{{Path: "q", Kind: "k", Message: "m"}}
		}),
	}
	sampler := newFakeSampler(ResourceSnapshot{}, ResourceSnapshot{}, ResourceSnapshot{}, ResourceSnapshot{})

	code := runFactorize(&bytes.Buffer{}, clk, ".", verifiers, nil, sampler)
	if code != 1 {
		t.Fatalf("runFactorize() code=%d, want 1", code)
	}
}

func TestRunFactorize_SortsByName(t *testing.T) {
	start := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clk := newFakeClock(t, start, 0, 0, 0, 0, 0, 0, 0, 0)
	verifiers := []Verifier{
		testVerifier("mu", func(string) []checks.Finding { return nil }),
		testVerifier("alpha", func(string) []checks.Finding { return nil }),
		testVerifier("zeta", func(string) []checks.Finding { return nil }),
	}
	sampler := newFakeSampler(
		ResourceSnapshot{}, ResourceSnapshot{}, ResourceSnapshot{}, ResourceSnapshot{},
		ResourceSnapshot{}, ResourceSnapshot{}, ResourceSnapshot{}, ResourceSnapshot{},
	)

	var out bytes.Buffer
	runFactorize(&out, clk, ".", verifiers, nil, sampler)

	got := out.String()
	wantSubstrs := []string{"  alpha: OK:", "  mu: OK:", "  zeta: OK:"}
	for _, s := range wantSubstrs {
		if !strings.Contains(got, s) {
			t.Fatalf("runFactorize() output missing %q in %q", s, got)
		}
	}
	alphaIdx := strings.Index(got, "alpha: OK:")
	muIdx := strings.Index(got, "mu: OK:")
	zetaIdx := strings.Index(got, "zeta: OK:")
	if !(alphaIdx < muIdx && muIdx < zetaIdx) {
		t.Fatalf("runFactorize() execution order wrong: alpha=%d mu=%d zeta=%d output=%q",
			alphaIdx, muIdx, zetaIdx, got)
	}
}
