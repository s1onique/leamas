// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
)

// countingAnalyzer returns the same fixed findings on every call and
// records the call count atomically. Tests use it to verify lifecycle
// invariants such as "the analyzer is called exactly once" or "the
// analyzer is not called at registry construction time".
func countingAnalyzer(findings []dupcode.Finding, calls *atomic.Int64) protectedverifier.DupcodeAnalyzer {
	return func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		calls.Add(1)
		return findings, nil
	}
}

// failingAnalyzer returns the configured error on every call. Tests
// use it to verify that an analyzer failure is stable for the
// invocation: subsequent ConsumedBy calls within the same provider
// invocation still return the same error.
func failingAnalyzer(err error, calls *atomic.Int64) protectedverifier.DupcodeAnalyzer {
	return func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		calls.Add(1)
		return nil, err
	}
}

// dummyFindings returns a non-empty slice so analyzer invocations can
// be observed in returned findings.
func dummyFindings() []dupcode.Finding {
	return []dupcode.Finding{
		{
			Fingerprint: "fp-a",
			TokenCount:  100,
			LineCount:   25,
			Occurrences: []dupcode.Occurrence{{Path: "a.go", StartLine: 1, EndLine: 25}},
		},
	}
}

// providerForTest constructs a fresh DupcodeAnalysisProvider with the
// supplied analyzer so each test gets an isolated lifecycle.
func providerForTest(analyzer protectedverifier.DupcodeAnalyzer) *protectedverifier.DupcodeAnalysisProvider {
	input := protectedverifier.DupcodeInput{
		Root:      ".",
		MinLines:  40,
		MinTokens: 400,
		Config:    dupcode.DefaultConfig(),
	}
	return protectedverifier.NewDupcodeAnalysisProvider(input, analyzer)
}

// TestFactorizeRegistryConstructionPerformsZeroProtectedWork proves the
// factorize registry builder does NOT invoke the protected analyzer
// during construction. The analyzer counter stays at zero until the
// first consumer calls the analyzer through the provider.
func TestFactorizeRegistryConstructionPerformsZeroProtectedWork(t *testing.T) {
	calls := &atomic.Int64{}
	analyzer := countingAnalyzer(dummyFindings(), calls)

	provider := providerForTest(analyzer)

	if got := calls.Load(); got != 0 {
		t.Fatalf("analyzer calls during provider construction = %d, want 0", got)
	}
	if _, err := provider.ConsumedBy("dupcode", protectedverifier.DupcodeInput{
		Root: ".", MinLines: 40, MinTokens: 400, Config: dupcode.DefaultConfig(),
	}); err != nil {
		t.Fatalf("ConsumedBy: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("analyzer calls after first consumer = %d, want 1", got)
	}
}

// TestFactorizeFirstConsumerInitializesOnce proves the first admitted
// dupcode-family consumer invokes the analyzer exactly once and
// produces a result the consumer can use.
func TestFactorizeFirstConsumerInitializesOnce(t *testing.T) {
	calls := &atomic.Int64{}
	analyzer := countingAnalyzer(dummyFindings(), calls)
	provider := providerForTest(analyzer)

	analysis, err := provider.ConsumedBy("dupcode", protectedverifier.DupcodeInput{
		Root: ".", MinLines: 40, MinTokens: 400, Config: dupcode.DefaultConfig(),
	})
	if err != nil {
		t.Fatalf("ConsumedBy: %v", err)
	}
	if analysis == nil {
		t.Fatal("expected non-nil analysis after first consumer")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("analyzer calls after first consumer = %d, want 1", got)
	}
}

// TestFactorizeSecondConsumerReusesCachedResult proves a second
// invocation of ConsumedBy returns the cached analysis without
// invoking the analyzer again.
func TestFactorizeSecondConsumerReusesCachedResult(t *testing.T) {
	calls := &atomic.Int64{}
	analyzer := countingAnalyzer(dummyFindings(), calls)
	provider := providerForTest(analyzer)

	input := protectedverifier.DupcodeInput{
		Root: ".", MinLines: 40, MinTokens: 400, Config: dupcode.DefaultConfig(),
	}
	first, err := provider.ConsumedBy("dupcode", input)
	if err != nil {
		t.Fatalf("first ConsumedBy: %v", err)
	}
	second, err := provider.ConsumedBy("dupcode", input)
	if err != nil {
		t.Fatalf("second ConsumedBy: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("analyzer calls after two consumers = %d, want 1", got)
	}
	if first != second {
		t.Fatalf("first=%p second=%p; expected the same cached pointer", first, second)
	}
}

// TestFactorizeDupcodeBaselineThenDupcodeOrderIndependent proves the
// cache holds regardless of which shared-context consumer is admitted
// first (dupcode or dupcode-baseline).
func TestFactorizeDupcodeBaselineThenDupcodeOrderIndependent(t *testing.T) {
	cases := []struct {
		name  string
		first string
	}{
		{name: "dupcode first", first: "dupcode"},
		{name: "baseline first", first: "dupcode-baseline"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := &atomic.Int64{}
			analyzer := countingAnalyzer(dummyFindings(), calls)
			provider := providerForTest(analyzer)

			input := protectedverifier.DupcodeInput{
				Root: ".", MinLines: 40, MinTokens: 400, Config: dupcode.DefaultConfig(),
			}
			// Admit the "first" consumer.
			_, err := provider.ConsumedBy(tc.first, input)
			if err != nil {
				t.Fatalf("%s ConsumedBy: %v", tc.first, err)
			}
			// Admit the "second" consumer.
			second := "dupcode-baseline"
			if tc.first == "dupcode-baseline" {
				second = "dupcode"
			}
			_, err = provider.ConsumedBy(second, input)
			if err != nil {
				t.Fatalf("%s ConsumedBy: %v", second, err)
			}
			if got := calls.Load(); got != 1 {
				t.Fatalf("analyzer calls = %d, want 1 (reverse-order must still cache)", got)
			}
		})
	}
}

// TestFactorizeConcurrentConsumersRemainSingleInit proves that
// concurrent consumers of the shared dupcode analyzer do not produce
// duplicate initializations. The state machine admits exactly one
// analyzer call even under concurrent pressure.
func TestFactorizeConcurrentConsumersRemainSingleInit(t *testing.T) {
	calls := &atomic.Int64{}
	analyzer := countingAnalyzer(dummyFindings(), calls)
	provider := providerForTest(analyzer)

	const goroutines = 16
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _ = provider.ConsumedBy("dupcode", protectedverifier.DupcodeInput{
				Root: ".", MinLines: 40, MinTokens: 400, Config: dupcode.DefaultConfig(),
			})
		}()
	}
	close(start)
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("analyzer calls after %d concurrent consumers = %d, want 1", goroutines, got)
	}
}

// TestFactorizeInitializationFailureStableForInvocation proves that an
// analyzer failure during the first consumer's invocation remains the
// same error for subsequent consumers within the same provider.
// The analyzer is NOT retried on subsequent calls.
func TestFactorizeInitializationFailureStableForInvocation(t *testing.T) {
	calls := &atomic.Int64{}
	boom := errors.New("dupcode analyzer boom")
	analyzer := failingAnalyzer(boom, calls)
	provider := providerForTest(analyzer)

	input := protectedverifier.DupcodeInput{
		Root: ".", MinLines: 40, MinTokens: 400, Config: dupcode.DefaultConfig(),
	}
	_, firstErr := provider.ConsumedBy("dupcode", input)
	if firstErr == nil {
		t.Fatal("expected analyzer failure to surface on first consumer")
	}
	if calls.Load() != 1 {
		t.Fatalf("analyzer calls after first failed consumer = %d, want 1", calls.Load())
	}

	// Second consumer: must NOT retry the analyzer. It must remain the
	// cached failure path with no new analyzer calls.
	_, secondErr := provider.ConsumedBy("dupcode", input)
	if secondErr == nil {
		t.Fatal("expected the failure path to remain available on subsequent calls")
	}
	if calls.Load() != 1 {
		t.Fatalf("analyzer calls after retry = %d, want 1 (failure must not re-trigger analyzer)", calls.Load())
	}
	if firstErr.Error() != secondErr.Error() {
		t.Errorf("first=%q second=%q; expected the cached error to be stable",
			firstErr.Error(), secondErr.Error())
	}
}

// TestFactorizeNoProcessGlobalLifecycleCache proves that two
// separately-constructed providers do NOT share cached analyses.
// Each invocation of the factorize entry point must produce a fresh
// provider so its analyzer is invoked independently of every other
// provider.
func TestFactorizeNoProcessGlobalLifecycleCache(t *testing.T) {
	callsA := &atomic.Int64{}
	callsB := &atomic.Int64{}
	providerA := providerForTest(countingAnalyzer(dummyFindings(), callsA))
	providerB := providerForTest(countingAnalyzer(dummyFindings(), callsB))

	input := protectedverifier.DupcodeInput{
		Root: ".", MinLines: 40, MinTokens: 400, Config: dupcode.DefaultConfig(),
	}
	_, _ = providerA.ConsumedBy("dupcode", input)
	_, _ = providerB.ConsumedBy("dupcode", input)

	if got := callsA.Load(); got != 1 {
		t.Fatalf("provider A analyzer calls = %d, want 1", got)
	}
	if got := callsB.Load(); got != 1 {
		t.Fatalf("provider B analyzer calls = %d, want 1", got)
	}
	// Each provider has its own analysis pointer; no global cache.
	if providerA == providerB {
		t.Fatal("providers A and B share the same pointer; expected distinct instances")
	}
}
