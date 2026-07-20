package gatesummary

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestNormalizationAliasing verifies that normalization produces independent copies.
func TestNormalizationAliasing(t *testing.T) {
	validDir := filepath.Join("testdata", "valid")
	fixture := "v2-minimal.json"
	data, err := os.ReadFile(filepath.Join(validDir, fixture))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	// Decode
	decodeResult := Decode(strings.NewReader(string(data)))
	if !decodeResult.Success() {
		t.Fatalf("decode failed: %v", decodeResult.Diagnostics)
	}

	// Normalize twice
	norm1 := Normalize(decodeResult.Document)
	norm2 := Normalize(decodeResult.Document)
	if !norm1.Success() || !norm2.Success() {
		t.Fatalf("normalization failed")
	}

	// Mutate the first summary
	if len(norm1.Summary.Checks) > 0 {
		originalName := norm1.Summary.Checks[0].Name
		norm1.Summary.Checks[0].Name = "mutated"
		norm1.Summary.GeneratedAt = "mutated-timestamp"
		if norm2.Summary.Checks[0].Name == "mutated" {
			t.Error("mutation affected the second normalization result")
		}
		if norm2.Summary.GeneratedAt == "mutated-timestamp" {
			t.Error("mutation affected the second normalization result")
		}
		// Restore for cleanliness
		norm1.Summary.Checks[0].Name = originalName
		norm1.Summary.GeneratedAt = string(data)
	}

	// Mutate check execution argv
	if len(norm1.Summary.Checks) > 0 && norm1.Summary.Checks[0].Execution != nil && len(norm1.Summary.Checks[0].Execution.Argv) > 0 {
		originalArgv := norm1.Summary.Checks[0].Execution.Argv[0]
		norm1.Summary.Checks[0].Execution.Argv[0] = "mutated-argv"
		if norm2.Summary.Checks[0].Execution.Argv[0] == "mutated-argv" {
			t.Error("argv mutation affected the second normalization result")
		}
		// Restore
		norm1.Summary.Checks[0].Execution.Argv[0] = originalArgv
	}

	t.Log("aliasing test passed: mutations do not affect other normalization results")
}

// TestNormalizationConcurrency verifies that concurrent normalization is safe.
func TestNormalizationConcurrency(t *testing.T) {
	validDir := filepath.Join("testdata", "valid")
	fixtures := []string{"v1-minimal.json", "v2-minimal.json", "v2-clinemm-microc3.json"}

	var wg sync.WaitGroup
	results := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			fixture := fixtures[idx%len(fixtures)]
			data, err := os.ReadFile(filepath.Join(validDir, fixture))
			if err != nil {
				results <- false
				return
			}

			// Decode
			decodeResult := Decode(strings.NewReader(string(data)))
			if !decodeResult.Success() {
				results <- false
				return
			}

			// Normalize
			normResult := Normalize(decodeResult.Document)
			results <- normResult.Success()
		}(i)
	}

	wg.Wait()
	close(results)

	failures := 0
	successes := 0
	for ok := range results {
		if ok {
			successes++
		} else {
			failures++
		}
	}

	if failures > 0 {
		t.Errorf("concurrent normalization: %d failures out of 100", failures)
	}
	t.Logf("concurrent normalization: %d successes out of 100", successes)
}

// TestNormalizationRace runs normalization under race detector.
func TestNormalizationRace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race test in short mode")
	}
	validDir := filepath.Join("testdata", "valid")
	fixture := "v2-minimal.json"
	data, err := os.ReadFile(filepath.Join(validDir, fixture))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	decodeResult := Decode(strings.NewReader(string(data)))
	if !decodeResult.Success() {
		t.Fatalf("decode failed: %v", decodeResult.Diagnostics)
	}

	// Run multiple normalizations concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			norm := Normalize(decodeResult.Document)
			if !norm.Success() {
				t.Errorf("normalization failed: %v", norm.Diagnostics)
			}
		}()
	}
	wg.Wait()
}
