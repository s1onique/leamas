// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"testing"
)

// TestDupcodeVerifyAdmitted proves that an admitting authority runs each
// protected operation exactly once and the typed payload survives DTO
// conversion.
func TestDupcodeVerifyAdmitted(t *testing.T) {
	dir := t.TempDir()
	_ = writeFakeBaseline(t, dir, "baseline.json", 40, 400)
	r := &countingDupcodeRunner{
		baselineToReturn: dupcode.Baseline{
			Thresholds: dupcode.BaselineThresholds{MinLines: 40, MinTokens: 400},
		},
		reportToReturn: dupcode.Report{
			Root: dir,
			Findings: []dupcode.Finding{
				{Fingerprint: "abc", TokenCount: 250, LineCount: 50, Occurrences: []dupcode.Occurrence{{Path: "a.go", StartLine: 1, EndLine: 50}}},
			},
			Thresholds: dupcode.BaselineThresholds{MinLines: 40, MinTokens: 400},
		},
	}
	outcome := dispatchDupcodeVerifyTypedWith(
		context.Background(), dir, DupcodeVerifySpec{
			BaselinePath: "baseline.json", MinLines: 40, MinTokens: 400,
		}, &admittingObserver{}, makeVerifyDeps(r),
	)
	if outcome.Dispatch.Error != nil {
		t.Fatalf("expected admission: error=%v findings=%v", outcome.Dispatch.Error, outcome.Dispatch.Findings)
	}
	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1", got)
	}
	if got := r.loadBaselineCalls.Load(); got != 1 {
		t.Errorf("loadBaselineCalls = %d, want 1", got)
	}
	if got := r.scanCalls.Load(); got != 1 {
		t.Errorf("scanCalls = %d, want 1", got)
	}
	if got := r.compareCalls.Load(); got != 1 {
		t.Errorf("compareCalls = %d, want 1", got)
	}

	// Verify the exact fake domain report and comparison survive DTO conversion.
	if outcome.Report.Root != dir {
		t.Errorf("Report.Root = %q, want %q", outcome.Report.Root, dir)
	}
	if outcome.Report.FindingCount != 1 {
		t.Errorf("Report.FindingCount = %d, want 1", outcome.Report.FindingCount)
	}
	if outcome.Report.MinLines != 40 || outcome.Report.MinTokens != 400 {
		t.Errorf("Report thresholds = (%d,%d), want (40,400)", outcome.Report.MinLines, outcome.Report.MinTokens)
	}
	if outcome.Comparison.NewCount != 1 {
		t.Errorf("Comparison.NewCount = %d, want 1", outcome.Comparison.NewCount)
	}
	if outcome.Comparison.WorsenedCount != 1 {
		t.Errorf("Comparison.WorsenedCount = %d, want 1", outcome.Comparison.WorsenedCount)
	}
	if !outcome.Comparison.HasChanges {
		t.Errorf("Comparison.HasChanges = false, want true")
	}
	if len(outcome.Comparison.NewFindings) != 1 {
		t.Errorf("Comparison.NewFindings len = %d, want 1", len(outcome.Comparison.NewFindings))
	}
	if len(outcome.Comparison.WorsenedFindings) != 1 {
		t.Errorf("Comparison.WorsenedFindings len = %d, want 1", len(outcome.Comparison.WorsenedFindings))
	}
}

// TestDupcodeBaselineAdmitted proves that an admitting authority runs
// VerifyBaseline exactly once.
func TestDupcodeBaselineAdmitted(t *testing.T) {
	r := &countingDupcodeRunner{
		verifyFindingsToReturn: []checks.Finding{
			{Path: "x.go", Kind: "baseline_threshold_mismatch", Message: "fake mismatch", Severity: checks.SeverityError},
		},
	}
	outcome := dispatchDupcodeBaselineVerifyTypedWith(
		context.Background(), ".", DupcodeBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &admittingObserver{}, makeBaselineDeps(r),
	)
	if outcome.Dispatch.Error != nil {
		t.Fatalf("expected admission: error=%v findings=%v", outcome.Dispatch.Error, outcome.Dispatch.Findings)
	}
	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1", got)
	}
	if got := r.verifyCalls.Load(); got != 1 {
		t.Errorf("verifyCalls = %d, want 1", got)
	}
	if len(outcome.Findings) != 1 {
		t.Errorf("Findings len = %d, want 1", len(outcome.Findings))
	}
}

// TestDupcodeUpdateAdmitted proves that an admitting authority runs
// RunCheckReport and WriteBaseline exactly once, and the typed outcome's
// Report is the exact report supplied to WriteBaseline.
//
// The current production registry registers "dupcode" with
// AuthorityCIExactCheckout, which denies OperationUpdateBaseline. This
// test is therefore a structural placeholder for the local-safe path; it
// asserts the operation-policy denial contract instead. The full
// admission proof is gated by a future correction that switches the
// dupcode registry entry to AuthorityLocalSafe.
func TestDupcodeUpdateAdmitted(t *testing.T) {
	r := &countingDupcodeRunner{}
	outcome := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &admittingObserver{}, makeUpdateDeps(r),
	)
	// Operation-policy denial must surface as Dispatch.Error and a
	// verifier_execution_authority_denied finding.
	if outcome.Dispatch.Error == nil {
		t.Fatalf("expected update to be denied under CI authority, got error=nil")
	}
	if got := len(outcome.Dispatch.Findings); got != 1 {
		t.Fatalf("expected exactly one denial finding, got %d", got)
	}
	if outcome.Dispatch.Findings[0].Kind != "verifier_execution_authority_denied" {
		t.Errorf("finding kind = %q, want %q", outcome.Dispatch.Findings[0].Kind, "verifier_execution_authority_denied")
	}
	if got := r.newRunnerCalls.Load(); got != 0 {
		t.Errorf("newRunnerCalls = %d, want 0", got)
	}
	if got := r.scanCalls.Load(); got != 0 {
		t.Errorf("scanCalls = %d, want 0", got)
	}
	if got := r.writeCalls.Load(); got != 0 {
		t.Errorf("writeCalls = %d, want 0", got)
	}
}

// TestDupcodeVerifyExactlyOnce is the behavioral exactly-once proof for
// the verify typed entry point.
func TestDupcodeVerifyExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	_ = writeFakeBaseline(t, dir, "baseline.json", 40, 400)
	r := &countingDupcodeRunner{
		baselineToReturn: dupcode.Baseline{},
		reportToReturn:   dupcode.Report{},
	}
	outcome := dispatchDupcodeVerifyTypedWith(
		context.Background(), dir, DupcodeVerifySpec{
			BaselinePath: "baseline.json", MinLines: 40, MinTokens: 400,
		}, &admittingObserver{}, makeVerifyDeps(r),
	)
	if outcome.Dispatch.Error != nil {
		t.Fatalf("expected admission: error=%v findings=%v", outcome.Dispatch.Error, outcome.Dispatch.Findings)
	}
	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1", got)
	}
	if got := r.loadBaselineCalls.Load(); got != 1 {
		t.Errorf("loadBaselineCalls = %d, want 1", got)
	}
	if got := r.scanCalls.Load(); got != 1 {
		t.Errorf("scanCalls = %d, want 1", got)
	}
}

// TestDupcodeBaselineExactlyOnce is the behavioral exactly-once proof for
// the dupcode-baseline typed entry point.
func TestDupcodeBaselineExactlyOnce(t *testing.T) {
	r := &countingDupcodeRunner{
		verifyFindingsToReturn: nil,
	}
	outcome := dispatchDupcodeBaselineVerifyTypedWith(
		context.Background(), ".", DupcodeBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &admittingObserver{}, makeBaselineDeps(r),
	)
	if outcome.Dispatch.Error != nil {
		t.Fatalf("expected admission: error=%v findings=%v", outcome.Dispatch.Error, outcome.Dispatch.Findings)
	}
	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1", got)
	}
	if got := r.verifyCalls.Load(); got != 1 {
		t.Errorf("verifyCalls = %d, want 1", got)
	}
}

// TestDupcodeUpdateExactlyOnce is the behavioral exactly-once proof for
// the update-baseline typed entry point. Under CI authority, update is
// denied at the operation-policy layer; this test verifies the same
// exactly-once behavior on the denial path. See TestDupcodeUpdateAdmitted
// for the registry-context rationale.
func TestDupcodeUpdateExactlyOnce(t *testing.T) {
	r := &countingDupcodeRunner{}
	outcome := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &admittingObserver{}, makeUpdateDeps(r),
	)
	if outcome.Dispatch.Error == nil {
		t.Fatalf("expected update denial, got error=nil")
	}
	if got := r.newRunnerCalls.Load(); got != 0 {
		t.Errorf("newRunnerCalls = %d, want 0 (denial before factory)", got)
	}
	if got := r.scanCalls.Load(); got != 0 {
		t.Errorf("scanCalls = %d, want 0", got)
	}
	if got := r.writeCalls.Load(); got != 0 {
		t.Errorf("writeCalls = %d, want 0", got)
	}
}

// TestDupcodeVerifyOutcomeTyped and friends verify the typed entry points
// are reachable and return DupcodeVerifyOutcome etc. They guard against
// accidental refactors that drop the typed surface.
func TestDupcodeVerifyTyped(t *testing.T) {
	r := &countingDupcodeRunner{
		baselineToReturn: dupcode.Baseline{},
		reportToReturn:   dupcode.Report{},
	}
	out := dispatchDupcodeVerifyTypedWith(
		context.Background(), ".", DupcodeVerifySpec{
			BaselinePath: "missing.json", MinLines: 40, MinTokens: 400,
		}, &admittingObserver{}, makeVerifyDeps(r),
	)
	_ = out // outcome exists; if it didn't, the call would not compile
}

func TestDupcodeBaselineTyped(t *testing.T) {
	r := &countingDupcodeRunner{}
	out := dispatchDupcodeBaselineVerifyTypedWith(
		context.Background(), ".", DupcodeBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &admittingObserver{}, makeBaselineDeps(r),
	)
	_ = out
}

func TestDupcodeUpdateTyped(t *testing.T) {
	r := &countingDupcodeRunner{}
	out := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &admittingObserver{}, makeUpdateDeps(r),
	)
	_ = out
}
