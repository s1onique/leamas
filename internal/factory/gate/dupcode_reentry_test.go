// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"sync"
	"sync/atomic"
	"testing"
)

// TestDupcodeVerifyReentryFailClosed proves the verify binder's
// concurrency-safe entry guard rejects the second invocation and returns
// exactly one dupcode_execution_reentered finding without re-running
// protected work.
func TestDupcodeVerifyReentry(t *testing.T) {
	r := &countingDupcodeRunner{
		baselineToReturn: dupcode.Baseline{},
		reportToReturn:   dupcode.Report{},
	}
	binder := newDupcodeVerifyBinderWithDeps(
		DupcodeVerifySpec{BaselinePath: "missing.json", MinLines: 40, MinTokens: 400},
		makeVerifyDeps(r),
	)

	// First invocation completes the guarded entry; protected work runs.
	first := binder.BindRunner()()(".")
	if len(first) != 1 || first[0].Kind != "missing_baseline" {
		t.Fatalf("first call findings = %+v, want one missing_baseline finding", first)
	}

	// Second invocation must produce exactly one dupcode_execution_reentered
	// finding without invoking the runner factory or any protected operation.
	second := binder.BindRunner()()(".")
	if len(second) != 1 {
		t.Fatalf("second call findings len = %d, want 1", len(second))
	}
	if second[0].Kind != "dupcode_execution_reentered" {
		t.Errorf("second call kind = %q, want %q", second[0].Kind, "dupcode_execution_reentered")
	}
	if second[0].Path != "dupcode" {
		t.Errorf("second call path = %q, want %q", second[0].Path, "dupcode")
	}
	if second[0].Severity != checks.SeverityError {
		t.Errorf("second call severity = %q, want %q", second[0].Severity, checks.SeverityError)
	}

	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1 (entry guard must not invoke factory again)", got)
	}
}

// TestDupcodeBaselineReentryFailClosed proves the dupcode-baseline binder's
// entry guard behaves identically.
func TestDupcodeBaselineReentry(t *testing.T) {
	r := &countingDupcodeRunner{
		verifyFindingsToReturn: []checks.Finding{
			{Path: "x", Kind: "baseline_threshold_mismatch", Message: "fake", Severity: checks.SeverityError},
		},
	}
	binder := newDupcodeBaselineBinderWithDeps(
		DupcodeBaselineSpec{BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400},
		makeBaselineDeps(r),
	)

	first := binder.BindRunner()()(".")
	if len(first) != 1 || first[0].Kind != "baseline_threshold_mismatch" {
		t.Fatalf("first call findings = %+v, want one baseline_threshold_mismatch finding", first)
	}

	second := binder.BindRunner()()(".")
	if len(second) != 1 {
		t.Fatalf("second call findings len = %d, want 1", len(second))
	}
	if second[0].Kind != "dupcode_execution_reentered" {
		t.Errorf("second call kind = %q, want %q", second[0].Kind, "dupcode_execution_reentered")
	}
	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1", got)
	}
}

// TestDupcodeUpdateReentryFailClosed proves the update-baseline binder's
// entry guard behaves identically.
func TestDupcodeUpdateReentry(t *testing.T) {
	r := &countingDupcodeRunner{
		reportToReturn: dupcode.Report{},
	}
	binder := newDupcodeUpdateBaselineBinderWithDeps(
		DupcodeUpdateBaselineSpec{BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400},
		makeUpdateDeps(r),
	)

	// First invocation: scan + write completes successfully.
	first := binder.BindRunner()()(".")
	if len(first) != 0 {
		t.Fatalf("first call findings = %+v, want empty", first)
	}

	second := binder.BindRunner()()(".")
	if len(second) != 1 {
		t.Fatalf("second call findings len = %d, want 1", len(second))
	}
	if second[0].Kind != "dupcode_execution_reentered" {
		t.Errorf("second call kind = %q, want %q", second[0].Kind, "dupcode_execution_reentered")
	}
	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1", got)
	}
}

// TestDupcodeVerifyConcurrentReentry proves the entry guard is
// concurrency-safe: two goroutines start at a barrier; protected work
// executes exactly once; one caller observes normal entry, the other
// observes the dupcode_execution_reentered finding.
func TestDupcodeVerifyConcurrentReentry(t *testing.T) {
	r := &countingDupcodeRunner{
		baselineToReturn: dupcode.Baseline{},
		reportToReturn:   dupcode.Report{},
	}
	binder := newDupcodeVerifyBinderWithDeps(
		DupcodeVerifySpec{BaselinePath: "missing.json", MinLines: 40, MinTokens: 400},
		makeVerifyDeps(r),
	)
	runner := binder.BindRunner()()

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([][]checks.Finding, 2)
	for i := 0; i < 2; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[i] = runner(".")
		}()
	}
	close(start)
	wg.Wait()

	// Exactly one caller observed protected work (missing_baseline) and
	// exactly one caller observed the re-entry finding.
	var sawProtected, sawReentry bool
	for _, findings := range results {
		if len(findings) == 0 {
			continue
		}
		switch findings[0].Kind {
		case "missing_baseline":
			sawProtected = true
		case "dupcode_execution_reentered":
			sawReentry = true
		default:
			t.Errorf("unexpected finding kind: %q", findings[0].Kind)
		}
	}
	if !sawProtected {
		t.Errorf("expected one caller to see protected-work finding (missing_baseline)")
	}
	if !sawReentry {
		t.Errorf("expected one caller to see dupcode_execution_reentered")
	}
	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1", got)
	}
}

// TestDupcodeBaselineConcurrentReentry is the concurrency-safe equivalent
// for the dupcode-baseline binder.
func TestDupcodeBaselineConcurrentReentry(t *testing.T) {
	r := &countingDupcodeRunner{
		verifyFindingsToReturn: nil,
	}
	binder := newDupcodeBaselineBinderWithDeps(
		DupcodeBaselineSpec{BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400},
		makeBaselineDeps(r),
	)
	runner := binder.BindRunner()()

	var wg sync.WaitGroup
	start := make(chan struct{})
	var sawReentry atomic.Bool
	var sawProtected atomic.Bool
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			findings := runner(".")
			if len(findings) == 1 && findings[0].Kind == "dupcode_execution_reentered" {
				sawReentry.Store(true)
			} else {
				sawProtected.Store(true)
			}
		}()
	}
	close(start)
	wg.Wait()
	if !sawReentry.Load() {
		t.Errorf("expected one caller to see dupcode_execution_reentered")
	}
	if !sawProtected.Load() {
		t.Errorf("expected one caller to see protected-work result")
	}
	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1", got)
	}
}

// TestDupcodeUpdateConcurrentReentry is the concurrency-safe equivalent
// for the update-baseline binder.
func TestDupcodeUpdateConcurrentReentry(t *testing.T) {
	r := &countingDupcodeRunner{
		reportToReturn: dupcode.Report{},
	}
	binder := newDupcodeUpdateBaselineBinderWithDeps(
		DupcodeUpdateBaselineSpec{BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400},
		makeUpdateDeps(r),
	)
	runner := binder.BindRunner()()

	var wg sync.WaitGroup
	start := make(chan struct{})
	var sawReentry atomic.Bool
	var sawProtected atomic.Bool
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			findings := runner(".")
			if len(findings) == 1 && findings[0].Kind == "dupcode_execution_reentered" {
				sawReentry.Store(true)
			} else {
				sawProtected.Store(true)
			}
		}()
	}
	close(start)
	wg.Wait()
	if !sawReentry.Load() {
		t.Errorf("expected one caller to see dupcode_execution_reentered")
	}
	if !sawProtected.Load() {
		t.Errorf("expected one caller to see protected-work result")
	}
	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1", got)
	}
}
