// Package gate provides tests for factorize cache classification from verifier metadata.
package gate

import (
	"sort"
	"testing"
)

const canonicalVerifierCount = 15

// TestCacheSemantics_AllVerifiersHaveCacheMetadata verifies all verifiers have cache semantics.
func TestCacheSemantics_AllVerifiersHaveCacheMetadata(t *testing.T) {
	verifiers := AllVerifiers()
	if len(verifiers) != canonicalVerifierCount {
		t.Fatalf("expected %d canonical verifiers, got %d", canonicalVerifierCount, len(verifiers))
	}

	for _, v := range verifiers {
		if v.Cache.GoBuildCache == "" {
			t.Errorf("verifier %q has no GoBuildCache", v.Name)
		}
		if v.Cache.GoTestResultCache == "" {
			t.Errorf("verifier %q has no GoTestResultCache", v.Name)
		}
	}
}

// TestCacheSemantics_DupcodeDisabled verifies dupcode uses disabled test cache.
func TestCacheSemantics_DupcodeDisabled(t *testing.T) {
	verifiers := AllVerifiers()

	for _, v := range verifiers {
		if v.Name == "dupcode" || v.Name == "dupcode-baseline" {
			if v.Cache.GoTestResultCache != CacheModeDisabled {
				t.Errorf("verifier %q must have test cache disabled, got %s", v.Name, v.Cache.GoTestResultCache)
			}
		}
	}
}

// TestCacheSemantics_ChildProcessVerifiers verifies child-process cache behavior.
func TestCacheSemantics_ChildProcessVerifiers(t *testing.T) {
	verifiers := AllVerifiers()

	for _, v := range verifiers {
		if v.Execution.Kind == ExecutionChild {
			if v.Cache.GoBuildCache == CacheNotApplicable {
				t.Errorf("child-process verifier %q should have cache relevance, got %s", v.Name, v.Cache.GoBuildCache)
			}
		}
	}
}

// verifierNames extracts just the names from a Verifier slice.
func verifierNames(verifiers []Verifier) []string {
	names := make([]string, len(verifiers))
	for i, v := range verifiers {
		names[i] = v.Name
	}
	sort.Strings(names)
	return names
}

// TestAllVerifiersCount_MatchesCanonical verifies AllVerifiers returns 15.
func TestAllVerifiersCount_MatchesCanonical(t *testing.T) {
	verifiers := AllVerifiers()

	if len(verifiers) != canonicalVerifierCount {
		t.Errorf("AllVerifiers() returned %d verifiers, expected %d:\n%v",
			len(verifiers), canonicalVerifierCount, verifierNames(verifiers))
	}
}

// TestVerifierNames_MatchesCanonicalList verifies all expected verifiers are present.
func TestVerifierNames_MatchesCanonicalList(t *testing.T) {
	verifiers := AllVerifiers()
	names := verifierNames(verifiers)

	if len(names) != canonicalVerifierCount {
		t.Fatalf("verifier count mismatch: got %d, expected %d", len(names), canonicalVerifierCount)
	}

	expectedSet := map[string]bool{
		"agent-context": true, "doctrine": true, "doctrine-agent-contracts": true,
		"docs": true, "dupcode": true, "dupcode-baseline": true,
		"domain-boundaries": true, "exec-gate": true, "executable-contract-first": true,
		"forbidden-patterns": true, "git-hooks": true, "language": true,
		"llm-friendly": true, "static-binary": true, "tooling-boundaries": true,
	}
	for _, name := range names {
		if !expectedSet[name] {
			t.Errorf("unexpected verifier: %q", name)
		}
	}
}

// TestMetricsSchema_IsV2 verifies schema version is v2.
func TestMetricsSchema_IsV2(t *testing.T) {
	if MetricsSchema != "factorize-performance-v2" {
		t.Errorf("expected schema v2, got %s", MetricsSchema)
	}
}

// TestExecutionKind_ValidValues verifies execution kind constants.
func TestExecutionKind_ValidValues(t *testing.T) {
	if ExecutionInProcess != "in-process" {
		t.Errorf("ExecutionInProcess = %q, expected %q", ExecutionInProcess, "in-process")
	}
	if ExecutionChild != "child-process" {
		t.Errorf("ExecutionChild = %q, expected %q", ExecutionChild, "child-process")
	}
}

// TestCacheRelevance_ValidValues verifies cache relevance constants.
func TestCacheRelevance_ValidValues(t *testing.T) {
	if CacheRelevant != "relevant" {
		t.Errorf("CacheRelevant = %q", CacheRelevant)
	}
	if CacheNotRelevant != "not-relevant" {
		t.Errorf("CacheNotRelevant = %q", CacheNotRelevant)
	}
	if CacheNotApplicable != "not-applicable" {
		t.Errorf("CacheNotApplicable = %q", CacheNotApplicable)
	}
}

// TestTestResultCacheMode_ValidValues verifies cache mode constants.
func TestTestResultCacheMode_ValidValues(t *testing.T) {
	if CacheModeEnabled != "enabled" {
		t.Errorf("CacheModeEnabled = %q", CacheModeEnabled)
	}
	if CacheModeDisabled != "disabled" {
		t.Errorf("CacheModeDisabled = %q", CacheModeDisabled)
	}
	if CacheModeNA != "not-applicable" {
		t.Errorf("CacheModeNA = %q", CacheModeNA)
	}
}

// TestAllVerifiers_HaveImplementationID verifies all verifiers have implementation IDs.
func TestAllVerifiers_HaveImplementationID(t *testing.T) {
	verifiers := AllVerifiers()

	for _, v := range verifiers {
		if v.Execution.ImplementationID == "" {
			t.Errorf("verifier %q has no ImplementationID", v.Name)
		}
	}
}

// TestAllVerifiers_HaveValidExecutionKind verifies all verifiers have valid execution kinds.
func TestAllVerifiers_HaveValidExecutionKind(t *testing.T) {
	verifiers := AllVerifiers()

	for _, v := range verifiers {
		if v.Execution.Kind != ExecutionInProcess && v.Execution.Kind != ExecutionChild {
			t.Errorf("verifier %q has invalid execution kind: %q", v.Name, v.Execution.Kind)
		}
	}
}
