// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
	"github.com/s1onique/leamas/internal/factory/registry"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

func assertFactorizeFailureResult(t *testing.T, name string, findings []checks.Finding, sentinel error) {
	t.Helper()
	if len(findings) != 1 {
		t.Fatalf("%s failure findings = %#v, want exactly one", name, findings)
	}
	if findings[0].Kind != "dupcode_error" {
		t.Fatalf("%s failure kind = %q, want dupcode_error", name, findings[0].Kind)
	}
	if !strings.Contains(findings[0].Message, sentinel.Error()) {
		t.Fatalf("%s failure message = %q, want sentinel %q", name, findings[0].Message, sentinel)
	}
}

func runBothFactorizeDupcodeVerifiers(
	t *testing.T,
	verifiers []registry.Verifier,
	sentinel error,
) ([]checks.Finding, []checks.Finding) {
	t.Helper()
	dupcodeResult := factorizeDupcodeVerifier(t, verifiers, "dupcode").Run("ignored")
	baselineResult := factorizeDupcodeVerifier(t, verifiers, "dupcode-baseline").Run("ignored")
	assertFactorizeFailureResult(t, "dupcode", dupcodeResult, sentinel)
	assertFactorizeFailureResult(t, "dupcode-baseline", baselineResult, sentinel)
	return dupcodeResult, baselineResult
}

func TestFactorizeFailureThresholdReadIsStableAndNotRetried(t *testing.T) {
	boom := errors.New("threshold read sentinel")
	counters := &factorizeDupcodeCounters{}
	deps := countingFactorizeDeps(counters, nil, nil)
	deps.ReadThresholds = func(string) (int, int, error) {
		counters.thresholdReads.Add(1)
		return 0, 0, boom
	}
	verifiers, err := factorizeVerifiersWithDeps(findRepoRoot(t), deps)
	if err != nil {
		t.Fatalf("factorizeVerifiersWithDeps: %v", err)
	}
	counters.assert(t, factorizeDupcodeTotals{})

	runBothFactorizeDupcodeVerifiers(t, verifiers, boom)
	counters.assert(t, factorizeDupcodeTotals{thresholdReads: 1})

	lifecycle := newFactorizeDupcodeLifecycle(findRepoRoot(t), deps)
	lifecycle.once.Do(lifecycle.initialize)
	if !errors.Is(lifecycle.initErr, boom) {
		t.Fatalf("initErr = %v, want errors.Is(threshold sentinel)", lifecycle.initErr)
	}
}

func TestFactorizeFailureAnalyzerCreationIsStableAndFailClosed(t *testing.T) {
	counters := &factorizeDupcodeCounters{}
	deps := countingFactorizeDeps(counters, nil, nil)
	deps.NewAnalyzer = func() protectedverifier.DupcodeAnalyzer {
		counters.analyzerCreations.Add(1)
		return nil
	}
	verifiers, err := factorizeVerifiersWithDeps(findRepoRoot(t), deps)
	if err != nil {
		t.Fatalf("factorizeVerifiersWithDeps: %v", err)
	}

	sentinel := factorizeAnalyzerConstructionError{}
	runBothFactorizeDupcodeVerifiers(t, verifiers, sentinel)
	counters.assert(t, factorizeDupcodeTotals{thresholdReads: 1, analyzerCreations: 1})

	separateCounters := &factorizeDupcodeCounters{}
	separateDeps := countingFactorizeDeps(separateCounters, nil, nil)
	separateDeps.NewAnalyzer = func() protectedverifier.DupcodeAnalyzer {
		separateCounters.analyzerCreations.Add(1)
		return nil
	}
	lifecycle := newFactorizeDupcodeLifecycle(findRepoRoot(t), separateDeps)
	lifecycle.once.Do(lifecycle.initialize)
	if !errors.Is(lifecycle.initErr, sentinel) {
		t.Fatalf("initErr = %v, want errors.Is(analyzer construction sentinel)", lifecycle.initErr)
	}
	separateCounters.assert(t, factorizeDupcodeTotals{thresholdReads: 1, analyzerCreations: 1})
}

func TestFactorizeFailureScanIsStableAndNotRetried(t *testing.T) {
	boom := errors.New("scan sentinel")
	counters := &factorizeDupcodeCounters{}
	deps := countingFactorizeDeps(counters, nil, boom)
	verifiers, err := factorizeVerifiersWithDeps(findRepoRoot(t), deps)
	if err != nil {
		t.Fatalf("factorizeVerifiersWithDeps: %v", err)
	}

	dupcodeResult, baselineResult := runBothFactorizeDupcodeVerifiers(t, verifiers, boom)
	counters.assert(t, factorizeDupcodeTotals{1, 1, 1, 1})
	if dupcodeResult[0].Message != baselineResult[0].Message {
		t.Fatalf("scan failure truth differs: dupcode=%q baseline=%q", dupcodeResult[0].Message, baselineResult[0].Message)
	}
}

func TestFactorizeFailureFreshInvocationMayRetry(t *testing.T) {
	boom := errors.New("first invocation threshold failure")
	failedCounters := &factorizeDupcodeCounters{}
	failedDeps := countingFactorizeDeps(failedCounters, nil, nil)
	failedDeps.ReadThresholds = func(string) (int, int, error) {
		failedCounters.thresholdReads.Add(1)
		return 0, 0, boom
	}
	failed, err := factorizeVerifiersWithDeps(findRepoRoot(t), failedDeps)
	if err != nil {
		t.Fatalf("failed registry construction: %v", err)
	}
	factorizeDupcodeVerifier(t, failed, "dupcode").Run("ignored")
	failedCounters.assert(t, factorizeDupcodeTotals{thresholdReads: 1})

	freshCounters := &factorizeDupcodeCounters{}
	fresh := factorizeLifecycleRegistry(t, freshCounters)
	result := factorizeDupcodeVerifier(t, fresh, "dupcode").Run("ignored")
	assertFactorizeActualResult(t, "dupcode", result)
	freshCounters.assert(t, factorizeDupcodeTotals{1, 1, 1, 1})
}

func TestFactorizeDeniedAuthorityPerformsZeroInitialization(t *testing.T) {
	counters := &factorizeDupcodeCounters{}
	actual := factorizeLifecycleRegistry(t, counters)
	dupcodeFamily := []registry.Verifier{
		factorizeDupcodeVerifier(t, actual, "dupcode"),
		factorizeDupcodeVerifier(t, actual, "dupcode-baseline"),
	}
	dispatcher, err := verifierdispatch.NewDispatcher(dupcodeFamily)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	requests := []verifierdispatch.ProfileRequest{
		{VerifierID: "dupcode", Operation: verifierauthority.OperationVerify},
		{VerifierID: "dupcode-baseline", Operation: verifierauthority.OperationVerify},
	}
	factoryCalls := 0
	binding, err := dispatcher.AuthorizeAndBindProfile(
		context.Background(),
		findRepoRoot(t),
		requests,
		&denyingObserver{},
		func([]verifierdispatch.VerifierMetadata) ([]verifierdispatch.FactoryRunner, error) {
			factoryCalls++
			return nil, errors.New("denied path invoked runner factory")
		},
	)
	if err != nil {
		t.Fatalf("AuthorizeAndBindProfile: %v", err)
	}
	if binding.Profile().AuthorizationSucceeded() {
		t.Fatal("denied profile unexpectedly authorized")
	}
	if factoryCalls != 0 {
		t.Fatalf("runner factory calls = %d, want 0", factoryCalls)
	}
	if _, err := binding.Execute(nil); err == nil {
		t.Fatal("denied binding Execute unexpectedly succeeded")
	}
	counters.assert(t, factorizeDupcodeTotals{})
}

func TestFactorizeInvocationIsolationUsesIndependentLifecycles(t *testing.T) {
	countersA := &factorizeDupcodeCounters{}
	countersB := &factorizeDupcodeCounters{}
	fingerprint := strings.Repeat("a", 64)
	depsA := countingFactorizeDeps(countersA, factorizeActualFindings(fingerprint), nil)
	depsB := countingFactorizeDeps(countersB, factorizeActualFindings(fingerprint), nil)

	var inputA, inputB protectedverifier.DupcodeInput
	var providerA, providerB *protectedverifier.DupcodeAnalysisProvider
	newProviderA := depsA.NewProvider
	depsA.NewProvider = func(input protectedverifier.DupcodeInput, analyzer protectedverifier.DupcodeAnalyzer) *protectedverifier.DupcodeAnalysisProvider {
		inputA = input
		providerA = newProviderA(input, analyzer)
		return providerA
	}
	newProviderB := depsB.NewProvider
	depsB.NewProvider = func(input protectedverifier.DupcodeInput, analyzer protectedverifier.DupcodeAnalyzer) *protectedverifier.DupcodeAnalysisProvider {
		inputB = input
		providerB = newProviderB(input, analyzer)
		return providerB
	}

	registryA, err := factorizeVerifiersWithDeps(findRepoRoot(t), depsA)
	if err != nil {
		t.Fatalf("registry A: %v", err)
	}
	registryB, err := factorizeVerifiersWithDeps(findRepoRoot(t), depsB)
	if err != nil {
		t.Fatalf("registry B: %v", err)
	}
	resultA := factorizeDupcodeVerifier(t, registryA, "dupcode").Run("ignored")
	resultB := factorizeDupcodeVerifier(t, registryB, "dupcode").Run("ignored")
	assertFactorizeActualResult(t, "dupcode", resultA)
	assertFactorizeActualResult(t, "dupcode", resultB)
	countersA.assert(t, factorizeDupcodeTotals{1, 1, 1, 1})
	countersB.assert(t, factorizeDupcodeTotals{1, 1, 1, 1})

	if providerA == nil || providerB == nil || providerA == providerB {
		t.Fatalf("providers are not isolated: A=%p B=%p", providerA, providerB)
	}
	inputA.Config.ExcludeDirs[0] = "invocation-a-mutation"
	if inputB.Config.ExcludeDirs[0] == "invocation-a-mutation" {
		t.Fatal("invocation configuration slices alias")
	}
	baselineA := factorizeDupcodeVerifier(t, registryA, "dupcode-baseline").Run("ignored")
	assertFactorizeActualResult(t, "dupcode-baseline", baselineA)
	countersA.assert(t, factorizeDupcodeTotals{1, 1, 1, 1})

	if !reflect.DeepEqual(resultA, resultB) {
		t.Fatalf("independent logical results differ: A=%#v B=%#v", resultA, resultB)
	}
	resultA[0].Message = "invocation A mutation"
	if resultB[0].Message == "invocation A mutation" {
		t.Fatal("invocation result slices alias")
	}
}

func TestFactorizeIsolationFreezesInvocationDependencies(t *testing.T) {
	boundCounters := &factorizeDupcodeCounters{}
	mutatedCounters := &factorizeDupcodeCounters{}
	fingerprint := strings.Repeat("a", 64)
	deps := countingFactorizeDeps(boundCounters, factorizeActualFindings(fingerprint), nil)
	verifiers, err := factorizeVerifiersWithDeps(findRepoRoot(t), deps)
	if err != nil {
		t.Fatalf("factorizeVerifiersWithDeps: %v", err)
	}

	deps = countingFactorizeDeps(mutatedCounters, factorizeActualFindings(strings.Repeat("b", 64)), nil)
	result := factorizeDupcodeVerifier(t, verifiers, "dupcode").Run("ignored")
	assertFactorizeActualResult(t, "dupcode", result)
	boundCounters.assert(t, factorizeDupcodeTotals{1, 1, 1, 1})
	mutatedCounters.assert(t, factorizeDupcodeTotals{})
}
