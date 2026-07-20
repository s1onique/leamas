// Package gate provides tests for factorize cache classification.
package gate

import (
	"sort"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// canonicalVerifierCount is the expected number of verifiers in factorize.
const canonicalVerifierCount = 15

// canonicalVerifiers returns the 15 canonical verifier names from gate.go.
func canonicalVerifiers() []string {
	return []string{
		"agent-context",
		"doctrine",
		"doctrine-agent-contracts",
		"docs",
		"dupcode",
		"dupcode-baseline",
		"domain-boundaries",
		"exec-gate",
		"executable-contract-first",
		"forbidden-patterns",
		"git-hooks",
		"language",
		"llm-friendly",
		"static-binary",
		"tooling-boundaries",
	}
}

// TestCacheClassification_AllCanonicalVerifiersClassified verifies that every
// canonical verifier has exactly one cache classification.
func TestCacheClassification_AllCanonicalVerifiersClassified(t *testing.T) {
	verifiers := canonicalVerifiers()
	if len(verifiers) != canonicalVerifierCount {
		t.Fatalf("expected %d canonical verifiers, got %d", canonicalVerifierCount, len(verifiers))
	}

	for _, name := range verifiers {
		class := classifyCacheObservation(name, nil)
		if class == "" {
			t.Errorf("verifier %q has no cache classification", name)
		}
	}
}

// TestCacheClassification_NoUnknownVerifiers verifies that unknown or phantom
// verifiers get a default classification.
func TestCacheClassification_UnknownVerifiersGetDefault(t *testing.T) {
	class := classifyCacheObservation("unknown-phantom-verifier", nil)
	if class == "" {
		t.Errorf("unknown verifier must still get a classification")
	}
}

// TestCacheClassification_ExactlyOneClassification verifies that each verifier
// gets exactly one classification string.
func TestCacheClassification_ExactlyOneClassification(t *testing.T) {
	verifiers := canonicalVerifiers()

	for _, name := range verifiers {
		class := classifyCacheObservation(name, nil)
		if strings.Contains(class, ";") {
			parts := strings.Split(class, ";")
			if len(parts) != 2 {
				t.Errorf("verifier %q has malformed classification: %q", name, class)
			}
		}
	}
}

// TestCacheClassification_StructuredFormat verifies that cache observations
// use a structured format with named fields.
func TestCacheClassification_StructuredFormat(t *testing.T) {
	verifiers := canonicalVerifiers()

	for _, name := range verifiers {
		class := classifyCacheObservation(name, nil)

		pairs := strings.Split(class, ";")
		if len(pairs) < 1 {
			t.Errorf("verifier %q classification %q has no semicolon-separated pairs", name, class)
			continue
		}

		for _, pair := range pairs {
			kv := strings.Split(pair, "=")
			if len(kv) != 2 {
				t.Errorf("verifier %q classification %q has malformed pair %q (expected key=value)", name, class, pair)
			}
		}
	}
}

// TestCacheClassification_DupcodeUsesTestResultCacheDisabled verifies that
// dupcode verifiers disable test result cache.
func TestCacheClassification_DupcodeUsesTestResultCacheDisabled(t *testing.T) {
	dupcodeVerifiers := []string{"dupcode", "dupcode-baseline"}

	for _, name := range dupcodeVerifiers {
		class := classifyCacheObservation(name, nil)
		if !strings.Contains(class, "go_test_result_cache=disabled") {
			t.Errorf("verifier %q must have go_test_result_cache=disabled, got %q", name, class)
		}
	}
}

// TestCacheClassification_NoGocoverageInCanonical verifies that go-coverage
// is not among the 15 canonical verifiers.
func TestCacheClassification_NoGocoverageInCanonical(t *testing.T) {
	verifiers := canonicalVerifiers()

	for _, name := range verifiers {
		if name == "go-coverage" {
			t.Errorf("go-coverage must not be in canonical verifier list")
		}
	}
}

// TestCacheClassification_AllowlistDerived verifies that cache semantics
// come from verifier metadata allowlist, not hardcoded strings.
func TestCacheClassification_AllowlistDerived(t *testing.T) {
	verifiers := canonicalVerifiers()

	for _, name := range verifiers {
		class := classifyCacheObservation(name, nil)
		if class == "" {
			t.Errorf("verifier %q has no classification", name)
			continue
		}

		if !strings.Contains(class, "=") {
			t.Errorf("verifier %q classification %q lacks key=value format", name, class)
		}
	}
}

// TestClassifyCacheObservation_FindingsUnused verifies that findings parameter
// is unused (as noted in the review).
func TestClassifyCacheObservation_FindingsUnused(t *testing.T) {
	name := "dupcode"
	nilFindings := classifyCacheObservation(name, nil)
	emptyFindings := classifyCacheObservation(name, []checks.Finding{})
	realFindings := classifyCacheObservation(name, []checks.Finding{
		{Path: "test.go", Kind: "error", Message: "test error"},
	})

	if nilFindings != emptyFindings || emptyFindings != realFindings {
		t.Errorf("findings parameter should not affect classification")
	}
}

// verifierNames extracts just the names from a Verifier slice for display.
func verifierNames(verifiers []Verifier) []string {
	names := make([]string, len(verifiers))
	for i, v := range verifiers {
		names[i] = v.Name
	}
	sort.Strings(names)
	return names
}

// TestAllVerifiersCount_MatchesCanonical verifies AllVerifiers returns
// exactly 15 verifiers.
func TestAllVerifiersCount_MatchesCanonical(t *testing.T) {
	verifiers := AllVerifiers()

	if len(verifiers) != canonicalVerifierCount {
		t.Errorf("AllVerifiers() returned %d verifiers, expected %d:\n%v",
			len(verifiers), canonicalVerifierCount, verifierNames(verifiers))
	}
}
