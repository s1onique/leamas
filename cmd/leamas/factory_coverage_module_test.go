// Package main provides tests for the factory coverage command module thresholds.
package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// moduleMinCoverageProfile is a minimal coverage profile for module threshold tests.
const moduleMinCoverageProfile = `mode: atomic
github.com/s1onique/leamas/cmd/leamas/main.go:1.1,10.2 100 100
github.com/s1onique/leamas/internal/factory/foo.go:1.1,10.2 100 100
`

// lowCoverageProfile is a profile with low coverage for module threshold failure tests.
const lowCoverageProfile = `mode: atomic
github.com/s1onique/leamas/cmd/leamas/main.go:1.1,10.2 100 0
github.com/s1onique/leamas/internal/factory/foo.go:1.1,10.2 100 100
`

func TestParseCoverageArgs_MinModule(t *testing.T) {
	args, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module", "cmd/leamas=50",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.profilePath != "/path" {
		t.Errorf("expected profilePath '/path', got '%s'", args.profilePath)
	}
	if args.minTotal != 60 {
		t.Errorf("expected minTotal 60, got %f", args.minTotal)
	}
	if args.minModulePercents == nil {
		t.Fatal("expected minModulePercents to be initialized")
	}
	floor, exists := args.minModulePercents["cmd/leamas"]
	if !exists {
		t.Error("expected cmd/leamas in minModulePercents")
	}
	if floor != 50 {
		t.Errorf("expected cmd/leamas floor 50, got %f", floor)
	}
}

func TestParseCoverageArgs_MinModuleRepeated(t *testing.T) {
	args, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module", "cmd/leamas=50",
		"--min-module", "internal/factory=67",
		"--min-module", "internal/hulk=90",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args.minModulePercents) != 3 {
		t.Errorf("expected 3 module floors, got %d", len(args.minModulePercents))
	}
	tests := map[string]float64{
		"cmd/leamas":       50,
		"internal/factory": 67,
		"internal/hulk":    90,
	}
	for module, expectedFloor := range tests {
		floor, exists := args.minModulePercents[module]
		if !exists {
			t.Errorf("expected %s in minModulePercents", module)
			continue
		}
		if floor != expectedFloor {
			t.Errorf("%s floor = %f, want %f", module, floor, expectedFloor)
		}
	}
}

func TestParseCoverageArgs_MinModuleMissingValue(t *testing.T) {
	_, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module",
	})
	if err == nil {
		t.Error("expected error for missing --min-module value")
	}
	if !strings.Contains(err.Error(), "--min-module requires a value") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestParseCoverageArgs_MinModuleMissingEquals(t *testing.T) {
	_, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module", "cmd-leamas-50",
	})
	if err == nil {
		t.Error("expected error for missing equals in --min-module")
	}
	if !strings.Contains(err.Error(), "--min-module requires format module=threshold") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestParseCoverageArgs_MinModuleInvalidFloat(t *testing.T) {
	_, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module", "cmd/leamas=not-a-number",
	})
	if err == nil {
		t.Error("expected error for invalid float in --min-module")
	}
	if !strings.Contains(err.Error(), "--min-module threshold must be a valid float") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestParseCoverageArgs_MinModuleNegativeThreshold(t *testing.T) {
	_, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module", "cmd/leamas=-10",
	})
	if err == nil {
		t.Error("expected error for negative threshold in --min-module")
	}
	if !strings.Contains(err.Error(), "--min-module threshold cannot be negative") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestParseCoverageArgs_MinModuleExceeds100(t *testing.T) {
	_, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module", "cmd/leamas=150",
	})
	if err == nil {
		t.Error("expected error for threshold > 100 in --min-module")
	}
	if !strings.Contains(err.Error(), "--min-module threshold cannot exceed 100") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestParseCoverageArgs_MinModuleEmptyName(t *testing.T) {
	_, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module", "=50",
	})
	if err == nil {
		t.Error("expected error for empty module name in --min-module")
	}
	if !strings.Contains(err.Error(), "--min-module module name cannot be empty") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestParseCoverageArgs_DefaultModuleFloors(t *testing.T) {
	args, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--default-module-floors",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]float64{
		"cmd/leamas":       50.0,
		"internal/factory": 67.0,
		"internal/hulk":    90.0,
		"internal/web":     70.0,
		"internal/witness": 80.0,
	}
	for module, expectedFloor := range expected {
		floor, exists := args.minModulePercents[module]
		if !exists {
			t.Errorf("expected %s in minModulePercents", module)
			continue
		}
		if floor != expectedFloor {
			t.Errorf("%s floor = %f, want %f", module, floor, expectedFloor)
		}
	}
}

func TestRunFactoryCoverage_ModuleThresholdPass(t *testing.T) {
	profilePath := writeTempProfile(t, moduleMinCoverageProfile)
	defer os.Remove(profilePath)

	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{
		"--profile", profilePath,
		"--min-total", "50",
		"--min-module", "cmd/leamas=40",
		"--min-module", "internal/factory=40",
		"--no-breakdown",
	}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "OK") {
		t.Errorf("expected 'OK' in stdout, got: %s", stdout.String())
	}
}

func TestRunFactoryCoverage_ModuleThresholdFail(t *testing.T) {
	profilePath := writeTempProfile(t, lowCoverageProfile)
	defer os.Remove(profilePath)

	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{
		"--profile", profilePath,
		"--min-total", "50",
		"--min-module", "cmd/leamas=50",
		"--min-module", "internal/factory=50",
		"--no-breakdown",
	}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d. stdout: %s, stderr: %s",
			code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "module_threshold_fail") {
		t.Errorf("expected 'module_threshold_fail' in stderr, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "cmd/leamas") {
		t.Errorf("expected 'cmd/leamas' in stderr, got: %s", stderr.String())
	}
}

func TestParseCoverageArgs_MinModuleUnknownModule(t *testing.T) {
	_, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module", "potato=50",
	})
	if err == nil {
		t.Error("expected error for unknown module name")
	}
	if !strings.Contains(err.Error(), "--min-module unknown module: potato") {
		t.Errorf("expected specific error message about unknown module, got: %v", err)
	}
}

func TestParseCoverageArgs_ExplicitOverridesDefaultFloors(t *testing.T) {
	// Explicit --min-module should win over --default-module-floors
	// regardless of order: explicit first
	args, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module", "cmd/leamas=55",
		"--default-module-floors",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	floor, exists := args.minModulePercents["cmd/leamas"]
	if !exists {
		t.Fatal("expected cmd/leamas in minModulePercents")
	}
	if floor != 55 {
		t.Errorf("expected explicit value 55, got %f", floor)
	}
}

func TestParseCoverageArgs_ExplicitOverridesDefaultFloorsReversed(t *testing.T) {
	// Explicit --min-module should win over --default-module-floors
	// regardless of order: default first
	args, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--default-module-floors",
		"--min-module", "cmd/leamas=55",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	floor, exists := args.minModulePercents["cmd/leamas"]
	if !exists {
		t.Fatal("expected cmd/leamas in minModulePercents")
	}
	if floor != 55 {
		t.Errorf("expected explicit value 55, got %f", floor)
	}
}

func TestParseCoverageArgs_DefaultFloorsFillMissing(t *testing.T) {
	// --default-module-floors should fill in missing modules
	args, err := parseCoverageArgs([]string{
		"--profile", "/path",
		"--min-total", "60",
		"--min-module", "cmd/leamas=50",
		"--default-module-floors",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// cmd/leamas should be explicit 50
	if floor, exists := args.minModulePercents["cmd/leamas"]; !exists || floor != 50 {
		t.Errorf("expected cmd/leamas=50, got %v", args.minModulePercents["cmd/leamas"])
	}
	// internal/factory should be default 67.0
	if floor, exists := args.minModulePercents["internal/factory"]; !exists || floor != 67.0 {
		t.Errorf("expected internal/factory=67.0, got %v", args.minModulePercents["internal/factory"])
	}
}

func TestRunFactoryCoverage_ModuleOKLines(t *testing.T) {
	profilePath := writeTempProfile(t, moduleMinCoverageProfile)
	defer os.Remove(profilePath)

	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{
		"--profile", profilePath,
		"--min-total", "50",
		"--min-module", "cmd/leamas=40",
		"--min-module", "internal/factory=40",
		"--no-breakdown",
	}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	// Check for one-line output format (coverage: total=X min=Y OK)
	if !strings.Contains(stdout.String(), "coverage:") {
		t.Errorf("expected 'coverage:' in stdout, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), " OK") {
		t.Errorf("expected 'OK' in stdout, got: %s", stdout.String())
	}
	// Should be exactly one line (no per-module breakdown in default output)
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line of output, got %d: %s", len(lines), stdout.String())
	}
}
