// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"errors"
	"strings"
	"testing"
)

func TestFactorizeLifecycleUnknownNameFailsClosed(t *testing.T) {
	counters := &factorizeDupcodeCounters{}
	lifecycle := newFactorizeDupcodeLifecycle(
		findRepoRoot(t),
		countingFactorizeDeps(counters, factorizeActualFindings(strings.Repeat("a", 64)), nil),
	)

	findings := lifecycle.run("unexpected-verifier")
	if len(findings) != 1 {
		t.Fatalf("unknown-name findings = %#v, want exactly one", findings)
	}
	if findings[0].Kind != "factorize_internal_invariant" {
		t.Fatalf("unknown-name kind = %q, want factorize_internal_invariant", findings[0].Kind)
	}
	counters.assert(t, factorizeDupcodeTotals{})
}

func TestFactorizeLifecyclePanicPolicyPropagates(t *testing.T) {
	panicSentinel := errors.New("dependency panic sentinel")
	counters := &factorizeDupcodeCounters{}
	deps := countingFactorizeDeps(counters, nil, nil)
	deps.ReadThresholds = func(string) (int, int, error) {
		panic(panicSentinel)
	}
	lifecycle := newFactorizeDupcodeLifecycle(findRepoRoot(t), deps)

	deferred := false
	func() {
		defer func() {
			deferred = recover() == panicSentinel
		}()
		_ = lifecycle.run("dupcode")
	}()
	if !deferred {
		t.Fatal("dependency panic was not propagated unchanged")
	}
}
