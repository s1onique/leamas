package coverage

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestMakefileCoverageDefaultsMatchCanonicalThresholds verifies that Makefile
// COVERAGE_MIN_* variables match the canonical thresholds in defaults.go.
// This test parses the Makefile and fails if defaults drift.
func TestMakefileCoverageDefaultsMatchCanonicalThresholds(t *testing.T) {
	// Find the Makefile in the repository root
	makefilePath := findMakefile(t)
	if makefilePath == "" {
		t.Skip("Makefile not found")
	}

	// Read the Makefile
	content, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("failed to read Makefile: %v", err)
	}
	makefileContent := string(content)

	// Get canonical thresholds
	defaultThreshold := DefaultThreshold()
	defaultModules := DefaultModuleThresholds()

	// Define expected Makefile variable mappings
	type makefileVar struct {
		name     string
		expected float64
	}
	expectedVars := []makefileVar{
		{"COVERAGE_MIN_TOTAL", defaultThreshold.MinTotalPercent},
		{"COVERAGE_MIN_CMD_LEAMAS", defaultModules["cmd/leamas"]},
		{"COVERAGE_MIN_INTERNAL_FACTORY", defaultModules["internal/factory"]},
		{"COVERAGE_MIN_INTERNAL_HULK", defaultModules["internal/hulk"]},
		{"COVERAGE_MIN_INTERNAL_WEB", defaultModules["internal/web"]},
		{"COVERAGE_MIN_INTERNAL_WITNESS", defaultModules["internal/witness"]},
	}

	// Check each variable matches canonical thresholds
	for _, v := range expectedVars {
		if !strings.Contains(makefileContent, v.name) {
			t.Errorf("Makefile missing variable %s (expected %.1f)", v.name, v.expected)
			continue
		}

		// Parse the variable value from Makefile
		// Format: COVERAGE_MIN_TOTAL ?= 64
		pattern := v.name + " ?= "
		idx := strings.Index(makefileContent, pattern)
		if idx == -1 {
			// Try without spaces
			pattern = v.name + "="
			idx = strings.Index(makefileContent, pattern)
		}
		if idx == -1 {
			t.Errorf("Makefile variable %s not found", v.name)
			continue
		}

		// Extract the value after the assignment
		rest := makefileContent[idx+len(pattern):]
		endIdx := 0
		for endIdx < len(rest) && (rest[endIdx] >= '0' && rest[endIdx] <= '9' || rest[endIdx] == '.') {
			endIdx++
		}
		valueStr := strings.TrimSpace(rest[:endIdx])

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			t.Errorf("Makefile %s has invalid value %q: %v", v.name, valueStr, err)
			continue
		}

		if value != v.expected {
			t.Errorf("Makefile %s = %.1f, want %.1f (canonical threshold)", v.name, value, v.expected)
		}
	}
}

// TestDefaultModuleThresholds_ReturnsDefensiveCopy verifies that
// DefaultModuleThresholds returns a defensive copy.
func TestDefaultModuleThresholds_ReturnsDefensiveCopy(t *testing.T) {
	first := DefaultModuleThresholds()
	second := DefaultModuleThresholds()

	// Modify the first
	first["cmd/leamas"] = 99.0

	// Second should be unchanged
	if second["cmd/leamas"] == 99.0 {
		t.Error("DefaultModuleThresholds() returned a shared map, not a defensive copy")
	}
}

// TestKnownEnforcedModules_DeterministicOrder verifies that KnownEnforcedModules
// returns modules in deterministic order.
func TestKnownEnforcedModules_DeterministicOrder(t *testing.T) {
	first := KnownEnforcedModules()
	second := KnownEnforcedModules()

	if len(first) != len(second) {
		t.Fatalf("KnownEnforcedModules() returned different lengths: %d vs %d", len(first), len(second))
	}

	for i := range first {
		if first[i] != second[i] {
			t.Errorf("KnownEnforcedModules() order is non-deterministic: index %d differs", i)
		}
	}
}

// TestKnownEnforcedModules_ReturnsDefensiveCopy verifies that KnownEnforcedModules
// returns a defensive copy that can be mutated without affecting future calls.
func TestKnownEnforcedModules_ReturnsDefensiveCopy(t *testing.T) {
	first := KnownEnforcedModules()
	originalFirstLen := len(first)

	// Mutate the first
	if len(first) > 0 {
		first[0] = "mutated-module"
	}

	// Second should be unchanged
	second := KnownEnforcedModules()
	if len(second) != originalFirstLen {
		t.Error("KnownEnforcedModules() returned a shared slice, mutation affected subsequent call")
	}
	if second[0] == "mutated-module" {
		t.Error("KnownEnforcedModules() returned a shared slice, mutation affected subsequent call")
	}
}

// TestIsKnownEnforcedModule verifies that IsKnownEnforcedModule works correctly.
func TestIsKnownEnforcedModule(t *testing.T) {
	// Known enforceable modules
	knownModules := []string{
		"cmd/leamas",
		"internal/factory",
		"internal/hulk",
		"internal/web",
		"internal/witness",
	}
	for _, m := range knownModules {
		if !IsKnownEnforcedModule(m) {
			t.Errorf("IsKnownEnforcedModule(%q) = false, want true", m)
		}
	}

	// Unknown/report-only modules
	unknownModules := []string{
		"other",
		"unknown",
		"some/other/module",
		"",
	}
	for _, m := range unknownModules {
		if IsKnownEnforcedModule(m) {
			t.Errorf("IsKnownEnforcedModule(%q) = true, want false", m)
		}
	}
}

// TestDefaultModuleFloorsDoesNotIncludeOther verifies that "other" is not in
// the enforceable modules (it's report-only).
func TestDefaultModuleFloorsDoesNotIncludeOther(t *testing.T) {
	modules := DefaultModuleThresholds()
	if _, exists := modules["other"]; exists {
		t.Error("DefaultModuleThresholds() should not include 'other' (report-only)")
	}
}

// findMakefile searches for the Makefile starting from the test directory.
func findMakefile(t *testing.T) string {
	t.Helper()

	// Start from the test file location
	dir := "."
	for i := 0; i < 10; i++ { // Limit search depth
		makefilePath := filepath.Join(dir, "Makefile")
		if _, err := os.Stat(makefilePath); err == nil {
			return makefilePath
		}
		// Go up one directory
		dir = filepath.Join(dir, "..")
	}
	return ""
}
