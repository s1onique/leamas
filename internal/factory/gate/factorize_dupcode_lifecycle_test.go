// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
	"github.com/s1onique/leamas/internal/factory/registry"
)

type factorizeDupcodeCounters struct {
	thresholdReads    atomic.Int64
	analyzerCreations atomic.Int64
	providerCreations atomic.Int64
	scans             atomic.Int64
}

type factorizeDupcodeTotals struct {
	thresholdReads    int64
	analyzerCreations int64
	providerCreations int64
	scans             int64
}

func (c *factorizeDupcodeCounters) totals() factorizeDupcodeTotals {
	return factorizeDupcodeTotals{
		thresholdReads:    c.thresholdReads.Load(),
		analyzerCreations: c.analyzerCreations.Load(),
		providerCreations: c.providerCreations.Load(),
		scans:             c.scans.Load(),
	}
}

func (c *factorizeDupcodeCounters) assert(t *testing.T, want factorizeDupcodeTotals) {
	t.Helper()
	if got := c.totals(); got != want {
		t.Fatalf("dependency totals = %+v, want %+v", got, want)
	}
}

func factorizeActualFindings(fingerprint string) []dupcode.Finding {
	return []dupcode.Finding{{
		Fingerprint:       fingerprint[:40],
		StableFingerprint: fingerprint,
		TokenCount:        400,
		LineCount:         40,
		Occurrences: []dupcode.Occurrence{{
			Path:      "actual.go",
			StartLine: 1,
			EndLine:   40,
		}},
	}}
}

func countingFactorizeDeps(
	counters *factorizeDupcodeCounters,
	findings []dupcode.Finding,
	scanErr error,
) factorizeDupcodeDeps {
	return factorizeDupcodeDeps{
		ReadThresholds: func(string) (int, int, error) {
			counters.thresholdReads.Add(1)
			return 40, 400, nil
		},
		NewAnalyzer: func() protectedverifier.DupcodeAnalyzer {
			counters.analyzerCreations.Add(1)
			return func(string, dupcode.Config) ([]dupcode.Finding, error) {
				counters.scans.Add(1)
				return findings, scanErr
			}
		},
		NewProvider: func(
			input protectedverifier.DupcodeInput,
			analyzer protectedverifier.DupcodeAnalyzer,
		) *protectedverifier.DupcodeAnalysisProvider {
			counters.providerCreations.Add(1)
			return protectedverifier.NewDupcodeAnalysisProvider(input, analyzer)
		},
	}
}

func factorizeDupcodeVerifier(t *testing.T, verifiers []registry.Verifier, name string) registry.Verifier {
	t.Helper()
	var matches []registry.Verifier
	for _, verifier := range verifiers {
		if verifier.Name == name {
			matches = append(matches, verifier)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("verifier %q matches = %d, want exactly 1", name, len(matches))
	}
	if matches[0].Run == nil {
		t.Fatalf("verifier %q has nil Run", name)
	}
	return matches[0]
}

func assertFactorizeActualResult(t *testing.T, name string, findings []checks.Finding) {
	t.Helper()
	wantKind := "new_duplicate"
	if name == "dupcode-baseline" {
		wantKind = "dupcode_baseline_drift"
	}
	if len(findings) != 1 {
		t.Fatalf("%s findings = %#v, want one %s finding", name, findings, wantKind)
	}
	if findings[0].Kind != wantKind {
		t.Fatalf("%s finding kind = %q, want %q", name, findings[0].Kind, wantKind)
	}
}

func factorizeLifecycleRegistry(t *testing.T, counters *factorizeDupcodeCounters) []registry.Verifier {
	t.Helper()
	fingerprint := strings.Repeat("a", 64)
	verifiers, err := factorizeVerifiersWithDeps(
		findRepoRoot(t),
		countingFactorizeDeps(counters, factorizeActualFindings(fingerprint), nil),
	)
	if err != nil {
		t.Fatalf("factorizeVerifiersWithDeps: %v", err)
	}
	return verifiers
}

func TestFactorizeConstructionPerformsZeroProtectedSetup(t *testing.T) {
	counters := &factorizeDupcodeCounters{}
	verifiers := factorizeLifecycleRegistry(t, counters)
	factorizeDupcodeVerifier(t, verifiers, "dupcode")
	factorizeDupcodeVerifier(t, verifiers, "dupcode-baseline")
	counters.assert(t, factorizeDupcodeTotals{})
}

func TestFactorizeFirstConsumerAndSecondConsumerUseOneLifecycle(t *testing.T) {
	counters := &factorizeDupcodeCounters{}
	verifiers := factorizeLifecycleRegistry(t, counters)
	dupcodeVerifier := factorizeDupcodeVerifier(t, verifiers, "dupcode")
	baselineVerifier := factorizeDupcodeVerifier(t, verifiers, "dupcode-baseline")

	first := dupcodeVerifier.Run("ignored-caller-root")
	assertFactorizeActualResult(t, "dupcode", first)
	counters.assert(t, factorizeDupcodeTotals{1, 1, 1, 1})

	beforeSecond := counters.totals()
	second := baselineVerifier.Run("another-ignored-root")
	assertFactorizeActualResult(t, "dupcode-baseline", second)
	if additional := subtractFactorizeTotals(counters.totals(), beforeSecond); additional != (factorizeDupcodeTotals{}) {
		t.Fatalf("second actual verifier additional totals = %+v, want zero", additional)
	}
}

func subtractFactorizeTotals(after, before factorizeDupcodeTotals) factorizeDupcodeTotals {
	return factorizeDupcodeTotals{
		thresholdReads:    after.thresholdReads - before.thresholdReads,
		analyzerCreations: after.analyzerCreations - before.analyzerCreations,
		providerCreations: after.providerCreations - before.providerCreations,
		scans:             after.scans - before.scans,
	}
}

func TestFactorizeReverseOrderUsesOneLifecyclePerInvocation(t *testing.T) {
	orders := [][]string{
		{"dupcode", "dupcode-baseline"},
		{"dupcode-baseline", "dupcode"},
	}
	for _, order := range orders {
		t.Run(strings.Join(order, "-then-"), func(t *testing.T) {
			counters := &factorizeDupcodeCounters{}
			verifiers := factorizeLifecycleRegistry(t, counters)
			for _, name := range order {
				result := factorizeDupcodeVerifier(t, verifiers, name).Run("untrusted-root")
				assertFactorizeActualResult(t, name, result)
			}
			counters.assert(t, factorizeDupcodeTotals{1, 1, 1, 1})
		})
	}
}

func TestFactorizeConcurrentActualClosuresShareOneInitialization(t *testing.T) {
	counters := &factorizeDupcodeCounters{}
	verifiers := factorizeLifecycleRegistry(t, counters)
	dupcodeRun := factorizeDupcodeVerifier(t, verifiers, "dupcode").Run
	baselineRun := factorizeDupcodeVerifier(t, verifiers, "dupcode-baseline").Run

	const callers = 24
	results := make([][]checks.Finding, callers)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for i := range results {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			if index%2 == 0 {
				results[index] = dupcodeRun("concurrent-root")
				return
			}
			results[index] = baselineRun("concurrent-root")
		}(i)
	}
	close(start)
	wait.Wait()

	counters.assert(t, factorizeDupcodeTotals{1, 1, 1, 1})
	for i, result := range results {
		name := "dupcode"
		if i%2 != 0 {
			name = "dupcode-baseline"
		}
		assertFactorizeActualResult(t, name, result)
		if !reflect.DeepEqual(result, results[i%2]) {
			t.Fatalf("caller %d result differs from equivalent %s result", i, name)
		}
	}
	results[0][0].Message = "consumer mutation"
	if results[2][0].Message == "consumer mutation" {
		t.Fatal("concurrent dupcode result slices alias")
	}
	results[1][0].Message = "baseline consumer mutation"
	if results[3][0].Message == "baseline consumer mutation" {
		t.Fatal("concurrent baseline result slices alias")
	}
}
