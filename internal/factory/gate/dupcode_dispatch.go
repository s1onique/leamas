// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"fmt"
	"path/filepath"
	"sync/atomic"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// dupcodeReentryKind is the canonical kind for a binder re-entry finding.
const dupcodeReentryKind = "dupcode_execution_reentered"

// dupcodeReentryMessage is the canonical message for a binder re-entry finding.
const dupcodeReentryMessage = "dupcode bound runner invoked more than once"

// binderEntryGuard is an invocation-local concurrency-safe entry guard for a
// bound runner. The first call succeeds and returns nil. Every subsequent
// call returns exactly one explicit invariant finding so a contract violation
// cannot be misread as a successful zero-finding dispatch.
//
// We do NOT use sync.Once: the contract requires surfacing the second
// invocation as an observable invariant breach, not silently suppressing it.
type binderEntryGuard struct {
	entered atomic.Bool
}

// enter attempts to claim the binder entry slot. It returns nil on the first
// successful claim and a single dupcode_execution_reentered finding on every
// later call. The guard is concurrency-safe: at most one caller observes a
// successful claim even under concurrent invocation.
func (g *binderEntryGuard) enter() []checks.Finding {
	if g.entered.CompareAndSwap(false, true) {
		return nil
	}
	return []checks.Finding{{
		Path:     "dupcode",
		Kind:     dupcodeReentryKind,
		Message:  dupcodeReentryMessage,
		Severity: checks.SeverityError,
	}}
}

// DupcodeVerifySpec is the data-only dispatch request for the dupcode verify lane.
type DupcodeVerifySpec struct {
	BaselinePath string
	MinLines     int
	MinTokens    int
}

// DupcodeBaselineSpec is the data-only dispatch request for the dupcode-baseline lane.
type DupcodeBaselineSpec struct {
	BaselinePath string
	MinLines     int
	MinTokens    int
}

// DupcodeUpdateBaselineSpec is the data-only dispatch request for the
// dupcode update-baseline lane.
type DupcodeUpdateBaselineSpec struct {
	BaselinePath string
	MinLines     int
	MinTokens    int
}

// dupcodeRunner is the invocation-local runner interface. The binder
// receives a deps function that produces a fresh runner; production
// dependencies construct a *protectedverifier.DupcodeRunner, tests
// supply counting fakes.
type dupcodeRunner interface {
	LoadBaseline(string) (dupcode.Baseline, error)
	RunCheckRepo(string, dupcode.Config) ([]dupcode.Finding, error)
	RunCheckReport(string, dupcode.Config) (dupcode.Report, error)
	VerifyBaseline(string, dupcode.BaselinePolicy) ([]checks.Finding, error)
	WriteBaseline(string, dupcode.Report) error
	CompareToBaseline(dupcode.Report, dupcode.Baseline) dupcode.CompareResult
}

// dupcodeBinderDeps is the invocation-local dependency bundle. The
// production default constructs a real *protectedverifier.DupcodeRunner.
// Tests supply counting fakes by overriding NewRunner.
type dupcodeBinderDeps struct {
	NewRunner func() dupcodeRunner
}

// DupcodeVerifyBinder is the named post-authority binder for the dupcode
// verify lane. It owns an invocation-local payload cell so the typed
// outcome can be read after Dispatch returns, without re-executing the
// protected work.
type DupcodeVerifyBinder struct {
	spec  DupcodeVerifySpec
	deps  dupcodeBinderDeps
	entry binderEntryGuard

	report  dupcode.Report
	compRes dupcode.CompareResult
}

// NewDupcodeVerifyBinder creates a binder for the given spec with
// production deps.
func NewDupcodeVerifyBinder(spec DupcodeVerifySpec) *DupcodeVerifyBinder {
	return newDupcodeVerifyBinderWithDeps(spec, productionDupcodeBinderDeps())
}

func newDupcodeVerifyBinderWithDeps(spec DupcodeVerifySpec, deps dupcodeBinderDeps) *DupcodeVerifyBinder {
	return &DupcodeVerifyBinder{spec: spec, deps: deps}
}

// Spec returns the bound spec (read-only).
func (b *DupcodeVerifyBinder) Spec() DupcodeVerifySpec {
	return b.spec
}

// BindRunner returns the dispatcher-compatible RunnerFactory.
func (b *DupcodeVerifyBinder) BindRunner() verifierdispatch.RunnerFactory {
	return func() func(root string) []checks.Finding {
		return b.run
	}
}

// run is the named bound runner. It executes the protected work AT MOST
// ONCE per binder instance and stores the typed payload for later
// retrieval by the typed dispatch entry point.
//
// Concurrency-safe entry: the binderEntryGuard rejects every invocation
// after the first with a dupcode_execution_reentered finding. This makes
// any double-invocation observable rather than silently skipped.
func (b *DupcodeVerifyBinder) run(root string) []checks.Finding {
	if findings := b.entry.enter(); findings != nil {
		return findings
	}

	runner := b.deps.NewRunner()

	baselinePath := b.spec.BaselinePath
	fullBaselinePath := baselinePath
	if root != "." && root != "" {
		fullBaselinePath = filepath.Join(root, baselinePath)
	}

	if !checks.FileExists(fullBaselinePath) {
		return []checks.Finding{
			{Path: baselinePath, Kind: "missing_baseline",
				Message:  "baseline file not found. Run 'make dupcode-baseline' to create it.",
				Severity: checks.SeverityError},
		}
	}
	baseline, err := runner.LoadBaseline(fullBaselinePath)
	if err != nil {
		return []checks.Finding{
			{Path: baselinePath, Kind: "baseline_error",
				Message:  fmt.Sprintf("failed to load baseline: %v", err),
				Severity: checks.SeverityError},
		}
	}
	cfg := dupcode.DefaultConfig()
	cfg.Root = root
	cfg.MinLines = b.spec.MinLines
	cfg.MinTokens = b.spec.MinTokens

	report, err := runner.RunCheckReport(root, cfg)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error",
				Message:  fmt.Sprintf("duplicate code scan failed: %v", err),
				Severity: checks.SeverityError},
		}
	}

	b.report = report
	b.compRes = runner.CompareToBaseline(report, baseline)
	return convertCompareResult(b.compRes)
}

// Report returns the exact dupcode.Report captured during the authorized
// invocation, or a zero-value if the runner was not admitted.
func (b *DupcodeVerifyBinder) Report() dupcode.Report {
	return b.report
}

// CompareResult returns the exact dupcode.CompareResult captured during
// the authorized invocation, or a zero-value if the runner was not admitted.
func (b *DupcodeVerifyBinder) CompareResult() dupcode.CompareResult {
	return b.compRes
}

// DupcodeBaselineBinder is the named post-authority binder for the
// dupcode-baseline lane.
type DupcodeBaselineBinder struct {
	spec     DupcodeBaselineSpec
	deps     dupcodeBinderDeps
	entry    binderEntryGuard
	findings []checks.Finding
}

// NewDupcodeBaselineBinder creates a binder for the given spec.
func NewDupcodeBaselineBinder(spec DupcodeBaselineSpec) *DupcodeBaselineBinder {
	return newDupcodeBaselineBinderWithDeps(spec, productionDupcodeBinderDeps())
}

func newDupcodeBaselineBinderWithDeps(spec DupcodeBaselineSpec, deps dupcodeBinderDeps) *DupcodeBaselineBinder {
	return &DupcodeBaselineBinder{spec: spec, deps: deps}
}

// Spec returns the bound spec (read-only).
func (b *DupcodeBaselineBinder) Spec() DupcodeBaselineSpec {
	return b.spec
}

// BindRunner returns the dispatcher-compatible RunnerFactory.
func (b *DupcodeBaselineBinder) BindRunner() verifierdispatch.RunnerFactory {
	return func() func(root string) []checks.Finding {
		return b.run
	}
}

// run is the named bound runner for the dupcode-baseline lane. The
// binderEntryGuard rejects every invocation after the first with a
// dupcode_execution_reentered finding.
func (b *DupcodeBaselineBinder) run(root string) []checks.Finding {
	if findings := b.entry.enter(); findings != nil {
		return findings
	}

	runner := b.deps.NewRunner()
	policy := dupcode.BaselinePolicy{
		Path:      b.spec.BaselinePath,
		MinLines:  b.spec.MinLines,
		MinTokens: b.spec.MinTokens,
	}
	if root != "." && root != "" {
		policy.Path = filepath.Join(root, policy.Path)
	}
	findings, err := runner.VerifyBaseline(root, policy)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error",
				Message:  fmt.Sprintf("baseline verification failed: %v", err),
				Severity: checks.SeverityError},
		}
	}
	b.findings = findings
	return findings
}

// Findings returns the typed findings captured during the authorized
// invocation, or nil if the runner was not admitted.
func (b *DupcodeBaselineBinder) Findings() []checks.Finding {
	return b.findings
}

// DupcodeUpdateBaselineBinder is the named post-authority binder for the
// dupcode update-baseline lane.
type DupcodeUpdateBaselineBinder struct {
	spec    DupcodeUpdateBaselineSpec
	deps    dupcodeBinderDeps
	entry   binderEntryGuard
	report  dupcode.Report
	written bool
}

// NewDupcodeUpdateBaselineBinder creates a binder for the given spec.
func NewDupcodeUpdateBaselineBinder(spec DupcodeUpdateBaselineSpec) *DupcodeUpdateBaselineBinder {
	return newDupcodeUpdateBaselineBinderWithDeps(spec, productionDupcodeBinderDeps())
}

func newDupcodeUpdateBaselineBinderWithDeps(spec DupcodeUpdateBaselineSpec, deps dupcodeBinderDeps) *DupcodeUpdateBaselineBinder {
	return &DupcodeUpdateBaselineBinder{spec: spec, deps: deps}
}

// Spec returns the bound spec (read-only).
func (b *DupcodeUpdateBaselineBinder) Spec() DupcodeUpdateBaselineSpec {
	return b.spec
}

// BindRunner returns the dispatcher-compatible RunnerFactory.
func (b *DupcodeUpdateBaselineBinder) BindRunner() verifierdispatch.RunnerFactory {
	return func() func(root string) []checks.Finding {
		return b.run
	}
}

// run is the named bound runner for the dupcode update-baseline lane.
// The binderEntryGuard rejects every invocation after the first with a
// dupcode_execution_reentered finding.
func (b *DupcodeUpdateBaselineBinder) run(root string) []checks.Finding {
	if findings := b.entry.enter(); findings != nil {
		return findings
	}

	runner := b.deps.NewRunner()
	cfg := dupcode.DefaultConfig()
	cfg.Root = root
	cfg.MinLines = b.spec.MinLines
	cfg.MinTokens = b.spec.MinTokens

	report, err := runner.RunCheckReport(root, cfg)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error",
				Message:  fmt.Sprintf("scan failed: %v", err),
				Severity: checks.SeverityError},
		}
	}
	if err := runner.WriteBaseline(b.spec.BaselinePath, report); err != nil {
		return []checks.Finding{
			{Path: b.spec.BaselinePath, Kind: "baseline_error",
				Message:  fmt.Sprintf("failed to write baseline: %v", err),
				Severity: checks.SeverityError},
		}
	}
	b.report = report
	b.written = true
	return nil
}

// Report returns the exact dupcode.Report written to the baseline, or a
// zero-value if the runner was not admitted.
func (b *DupcodeUpdateBaselineBinder) Report() dupcode.Report {
	return b.report
}

// Written reports whether the baseline file was successfully written.
func (b *DupcodeUpdateBaselineBinder) Written() bool {
	return b.written
}

// dispatchDupcodeVerifyTypedWith is the testable typed dispatch entry
// point for the dupcode verify lane. It accepts a custom authority
// observer and binder deps for dynamic test injection. The binder is
// invoked AT MOST ONCE inside Dispatch; the typed outcome reads from
// the binder's invocation-local payload cell.
func dispatchDupcodeVerifyTypedWith(
	ctx context.Context,
	root string,
	spec DupcodeVerifySpec,
	observer verifierdispatch.ContextObserver,
	deps dupcodeBinderDeps,
) DupcodeVerifyOutcome {
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		return DupcodeVerifyOutcome{
			Dispatch: verifierdispatch.Result{
				Error: fmt.Errorf("dupcode verifier not found in registry"),
			},
		}
	}
	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       root,
	}
	binder := newDupcodeVerifyBinderWithDeps(spec, deps)

	outcome := DupcodeVerifyOutcome{
		Dispatch: dispatcher.Dispatch(ctx, request, observer, binder.BindRunner()),
	}

	// Read the typed payload from the binder's invocation-local cell.
	// Do NOT execute the protected work again. The report and compare
	// result are zero-valued if the runner was not admitted (denial path).
	outcome.Report = reportToDTO(binder.Report(), spec)
	outcome.Comparison = compareResultToDTO(binder.CompareResult())
	return outcome
}
