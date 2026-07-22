package gate

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// sentinelFinding produces a sentinel Finding that the test can
// recognise by Kind value. Sentinel functions are used in place of
// function-pointer comparison so that tests verify actual replacement
// behavior rather than just non-nil pointers.
func sentinelFinding(kind string) []checks.Finding {
	return []checks.Finding{{Path: "/sentinel", Kind: kind, Message: "sentinel", Severity: checks.SeverityError}}
}

func findVerifierByName(verifiers []Verifier, name string) *Verifier {
	for i := range verifiers {
		if verifiers[i].Name == name {
			return &verifiers[i]
		}
	}
	return nil
}

// TestReplaceDupcodeVerifierRuns_BothReplaced verifies that the helper
// succeeds when both dupcode and dupcode-baseline entries are present in
// the registry and both Run functions are replaced. The test invokes
// each replaced Run function and confirms it returns the sentinel
// output, then verifies the caller's input slice still invokes the
// original Run functions.
func TestReplaceDupcodeVerifierRuns_BothReplaced(t *testing.T) {
	dupcodeOrig := func(string) []checks.Finding { return sentinelFinding("dupcode-original") }
	baselineOrig := func(string) []checks.Finding { return sentinelFinding("baseline-original") }

	verifiers := []Verifier{
		{Name: "agent-context", Run: func(string) []checks.Finding { return nil }},
		{Name: "dupcode", Run: dupcodeOrig},
		{Name: "dupcode-baseline", Run: baselineOrig},
	}

	dupcodeNew := func(string) []checks.Finding { return sentinelFinding("dupcode-replaced") }
	baselineNew := func(string) []checks.Finding { return sentinelFinding("baseline-replaced") }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeNew, baselineNew)
	if err != nil {
		t.Fatalf("replaceDupcodeVerifierRuns failed: %v", err)
	}
	if len(out) != len(verifiers) {
		t.Fatalf("verifier count = %d, want %d", len(out), len(verifiers))
	}

	// Replaced Run functions must invoke the new sentinels.
	if findVerifierByName(out, "dupcode").Run("") == nil {
		t.Fatal("dupcode Run returned nil after replacement")
	}
	if got := findVerifierByName(out, "dupcode").Run(""); len(got) == 0 || got[0].Kind != "dupcode-replaced" {
		t.Errorf("dupcode Run did not return sentinel; got %#v", got)
	}
	if got := findVerifierByName(out, "dupcode-baseline").Run(""); len(got) == 0 || got[0].Kind != "baseline-replaced" {
		t.Errorf("dupcode-baseline Run did not return sentinel; got %#v", got)
	}

	// Caller's input slice must still invoke the original sentinels.
	if got := verifiers[1].Run(""); len(got) == 0 || got[0].Kind != "dupcode-original" {
		t.Errorf("caller dupcode Run was mutated; got %#v", got)
	}
	if got := verifiers[2].Run(""); len(got) == 0 || got[0].Kind != "baseline-original" {
		t.Errorf("caller dupcode-baseline Run was mutated; got %#v", got)
	}
}

// TestReplaceDupcodeVerifierRuns_MissingDupcode verifies fail-closed
// behaviour when the dupcode entry is absent from the registry. The
// caller's input slice must remain unchanged.
func TestReplaceDupcodeVerifierRuns_MissingDupcode(t *testing.T) {
	baselineOrig := func(string) []checks.Finding { return sentinelFinding("baseline-original") }
	verifiers := []Verifier{
		{Name: "dupcode-baseline", Run: baselineOrig},
		{Name: "agent-context", Run: func(string) []checks.Finding { return nil }},
	}

	dupcodeNew := func(string) []checks.Finding { return sentinelFinding("dupcode-replaced") }
	baselineNew := func(string) []checks.Finding { return sentinelFinding("baseline-replaced") }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeNew, baselineNew)
	if err == nil {
		t.Fatal("expected fail-closed error when dupcode is missing, got nil")
	}
	if out != nil {
		t.Errorf("expected nil registry on failure, got %d entries", len(out))
	}
	if !strings.Contains(err.Error(), "dupcode") {
		t.Errorf("error message %q should mention dupcode", err.Error())
	}

	// Caller's input slice must remain unchanged.
	if got := verifiers[0].Run(""); len(got) == 0 || got[0].Kind != "baseline-original" {
		t.Errorf("caller dupcode-baseline Run was mutated on failure; got %#v", got)
	}
}

// TestReplaceDupcodeVerifierRuns_MissingBaseline verifies fail-closed
// behaviour when the dupcode-baseline entry is absent from the registry.
// The caller's input slice must remain unchanged.
func TestReplaceDupcodeVerifierRuns_MissingBaseline(t *testing.T) {
	dupcodeOrig := func(string) []checks.Finding { return sentinelFinding("dupcode-original") }
	verifiers := []Verifier{
		{Name: "dupcode", Run: dupcodeOrig},
		{Name: "agent-context", Run: func(string) []checks.Finding { return nil }},
	}

	dupcodeNew := func(string) []checks.Finding { return sentinelFinding("dupcode-replaced") }
	baselineNew := func(string) []checks.Finding { return sentinelFinding("baseline-replaced") }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeNew, baselineNew)
	if err == nil {
		t.Fatal("expected fail-closed error when dupcode-baseline is missing, got nil")
	}
	if out != nil {
		t.Errorf("expected nil registry on failure, got %d entries", len(out))
	}
	if !strings.Contains(err.Error(), "dupcode-baseline") {
		t.Errorf("error message %q should mention dupcode-baseline", err.Error())
	}

	// Caller's input slice must remain unchanged.
	if got := verifiers[0].Run(""); len(got) == 0 || got[0].Kind != "dupcode-original" {
		t.Errorf("caller dupcode Run was mutated on failure; got %#v", got)
	}
}

// TestReplaceDupcodeVerifierRuns_MissingBoth verifies fail-closed
// behaviour when both entries are absent from the registry. The
// caller's input slice must remain unchanged.
func TestReplaceDupcodeVerifierRuns_MissingBoth(t *testing.T) {
	agentOrig := func(string) []checks.Finding { return sentinelFinding("agent-original") }
	verifiers := []Verifier{
		{Name: "agent-context", Run: agentOrig},
	}

	dupcodeNew := func(string) []checks.Finding { return sentinelFinding("dupcode-replaced") }
	baselineNew := func(string) []checks.Finding { return sentinelFinding("baseline-replaced") }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeNew, baselineNew)
	if err == nil {
		t.Fatal("expected fail-closed error when both entries are missing, got nil")
	}
	if out != nil {
		t.Errorf("expected nil registry on failure, got %d entries", len(out))
	}

	// Caller's input slice must remain unchanged.
	if got := verifiers[0].Run(""); len(got) == 0 || got[0].Kind != "agent-original" {
		t.Errorf("caller agent-context Run was mutated on failure; got %#v", got)
	}
}

// TestReplaceDupcodeVerifierRuns_EmptyRegistry verifies fail-closed
// behaviour when the registry is empty. The function must not panic
// and must not mutate the (empty) input slice.
func TestReplaceDupcodeVerifierRuns_EmptyRegistry(t *testing.T) {
	verifiers := []Verifier{}

	dupcodeNew := func(string) []checks.Finding { return sentinelFinding("dupcode-replaced") }
	baselineNew := func(string) []checks.Finding { return sentinelFinding("baseline-replaced") }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeNew, baselineNew)
	if err == nil {
		t.Fatal("expected fail-closed error on empty registry, got nil")
	}
	if out != nil {
		t.Errorf("expected nil registry on failure, got %d entries", len(out))
	}
	if len(verifiers) != 0 {
		t.Errorf("input slice length changed: got %d, want 0", len(verifiers))
	}
}

// TestReplaceDupcodeVerifierRuns_InputUnchangedOnSuccess verifies
// that on success the caller's input slice still invokes the original
// Run functions (the helper returns a copy).
func TestReplaceDupcodeVerifierRuns_InputUnchangedOnSuccess(t *testing.T) {
	dupcodeOrig := func(string) []checks.Finding { return sentinelFinding("dupcode-original") }
	baselineOrig := func(string) []checks.Finding { return sentinelFinding("baseline-original") }

	verifiers := []Verifier{
		{Name: "agent-context", Run: func(string) []checks.Finding { return nil }},
		{Name: "dupcode", Run: dupcodeOrig},
		{Name: "dupcode-baseline", Run: baselineOrig},
	}

	dupcodeNew := func(string) []checks.Finding { return sentinelFinding("dupcode-replaced") }
	baselineNew := func(string) []checks.Finding { return sentinelFinding("baseline-replaced") }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeNew, baselineNew)
	if err != nil {
		t.Fatalf("replaceDupcodeVerifierRuns failed: %v", err)
	}

	// Caller's input slice must still invoke the original sentinels.
	if got := verifiers[1].Run(""); len(got) == 0 || got[0].Kind != "dupcode-original" {
		t.Errorf("caller dupcode Run was mutated on success; got %#v", got)
	}
	if got := verifiers[2].Run(""); len(got) == 0 || got[0].Kind != "baseline-original" {
		t.Errorf("caller dupcode-baseline Run was mutated on success; got %#v", got)
	}

	// Output must be a different slice (failure-atomic guarantee).
	if len(out) == len(verifiers) && &out[0] == &verifiers[0] {
		t.Error("output slice shares underlying array with input; replacement is not failure-atomic")
	}
}
