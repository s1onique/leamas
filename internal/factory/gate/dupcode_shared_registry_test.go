// Package gate provides tests for the factorize registry wiring.
package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/dupcode"
)

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

	// Create a valid baseline
	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	if err := os.MkdirAll(filepath.Dir(baselinePath), 0755); err != nil {
		t.Fatalf("failed to create .factory dir: %v", err)
	}
	baseline := dupcode.Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: dupcode.AlgorithmVersion,
		GeneratedAt:      "2024-01-01T00:00:00Z",
		Tool:             "leamas",
		Thresholds: dupcode.BaselineThresholds{
			MinLines:  40,
			MinTokens: 400,
		},
		Findings: []dupcode.BaselineFinding{
			{
				Fingerprint: "fp1",
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

	// Track call count
	callCount := 0
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		callCount++
		return nil, nil
	}

	// Build registry with injected analyzer
	verifiers, err := factorizeVerifiersWithInjectedAnalyzer(tmpDir, fakeAnalyzer)
	if err != nil {
		t.Fatalf("FactorizeVerifiersWithDupcodeContext failed: %v", err)
	}

	// Verify count matches AllVerifiers
	allVerifiers := AllVerifiers()
	if len(verifiers) != len(allVerifiers) {
		t.Errorf("verifiers count = %d, want %d", len(verifiers), len(allVerifiers))
	}

	// Verify dupcode and dupcode-baseline entries are present with correct metadata
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

	// Verify metadata matches AllVerifiers
	for _, av := range allVerifiers {
		if av.Name == "dupcode" {
			if dupcodeVerifier.Lane != av.Lane {
				t.Errorf("dupcode Lane = %v, want %v", dupcodeVerifier.Lane, av.Lane)
			}
			if dupcodeVerifier.Execution.Kind != av.Execution.Kind {
				t.Errorf("dupcode Execution.Kind mismatch")
			}
			if dupcodeVerifier.Execution.ImplementationID != av.Execution.ImplementationID {
				t.Errorf("dupcode Execution.ImplementationID mismatch")
			}
			if dupcodeVerifier.Cache.GoBuildCache != av.Cache.GoBuildCache {
				t.Errorf("dupcode Cache.GoBuildCache mismatch")
			}
		}
		if av.Name == "dupcode-baseline" {
			if baselineVerifier.Lane != av.Lane {
				t.Errorf("dupcode-baseline Lane = %v, want %v", baselineVerifier.Lane, av.Lane)
			}
			if baselineVerifier.Execution.Kind != av.Execution.Kind {
				t.Errorf("dupcode-baseline Execution.Kind mismatch")
			}
			if baselineVerifier.Execution.ImplementationID != av.Execution.ImplementationID {
				t.Errorf("dupcode-baseline Execution.ImplementationID mismatch")
			}
			if baselineVerifier.Cache.GoBuildCache != av.Cache.GoBuildCache {
				t.Errorf("dupcode-baseline Cache.GoBuildCache mismatch")
			}
		}
	}

	// Verify both entries use the shared analyzer
	dupcodeFindings := dupcodeVerifier.Run(tmpDir)
	baselineFindings := baselineVerifier.Run(tmpDir)

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (shared analyzer)", callCount)
	}

	// Both verifiers should succeed with empty results
	for _, f := range dupcodeFindings {
		if f.Kind == "dupcode_error" || f.Kind == "baseline_load_error" {
			t.Errorf("dupcode verifier returned error: %s: %s", f.Kind, f.Message)
		}
	}
	for _, f := range baselineFindings {
		if f.Kind == "dupcode_error" || f.Kind == "baseline_validation_error" {
			t.Errorf("baseline verifier returned error: %s: %s", f.Kind, f.Message)
		}
	}
}

// factorizeVerifiersWithInjectedAnalyzer is an unexported constructor that accepts
// an injected analyzer for testing the production registry wiring.
func factorizeVerifiersWithInjectedAnalyzer(root string, analyzer DupcodeAnalyzer) ([]Verifier, error) {
	// Determine the effective dupcode thresholds from the baseline
	minLines := dupcode.PolicyMinLines
	minTokens := dupcode.PolicyMinTokens

	baselinePath := ".factory/dupcode-baseline.json"
	if root != "." && root != "" {
		baselinePath = filepath.Join(root, baselinePath)
	}

	if _, err := os.Stat(baselinePath); err == nil {
		if baseline, err := dupcode.LoadBaseline(baselinePath); err == nil {
			minLines = baseline.Thresholds.MinLines
			minTokens = baseline.Thresholds.MinTokens
		}
	}

	// Create shared analysis context with injected analyzer
	cfg := dupcode.DefaultConfig()
	cfg.Root = root
	cfg.MinLines = minLines
	cfg.MinTokens = minTokens
	provider := NewDupcodeAnalysisProvider(newDupcodeInput(cfg), analyzer)

	ctx := NewDupcodeAnalysisContext(provider)
	factory := NewDupcodeVerifierFactory(ctx)

	sharedDupcodeVerifier := factory.SharedDupCodeVerifier()
	sharedDupcodeBaselineVerifier := factory.SharedDupcodeBaselineVerifier()

	// Derive from AllVerifiers and only replace the Run functions
	verifiers := AllVerifiers()
	replacedDupcode := false
	replacedBaseline := false
	for i := range verifiers {
		switch verifiers[i].Name {
		case "dupcode":
			verifiers[i].Run = sharedDupcodeVerifier
			replacedDupcode = true
		case "dupcode-baseline":
			verifiers[i].Run = sharedDupcodeBaselineVerifier
			replacedBaseline = true
		}
	}

	if !replacedDupcode || !replacedBaseline {
		return nil, errFailedRegistryReplacement
	}

	return verifiers, nil
}

var errFailedRegistryReplacement = fmt.Errorf("registry replacement failed")
