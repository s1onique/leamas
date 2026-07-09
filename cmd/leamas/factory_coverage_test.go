// Package main provides tests for the factory coverage command.
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	// minimalCoverageProfile is a minimal coverage profile representing ~60% coverage.
	minimalCoverageProfile = `mode: atomic
github.com/s1onique/leamas/internal/factory/foo.go:1.1,5.2 60 1
github.com/s1onique/leamas/internal/factory/bar.go:10.1,20.2 40 0
`
)

func writeTempProfile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "coverage-*.out")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return f.Name()
}

func TestParseCoverageArgs_MissingProfileArgument(t *testing.T) {
	_, err := parseCoverageArgs([]string{"--profile"})
	if err == nil {
		t.Error("expected error for missing --profile argument")
	}
	if !strings.Contains(err.Error(), "requires a path argument") {
		t.Errorf("expected 'requires a path argument' in error, got: %v", err)
	}
}

func TestParseCoverageArgs_MissingMinTotal(t *testing.T) {
	// --min-total is not required by parseCoverageArgs, it's validated in runFactoryCoverage
	// This test verifies that parseCoverageArgs doesn't error on missing --min-total
	args, err := parseCoverageArgs([]string{"--profile", "/path/to/profile"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.profilePath != "/path/to/profile" {
		t.Errorf("expected profilePath '/path/to/profile', got '%s'", args.profilePath)
	}
	// minTotal defaults to 0
	if args.minTotal != 0 {
		t.Errorf("expected minTotal 0, got %f", args.minTotal)
	}
}

func TestParseCoverageArgs_MissingMinTotalArgument(t *testing.T) {
	_, err := parseCoverageArgs([]string{"--profile", "/path", "--min-total"})
	if err == nil {
		t.Error("expected error for missing --min-total argument")
	}
	if !strings.Contains(err.Error(), "--min-total requires a float argument") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestParseCoverageArgs_InvalidMinTotal(t *testing.T) {
	_, err := parseCoverageArgs([]string{"--profile", "/path", "--min-total", "not-a-number"})
	if err == nil {
		t.Error("expected error for invalid --min-total")
	}
	if !strings.Contains(err.Error(), "invalid --min-total value") {
		t.Errorf("expected 'invalid --min-total value' in error, got: %v", err)
	}
}

func TestParseCoverageArgs_UnknownFlag(t *testing.T) {
	_, err := parseCoverageArgs([]string{"--profile", "/path", "--min-total", "60", "--unknown"})
	if err == nil {
		t.Error("expected error for unknown flag")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("expected 'unknown flag' in error, got: %v", err)
	}
}

func TestParseCoverageArgs_ValidWithNoBreakdown(t *testing.T) {
	args, err := parseCoverageArgs([]string{"--profile", "/path", "--min-total", "60", "--no-breakdown"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.profilePath != "/path" {
		t.Errorf("expected profilePath '/path', got '%s'", args.profilePath)
	}
	if args.minTotal != 60 {
		t.Errorf("expected minTotal 60, got %f", args.minTotal)
	}
	if args.showBreakdown {
		t.Error("expected showBreakdown to be false")
	}
}

func TestParseCoverageArgs_ValidWithBreakdown(t *testing.T) {
	args, err := parseCoverageArgs([]string{"--profile", "/path", "--min-total", "60", "--breakdown"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !args.showBreakdown {
		t.Error("expected showBreakdown to be true")
	}
}

func TestParseCoverageArgs_ValidWithJSONOutput(t *testing.T) {
	args, err := parseCoverageArgs([]string{"--profile", "/path", "--min-total", "60", "--json-output", "/out.json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.jsonOutputPath != "/out.json" {
		t.Errorf("expected jsonOutputPath '/out.json', got '%s'", args.jsonOutputPath)
	}
}

func TestParseCoverageArgs_DefaultBreakdown(t *testing.T) {
	args, err := parseCoverageArgs([]string{"--profile", "/path", "--min-total", "60"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !args.showBreakdown {
		t.Error("expected showBreakdown to be true by default")
	}
}

func TestRunFactoryCoverage_MissingProfile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{"--min-total", "60"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--profile is required") {
		t.Errorf("expected '--profile is required' in stderr, got: %s", stderr.String())
	}
}

func TestRunFactoryCoverage_InvalidProfile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{"--profile", "/nonexistent", "--min-total", "60"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "error parsing profile") {
		t.Errorf("expected 'error parsing profile' in stderr, got: %s", stderr.String())
	}
}

func TestRunFactoryCoverage_ThresholdPass(t *testing.T) {
	profilePath := writeTempProfile(t, minimalCoverageProfile)
	defer os.Remove(profilePath)

	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{"--profile", profilePath, "--min-total", "50", "--no-breakdown"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "OK") {
		t.Errorf("expected 'OK' in stdout, got: %s", stdout.String())
	}
}

func TestRunFactoryCoverage_ThresholdFail(t *testing.T) {
	// Profile with zero coverage to ensure threshold failure
	zeroCoverageProfile := `mode: atomic
github.com/s1onique/leamas/internal/factory/foo.go:1.1,5.2 60 0
`
	profilePath := writeTempProfile(t, zeroCoverageProfile)
	defer os.Remove(profilePath)

	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{"--profile", profilePath, "--min-total", "50", "--no-breakdown"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "below minimum") {
		t.Errorf("expected 'below minimum' in stderr, got: %s", stderr.String())
	}
}

func TestRunFactoryCoverage_WritesJSON(t *testing.T) {
	profilePath := writeTempProfile(t, minimalCoverageProfile)
	defer os.Remove(profilePath)

	jsonPath := filepath.Join(t.TempDir(), "coverage.json")

	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{"--profile", profilePath, "--min-total", "50", "--no-breakdown", "--json-output", jsonPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON output: %v", err)
	}
	if !strings.Contains(string(data), "total") {
		t.Errorf("expected JSON to contain 'total', got: %s", string(data))
	}
}

func TestRunFactoryCoverage_NoBreakdown(t *testing.T) {
	profilePath := writeTempProfile(t, minimalCoverageProfile)
	defer os.Remove(profilePath)

	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{"--profile", profilePath, "--min-total", "50", "--no-breakdown"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	// When --no-breakdown is used, module table should not be printed
	// The output should only contain the OK line
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line of output with --no-breakdown, got %d: %v", len(lines), lines)
	}
}

func TestRunFactoryCoverage_InvalidMinTotal(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{"--profile", "/path", "--min-total", "not-a-float"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid --min-total value") {
		t.Errorf("expected 'invalid --min-total value' in stderr, got: %s", stderr.String())
	}
}

func TestRunFactoryCoverage_UnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{"--profile", "/path", "--min-total", "60", "--unknown-flag"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Errorf("expected 'unknown flag' in stderr, got: %s", stderr.String())
	}
}

func TestCoverageUsageText_IncludesRequiredFlags(t *testing.T) {
	var buf bytes.Buffer
	printCoverageUsageTo(&buf)
	if !strings.Contains(buf.String(), "--profile") {
		t.Error("usage text should include --profile flag")
	}
	if !strings.Contains(buf.String(), "--min-total") {
		t.Error("usage text should include --min-total flag")
	}
}

func TestRunFactoryCoverage_ExactThreshold(t *testing.T) {
	profilePath := writeTempProfile(t, minimalCoverageProfile)
	defer os.Remove(profilePath)

	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{"--profile", profilePath, "--min-total", "50", "--no-breakdown"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit code 0 for exact threshold, got %d", code)
	}
}

func TestRunFactoryCoverage_OneOverThreshold(t *testing.T) {
	profilePath := writeTempProfile(t, minimalCoverageProfile)
	defer os.Remove(profilePath)

	var stdout, stderr bytes.Buffer
	code := runFactoryCoverage([]string{"--profile", profilePath, "--min-total", "49", "--no-breakdown"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit code 0 for one over threshold, got %d", code)
	}
}
