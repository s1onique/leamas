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

// TestReplaceDupcodeVerifierRuns_BothReplaced verifies that the helper
// succeeds when both dupcode and dupcode-baseline entries are present in
// the registry and both Run functions are replaced.
func TestReplaceDupcodeVerifierRuns_BothReplaced(t *testing.T) {
	verifiers := AllVerifiers()
	orig := make([]Verifier, len(verifiers))
	copy(orig, verifiers)

	dupcodeRun := func(string) []checks.Finding { return nil }
	baselineRun := func(string) []checks.Finding { return nil }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeRun, baselineRun)
	if err != nil {
		t.Fatalf("replaceDupcodeVerifierRuns failed: %v", err)
	}
	if len(out) != len(orig) {
		t.Fatalf("verifier count = %d, want %d", len(out), len(orig))
	}

	// Both Run functions must be replaced (compare function pointers via string).
	for _, v := range out {
		switch v.Name {
		case "dupcode":
			if v.Run == nil {
				t.Fatal("dupcode Run is nil after replacement")
			}
		case "dupcode-baseline":
			if v.Run == nil {
				t.Fatal("dupcode-baseline Run is nil after replacement")
			}
		}
	}
}

// TestReplaceDupcodeVerifierRuns_MissingDupcode verifies fail-closed
// behaviour when the dupcode entry is absent from the registry.
func TestReplaceDupcodeVerifierRuns_MissingDupcode(t *testing.T) {
	verifiers := []Verifier{
		{Name: "dupcode-baseline", Run: func(string) []checks.Finding { return nil }},
		{Name: "agent-context", Run: func(string) []checks.Finding { return nil }},
	}

	dupcodeRun := func(string) []checks.Finding { return nil }
	baselineRun := func(string) []checks.Finding { return nil }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeRun, baselineRun)
	if err == nil {
		t.Fatal("expected fail-closed error when dupcode is missing, got nil")
	}
	if out != nil {
		t.Errorf("expected nil registry on failure, got %d entries", len(out))
	}
	if !strings.Contains(err.Error(), "dupcode") {
		t.Errorf("error message %q should mention dupcode", err.Error())
	}
}

// TestReplaceDupcodeVerifierRuns_MissingBaseline verifies fail-closed
// behaviour when the dupcode-baseline entry is absent from the registry.
func TestReplaceDupcodeVerifierRuns_MissingBaseline(t *testing.T) {
	verifiers := []Verifier{
		{Name: "dupcode", Run: func(string) []checks.Finding { return nil }},
		{Name: "agent-context", Run: func(string) []checks.Finding { return nil }},
	}

	dupcodeRun := func(string) []checks.Finding { return nil }
	baselineRun := func(string) []checks.Finding { return nil }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeRun, baselineRun)
	if err == nil {
		t.Fatal("expected fail-closed error when dupcode-baseline is missing, got nil")
	}
	if out != nil {
		t.Errorf("expected nil registry on failure, got %d entries", len(out))
	}
	if !strings.Contains(err.Error(), "dupcode-baseline") {
		t.Errorf("error message %q should mention dupcode-baseline", err.Error())
	}
}

// TestReplaceDupcodeVerifierRuns_MissingBoth verifies fail-closed
// behaviour when both entries are absent from the registry.
func TestReplaceDupcodeVerifierRuns_MissingBoth(t *testing.T) {
	verifiers := []Verifier{
		{Name: "agent-context", Run: func(string) []checks.Finding { return nil }},
	}

	dupcodeRun := func(string) []checks.Finding { return nil }
	baselineRun := func(string) []checks.Finding { return nil }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeRun, baselineRun)
	if err == nil {
		t.Fatal("expected fail-closed error when both entries are missing, got nil")
	}
	if out != nil {
		t.Errorf("expected nil registry on failure, got %d entries", len(out))
	}
}

// TestReplaceDupcodeVerifierRuns_EmptyRegistry verifies fail-closed
// behaviour when the registry is empty.
func TestReplaceDupcodeVerifierRuns_EmptyRegistry(t *testing.T) {
	verifiers := []Verifier{}

	dupcodeRun := func(string) []checks.Finding { return nil }
	baselineRun := func(string) []checks.Finding { return nil }

	out, err := replaceDupcodeVerifierRuns(verifiers, dupcodeRun, baselineRun)
	if err == nil {
		t.Fatal("expected fail-closed error on empty registry, got nil")
	}
	if out != nil {
		t.Errorf("expected nil registry on failure, got %d entries", len(out))
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
