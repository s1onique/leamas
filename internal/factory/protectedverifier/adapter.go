// SPDX-License-Identifier: Apache-2.0

package protectedverifier

import (
	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// RunnerFunc is the type for protected verifier runner functions.
type RunnerFunc func(root string) []interface{}

// DupcodeRunner wraps dupcode capabilities for authorized execution.
// This is the ONLY production entry point for dupcode operations.
type DupcodeRunner struct{}

// NewDupcodeRunner creates a new authorized dupcode runner.
func NewDupcodeRunner() *DupcodeRunner {
	return &DupcodeRunner{}
}

// RunCheckReport executes a dupcode check and returns the report.
// This function may only be called by the verifierdispatch.Dispatcher
// after CI-exact-checkout authority validation passes.
func (r *DupcodeRunner) RunCheckReport(root string, cfg dupcode.Config) (dupcode.Report, error) {
	return dupcode.CheckReport(root, cfg)
}

// RunCheckRepo executes a dupcode repository check.
// This function may only be called by the verifierdispatch.Dispatcher
// after CI-exact-checkout authority validation passes.
func (r *DupcodeRunner) RunCheckRepo(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
	return dupcode.CheckRepo(root, cfg)
}

// LoadBaseline loads a dupcode baseline from disk.
// This function may only be called by the verifierdispatch.Dispatcher
// after CI-exact-checkout authority validation passes.
func (r *DupcodeRunner) LoadBaseline(path string) (dupcode.Baseline, error) {
	return dupcode.LoadBaseline(path)
}

// WriteBaseline writes a dupcode baseline to disk.
// This function may only be called by the verifierdispatch.Dispatcher
// after appropriate mutation authority passes.
func (r *DupcodeRunner) WriteBaseline(path string, report dupcode.Report) error {
	return dupcode.WriteBaseline(path, report)
}

// CompareToBaseline compares a report to a baseline.
// This function may only be called by the verifierdispatch.Dispatcher
// after CI-exact-checkout authority validation passes.
func (r *DupcodeRunner) CompareToBaseline(report dupcode.Report, baseline dupcode.Baseline) dupcode.CompareResult {
	return dupcode.CompareToBaseline(report, baseline)
}

// DupcodeAnalyzer is the function type for authorized dupcode analysis.
type DupcodeAnalyzer func(root string, cfg dupcode.Config) ([]dupcode.Finding, error)

// DefaultAnalyzer is the production dupcode analyzer.
var DefaultAnalyzer DupcodeAnalyzer = dupcode.CheckRepo

// Analyzer returns the authorized dupcode analyzer.
func Analyzer() DupcodeAnalyzer {
	return DefaultAnalyzer
}

// SetAnalyzer sets the analyzer (for testing only).
func SetAnalyzer(a DupcodeAnalyzer) {
	DefaultAnalyzer = a
}
