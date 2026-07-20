// Package main provides factory test-long handler.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/s1onique/leamas/internal/execution"
	"github.com/s1onique/leamas/internal/factory/longtest"
)

type testLongResult struct {
	ID       string `json:"id"`
	Package  string `json:"package"`
	Test     string `json:"test"`
	Passed   bool   `json:"passed"`
	Duration string `json:"duration,omitempty"`
	Error    string `json:"error,omitempty"`
}

type testLongSummary struct {
	SchemaVersion int              `json:"schema_version"`
	GeneratedAt   string           `json:"generated_at"`
	Tests         []testLongResult `json:"tests"`
	Passed        int              `json:"passed"`
	Failed        int              `json:"failed"`
	Total         int              `json:"total"`
}

func handleFactoryTestLong() {
	args := os.Args[3:]
	var groupFilter string
	fs := flag.NewFlagSet("factory test-long", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&groupFilter, "group", "", "filter by ci_group")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: leamas factory test-long [--group=<ci_group>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "test-long: %v\n", err)
		os.Exit(1)
	}
	root := "."
	if fs.NArg() > 0 {
		root = fs.Arg(0)
	}
	baseline, err := longtest.LoadBaseline(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "test-long: load baseline: %v\n", err)
		os.Exit(1)
	}
	if err := longtest.ValidateBaseline(baseline); err != nil {
		fmt.Fprintf(os.Stderr, "test-long: validate baseline: %v\n", err)
		os.Exit(1)
	}
	var filtered []longtest.TestSpec
	for _, tt := range baseline.Tests {
		if groupFilter == "" || tt.CIGroup == groupFilter {
			filtered = append(filtered, tt)
		}
	}
	if len(filtered) == 0 {
		if groupFilter != "" {
			fmt.Fprintf(os.Stderr, "test-long: no tests found for group %q\n", groupFilter)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "test-long: no tests in baseline\n")
		os.Exit(0)
	}
	sort.Slice(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		if a.CIGroup != b.CIGroup {
			return a.CIGroup < b.CIGroup
		}
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		if a.Test != b.Test {
			return a.Test < b.Test
		}
		return a.ID < b.ID
	})
	var results []testLongResult
	failed := 0
	for _, tt := range filtered {
		result := testLongResult{ID: tt.ID, Package: tt.Package, Test: tt.Test}
		passed, duration, err := runTest(root, tt)
		result.Passed = passed
		if passed {
			result.Duration = duration.String()
		} else {
			result.Error = err.Error()
			failed++
		}
		results = append(results, result)
	}
	summary := testLongSummary{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Tests:         results,
		Passed:        len(results) - failed,
		Failed:        failed,
		Total:         len(results),
	}
	summaryPath := ".factory/gate-long-summary.json"
	summaryData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "test-long: marshal summary: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(summaryPath, summaryData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "test-long: write summary: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("test-long: ran %d tests, %d passed, %d failed\n", summary.Total, summary.Passed, summary.Failed)
	if failed > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

func runTest(root string, tt longtest.TestSpec) (bool, time.Duration, error) {
	timeout, err := time.ParseDuration(tt.CITimeout)
	if err != nil {
		return false, 0, fmt.Errorf("invalid timeout: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	pattern := "^" + regexp.QuoteMeta(tt.Test) + "$"
	args := []string{"test", "-count=1", "-timeout=" + tt.CITimeout, "-run=" + pattern, tt.Package}
	start := time.Now()
	result := execution.RunGoTest(ctx, args...)
	duration := time.Since(start)
	if ctx.Err() == context.DeadlineExceeded {
		return false, duration, fmt.Errorf("timeout after %s", tt.CITimeout)
	}
	if result.Error != nil || result.ExitCode != 0 {
		return false, duration, fmt.Errorf("test failed: %v", result.Error)
	}
	return true, duration, nil
}
