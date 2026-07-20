// Package gate provides tests for factorize cache classification from verifier metadata.
package gate

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
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

// TestValidateVerifier_EmptyName verifies validation fails for empty name.
func TestValidateVerifier_EmptyName(t *testing.T) {
	v := Verifier{Name: "", Run: func(string) []checks.Finding { return nil }}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

// TestValidateVerifier_DuplicateEnvKey verifies validation fails for duplicate env keys.
func TestValidateVerifier_DuplicateEnvKey(t *testing.T) {
	v := Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Execution: ExecutionDefinition{
			Kind:             ExecutionInProcess,
			ImplementationID: "test",
			EnvVars:          []string{"GOFLAGS", "GOFLAGS"},
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for duplicate env key")
	}
}

// TestValidateVerifier_MalformedEnvKeyWithEquals verifies validation fails for env key with =.
func TestValidateVerifier_MalformedEnvKeyWithEquals(t *testing.T) {
	v := Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Execution: ExecutionDefinition{
			Kind:             ExecutionInProcess,
			ImplementationID: "test",
			EnvVars:          []string{"GOFLAGS=value"},
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for malformed env key with =")
	}
}

// TestValidateVerifier_MalformedEnvKeyWithWhitespace verifies validation fails for env key with whitespace.
func TestValidateVerifier_MalformedEnvKeyWithWhitespace(t *testing.T) {
	v := Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Execution: ExecutionDefinition{
			Kind:             ExecutionInProcess,
			ImplementationID: "test",
			EnvVars:          []string{" GOFLAGS"},
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for env key with whitespace")
	}
}

// TestValidateVerifier_MalformedEnvKeyInvalidName verifies validation fails for invalid env key name.
func TestValidateVerifier_MalformedEnvKeyInvalidName(t *testing.T) {
	v := Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Execution: ExecutionDefinition{
			Kind:             ExecutionInProcess,
			ImplementationID: "test",
			EnvVars:          []string{"123INVALID"},
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for invalid env key name")
	}
}

// TestValidateVerifier_InvalidGoBuildCache verifies validation fails for invalid GoBuildCache.
func TestValidateVerifier_InvalidGoBuildCache(t *testing.T) {
	v := Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Execution: ExecutionDefinition{
			Kind:             ExecutionInProcess,
			ImplementationID: "test",
		},
		Cache: CacheSemantics{
			GoBuildCache:      "invalid",
			GoTestResultCache: CacheModeNA,
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for invalid GoBuildCache")
	}
}

// TestValidateVerifier_InvalidGoTestResultCache verifies validation fails for invalid GoTestResultCache.
func TestValidateVerifier_InvalidGoTestResultCache(t *testing.T) {
	v := Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Execution: ExecutionDefinition{
			Kind:             ExecutionInProcess,
			ImplementationID: "test",
		},
		Cache: CacheSemantics{
			GoBuildCache:      CacheNotApplicable,
			GoTestResultCache: "invalid",
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for invalid GoTestResultCache")
	}
}

// TestValidateVerifier_ValidEmptyEnvVars verifies validation passes for empty env vars.
func TestValidateVerifier_ValidEmptyEnvVars(t *testing.T) {
	v := Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Execution: ExecutionDefinition{
			Kind:             ExecutionInProcess,
			ImplementationID: "test",
			EnvVars:          []string{},
		},
		Cache: CacheSemantics{
			GoBuildCache:      CacheNotApplicable,
			GoTestResultCache: CacheModeNA,
		},
	}
	err := ValidateVerifier(v)
	if err != nil {
		t.Errorf("expected no error for empty env vars: %v", err)
	}
}

// TestValidateVerifier_NilRun verifies validation fails for nil Run.
func TestValidateVerifier_NilRun(t *testing.T) {
	v := Verifier{Name: "test", Run: nil}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for nil Run")
	}
}

// TestValidateVerifier_InvalidKind verifies validation fails for invalid kind.
func TestValidateVerifier_InvalidKind(t *testing.T) {
	v := Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Execution: ExecutionDefinition{
			Kind:             "invalid",
			ImplementationID: "test",
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for invalid kind")
	}
}

// TestValidateVerifier_EmptyImplID verifies validation fails for empty ImplementationID.
func TestValidateVerifier_EmptyImplID(t *testing.T) {
	v := Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Execution: ExecutionDefinition{
			Kind:             ExecutionInProcess,
			ImplementationID: "",
		},
	}
	err := ValidateVerifier(v)
	if err == nil {
		t.Error("expected error for empty ImplementationID")
	}
}

// TestValidateVerifiers_NoDuplicates verifies validation fails for duplicate names.
func TestValidateVerifiers_NoDuplicates(t *testing.T) {
	v := Verifier{
		Name: "test",
		Run:  func(string) []checks.Finding { return nil },
		Execution: ExecutionDefinition{
			Kind:             ExecutionInProcess,
			ImplementationID: "test",
		},
	}
	verifiers := []Verifier{v, v}
	err := ValidateVerifiers(verifiers)
	if err == nil {
		t.Error("expected error for duplicate names")
	}
}

// TestValidateVerifiers_AllCanonical verifies all canonical verifiers pass validation.
func TestValidateVerifiers_AllCanonical(t *testing.T) {
	verifiers := AllVerifiers()
	err := ValidateVerifiers(verifiers)
	if err != nil {
		t.Errorf("canonical verifiers should pass validation: %v", err)
	}
}
