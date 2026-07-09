package coverage

import (
	"bufio"
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseProfilePath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
		wantErr  bool
	}{
		{
			"github.com/s1onique/leamas/internal/foo/foo.go:10.1,20.2",
			"github.com/s1onique/leamas/internal/foo", false,
		},
		{
			"github.com/s1onique/leamas/cmd/leamas/main.go:1.1,5.2",
			"github.com/s1onique/leamas/cmd/leamas", false,
		},
		{"invalid", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := ParseProfilePath(tt.path)
			if tt.wantErr && err == nil {
				t.Errorf("ParseProfilePath(%q) expected error", tt.path)
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ParseProfilePath(%q) unexpected error: %v", tt.path, err)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("ParseProfilePath(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestParseProfileBlock(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantNil bool
		wantNum int
		wantCov int
	}{
		{"empty", "", true, 0, 0},
		{"mode line", "mode: atomic", true, 0, 0},
		{"covered", "github.com/s1onique/leamas/internal/foo/foo.go:10.1,20.2 3 3", false, 3, 3},
		{"uncovered", "github.com/s1onique/leamas/internal/foo/foo.go:10.1,20.2 5 0", false, 5, 0},
		{"malformed", "invalid line", true, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseProfileBlock(tt.line)
			if tt.wantNil && got != nil {
				t.Errorf("ParseProfileBlock(%q) = %v, want nil", tt.line, got)
				return
			}
			if !tt.wantNil && (err != nil || got == nil) {
				t.Errorf("ParseProfileBlock(%q) unexpected result: %v, err=%v", tt.line, got, err)
				return
			}
			if !tt.wantNil {
				if got.NumStatements != tt.wantNum {
					t.Errorf("NumStatements = %d, want %d", got.NumStatements, tt.wantNum)
				}
				if got.Count != tt.wantCov {
					t.Errorf("Count = %d, want %d", got.Count, tt.wantCov)
				}
			}
		})
	}
}

func TestParseProfileReader(t *testing.T) {
	input := `mode: atomic
github.com/s1onique/leamas/cmd/leamas/main.go:1.1,5.2 10 10
github.com/s1onique/leamas/internal/factory/foo.go:10.1,20.2 100 75
github.com/s1onique/leamas/internal/factory/bar.go:5.1,8.2 50 25
`
	report, err := ParseProfileReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseProfileReader() error: %v", err)
	}

	// 3 blocks, all have count > 0 so all covered
	// 10+100+50=160 total statements
	if report.TotalStatements != 160 {
		t.Errorf("TotalStatements = %d, want 160", report.TotalStatements)
	}
	// All blocks covered (count > 0)
	if report.TotalCovered != 160 {
		t.Errorf("TotalCovered = %d, want 160", report.TotalCovered)
	}
	if report.TotalPercent != 100.0 {
		t.Errorf("TotalPercent = %v, want 100.0", report.TotalPercent)
	}

	if len(report.Modules) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(report.Modules))
	}

	if report.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, want 2", report.SchemaVersion)
	}
}

func TestParseProfileReader_ZeroStatements(t *testing.T) {
	// Test zero-statement handling
	input := `mode: atomic
github.com/s1onique/leamas/cmd/leamas/main.go:1.1,5.2 0 0
github.com/s1onique/leamas/internal/foo/foo.go:10.1,20.2 100 100
`
	report, err := ParseProfileReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseProfileReader() error: %v", err)
	}

	// Total: 100/100 = 100%
	if report.TotalPercent != 100.0 {
		t.Errorf("TotalPercent = %v, want 100.0", report.TotalPercent)
	}
}

func TestParseProfileReader_DeterministicOrdering(t *testing.T) {
	input := `mode: atomic
github.com/s1onique/leamas/cmd/leamas/main.go:1.1,5.2 10 10
github.com/s1onique/leamas/internal/factory/foo.go:10.1,20.2 100 75
github.com/s1onique/leamas/internal/hulk/bar.go:5.1,8.2 50 25
`
	var firstOrder []string
	for i := 0; i < 3; i++ {
		report, err := ParseProfileReader(strings.NewReader(input))
		if err != nil {
			t.Fatalf("ParseProfileReader() error: %v", err)
		}
		order := make([]string, len(report.Modules))
		for j, m := range report.Modules {
			order[j] = m.Module
		}
		if i == 0 {
			firstOrder = order
		} else {
			for j, mod := range order {
				if mod != firstOrder[j] {
					t.Errorf("Order not deterministic at pos %d", j)
				}
			}
		}
	}
}

func TestParseProfileReader_MalformedInput(t *testing.T) {
	input := `mode: atomic
github.com/s1onique/leamas/cmd/leamas/main.go:1.1,5.2 10 10
malformed line
github.com/s1onique/leamas/internal/foo/foo.go:10.1,20.2 100 100
`
	report, err := ParseProfileReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseProfileReader() should not error on malformed lines: %v", err)
	}

	// Should have valid data despite malformed line
	if report.TotalPercent != 100.0 {
		t.Errorf("TotalPercent = %v, want 100.0", report.TotalPercent)
	}
}

func TestWeightedVsNaive(t *testing.T) {
	// Regression test: module A has tiny package at 100% and huge package at 0%
	// Naive average would be 50%, but weighted should be 1%

	// This tests the concept - actual profile format is different
	// The weighted calculation correctly accounts for statement counts
	profileInput := `mode: atomic
github.com/s1onique/leamas/internal/foo/tiny.go:1.1,2.2 1 1
github.com/s1onique/leamas/internal/foo/huge.go:10.1,100.2 99 0
`
	report, err := ParseProfileReader(strings.NewReader(profileInput))
	if err != nil {
		t.Fatalf("ParseProfileReader() error: %v", err)
	}

	// Total: 1/100 = 1.0%
	if report.TotalPercent != 1.0 {
		t.Errorf("Weighted TotalPercent = %v, want 1.0", report.TotalPercent)
	}
}

func TestProfileReport_ToJSON(t *testing.T) {
	report := &ProfileReport{
		SchemaVersion:   2,
		TotalPercent:    63.5,
		TotalCovered:    635,
		TotalStatements: 1000,
		Modules: []WeightedModuleSummary{
			{Module: "cmd/leamas", Percent: 87.1, Packages: 1, CoveredStatements: 87, TotalStatements: 100},
		},
	}
	data, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	if !strings.Contains(string(data), `"schema_version": 2`) {
		t.Error("JSON should contain schema_version 2")
	}
	if !strings.Contains(string(data), `"covered_statements"`) {
		t.Error("JSON should contain covered_statements")
	}
}

func TestIsZeroStatementBlock(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"github.com/s1onique/leamas/foo.go:1.1,2.2 0 0", true},
		{"github.com/s1onique/leamas/foo.go:1.1,2.2 10 5", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		got := IsZeroStatementBlock(tt.line)
		if got != tt.expected {
			t.Errorf("IsZeroStatementBlock(%q) = %v, want %v", tt.line, got, tt.expected)
		}
	}
}

func TestCountProfileBlocks(t *testing.T) {
	input := `mode: atomic
github.com/s1onique/leamas/cmd/leamas/main.go:1.1,5.2 10 10
github.com/s1onique/leamas/internal/foo/foo.go:10.1,20.2 100 75
`
	covered, total, err := CountProfileBlocksReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("CountProfileBlocksReader() error: %v", err)
	}
	// Both blocks have count > 0, so both are covered
	// total=10+100=110, covered=10+100=110
	if covered != 110 {
		t.Errorf("covered = %d, want 110", covered)
	}
	if total != 110 {
		t.Errorf("total = %d, want 110", total)
	}
}

// CountProfileBlocksReader is a helper for testing
func CountProfileBlocksReader(r io.Reader) (covered, total int, err error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		block, err := ParseProfileBlock(scanner.Text())
		if err != nil || block == nil {
			continue
		}
		total += block.NumStatements
		if block.Count > 0 {
			covered += block.NumStatements
		}
	}
	return covered, total, scanner.Err()
}

func TestCheckThreshold(t *testing.T) {
	tests := []struct {
		name     string
		totalPct float64
		minTotal float64
		wantErr  bool
	}{
		// Legacy tests (60% threshold)
		{"pass at exactly 60.0", 60.0, 60.0, false},
		{"fail at 59.9", 59.9, 60.0, true},
		{"pass at 62.2", 62.2, 60.0, false},
		{"fail below threshold", 50.0, 60.0, true},
		{"pass above threshold", 100.0, 60.0, false},
		{"pass at zero threshold", 50.0, 0.0, false},
		// Ratchet02 tests (64% threshold)
		{"pass at exactly 64.0", 64.0, 64.0, false},
		{"fail at 63.9", 63.9, 64.0, true},
		{"pass at 64.1", 64.1, 64.0, false},
		{"pass at 66.6", 66.6, 64.0, false},
		{"fail below 64 threshold", 63.0, 64.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &Report{TotalPercent: tt.totalPct}
			threshold := &Threshold{MinTotalPercent: tt.minTotal}
			err := CheckThreshold(report, threshold)
			if tt.wantErr && err == nil {
				t.Errorf("CheckThreshold() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("CheckThreshold() unexpected error: %v", err)
			}
		})
	}
}

// TestCheckThreshold_ErrorMessage is kept for backward compatibility with existing test coverage.
func TestCheckThreshold_ErrorMessage(t *testing.T) {
	report := &Report{TotalPercent: 59.9}
	threshold := &Threshold{MinTotalPercent: 60.0}
	err := CheckThreshold(report, threshold)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	covErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if covErr.Kind != "threshold_fail" {
		t.Errorf("Kind = %q, want %q", covErr.Kind, "threshold_fail")
	}
}

func TestAnalyze(t *testing.T) {
	// Create a temporary profile file with partial coverage
	// In atomic mode, count > 0 means the block is covered
	// Block 1: 30 statements, count=1 (covered - all 30 statements counted as covered)
	// Block 2: 70 statements, count=0 (uncovered - 0 statements counted as covered)
	// Total: 30/100 = 30%
	profileInput := `mode: atomic
github.com/s1onique/leamas/internal/factory/foo.go:1.1,5.2 30 1
github.com/s1onique/leamas/internal/factory/bar.go:10.1,20.2 70 0
`
	profilePath := t.TempDir() + "/coverage.out"
	if err := os.WriteFile(profilePath, []byte(profileInput), 0644); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	// Test passing threshold (30% >= 25%)
	t.Run("pass at 25 threshold", func(t *testing.T) {
		threshold := &Threshold{MinTotalPercent: 25.0}
		report, err := Analyze(profilePath, threshold)
		if err != nil {
			t.Errorf("Analyze() unexpected error: %v", err)
		}
		if report == nil {
			t.Error("Analyze() returned nil report")
		}
	})

	// Test passing at exactly 30%
	t.Run("pass at exactly 30", func(t *testing.T) {
		threshold := &Threshold{MinTotalPercent: 30.0}
		report, err := Analyze(profilePath, threshold)
		if err != nil {
			t.Errorf("Analyze() unexpected error: %v", err)
		}
		if report == nil {
			t.Error("Analyze() returned nil report")
		}
	})

	// Test failing threshold (30% < 35%)
	t.Run("fail above threshold", func(t *testing.T) {
		threshold := &Threshold{MinTotalPercent: 35.0}
		report, err := Analyze(profilePath, threshold)
		if err == nil {
			t.Error("Analyze() expected error, got nil")
		}
		if report != nil {
			t.Error("Analyze() should return nil on error")
		}
	})
}

func TestAnalyze_FileNotFound(t *testing.T) {
	threshold := &Threshold{MinTotalPercent: 60.0}
	_, err := Analyze("/nonexistent/path/coverage.out", threshold)
	if err == nil {
		t.Error("Analyze() expected error for missing file, got nil")
	}
}
