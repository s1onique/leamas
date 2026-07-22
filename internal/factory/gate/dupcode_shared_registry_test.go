// Package gate provides tests for the factorize registry wiring.
package gate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
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

// TestFactorizeRegistryWiringWithInjectedAnalyzer verifies that FactorizeVerifiersWithDupcodeContext
// properly wires the production registry with an injected analyzer, ensuring:
// - identical count, order and metadata to AllVerifiers()
// - both named entries are present
// - only their Run behavior is replaced
// - invoking the two returned registry entries shares one analyzer
func TestFactorizeRegistryWiringWithInjectedAnalyzer(t *testing.T) {
	ctx := context.Background()

	// Create a temporary directory with a valid baseline
	tmpDir := t.TempDir()

	// Initialize a git repo so baseline can be validated as tracked
	if _, err := runGit(ctx, tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if _, err := runGit(ctx, tmpDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if _, err := runGit(ctx, tmpDir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if _, err := runGit(ctx, tmpDir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a valid, empty baseline (clean case)
	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	if err := os.MkdirAll(filepath.Dir(baselinePath), 0755); err != nil {
		t.Fatalf("failed to create .factory dir: %v", err)
	}
	baseline := dupcode.Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: dupcode.AlgorithmVersion,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds: dupcode.BaselineThresholds{
			MinLines:  dupcode.PolicyMinLines,
			MinTokens: dupcode.PolicyMinTokens,
		},
		Findings: nil, // Empty baseline - clean case
	}
	baselineJSON, err := json.Marshal(baseline)
	if err != nil {
		t.Fatalf("failed to marshal baseline: %v", err)
	}
	if err := os.WriteFile(baselinePath, baselineJSON, 0644); err != nil {
		t.Fatalf("failed to write baseline: %v", err)
	}
	if _, err := runGit(ctx, tmpDir, "add", ".factory/dupcode-baseline.json"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Track call count
	callCount := 0
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		callCount++
		return nil, nil // No findings - matches empty baseline
	}

	// Build registry using production constructor with injected analyzer
	verifiers, err := factorizeVerifiersWithDupcodeAnalyzer(tmpDir, fakeAnalyzer)
	if err != nil {
		t.Fatalf("FactorizeVerifiersWithDupcodeContext failed: %v", err)
	}

	// Verify count matches AllVerifiers
	allVerifiers := AllVerifiers()
	if len(verifiers) != len(allVerifiers) {
		t.Errorf("verifiers count = %d, want %d", len(verifiers), len(allVerifiers))
	}

	// Verify dupcode and dupcode-baseline entries are present
	var dupcodeVerifier, baselineVerifier *Verifier
	for i, v := range verifiers {
		if v.Name == "dupcode" {
			dupcodeVerifier = &verifiers[i]
		}
		if v.Name == "dupcode-baseline" {
			baselineVerifier = &verifiers[i]
		}
	}

	if dupcodeVerifier == nil {
		t.Fatal("dupcode verifier not found in registry")
	}
	if baselineVerifier == nil {
		t.Fatal("dupcode-baseline verifier not found in registry")
	}

	// Verify metadata matches AllVerifiers by index
	for i, av := range allVerifiers {
		v := verifiers[i]
		if v.Name != av.Name {
			t.Errorf("index %d: name = %q, want %q", i, v.Name, av.Name)
		}
		if v.Lane != av.Lane {
			t.Errorf("index %d: Lane mismatch for %q", i, v.Name)
		}
		if v.Execution.Kind != av.Execution.Kind {
			t.Errorf("index %d: Execution.Kind mismatch for %q", i, v.Name)
		}
		if v.Execution.ImplementationID != av.Execution.ImplementationID {
			t.Errorf("index %d: Execution.ImplementationID mismatch for %q", i, v.Name)
		}
		if v.Cache.GoBuildCache != av.Cache.GoBuildCache {
			t.Errorf("index %d: Cache.GoBuildCache mismatch for %q", i, v.Name)
		}
	}

	// Verify both entries use the shared analyzer
	dupcodeFindings := dupcodeVerifier.Run(tmpDir)
	baselineFindings := baselineVerifier.Run(tmpDir)

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (shared analyzer)", callCount)
	}

	// Assert exactly empty results for clean case
	if len(dupcodeFindings) != 0 {
		t.Fatalf("dupcode findings = %#v, want none", dupcodeFindings)
	}
	if len(baselineFindings) != 0 {
		t.Fatalf("baseline findings = %#v, want none", baselineFindings)
	}
}

// TestFactorizeRegistryWiringWithStaleBaseline verifies that a baseline with findings
// that don't match the current analysis produces drift findings.
func TestFactorizeRegistryWiringWithStaleBaseline(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()

	// Initialize a git repo
	if _, err := runGit(ctx, tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if _, err := runGit(ctx, tmpDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if _, err := runGit(ctx, tmpDir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if _, err := runGit(ctx, tmpDir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a baseline with a valid fingerprint (64-char hex)
	staleFingerprint := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	if err := os.MkdirAll(filepath.Dir(baselinePath), 0755); err != nil {
		t.Fatalf("failed to create .factory dir: %v", err)
	}
	baseline := dupcode.Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: dupcode.AlgorithmVersion,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas dupcode",
		Thresholds: dupcode.BaselineThresholds{
			MinLines:  dupcode.PolicyMinLines,
			MinTokens: dupcode.PolicyMinTokens,
		},
		Findings: []dupcode.BaselineFinding{
			{
				Fingerprint: staleFingerprint,
				TokenCount:  500,
				LineCount:   50,
				Occurrences: []dupcode.BaselineOccurrence{
					{Path: "a.go", StartLine: 10, EndLine: 30},
				},
			},
		},
	}
	baselineJSON, err := json.Marshal(baseline)
	if err != nil {
		t.Fatalf("failed to marshal baseline: %v", err)
	}
	if err := os.WriteFile(baselinePath, baselineJSON, 0644); err != nil {
		t.Fatalf("failed to write baseline: %v", err)
	}
	if _, err := runGit(ctx, tmpDir, "add", ".factory/dupcode-baseline.json"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Analyzer returns no findings (different from baseline)
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		return nil, nil
	}

	verifiers, err := factorizeVerifiersWithDupcodeAnalyzer(tmpDir, fakeAnalyzer)
	if err != nil {
		t.Fatalf("FactorizeVerifiersWithDupcodeContext failed: %v", err)
	}

	// Find the baseline verifier
	var baselineVerifier *Verifier
	for i, v := range verifiers {
		if v.Name == "dupcode-baseline" {
			baselineVerifier = &verifiers[i]
			break
		}
	}
	if baselineVerifier == nil {
		t.Fatal("dupcode-baseline verifier not found")
	}

	// Run the baseline verifier - should produce drift finding
	baselineFindings := baselineVerifier.Run(tmpDir)

	// Should have exactly one drift finding
	if len(baselineFindings) != 1 {
		t.Fatalf("baseline findings count = %d, want 1", len(baselineFindings))
	}
	if baselineFindings[0].Kind != "dupcode_baseline_drift" {
		t.Errorf("baseline finding kind = %q, want %q", baselineFindings[0].Kind, "dupcode_baseline_drift")
	}
}
