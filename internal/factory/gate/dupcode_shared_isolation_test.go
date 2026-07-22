// Package gate provides tests for dupcode isolation properties.
package gate

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// TestDupcodeAnalysisProviderMutationSafety tests that provider is immune to
// caller mutation of the config after provider creation.
func TestDupcodeAnalysisProviderMutationSafety(t *testing.T) {
	callCount := 0
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		callCount++
		return nil, nil
	}

	// Create a mutable config
	mutableCfg := dupcode.Config{
		Root:                ".",
		MinLines:            40,
		MinTokens:           400,
		ExcludeDirs:         []string{"a", "b"},
		ExcludeFileSuffixes: []string{".gen.go"},
		IgnoreGenerated:     false,
	}

	// Create provider
	provider := NewDupcodeAnalysisProvider(newDupcodeInput(mutableCfg), fakeAnalyzer)

	// Mutate the original config after provider creation
	mutableCfg.MinLines = 999
	mutableCfg.ExcludeDirs[0] = "modified"
	mutableCfg.ExcludeFileSuffixes[0] = "modified"
	mutableCfg.IgnoreGenerated = true

	// First consumer should still work (provider has its own copy)
	_, err := provider.ConsumedBy("dupcode", testInput(".", 40, 400, []string{"a", "b"}, []string{".gen.go"}, false))
	if err != nil {
		t.Fatalf("ConsumedBy failed after config mutation: %v", err)
	}

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (mutation should not affect provider)", callCount)
	}

	// Verify second consumer with original values still works
	_, err2 := provider.ConsumedBy("dupcode-baseline", testInput(".", 40, 400, []string{"a", "b"}, []string{".gen.go"}, false))
	if err2 != nil {
		t.Fatalf("ConsumedBy with original config failed: %v", err2)
	}

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (should still be memoized)", callCount)
	}
}

// TestDupcodeSharedResultCannotBeMutatedAcrossConsumers tests that findings
// are deep-copied and cannot be mutated across consumers.
func TestDupcodeSharedResultCannotBeMutatedAcrossConsumers(t *testing.T) {
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		return []dupcode.Finding{
			{
				Fingerprint:       "test-fp-12345678901234567890123456789012345678901234",
				StableFingerprint: "test-fp-1234567890123456789012345678901234567890123456789012",
				TokenCount:        100,
				LineCount:         20,
				Occurrences: []dupcode.Occurrence{
					{Path: "a.go", StartLine: 10, EndLine: 30},
				},
			},
		}, nil
	}

	provider := NewDupcodeAnalysisProvider(testInput(".", 40, 400, nil, nil, false), fakeAnalyzer)

	// First consumer gets the result
	analysis1, err := provider.ConsumedBy("dupcode", testInput(".", 40, 400, nil, nil, false))
	if err != nil {
		t.Fatalf("first ConsumedBy failed: %v", err)
	}

	// Mutate the findings from first consumer
	analysis1.Findings[0].TokenCount = 9999
	analysis1.Findings[0].Occurrences[0].Path = "modified.go"

	// Second consumer should see original data
	analysis2, err := provider.ConsumedBy("dupcode-baseline", testInput(".", 40, 400, nil, nil, false))
	if err != nil {
		t.Fatalf("second ConsumedBy failed: %v", err)
	}

	// Verify second consumer sees original values
	if analysis2.Findings[0].TokenCount == 9999 {
		t.Error("mutation leaked to second consumer: TokenCount was mutated")
	}
	if analysis2.Findings[0].Occurrences[0].Path == "modified.go" {
		t.Error("mutation leaked to second consumer: Occurrence.Path was mutated")
	}
	if analysis2.Findings[0].TokenCount != 100 {
		t.Errorf("analysis2.Findings[0].TokenCount = %d, want 100", analysis2.Findings[0].TokenCount)
	}
	if analysis2.Findings[0].Occurrences[0].Path != "a.go" {
		t.Errorf("analysis2.Findings[0].Occurrences[0].Path = %q, want %q",
			analysis2.Findings[0].Occurrences[0].Path, "a.go")
	}
}

// TestDeepCopyFindings tests the deep copy function.
func TestDeepCopyFindings(t *testing.T) {
	original := []dupcode.Finding{
		{
			Fingerprint:       "fp1",
			StableFingerprint: "fp1-stable",
			TokenCount:        100,
			LineCount:         20,
			Occurrences: []dupcode.Occurrence{
				{Path: "a.go", StartLine: 10, EndLine: 30},
				{Path: "b.go", StartLine: 20, EndLine: 40},
			},
		},
	}

	copied := deepCopyFindings(original)

	// Modify original
	original[0].TokenCount = 9999
	original[0].Occurrences[0].Path = "modified.go"

	// Verify copy is unaffected
	if copied[0].TokenCount == 9999 {
		t.Error("deepCopyFindings did not copy TokenCount")
	}
	if copied[0].Occurrences[0].Path == "modified.go" {
		t.Error("deepCopyFindings did not copy Occurrences")
	}

	// Verify same values
	if copied[0].TokenCount != 100 {
		t.Errorf("copied.TokenCount = %d, want 100", copied[0].TokenCount)
	}
	if copied[0].Occurrences[0].Path != "a.go" {
		t.Errorf("copied.Occurrences[0].Path = %q, want %q", copied[0].Occurrences[0].Path, "a.go")
	}
}

// TestDeepCopyFindingsNil tests deep copy with nil input.
func TestDeepCopyFindingsNil(t *testing.T) {
	result := deepCopyFindings(nil)
	if result != nil {
		t.Errorf("deepCopyFindings(nil) = %v, want nil", result)
	}
}

// TestDeepCopyOccurrencesNil tests deep copy occurrences with nil input.
func TestDeepCopyOccurrencesNil(t *testing.T) {
	result := deepCopyOccurrences(nil)
	if result != nil {
		t.Errorf("deepCopyOccurrences(nil) = %v, want nil", result)
	}
}

// TestNewDupcodeAnalysisProviderClonesRawInput tests that the provider clones
// its input even when given a raw DupcodeInput (not via newDupcodeInput).
func TestNewDupcodeAnalysisProviderClonesRawInput(t *testing.T) {
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		return nil, nil
	}

	// Create a raw DupcodeInput with mutable slices
	mutableCfg := dupcode.Config{
		Root:                ".",
		MinLines:            40,
		MinTokens:           400,
		ExcludeDirs:         []string{"a", "b"},
		ExcludeFileSuffixes: []string{".gen.go"},
		IgnoreGenerated:     false,
	}

	// Pass raw input directly to constructor (not via newDupcodeInput)
	rawInput := DupcodeInput{Config: mutableCfg}
	provider := NewDupcodeAnalysisProvider(rawInput, fakeAnalyzer)

	// Mutate the original config after provider creation
	mutableCfg.ExcludeDirs[0] = "mutated"
	mutableCfg.ExcludeFileSuffixes[0] = "mutated"

	// First consumer should still work (provider has its own copy)
	_, err := provider.ConsumedBy("dupcode", testInput(".", 40, 400, []string{"a", "b"}, []string{".gen.go"}, false))
	if err != nil {
		t.Fatalf("ConsumedBy failed after raw input mutation: %v", err)
	}
}

// TestConsumedByCanonicalizesRawInput tests that ConsumedBy canonicalizes its
// argument so that semantically equivalent configurations are treated equally.
func TestConsumedByCanonicalizesRawInput(t *testing.T) {
	callCount := 0
	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		callCount++
		return nil, nil
	}

	// Create provider with nil exclusions (canonicalized to defaults)
	rawInput := DupcodeInput{
		Config: dupcode.Config{
			Root:      ".",
			MinLines:  40,
			MinTokens: 400,
			// nil exclusions → canonicalized to defaults
		},
	}
	provider := NewDupcodeAnalysisProvider(rawInput, fakeAnalyzer)

	// ConsumedBy with raw input (nil exclusions) should work
	_, err := provider.ConsumedBy("dupcode", rawInput)
	if err != nil {
		t.Fatalf("ConsumedBy raw input with nil exclusions: %v", err)
	}

	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Second call with the same raw input should use memoized result
	_, err2 := provider.ConsumedBy("dupcode-baseline", rawInput)
	if err2 != nil {
		t.Fatalf("ConsumedBy second call: %v", err2)
	}

	if callCount != 1 {
		t.Errorf("callCount after second raw call = %d, want 1 (memoized)", callCount)
	}
}

// TestDupcodeAnalyzerCannotMutateProviderConfig tests that the analyzer receives
// a clone of the config and cannot affect the provider's bound configuration.
func TestDupcodeAnalyzerCannotMutateProviderConfig(t *testing.T) {
	callCount := 0
	var receivedCfg *dupcode.Config

	fakeAnalyzer := func(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
		callCount++
		// Store a pointer to the received config
		receivedCfg = &cfg
		// Mutate the config to prove it doesn't affect the provider
		cfg.ExcludeDirs[0] = "analyzer-mutated"
		cfg.ExcludeFileSuffixes[0] = "analyzer-mutated"
		return nil, nil
	}

	// Create provider with explicit exclusion settings
	provider := NewDupcodeAnalysisProvider(testInput(".", 40, 400, []string{"original"}, []string{".original"}, false), fakeAnalyzer)

	// First consumer
	_, err := provider.ConsumedBy("dupcode", testInput(".", 40, 400, []string{"original"}, []string{".original"}, false))
	if err != nil {
		t.Fatalf("first ConsumedBy failed: %v", err)
	}

	// Verify analyzer received the cloned config
	if receivedCfg == nil {
		t.Fatal("analyzer was not called")
	}
	if receivedCfg.ExcludeDirs[0] != "analyzer-mutated" {
		t.Errorf("analyzer did not receive cloned config (mutations not observed)")
	}

	// Second consumer should still match the original bound configuration
	_, err2 := provider.ConsumedBy("dupcode-baseline", testInput(".", 40, 400, []string{"original"}, []string{".original"}, false))
	if err2 != nil {
		t.Fatalf("second ConsumedBy failed: %v", err2)
	}

	// Should still be only one execution (memoized)
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (analyzer mutations should not affect memoization)", callCount)
	}
}
