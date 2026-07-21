// Package gate provides tests for factorize cache classification from verifier metadata.
package gate

import (
	"encoding/json"
	"sort"
	"testing"
)

const canonicalVerifierCount = 16

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

// TestCacheSemantics_DupcodeUsesNACache verifies dupcode verifiers use NA cache semantics.
func TestCacheSemantics_DupcodeUsesNACache(t *testing.T) {
	verifiers := AllVerifiers()

	for _, v := range verifiers {
		if v.Name == "dupcode" || v.Name == "dupcode-baseline" {
			if v.Cache.GoBuildCache != CacheNotApplicable {
				t.Errorf("verifier %q should have GoBuildCache=not-applicable, got %s", v.Name, v.Cache.GoBuildCache)
			}
			if v.Cache.GoTestResultCache != CacheModeNA {
				t.Errorf("verifier %q should have GoTestResultCache=not-applicable, got %s", v.Name, v.Cache.GoTestResultCache)
			}
		}
	}
}

// TestCacheSemantics_StaticBinaryUsesBuildCache verifies static-binary uses build cache.
func TestCacheSemantics_StaticBinaryUsesBuildCache(t *testing.T) {
	verifiers := AllVerifiers()

	for _, v := range verifiers {
		if v.Name == "static-binary" {
			if v.Cache.GoBuildCache != CacheRelevant {
				t.Errorf("verifier %q should have GoBuildCache=relevant, got %s", v.Name, v.Cache.GoBuildCache)
			}
		}
	}
}

// TestCacheSemantics_AllInProcess verifies all verifiers are in-process.
func TestCacheSemantics_AllInProcess(t *testing.T) {
	verifiers := AllVerifiers()

	for _, v := range verifiers {
		if v.Execution.Kind != ExecutionInProcess {
			t.Errorf("verifier %q should be ExecutionInProcess, got %s", v.Name, v.Execution.Kind)
		}
	}
}

// TestCacheSemantics_JsonSerialization verifies JSON serialization uses schema field names.
func TestCacheSemantics_JsonSerialization(t *testing.T) {
	cache := CacheSemantics{
		GoBuildCache:      CacheRelevant,
		GoTestResultCache: CacheModeDisabled,
	}

	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("failed to marshal CacheSemantics: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if got, want := parsed["go_build_cache"], "relevant"; got != want {
		t.Errorf("go_build_cache = %q, want %q", got, want)
	}
	if got, want := parsed["go_test_result_cache"], "disabled"; got != want {
		t.Errorf("go_test_result_cache = %q, want %q", got, want)
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
		"llm-friendly": true, "long-test-policy": true, "static-binary": true,
		"tooling-boundaries": true,
	}
	for _, name := range names {
		if !expectedSet[name] {
			t.Errorf("unexpected verifier: %q", name)
		}
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
