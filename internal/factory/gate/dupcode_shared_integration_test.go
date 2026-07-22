// Package gate provides integration tests for the dupcode shared analysis path.
package gate

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// TestSharedVerifiersExecuteOneAnalysis tests that calling both verifier closures
// results in exactly one analyzer execution.
func TestSharedVerifiersExecuteOneAnalysis(t *testing.T) {
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

	// Commit initial state so .factory can be added later
	if _, err := runGit(ctx, tmpDir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a counting analyzer
	callCount := 0
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		callCount++
		return nil, nil
	}

	// Create provider and factory
	// Use IgnoreGenerated=true and the actual root to match DefaultConfig() used by verifiers
	provider := NewDupcodeAnalysisProvider(testInput(tmpDir, 40, 400, nil, nil, true), fakeAnalyzer)
	analysisCtx := NewDupcodeAnalysisContext(provider)
	factory := NewDupcodeVerifierFactory(analysisCtx)
	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	if err := os.MkdirAll(filepath.Dir(baselinePath), 0755); err != nil {
		t.Fatalf("failed to create .factory dir: %v", err)
	}

	// Create a valid baseline using the actual struct
	// Use AlgorithmVersion 4 to match the current detector
	baseline := dupcode.Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: 4,
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

	// Add baseline to git
	if _, err := runGit(ctx, tmpDir, "add", ".factory/dupcode-baseline.json"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Get the verifier closures
	dupcodeVerifier := factory.SharedDupCodeVerifier()
	baselineVerifier := factory.SharedDupcodeBaselineVerifier()

	// Call baseline verifier first (it validates and performs the scan)
	baselineFindings := baselineVerifier(tmpDir)

	// Verify call count after baseline verifier
	if callCount != 1 {
		t.Errorf("callCount after baseline verifier = %d, want 1", callCount)
	}

	// Verify no errors from baseline verifier
	for _, f := range baselineFindings {
		if f.Kind == "baseline_validation_error" || f.Kind == "dupcode_error" {
			t.Errorf("baseline verifier returned error finding: %s: %s", f.Kind, f.Message)
		}
	}

	// Call dupcode verifier (should reuse the scan)
	dupcodeFindings := dupcodeVerifier(tmpDir)

	// Verify call count is still 1 (shared, not re-scanned)
	if callCount != 1 {
		t.Errorf("callCount after dupcode verifier = %d, want 1 (shared)", callCount)
	}

	// Verify no errors from dupcode verifier
	for _, f := range dupcodeFindings {
		if f.Kind == "baseline_load_error" || f.Kind == "dupcode_error" {
			t.Errorf("dupcode verifier returned error finding: %s: %s", f.Kind, f.Message)
		}
	}

	// Verify execution count
	if provider.Executions() != 1 {
		t.Errorf("provider.Executions() = %d, want 1", provider.Executions())
	}

	// Verify both consumers reached the provider
	if provider.result == nil {
		t.Fatal("provider has no successful result")
	}
	if got := provider.result.Consumers; got != 2 {
		t.Errorf("provider consumers = %d, want 2", got)
	}
}

// TestSharedVerifiersExecuteOneAnalysisInReverseOrder tests the same but with
// the verifiers called in reverse order.
func TestSharedVerifiersExecuteOneAnalysisInReverseOrder(t *testing.T) {
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

	// Commit initial state so .factory can be added later
	if _, err := runGit(ctx, tmpDir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a counting analyzer
	callCount := 0
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		callCount++
		return nil, nil
	}

	// Create provider and factory
	// Use IgnoreGenerated=true and the actual root to match DefaultConfig() used by verifiers
	provider := NewDupcodeAnalysisProvider(testInput(tmpDir, 40, 400, nil, nil, true), fakeAnalyzer)
	analysisCtx := NewDupcodeAnalysisContext(provider)
	factory := NewDupcodeVerifierFactory(analysisCtx)
	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	if err := os.MkdirAll(filepath.Dir(baselinePath), 0755); err != nil {
		t.Fatalf("failed to create .factory dir: %v", err)
	}

	// Create a valid baseline
	// Use AlgorithmVersion 4 to match the current detector
	baseline := dupcode.Baseline{
		SchemaVersion:    1,
		AlgorithmVersion: 4,
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

	// Add baseline to git
	if _, err := runGit(ctx, tmpDir, "add", ".factory/dupcode-baseline.json"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Get the verifier closures
	dupcodeVerifier := factory.SharedDupCodeVerifier()
	baselineVerifier := factory.SharedDupcodeBaselineVerifier()

	// Call dupcode verifier first
	dupcodeFindings := dupcodeVerifier(tmpDir)

	// Verify no errors
	for _, f := range dupcodeFindings {
		if f.Kind == "baseline_load_error" || f.Kind == "dupcode_error" {
			t.Errorf("dupcode verifier returned error finding: %s: %s", f.Kind, f.Message)
		}
	}

	// Verify call count after dupcode verifier
	if callCount != 1 {
		t.Errorf("callCount after dupcode verifier = %d, want 1", callCount)
	}

	// Call baseline verifier second (should reuse the scan)
	baselineFindings := baselineVerifier(tmpDir)

	// Verify no errors
	for _, f := range baselineFindings {
		if f.Kind == "baseline_validation_error" || f.Kind == "dupcode_error" {
			t.Errorf("baseline verifier returned error finding: %s: %s", f.Kind, f.Message)
		}
	}

	// Verify call count is still 1 (shared)
	if callCount != 1 {
		t.Errorf("callCount after baseline verifier = %d, want 1 (shared)", callCount)
	}

	// Verify execution count
	if provider.Executions() != 1 {
		t.Errorf("provider.Executions() = %d, want 1", provider.Executions())
	}

	// Verify both consumers reached the provider
	if provider.result == nil {
		t.Fatal("provider has no successful result")
	}
	if got := provider.result.Consumers; got != 2 {
		t.Errorf("provider consumers = %d, want 2", got)
	}
}

// requireFindingKind verifies that at least one finding of the given kind exists.
func requireFindingKind(t *testing.T, findings []checks.Finding, kind, label string) {
	t.Helper()
	for _, f := range findings {
		if f.Kind == kind {
			return
		}
	}
	t.Fatalf("finding kind %q absent in %s: %#v", kind, label, findings)
}

// TestSharedVerifiersMemoizeAnalysisFailure tests that when the analyzer fails,
// both verifier closures receive the same error without retrying.
// Verifier order: baseline first, then dupcode.
func TestSharedVerifiersMemoizeAnalysisFailure(t *testing.T) {
	testMemoizeAnalysisFailure(t, "baseline-first")
}

// TestSharedVerifiersMemoizeAnalysisFailureReverseOrder tests the same failure
// scenario but with verifiers called in reverse order: dupcode first, then baseline.
func TestSharedVerifiersMemoizeAnalysisFailureReverseOrder(t *testing.T) {
	testMemoizeAnalysisFailure(t, "dupcode-first")
}

func testMemoizeAnalysisFailure(t *testing.T, order string) {
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

	// Commit initial state
	if _, err := runGit(ctx, tmpDir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a failing analyzer
	failErr := errors.New("simulated scan failure")
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		return nil, failErr
	}

	// Create provider and factory
	provider := NewDupcodeAnalysisProvider(testInput(tmpDir, 40, 400, nil, nil, true), fakeAnalyzer)
	analysisCtx := NewDupcodeAnalysisContext(provider)
	factory := NewDupcodeVerifierFactory(analysisCtx)
	baselinePath := filepath.Join(tmpDir, ".factory", "dupcode-baseline.json")
	if err := os.MkdirAll(filepath.Dir(baselinePath), 0755); err != nil {
		t.Fatalf("failed to create .factory dir: %v", err)
	}

	// Create a valid baseline using the actual algorithm version
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

	// Add baseline to git
	if _, err := runGit(ctx, tmpDir, "add", ".factory/dupcode-baseline.json"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Get the verifier closures
	dupcodeVerifier := factory.SharedDupCodeVerifier()
	baselineVerifier := factory.SharedDupcodeBaselineVerifier()

	var dupcodeFindings, baselineFindings []checks.Finding

	// Execute verifiers based on order
	switch order {
	case "baseline-first":
		baselineFindings = baselineVerifier(tmpDir)
		dupcodeFindings = dupcodeVerifier(tmpDir)
	case "dupcode-first":
		dupcodeFindings = dupcodeVerifier(tmpDir)
		baselineFindings = baselineVerifier(tmpDir)
	default:
		t.Fatalf("unknown order: %s", order)
	}

	// Independently verify both closures received dupcode_error
	requireFindingKind(t, dupcodeFindings, "dupcode_error", "dupcodeFindings")
	requireFindingKind(t, baselineFindings, "dupcode_error", "baselineFindings")

	// Verify only one analyzer execution (no retry after failure)
	if got := provider.Executions(); got != 1 {
		t.Errorf("provider.Executions() = %d, want 1 (no retry after failure)", got)
	}
}
