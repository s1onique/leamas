package doctrinecompiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// cycleTestDeadline bounds the runtime of every cycle-detection test
// in this file. A test that runs past this deadline fails with a
// hang message, guaranteeing the verifier terminates regardless of
// input shape. The value is intentionally loose (10s) so that slow
// CI does not flake the test, but tight enough to catch a real
// non-terminating DFS in seconds rather than minutes.
const cycleTestDeadline = 10 * time.Second

// runWithDeadline runs fn() in a worker goroutine and returns its
// result over a channel. It is the correct way to bound the runtime
// of an operation that may hang: only the operation runs in the
// worker; setup, assertions, and any t.Fatal/t.Fatalf calls remain
// on the test goroutine, where the testing package requires them.
//
// Callers MUST NOT call any t.Fatal-family method from inside fn; the
// result is inspected on the test goroutine after the deadline.
func runWithDeadline(t *testing.T, timeout time.Duration, fn func() error) error {
	t.Helper()
	result := make(chan error, 1)
	go func() {
		result <- fn()
	}()
	select {
	case err := <-result:
		return err
	case <-time.After(timeout):
		t.Fatalf("operation exceeded %s deadline; verifier likely hangs", timeout)
		return nil
	}
}

// makeResolver is a tiny shim used by the make-cycle tests. The
// resolver is only required for the contract-style assertions
// (verifyMakefileTargetDep); here we only exercise parseMakeDeps
// and makeReachability, which need a TargetPath-derived string but
// operate on plain Makefile bytes.
func makeResolverFor(t *testing.T, target string) *Resolver {
	t.Helper()
	r, err := NewResolver(target)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	return r
}

// writeMakefile writes body to <target>/Makefile.
func writeMakefile(t *testing.T, target, body string) {
	t.Helper()
	if err := osWriteFileImpl(filepath.Join(target, "Makefile"), []byte(body), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
}

// TestMakeCycleDirectMultiNode proves the verifier rejects a
// direct two-node cycle and returns deterministically without
// hanging.
func TestMakeCycleDirectMultiNode(t *testing.T) {
	target := t.TempDir()
	writeMakefile(t, target, "gate: a\na: gate\n")
	resolver := makeResolverFor(t, target)
	err := runWithDeadline(t, cycleTestDeadline, func() error {
		return verifyMakefileTargetDep(resolver, TargetPath("Makefile"), "gate", "factorize")
	})
	if err == nil {
		t.Fatalf("expected cycle rejection")
	}
	if !strings.Contains(err.Error(), "cycle") &&
		!strings.Contains(err.Error(), "a") {
		t.Errorf("error does not mention the cycle: %v", err)
	}
}

// TestMakeCycleLongerChain proves the verifier rejects a
// longer cycle a -> b -> c -> a without hanging.
func TestMakeCycleLongerChain(t *testing.T) {
	target := t.TempDir()
	writeMakefile(t, target, "gate: a\na: b\nb: c\nc: a\n")
	resolver := makeResolverFor(t, target)
	err := runWithDeadline(t, cycleTestDeadline, func() error {
		return verifyMakefileTargetDep(resolver, TargetPath("Makefile"), "gate", "factorize")
	})
	if err == nil {
		t.Fatalf("expected cycle rejection")
	}
}

// TestMakeCycleWithReachableDesiredDep covers the case where a
// reachable desired dependency exists alongside an unreachable cycle
// in a sibling branch. The verifier must reject the cycle rather
// than return success based on the reachable success path.
func TestMakeCycleWithReachableDesiredDep(t *testing.T) {
	target := t.TempDir()
	writeMakefile(t, target, "gate: factorize cycle-side\nfactorize:\n\t@echo ok\ncycle-side: a\na: cycle-side\n")
	resolver := makeResolverFor(t, target)
	err := runWithDeadline(t, cycleTestDeadline, func() error {
		return verifyMakefileTargetDep(resolver, TargetPath("Makefile"), "gate", "factorize")
	})
	if err == nil {
		t.Fatalf("expected cycle rejection even with reachable desired dep")
	}
}

// TestMakeCycleContinuationSyntax covers a cycle embedded inside a
// backslash-newline dependency continuation. The parser must still
// detect the cycle.
func TestMakeCycleContinuationSyntax(t *testing.T) {
	target := t.TempDir()
	writeMakefile(t, target, "gate: a \\\n    more\na: gate\n")
	resolver := makeResolverFor(t, target)
	err := runWithDeadline(t, cycleTestDeadline, func() error {
		return verifyMakefileTargetDep(resolver, TargetPath("Makefile"), "gate", "factorize")
	})
	if err == nil {
		t.Fatalf("expected cycle rejection under continuation")
	}
}

// TestMakeReachabilityDetectsCycle is a unit test that exercises
// makeReachability directly, so the regression is locked even if
// the higher-level verifyMakefileTargetDep helper changes shape.
func TestMakeReachabilityDetectsCycle(t *testing.T) {
	deps := map[string][]string{
		"gate": {"a"},
		"a":    {"gate"},
	}
	var (
		reach map[string]bool
		cycle string
	)
	runWithDeadline(t, cycleTestDeadline, func() error {
		reach, cycle = makeReachability(deps, "gate")
		return nil
	})
	if cycle == "" {
		t.Errorf("expected cycle node, got none")
	}
	if !reach["a"] || !reach["gate"] {
		t.Errorf("expected both gate and a reachable")
	}
}

// TestMakeReachabilityLongCycle covers a longer cycle.
func TestMakeReachabilityLongCycle(t *testing.T) {
	deps := map[string][]string{
		"gate": {"a"},
		"a":    {"b"},
		"b":    {"c"},
		"c":    {"a"},
	}
	var cycle string
	runWithDeadline(t, cycleTestDeadline, func() error {
		_, cycle = makeReachability(deps, "gate")
		return nil
	})
	if cycle == "" {
		t.Errorf("expected cycle node, got none")
	}
}

// TestMakeReachabilityBranchWithCycle verifies that a sibling cycle
// is detected even when the desired dependency is reachable.
func TestMakeReachabilityBranchWithCycle(t *testing.T) {
	deps := map[string][]string{
		"gate":      {"factorize", "branch"},
		"factorize": {},
		"branch":    {"x"},
		"x":         {"y"},
		"y":         {"x"},
	}
	var (
		reach map[string]bool
		cycle string
	)
	runWithDeadline(t, cycleTestDeadline, func() error {
		reach, cycle = makeReachability(deps, "gate")
		return nil
	})
	if !reach["factorize"] {
		t.Errorf("factorize should be reachable")
	}
	if cycle == "" {
		t.Errorf("expected cycle node in sibling branch")
	}
}

// _ ensures os is imported for the tempdir helpers above.
var _ = os.Stat
