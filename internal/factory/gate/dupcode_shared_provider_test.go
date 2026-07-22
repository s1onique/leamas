// Package gate provides tests for the dupcode shared analysis provider.
package gate

import (
	"errors"
	"testing"

	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// TestNewDupcodeAnalysisProvider tests provider creation.
func TestNewDupcodeAnalysisProvider(t *testing.T) {
	// Test with nil analyzer (should use default)
	provider := NewDupcodeAnalysisProvider(testInput(".", 40, 400, nil, nil, false), nil)
	if provider == nil {
		t.Fatal("NewDupcodeAnalysisProvider returned nil")
	}
	if provider.analyzer == nil {
		t.Error("provider.analyzer is nil, expected default")
	}
	if provider.state != stateEmpty {
		t.Errorf("provider.state = %v, want stateEmpty", provider.state)
	}
}

// TestFactorizeDupcodeVerifiersShareSingleAnalysis tests that both verifiers
// execute but analysis function is called exactly once.
func TestFactorizeDupcodeVerifiersShareSingleAnalysis(t *testing.T) {
	// Create a counting analyzer
	callCount := 0
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		callCount++
		return nil, nil // No findings
	}

	// Create provider with fake analyzer
	provider := NewDupcodeAnalysisProvider(testInput(".", 40, 400, nil, nil, false), fakeAnalyzer)

	// First consumer
	_, err1 := provider.ConsumedBy("dupcode", testInput(".", 40, 400, nil, nil, false))
	if err1 != nil {
		t.Fatalf("first ConsumedBy failed: %v", err1)
	}

	// Verify call count after first consumer
	if callCount != 1 {
		t.Errorf("callCount after first consumer = %d, want 1", callCount)
	}

	// Second consumer should not trigger another scan
	_, err2 := provider.ConsumedBy("dupcode-baseline", testInput(".", 40, 400, nil, nil, false))
	if err2 != nil {
		t.Fatalf("second ConsumedBy failed: %v", err2)
	}

	// Verify call count is still 1
	if callCount != 1 {
		t.Errorf("callCount after second consumer = %d, want 1 (shared, not re-scanned)", callCount)
	}

	// Verify execution count
	if provider.Executions() != 1 {
		t.Errorf("provider.Executions() = %d, want 1", provider.Executions())
	}
}

// TestFactorizeDupcodeVerifiersShareAnalysisFailure tests that when the first
// analysis fails, the second consumer also fails without retrying.
func TestFactorizeDupcodeVerifiersShareAnalysisFailure(t *testing.T) {
	// Create a failing analyzer
	failErr := errors.New("simulated scan failure")
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		return nil, failErr
	}

	// Create provider with failing analyzer
	provider := NewDupcodeAnalysisProvider(testInput(".", 40, 400, nil, nil, false), fakeAnalyzer)

	// First consumer
	_, err1 := provider.ConsumedBy("dupcode", testInput(".", 40, 400, nil, nil, false))
	if err1 == nil {
		t.Fatal("first ConsumedBy expected error, got nil")
	}
	if !errors.Is(err1, failErr) {
		t.Errorf("first error = %v, want %v", err1, failErr)
	}

	// Second consumer should get the same error without retry
	_, err2 := provider.ConsumedBy("dupcode-baseline", testInput(".", 40, 400, nil, nil, false))
	if err2 == nil {
		t.Fatal("second ConsumedBy expected error, got nil")
	}
	if !errors.Is(err2, failErr) {
		t.Errorf("second error = %v, want same memoized error", err2)
	}

	// Should only have executed once
	if provider.Executions() != 1 {
		t.Errorf("provider.Executions() = %d, want 1 (no retry after failure)", provider.Executions())
	}
}

// TestDupcodeAnalysisProviderRejectsConfigurationMismatch tests that the provider
// rejects attempts to reuse results for different configurations.
func TestDupcodeAnalysisProviderRejectsConfigurationMismatch(t *testing.T) {
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		return nil, nil
	}

	provider := NewDupcodeAnalysisProvider(testInput(".", 40, 400, nil, nil, false), fakeAnalyzer)

	// First consumer with matching config
	_, err1 := provider.ConsumedBy("dupcode", testInput(".", 40, 400, nil, nil, false))
	if err1 != nil {
		t.Fatalf("matching config failed: %v", err1)
	}

	// Second consumer with different minLines
	_, err2 := provider.ConsumedBy("dupcode-baseline", testInput(".", 50, 400, nil, nil, false))
	if err2 == nil {
		t.Fatal("different minLines should have been rejected")
	}

	// Third consumer with different minTokens
	_, err3 := provider.ConsumedBy("dupcode", testInput(".", 40, 500, nil, nil, false))
	if err3 == nil {
		t.Fatal("different minTokens should have been rejected")
	}

	// Fourth consumer with different root
	_, err4 := provider.ConsumedBy("dupcode", testInput("/other", 40, 400, nil, nil, false))
	if err4 == nil {
		t.Fatal("different root should have been rejected")
	}

	// Fifth consumer with different ExcludeDirs
	_, err5 := provider.ConsumedBy("dupcode", testInput(".", 40, 400, []string{"extra"}, nil, false))
	if err5 == nil {
		t.Fatal("different ExcludeDirs should have been rejected")
	}

	// Sixth consumer with different ExcludeFileSuffixes
	_, err6 := provider.ConsumedBy("dupcode", testInput(".", 40, 400, nil, []string{".gen.go"}, false))
	if err6 == nil {
		t.Fatal("different ExcludeFileSuffixes should have been rejected")
	}

	// Seventh consumer with different IgnoreGenerated
	_, err7 := provider.ConsumedBy("dupcode", testInput(".", 40, 400, nil, nil, true))
	if err7 == nil {
		t.Fatal("different IgnoreGenerated should have been rejected")
	}

	// Should only have executed once (the matching config call)
	if provider.Executions() != 1 {
		t.Errorf("provider.Executions() = %d, want 1", provider.Executions())
	}
}

// TestDupcodeAnalysisProviderConsumersCount tests that the consumer count
// is correctly tracked.
func TestDupcodeAnalysisProviderConsumersCount(t *testing.T) {
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		return nil, nil
	}

	provider := NewDupcodeAnalysisProvider(testInput(".", 40, 400, nil, nil, false), fakeAnalyzer)

	// First consumer
	analysis1, err := provider.ConsumedBy("dupcode", testInput(".", 40, 400, nil, nil, false))
	if err != nil {
		t.Fatalf("first ConsumedBy failed: %v", err)
	}
	if analysis1.Consumers != 1 {
		t.Errorf("first consumer count = %d, want 1", analysis1.Consumers)
	}

	// Second consumer
	analysis2, err := provider.ConsumedBy("dupcode-baseline", testInput(".", 40, 400, nil, nil, false))
	if err != nil {
		t.Fatalf("second ConsumedBy failed: %v", err)
	}
	if analysis2.Consumers != 1 {
		t.Errorf("second consumer count = %d, want 1 (each gets its own copy)", analysis2.Consumers)
	}

	// Verify they are different pointers (each gets its own copy)
	if analysis1 == analysis2 {
		t.Error("consumers should get different result pointers (deep copied)")
	}

	// Verify the provider tracks total consumers correctly
	if provider.Executions() != 1 {
		t.Errorf("provider.Executions() = %d, want 1", provider.Executions())
	}
}
